(* Gradient utilities (SPEC §10.8, "Gradients"). *)

open Ast
open Candidate
open Utilities
open Utils

(* The stops value for from-* and to-*. *)
let gradient_stops =
  "var(--tw-gradient-via-stops, var(--tw-gradient-position), var(--tw-gradient-from) var(--tw-gradient-from-position), var(--tw-gradient-to) var(--tw-gradient-to-position))"

(* The stops value set by via-*. *)
let gradient_via_stops =
  "var(--tw-gradient-position), var(--tw-gradient-from) var(--tw-gradient-from-position), var(--tw-gradient-via) var(--tw-gradient-via-position), var(--tw-gradient-to) var(--tw-gradient-to-position)"

let interpolation_methods =
  [ "srgb"; "srgb-linear"; "display-p3"; "a98-rgb"; "prophoto-rgb"; "rec2020"; "lab"; "oklab"; "xyz"; "xyz-d50"; "xyz-d65";
    "hsl"; "hwb"; "lch"; "oklch" ]

let hue_methods = [ "longer"; "shorter"; "increasing"; "decreasing" ]

let gradient_sides =
  [ ("t", "top"); ("tr", "top right"); ("r", "right"); ("br", "bottom right"); ("b", "bottom"); ("bl", "bottom left");
    ("l", "left"); ("tl", "top left") ]

(* The registrations emitted by every stop utility. *)
let gradient_registrations () =
  let open Builtin_variants in
  let transparent = Some "#0000" in
  [ property "--tw-gradient-position" None;
    property_registration "--tw-gradient-from" transparent "<color>" false;
    property_registration "--tw-gradient-via" transparent "<color>" false;
    property_registration "--tw-gradient-to" transparent "<color>" false;
    property "--tw-gradient-stops" None; property "--tw-gradient-via-stops" None;
    property_registration "--tw-gradient-from-position" (Some "0%") "<length-percentage>" false;
    property_registration "--tw-gradient-via-position" (Some "50%") "<length-percentage>" false;
    property_registration "--tw-gradient-to-position" (Some "100%") "<length-percentage>" false ]

(* Resolves an interpolation modifier. *)
let interpolation (modifier : modifier option) =
  match modifier with
  | None -> Some "in oklab"
  | Some { mkind = Arbitrary; mvalue } -> Some mvalue
  | Some { mvalue; _ } ->
      if List.mem mvalue interpolation_methods then Some ("in " ^ mvalue)
      else if List.mem mvalue hue_methods then Some ("in oklch " ^ mvalue ^ " hue")
      else None

(* Registers the gradient utilities. *)
let register u theme =
  let image shape = decl "background-image" (shape ^ "-gradient(var(--tw-gradient-stops))") in
  let positioned position modifier shape =
    match interpolation modifier with
    | None -> not_handled
    | Some method_ ->
        let value = if position = "" then method_ else position ^ " " ^ method_ in
        handled [ decl "--tw-gradient-position" value; image shape ]
  in
  let linear (c : candidate) negative legacy =
    match c.cvalue with
    | None -> not_handled
    | Some { vkind = Arbitrary; value; _ } ->
        if legacy || negative then not_handled
        else if arbitrary_type c [ "angle" ] = "angle" then positioned value c.cmodifier "linear"
        else if c.cmodifier <> None then not_handled
        else handled [ decl "background-image" ("linear-gradient(" ^ value ^ ")") ]
    | Some { value = named; _ } ->
        if has_prefix named "to-" then
          if negative then not_handled
          else
            match List.assoc_opt (after named 3) gradient_sides with
            | Some side -> positioned ("to " ^ side) c.cmodifier "linear"
            | None -> not_handled
        else if legacy || not (is_positive_integer named) then not_handled
        else
          let angle = if negative then "calc(" ^ named ^ "deg * -1)" else named ^ "deg" in
          positioned angle c.cmodifier "linear"
  in
  functional u "bg-linear" (fun c -> linear c false false);
  functional u "-bg-linear" (fun c -> linear c true false);
  functional u "bg-gradient" (fun c -> linear c false true);
  functional u "bg-radial" (fun c ->
      match c.cvalue with
      | None -> positioned "" c.cmodifier "radial"
      | Some { vkind = Arbitrary; value; _ } when c.cmodifier = None ->
          handled [ decl "--tw-gradient-position" value; image "radial" ]
      | _ -> not_handled);
  let conic (c : candidate) negative =
    match c.cvalue with
    | None -> if negative then not_handled else positioned "" c.cmodifier "conic"
    | Some { vkind = Arbitrary; value; _ } ->
        if negative || c.cmodifier <> None then not_handled else handled [ decl "--tw-gradient-position" value; image "conic" ]
    | Some { value = named; _ } ->
        if not (is_positive_integer named) then not_handled
        else
          let angle = if negative then "calc(" ^ named ^ "deg * -1)" else named ^ "deg" in
          positioned ("from " ^ angle) c.cmodifier "conic"
  in
  functional u "bg-conic" (fun c -> conic c false);
  functional u "-bg-conic" (fun c -> conic c true);
  let stop name color_variable position_variable stops =
    let color value =
      handled (gradient_registrations () @ [ decl color_variable value ] @ List.map (fun (p, v) -> decl p v) stops)
    in
    let position value = handled (gradient_registrations () @ [ decl position_variable value ]) in
    functional u name (fun c ->
        match c.cvalue with
        | None -> not_handled
        | Some { vkind = Arbitrary; value; _ } -> (
            match arbitrary_type c [ "color"; "length"; "percentage" ] with
            | "length" | "percentage" -> if c.cmodifier <> None then not_handled else position value
            | _ -> ( match as_color value c.cmodifier theme with Some r -> color r | None -> not_handled))
        | Some { value = named; _ } -> (
            match resolve_theme_color c [ "--background-color"; "--color" ] theme with
            | Some theme_color -> color theme_color
            | None ->
                if c.cmodifier <> None then not_handled
                else if has_suffix named "%" && is_positive_integer (trim_suffix named "%") then position named
                else not_handled))
  in
  stop "from" "--tw-gradient-from" "--tw-gradient-from-position" [ ("--tw-gradient-stops", gradient_stops) ];
  stop "via" "--tw-gradient-via" "--tw-gradient-via-position"
    [ ("--tw-gradient-via-stops", gradient_via_stops); ("--tw-gradient-stops", "var(--tw-gradient-via-stops)") ];
  stop "to" "--tw-gradient-to" "--tw-gradient-to-position" [ ("--tw-gradient-stops", gradient_stops) ]
