(* @apply expansion (SPEC §6.10). *)

open Ast
open Utils

(* Orders names so that every name comes after its dependencies; [on_cycle]
   raises for a cycle. *)
let topological_sort names (dependencies : (string * string list) list) on_cycle =
  let result = ref [] in
  let state = Hashtbl.create 16 in
  let rec visit name =
    match Hashtbl.find_opt state name with
    | Some 2 -> ()
    | Some _ -> on_cycle name
    | None ->
        Hashtbl.replace state name 1;
        let deps = match List.assoc_opt name dependencies with Some d -> d | None -> [] in
        let deps = List.sort compare (List.filter (fun d -> List.mem_assoc d dependencies) deps) in
        List.iter visit deps;
        Hashtbl.replace state name 2;
        result := name :: !result
  in
  List.iter visit names;
  List.rev !result

(* Builds the error message for an unknown @apply candidate. *)
let apply_error_message candidate (ds : Design_system.t) =
  let prefix = ds.theme.prefix in
  if prefix <> "" && not (has_prefix candidate (prefix ^ ":")) then
    "Cannot apply unknown utility class `" ^ candidate ^ "`. Did you forget the `" ^ prefix ^ ":` prefix?"
  else if Hashtbl.mem ds.invalid_candidates candidate then
    "Cannot apply utility class `" ^ candidate ^ "` because it is explicitly disabled by `@source not inline(...)`."
  else begin
    let parts = segment candidate ':' in
    let variants = List.filteri (fun i _ -> i < List.length parts - 1) parts in
    let variants = match variants with _ :: rest when prefix <> "" -> rest | v -> v in
    match List.find_opt (fun v -> Design_system.parse_variant ds v = None) variants with
    | Some v -> "Cannot apply unknown variant `" ^ v ^ "` in `" ^ candidate ^ "`."
    | None ->
        if Theme.size ds.theme = 0 then
          "Cannot apply unknown utility class `" ^ candidate
          ^ "`. The theme is empty; are you missing `@import \"twill\";` or `@reference \"twill\";`?"
        else "Cannot apply unknown utility class `" ^ candidate ^ "`."
  end

let expand_apply_in (nodes : nodes) (root : node option) (ds : Design_system.t) =
  let features = ref 0 in
  let path = match root with Some r -> [ r ] | None -> [] in
  ignore
    (walk_from nodes
       (fun n u ->
         match n with
         | At_rule at when at.name = "@utility" -> Skip
         | At_rule at when at.name = "@apply" -> (
             let parent = match u.parent with Some p -> Some p | None -> root in
             match parent with
             | None -> Continue (* A top-level @apply is left untouched. *)
             | Some (Context _) when Directives.is_top_level_path u.path -> Continue
             | Some _ ->
                 if !(at.at_nodes) <> [] then Twill_error.fail "`@apply` cannot have a body.";
                 List.iter
                   (function
                     | At_rule a when a.name = "@keyframes" -> Twill_error.fail "You cannot use `@apply` inside `@keyframes`."
                     | _ -> ())
                   u.path;
                 let candidates = fields at.params in
                 let mixins = List.length (List.filter (fun c -> has_prefix c "--") candidates) in
                 if mixins = List.length candidates && candidates <> [] then Continue
                 else begin
                   if mixins > 0 then
                     Twill_error.failf "You cannot mix CSS mixins with utility classes in `@apply %s`." at.params;
                   let invalid = ref None in
                   let compiled =
                     Compile_candidates.compile_candidates ~ignore_important:true
                       ~on_invalid:(fun c -> if !invalid = None then invalid := Some (apply_error_message c ds))
                       candidates ds
                   in
                   (match !invalid with Some message -> Twill_error.fail message | None -> ());
                   let replacement = List.concat_map (fun (r : rule) -> !(r.rule_nodes)) compiled in
                   features := !features lor Features.at_apply;
                   replace_with u replacement;
                   Continue
                 end)
         | _ -> Continue)
       root ctx_empty path);
  !features

(* Expands every @apply in place. @apply inside @utility bodies is expanded
   first, in dependency order. Returns the features. *)
let substitute (ast : nodes) (ds : Design_system.t) =
  let features = ref 0 in
  let utility_nodes = ref [] in
  ignore
    (walk ast (fun n _ ->
         match n with
         | At_rule at when at.name = "@utility" ->
             let root = trim_suffix (String.trim at.params) "-*" in
             utility_nodes := (root, at) :: List.remove_assoc root !utility_nodes;
             Skip
         | _ -> Continue));
  let utility_order = List.rev_map fst !utility_nodes in
  let utility_order = List.fold_left (fun acc r -> if List.mem r acc then acc else acc @ [ r ]) [] utility_order in
  if !utility_nodes <> [] then begin
    let dependencies =
      List.map
        (fun root ->
          let node = List.assoc root !utility_nodes in
          let deps = ref [] in
          ignore
            (walk node.at_nodes (fun child _ ->
                 (match child with
                 | At_rule at when at.name = "@apply" ->
                     List.iter
                       (fun candidate ->
                         List.iter
                           (fun (parsed : Candidate.candidate) ->
                             if parsed.ckind = Candidate.Arbitrary_candidate then ()
                             else if not (List.mem_assoc parsed.croot !utility_nodes) then ()
                             else if parsed.croot = root then
                               Twill_error.failf
                                 "You cannot `@apply` the `%s` utility here because it creates a circular dependency." candidate
                             else if not (List.mem parsed.croot !deps) then deps := !deps @ [ parsed.croot ])
                           (Design_system.parse_candidate ds candidate))
                       (fields at.params)
                 | _ -> ());
                 Continue));
          (root, !deps))
        utility_order
    in
    let sorted =
      topological_sort utility_order dependencies (fun cycle ->
          Twill_error.failf "You cannot `@apply` the `%s` utility here because it creates a circular dependency." cycle)
    in
    List.iter
      (fun root ->
        let node = List.assoc root !utility_nodes in
        features := !features lor expand_apply_in node.at_nodes (Some (At_rule node)) ds)
      sorted
  end;
  !features lor expand_apply_in ast None ds
