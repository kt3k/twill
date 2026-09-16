(* @custom-variant parsing (SPEC §6.7). Registration happens in the design
   system once the variant registry exists. *)

open Ast
open Utils

type form =
  | Selector of string list
  (* The cloned body and the selectors it contains (for the compounds kind). *)
  | Body of node list * string list

type t = { name : string; dependencies : string list; form : form }

(* Whether a top-level @variant is a compatibility form of @custom-variant. *)
let is_custom_variant_compat (node : at_rule) =
  if !(node.at_nodes) = [] then String.contains node.params '('
  else begin
    let has_slot = ref false in
    ignore
      (walk node.at_nodes (fun n _ ->
           match n with At_rule at when at.name = "@slot" -> has_slot := true; Stop | _ -> Continue));
    !has_slot
  end

let index_any s chars =
  let n = String.length s in
  let rec go i = if i >= n then -1 else if String.contains chars s.[i] then i else go (i + 1) in
  go 0

(* Parses an @custom-variant node. *)
let parse (node : at_rule) =
  let params = String.trim node.params in
  let name, rest =
    match index_any params " \t\n" with
    | -1 -> (params, "")
    | space -> (String.sub params 0 space, String.trim (after params space))
  in
  if not (is_valid_variant_name name) then
    Twill_error.failf
      "`@custom-variant %s` defines an invalid variant name. Variants should only contain alphanumeric, dashes or underscore characters."
      name;
  let has_body = !(node.at_nodes) <> [] in
  if rest <> "" && has_body then Twill_error.failf "`@custom-variant %s` cannot have both a selector and a body." name;
  if rest = "" && not has_body then Twill_error.failf "`@custom-variant %s` has no selector or body." name;
  if rest <> "" then begin
    if not (has_prefix rest "(" && has_suffix rest ")") then
      Twill_error.failf "`@custom-variant %s %s` has an invalid selector." name rest;
    let selectors =
      List.map
        (fun s ->
          let s = String.trim s in
          if s = "" then Twill_error.failf "`@custom-variant %s %s` has an empty selector." name rest;
          s)
        (segment (String.sub rest 1 (String.length rest - 2)) ',')
    in
    { name; dependencies = []; form = Selector selectors }
  end
  else begin
    let body = clone_nodes !(node.at_nodes) in
    let dependencies = ref [] and selectors = ref [] in
    let body_ref = ref body in
    ignore
      (walk body_ref (fun n _ ->
           (match n with
           | Rule r -> selectors := !selectors @ [ r.selector ]
           | At_rule a when a.name = "@variant" ->
               List.iter
                 (fun group ->
                   List.iter
                     (fun v ->
                       let v = String.trim v in
                       if v <> "" && not (List.mem v !dependencies) then dependencies := !dependencies @ [ v ])
                     (segment group ':'))
                 (segment a.params ',')
           | At_rule a when a.name <> "@slot" -> selectors := !selectors @ [ String.trim (a.name ^ " " ^ a.params) ]
           | _ -> ());
           Continue));
    { name; dependencies = !dependencies; form = Body (!body_ref, !selectors) }
  end
