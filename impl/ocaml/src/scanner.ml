(* Source scanning: file discovery, candidate extraction, and incremental
   scans (SPEC §13). *)

open Utils

let ignored_directories =
  [ ".git"; ".hg"; ".jj"; ".next"; ".parcel-cache"; ".pnpm-store"; ".svelte-kit"; ".svn"; ".turbo"; ".venv"; ".vercel"; ".yarn";
    "__pycache__"; "node_modules"; "venv" ]

let ignored_extensions =
  fields
    "less lock sass scss styl log png jpg jpeg gif webp avif ico bmp tif tiff heic psd mp3 wav ogg flac aac m4a mp4 webm mov avi \
     mkv m4v zip gz tar tgz bz2 xz 7z rar woff woff2 ttf otf eot exe dll so dylib bin wasm pdf class jar pyc o a"

let ignored_files = [ "package-lock.json"; "pnpm-lock.yaml"; "bun.lockb"; ".gitignore"; ".env" ]

let is_ignored_file_name name =
  List.mem name ignored_files || has_prefix name ".env."
  ||
  match String.rindex_opt name '.' with
  | Some dot when dot > 0 -> List.mem (String.lowercase_ascii (after name (dot + 1))) ignored_extensions
  | _ -> false

(* Whether a pattern contains glob metacharacters. *)
let is_glob pattern = List.exists (fun c -> String.contains pattern c) [ '*'; '?'; '['; '{' ]

(* Converts an absolute posix glob (`**`, `*`, `?`, `{a,b}`) into a regular
   expression matching whole paths. *)
let glob_to_regexp glob =
  let b = Buffer.create 64 in
  Buffer.add_string b "^";
  let n = String.length glob in
  let braces = ref 0 in
  let i = ref 0 in
  while !i < n do
    (match glob.[!i] with
    | '*' ->
        if !i + 1 < n && glob.[!i + 1] = '*' then begin
          incr i;
          if !i + 1 < n && glob.[!i + 1] = '/' then (incr i; Buffer.add_string b "\\(.*/\\)?") else Buffer.add_string b ".*"
        end
        else Buffer.add_string b "[^/]*"
    | '?' -> Buffer.add_string b "[^/]"
    | '{' ->
        incr braces;
        Buffer.add_string b "\\("
    | '}' when !braces > 0 ->
        decr braces;
        Buffer.add_string b "\\)"
    | ',' when !braces > 0 -> Buffer.add_string b "\\|"
    | '\\' when !i + 1 < n ->
        incr i;
        Buffer.add_string b (Str.quote (String.make 1 glob.[!i]))
    | c -> Buffer.add_string b (Str.quote (String.make 1 c)));
    incr i
  done;
  Buffer.add_string b "$";
  Str.regexp (Buffer.contents b)

let matches re s = Str.string_match re s 0

(* Normalizes `.` and `..` segments and repeated separators. *)
let clean path =
  let absolute = has_prefix path "/" in
  let rec go acc = function
    | [] -> List.rev acc
    | ("" | ".") :: rest -> go acc rest
    | ".." :: rest -> (
        match acc with
        | x :: acc' when x <> ".." -> go acc' rest
        | _ -> if absolute then go acc rest else go (".." :: acc) rest)
    | p :: rest -> go (p :: acc) rest
  in
  let joined = String.concat "/" (go [] (String.split_on_char '/' path)) in
  if absolute then "/" ^ joined else if joined = "" then "." else joined

let abs_path path = clean (if Filename.is_relative path then Filename.concat (Sys.getcwd ()) path else path)
let join a b = if Filename.is_relative b then clean (Filename.concat a b) else clean b
let is_directory_path path = try Sys.is_directory path with Sys_error _ -> false
let is_file_path path = try (Unix.stat path).st_kind = Unix.S_REG with Unix.Unix_error _ -> false
let mtime path = try Some (Unix.stat path).st_mtime with Unix.Unix_error _ -> None
let read_file path = try Some (In_channel.with_open_bin path In_channel.input_all) with Sys_error _ -> None

let entries directory =
  match Sys.readdir directory with
  | names ->
      let names = Array.to_list names in
      List.sort compare names
  | exception Sys_error _ -> []

(* A compiled exclusion from a negated source: every path at or under a
   directory, or files matching a pattern. *)
type exclusion = Excluded_directory of string | Excluded_pattern of Str.regexp

type scoped_ignore = { directory : string; matcher : Gitignore.t }

type t = {
  sources : Directives.source_entry list;
  candidates : Extract.candidate_set;
  mtimes : (string, float) Hashtbl.t;
  mutable exclusions : exclusion list option;
  (* The files read by the most recent [scan]. *)
  mutable scanned_files : string list;
}

let create sources =
  { sources; candidates = Extract.new_candidate_set (); mtimes = Hashtbl.create 256; exclusions = None; scanned_files = [] }

let sources s = s.sources

(* The distinct absolute base directories of the positive sources. *)
let bases s =
  List.fold_left
    (fun acc (source : Directives.source_entry) ->
      if source.negated then acc
      else
        let base = abs_path source.base in
        if List.mem base acc then acc else acc @ [ base ])
    [] s.sources

let exclusions_for s =
  match s.exclusions with
  | Some e -> e
  | None ->
      let e =
        List.filter_map
          (fun (source : Directives.source_entry) ->
            if not source.negated then None
            else
              let target = abs_path (join source.base source.pattern) in
              if not (is_glob source.pattern) then
                if is_directory_path target then Some (Excluded_directory target)
                else Some (Excluded_pattern (Str.regexp ("^" ^ Str.quote target ^ "$")))
              else Some (Excluded_pattern (glob_to_regexp target)))
          s.sources
      in
      s.exclusions <- Some e;
      e

let is_excluded path exclusions =
  List.exists
    (function
      | Excluded_directory d -> path = d || has_prefix path (d ^ "/")
      | Excluded_pattern re -> matches re path)
    exclusions

(* An insertion-ordered file set. *)
type file_set = { fseen : (string, unit) Hashtbl.t; mutable flist_rev : string list }

let add_file f path =
  if not (Hashtbl.mem f.fseen path) then begin
    Hashtbl.replace f.fseen path ();
    f.flist_rev <- path :: f.flist_rev
  end

let is_dir_entry path =
  match Unix.lstat path with
  | { Unix.st_kind = Unix.S_DIR; _ } -> true
  | { Unix.st_kind = Unix.S_LNK; _ } -> is_directory_path path
  | _ -> false
  | exception Unix.Unix_error _ -> false

let is_gitignored path is_dir ignores =
  List.exists
    (fun ignore ->
      let prefix = ignore.directory ^ "/" in
      let rel = if has_prefix path prefix then after path (String.length prefix) else path in
      Gitignore.ignores ignore.matcher rel is_dir)
    ignores

let rec walk directory ignores exclusions found =
  let scoped =
    match read_file (Filename.concat directory ".gitignore") with
    | Some content ->
        let matcher = Gitignore.parse content in
        if Gitignore.size matcher > 0 then ignores @ [ { directory; matcher } ] else ignores
    | None -> ignores
  in
  List.iter
    (fun name ->
      let path = Filename.concat directory name in
      if is_dir_entry path then begin
        if List.mem name ignored_directories || is_gitignored path true scoped || is_excluded path exclusions then ()
        else walk path scoped exclusions found
      end
      else if is_ignored_file_name name || is_gitignored path false scoped || is_excluded path exclusions then ()
      else add_file found path)
    (entries directory)

let rec walk_all directory pattern exclusions found =
  List.iter
    (fun name ->
      let path = Filename.concat directory name in
      if is_dir_entry path then begin
        if name = ".git" || is_excluded path exclusions then () else walk_all path pattern exclusions found
      end
      else if (not (matches pattern path)) || is_excluded path exclusions then ()
      else add_file found path)
    (entries directory)

let glob_files target exclusions found =
  let pattern = glob_to_regexp target in
  (* Walk from the first non-glob segment of the absolute pattern. *)
  let rec fixed acc = function [] -> List.rev acc | seg :: rest -> if is_glob seg then List.rev acc else fixed (seg :: acc) rest in
  let start = String.concat "/" (fixed [] (String.split_on_char '/' target)) in
  let start = if start = "" then "/" else start in
  if is_file_path start then (if not (is_excluded start exclusions) then add_file found start)
  else walk_all start pattern exclusions found

(* Enumerates every file covered by the sources. *)
let files s =
  let exclusions = exclusions_for s in
  let found = { fseen = Hashtbl.create 256; flist_rev = [] } in
  List.iter
    (fun (source : Directives.source_entry) ->
      if source.negated then ()
      else
        let base = abs_path source.base in
        if source.pattern = "**/*" then walk base [] exclusions found
        else
          let target = abs_path (join base source.pattern) in
          if not (is_glob source.pattern) then begin
            if is_directory_path target then walk target [] exclusions found
            else if is_file_path target && not (is_excluded target exclusions) then add_file found target
          end
          else glob_files target exclusions found)
    s.sources;
  List.rev found.flist_rev

let read s path =
  match read_file path with
  | Some content ->
      Extract.extract_into content s.candidates;
      true
  | None -> false

(* Walks every source, reads files whose modification time changed since
   the last scan (all files on the first scan), and returns the full
   deduplicated candidate set seen so far. *)
let scan s =
  let read_files =
    List.filter
      (fun path ->
        match mtime path with
        | None -> false
        | Some m -> (
            match Hashtbl.find_opt s.mtimes path with
            | Some previous when previous = m -> false
            | _ ->
                Hashtbl.replace s.mtimes path m;
                read s path))
      (files s)
  in
  s.scanned_files <- read_files;
  Extract.to_list s.candidates

(* Reads only the given files and returns the candidates not seen before. *)
let scan_files s changed =
  let before = s.candidates.size in
  List.iter
    (fun path ->
      if is_file_path path then begin
        (match mtime path with Some m -> Hashtbl.replace s.mtimes path m | None -> ());
        ignore (read s path)
      end)
    changed;
  List.filteri (fun i _ -> i >= before) (Extract.to_list s.candidates)

(* Every candidate seen so far. *)
let candidates s = Extract.to_list s.candidates
