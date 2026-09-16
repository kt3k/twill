(* Typography, background, layout, table, scrolling, and interactivity extensions (SPEC §10.8). *)

open Ast
open Candidate
open Utilities
open Utils

let property = Builtin_variants.property

let blend_modes =
  [ "normal"; "multiply"; "screen"; "overlay"; "darken"; "lighten"; "color-dodge"; "color-burn"; "hard-light"; "soft-light";
    "difference"; "exclusion"; "hue"; "saturation"; "color"; "luminosity" ]

let numeric_variables = [ "--tw-ordinal"; "--tw-slashed-zero"; "--tw-numeric-figure"; "--tw-numeric-spacing"; "--tw-numeric-fraction" ]

(* Registers the extension utilities. *)
let register u theme =
  let stat name declarations = static_utility u name declarations in
  let stat_fn name f = static_utility_fn u name f in
  let fn root desc = functional_utility u theme root desc in
  let single property = fun value _ -> [ decl property value ] in
  let pixels = bare_suffix "px" in
  (* Typography. *)
  stat "line-clamp-none"
    [ ("overflow", "visible"); ("display", "block"); ("-webkit-box-orient", "horizontal"); ("-webkit-line-clamp", "unset") ];
  fn "line-clamp"
    (describe ~theme_keys:[ "--line-clamp" ] ~handle_bare_value:bare_integer (fun value _ ->
         [ decl "overflow" "hidden"; decl "display" "-webkit-box"; decl "-webkit-box-orient" "vertical"; decl "-webkit-line-clamp" value ]));
  List.iter (fun style -> stat ("decoration-" ^ style) [ ("text-decoration-style", style) ]) [ "solid"; "double"; "dotted"; "dashed"; "wavy" ];
  stat "decoration-from-font" [ ("text-decoration-thickness", "from-font") ];
  stat "decoration-auto" [ ("text-decoration-thickness", "auto") ];
  functional u "decoration" (fun c ->
      match c.cvalue with
      | None -> not_handled
      | Some { vkind = Arbitrary; value; _ } -> (
          match arbitrary_type c [ "color"; "length"; "percentage" ] with
          | "length" | "percentage" -> if c.cmodifier <> None then not_handled else handled [ decl "text-decoration-thickness" value ]
          | _ -> (
              match as_color value c.cmodifier theme with
              | Some r -> handled [ decl "text-decoration-color" r ]
              | None -> not_handled))
      | Some named -> (
          match resolve_theme_color c [ "--text-decoration-color"; "--color" ] theme with
          | Some color -> handled [ decl "text-decoration-color" color ]
          | None -> (
              if c.cmodifier <> None then not_handled
              else
                match Theme.resolve theme (Some named.value) [ "--text-decoration-thickness" ] 0 with
                | Some t -> handled [ decl "text-decoration-thickness" t ]
                | None -> (
                    match pixels named with
                    | Some bare -> handled [ decl "text-decoration-thickness" bare ]
                    | None -> not_handled))));
  List.iter (fun v -> stat ("hyphens-" ^ v) [ ("-webkit-hyphens", v); ("hyphens", v) ]) [ "none"; "manual"; "auto" ];
  stat "normal-nums" [ ("font-variant-numeric", "normal") ];
  let numeric_value = String.concat " " (List.map (fun v -> "var(" ^ v ^ ",)") numeric_variables) in
  let numeric name variable =
    stat_fn name (fun () ->
        List.map (fun v -> property v None) numeric_variables @ [ decl variable name; decl "font-variant-numeric" numeric_value ])
  in
  numeric "ordinal" "--tw-ordinal";
  numeric "slashed-zero" "--tw-slashed-zero";
  numeric "lining-nums" "--tw-numeric-figure";
  numeric "oldstyle-nums" "--tw-numeric-figure";
  numeric "proportional-nums" "--tw-numeric-spacing";
  numeric "tabular-nums" "--tw-numeric-spacing";
  numeric "diagonal-fractions" "--tw-numeric-fraction";
  numeric "stacked-fractions" "--tw-numeric-fraction";
  List.iter (fun v -> stat ("align-" ^ v) [ ("vertical-align", v) ])
    [ "baseline"; "top"; "middle"; "bottom"; "text-top"; "text-bottom"; "sub"; "super" ];
  fn "align" (describe (single "vertical-align"));
  List.iter (fun v -> stat ("font-stretch-" ^ v) [ ("font-stretch", v) ])
    [ "ultra-condensed"; "extra-condensed"; "condensed"; "semi-condensed"; "normal"; "semi-expanded"; "expanded"; "extra-expanded"; "ultra-expanded" ];
  fn "font-stretch"
    (describe ~theme_keys:[ "--font-stretch" ]
       ~handle_bare_value:(fun (v : candidate_value) ->
         if has_suffix v.value "%" && is_positive_integer (trim_suffix v.value "%") then Some v.value else None)
       (single "font-stretch"));
  stat "text-shadow-none" [ ("text-shadow", "none") ];
  functional u "text-shadow" (fun c ->
      let color value = handled [ property "--tw-text-shadow-color" None; decl "--tw-text-shadow-color" value ] in
      let wrap value =
        match Shadows.wrap_shadow_colors value c theme "--tw-text-shadow-color" false with
        | Some replaced -> handled [ decl "text-shadow" replaced ]
        | None -> not_handled
      in
      match c.cvalue with
      | None -> ( match Theme.resolve_value theme None [ "--text-shadow" ] with Some v -> wrap v | None -> not_handled)
      | Some { vkind = Arbitrary; value; _ } -> (
          if arbitrary_type c [ "color" ] = "color" then
            match as_color value c.cmodifier theme with Some r -> color r | None -> not_handled
          else wrap value)
      | Some named -> (
          match resolve_theme_color c [ "--text-shadow-color"; "--color" ] theme with
          | Some theme_color -> color theme_color
          | None -> (
              match Theme.resolve_value theme (Some named.value) [ "--text-shadow" ] with
              | Some v -> wrap v
              | None -> not_handled)));
  stat "wrap-break-word" [ ("overflow-wrap", "break-word") ];
  stat "wrap-anywhere" [ ("overflow-wrap", "anywhere") ];
  stat "wrap-normal" [ ("overflow-wrap", "normal") ];
  stat "list-image-none" [ ("list-style-image", "none") ];
  fn "list-image" (describe (single "list-style-image"));
  stat_fn "content-none" (fun () -> [ property "--tw-content" (Some "\"\""); decl "--tw-content" "none"; decl "content" "none" ]);
  (* Backgrounds and blending. *)
  fn "bg-position" (describe (single "background-position"));
  fn "bg-size" (describe (single "background-size"));
  List.iter
    (fun mode ->
      stat ("bg-blend-" ^ mode) [ ("background-blend-mode", mode) ];
      stat ("mix-blend-" ^ mode) [ ("mix-blend-mode", mode) ])
    blend_modes;
  stat "mix-blend-plus-darker" [ ("mix-blend-mode", "plus-darker") ];
  stat "mix-blend-plus-lighter" [ ("mix-blend-mode", "plus-lighter") ];
  (* Layout. *)
  stat_fn "container" (fun () ->
      let breakpoints =
        List.filter_map
          (fun (e : Theme.namespace_entry) -> if e.self || has_prefix e.key "--" then None else Some e.value)
          (Theme.namespace theme "--breakpoint")
      in
      let breakpoints = List.stable_sort Builtin_variants.compare_query_values breakpoints in
      decl "width" "100%"
      :: List.map (fun value -> at_rule "@media" ("(width >= " ^ value ^ ")") ~nodes:[ decl "max-width" value ]) breakpoints);
  List.iter
    (fun v ->
      stat ("break-after-" ^ v) [ ("break-after", v) ];
      stat ("break-before-" ^ v) [ ("break-before", v) ])
    [ "auto"; "avoid"; "all"; "avoid-page"; "page"; "left"; "right"; "column" ];
  List.iter (fun v -> stat ("break-inside-" ^ v) [ ("break-inside", v) ]) [ "auto"; "avoid"; "avoid-page"; "avoid-column" ];
  stat "box-decoration-clone" [ ("box-decoration-break", "clone") ];
  stat "box-decoration-slice" [ ("box-decoration-break", "slice") ];
  fn "object" (describe (single "object-position"));
  (* Tables. *)
  stat "border-collapse" [ ("border-collapse", "collapse") ];
  stat "border-separate" [ ("border-collapse", "separate") ];
  let border_spacing name variables =
    spacing_utility u theme name [ "--border-spacing"; "--spacing" ]
      (fun value ->
        [ property "--tw-border-spacing-x" (Some "0"); property "--tw-border-spacing-y" (Some "0") ]
        @ List.map (fun v -> decl v value) variables
        @ [ decl "border-spacing" "var(--tw-border-spacing-x) var(--tw-border-spacing-y)" ])
      false false
  in
  border_spacing "border-spacing" [ "--tw-border-spacing-x"; "--tw-border-spacing-y" ];
  border_spacing "border-spacing-x" [ "--tw-border-spacing-x" ];
  border_spacing "border-spacing-y" [ "--tw-border-spacing-y" ];
  stat "table-auto" [ ("table-layout", "auto") ];
  stat "table-fixed" [ ("table-layout", "fixed") ];
  stat "caption-top" [ ("caption-side", "top") ];
  stat "caption-bottom" [ ("caption-side", "bottom") ];
  (* Scrolling and touch. *)
  List.iter
    (fun (suffix, side) ->
      let margin = "scroll-margin" ^ side and padding = "scroll-padding" ^ side in
      spacing_utility u theme ("scroll-m" ^ suffix) [ "--scroll-margin"; "--spacing" ] (fun value -> [ decl margin value ]) true false;
      spacing_utility u theme ("scroll-p" ^ suffix) [ "--scroll-padding"; "--spacing" ] (fun value -> [ decl padding value ]) false false)
    [ ("", ""); ("x", "-inline"); ("y", "-block"); ("s", "-inline-start"); ("e", "-inline-end"); ("t", "-top"); ("r", "-right");
      ("b", "-bottom"); ("l", "-left") ];
  stat "snap-none" [ ("scroll-snap-type", "none") ];
  List.iter
    (fun axis ->
      stat_fn ("snap-" ^ axis) (fun () ->
          [ property "--tw-scroll-snap-strictness" (Some "proximity");
            decl "scroll-snap-type" (axis ^ " var(--tw-scroll-snap-strictness)") ]))
    [ "x"; "y"; "both" ];
  List.iter
    (fun strictness ->
      stat_fn ("snap-" ^ strictness) (fun () ->
          [ property "--tw-scroll-snap-strictness" (Some "proximity"); decl "--tw-scroll-snap-strictness" strictness ]))
    [ "mandatory"; "proximity" ];
  List.iter (fun v -> stat ("snap-" ^ v) [ ("scroll-snap-align", v) ]) [ "start"; "end"; "center" ];
  stat "snap-align-none" [ ("scroll-snap-align", "none") ];
  stat "snap-normal" [ ("scroll-snap-stop", "normal") ];
  stat "snap-always" [ ("scroll-snap-stop", "always") ];
  List.iter (fun v -> stat ("touch-" ^ v) [ ("touch-action", v) ]) [ "auto"; "none"; "manipulation" ];
  let touch value variable =
    stat_fn ("touch-" ^ value) (fun () ->
        [ property "--tw-pan-x" None; property "--tw-pan-y" None; property "--tw-pinch-zoom" None; decl variable value;
          decl "touch-action" "var(--tw-pan-x,) var(--tw-pan-y,) var(--tw-pinch-zoom,)" ])
  in
  List.iter (fun v -> touch v "--tw-pan-x") [ "pan-x"; "pan-left"; "pan-right" ];
  List.iter (fun v -> touch v "--tw-pan-y") [ "pan-y"; "pan-up"; "pan-down" ];
  touch "pinch-zoom" "--tw-pinch-zoom";
  (* Interactivity and SVG. *)
  functional u "stroke" (fun c ->
      match c.cvalue with
      | None -> not_handled
      | Some { vkind = Arbitrary; value; _ } -> (
          match arbitrary_type c [ "color"; "length"; "number"; "percentage" ] with
          | "length" | "number" | "percentage" -> if c.cmodifier <> None then not_handled else handled [ decl "stroke-width" value ]
          | _ -> ( match as_color value c.cmodifier theme with Some r -> handled [ decl "stroke" r ] | None -> not_handled))
      | Some named -> (
          match resolve_theme_color c [ "--stroke"; "--color" ] theme with
          | Some color -> handled [ decl "stroke" color ]
          | None -> (
              if c.cmodifier <> None then not_handled
              else
                match Theme.resolve theme (Some named.value) [ "--stroke-width" ] 0 with
                | Some w -> handled [ decl "stroke-width" w ]
                | None -> if is_positive_integer named.value then handled [ decl "stroke-width" named.value ] else not_handled)));
  List.iter (fun (name, value) -> stat ("scheme-" ^ name) [ ("color-scheme", value) ])
    [ ("normal", "normal"); ("dark", "dark"); ("light", "light"); ("light-dark", "light dark"); ("only-dark", "only dark"); ("only-light", "only light") ];
  stat "field-sizing-content" [ ("field-sizing", "content") ];
  stat "field-sizing-fixed" [ ("field-sizing", "fixed") ];
  stat "forced-color-adjust-auto" [ ("forced-color-adjust", "auto") ];
  stat "forced-color-adjust-none" [ ("forced-color-adjust", "none") ]
