(* Directive collection: @theme, @source, @twill utilities, @custom-variant,
   @utility, and the media-position import parameters (SPEC §6). *)

open Ast
open Utils

type source_entry = { base : string; pattern : string; negated : bool }
type source_root = Root_none | Root_entry of { root_base : string; root_pattern : string }

type collected = {
  mutable features : int;
  mutable important : bool;
  mutable sources : source_entry list;
  mutable root : source_root option;
  mutable inline_candidates : string list;
  mutable ignored_candidates : string list;
  mutable utilities_node : at_rule option;
  mutable first_theme_rule : rule option;
  mutable custom_variants : Custom_variant.t list;
  mutable custom_utilities : At_utility.t list;
}

let is_top_level_path path = List.for_all (function Context _ -> true | _ -> false) path

let parse_utilities_source params =
  let rest = String.trim (after params (String.length "utilities")) in
  if rest = "" then `None_given
  else if not (has_prefix rest "source(" && has_suffix rest ")") then Twill_error.failf "Invalid `@twill %s`" params
  else
    let inner = String.trim (String.sub rest 7 (String.length rest - 8)) in
    if inner = "none" then `Disabled
    else
      match unquote inner with
      | Some q -> `Path q
      | None ->
          Twill_error.failf "`source(%s)` paths must be quoted.\n\nInstead use:\n@twill utilities source(\"%s\");" inner inner

let parse_theme_options params in_reference (theme : Theme.t) =
  let options = ref (if in_reference then Theme.reference else 0) in
  List.iter
    (fun option ->
      if option = "" then ()
      else if option = "reference" then options := !options lor Theme.reference
      else if option = "inline" then options := !options lor Theme.inline
      else if option = "default" then options := !options lor Theme.default
      else if option = "static" then options := !options lor Theme.static
      else if has_prefix option "prefix(" && has_suffix option ")" then begin
        let prefix = String.sub option 7 (String.length option - 8) in
        if not (is_valid_prefix prefix) then
          Twill_error.failf "The prefix \"%s\" is invalid. Prefixes must be alphabetic and lowercase." prefix;
        theme.prefix <- prefix
      end
      else Twill_error.failf "Unknown `@theme` option `%s`" option)
    (segment params ' ');
  !options

let handle_source (node : at_rule) base state =
  let params = String.trim node.params in
  let negated, params = if has_prefix params "not " then (true, String.trim (after params 4)) else (false, params) in
  if has_prefix params "inline(" then begin
    if not (has_suffix params ")") then Twill_error.failf "Invalid `@source %s`" node.params;
    match unquote (String.trim (String.sub params 7 (String.length params - 8))) with
    | None -> Twill_error.fail "`@source inline(...)` patterns must be quoted."
    | Some inner ->
        List.iter
          (fun item ->
            let expanded = expand_braces item in
            if negated then state.ignored_candidates <- state.ignored_candidates @ expanded
            else state.inline_candidates <- state.inline_candidates @ expanded)
          (fields inner)
  end
  else
    match unquote params with
    | None ->
        Twill_error.failf "`@source` paths must be quoted.\n\nInstead use:\n@source %s\"%s\";" (if negated then "not " else "") params
    | Some pattern -> state.sources <- state.sources @ [ { base; pattern; negated } ]

(* Interprets the media-position import parameters. *)
let handle_import_media (node : at_rule) (u : utils) state =
  let remaining = ref [] and consumed = ref false in
  List.iter
    (fun param ->
      if param = "" then ()
      else if param = "reference" then begin
        node.at_nodes := [ context (ctx_of_list [ ("reference", "true") ]) ~nodes:!(node.at_nodes) ];
        consumed := true
      end
      else if has_prefix param "theme(" && has_suffix param ")" then begin
        let opts = String.trim (String.sub param 6 (String.length param - 7)) in
        let is_reference = List.mem "reference" (segment opts ' ') in
        ignore
          (walk node.at_nodes (fun child _ ->
               match child with
               | At_rule c when c.name = "@theme" ->
                   c.params <- String.trim (c.params ^ " " ^ opts);
                   Skip
               | At_rule c when c.name = "@layer" || c.name = "@media" -> Continue
               | Context _ | Comment _ -> Continue
               | _ ->
                   if is_reference then
                     Twill_error.fail
                       "Importing a stylesheet with `theme(reference)` is only allowed for stylesheets that contain only `@theme` blocks.";
                   Skip));
        consumed := true
      end
      else if has_prefix param "prefix(" && has_suffix param ")" then begin
        ignore
          (walk node.at_nodes (fun child _ ->
               match child with
               | At_rule c when c.name = "@theme" ->
                   c.params <- String.trim (c.params ^ " " ^ param);
                   Skip
               | _ -> Continue));
        consumed := true
      end
      else if param = "important" then (state.important <- true; consumed := true)
      else if has_prefix param "source(" && has_suffix param ")" then begin
        let source_base = match ctx_get u.wctx "base" with Some b -> b | None -> "" in
        ignore
          (walk node.at_nodes (fun child cu ->
               match child with
               | At_rule c when c.name = "@twill" && c.params = "utilities" ->
                   c.params <- "utilities " ^ param;
                   replace_with cu [ context (ctx_of_list [ ("sourceBase", source_base) ]) ~nodes:[ child ] ];
                   Stop
               | _ -> Continue));
        consumed := true
      end
      else remaining := !remaining @ [ param ])
    (segment node.params ' ');
  if !consumed then
    if !remaining = [] then replace_with u !(node.at_nodes) else node.params <- String.concat " " !remaining

(* Walks the AST once and registers every directive (SPEC §6.1 step 2). *)
let collect (ast : nodes) (theme : Theme.t) =
  let state =
    { features = 0; important = false; sources = []; root = None; inline_candidates = []; ignored_candidates = [];
      utilities_node = None; first_theme_rule = None; custom_variants = []; custom_utilities = [] }
  in
  ignore
    (walk ast (fun n u ->
         match n with
         | At_rule at when at.name = "@media" ->
             handle_import_media at u state;
             Continue
         | At_rule at
           when at.name = "@custom-variant"
                || (at.name = "@variant" && is_top_level_path u.path && Custom_variant.is_custom_variant_compat at) ->
             if not (is_top_level_path u.path) then Twill_error.failf "`%s %s` cannot be nested." at.name at.params;
             state.custom_variants <- state.custom_variants @ [ Custom_variant.parse at ];
             replace_with u [];
             Continue
         | At_rule at when at.name = "@utility" ->
             if not (is_top_level_path u.path) then Twill_error.failf "`@utility %s` cannot be nested." at.params;
             state.custom_utilities <- state.custom_utilities @ [ At_utility.parse at ];
             Skip
         | At_rule at when at.name = "@twill" ->
             if not (has_prefix at.params "utilities") then Continue
             else if ctx_bool u.wctx "reference" || state.utilities_node <> None then (replace_with u []; Continue)
             else begin
               (match parse_utilities_source at.params with
               | `Disabled -> state.root <- Some Root_none
               | `Path source ->
                   let base =
                     match ctx_get u.wctx "sourceBase" with
                     | Some b when b <> "" -> b
                     | _ -> ( match ctx_get u.wctx "base" with Some b -> b | None -> "")
                   in
                   state.root <- Some (Root_entry { root_base = base; root_pattern = source })
               | `None_given -> ());
               state.utilities_node <- Some at;
               state.features <- state.features lor Features.utilities;
               Skip
             end
         | At_rule at when at.name = "@theme" ->
             let options = parse_theme_options at.params (ctx_bool u.wctx "reference") theme in
             List.iter
               (fun child ->
                 match child with
                 | Comment _ -> ()
                 | Declaration d when has_prefix d.property "--" -> Theme.add theme (unescape d.property) d.value options
                 | At_rule k when k.name = "@keyframes" ->
                     if options land Theme.reference = 0 then Theme.add_keyframes theme k
                 | _ ->
                     let lines = String.split_on_char '\n' (Serializer.serialize [ child ]) in
                     let lines = List.filteri (fun i _ -> i < 3) lines in
                     Twill_error.failf "`@theme` blocks must only contain custom properties or `@keyframes`.\n\n%s"
                       (String.concat "\n" lines))
               !(at.at_nodes);
             state.features <- state.features lor Features.at_theme;
             if state.first_theme_rule = None && options land Theme.reference = 0 then begin
               let r = { selector = ":root, :host"; rule_nodes = ref [] } in
               state.first_theme_rule <- Some r;
               replace_with u [ Rule r ]
             end
             else replace_with u [];
             Continue
         | At_rule at when at.name = "@source" ->
             if !(at.at_nodes) <> [] then Twill_error.fail "`@source` cannot have a body.";
             List.iter (function Rule _ | At_rule _ -> Twill_error.fail "`@source` cannot be nested." | _ -> ()) u.path;
             handle_source at (match ctx_get u.wctx "base" with Some b -> b | None -> "") state;
             replace_with u [];
             Continue
         | _ -> Continue));
  state
