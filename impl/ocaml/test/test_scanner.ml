open Twill
open Harness

let rec mkdir_p dir =
  if dir = "" || dir = "/" || Sys.file_exists dir then ()
  else begin
    mkdir_p (Filename.dirname dir);
    Unix.mkdir dir 0o755
  end

let write_file path content =
  mkdir_p (Filename.dirname path);
  Out_channel.with_open_bin path (fun oc -> output_string oc content)

let temp_dir () =
  let path = Filename.temp_file "twill" "" in
  Sys.remove path;
  Unix.mkdir path 0o755;
  path

let ( / ) = Filename.concat

let scan_fixture () =
  let root = temp_dir () in
  write_file (root / ".gitignore") "ignored/\n*.tmp\n";
  write_file (root / "index.html") "<div class=\"flex p-4\"></div>";
  write_file (root / "src" / "app.js") "el.className = \"hover:underline\";";
  write_file (root / "src" / "sub" / "page.html") "<p class=\"md:grid\">";
  write_file (root / "src" / "notes.tmp") "tmp-only";
  write_file (root / "styles.scss") ".scss-only {}";
  write_file (root / "logo.png") "png-only";
  write_file (root / "package-lock.json") "lock-only";
  write_file (root / ".env") "env-only";
  write_file (root / "node_modules" / "pkg" / "x.js") "nm-only";
  write_file (root / "vendor" / "v.html") "vendor-only";
  write_file (root / "ignored" / "i.html") "ignored-only";
  root

let source ?(negated = false) base pattern : Directives.source_entry = { base; pattern; negated }

let run () =
  let root = scan_fixture () in
  let scanner = Scanner.create [ source root "**/*" ] in
  let candidates = Scanner.scan scanner in
  List.iter (fun c -> check ("auto " ^ c) (List.mem c candidates)) [ "flex"; "p-4"; "hover:underline"; "md:grid"; "vendor-only" ];
  List.iter
    (fun c -> check ("auto excludes " ^ c) (not (List.mem c candidates)))
    [ "tmp-only"; "scss-only"; "png-only"; "lock-only"; "env-only"; "nm-only"; "ignored-only" ];
  equal "scanned files" (string_of_int (List.length scanner.scanned_files)) "4";

  let scanner =
    Scanner.create
      [ source root "**/*"; source root "node_modules/**/*.js"; source root "./ignored/*.html"; source ~negated:true root "vendor" ]
  in
  let candidates = Scanner.scan scanner in
  check "explicit node_modules" (List.mem "nm-only" candidates);
  check "explicit ignored" (List.mem "ignored-only" candidates);
  check "negated vendor" (not (List.mem "vendor-only" candidates));
  check "still flex" (List.mem "flex" candidates);

  let candidates = Scanner.scan (Scanner.create [ source root "src/**/*.{js,html}" ]) in
  check "brace js" (List.mem "hover:underline" candidates);
  check "brace html" (List.mem "md:grid" candidates);
  check "brace tmp" (not (List.mem "tmp-only" candidates));
  check "brace flex" (not (List.mem "flex" candidates));

  let candidates =
    Scanner.scan (Scanner.create [ source root "src"; source root "./index.html"; source ~negated:true root "src/sub/*.html" ])
  in
  check "dir source" (List.mem "hover:underline" candidates);
  check "file source" (List.mem "flex" candidates);
  check "negated glob" (not (List.mem "md:grid" candidates));
  check "outside" (not (List.mem "vendor-only" candidates));

  let scanner = Scanner.create [ source root "**/*" ] in
  ignore (Scanner.scan scanner);
  let again = Scanner.scan scanner in
  check "no rescans" (scanner.scanned_files = []);
  check "still has flex" (List.mem "flex" again);
  let changed = root / "src" / "app.js" in
  write_file changed "el.className = \"hover:underline new-one\";";
  let future = Unix.gettimeofday () +. 5.0 in
  Unix.utimes changed future future;
  let third = Scanner.scan scanner in
  check "rescanned changed" (scanner.scanned_files = [ changed ]);
  check "new candidate" (List.mem "new-one" third);
  let fresh = root / "fresh.html" in
  write_file fresh "<i class=\"fresh-one flex\">";
  check "scan files" (Scanner.scan_files scanner [ fresh; root / "missing.html" ] = [ "fresh-one" ]);
  check "scan files again" (Scanner.scan_files scanner [ fresh ] = []);
  check "candidates" (List.mem "fresh-one" (Scanner.candidates scanner));

  let g = Gitignore.parse "# comment\n\nbuild/\n*.log\n!keep.log\n/root.txt\ndocs/*.md\nfoo?\n**/deep\n" in
  equal "gitignore size" (string_of_int (Gitignore.size g)) "7";
  List.iter
    (fun (path, is_dir, want) -> check ("ignores " ^ path) (Gitignore.ignores g path is_dir = want))
    [ ("build", true, true); ("build", false, false); ("a/b/build", true, true); ("x.log", false, true); ("sub/x.log", false, true);
      ("keep.log", false, false); ("root.txt", false, true); ("sub/root.txt", false, false); ("docs/a.md", false, true);
      ("docs/sub/a.md", false, false); ("food", false, true); ("fooddd", false, false); ("a/b/deep", false, true);
      ("deep", false, true) ]
