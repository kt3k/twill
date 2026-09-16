(* @import and @reference (SPEC §6.2). *)

open Ast
open Utils

let max_import_depth = 100

type params = { uri : string; layer : string option; media : string option; supports : string option }

(* Parses @import params; None for imports that must be left untouched. *)
let parse_import_params text =
  let rec drop_seps = function Value.Sep _ :: rest -> drop_seps rest | l -> l in
  match drop_seps (Value.parse text) with
  | Value.Word first :: rest -> (
      match unquote first.w with
      | None -> None
      | Some uri when has_prefix uri "data:" || has_prefix uri "http://" || has_prefix uri "https://" -> None
      | Some uri ->
          let layer = ref None and supports = ref None and media = ref [] and saw_media = ref false in
          List.iter
            (function
              | Value.Sep s -> if !media <> [] then media := !media @ [ s ]
              | Value.Fn f when f.fname = "layer" ->
                  if !supports <> None || !saw_media then
                    Twill_error.failf "`layer(...)` must appear before `supports(...)` and media queries in `@import %s`" text;
                  if !layer <> None then Twill_error.failf "Duplicate `layer(...)` in `@import %s`" text;
                  layer := Some (String.trim (Value.to_css f.fnodes))
              | Value.Fn f when f.fname = "supports" ->
                  if !saw_media then Twill_error.failf "`supports(...)` must appear before media queries in `@import %s`" text;
                  supports := Some (String.trim (Value.to_css f.fnodes))
              | Value.Fn f ->
                  saw_media := true;
                  media := !media @ [ Value.to_css [ Value.Fn f ] ]
              | Value.Word w when w.w = "layer" ->
                  if !supports <> None || !saw_media then
                    Twill_error.failf "`layer` must appear before `supports(...)` and media queries in `@import %s`" text;
                  layer := Some ""
              | Value.Word w ->
                  saw_media := true;
                  media := !media @ [ w.w ])
            rest;
          let media_text = String.trim (String.concat "" !media) in
          Some { uri; layer = !layer; supports = !supports; media = (if media_text = "" then None else Some media_text) })
  | _ -> None

(* Wraps imported nodes per SPEC §6.2. *)
let build_import_nodes nodes layer media supports =
  let root = match layer with Some l -> [ at_rule "@layer" l ~nodes ] | None -> nodes in
  let root = match media with Some m -> [ at_rule "@media" m ~nodes:root ] | None -> root in
  match supports with
  | Some s ->
      let condition = if has_prefix s "(" then s else "(" ^ s ^ ")" in
      [ at_rule "@supports" condition ~nodes:root ]
  | None -> root

(* Expands @import and @reference in place; returns the feature flags. *)
let rec substitute_at_imports_depth (ast : nodes) base (load : Builtin.loader) depth =
  let features = ref 0 in
  ignore
    (walk ast (fun n u ->
         match n with
         | At_rule at when at.name = "@import" || at.name = "@reference" -> (
             match parse_import_params at.params with
             | None -> Continue
             | Some parsed ->
                 let parsed = if at.name = "@reference" then { parsed with media = Some "reference" } else parsed in
                 if depth > max_import_depth then
                   Twill_error.failf "Exceeded maximum recursion depth while resolving `%s %s`" at.name at.params;
                 features := !features lor Features.at_import;
                 let loaded = load parsed.uri base in
                 let imported = ref (Parser.parse loaded.content) in
                 features := !features lor substitute_at_imports_depth imported loaded.base load (depth + 1);
                 let wrapped =
                   build_import_nodes
                     [ context (ctx_of_list [ ("base", loaded.base) ]) ~nodes:!imported ]
                     parsed.layer parsed.media parsed.supports
                 in
                 replace_with u [ context ctx_empty ~nodes:wrapped ];
                 Skip)
         | _ -> Continue));
  !features

let substitute_at_imports ast base load = substitute_at_imports_depth ast base load 0
