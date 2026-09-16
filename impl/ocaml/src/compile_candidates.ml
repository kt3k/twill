(* Candidate compilation and ordering (SPEC §11). *)

open Ast
open Candidate
open Design_system

(* Applies the design-system-wide important flag. *)
let respect_important = 1

(* Computes the property sort key of a node list. *)
let get_property_sort nodes =
  let seen = Hashtbl.create 16 and count = ref 0 in
  let queue = Queue.create () in
  List.iter (fun n -> Queue.add n queue) nodes;
  (try
     while not (Queue.is_empty queue) do
       match Queue.pop queue with
       | Declaration d when d.no_value -> ()
       | Declaration d ->
           incr count;
           if d.property = "--tw-sort" then begin
             let index = Property_order.index d.value in
             if index <> -1 then begin
               Hashtbl.replace seen index ();
               raise Exit
             end
           end
           else
             let index = Property_order.index d.property in
             if index <> -1 then Hashtbl.replace seen index ()
       | Rule r -> List.iter (fun n -> Queue.add n queue) !(r.rule_nodes)
       | At_rule a -> List.iter (fun n -> Queue.add n queue) !(a.at_nodes)
       | Context c -> List.iter (fun n -> Queue.add n queue) !(c.ctx_nodes)
       | _ -> ()
     done
   with Exit -> ());
  let order = List.sort compare (Hashtbl.fold (fun k () acc -> k :: acc) seen []) in
  { order; count = !count }

let compile_base_utility (c : candidate) (ds : Design_system.t) =
  if c.ckind = Arbitrary_candidate then
    match Utilities.as_color c.arbitrary_value c.cmodifier ds.theme with
    | Some value -> [ [ decl c.property value ] ]
    | None -> []
  else begin
    let defs =
      List.filter
        (fun (d : Utilities.definition) -> d.ukind = Static = (c.ckind = Static_candidate))
        (Utilities.get ds.utilities c.croot)
    in
    let ordered = List.filter (fun d -> not (Utilities.is_fallback d)) defs @ List.filter Utilities.is_fallback defs in
    let results = ref [] in
    (try
       List.iter
         (fun (d : Utilities.definition) ->
           match d.compile c with
           | _, Utilities.Not_handled -> ()
           | _, Utilities.Invalid -> if d.types <> None then raise Exit
           | nodes, Utilities.Handled -> results := nodes :: !results)
         ordered
     with Exit -> ());
    List.rev !results
  end

let apply_important nodes =
  ignore
    (walk (ref nodes) (fun n _ ->
         match n with
         | At_root _ -> Skip
         | Declaration d ->
             d.important <- true;
             Continue
         | _ -> Continue))

(* Compiles one candidate interpretation (SPEC §11.2). *)
let compile_ast_nodes (c : candidate) flags (ds : Design_system.t) =
  let node_lists = compile_base_utility c ds in
  if node_lists = [] then []
  else begin
    let important = c.important || (ds.important && flags land respect_important <> 0) in
    try
      List.map
        (fun nodes ->
          let property_sort = get_property_sort nodes in
          if important then apply_important nodes;
          let r = { selector = "." ^ Utils.escape c.raw; rule_nodes = ref nodes } in
          if not (List.for_all (fun v -> Variants.apply_variant (Rule r) v ds.variants 0) c.variants) then raise Exit;
          let wrapped = ref [ Rule r ] in
          (try
             ignore (Theme_functions.substitute_functions wrapped ds);
             ignore (At_variant.substitute wrapped ds)
           with Twill_error.Error _ -> raise Exit);
          { node = r; property_sort })
        node_lists
    with Exit -> []
  end

(* Compiles a candidate interpretation, memoized per candidate and flags.
   Cached results are cloned. *)
let compile_ast_nodes_cached (ds : Design_system.t) (c : candidate) flags =
  let cached =
    match Hashtbl.find_opt ds.compile_cache (c, flags) with
    | Some cached -> cached
    | None ->
        let compiled = compile_ast_nodes c flags ds in
        Hashtbl.replace ds.compile_cache (c, flags) compiled;
        compiled
  in
  List.map
    (fun r ->
      match clone_node (Rule r.node) with Rule cloned -> { r with node = cloned } | _ -> assert false)
    cached

type sorted_rule = { srule : Ast.rule; sort : property_sort; variant_order : int list; candidate : string }

let compare_sorted_rules a z =
  let c = compare a.variant_order z.variant_order in
  if c <> 0 then c
  else
    let rec go ao zo =
      match (ao, zo) with
      | [], [] -> 0
      | [], _ -> 1
      | _, [] -> -1
      | ai :: ar, zi :: zr -> if ai = zi then go ar zr else ai - zi
    in
    let c = go a.sort.order z.sort.order in
    if c <> 0 then c
    else if a.sort.count <> z.sort.count then z.sort.count - a.sort.count
    else Utils.compare_natural a.candidate z.candidate

(* Compiles and sorts raw candidates (SPEC §11.1). [on_invalid] receives
   each invalid raw candidate. *)
let compile_candidates ?(ignore_important = false) ?(on_invalid = fun _ -> ()) raw_candidates (ds : Design_system.t) =
  let flags = if ignore_important then 0 else respect_important in
  let matches =
    List.filter_map
      (fun raw ->
        if Hashtbl.mem ds.invalid_candidates raw then (on_invalid raw; None)
        else
          match Design_system.parse_candidate ds raw with
          | [] -> on_invalid raw; None
          | parsed -> Some (raw, parsed))
      raw_candidates
  in
  let order = Design_system.variant_order ds in
  let rules =
    List.concat_map
      (fun (raw, candidates) ->
        let found = ref false in
        let rules =
          List.concat_map
            (fun (c : candidate) ->
              List.map
                (fun (compiled : compiled_rule) ->
                  found := true;
                  let bits =
                    List.sort_uniq (fun a b -> compare b a)
                      (List.map (fun (v : variant) -> match Hashtbl.find_opt order v.vraw with Some i -> i | None -> 0) c.variants)
                  in
                  { srule = compiled.node; sort = compiled.property_sort; variant_order = bits; candidate = raw })
                (compile_ast_nodes_cached ds c flags))
            candidates
        in
        if not !found then on_invalid raw;
        rules)
      matches
  in
  List.map (fun r -> r.srule) (List.stable_sort compare_sorted_rules rules)
