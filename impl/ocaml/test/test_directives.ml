open Twill
open Ast
open Harness

(* Serves the built-in stylesheets and an in-memory file map. *)
let memory_loader files id base =
  match Builtin.resolve_builtin id base with
  | Some l -> l
  | None ->
      let path = base ^ "/" ^ Utils.trim_prefix id "./" in
      let content = match List.assoc_opt path files with Some c -> c | None -> Twill_error.failf "Cannot find %s" path in
      { Builtin.path; base = String.sub path 0 (String.rindex path '/'); content }

let import_and_collect ?(files = []) css =
  let th = Theme.create () in
  let ast = ref [ context (ctx_of_list [ ("base", "/root") ]) ~nodes:(Parser.parse css) ] in
  ignore (Import.substitute_at_imports ast "/root" (memory_loader files));
  let state = Directives.collect ast th in
  (th, !ast, state)

let ser = Serializer.serialize

let run () =
  (match Import.parse_import_params "\"a.css\" layer(base) supports(display: grid) screen and (min-width: 1px)" with
  | Some p -> check "import params" (p = { Import.uri = "a.css"; layer = Some "base"; media = Some "screen and (min-width: 1px)"; supports = Some "display: grid" })
  | None -> check "import params" false);
  (match Import.parse_import_params "\"twill\" important theme(reference) prefix(tw)" with
  | Some p -> check "import params 2" (p = { Import.uri = "twill"; layer = None; media = Some "important theme(reference) prefix(tw)"; supports = None })
  | None -> check "import params 2" false);
  List.iter
    (fun input -> check ("untouched " ^ input) (Import.parse_import_params input = None))
    [ "url(a.css)"; "\"data:text/css,a\""; "\"https://example.com/a.css\""; "a.css" ];
  List.iter
    (fun input -> expect_error ("import order " ^ input) (fun () -> Import.parse_import_params input) "must appear")
    [ "\"a.css\" supports(x) layer(base)"; "\"a.css\" screen layer(base)" ];
  let _, ast, _ = import_and_collect ~files:[ ("/root/a.css", ".a { color: red }") ] "@import \"./a.css\" layer(base) supports(display: grid) print;" in
  equal "import wrapping" (ser ast) "@supports (display: grid) {\n  @media print {\n    @layer base {\n      .a {\n        color: red;\n      }\n    }\n  }\n}\n";
  let _, ast, _ =
    import_and_collect
      ~files:[ ("/root/a.css", "@import \"./sub/b.css\";"); ("/root/sub/b.css", "@import \"./c.css\";"); ("/root/sub/c.css", ".c { color: red }") ]
      "@import \"./a.css\";"
  in
  equal "nested imports" (ser ast) ".c {\n  color: red;\n}\n";
  let css = "@import url(a.css);\n@import \"https://example.com/a.css\";\n@import \"data:text/css,a\";\n" in
  let _, ast, _ = import_and_collect css in
  equal "external imports" (ser ast) css;
  expect_error "recursion"
    (fun () ->
      let loop = ref [ context (ctx_of_list [ ("base", "/root") ]) ~nodes:(Parser.parse "@import \"./a.css\";") ] in
      Import.substitute_at_imports loop "/root" (memory_loader [ ("/root/a.css", "@import \"./a.css\";") ]))
    "recursion";
  let _, ast, state = import_and_collect "@import \"twill\";" in
  let css = ser ast in
  check "builtin prefix" (Utils.has_prefix css "@layer theme, base, components, utilities;\n@layer theme {\n");
  check "builtin utilities node" (Utils.contains css "@layer utilities {\n  @twill utilities;\n}\n" && state.utilities_node <> None);
  check "relative builtin" (Builtin.resolve_builtin "./theme.css" "/root" = None);
  equal "builtin relative" (Option.get (Builtin.resolve_builtin "./theme.css" "twill:")).path "twill:theme.css";
  equal "builtin preflight" (Option.get (Builtin.resolve_builtin "twill/preflight.css" "/root")).path "twill:preflight.css";
  expect_error "unknown builtin" (fun () -> Builtin.resolve_builtin "./nope.css" "twill:") "Unknown built-in";
  List.iter
    (fun name ->
      match Builtin.builtin_css name with
      | Some content -> ignore (Parser.parse content); check ("builtin parses " ^ name) true
      | None -> check ("builtin " ^ name) false)
    [ "index.css"; "theme.css"; "preflight.css"; "utilities.css" ];
  let th, ast, state =
    import_and_collect
      "@theme {\n --color-black: #000;\n /* comment */\n --breakpoint-md: 768px;\n @keyframes spin { to { transform: rotate(360deg) } }\n}\n@theme { --color-white: #fff; }"
  in
  equal "theme black" (Option.get (Theme.entry th "--color-black")).value "#000";
  equal "theme white" (Option.get (Theme.entry th "--color-white")).value "#fff";
  check "theme keyframes" (List.length (Theme.keyframes th) = 1);
  equal "theme output" (ser ast) ":root, :host {\n}\n";
  check "first theme rule" (match state.first_theme_rule with Some r -> r.selector = ":root, :host" | None -> false);
  check "theme feature" (Features.has state.features Features.at_theme);
  let th, _, _ =
    import_and_collect
      "@theme reference { --a: 1; }\n@theme inline { --b: 2; }\n@theme default { --c: 3; }\n@theme static { --d: 4; }\n@theme prefix(tw) { --e: 5; }\n@theme { --container-1\\.5: 1.5rem; }"
  in
  check "theme options"
    (Theme.options th "--a" = Theme.reference && Theme.options th "--b" = Theme.inline && Theme.options th "--c" = Theme.default
    && Theme.options th "--d" = Theme.static && th.prefix = "tw" && Theme.has th "--container-1.5");
  let _, ast, _ = import_and_collect "@theme reference { --a: 1; }\n@theme { --b: 2; }" in
  equal "reference then theme" (ser ast) ":root, :host {\n}\n";
  List.iter
    (fun (css, msg) -> expect_error ("directive error " ^ css) (fun () -> import_and_collect css) msg)
    [ ("@theme prefix(Tw) {}", "prefix"); ("@theme { .foo { color: red; } }", ".foo {"); ("@theme { color: red; }", "@theme");
      ("@source \"./a\" { color: red; }", "body"); (".a { @source \"./a\"; }", "nested"); ("@media x { @source \"./a\"; }", "nested");
      ("@source ./a;", "quoted"); ("@source inline(p-4);", "quoted"); ("@twill utilities source(../app);", "quoted") ];
  let _, ast, state =
    import_and_collect
      "@source \"./src/**/*.html\";\n@source not '../vendor/**';\n@source inline(\"p-{1..3} {hover:,}flex\");\n@source not inline(\"bg-red-{100,200}\");"
  in
  check "sources"
    (state.sources
    = [ { Directives.base = "/root"; pattern = "./src/**/*.html"; negated = false }; { base = "/root"; pattern = "../vendor/**"; negated = true } ]);
  check "inline candidates" (state.inline_candidates = [ "p-1"; "p-2"; "p-3"; "hover:flex"; "flex" ]);
  check "ignored candidates" (state.ignored_candidates = [ "bg-red-100"; "bg-red-200" ]);
  equal "sources output" (ser ast) "";
  let _, ast, state = import_and_collect "@twill utilities;\n@twill utilities;" in
  equal "utilities once" (ser ast) "@twill utilities;\n";
  check "root none" (state.root = None && Features.has state.features Features.utilities);
  let _, _, state = import_and_collect "@twill utilities source(none);" in
  check "source none" (state.root = Some Directives.Root_none);
  let _, _, state = import_and_collect "@twill utilities source(\"../app\");" in
  check "source path" (state.root = Some (Directives.Root_entry { root_base = "/root"; root_pattern = "../app" }));
  let _, ast, state = import_and_collect "@media reference { @twill utilities; }" in
  check "reference utilities" (state.utilities_node = None && ser ast = "");
  let th, ast, _ = import_and_collect ~files:[ ("/root/a.css", "@theme { --a: 1; } .x { color: red; }") ] "@reference \"./a.css\";" in
  check "reference option" (Theme.options th "--a" land Theme.reference <> 0);
  check "reference structure"
    (ast
    = [ context (ctx_of_list [ ("base", "/root") ])
          ~nodes:[ context ctx_empty
                     ~nodes:[ context (ctx_of_list [ ("reference", "true") ])
                                ~nodes:[ context (ctx_of_list [ ("base", "/root") ]) ~nodes:[ style_rule ".x" ~nodes:[ decl "color" "red" ] ] ] ] ] ]);
  let th, _, state = import_and_collect ~files:[ ("/root/a.css", "@theme { --a: 1; }") ] "@import \"./a.css\" theme(static) prefix(tw) important;" in
  check "import theme options" (Theme.options th "--a" = Theme.static && th.prefix = "tw" && state.important);
  expect_error "theme(reference) error"
    (fun () -> import_and_collect ~files:[ ("/root/a.css", "@theme { --a: 1; } .x { color: red; }") ] "@import \"./a.css\" theme(reference);")
    "theme(reference)";
  let _, _, state = import_and_collect ~files:[ ("/root/sub/a.css", "@twill utilities;") ] "@import \"./sub/a.css\" source(\"../app\");" in
  check "import source" (state.root = Some (Directives.Root_entry { root_base = "/root"; root_pattern = "../app" }));
  check "import source params" (match state.utilities_node with Some n -> n.params = "utilities source(\"../app\")" | None -> false);
  let _, ast, _ = import_and_collect ~files:[ ("/root/a.css", ".x { color: red; }") ] "@import \"./a.css\" print important;" in
  equal "import media remaining" (ser ast) "@media print {\n  .x {\n    color: red;\n  }\n}\n";
  let th, ast, state =
    import_and_collect "@import \"twill\";\n@theme {\n --color-*: initial;\n --color-primary: oklch(0.6 0.2 250);\n --breakpoint-3xl: 120rem;\n}"
  in
  let get value ns = match Theme.resolve_value th (if value = "" then None else Some value) ns with Some v -> v | None -> "<nil>" in
  equal "builtin spacing" (get "" [ "--spacing" ]) "0.25rem";
  equal "builtin md" (get "md" [ "--breakpoint" ]) "48rem";
  equal "builtin 3xl" (get "3xl" [ "--breakpoint" ]) "120rem";
  equal "builtin container" (get "md" [ "--container" ]) "28rem";
  equal "builtin text" (get "lg" [ "--text" ]) "1.125rem";
  equal "builtin weight" (get "bold" [ "--font-weight" ]) "700";
  equal "builtin radius" (get "lg" [ "--radius" ]) "0.5rem";
  equal "builtin duration" (get "" [ "--default-transition-duration" ]) "150ms";
  equal "builtin cleared" (get "red-500" [ "--color" ]) "<nil>";
  equal "builtin primary" (get "primary" [ "--color" ]) "oklch(0.6 0.2 250)";
  check "builtin keyframes" (List.map (fun (k : at_rule) -> k.params) (Theme.keyframes th) = [ "spin"; "ping"; "pulse"; "bounce" ]);
  check "builtin utilities" (state.utilities_node <> None);
  let css = ser ast in
  check "builtin theme rule" ((not (Utils.contains css "@theme")) && Utils.contains css "@layer theme {\n  :root, :host {\n  }\n}")
