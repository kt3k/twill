(* A small .gitignore matcher for source auto-detection (SPEC §13.2).

   Supports blank lines, `#` comments, `!` negation, trailing `/` for
   directories, anchoring with `/`, and the `*`, `**`, and `?` wildcards. *)

open Utils

type rule = {
  pattern : Str.regexp;
  negated : bool;
  directory_only : bool;
  (* Anchored patterns match the full path relative to the ignore file. *)
  anchored : bool;
}

type t = { rules : rule list }

let glob_to_regexp glob =
  let b = Buffer.create 64 in
  Buffer.add_string b "^";
  let n = String.length glob in
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
    | '\\' when !i + 1 < n ->
        incr i;
        Buffer.add_string b (Str.quote (String.make 1 glob.[!i]))
    | c -> Buffer.add_string b (Str.quote (String.make 1 c)));
    incr i
  done;
  Buffer.add_string b "$";
  Str.regexp (Buffer.contents b)

let trim_unescaped_trailing_space line =
  let n = String.length line in
  let e = ref n in
  while !e > 0 && (line.[!e - 1] = ' ' || line.[!e - 1] = '\t') do decr e done;
  if !e > 0 && !e < n && line.[!e - 1] = '\\' then incr e;
  String.sub line 0 !e

(* Parses the content of a .gitignore file. *)
let parse content =
  let rules =
    List.filter_map
      (fun line ->
        let line = trim_suffix line "\r" in
        if String.trim line = "" || has_prefix line "#" then None
        else begin
          let line = trim_unescaped_trailing_space line in
          let negated, line =
            if has_prefix line "!" then (true, after line 1)
            else if has_prefix line "\\!" || has_prefix line "\\#" then (false, after line 1)
            else (false, line)
          in
          let directory_only, line = if has_suffix line "/" then (true, trim_suffix line "/") else (false, line) in
          let anchored, line =
            if has_prefix line "/" then (true, after line 1) else (String.contains line '/', line)
          in
          if line = "" then None else Some { pattern = glob_to_regexp line; negated; directory_only; anchored }
        end)
      (String.split_on_char '\n' content)
  in
  { rules }

let size g = List.length g.rules

(* Whether [relative_path] (relative to the directory of the ignore file,
   using `/` separators) is ignored. *)
let ignores g relative_path is_directory =
  let basename = match String.rindex_opt relative_path '/' with Some i -> after relative_path (i + 1) | None -> relative_path in
  List.fold_left
    (fun ignored rule ->
      if rule.directory_only && not is_directory then ignored
      else
        let matched = Str.string_match rule.pattern (if rule.anchored then relative_path else basename) 0 in
        if matched then not rule.negated else ignored)
    false g.rules
