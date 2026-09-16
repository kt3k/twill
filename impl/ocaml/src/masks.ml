(* Mask utilities (SPEC §10.8, "Masks"). *)

open Ast
open Candidate
open Utilities
open Utils

let property = Builtin_variants.property
let mask_edges = [ "top"; "right"; "bottom"; "left" ]
let mask_white = "linear-gradient(#fff, #fff)"

let mask_stop_registrations name =
  [ property ("--tw-mask-" ^ name ^ "-from-color") (Some "black"); property ("--tw-mask-" ^ name ^ "-from-position") (Some "0%");
    property ("--tw-mask-" ^ name ^ "-to-color") (Some "transparent"); property ("--tw-mask-" ^ name ^ "-to-position") (Some "100%") ]

(* The registrations emitted by every gradient mask utility. *)
let mask_registrations () =
  List.concat_map (fun edge -> property ("--tw-mask-" ^ edge) (Some mask_white) :: mask_stop_registrations edge) mask_edges
  @ [ property "--tw-mask-linear" (Some mask_white); property "--tw-mask-linear-position" (Some "0deg") ]
  @ mask_stop_registrations "linear"
  @ [ property "--tw-mask-radial" (Some mask_white); property "--tw-mask-radial-shape" (Some "ellipse");
      property "--tw-mask-radial-size" (Some "farthest-corner"); property "--tw-mask-radial-position" (Some "center") ]
  @ mask_stop_registrations "radial"
  @ [ property "--tw-mask-conic" (Some mask_white); property "--tw-mask-conic-position" (Some "0deg") ]
  @ mask_stop_registrations "conic"

let mask_edge_list = "var(--tw-mask-left), var(--tw-mask-right), var(--tw-mask-bottom), var(--tw-mask-top)"

let mask_linear =
  "linear-gradient(var(--tw-mask-linear-position), var(--tw-mask-linear-from-color) var(--tw-mask-linear-from-position), var(--tw-mask-linear-to-color) var(--tw-mask-linear-to-position))"

let mask_radial =
  "radial-gradient(var(--tw-mask-radial-shape) var(--tw-mask-radial-size) at var(--tw-mask-radial-position), var(--tw-mask-radial-from-color) var(--tw-mask-radial-from-position), var(--tw-mask-radial-to-color) var(--tw-mask-radial-to-position))"

let mask_conic =
  "conic-gradient(from var(--tw-mask-conic-position), var(--tw-mask-conic-from-color) var(--tw-mask-conic-from-position), var(--tw-mask-conic-to-color) var(--tw-mask-conic-to-position))"

let mask_edge_gradient edge =
  "linear-gradient(to " ^ edge ^ ", var(--tw-mask-" ^ edge ^ "-from-color) var(--tw-mask-" ^ edge ^ "-from-position), var(--tw-mask-"
  ^ edge ^ "-to-color) var(--tw-mask-" ^ edge ^ "-to-position))"

let mask_positions =
  [ ("top", "top"); ("top-left", "top left"); ("top-right", "top right"); ("left", "left"); ("center", "center"); ("right", "right");
    ("bottom", "bottom"); ("bottom-left", "bottom left"); ("bottom-right", "bottom right") ]

(* Registers the mask utilities. *)
let register u theme =
  let stat name declarations = static_utility u name declarations in
  let fn root desc = functional_utility u theme root desc in
  let single property = fun value _ -> [ decl property value ] in
  stat "mask-none" [ ("mask-image", "none") ];
  fn "mask" (describe (single "mask-image"));
  List.iter (fun v -> stat ("mask-" ^ v) [ ("mask-composite", v) ]) [ "add"; "subtract"; "intersect"; "exclude" ];
  stat "mask-alpha" [ ("mask-mode", "alpha") ];
  stat "mask-luminance" [ ("mask-mode", "luminance") ];
  stat "mask-match" [ ("mask-mode", "match-source") ];
  stat "mask-type-alpha" [ ("mask-type", "alpha") ];
  stat "mask-type-luminance" [ ("mask-type", "luminance") ];
  List.iter (fun v -> stat ("mask-" ^ v) [ ("mask-size", v) ]) [ "auto"; "cover"; "contain" ];
  fn "mask-size" (describe (single "mask-size"));
  List.iter
    (fun box ->
      stat ("mask-clip-" ^ box) [ ("mask-clip", box ^ "-box") ];
      stat ("mask-origin-" ^ box) [ ("mask-origin", box ^ "-box") ])
    [ "border"; "padding"; "content"; "fill"; "stroke"; "view" ];
  stat "mask-no-clip" [ ("mask-clip", "no-clip") ];
  List.iter (fun (name, value) -> stat ("mask-" ^ name) [ ("mask-position", value) ]) mask_positions;
  fn "mask-position" (describe (single "mask-position"));
  List.iter (fun (name, value) -> stat ("mask-" ^ name) [ ("mask-repeat", value) ])
    [ ("repeat", "repeat"); ("no-repeat", "no-repeat"); ("repeat-x", "repeat-x"); ("repeat-y", "repeat-y"); ("repeat-space", "space"); ("repeat-round", "round") ];
  (* A stop is a (kind, value) pair where kind is "position" or "color". *)
  let stop_of (c : candidate) =
    match c.cvalue with
    | None -> None
    | Some { vkind = Arbitrary; value; _ } -> (
        match arbitrary_type c [ "length"; "percentage"; "color" ] with
        | "length" | "percentage" -> if c.cmodifier <> None then None else Some ("position", value)
        | _ -> ( match as_color value c.cmodifier theme with Some r -> Some ("color", r) | None -> None))
    | Some named -> (
        match resolve_theme_color c [ "--color" ] theme with
        | Some color -> Some ("color", color)
        | None ->
            if c.cmodifier <> None then None
            else
              let v = named.value in
              if String.length v > 1 && has_suffix v "%" && is_positive_integer (trim_suffix v "%") then Some ("position", v)
              else if is_positive_integer v then
                if Theme.resolve theme None [ "--spacing" ] 0 = None then None else Some ("position", "--spacing(" ^ v ^ ")")
              else None)
  in
  let composed compositions =
    mask_registrations ()
    @ [ decl "mask-image" "var(--tw-mask-linear), var(--tw-mask-radial), var(--tw-mask-conic)"; decl "mask-composite" "intersect" ]
    @ List.map (fun (p, v) -> decl p v) compositions
  in
  let stop_utility root side compositions variables =
    functional u (root ^ "-" ^ side) (fun c ->
        match stop_of c with
        | None -> not_handled
        | Some (kind, value) ->
            handled (composed compositions @ List.map (fun v -> decl (v ^ "-" ^ side ^ "-" ^ kind) value) variables))
  in
  List.iter
    (fun (name, edges) ->
      let compositions = ("--tw-mask-linear", mask_edge_list) :: List.map (fun e -> ("--tw-mask-" ^ e, mask_edge_gradient e)) edges in
      let variables = List.map (fun e -> "--tw-mask-" ^ e) edges in
      stop_utility ("mask-" ^ name) "from" compositions variables;
      stop_utility ("mask-" ^ name) "to" compositions variables)
    [ ("t", [ "top" ]); ("r", [ "right" ]); ("b", [ "bottom" ]); ("l", [ "left" ]); ("x", [ "left"; "right" ]); ("y", [ "top"; "bottom" ]) ];
  let angle root variable composition =
    let compile (c : candidate) negative =
      match c.cvalue with
      | None -> not_handled
      | Some _ when c.cmodifier <> None -> not_handled
      | Some { vkind = Arbitrary; value; _ } ->
          if negative || arbitrary_type c [ "angle" ] <> "angle" then not_handled
          else handled (composed [ (variable, composition) ] @ [ decl (variable ^ "-position") value ])
      | Some named ->
          if not (is_positive_integer named.value) then not_handled
          else
            let value = if negative then "calc(" ^ named.value ^ "deg * -1)" else named.value ^ "deg" in
            handled (composed [ (variable, composition) ] @ [ decl (variable ^ "-position") value ])
    in
    functional u root (fun c -> compile c false);
    functional u ("-" ^ root) (fun c -> compile c true)
  in
  angle "mask-linear" "--tw-mask-linear" mask_linear;
  stop_utility "mask-linear" "from" [ ("--tw-mask-linear", mask_linear) ] [ "--tw-mask-linear" ];
  stop_utility "mask-linear" "to" [ ("--tw-mask-linear", mask_linear) ] [ "--tw-mask-linear" ];
  functional u "mask-radial" (fun c ->
      match c.cvalue with
      | Some { vkind = Arbitrary; value; _ } when c.cmodifier = None ->
          handled (composed [ ("--tw-mask-radial", mask_radial) ] @ [ decl "--tw-mask-radial-size" value ])
      | _ -> not_handled);
  stop_utility "mask-radial" "from" [ ("--tw-mask-radial", mask_radial) ] [ "--tw-mask-radial" ];
  stop_utility "mask-radial" "to" [ ("--tw-mask-radial", mask_radial) ] [ "--tw-mask-radial" ];
  stat "mask-circle" [ ("--tw-mask-radial-shape", "circle") ];
  stat "mask-ellipse" [ ("--tw-mask-radial-shape", "ellipse") ];
  List.iter (fun size -> stat ("mask-radial-" ^ size) [ ("--tw-mask-radial-size", size) ])
    [ "closest-side"; "farthest-side"; "closest-corner"; "farthest-corner" ];
  List.iter (fun (name, value) -> stat ("mask-radial-at-" ^ name) [ ("--tw-mask-radial-position", value) ]) mask_positions;
  fn "mask-radial-at" (describe (single "--tw-mask-radial-position"));
  angle "mask-conic" "--tw-mask-conic" mask_conic;
  stop_utility "mask-conic" "from" [ ("--tw-mask-conic", mask_conic) ] [ "--tw-mask-conic" ];
  stop_utility "mask-conic" "to" [ ("--tw-mask-conic", mask_conic) ] [ "--tw-mask-conic" ]
