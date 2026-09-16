(* The variant registry (SPEC §4.1.8, §9.1, §9.3, §9.4). *)

open Ast
open Candidate

(* The kinds of rules a variant generates or accepts, as a bit set. *)
let compounds_never = 0
let compounds_at_rules = 1
let compounds_style_rules = 2

(* Mutates the children of a rule or at-rule in place; false rejects the candidate. *)
type apply = node -> variant -> bool

type definition = {
  mutable dkind : variant_kind;
  order : int;
  mutable apply : apply;
  mutable compounds : int;
  compounds_with : int;
}

type compare_fn = variant -> variant -> int

type t = {
  defs : (string, definition) Hashtbl.t;
  mutable names_rev : string list;
  compare_fns : (int, compare_fn) Hashtbl.t;
  mutable last_order : int;
  mutable group_order : int;
}

let create () = { defs = Hashtbl.create 128; names_rev = []; compare_fns = Hashtbl.create 8; last_order = 0; group_order = 0 }

let register v name kind apply compounds compounds_with =
  match Hashtbl.find_opt v.defs name with
  | Some existing ->
      existing.dkind <- kind;
      existing.apply <- apply;
      existing.compounds <- compounds
  | None ->
      let order =
        if v.group_order <> 0 then v.group_order
        else begin
          v.last_order <- v.last_order + 1;
          v.last_order
        end
      in
      Hashtbl.replace v.defs name { dkind = kind; order; apply; compounds; compounds_with };
      v.names_rev <- name :: v.names_rev

let static v name apply compounds = register v name Static_variant apply compounds compounds_never
let functional v name apply compounds = register v name Functional_variant apply compounds compounds_never
let compound v name compounds_with apply compounds = register v name Compound_variant apply compounds compounds_with

(* Registers every name inside [f] with one shared order. *)
let group v ?compare f =
  v.last_order <- v.last_order + 1;
  v.group_order <- v.last_order;
  (match compare with Some c -> Hashtbl.replace v.compare_fns v.group_order c | None -> ());
  f ();
  v.group_order <- 0

let has v name = Hashtbl.mem v.defs name
let get v name = Hashtbl.find_opt v.defs name
let kind v name = match get v name with Some d -> Some d.dkind | None -> None
let names v = List.rev v.names_rev

(* Computes the compounds value of a selector list. *)
let compounds_for_selectors selectors =
  let rec go acc = function
    | [] -> acc
    | s :: rest ->
        if Utils.has_prefix s "@" then
          if not (Utils.has_prefix s "@media" || Utils.has_prefix s "@supports" || Utils.has_prefix s "@container") then
            compounds_never
          else go (acc lor compounds_at_rules) rest
        else if Utils.contains s "::" then compounds_never
        else go (acc lor compounds_style_rules) rest
  in
  go compounds_never selectors

let compounds_of v (variant : variant) =
  if variant.kind = Arbitrary_variant then compounds_for_selectors [ variant.selector ]
  else match get v variant.root with Some d -> d.compounds | None -> compounds_never

(* Reports whether the compound [parent] accepts [child]. *)
let compounds_with v parent child =
  match get v parent with
  | Some d when d.dkind = Compound_variant ->
      let c = compounds_of v child in
      c <> compounds_never && d.compounds_with <> compounds_never && c land d.compounds_with <> 0
  | _ -> false

let compare_strings (a : string) z = compare a z

(* Compares two parsed variants for output ordering (SPEC §9.4). *)
let rec compare v (a : variant) (z : variant) =
  if a == z then 0
  else if a.kind = Arbitrary_variant && z.kind = Arbitrary_variant then compare_strings a.selector z.selector
  else if a.kind = Arbitrary_variant then 1
  else if z.kind = Arbitrary_variant then -1
  else
    let order_of x = match get v x.root with Some d -> d.order | None -> 1 lsl 30 in
    let a_order = order_of a and z_order = order_of z in
    if a_order <> z_order then a_order - z_order
    else if a.kind = Compound_variant && z.kind = Compound_variant then
      let inner =
        match (a.inner, z.inner) with
        | Some ai, Some zi -> compare v ai zi
        | None, None -> 0
        | None, Some _ -> -1
        | Some _, None -> 1
      in
      if inner <> 0 then inner
      else
        match (a.vmodifier, z.vmodifier) with
        | Some am, Some zm -> compare_strings am.mvalue zm.mvalue
        | Some _, None -> 1
        | None, Some _ -> -1
        | None, None -> 0
    else
      match Hashtbl.find_opt v.compare_fns a_order with
      | Some f -> f a z
      | None ->
          if a.root <> z.root then compare_strings a.root z.root
          else
            let value_of x = if x.kind = Functional_variant then x.vvalue else None in
            match (value_of a, value_of z) with
            | None, None -> 0
            | None, Some _ -> -1
            | Some _, None -> 1
            | Some av, Some zv ->
                if av.vvkind <> zv.vvkind then if av.vvkind = Arbitrary then 1 else -1
                else compare_strings av.vvalue zv.vvalue

(* Registers a static variant wrapping the children in one rule per selector. *)
let static_variant v name selectors compounds use_default =
  let compounds = if use_default then compounds_for_selectors selectors else compounds in
  static v name
    (fun node _ ->
      match children node with
      | None -> false
      | Some ch ->
          let inner i = if i = 0 then !ch else clone_nodes !ch in
          ch := List.mapi (fun i selector -> rule ~nodes:(inner i) selector) selectors;
          true)
    compounds

let is_rule_like = function Rule _ | At_rule _ -> true | _ -> false

(* Applies a parsed variant to a rule node (SPEC §9.3). *)
let rec apply_variant node (variant : variant) v depth =
  match children node with
  | None -> false
  | Some ch -> (
      if variant.kind = Arbitrary_variant then
        if variant.relative && depth = 0 then false
        else begin
          ch := [ rule ~nodes:!ch variant.selector ];
          true
        end
      else
        match get v variant.root with
        | None -> false
        | Some def ->
            if variant.kind = Compound_variant then begin
              let isolated = { name = "@slot"; params = ""; at_nodes = ref [] } in
              let inner = match variant.inner with Some i -> i | None -> variant in
              if not (apply_variant (At_rule isolated) inner v (depth + 1)) then false
              else if variant.root = "not" && List.length !(isolated.at_nodes) > 1 then false
              else if not (List.for_all (fun child -> is_rule_like child && def.apply child variant) !(isolated.at_nodes)) then false
              else begin
                let first = ref true in
                let rec fill nodes =
                  List.iter
                    (fun child ->
                      match children child with
                      | Some cc when is_rule_like child ->
                          if !cc = [] then begin
                            cc := if !first then !ch else clone_nodes !ch;
                            first := false
                          end
                          else fill !cc
                      | _ -> ())
                    nodes
                in
                fill !(isolated.at_nodes);
                ch := !(isolated.at_nodes);
                true
              end
            end
            else def.apply node variant)

(* Reports whether a tree contains nested style rules. *)
let has_nested_style_rules nodes =
  let found = ref false in
  ignore (walk (ref nodes) (fun n _ -> match n with Rule _ -> found := true; Stop | _ -> Continue));
  !found
