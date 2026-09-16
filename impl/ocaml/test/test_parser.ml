open Twill
open Ast
open Harness

let parse = Parser.parse
let ser = Serializer.serialize
let equal_ast name got want =
  equal name (ser got) (ser want);
  check (name ^ " (structure)") (got = want)

let run () =
  equal_ast "declarations and rules"
    (parse ".a { color: red; background: blue }")
    [ style_rule ".a" ~nodes:[ decl "color" "red"; decl "background" "blue" ] ];
  equal_ast "nesting"
    (parse ".a {\n color: red;\n &:hover { color: blue; }\n @media (hover: hover) {\n .b & { color: green; }\n }\n }")
    [ style_rule ".a"
        ~nodes:[ decl "color" "red"; style_rule "&:hover" ~nodes:[ decl "color" "blue" ];
                 at_rule "@media" "(hover: hover)" ~nodes:[ style_rule ".b &" ~nodes:[ decl "color" "green" ] ] ] ];
  equal_ast "statement at-rules"
    (parse "@import \"foo.css\" layer(base);\n@charset \"utf-8\";\n@twill utilities;")
    [ at_rule "@import" "\"foo.css\" layer(base)"; at_rule "@charset" "\"utf-8\""; at_rule "@twill" "utilities" ];
  equal_ast "at-rule without space"
    (parse "@media(width>=1px){.a{x:y}}")
    [ at_rule "@media" "(width>=1px)" ~nodes:[ style_rule ".a" ~nodes:[ decl "x" "y" ] ] ];
  equal_ast "comments"
    (parse "/* license? no */\n/*! license */\n.a { /* inner */ color: red; }")
    [ comment "! license "; style_rule ".a" ~nodes:[ decl "color" "red" ] ];
  equal_ast "important"
    (parse ".a { color: red !important; width: 1px!IMPORTANT }")
    [ style_rule ".a" ~nodes:[ important_decl "color" "red"; important_decl "width" "1px" ] ];
  equal_ast "custom properties"
    (parse ":root { --a: 1px; --b:; --c: { foo: bar; }; --d: calc(1px + 2px) }")
    [ style_rule ":root" ~nodes:[ decl "--a" "1px"; decl "--b" ""; decl "--c" "{ foo: bar; }"; decl "--d" "calc(1px + 2px)" ] ];
  equal_ast "delimiters in quotes and parens"
    (parse ".a { content: \"a;b{c}\"; background: url(x;y{z}); font-family: 'q}q' }")
    [ style_rule ".a" ~nodes:[ decl "content" "\"a;b{c}\""; decl "background" "url(x;y{z})"; decl "font-family" "'q}q'" ] ];
  equal_ast "escapes" (parse ".a\\:b { color: red }") [ style_rule ".a\\:b" ~nodes:[ decl "color" "red" ] ];
  equal_ast "escaped quote" (parse ".a { content: \"\\\"\" }") [ style_rule ".a" ~nodes:[ decl "content" "\"\\\"\"" ] ];
  equal_ast "missing semicolon" (parse ".a { color: red }") [ style_rule ".a" ~nodes:[ decl "color" "red" ] ];
  equal_ast "statement without semicolon" (parse "@import \"a\"") [ at_rule "@import" "\"a\"" ];
  equal_ast "crlf" (parse ".a {\r\n  color: red;\r\n}\r\n") [ style_rule ".a" ~nodes:[ decl "color" "red" ] ];
  equal_ast "empty" (parse "") [];
  equal_ast "whitespace" (parse "  \n ") [];
  (match Parser.parse ".a {\n  color: red;\n" with
  | exception Twill_error.Error (_, Some p) -> check "error position" (p = { Twill_error.line = 3; column = 1 })
  | _ -> check "error position" false);
  (match Parser.parse ".a { color: red; }\n}" with
  | exception Twill_error.Error (_, Some p) -> check "unexpected brace position" (p = { Twill_error.line = 2; column = 1 })
  | _ -> check "unexpected brace position" false);
  List.iter
    (fun (input, message) -> expect_error ("error: " ^ input) (fun () -> Parser.parse input) message)
    [ (".a { color: 'red }", "Unterminated string"); (".a { color: red; /* x", "Unterminated comment");
      (".a { color: calc(1px; }", "Missing closing"); (".a { color }", "Invalid declaration") ];
  let ast =
    [ comment "! license"; at_rule "@import" "\"a.css\"";
      style_rule ".a"
        ~nodes:[ decl "color" "red"; important_decl "width" "1px"; no_value_decl "hidden";
                 at_rule "@media" "(hover: hover)" ~nodes:[ style_rule "&:hover" ~nodes:[ decl "color" "blue" ] ];
                 at_rule "@starting-style" "" ~nodes:[ decl "opacity" "0" ] ] ]
  in
  equal "serialize" (ser ast)
    "/*! license*/\n@import \"a.css\";\n.a {\n  color: red;\n  width: 1px !important;\n  @media (hover: hover) {\n    &:hover {\n      color: blue;\n    }\n  }\n  @starting-style {\n    opacity: 0;\n  }\n}\n";
  let ctx_ast =
    [ context (ctx_of_list [ ("base", "/x") ]) ~nodes:[ style_rule ".a" ~nodes:[ decl "color" "red" ] ];
      at_root [ style_rule ".b" ~nodes:[ decl "color" "blue" ] ] ]
  in
  equal "serialize context and at-root" (ser ctx_ast) ".a {\n  color: red;\n}\n.b {\n  color: blue;\n}\n";
  let css = "@layer theme, base;\n:root, :host {\n  --spacing: 0.25rem;\n}\n@media (width >= 40rem) {\n  .sm\\:flex {\n    display: flex;\n  }\n}\n" in
  equal "round trip" (ser (parse css)) css;
  equal "compact"
    (Serializer.serialize_compact (parse ".a { color: red !important; @media (x) { &:hover { y: z } } }\n@import \"a\";"))
    ".a{color:red!important;@media (x){&:hover{y:z;}}}@import \"a\";";
  let nodes = ref (parse ".a { color: red; } .b { color: blue; }") in
  ignore
    (walk nodes (fun n u ->
         (match n with
         | Rule r when r.selector = ".a" ->
             replace_with u [ style_rule ".c" ~nodes:[ decl "x" "y" ]; style_rule ".d" ]
         | _ -> ());
         Continue));
  equal "walk replace" (ser !nodes) ".c {\n  x: y;\n}\n.d {\n}\n.b {\n  color: blue;\n}\n";
  let seen = ref [] in
  ignore (walk nodes (fun n _ -> (match n with Declaration d -> seen := d.property :: !seen | _ -> ()); Continue));
  check "walk order" (List.rev !seen = [ "x"; "color" ]);
  (* ReplaceWith followed by Stop keeps the replacement. *)
  let nodes = ref (parse ".a { color: red; }") in
  ignore (walk nodes (fun n u -> (match n with Rule _ -> replace_with u [ style_rule ".z" ] | _ -> ()); Stop));
  equal "replace then stop" (ser !nodes) ".z {\n}\n"
