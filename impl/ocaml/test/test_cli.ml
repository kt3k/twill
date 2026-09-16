open Twill
open Harness
open Test_scanner

let ( / ) = Filename.concat

let project () =
  let root = temp_dir () in
  write_file (root / "input.css") "@import \"twill/theme.css\";\n@source not \"dist\";\n@twill utilities;\n";
  write_file (root / "index.html") "<div class=\"flex p-4 hover:underline\">";
  write_file (root / "dist" / "old.html") "<div class=\"hidden\">";
  root

type result = { code : int; stdout : string; stderr : string }

let run_cli ?(stdin = "") ?(stop = fun () -> false) args cwd =
  let args = if cwd <> "" then [ "--cwd"; cwd ] @ args else args in
  let out = Buffer.create 1024 and err = Buffer.create 256 in
  let io =
    { Cli.read_stdin = (fun () -> stdin); is_tty = false; stdin_eof = (fun () -> true); out = Buffer.add_string out;
      err = Buffer.add_string err; executable = Sys.executable_name; stop }
  in
  let code = Cli.main args io in
  { code; stdout = Buffer.contents out; stderr = Buffer.contents err }

let run () =
  let open Cli in
  check "defaults" (parse_args [] = default_options);
  check "full"
    (parse_args [ "build"; "-i"; "a.css"; "-o"; "b.css"; "--watch"; "always"; "--poll"; "500"; "-m"; "--optimize"; "--cwd"; "x"; "--silent" ]
    = { input = "a.css"; output = "b.css"; watch = Watch_always; poll = 0.5; minify = true; optimize = true; cwd = "x"; silent = true;
        help = false });
  let w = parse_args [ "-w"; "-i"; "a.css" ] in
  check "watch" (w.watch = Watch_on && w.input = "a.css");
  check "watch always" ((parse_args [ "--watch=always" ]).watch = Watch_always);
  check "poll default" ((parse_args [ "--poll" ]).poll = default_poll_interval);
  check "poll 100" ((parse_args [ "--poll=100" ]).poll = 0.1);
  let inline = parse_args [ "--input=in.css"; "--output=out.css" ] in
  check "inline" (inline.input = "in.css" && inline.output = "out.css");
  check "help" (parse_args [ "-h" ]).help;
  let fails args part = match parse_args args with exception Usage_error m -> Utils.contains m part | _ -> false in
  check "poll zero" (fails [ "--poll=0" ] "positive");
  check "unknown option" (fails [ "--nope" ] "Unknown option");
  check "missing value" (fails [ "-i" ] "Missing value");

  let sources = [ { Directives.base = "/p"; pattern = "./src/**/*"; negated = false } ] in
  check "assemble"
    (assemble_sources ~root:None ~sources "/p" "/p/in.css" "/bin/twill"
    = [ { Directives.base = "/p"; pattern = "**/*"; negated = false }; { base = "/p"; pattern = "./src/**/*"; negated = false };
        { base = "/bin"; pattern = "twill"; negated = true }; { base = "/p"; pattern = "in.css"; negated = false } ]);
  check "assemble none" (assemble_sources ~root:(Some Directives.Root_none) ~sources:[] "/p" "" "" = []);
  check "assemble rooted"
    (assemble_sources ~root:(Some (Directives.Root_entry { root_base = "/p"; root_pattern = "../app" })) ~sources:[] "/p" "" ""
    = [ { Directives.base = "/p"; pattern = "../app"; negated = false } ]);
  equal "minify" (minify_css ".a {\n  color: red !important;\n}\n@media (x) {\n  .b {\n    y: z;\n  }\n}\n")
    ".a{color:red!important;}@media (x){.b{y:z;}}";

  let root = project () in
  let r = run_cli [ "-i"; "input.css" ] root in
  equal "build code" (string_of_int r.code) "0";
  includes "build flex" r.stdout ".flex {\n  display: flex;\n}";
  includes "build p-4" r.stdout ".p-4 {\n  padding: calc(var(--spacing) * 4);\n}";
  includes "build hover" r.stdout ".hover\\:underline:hover";
  excludes "build excluded" r.stdout ".hidden";
  includes "banner" r.stderr "twill";
  includes "done" r.stderr "Done in";
  let to_file = run_cli [ "-i"; "input.css"; "-o"; "out/build.css"; "--silent" ] root in
  check "to file" (to_file.code = 0 && to_file.stdout = "" && to_file.stderr = "");
  includes "written" (Option.get (Scanner.read_file (root / "out" / "build.css"))) ".flex {";
  let minified = run_cli [ "-i"; "input.css"; "--minify"; "--silent" ] root in
  check "minified code" (minified.code = 0);
  includes "minified" minified.stdout ".flex{display:flex;}";

  let r = run_cli ~stdin:"@import \"twill/theme.css\";\n@twill utilities;" [ "-i"; "-"; "--silent" ] root in
  check "stdin code" (r.code = 0);
  includes "stdin" r.stdout ".flex {";
  let defaulted = run_cli [ "--silent" ] root in
  check "default code" (defaulted.code = 0);
  includes "default layers" defaulted.stdout "@layer theme, base, components, utilities;";
  includes "default flex" defaulted.stdout ".flex {";

  let missing = run_cli [ "-i"; "nope.css" ] root in
  check "missing code" (missing.code = 1);
  includes "missing" missing.stderr "does not exist";
  let same = run_cli [ "-i"; "input.css"; "-o"; "input.css" ] root in
  check "same code" (same.code = 1);
  includes "same" same.stderr "identical";
  write_file (root / "bad.css") ".a { color: red;";
  let bad = run_cli [ "-i"; "bad.css"; "--silent" ] root in
  check "bad code" (bad.code = 1);
  includes "bad" bad.stderr "Missing closing";
  let unknown = run_cli [ "--nope" ] "" in
  check "unknown code" (unknown.code = 1);
  includes "unknown" unknown.stderr "Unknown option";
  includes "unknown usage" unknown.stderr "Usage: twill";
  let help = run_cli [ "--help" ] "" in
  check "help code" (help.code = 0);
  includes "help usage" help.stderr "Usage: twill";

  (* Polling: one tick picks up a source change incrementally, and a
     stylesheet change triggers a full rebuild. *)
  let root = project () in
  let output = root / "out.css" in
  let ticks = ref 0 in
  let stop () =
    incr ticks;
    (match !ticks with
    | 1 -> write_file (root / "page.html") "<i class=\"hidden\">"
    | 2 ->
        includes "poll incremental" (Option.get (Scanner.read_file output)) ".hidden {";
        write_file (root / "input.css") "@import \"twill/theme.css\";\n@theme { --color-brand: blue; }\n@twill utilities;\n";
        write_file (root / "page.html") "<i class=\"hidden bg-brand\">";
        let future = Unix.gettimeofday () +. 5.0 in
        Unix.utimes (root / "input.css") future future;
        Unix.utimes (root / "page.html") future future
    | _ -> ());
    !ticks > 3
  in
  let r = run_cli ~stop [ "-i"; "input.css"; "-o"; "out.css"; "--poll"; "10"; "--silent" ] root in
  check "poll code" (r.code = 0);
  includes "poll full rebuild" (Option.get (Scanner.read_file output)) ".bg-brand {";
  equal "poll stderr" r.stderr "";

  (* Watch mode: a change batch is handled, and stdin end of file stops it. *)
  let root = project () in
  let output = root / "out.css" in
  let ticks = ref 0 in
  let stop () =
    incr ticks;
    (match !ticks with
    | 1 ->
        write_file (root / "page.html") "<i class=\"underline\">";
        let future = Unix.gettimeofday () +. 5.0 in
        Unix.utimes (root / "page.html") future future
    | _ -> ());
    !ticks > 2
  in
  let r = run_cli ~stop [ "-i"; "input.css"; "-o"; "out.css"; "--watch=always"; "--silent" ] root in
  check "watch code" (r.code = 0);
  includes "watch change" (Option.get (Scanner.read_file output)) ".underline {";
  let r = run_cli [ "-i"; "input.css"; "-o"; "out.css"; "--watch"; "--silent" ] root in
  check "watch exits on stdin eof" (r.code = 0)
