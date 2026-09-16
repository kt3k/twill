(* Nested @variant expansion (SPEC §6.9). *)

open Ast
open Utils

(* Expands every nested @variant in place and returns the features. *)
let substitute (ast : nodes) (ds : Design_system.t) =
  let features = ref 0 in
  ignore
    (walk ast (fun n u ->
         match n with
         | At_rule at when at.name = "@variant" ->
             features := !features lor Features.variants;
             let groups = segment at.params ',' in
             let last = List.length groups - 1 in
             let result =
               List.concat
                 (List.mapi
                    (fun i group ->
                      let names = segment (String.trim group) ':' in
                      let nodes = if i < last then clone_nodes !(at.at_nodes) else !(at.at_nodes) in
                      let r = { selector = "&"; rule_nodes = ref nodes } in
                      List.iter
                        (fun name ->
                          let name = String.trim name in
                          if name = "" then
                            Twill_error.failf "Cannot use `@variant` with an empty variant name in `@variant %s`." at.params;
                          match Design_system.parse_variant ds name with
                          | None -> Twill_error.failf "Cannot use `@variant` with unknown variant: %s" name
                          | Some variant ->
                              if not (Variants.apply_variant (Rule r) variant ds.variants 0) then
                                Twill_error.failf "Cannot use `@variant` with variant: %s" name)
                        (List.rev names);
                      if r.selector = "&" then !(r.rule_nodes) else [ Rule r ])
                    groups)
             in
             replace_with u result;
             Continue
         | _ -> Continue));
  !features
