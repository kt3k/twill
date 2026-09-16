open Twill
open Harness

let make_theme () =
  let th = Theme.create () in
  List.iter
    (fun (k, v) -> Theme.add th k v 0)
    [ ("--color-red-500", "red"); ("--color-blue-500", "blue"); ("--spacing", "0.25rem"); ("--text-lg", "1.125rem");
      ("--text-lg--line-height", "calc(1.75 / 1.125)"); ("--text-shadow-sm", "0 1px 1px black"); ("--font-sans", "ui-sans-serif");
      ("--font-weight-bold", "700"); ("--breakpoint-md", "48rem"); ("--container-1_5", "1.5rem") ];
  th

let resolve th value namespaces =
  match Theme.resolve th (if value = "" then None else Some value) namespaces 0 with Some v -> v | None -> "<nil>"

let run () =
  let th = make_theme () in
  equal "resolve color" (resolve th "red-500" [ "--color" ]) "var(--color-red-500)";
  equal "resolve namespace" (resolve th "" [ "--spacing" ]) "var(--spacing)";
  equal "resolve missing" (resolve th "missing" [ "--color" ]) "<nil>";
  equal "resolve fallthrough" (resolve th "red-500" [ "--text-color"; "--color" ]) "var(--color-red-500)";
  equal "resolve dot" (resolve th "1.5" [ "--container" ]) "var(--container-1_5)";
  equal "resolve ignored" (resolve th "shadow-sm" [ "--text" ]) "<nil>";
  equal "resolve ignored font" (resolve th "weight-bold" [ "--font" ]) "<nil>";
  equal "resolve weight" (resolve th "bold" [ "--font-weight" ]) "var(--font-weight-bold)";
  let th2 = Theme.create () in
  Theme.add th2 "--a" "1" Theme.inline;
  Theme.add th2 "--b" "2" Theme.reference;
  Theme.add th2 "--c" "3" 0;
  equal "inline" (resolve th2 "" [ "--a" ]) "1";
  equal "reference" (resolve th2 "" [ "--b" ]) "var(--b, 2)";
  equal "plain" (resolve th2 "" [ "--c" ]) "var(--c)";
  equal "force inline" (Option.get (Theme.resolve th2 None [ "--c" ] Theme.inline)) "3";
  let th = Theme.create () in
  Theme.add th "--color-a" "author" 0;
  Theme.add th "--color-a" "default" Theme.default;
  equal "default keeps author" (Option.get (Theme.entry th "--color-a")).value "author";
  Theme.add th "--color-b" "default" Theme.default;
  Theme.add th "--color-b" "author" 0;
  equal "author overrides default" (Option.get (Theme.entry th "--color-b")).value "author";
  Theme.add th "--color-c" "default-1" Theme.default;
  Theme.add th "--color-c" "default-2" Theme.default;
  equal "default overrides default" (Option.get (Theme.entry th "--color-c")).value "default-2";
  let th = make_theme () in
  Theme.add th "--color-red-500" "initial" 0;
  check "initial deletes" (not (Theme.has th "--color-red-500"));
  Theme.add th "--color-*" "initial" 0;
  check "namespace clear" ((not (Theme.has th "--color-blue-500")) && Theme.has th "--spacing");
  Theme.add th "--text-*" "initial" 0;
  check "ignored sub-namespace survives" ((not (Theme.has th "--text-lg")) && Theme.has th "--text-shadow-sm");
  Theme.add th "--font-*" "initial" 0;
  check "font clear" ((not (Theme.has th "--font-sans")) && Theme.has th "--font-weight-bold");
  expect_error "namespace value" (fun () -> Theme.add th "--color-*" "red" 0) "Invalid theme value";
  Theme.add th "--*" "initial" 0;
  check "clear all" (Theme.size th = 0);
  let th = make_theme () in
  (match Theme.resolve_with th (Some "lg") [ "--text" ] [ "--line-height"; "--letter-spacing" ] with
  | Some (v, extra) ->
      equal "resolve_with value" v "var(--text-lg)";
      check "resolve_with extra" (extra = [ ("--line-height", "var(--text-lg--line-height)") ])
  | None -> check "resolve_with" false);
  check "namespace text"
    (Theme.namespace th "--text"
    = [ { Theme.key = "lg"; self = false; value = "1.125rem" }; { key = "lg--line-height"; self = false; value = "calc(1.75 / 1.125)" };
        { key = "shadow-sm"; self = false; value = "0 1px 1px black" } ]);
  check "namespace self" (Theme.namespace th "--spacing" = [ { Theme.key = ""; self = true; value = "0.25rem" } ]);
  check "keys in namespaces" (Theme.keys_in_namespaces th [ "--text" ] = [ "lg" ]);
  check "keys in namespaces 2" (Theme.keys_in_namespaces th [ "--color"; "--breakpoint" ] = [ "red-500"; "blue-500"; "md" ]);
  equal "get" (Option.get (Theme.get th [ "--missing"; "--spacing" ])) "0.25rem";
  let th = make_theme () in
  th.prefix <- "tw";
  equal "prefix resolve" (resolve th "red-500" [ "--color" ]) "var(--tw-color-red-500)";
  equal "prefix key" (Theme.prefix_key th "--spacing") "--tw-spacing";
  check "mark used" (Theme.mark_used_variable th "--tw-color-red-500");
  check "used option" (Theme.options th "--color-red-500" land Theme.used <> 0);
  check "mark used again" (not (Theme.mark_used_variable th "--tw-color-red-500"));
  check "mark missing" (not (Theme.mark_used_variable th "--missing"));
  let th = make_theme () in
  check "mark escaped dot" (not (Theme.mark_used_variable th "--container-1\\.5"));
  check "mark underscore" (Theme.mark_used_variable th "--container-1_5");
  equal "theme value inline" (Option.get (Theme.resolve_theme_value th "--color-red-500" true)) "red";
  equal "theme value var" (Option.get (Theme.resolve_theme_value th "--color-red-500" false)) "var(--color-red-500)";
  equal "theme value alpha" (Option.get (Theme.resolve_theme_value th "--color-red-500/0.5" true)) "color-mix(in oklab, red 50%, transparent)";
  equal "theme value 100%" (Option.get (Theme.resolve_theme_value th "--color-red-500 / 100%" true)) "red";
  check "theme value missing" (Theme.resolve_theme_value th "--missing" true = None)
