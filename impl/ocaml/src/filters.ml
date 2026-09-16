(* Filter and backdrop-filter utilities (SPEC §10.8, "Filters"). *)

open Ast
open Candidate
open Utilities

let property = Builtin_variants.property
let filter_names = [ "blur"; "brightness"; "contrast"; "grayscale"; "hue-rotate"; "invert"; "saturate"; "sepia"; "drop-shadow" ]
let backdrop_names = [ "blur"; "brightness"; "contrast"; "grayscale"; "hue-rotate"; "invert"; "opacity"; "saturate"; "sepia" ]
let composed_filter prefix names = String.concat " " (List.map (fun n -> "var(" ^ prefix ^ n ^ ",)") names)

(* The composed filter value. *)
let filter = composed_filter "--tw-" filter_names

(* The composed backdrop-filter value. *)
let backdrop_filter = composed_filter "--tw-backdrop-" backdrop_names

let filter_registrations backdrop =
  let prefix, names = if backdrop then ("--tw-backdrop-", backdrop_names) else ("--tw-", filter_names) in
  List.map (fun n -> property (prefix ^ n) None) names

let filter_declarations backdrop =
  if backdrop then [ decl "-webkit-backdrop-filter" backdrop_filter; decl "backdrop-filter" backdrop_filter ]
  else [ decl "filter" filter ]

(* Registers the filter and backdrop-filter utilities. *)
let register u theme =
  let stat_fn name f = static_utility_fn u name f in
  let fn root desc = functional_utility u theme root desc in
  let percentage = bare_suffix "%" and degrees = bare_suffix "deg" in
  List.iter
    (fun backdrop ->
      let prefix, variable = if backdrop then ("backdrop-", "--tw-backdrop-") else ("", "--tw-") in
      let keys name = if backdrop then [ "--backdrop-" ^ name; "--" ^ name ] else [ "--" ^ name ] in
      let emit name value = filter_registrations backdrop @ [ decl (variable ^ name) value ] @ filter_declarations backdrop in
      let direct value =
        if backdrop then [ decl "-webkit-backdrop-filter" value; decl "backdrop-filter" value ] else [ decl "filter" value ]
      in
      stat_fn (prefix ^ "filter-none") (fun () -> direct "none");
      stat_fn (prefix ^ "filter") (fun () -> filter_registrations backdrop @ filter_declarations backdrop);
      fn (prefix ^ "filter") (describe (fun value _ -> direct value));
      fn (prefix ^ "blur") (describe ~theme_keys:(keys "blur") (fun value _ -> emit "blur" ("blur(" ^ value ^ ")")));
      stat_fn (prefix ^ "blur-none") (fun () -> emit "blur" "");
      List.iter
        (fun name ->
          fn (prefix ^ name)
            (describe ~theme_keys:(keys name) ~handle_bare_value:percentage (fun value _ -> emit name (name ^ "(" ^ value ^ ")"))))
        [ "brightness"; "contrast"; "saturate" ];
      List.iter
        (fun name ->
          fn (prefix ^ name)
            (describe ~theme_keys:(keys name) ~default_value:"100%" ~handle_bare_value:percentage (fun value _ ->
                 emit name (name ^ "(" ^ value ^ ")"))))
        [ "grayscale"; "invert"; "sepia" ];
      fn (prefix ^ "hue-rotate")
        (describe ~theme_keys:(keys "hue-rotate") ~supports_negative:true ~handle_bare_value:degrees (fun value _ ->
             emit "hue-rotate" ("hue-rotate(" ^ value ^ ")")));
      if backdrop then
        fn "backdrop-opacity"
          (describe ~theme_keys:[ "--backdrop-opacity"; "--opacity" ] ~handle_bare_value:percentage (fun value _ ->
               emit "opacity" ("opacity(" ^ value ^ ")")))
      else begin
        stat_fn "drop-shadow-none" (fun () -> emit "drop-shadow" "");
        functional u "drop-shadow" (fun c ->
            let color value = handled [ property "--tw-drop-shadow-color" None; decl "--tw-drop-shadow-color" value ] in
            let wrap value =
              match Shadows.wrap_shadow_colors value c theme "--tw-drop-shadow-color" false with
              | None -> not_handled
              | Some replaced ->
                  let shadows = List.map (fun s -> "drop-shadow(" ^ String.trim s ^ ")") (Utils.segment replaced ',') in
                  handled (emit "drop-shadow" (String.concat " " shadows))
            in
            match c.cvalue with
            | None -> (
                match Theme.resolve_value theme None [ "--drop-shadow" ] with Some v -> wrap v | None -> not_handled)
            | Some { vkind = Arbitrary; value; _ } -> (
                if arbitrary_type c [ "color" ] = "color" then
                  match as_color value c.cmodifier theme with Some r -> color r | None -> not_handled
                else wrap value)
            | Some { value; _ } -> (
                match resolve_theme_color c [ "--drop-shadow-color"; "--color" ] theme with
                | Some theme_color -> color theme_color
                | None -> (
                    match Theme.resolve_value theme (Some value) [ "--drop-shadow" ] with
                    | Some v -> wrap v
                    | None -> not_handled)))
      end)
    [ false; true ]
