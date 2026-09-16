(* The compile entry point and the build step (SPEC §4.1.11, §15). *)

open Ast
open Utils

(* The handle returned by [compile]. *)
type t = {
  sources : Directives.source_entry list;
  root : Directives.source_root option;
  features : int;
  design_system : Design_system.t;
  css : string;
  ast : nodes;
  utilities_node : Ast.context option;
  valid_candidates : (string, unit) Hashtbl.t;
  mutable candidate_order : string list;
  mutable pending_inline : bool;
  mutable cached : string option;
  mutable previous_count : int;
}

(* Compiles a stylesheet (SPEC §15.1). [base] is the directory for resolving
   @source and relative imports; [load] loads a stylesheet by id. *)
let compile ?(base = "") ?(load = Builtin.builtin_loader) css =
  let ast = ref [ context (ctx_of_list [ ("base", base) ]) ~nodes:(Parser.parse css) ] in
  let features = ref (Import.substitute_at_imports ast base load) in
  let theme = Theme.create () in
  let state = Directives.collect ast theme in
  features := !features lor state.features;
  let ds = Design_system.build theme in
  ds.important <- state.important;
  List.iter (fun c -> Hashtbl.replace ds.invalid_candidates c ()) state.ignored_candidates;
  let custom_names = List.map (fun (v : Custom_variant.t) -> v.name) state.custom_variants in
  List.iter
    (fun (v : Custom_variant.t) -> Variants.static ds.variants v.name (fun _ _ -> true) Variants.compounds_style_rules)
    state.custom_variants;
  let dependencies =
    List.map
      (fun (v : Custom_variant.t) ->
        let deps =
          List.filter
            (fun dep ->
              if dep = v.name then
                Twill_error.failf "Custom variant `%s` depends on itself, creating a circular dependency." v.name;
              List.mem dep custom_names)
            v.dependencies
        in
        (v.name, deps))
      state.custom_variants
  in
  let sorted =
    At_apply.topological_sort custom_names dependencies (fun cycle ->
        Twill_error.failf "Custom variant `%s` is part of a circular dependency between custom variants." cycle)
  in
  List.iter
    (fun name -> Custom_variants.register ds (List.find (fun (v : Custom_variant.t) -> v.name = name) state.custom_variants))
    sorted;
  List.iter (Utility_body.register ds) state.custom_utilities;
  (match state.first_theme_rule with
  | Some r ->
      let declarations =
        List.filter_map
          (fun key ->
            match Theme.entry theme key with
            | Some e when e.options land Theme.reference = 0 -> Some (decl (escape (Theme.prefix_key theme key)) e.value)
            | _ -> None)
          (Theme.keys theme)
      in
      r.rule_nodes := [ context (ctx_of_list [ ("theme", "true") ]) ~nodes:declarations ]
  | None -> ());
  List.iter
    (fun k -> ast := !ast @ [ context (ctx_of_list [ ("theme", "true") ]) ~nodes:[ at_root [ At_rule k ] ] ])
    (Theme.keyframes theme);
  features := !features lor At_variant.substitute ast ds;
  features := !features lor Theme_functions.substitute_functions ast ds;
  features := !features lor At_apply.substitute ast ds;
  let utilities_node =
    match state.utilities_node with
    | None -> None
    | Some target ->
        let node = { context = ctx_empty; ctx_nodes = ref [] } in
        ignore
          (walk ast (fun n u ->
               match n with
               | At_rule a when a == target ->
                   replace_with u [ Context node ];
                   Stop
               | _ -> Continue));
        Some node
  in
  ignore
    (walk ast (fun n u ->
         match n with
         | At_rule a when a.name = "@utility" ->
             replace_with u [];
             Skip
         | _ -> Continue));
  let c =
    { sources = state.sources; root = state.root; features = !features; design_system = ds; css; ast; utilities_node;
      valid_candidates = Hashtbl.create 256; candidate_order = []; pending_inline = state.inline_candidates <> [];
      cached = None; previous_count = -1 }
  in
  List.iter
    (fun inline ->
      if not (Hashtbl.mem c.valid_candidates inline) then begin
        Hashtbl.replace c.valid_candidates inline ();
        c.candidate_order <- inline :: c.candidate_order
      end)
    state.inline_candidates;
  c

(* Builds the CSS for the accumulated candidates (SPEC §15.2). *)
let build c candidates =
  if c.features = 0 then c.css
  else
    let ds = c.design_system in
    match c.utilities_node with
    | None -> (
        match c.cached with
        | Some out -> out
        | None ->
            let out = Serializer.serialize (Optimize.optimize_ast !(c.ast) ds) in
            c.cached <- Some out;
            out)
    | Some utilities_node -> (
        let changed = ref c.pending_inline in
        c.pending_inline <- false;
        let marked_variable = ref false in
        List.iter
          (fun candidate ->
            if Hashtbl.mem ds.invalid_candidates candidate then ()
            else if has_prefix candidate "--" then begin
              if Theme.mark_used_variable ds.theme candidate then begin
                changed := true;
                marked_variable := true
              end
            end
            else if not (Hashtbl.mem c.valid_candidates candidate) then begin
              Hashtbl.replace c.valid_candidates candidate ();
              c.candidate_order <- candidate :: c.candidate_order;
              changed := true
            end)
          candidates;
        match c.cached with
        | Some out when not !changed -> out
        | _ -> (
            let raws = List.filter (fun raw -> Hashtbl.mem c.valid_candidates raw) (List.rev c.candidate_order) in
            let nodes =
              Compile_candidates.compile_candidates
                ~on_invalid:(fun candidate ->
                  Hashtbl.replace ds.invalid_candidates candidate ();
                  Hashtbl.remove c.valid_candidates candidate)
                raws ds
            in
            match c.cached with
            | Some out when List.length nodes = c.previous_count && not !marked_variable -> out
            | _ ->
                c.previous_count <- List.length nodes;
                utilities_node.ctx_nodes := List.map (fun r -> Rule r) nodes;
                let out = Serializer.serialize (Optimize.optimize_ast !(c.ast) ds) in
                c.cached <- Some out;
                out))
