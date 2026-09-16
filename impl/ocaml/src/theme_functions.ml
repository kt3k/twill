(* Theme function substitution (SPEC §6.11). *)

open Ast
open Utils

let function_names = [ "--spacing("; "--alpha("; "--theme("; "theme(" ]

(* Whether a value contains a theme function call. *)
let has_theme_function value =
  List.exists
    (fun name ->
      let rec find from =
        match index_of (after value from) name with
        | -1 -> false
        | i ->
            let i = from + i in
            if i = 0 || not (is_ident_byte value.[i - 1]) then true else find (i + 1)
      in
      find 0)
    function_names

let function_args nodes = List.map String.trim (segment (Value.to_css nodes) ',')

let spacing_function nodes (ds : Design_system.t) =
  let args = function_args nodes in
  (match args with
  | [ a ] when a <> "" -> ()
  | _ ->
      Twill_error.failf "The --spacing(…) function requires exactly one argument, but received %d."
        (List.length (List.filter (fun a -> a <> "") args)));
  let multiplier =
    match Theme.resolve ds.theme None [ "--spacing" ] 0 with
    | Some m -> m
    | None -> Twill_error.fail "The --spacing(…) function requires that the `--spacing` theme variable exists, but it was not found."
  in
  match List.hd args with "0" -> "0px" | "1" -> multiplier | arg -> "calc(" ^ multiplier ^ " * " ^ arg ^ ")"

let alpha_function nodes =
  let text = Value.to_css nodes in
  if List.length (segment text ',') <> 1 then
    Twill_error.failf "The --alpha(…) function requires exactly one argument in the form `<color> / <alpha>`, but received `%s`." text;
  match segment text '/' with
  | [ color; alpha ] when String.trim color <> "" && String.trim alpha <> "" -> Theme.with_alpha (String.trim color) (String.trim alpha)
  | _ -> Twill_error.failf "The --alpha(…) function requires a color and an alpha value separated by `/`, but received `%s`." text

let is_theme_reference name = name = "var" || name = "theme" || name = "--theme"

let rec inject_fallback_nodes (nodes : Value.vnode list) fallback =
  List.exists
    (fun n ->
      match n with
      | Value.Fn f ->
          if inject_fallback_nodes f.fnodes fallback then true
          else if not (is_theme_reference f.fname) then false
          else begin
            let rec find i = function
              | [] -> -1
              | Value.Sep "," :: _ -> i
              | _ :: rest -> find (i + 1) rest
            in
            match find 0 f.fnodes with
            | -1 ->
                f.fnodes <- f.fnodes @ [ Value.sep ","; Value.sep " "; Value.word fallback ];
                true
            | comma ->
                let before = List.filteri (fun i _ -> i <= comma) f.fnodes in
                let after_comma = List.filteri (fun i _ -> i > comma) f.fnodes in
                if String.trim (Value.to_css after_comma) = "initial" then begin
                  f.fnodes <- before @ [ Value.sep " "; Value.word fallback ];
                  true
                end
                else false
          end
      | _ -> false)
    nodes

(* Injects [fallback] into the innermost var(...), theme(...), or
   --theme(...) call that has no fallback or whose fallback is `initial`. *)
let inject_fallback value fallback =
  let ast = Value.parse value in
  ignore (inject_fallback_nodes ast fallback);
  Value.to_css ast

let theme_function nodes (ds : Design_system.t) in_at_rule =
  let args = function_args nodes in
  let key = match args with k :: _ -> k | [] -> "" in
  let inline, key =
    if has_suffix key " inline" then (true, String.trim (trim_suffix key " inline")) else (in_at_rule, key)
  in
  let has_fallback = List.length args > 1 in
  let fallback = if has_fallback then String.concat ", " (List.tl args) else "" in
  if not (has_prefix key "--") then
    Twill_error.failf "The --theme(…) function can only be used with CSS variables from your theme, but received `%s`." key;
  match Theme.resolve_theme_value ds.theme key inline with
  | None ->
      if has_fallback then fallback else Twill_error.failf "Could not resolve value for theme function: `--theme(%s)`." key
  | Some resolved ->
      if (not has_fallback) || fallback = "initial" then resolved
      else if resolved = "initial" then fallback
      else if has_prefix resolved "var(" || has_prefix resolved "theme(" || has_prefix resolved "--theme(" then
        inject_fallback resolved fallback
      else resolved

let legacy_theme_function nodes (ds : Design_system.t) =
  let args = function_args nodes in
  let key = match args with k :: _ -> k | [] -> "" in
  let key = match unquote key with Some k -> k | None -> key in
  match Theme.resolve_theme_value ds.theme key true with
  | Some resolved -> resolved
  | None ->
      if List.length args > 1 then String.concat ", " (List.tl args)
      else Twill_error.failf "Could not resolve value for theme function: `theme(%s)`." key

(* Substitutes theme functions in one value. *)
let substitute_in_value value ds in_at_rule =
  let ast = ref (Value.parse value) in
  Value.walk ast (fun n _ ->
      match n with
      | Value.Fn f -> (
          let result =
            match f.fname with
            | "--spacing" -> Some (spacing_function f.fnodes ds)
            | "--alpha" -> Some (alpha_function f.fnodes)
            | "--theme" -> Some (theme_function f.fnodes ds in_at_rule)
            | "theme" -> Some (legacy_theme_function f.fnodes ds)
            | _ -> None
          in
          match result with Some r -> Value.VReplace [ Value.word r ] | None -> Value.VContinue)
      | _ -> Value.VContinue);
  Value.to_css !ast

let at_rules_with_params = [ "@media"; "@custom-media"; "@container"; "@supports" ]

(* Substitutes theme functions in declaration values and in the params of
   @media, @custom-media, @container, and @supports. Returns the features. *)
let substitute_functions (ast : nodes) ds =
  let features = ref 0 in
  ignore
    (walk ast (fun n _ ->
         (match n with
         | Declaration d when (not d.no_value) && has_theme_function d.value ->
             d.value <- substitute_in_value d.value ds false;
             features := !features lor Features.theme_function
         | At_rule at when List.mem at.name at_rules_with_params && has_theme_function at.params ->
             at.params <- substitute_in_value at.params ds true;
             features := !features lor Features.theme_function
         | _ -> ());
         Continue));
  !features
