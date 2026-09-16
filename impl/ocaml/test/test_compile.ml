open Twill
open Harness

let loader files = Test_directives.memory_loader files

(* Compiles and builds, returning the CSS. *)
let run ?(files = []) css candidates = Compile.build (Compile.compile ~base:"/root" ~load:(loader files) css) candidates

let expect_err css want = expect_error ("error " ^ want) (fun () -> Compile.compile ~base:"/root" ~load:(loader []) css) want

(* Like [run] but drops the leading `:root, :host { ... }` block. *)
let run_utilities css candidates =
  let output = run css candidates in
  if Utils.has_prefix output ":root, :host {\n" then
    match Utils.index_of output "\n}\n" with -1 -> output | e -> Utils.after output (e + 3)
  else output

let theme_css = "@import \"twill/theme.css\";\n@twill utilities;\n"

let escape_candidate c =
  let b = Buffer.create (String.length c) in
  String.iter
    (fun ch -> if Utils.is_alnum ch || ch = '_' || ch = '-' then Buffer.add_char b ch else (Buffer.add_char b '\\'; Buffer.add_char b ch))
    c;
  Buffer.contents b

let basic_document () =
  let output =
    run "@theme {\n  --color-black: #000;\n  --breakpoint-md: 768px;\n}\n@layer utilities {\n  @twill utilities;\n}"
      [ "dark:bg-black"; "hover:underline"; "md:grid"; "flex" ]
  in
  equal "basic document" output
    ":root, :host {\n  --color-black: #000;\n}\n@layer utilities {\n  .flex {\n    display: flex;\n  }\n  @media (hover: hover) {\n    .hover\\:underline:hover {\n      text-decoration-line: underline;\n    }\n  }\n  @media (width >= 768px) {\n    .md\\:grid {\n      display: grid;\n    }\n  }\n  @media (prefers-color-scheme: dark) {\n    .dark\\:bg-black {\n      background-color: var(--color-black);\n    }\n  }\n}\n"

let value_forms () =
  List.iter
    (fun (raw, want) ->
      let output = run_utilities theme_css [ raw ] in
      includes raw output ("." ^ escape_candidate raw ^ " {\n  " ^ want ^ "\n}\n"))
    [ ("p-4", "padding: calc(var(--spacing) * 4);"); ("p-1", "padding: var(--spacing);"); ("p-0", "padding: 0px;");
      ("p-px", "padding: 1px;"); ("-mt-2", "margin-top: calc(var(--spacing) * -2);"); ("w-1/2", "width: calc(1 / 2 * 100%);");
      ("w-[13px]", "width: 13px;"); ("w-(--my-w)", "width: var(--my-w);"); ("max-w-md", "max-width: var(--container-md);");
      ("bg-red-500", "background-color: var(--color-red-500);");
      ("bg-red-500/50", "background-color: color-mix(in oklab, var(--color-red-500) 50%, transparent);");
      ("bg-[#0088cc]", "background-color: #0088cc;"); ("bg-[url(/a_b.png)]", "background-image: url(/a_b.png);");
      ("bg-[length:10px_20px]", "background-size: 10px 20px;");
      ("text-lg", "font-size: var(--text-lg);\n  line-height: var(--tw-leading, var(--text-lg--line-height));");
      ("text-lg/8", "font-size: var(--text-lg);\n  line-height: calc(var(--spacing) * 8);");
      ("text-red-500", "color: var(--color-red-500);");
      ("font-bold", "--tw-font-weight: var(--font-weight-bold);\n  font-weight: var(--font-weight-bold);");
      ("rounded-lg", "border-radius: var(--radius-lg);"); ("rounded-full", "border-radius: calc(infinity * 1px);");
      ("border", "border-style: var(--tw-border-style);\n  border-width: 1px;");
      ("border-2", "border-style: var(--tw-border-style);\n  border-width: 2px;"); ("z-10", "z-index: 10;");
      ("-z-10", "z-index: calc(10 * -1);"); ("flex-1", "flex: 1;"); ("opacity-50", "opacity: 50%;");
      ("[mask-type:luminance]", "mask-type: luminance;"); ("[--my-var:1px]", "--my-var: 1px;");
      ("underline!", "text-decoration-line: underline !important;") ];
  includes "font-bold property" (run_utilities theme_css [ "font-bold" ])
    "@property --tw-font-weight {\n  syntax: \"*\";\n  inherits: false;\n}\n";
  includes "border property" (run_utilities theme_css [ "border" ])
    "@property --tw-border-style {\n  syntax: \"*\";\n  inherits: false;\n  initial-value: solid;\n}\n"

let variant_forms () =
  List.iter
    (fun (raw, want) -> includes raw (run_utilities theme_css [ raw ]) want)
    [ ("hover:flex", "@media (hover: hover) {\n  .hover\\:flex:hover {\n    display: flex;\n  }\n}\n");
      ("focus:flex", ".focus\\:flex:focus {\n  display: flex;\n}\n");
      ("sm:flex", "@media (width >= 40rem) {\n  .sm\\:flex {\n    display: flex;\n  }\n}\n");
      ("max-md:flex", "@media (width < 48rem) {\n  .max-md\\:flex {\n    display: flex;\n  }\n}\n");
      ("min-[600px]:flex", "@media (width >= 600px) {\n  .min-\\[600px\\]\\:flex {\n    display: flex;\n  }\n}\n");
      ("@md:flex", "@container (width >= 28rem) {\n  .\\@md\\:flex {\n    display: flex;\n  }\n}\n");
      ("@md/main:flex", "@container main (width >= 28rem) {\n  .\\@md\\/main\\:flex {\n    display: flex;\n  }\n}\n");
      ("group-hover:flex", "@media (hover: hover) {\n  .group-hover\\:flex:is(:where(.group):hover *) {\n    display: flex;\n  }\n}\n");
      ( "group-hover/item:flex",
        "@media (hover: hover) {\n  .group-hover\\/item\\:flex:is(:where(.group\\/item):hover *) {\n    display: flex;\n  }\n}\n" );
      ("peer-checked:flex", ".peer-checked\\:flex:is(:where(.peer):checked ~ *) {\n  display: flex;\n}\n");
      ("has-[>img]:flex", ".has-\\[\\>img\\]\\:flex:has(> img) {\n  display: flex;\n}\n");
      ("in-data-visible:flex", ":where(*[data-visible]) .in-data-visible\\:flex {\n  display: flex;\n}\n");
      ( "not-hover:flex",
        ".not-hover\\:flex:not(:hover) {\n  display: flex;\n}\n@media not all and (hover: hover) {\n  .not-hover\\:flex {\n    display: flex;\n  }\n}\n"
      );
      ("not-supports-grid:flex", "@supports not (grid: var(--tw)) {\n  .not-supports-grid\\:flex {\n    display: flex;\n  }\n}\n");
      ("data-[state=open]:flex", ".data-\\[state\\=open\\]\\:flex[data-state=\"open\"] {\n  display: flex;\n}\n");
      ("aria-checked:flex", ".aria-checked\\:flex[aria-checked=\"true\"] {\n  display: flex;\n}\n");
      ("nth-3:flex", ".nth-3\\:flex:nth-child(3) {\n  display: flex;\n}\n");
      ("[&_p]:flex", ".\\[\\&_p\\]\\:flex p {\n  display: flex;\n}\n");
      ( "[@media(width>=100px)]:flex",
        "@media (width>=100px) {\n  .\\[\\@media\\(width\\>\\=100px\\)\\]\\:flex {\n    display: flex;\n  }\n}\n" );
      ("*:flex", ":is(.\\*\\:flex > *) {\n  display: flex;\n}\n");
      ("before:block", ".before\\:block::before {\n  content: var(--tw-content);\n  display: block;\n}\n");
      ( "dark:hover:flex",
        "@media (prefers-color-scheme: dark) {\n  @media (hover: hover) {\n    .dark\\:hover\\:flex:hover {\n      display: flex;\n    }\n  }\n}\n"
      ) ]

let custom_utilities_and_apply () =
  let output =
    run_utilities
      "@import \"twill/theme.css\";\n@utility tab-* {\n  tab-size: --value(integer);\n  tab-size: --value(--tab-size-*);\n  tab-size: --value([integer]);\n}\n@utility content-auto {\n  content-visibility: auto;\n}\n.btn {\n  @apply rounded-lg px-4 py-2 hover:bg-red-500;\n}\n@twill utilities;"
      [ "tab-4"; "tab-[8]"; "content-auto" ]
  in
  (* tab-size has a position in the global property order while
     content-visibility does not, so tab-* sorts first (SPEC §11.3). *)
  equal "custom utilities and apply" output
    ".btn {\n  border-radius: var(--radius-lg);\n  padding-inline: calc(var(--spacing) * 4);\n  padding-block: calc(var(--spacing) * 2);\n}\n@media (hover: hover) {\n  .btn:hover {\n    background-color: var(--color-red-500);\n  }\n}\n.tab-4 {\n  tab-size: 4;\n}\n.tab-\\[8\\] {\n  tab-size: 8;\n}\n.content-auto {\n  content-visibility: auto;\n}\n"

let theme_customization () =
  let css =
    "@import \"twill\";\n@theme {\n  --color-*: initial;\n  --color-primary: oklch(0.6 0.2 250);\n  --breakpoint-3xl: 120rem;\n  --font-display: \"Inter\", sans-serif;\n}\n@custom-variant dark (&:where(.dark, .dark *));"
  in
  let output = run css [ "bg-primary"; "3xl:flex"; "font-display"; "bg-red-500"; "dark:flex" ] in
  includes "bg-primary" output ".bg-primary {\n    background-color: var(--color-primary);\n  }";
  includes "3xl" output "@media (width >= 120rem) {\n    .\\33 xl\\:flex {\n      display: flex;\n    }\n  }";
  includes "font-display" output ".font-display {\n    font-family: var(--font-display);\n  }";
  includes "custom dark" output ".dark\\:flex:where(.dark, .dark *) {\n    display: flex;\n  }";
  excludes "bg-red-500" output "bg-red-500";
  excludes "--color-red-500" output "--color-red-500";
  includes "--color-primary" output "--color-primary: oklch(0.6 0.2 250);"

let build_behavior () =
  let a = run theme_css [ "p-4"; "hover:flex"; "sm:p-2"; "flex"; "m-1" ] in
  let b = run theme_css [ "m-1"; "flex"; "sm:p-2"; "hover:flex"; "p-4" ] in
  equal "order independent" a b;
  let compiler = Compile.compile theme_css in
  let first = Compile.build compiler [ "flex"; "p-4" ] in
  let second = Compile.build compiler [ "p-4" ] in
  equal "stable" second first;
  let third = Compile.build compiler [ "flex"; "nope" ] in
  equal "invalid ignored" third first;
  let fourth = Compile.build compiler [ "m-2" ] in
  check "new candidate changes output" (fourth <> first);
  includes "m-2" fourth ".m-2";
  let compiler = Compile.compile theme_css in
  ignore (Compile.build compiler [ "flex" ]);
  let output = Compile.build compiler [ "hidden" ] in
  includes "accumulate flex" output ".flex {";
  includes "accumulate hidden" output ".hidden {";
  let plain = Compile.compile ".a { color: red; }" in
  equal "no features" (string_of_int plain.features) "0";
  equal "no features passthrough" (Compile.build plain [ "flex" ]) ".a { color: red; }";
  let themed = Compile.compile "@theme { --a: 1; } .a { color: var(--a); }" in
  check "at theme feature" (Features.has themed.features Features.at_theme);
  equal "themed" (Compile.build themed [ "flex" ]) ":root, :host {\n  --a: 1;\n}\n.a {\n  color: var(--a);\n}\n";
  let full = Compile.compile "@import \"twill\"; .a { @apply flex; } .b { @variant hover { color: red; } }" in
  check "features"
    (Features.has full.features Features.at_import
    && Features.has full.features Features.at_apply
    && Features.has full.features Features.variants
    && Features.has full.features Features.utilities
    && Features.has full.features Features.theme_function)

let important_flag () =
  let output = run "@import \"twill/theme.css\" important;\n@twill utilities;" [ "flex"; "font-bold" ] in
  includes "important flex" output "display: flex !important;";
  includes "important font" output "font-weight: var(--font-weight-bold) !important;";
  includes "inherits" output "inherits: false;\n";
  excludes "inherits important" output "inherits: false !important";
  let applied = run "@import \"twill/theme.css\" important;\n.a { @apply flex underline!; }\n@twill utilities;" [] in
  includes "apply important" applied ".a {\n  display: flex;\n  text-decoration-line: underline !important;\n}"

let theme_functions () =
  let output =
    run
      "@import \"twill/theme.css\";\n.a {\n  padding: --spacing(4);\n  margin: --spacing(0) --spacing(1);\n  color: --alpha(var(--color-red-500) / 50%);\n  width: --theme(--container-md);\n  height: --theme(--container-md inline);\n  gap: --theme(--nope, 1px, 2px);\n  font: theme(--text-lg);\n  border-color: --theme(--color-red-500/0.5);\n}\n@media (width >= --theme(--breakpoint-md)) {\n  .b { color: red; }\n}"
      []
  in
  includes "spacing" output "padding: calc(var(--spacing) * 4);";
  includes "spacing 0 1" output "margin: 0px var(--spacing);";
  includes "alpha" output "color: color-mix(in oklab, var(--color-red-500) 50%, transparent);";
  includes "theme" output "width: var(--container-md);";
  includes "theme inline" output "height: 28rem;";
  includes "theme fallback" output "gap: 1px, 2px;";
  includes "legacy theme" output "font: 1.125rem;";
  includes "theme alpha" output "border-color: color-mix(in oklab, var(--color-red-500) 50%, transparent);";
  includes "media theme" output "@media (width >= 48rem) {";
  includes "container used" output "--container-md: 28rem;";
  let output =
    run
      "@theme { --a: var(--b); --c: initial; --d: 1px; }\n.x { color: --theme(--a, red); background: --theme(--a inline, red); width: --theme(--d, initial); height: --theme(--c, 2px); }"
      []
  in
  includes "fallback var" output "color: var(--a, red);";
  includes "fallback inline" output "background: var(--b, red);";
  includes "fallback initial" output "width: var(--d);";
  includes "initial value" output "height: 2px;";
  expect_err ".a { padding: --spacing(); }" "--spacing";
  expect_err ".a { padding: --spacing(1, 2); }" "--spacing";
  expect_err ".a { padding: --spacing(4); }" "--spacing";
  expect_err ".a { color: --alpha(red); }" "--alpha";
  expect_err ".a { color: --theme(spacing); }" "--theme";
  expect_err ".a { color: --theme(--nope); }" "resolve";
  expect_err ".a { color: theme(--nope); }" "resolve";
  let output = run "@theme { --color-a: red; }\n@twill utilities;" [ "p-4"; "bg-a" ] in
  excludes "failed candidate" output "p-4";
  includes "bg-a" output ".bg-a {"

let apply () =
  let output =
    run "@import \"twill/theme.css\";\n.a { @apply flex hover:underline sm:p-2; }\n.b { @apply --my-mixin; }\n@apply flex;" []
  in
  includes "apply" output
    ".a {\n  display: flex;\n}\n@media (hover: hover) {\n  .a:hover {\n    text-decoration-line: underline;\n  }\n}\n@media (width >= 40rem) {\n  .a {\n    padding: calc(var(--spacing) * 2);\n  }\n}\n.b {\n  @apply --my-mixin;\n}\n@apply flex;\n";
  expect_err ".a { @apply --x flex; }" "mix";
  expect_err ".a { @apply flex { color: red; } }" "body";
  expect_err "@keyframes x { to { @apply flex; } }" "@keyframes";
  expect_err ".a { @apply nope; }" "empty";
  expect_err "@import \"twill/theme.css\"; .a { @apply nope; }" "unknown utility";
  expect_err "@import \"twill/theme.css\"; .a { @apply nope:flex; }" "unknown variant";
  expect_err "@import \"twill/theme.css\" prefix(tw); .a { @apply flex; }" "prefix";
  expect_err "@import \"twill/theme.css\"; @source not inline(\"flex\"); .a { @apply flex; }" "disabled";
  let output =
    run "@import \"twill/theme.css\";\n@utility foo { @apply bar p-1; }\n@utility bar { color: red; }\n.x { @apply foo; }\n@twill utilities;"
      [ "foo" ]
  in
  includes "apply custom" output ".x {\n  padding: var(--spacing);\n  color: red;\n}";
  includes "custom expanded" output ".foo {\n  padding: var(--spacing);\n  color: red;\n}";
  expect_err "@utility foo { @apply bar; }\n@utility bar { @apply foo; }" "circular";
  expect_err "@utility foo { @apply foo; }" "circular"

let custom_utilities () =
  let output =
    run
      "@import \"twill/theme.css\";\n@theme { --tab-size-github: 8; --leading-tight: 1.25; }\n@utility tab-* {\n  tab-size: --value(integer);\n  tab-size: --value(--tab-size-*);\n  tab-size: --value([integer]);\n}\n@utility aspect-* {\n  aspect-ratio: --value(ratio, --aspect-*, [ratio]);\n  width: --value(number);\n}\n@utility opacity-* {\n  opacity: --value(percentage);\n  opacity: --modifier(number);\n}\n@utility text-* {\n  font-size: --value(--text-*);\n  line-height: --value(--text-*--line-height);\n  line-height: --modifier(--leading-*);\n}\n@utility lit-* {\n  color: --value(\"red\", 'blue');\n}\n@utility any-* {\n  color: --value([*]);\n}\n@utility content-auto { content-visibility: auto; }\n@utility foo-1\\/2 { color: red; }\n@twill utilities;"
      [ "tab-4"; "tab-github"; "tab-[8]"; "tab-[foo]"; "tab-x"; "aspect-16/9"; "aspect-video"; "aspect-[4/3]"; "aspect-3";
        "aspect-16/9/2"; "opacity-50%"; "opacity-50%/2"; "opacity-50%/foo"; "text-lg"; "text-lg/tight"; "lit-red"; "lit-blue";
        "lit-green"; "any-[foo]"; "content-auto"; "foo-1/2" ]
  in
  includes "tab-4" output ".tab-4 {\n  tab-size: 4;\n}";
  includes "tab-github" output ".tab-github {\n  tab-size: var(--tab-size-github);\n}";
  includes "tab-[8]" output ".tab-\\[8\\] {\n  tab-size: 8;\n}";
  excludes "tab-[foo]" output "tab-\\[foo\\]";
  excludes "tab-x" output "tab-x";
  includes "aspect ratio" output ".aspect-16\\/9 {\n  aspect-ratio: 16 / 9;\n}";
  includes "aspect theme" output ".aspect-video {\n  aspect-ratio: var(--aspect-video);\n}";
  includes "aspect arbitrary" output ".aspect-\\[4\\/3\\] {\n  aspect-ratio: 4/3;\n}";
  includes "aspect number" output ".aspect-3 {\n  width: 3;\n}";
  excludes "aspect modifier" output "aspect-16\\/9\\/2";
  includes "opacity" output ".opacity-50\\% {\n  opacity: 50%;\n}";
  includes "opacity modifier" output ".opacity-50\\%\\/2 {\n  opacity: 50%;\n  opacity: 2;\n}";
  excludes "opacity bad modifier" output "opacity-50\\%\\/foo";
  includes "text" output ".text-lg {\n  font-size: var(--text-lg);\n  line-height: var(--text-lg--line-height);\n}";
  includes "text modifier" output
    ".text-lg\\/tight {\n  font-size: var(--text-lg);\n  line-height: var(--text-lg--line-height);\n  line-height: var(--leading-tight);\n}";
  includes "lit red" output ".lit-red {\n  color: red;\n}";
  includes "lit blue" output ".lit-blue {\n  color: blue;\n}";
  excludes "lit green" output "lit-green";
  includes "any" output ".any-\\[foo\\] {\n  color: foo;\n}";
  includes "content-auto" output ".content-auto {\n  content-visibility: auto;\n}";
  includes "foo-1/2" output ".foo-1\\/2 {\n  color: red;\n}";
  expect_err "@utility foo {}" "empty";
  expect_err "@utility foo* { color: red; }" "-*";
  expect_err "@utility fo*o { color: red; }" "end";
  expect_err "@utility Foo { color: red; }" "invalid utility name";
  expect_err "@utility foo- { color: red; }" "invalid utility name";
  expect_err ".a { @utility foo { color: red; } }" "nested"

let custom_variants () =
  let output =
    run
      "@import \"twill/theme.css\";\n@custom-variant dark (&:where(.dark, .dark *));\n@custom-variant hocus (&:hover, &:focus);\n@custom-variant wide (@media (width >= 100px), @supports (display: grid));\n@custom-variant mixed (&:hover, @media print);\n@twill utilities;"
      [ "dark:flex"; "hocus:flex"; "wide:flex"; "mixed:flex"; "not-hocus:flex" ]
  in
  includes "dark" output ".dark\\:flex:where(.dark, .dark *) {\n  display: flex;\n}";
  includes "hocus" output ".hocus\\:flex:hover, .hocus\\:flex:focus {\n  display: flex;\n}";
  includes "wide" output
    "@media (width >= 100px) {\n  .wide\\:flex {\n    display: flex;\n  }\n}\n@supports (display: grid) {\n  .wide\\:flex {\n    display: flex;\n  }\n}";
  includes "mixed" output ".mixed\\:flex:hover {\n  display: flex;\n}\n@media print {\n  .mixed\\:flex {\n    display: flex;\n  }\n}";
  includes "not-hocus" output ".not-hocus\\:flex:not(:hover), .not-hocus\\:flex:not(:focus) {";
  expect_err "@custom-variant foo (&:hover) { color: red; }" "both";
  expect_err "@custom-variant foo;" "no selector";
  expect_err "@custom-variant Foo (&:hover);" "invalid variant name";
  expect_err "@custom-variant foo (&:hover,);" "empty";
  expect_err ".a { @custom-variant foo (&:hover); }" "nested";
  let output =
    run
      "@import \"twill/theme.css\";\n@custom-variant theme-dark {\n  &:where([data-theme=\"dark\"], [data-theme=\"dark\"] *) {\n    @slot;\n  }\n}\n@custom-variant dark-hover {\n  @variant theme-dark {\n    &:hover {\n      @slot;\n    }\n  }\n}\n@custom-variant animated {\n  @keyframes spin-custom { to { transform: rotate(1turn); } }\n  animation: spin-custom 1s;\n  @slot;\n}\n@twill utilities;"
      [ "theme-dark:flex"; "dark-hover:flex"; "animated:flex" ]
  in
  includes "theme-dark" output ".theme-dark\\:flex:where([data-theme=\"dark\"], [data-theme=\"dark\"] *) {\n  display: flex;\n}";
  includes "dark-hover" output
    ".dark-hover\\:flex:where([data-theme=\"dark\"], [data-theme=\"dark\"] *):hover {\n  display: flex;\n}";
  includes "animated" output ".animated\\:flex {\n  animation: spin-custom 1s;\n  display: flex;\n}";
  includes "keyframes hoisted" output "@keyframes spin-custom {\n  to {\n    transform: rotate(1turn);\n  }\n}";
  expect_err "@custom-variant a { @variant b { @slot; } }\n@custom-variant b { @variant a { @slot; } }" "circular";
  let output =
    run "@import \"twill/theme.css\";\n@variant hocus (&:hover, &:focus);\n@variant dark { &:where(.dark, .dark *) { @slot; } }\n@twill utilities;"
      [ "hocus:flex"; "dark:flex" ]
  in
  includes "compat hocus" output ".hocus\\:flex:hover, .hocus\\:flex:focus {";
  includes "compat dark" output ".dark\\:flex:where(.dark, .dark *) {";
  excludes "compat removed" output "@variant"

let nested_variant () =
  let output =
    run "@import \"twill/theme.css\";\n.a {\n  color: red;\n  @variant hover { color: blue; }\n  @variant dark:hover, sm { color: green; }\n}" []
  in
  equal "nested variant" output
    ".a {\n  color: red;\n}\n@media (hover: hover) {\n  .a:hover {\n    color: blue;\n  }\n}\n@media (prefers-color-scheme: dark) {\n  @media (hover: hover) {\n    .a:hover {\n      color: green;\n    }\n  }\n}\n@media (width >= 40rem) {\n  .a {\n    color: green;\n  }\n}\n";
  expect_err ".a { @variant nope { color: red; } }" "unknown variant";
  expect_err "@import \"twill/theme.css\"; .a { @variant hover: { color: red; } }" "empty"

let count_occurrences s sub =
  let n = String.length s and m = String.length sub in
  let rec go i acc = if i + m > n then acc else if String.sub s i m = sub then go (i + m) (acc + 1) else go (i + 1) acc in
  go 0 0

let optimizer () =
  let output = run theme_css [ "font-bold"; "font-thin"; "leading-6" ] in
  equal "registration once" (string_of_int (count_occurrences output "@property --tw-font-weight")) "1";
  let registration = Utils.index_of output "@property" and last_rule = Utils.last_index_of output ".leading-6" in
  check "registrations after rules" (registration > last_rule);
  check "ends with block" (Utils.has_suffix (String.trim output) "}");
  let css =
    "@import \"twill/theme.css\";\n@theme { --color-keep: red; --keep-static: 1px; --chain-a: var(--chain-b); --chain-b: 2px; }\n@theme static { --always: 3px; }\n@twill utilities;"
  in
  let output = run css [ "bg-keep"; "[width:var(--chain-a)]" ] in
  includes "keep color" output "--color-keep: red;";
  includes "chain a" output "--chain-a: var(--chain-b);";
  includes "chain b" output "--chain-b: 2px;";
  includes "always" output "--always: 3px;";
  excludes "keep static pruned" output "--keep-static";
  excludes "red pruned" output "--color-red-500";
  excludes "keyframes pruned" output "@keyframes";
  let spin = run css [ "animate-spin" ] in
  includes "spin keyframes" spin "@keyframes spin {";
  excludes "ping pruned" spin "@keyframes ping";
  includes "animate spin" spin "--animate-spin: spin 1s linear infinite;";
  let marked = run css [ "--color-blue-500" ] in
  includes "marked variable" marked "--color-blue-500:";
  let empty = run "@layer theme, base;\n@import \"twill/theme.css\" layer(theme);\n@twill utilities;" [] in
  equal "empty layer" empty "@layer theme, base;\n";
  let output =
    run
      "@import \"twill/theme.css\";\n.a, .b {\n  color: red;\n  &:hover { color: blue; }\n  .c & { color: green; }\n  > .d { color: purple; }\n  @media (x) { color: orange; .e { color: black; } }\n  @supports (y) { @media (z) { & .f { color: white; } } }\n}\n@keyframes k { 50% { opacity: 0; } }"
      []
  in
  equal "flatten" output
    ".a, .b {\n  color: red;\n}\n:is(.a, .b):hover {\n  color: blue;\n}\n.c :is(.a, .b) {\n  color: green;\n}\n:is(.a, .b) > .d {\n  color: purple;\n}\n@media (x) {\n  .a, .b {\n    color: orange;\n  }\n  :is(.a, .b).e {\n    color: black;\n  }\n}\n@supports (y) {\n  @media (z) {\n    :is(.a, .b) .f {\n      color: white;\n    }\n  }\n}\n@keyframes k {\n  50% {\n    opacity: 0;\n  }\n}\n"

let misc () =
  let output = run "@reference \"twill\";\n.a { @apply flex p-4; }" [] in
  equal "reference" output ".a {\n  display: flex;\n  padding: calc(var(--spacing, 0.25rem) * 4);\n}\n";
  let output =
    run "@import \"twill/theme.css\" prefix(tw);\n@twill utilities;" [ "tw:flex"; "tw:bg-red-500"; "tw:group-hover:flex"; "flex" ]
  in
  includes "prefixed variable" output "--tw-color-red-500: oklch(63.7% 0.237 25.331);";
  includes "prefixed flex" output ".tw\\:flex {\n  display: flex;\n}";
  includes "prefixed bg" output ".tw\\:bg-red-500 {\n  background-color: var(--tw-color-red-500);\n}";
  includes "prefixed group" output ".tw\\:group-hover\\:flex:is(:where(.tw\\:group):hover *)";
  excludes "unprefixed" output "\n.flex";
  let output =
    run "@import \"twill/theme.css\";\n@source inline(\"p-{1,2}\");\n@source not inline(\"flex\");\n@twill utilities;" [ "flex"; "hidden" ]
  in
  includes "inline p-1" output ".p-1 {";
  includes "inline p-2" output ".p-2 {";
  includes "hidden" output ".hidden {";
  excludes "not inline" output ".flex";
  let output =
    run "/*! license */\n@import url(x.css);\n@import \"https://x/y.css\";\n@theme { --a: 1; }\n.a { color: var(--a); }" []
  in
  equal "license and external imports" output
    "/*! license */\n@import url(x.css);\n@import \"https://x/y.css\";\n:root, :host {\n  --a: 1;\n}\n.a {\n  color: var(--a);\n}\n";
  let output =
    run ~files:[ ("/root/theme.css", "@theme { --color-brand: blue; }") ]
      "@import \"./theme.css\";\n@import \"twill/theme.css\";\n@twill utilities;" [ "bg-brand" ]
  in
  includes "user file" output ".bg-brand {\n  background-color: var(--color-brand);\n}"

let run () =
  basic_document ();
  value_forms ();
  variant_forms ();
  custom_utilities_and_apply ();
  theme_customization ();
  build_behavior ();
  important_flag ();
  theme_functions ();
  apply ();
  custom_utilities ();
  custom_variants ();
  nested_variant ();
  optimizer ();
  misc ()
