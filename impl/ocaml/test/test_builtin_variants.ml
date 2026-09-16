open Twill
open Ast
open Candidate
open Harness
open Testing

let run () =
  let ds = design_system_for "" in
  let flex = "display: flex;" in
  let simple wrapper = wrapper ^ " {\n  " ^ flex ^ "\n}" in
  List.iter
    (fun (raw, want) -> equal raw (compile_raw ds raw) want)
    [ ("hover:flex", block ".hover\\:flex" (block "&:hover" (block "@media (hover: hover)" flex)));
      ("focus:flex", block ".focus\\:flex" (simple "&:focus"));
      ("first:flex", block ".first\\:flex" (simple "&:first-child"));
      ("odd:flex", block ".odd\\:flex" (simple "&:nth-child(odd)"));
      ("open:flex", block ".open\\:flex" (simple "&:is([open], :popover-open, :open)"));
      ("inert:flex", block ".inert\\:flex" (simple "&:is([inert], [inert] *)"));
      ("ltr:flex", block ".ltr\\:flex" (simple "&:where(:dir(ltr), [dir=\"ltr\"], [dir=\"ltr\"] *)"));
      ( "before:block",
        ".before\\:block {\n  &::before {\n    @property --tw-content {\n      syntax: \"*\";\n      inherits: false;\n      initial-value: \"\";\n    }\n    content: var(--tw-content);\n    display: block;\n  }\n}\n"
      );
      ( "marker:flex",
        ".marker\\:flex {\n  & *::marker {\n    display: flex;\n  }\n  &::marker {\n    display: flex;\n  }\n  & *::-webkit-details-marker {\n    display: flex;\n  }\n  &::-webkit-details-marker {\n    display: flex;\n  }\n}\n"
      );
      ("placeholder:flex", block ".placeholder\\:flex" (simple "&::placeholder"));
      ("*:flex", block ".\\*\\:flex" (simple ":is(& > *)"));
      ("**:flex", block ".\\*\\*\\:flex" (simple ":is(& *)"));
      ("sm:flex", block ".sm\\:flex" (simple "@media (width >= 40rem)"));
      ("max-md:flex", block ".max-md\\:flex" (simple "@media (width < 48rem)"));
      ("min-[600px]:flex", block ".min-\\[600px\\]\\:flex" (simple "@media (width >= 600px)"));
      ("min-md:flex", block ".min-md\\:flex" (simple "@media (width >= 48rem)"));
      ("@md:flex", block ".\\@md\\:flex" (simple "@container (width >= 28rem)"));
      ("@md/main:flex", block ".\\@md\\/main\\:flex" (simple "@container main (width >= 28rem)"));
      ("@max-md:flex", block ".\\@max-md\\:flex" (simple "@container (width < 28rem)"));
      ("@min-[300px]:flex", block ".\\@min-\\[300px\\]\\:flex" (simple "@container (width >= 300px)"));
      ("group-hover:flex", block ".group-hover\\:flex" (block "&:is(:where(.group):hover *)" (block "@media (hover: hover)" flex)));
      ( "group-hover/item:flex",
        block ".group-hover\\/item\\:flex" (block "&:is(:where(.group\\/item):hover *)" (block "@media (hover: hover)" flex)) );
      ("peer-checked:flex", block ".peer-checked\\:flex" (simple "&:is(:where(.peer):checked ~ *)"));
      ("has-[>img]:flex", block ".has-\\[\\>img\\]\\:flex" (simple "&:has(> img)"));
      ("has-checked:flex", block ".has-checked\\:flex" (simple "&:has(*:checked)"));
      ("in-data-visible:flex", block ".in-data-visible\\:flex" (simple ":where(*[data-visible]) &"));
      ( "not-hover:flex",
        ".not-hover\\:flex {\n  & {\n    &:not(:hover) {\n      display: flex;\n    }\n    @media not all and (hover: hover) {\n      display: flex;\n    }\n  }\n}\n"
      );
      ("not-supports-grid:flex", block ".not-supports-grid\\:flex" (simple "@supports not (grid: var(--tw))"));
      ("not-sm:flex", block ".not-sm\\:flex" (simple "@media not all and (width >= 40rem)"));
      ("not-print:flex", block ".not-print\\:flex" (simple "@media not print"));
      ("not-@md:flex", block ".not-\\@md\\:flex" (simple "@container not (width >= 28rem)"));
      ( "not-group-hover:flex",
        ".not-group-hover\\:flex {\n  & {\n    &:not(:is(:where(.group):hover *)) {\n      display: flex;\n    }\n    @media not all and (hover: hover) {\n      display: flex;\n    }\n  }\n}\n"
      );
      ("aria-checked:flex", block ".aria-checked\\:flex" (simple "&[aria-checked=\"true\"]"));
      ("aria-[label=foo]:flex", block ".aria-\\[label\\=foo\\]\\:flex" (simple "&[aria-label=\"foo\"]"));
      ("data-[state=open]:flex", block ".data-\\[state\\=open\\]\\:flex" (simple "&[data-state=\"open\"]"));
      ("data-[state=open_i]:flex", block ".data-\\[state\\=open_i\\]\\:flex" (simple "&[data-state=\"open\" i]"));
      ("data-visible:flex", block ".data-visible\\:flex" (simple "&[data-visible]"));
      ("nth-3:flex", block ".nth-3\\:flex" (simple "&:nth-child(3)"));
      ("nth-last-[2n+1]:flex", block ".nth-last-\\[2n\\+1\\]\\:flex" (simple "&:nth-last-child(2n+1)"));
      ("supports-[display:grid]:flex", block ".supports-\\[display\\:grid\\]\\:flex" (simple "@supports (display:grid)"));
      ("supports-grid:flex", block ".supports-grid\\:flex" (simple "@supports (grid: var(--tw))"));
      ("supports-[not(display:grid)]:flex", block ".supports-\\[not\\(display\\:grid\\)\\]\\:flex" (simple "@supports not (display:grid)"));
      ("dark:flex", block ".dark\\:flex" (simple "@media (prefers-color-scheme: dark)"));
      ("print:flex", block ".print\\:flex" (simple "@media print"));
      ("motion-reduce:flex", block ".motion-reduce\\:flex" (simple "@media (prefers-reduced-motion: reduce)"));
      ("starting:flex", block ".starting\\:flex" (simple "@starting-style"));
      ("portrait:flex", block ".portrait\\:flex" (simple "@media (orientation: portrait)"));
      ("pointer-coarse:flex", block ".pointer-coarse\\:flex" (simple "@media (pointer: coarse)"));
      ("noscript:flex", block ".noscript\\:flex" (simple "@media (scripting: none)"));
      ("[&_p]:flex", block ".\\[\\&_p\\]\\:flex" (simple "& p"));
      ("[@media(width>=100px)]:flex", block ".\\[\\@media\\(width\\>\\=100px\\)\\]\\:flex" (simple "@media (width>=100px)"));
      ("[p]:flex", block ".\\[p\\]\\:flex" (simple "&:is(p)"));
      ( "dark:hover:flex",
        block ".dark\\:hover\\:flex" (block "@media (prefers-color-scheme: dark)" (block "&:hover" (block "@media (hover: hover)" flex))) ) ];
  expect_invalid ds
    [ "group-*:flex"; "not-before:flex"; "min-nope:flex"; "sm/foo:flex"; "min-[var(--x)]:flex"; "[>img]:flex"; "group-[>img]:flex";
      "group-sm:flex"; "in-hover/x:flex"; "not-marker:flex"; "not-hover/x:flex"; "nth-foo:flex"; "aria-checked/x:flex" ];

  (* A custom dark variant overrides the built-in one in place. *)
  let ds = design_system_for "" in
  Variants.static ds.variants "dark"
    (fun node _ ->
      match children node with
      | Some ch ->
          ch := [ style_rule "&:where(.dark, .dark *)" ~nodes:!ch ];
          true
      | None -> false)
    Variants.compounds_style_rules;
  equal "custom dark" (compile_raw ds "dark:flex") (block ".dark\\:flex" (simple "&:where(.dark, .dark *)"));

  (* Ordering. *)
  let ds = design_system_for "" in
  let cmp a z =
    match (Design_system.parse_variant ds a, Design_system.parse_variant ds z) with
    | Some a, Some z ->
        let c = Variants.compare ds.variants a z in
        if c < 0 then -1 else if c > 0 then 1 else 0
    | _ -> 99
  in
  List.iter
    (fun (a, z, want) -> equal ("compare " ^ a ^ " " ^ z) (string_of_int (cmp a z)) (string_of_int want))
    [ ("hover", "focus", -1); ("focus", "hover", 1); ("hover", "hover", 0); ("sm", "md", -1); ("md", "lg", -1); ("lg", "min-[600px]", 1);
      ("max-md", "max-sm", -1); ("max-lg", "sm", -1); ("@md", "@lg", -1); ("sm", "dark", -1); ("hover", "[&_p]", -1);
      ("[&_a]", "[&_p]", -1); ("group-hover", "group-focus", -1); ("group-hover", "group-hover/item", -1); ("data-a", "data-b", -1);
      ("data-a", "data-[b]", -1); ("nth-3", "nth-last-3", -1); ("not-hover", "not-focus", -1); ("not-hover", "hover", -1) ];
  let ds = design_system_for "" in
  let get raw = Option.get (Design_system.parse_variant ds raw) in
  let hover = get "hover" and focus = get "focus" and sm = get "sm" in
  let order v = Design_system.order_of ds v in
  check "order" (order hover < order focus && order focus < order sm);
  let dark = get "dark" in
  check "order cache invalidation" (Design_system.order_of ds dark > Design_system.order_of ds sm);

  (* Negation and attribute quoting. *)
  let open Builtin_variants in
  check "negate hover" (negate_selector "&:hover" = Some "&:not(:hover)");
  check "negate attr" (negate_selector "&[data-x]" = Some "&:not([data-x])");
  check "negate where" (negate_selector ":where(.a) &" = Some "&:not(:where(.a) *)");
  check "negate group" (negate_selector "&:is(:where(.group):hover *)" = Some "&:not(:is(:where(.group):hover *))");
  check "negate pseudo element" (negate_selector "&::before" = None);
  check "negate bare" (negate_selector "&" = None);
  check "negate media" (negate_at_rule "@media" "(hover: hover)" = Some ("@media", "not all and (hover: hover)"));
  check "negate print" (negate_at_rule "@media" "print" = Some ("@media", "not print"));
  check "negate supports" (negate_at_rule "@supports" "(display: grid)" = Some ("@supports", "not (display: grid)"));
  check "negate container" (negate_at_rule "@container" "main (width >= 1px)" = Some ("@container", "main not (width >= 1px)"));
  check "negate starting" (negate_at_rule "@starting-style" "" = None);
  List.iter
    (fun (input, want) -> check ("quote " ^ input) (quote_attribute_value input = Some want))
    [ ("state", "state"); ("state=open", "state=\"open\""); ("state=\"open\"", "state=\"open\""); ("state='open'", "state='open'");
      ("state=open i", "state=\"open\" i"); ("state^=op", "state^=\"op\"") ];
  List.iter (fun input -> check ("quote fails " ^ input) (quote_attribute_value input = None)) [ "state="; "st ate=open" ];

  (* Registry. *)
  let v = Variants.create () in
  let noop _ _ = true in
  Variants.static v "a" noop Variants.compounds_style_rules;
  Variants.group v (fun () ->
      Variants.static v "b" noop Variants.compounds_style_rules;
      Variants.functional v "c" noop Variants.compounds_style_rules);
  Variants.static v "d" noop Variants.compounds_style_rules;
  let order name = (Option.get (Variants.get v name)).Variants.order in
  check "orders" (order "a" = 1 && order "b" = 2 && order "c" = 2 && order "d" = 3);
  Variants.functional v "a" noop Variants.compounds_at_rules;
  let a = Option.get (Variants.get v "a") in
  check "re-register keeps order" (a.order = 1 && a.dkind = Functional_variant && a.compounds = Variants.compounds_at_rules);
  check "names" (Variants.names v = [ "a"; "b"; "c"; "d" ]);
  check "compounds style" (Variants.compounds_for_selectors [ "&:hover" ] = Variants.compounds_style_rules);
  check "compounds at" (Variants.compounds_for_selectors [ "@media (x)" ] = Variants.compounds_at_rules);
  check "compounds both"
    (Variants.compounds_for_selectors [ "&:hover"; "@supports (x)" ] = Variants.compounds_style_rules lor Variants.compounds_at_rules);
  check "compounds starting" (Variants.compounds_for_selectors [ "@starting-style" ] = Variants.compounds_never);
  check "compounds pseudo" (Variants.compounds_for_selectors [ "&::before" ] = Variants.compounds_never)
