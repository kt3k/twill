(* The command-line interface (SPEC §14). *)

open Utils

let usage =
  "Usage: twill [build] [options]\n\n\
   Options:\n\
  \  -i, --input <path>     Entry stylesheet (\"-\" reads stdin; default: @import 'twill';)\n\
  \  -o, --output <path>    Output file (\"-\" writes stdout; default: -)\n\
  \  -w, --watch [always]   Rebuild on changes (\"always\" keeps watching after stdin closes)\n\
  \      --poll [ms]        Poll for changes instead of using filesystem events (default: 250)\n\
  \  -m, --minify           Optimize and minify the output\n\
  \      --optimize         Optimize without minifying\n\
  \      --cwd <dir>        Base directory (default: .)\n\
  \      --silent           Suppress everything except errors\n\
  \  -h, --help             Show this help\n"

let default_input = "@import 'twill';\n"

(* The --poll interval in seconds when none is given. *)
let default_poll_interval = 0.25

(* How often watch mode checks the watched trees for changes. Watch mode
   detects change batches by comparing modification times. *)
let watch_interval = 0.1

type watch_mode = Watch_off | Watch_on | Watch_always

type options = {
  (* The entry stylesheet; "" means the default input. *)
  input : string;
  output : string;
  watch : watch_mode;
  (* The polling interval in seconds; zero disables polling. *)
  poll : float;
  minify : bool;
  optimize : bool;
  cwd : string;
  silent : bool;
  help : bool;
}

let default_options =
  { input = ""; output = "-"; watch = Watch_off; poll = 0.0; minify = false; optimize = false; cwd = "."; silent = false; help = false }

let is_digits s = s <> "" && for_all_chars is_digit s

exception Usage_error of string

(* Parses command-line arguments. *)
let parse_args argv =
  let argv = Array.of_list argv in
  let n = Array.length argv in
  let o = ref default_options in
  let poll_set = ref false in
  let i = ref 0 in
  let value name =
    if !i + 1 >= n then raise (Usage_error ("Missing value for " ^ name));
    incr i;
    argv.(!i)
  in
  while !i < n do
    let arg = argv.(!i) in
    let name, inline =
      if has_prefix arg "-" then
        match String.index_opt arg '=' with Some eq -> (String.sub arg 0 eq, Some (after arg (eq + 1))) | None -> (arg, None)
      else (arg, None)
    in
    let take () = match inline with Some v -> v | None -> value name in
    (match name with
    | "build" -> if has_prefix arg "-" then raise (Usage_error ("Unknown option: " ^ arg))
    | "-i" | "--input" -> o := { !o with input = take () }
    | "-o" | "--output" -> o := { !o with output = take () }
    | "--cwd" -> o := { !o with cwd = take () }
    | "-w" | "--watch" -> (
        o := { !o with watch = Watch_on };
        match inline with
        | Some v -> if v <> "always" then raise (Usage_error ("Invalid --watch value: " ^ v)) else o := { !o with watch = Watch_always }
        | None -> if !i + 1 < n && argv.(!i + 1) = "always" then (o := { !o with watch = Watch_always }; incr i))
    | "--poll" ->
        poll_set := true;
        let ms =
          match inline with
          | Some v -> if not (is_digits v) then raise (Usage_error ("Invalid --poll interval: " ^ v)) else v
          | None -> if !i + 1 < n && is_digits argv.(!i + 1) then (incr i; argv.(!i)) else ""
        in
        o := { !o with poll = (if ms = "" then default_poll_interval else float_of_string ms /. 1000.0) }
    | "-m" | "--minify" -> o := { !o with minify = true }
    | "--optimize" -> o := { !o with optimize = true }
    | "--silent" -> o := { !o with silent = true }
    | "-h" | "--help" -> o := { !o with help = true }
    | _ -> if has_prefix arg "-" then raise (Usage_error ("Unknown option: " ^ arg)));
    incr i
  done;
  if !poll_set && !o.poll <= 0.0 then raise (Usage_error "The --poll interval must be a positive number of milliseconds.");
  !o

(* A loader resolving relative ids against the filesystem and recording
   every loaded path as a full-rebuild path. *)
let new_loader (full_rebuild_paths : (string, unit) Hashtbl.t) : Builtin.loader =
 fun id base ->
  match Builtin.resolve_builtin id base with
  | Some loaded -> loaded
  | None ->
      if Builtin.is_builtin_path base then Twill_error.failf "Cannot import `%s` from a built-in stylesheet." id;
      let path = Scanner.join base id in
      let content =
        match Scanner.read_file path with Some c -> c | None -> Twill_error.failf "Cannot read `%s`: no such file." path
      in
      Hashtbl.replace full_rebuild_paths path ();
      { Builtin.path; base = Filename.dirname path; content }

(* Assembles the source set (SPEC §13.1). [input_path] and [executable]
   may be empty. *)
let assemble_sources ~(root : Directives.source_root option) ~(sources : Directives.source_entry list) cwd input_path executable =
  let auto : Directives.source_entry list =
    match root with
    | Some Directives.Root_none -> []
    | None -> [ { base = cwd; pattern = "**/*"; negated = false } ]
    | Some (Directives.Root_entry { root_base; root_pattern }) -> [ { base = root_base; pattern = root_pattern; negated = false } ]
  in
  let exe : Directives.source_entry list =
    if executable = "" then []
    else [ { base = Filename.dirname executable; pattern = Filename.basename executable; negated = true } ]
  in
  let input : Directives.source_entry list =
    if input_path = "" then [] else [ { base = Filename.dirname input_path; pattern = Filename.basename input_path; negated = false } ]
  in
  auto @ sources @ exe @ input

(* Minifies CSS by re-serializing it compactly. *)
let minify_css css = Serializer.serialize_compact (Parser.parse css)

let format_duration seconds = if seconds < 1.0 then Printf.sprintf "%dms" (int_of_float (seconds *. 1000.0)) else Printf.sprintf "%.2fs" seconds

(* The process environment the runner talks to. *)
type io = {
  (* Reads all of standard input. *)
  read_stdin : unit -> string;
  (* Whether standard input is a terminal. *)
  is_tty : bool;
  (* Whether standard input has reached end of file (non-blocking). *)
  stdin_eof : unit -> bool;
  out : string -> unit;
  err : string -> unit;
  (* The running executable's path, or "". *)
  executable : string;
  (* Whether watch and polling loops should stop. *)
  stop : unit -> bool;
}

type state = { compiler : Compile.t; scanner : Scanner.t; mutable full_rebuild_paths : (string, unit) Hashtbl.t }

type runner = {
  options : options;
  io : io;
  cwd : string;
  input_path : string;
  output_path : string;
  mutable state : state option;
  mutable previous : (string * string) option;
  mutable last_stdout : string option;
}

let describe = function
  | Twill_error.Error _ as e -> Twill_error.message e
  | Sys_error s -> s
  | Failure s -> s
  | Unix.Unix_error (e, fn, arg) -> Printf.sprintf "%s: %s %s" (Unix.error_message e) fn arg
  | e -> Printexc.to_string e

let resolve_in cwd path = if Filename.is_relative path then Scanner.clean (Filename.concat cwd path) else Scanner.clean path

let rec mkdir_p dir =
  if dir = "" || dir = "." || dir = "/" || Sys.file_exists dir then ()
  else begin
    mkdir_p (Filename.dirname dir);
    try Unix.mkdir dir 0o755 with Unix.Unix_error (Unix.EEXIST, _, _) -> ()
  end

let write_file path content =
  mkdir_p (Filename.dirname path);
  Out_channel.with_open_bin path (fun oc -> output_string oc content)

let read_input r =
  if r.input_path <> "" then
    match Scanner.read_file r.input_path with Some c -> c | None -> Twill_error.failf "Cannot read `%s`." r.input_path
  else if r.options.input = "-" then r.io.read_stdin ()
  else default_input

let create_state r =
  let css = read_input r in
  let full_rebuild_paths = Hashtbl.create 8 in
  let base = if r.input_path <> "" then (Hashtbl.replace full_rebuild_paths r.input_path (); Filename.dirname r.input_path) else r.cwd in
  let compiler = Compile.compile ~base ~load:(new_loader full_rebuild_paths) css in
  let sources = assemble_sources ~root:compiler.root ~sources:compiler.sources r.cwd r.input_path r.io.executable in
  { compiler; scanner = Scanner.create sources; full_rebuild_paths }

let state r = match r.state with Some s -> s | None -> assert false

(* Writes the CSS to the output (SPEC §14.5). *)
let write r css =
  let output =
    if r.options.minify || r.options.optimize then
      match r.previous with Some (prev_css, prev_written) when prev_css = css -> prev_written | _ -> minify_css css
    else css
  in
  r.previous <- Some (css, output);
  if r.output_path <> "" then write_file r.output_path output
  else if r.last_stdout <> Some output then r.io.out output;
  r.last_stdout <- Some output

let timed r f =
  let start = Unix.gettimeofday () in
  f ();
  if not r.options.silent then r.io.err ("Done in " ^ format_duration (Unix.gettimeofday () -. start) ^ "\n")

let report r e = r.io.err (describe e ^ "\n")

(* Re-reads the input and recreates the compiler and scanner. *)
let full_rebuild r =
  let previous = state r in
  match create_state r with
  | exception e ->
      (* Keep the previous dependency list so a later change to a deleted
         dependency still triggers a rebuild. *)
      ignore previous;
      raise e
  | next ->
      let candidates = Scanner.scan next.scanner in
      r.state <- Some next;
      write r (Compile.build next.compiler candidates)

let poll_tick r =
  let st = state r in
  let candidates = Scanner.scan st.scanner in
  let files = List.filter (fun f -> f <> r.output_path) st.scanner.scanned_files in
  if files = [] then ()
  else if List.exists (fun f -> Hashtbl.mem st.full_rebuild_paths f) files then timed r (fun () -> full_rebuild r)
  else if candidates = [] then ()
  else timed r (fun () -> write r (Compile.build st.compiler candidates))

(* Polling watch mode (SPEC §14.4). *)
let poll r =
  while not (r.io.stop ()) do
    Unix.sleepf r.options.poll;
    try poll_tick r with e -> report r e
  done

let snapshot r =
  let st = state r in
  let snap = Hashtbl.create 256 in
  let record path = Hashtbl.replace snap path (match Scanner.mtime path with Some m -> m | None -> -1.0) in
  List.iter record (Scanner.files st.scanner);
  Hashtbl.iter (fun path () -> if not (Builtin.is_builtin_path path) then record path) st.full_rebuild_paths;
  snap

let diff_snapshots previous current =
  let changed = ref [] in
  Hashtbl.iter
    (fun path m -> match Hashtbl.find_opt previous path with Some before when before = m -> () | _ -> changed := path :: !changed)
    current;
  Hashtbl.iter (fun path _ -> if not (Hashtbl.mem current path) then changed := path :: !changed) previous;
  List.sort compare !changed

let handle r paths =
  let st = state r in
  let relevant = List.filter (fun p -> p <> r.output_path) paths in
  if relevant = [] then ()
  else if List.exists (fun p -> Hashtbl.mem st.full_rebuild_paths p) relevant then timed r (fun () -> full_rebuild r)
  else
    let fresh = Scanner.scan_files st.scanner relevant in
    if fresh = [] then () else timed r (fun () -> write r (Compile.build st.compiler fresh))

(* Event-driven watch mode (SPEC §14.3). Change batches are detected by
   comparing modification times across the watched trees. *)
let watch r =
  let previous = ref (snapshot r) in
  let running = ref true in
  if r.options.watch <> Watch_always && r.io.stdin_eof () then running := false;
  while !running && not (r.io.stop ()) do
    Unix.sleepf watch_interval;
    if r.options.watch <> Watch_always && r.io.stdin_eof () then running := false
    else begin
      let current = snapshot r in
      let changed = diff_snapshots !previous current in
      previous := current;
      if changed <> [] then try handle r changed with e -> report r e
    end
  done

(* Runs the CLI and returns the exit code. *)
let main argv io =
  match parse_args argv with
  | exception Usage_error message ->
      io.err (message ^ "\n\n" ^ usage);
      1
  | options ->
      if options.help then (io.err usage; 0)
      else if argv = [] && io.is_tty then (io.err usage; 0)
      else begin
        let cwd = Scanner.abs_path options.cwd in
        let input_path = if options.input <> "" && options.input <> "-" then resolve_in cwd options.input else "" in
        let output_path = if options.output <> "-" then resolve_in cwd options.output else "" in
        let r = { options; io; cwd; input_path; output_path; state = None; previous = None; last_stdout = None } in
        if input_path <> "" && not (Scanner.is_file_path input_path) then begin
          io.err (Printf.sprintf "Specified input file `%s` does not exist.\n" options.input);
          1
        end
        else if input_path <> "" && output_path <> "" && input_path = output_path then begin
          io.err "Specified input file and output file are identical.\n";
          1
        end
        else begin
          if not options.silent then io.err "twill\n\n";
          match
            timed r (fun () ->
                let st = create_state r in
                r.state <- Some st;
                write r (Compile.build st.compiler (Scanner.scan st.scanner)))
          with
          | exception e ->
              report r e;
              1
          | () ->
              if options.watch = Watch_off && options.poll = 0.0 then 0
              else if options.poll > 0.0 then (poll r; 0)
              else (watch r; 0)
        end
      end
