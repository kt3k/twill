(* Registration of parsed @custom-variant definitions (SPEC §6.7). *)

open Ast

let take_from children =
  let first = ref true in
  fun () ->
    if !first then (first := false; !children) else clone_nodes !children

(* Registers a parsed custom variant on the design system. *)
let register (ds : Design_system.t) (cv : Custom_variant.t) =
  match cv.form with
  | Custom_variant.Selector selectors ->
      let style_selectors = List.filter (fun s -> not (Utils.has_prefix s "@")) selectors in
      let at_rule_selectors = List.filter (fun s -> Utils.has_prefix s "@") selectors in
      let compounds = Variants.compounds_for_selectors selectors in
      Variants.static ds.variants cv.name
        (fun target _ ->
          match children target with
          | None -> false
          | Some ch ->
              let take = take_from ch in
              let nodes =
                (if style_selectors <> [] then [ style_rule (String.concat ", " style_selectors) ~nodes:(take ()) ] else [])
                @ List.map (fun s -> rule s ~nodes:(take ())) at_rule_selectors
              in
              ch := nodes;
              true)
        compounds
  | Custom_variant.Body (body, selectors) ->
      let compounds = Variants.compounds_for_selectors selectors in
      Variants.static ds.variants cv.name
        (fun target _ ->
          match children target with
          | None -> false
          | Some ch ->
              let clone = ref (clone_nodes body) in
              let take = take_from ch in
              ignore
                (walk clone (fun n u ->
                     match (n, u.parent) with
                     | At_rule _, Some (At_root _) -> Continue
                     | At_rule at, _ when at.name = "@slot" ->
                         replace_with u (take ());
                         Skip
                     | At_rule at, _ when at.name = "@keyframes" || at.name = "@property" ->
                         replace_with u [ at_root [ n ] ];
                         Skip
                     | _ -> Continue));
              ch := !clone;
              true)
        compounds
