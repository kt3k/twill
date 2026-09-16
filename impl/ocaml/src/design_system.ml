(* The design system: theme, registries, memoization caches, and the set of
   known-invalid candidates (SPEC §4.1.9). *)

open Candidate

type t = {
  theme : Theme.t;
  utilities : Utilities.t;
  variants : Variants.t;
  invalid_candidates : (string, unit) Hashtbl.t;
  (* Marks every generated declaration !important. *)
  mutable important : bool;
  candidate_cache : (string, candidate list) Hashtbl.t;
  variant_cache : (string, variant) Hashtbl.t;
  variant_missing : (string, unit) Hashtbl.t;
  mutable variant_order : (string, int) Hashtbl.t option;
}

let create theme =
  { theme; utilities = Utilities.create (); variants = Variants.create (); invalid_candidates = Hashtbl.create 64; important = false;
    candidate_cache = Hashtbl.create 1024; variant_cache = Hashtbl.create 64; variant_missing = Hashtbl.create 64; variant_order = None }

(* Parses a variant, memoized so equal inputs share one value. *)
let rec parse_variant ds raw =
  match Hashtbl.find_opt ds.variant_cache raw with
  | Some v -> Some v
  | None -> (
      if Hashtbl.mem ds.variant_missing raw then None
      else
        match Candidate.parse_variant raw (context ds) with
        | None ->
            Hashtbl.replace ds.variant_missing raw ();
            None
        | Some v ->
            Hashtbl.replace ds.variant_cache raw v;
            ds.variant_order <- None;
            Some v)

(* The parser context backed by this design system. *)
and context ds : Candidate.context =
  { theme_prefix = (fun () -> ds.theme.prefix);
    has_utility = Utilities.has ds.utilities;
    has_variant = Variants.has ds.variants;
    variant_kind_of = Variants.kind ds.variants;
    compounds_with = Variants.compounds_with ds.variants;
    parse_variant = parse_variant ds }

(* Creates a design system with the built-in variants and utilities. *)
let build theme =
  let ds = create theme in
  Builtin_variants.register ds.variants theme;
  Builtin_utilities.register ds.utilities theme;
  ds

(* Returns every interpretation of a raw class name, memoized. *)
let parse_candidate ds raw =
  match Hashtbl.find_opt ds.candidate_cache raw with
  | Some cached -> cached
  | None ->
      let parsed = Candidate.parse_candidate raw (context ds) in
      Hashtbl.replace ds.candidate_cache raw parsed;
      parsed

(* Sorts every parsed variant and assigns an index that increases each time
   the comparison reports a difference (SPEC §9.4). Keyed by the variant text. *)
let variant_order ds =
  match ds.variant_order with
  | Some order -> order
  | None ->
      let parsed = Hashtbl.fold (fun _ v acc -> v :: acc) ds.variant_cache [] in
      let parsed = List.stable_sort (fun a b -> compare a.vraw b.vraw) parsed in
      let parsed = List.stable_sort (Variants.compare ds.variants) parsed in
      let order = Hashtbl.create (List.length parsed) in
      let index = ref 0 and previous = ref None in
      List.iter
        (fun v ->
          (match !previous with Some p when Variants.compare ds.variants p v <> 0 -> incr index | _ -> ());
          Hashtbl.replace order v.vraw !index;
          previous := Some v)
        parsed;
      ds.variant_order <- Some order;
      order

let order_of ds (v : variant) = match Hashtbl.find_opt (variant_order ds) v.vraw with Some i -> i | None -> 0

(* Drops memoized candidates. *)
let clear_candidate_cache ds = Hashtbl.reset ds.candidate_cache
