(* Transform utilities (SPEC §10.8, "Transforms"). *)

open Ast
open Candidate
open Utilities

let property = Builtin_variants.property

(* The composed transform value. *)
let transform = "var(--tw-rotate-x,) var(--tw-rotate-y,) var(--tw-rotate-z,) var(--tw-skew-x,) var(--tw-skew-y,)"
let translate_2d = "var(--tw-translate-x) var(--tw-translate-y)"
let translate_3d = translate_2d ^ " var(--tw-translate-z)"
let scale_2d = "var(--tw-scale-x) var(--tw-scale-y)"
let scale_3d = scale_2d ^ " var(--tw-scale-z)"

let transform_registrations () =
  List.map (fun n -> property n None) [ "--tw-rotate-x"; "--tw-rotate-y"; "--tw-rotate-z"; "--tw-skew-x"; "--tw-skew-y" ]

let translate_registrations () =
  List.map (fun n -> property n (Some "0")) [ "--tw-translate-x"; "--tw-translate-y"; "--tw-translate-z" ]

let scale_registrations () = List.map (fun n -> property n (Some "1")) [ "--tw-scale-x"; "--tw-scale-y"; "--tw-scale-z" ]

let transform_origins =
  [ ("center", "center"); ("top", "top"); ("top-right", "top right"); ("right", "right");
    ("bottom-right", "bottom right"); ("bottom", "bottom"); ("bottom-left", "bottom left"); ("left", "left");
    ("top-left", "top left") ]

(* Registers the transform utilities. *)
let register u theme =
  let stat name declarations = static_utility u name declarations in
  let stat_fn name f = static_utility_fn u name f in
  let fn root desc = functional_utility u theme root desc in
  let degrees = bare_suffix "deg" and percentage = bare_suffix "%" in
  let single property = fun value _ -> [ decl property value ] in
  stat "transform-none" [ ("transform", "none") ];
  stat_fn "transform-gpu" (fun () -> transform_registrations () @ [ decl "transform" ("translateZ(0) " ^ transform) ]);
  stat_fn "transform-cpu" (fun () -> transform_registrations () @ [ decl "transform" transform ]);
  fn "transform" (describe (single "transform"));
  stat "transform-flat" [ ("transform-style", "flat") ];
  stat "transform-3d" [ ("transform-style", "preserve-3d") ];
  stat "backface-visible" [ ("backface-visibility", "visible") ];
  stat "backface-hidden" [ ("backface-visibility", "hidden") ];
  stat "perspective-none" [ ("perspective", "none") ];
  fn "perspective" (describe ~theme_keys:[ "--perspective" ] (single "perspective"));
  List.iter
    (fun (name, value) ->
      stat ("perspective-origin-" ^ name) [ ("perspective-origin", value) ];
      stat ("origin-" ^ name) [ ("transform-origin", value) ])
    transform_origins;
  fn "perspective-origin" (describe (single "perspective-origin"));
  fn "origin" (describe (single "transform-origin"));
  let translate name variables composed fractions =
    let handle value =
      translate_registrations () @ List.map (fun v -> decl v value) variables @ [ decl "translate" composed ]
    in
    spacing_utility u theme name [ "--translate"; "--spacing" ] handle true fractions;
    if fractions then begin
      stat_fn (name ^ "-full") (fun () -> handle "100%");
      stat_fn ("-" ^ name ^ "-full") (fun () -> handle "-100%")
    end
  in
  translate "translate" [ "--tw-translate-x"; "--tw-translate-y" ] translate_2d true;
  translate "translate-x" [ "--tw-translate-x" ] translate_2d true;
  translate "translate-y" [ "--tw-translate-y" ] translate_2d true;
  translate "translate-z" [ "--tw-translate-z" ] translate_3d false;
  stat_fn "translate-3d" (fun () -> translate_registrations () @ [ decl "translate" translate_3d ]);
  stat "translate-none" [ ("translate", "none") ];
  let scale_compile (c : candidate) negative =
    match c.cvalue with
    | None -> not_handled
    | Some _ when c.cmodifier <> None -> not_handled
    | Some { vkind = Arbitrary; value; _ } -> if negative then not_handled else handled [ decl "scale" value ]
    | Some named -> (
        let value = match Theme.resolve theme (Some named.value) [ "--scale" ] 0 with Some v -> Some v | None -> percentage named in
        match value with
        | None -> not_handled
        | Some value ->
            let value = if negative then "calc(" ^ value ^ " * -1)" else value in
            handled
              (scale_registrations ()
              @ [ decl "--tw-scale-x" value; decl "--tw-scale-y" value; decl "--tw-scale-z" value; decl "scale" scale_2d ]))
  in
  functional u "scale" (fun c -> scale_compile c false);
  functional u "-scale" (fun c -> scale_compile c true);
  List.iter
    (fun axis ->
      let composed = if axis = "z" then scale_3d else scale_2d in
      fn ("scale-" ^ axis)
        (describe ~theme_keys:[ "--scale" ] ~supports_negative:true ~handle_bare_value:percentage (fun value _ ->
             scale_registrations () @ [ decl ("--tw-scale-" ^ axis) value; decl "scale" composed ])))
    [ "x"; "y"; "z" ];
  stat_fn "scale-3d" (fun () -> scale_registrations () @ [ decl "scale" scale_3d ]);
  stat "scale-none" [ ("scale", "none") ];
  fn "rotate" (describe ~theme_keys:[ "--rotate" ] ~supports_negative:true ~handle_bare_value:degrees (single "rotate"));
  stat "rotate-none" [ ("rotate", "none") ];
  List.iter
    (fun axis ->
      fn ("rotate-" ^ axis)
        (describe ~theme_keys:[ "--rotate" ] ~supports_negative:true ~handle_bare_value:degrees (fun value _ ->
             transform_registrations ()
             @ [ decl ("--tw-rotate-" ^ axis) ("rotate" ^ String.uppercase_ascii axis ^ "(" ^ value ^ ")");
                 decl "transform" transform ])))
    [ "x"; "y"; "z" ];
  let skew name axes =
    fn name
      (describe ~theme_keys:[ "--skew" ] ~supports_negative:true ~handle_bare_value:degrees (fun value _ ->
           transform_registrations ()
           @ List.map (fun axis -> decl ("--tw-skew-" ^ axis) ("skew" ^ String.uppercase_ascii axis ^ "(" ^ value ^ ")")) axes
           @ [ decl "transform" transform ]))
  in
  skew "skew" [ "x"; "y" ];
  skew "skew-x" [ "x" ];
  skew "skew-y" [ "y" ]
