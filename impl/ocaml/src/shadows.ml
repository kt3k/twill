(* Shadow and ring utilities (SPEC §10.8, "Shadows and rings"). *)

open Ast
open Candidate
open Utilities
open Utils

(* The composed box-shadow value. *)
let box_shadow =
  "var(--tw-inset-shadow), var(--tw-inset-ring-shadow), var(--tw-ring-offset-shadow), var(--tw-ring-shadow), var(--tw-shadow)"

let shadow_keywords = [ "inset"; "inherit"; "initial"; "revert"; "unset" ]

let is_length_token t =
  String.length t > 0
  && (is_digit t.[0] || t.[0] = '.' || (t.[0] = '-' && String.length t > 1 && (is_digit t.[1] || t.[1] = '.')))

(* Rewrites the color of every shadow in a box-shadow value. Shadows with
   fewer than two length tokens are left unchanged. With [inset], shadows
   without the inset keyword are prefixed with it. *)
let replace_shadow_colors value f inset =
  let shadows = segment value ',' in
  String.concat ", "
    (List.map
       (fun shadow ->
         let tokens = Array.of_list (List.filter (fun t -> t <> "") (segment (String.concat " " (fields shadow)) ' ')) in
         let lengths = ref 0 and color_index = ref (-1) and has_inset = ref false in
         Array.iteri
           (fun i token ->
             if List.mem token shadow_keywords then (if token = "inset" then has_inset := true)
             else if is_length_token token then incr lengths
             else if !color_index = -1 then color_index := i)
           tokens;
         if !lengths < 2 then String.trim shadow
         else begin
           let tokens =
             if !color_index = -1 then Array.append tokens [| f "currentcolor" |]
             else begin
               tokens.(!color_index) <- f tokens.(!color_index);
               tokens
             end
           in
           let tokens = if inset && not !has_inset then Array.append [| "inset" |] tokens else tokens in
           String.concat " " (Array.to_list tokens)
         end)
       shadows)

let property = Builtin_variants.property

(* The registrations emitted by every shadow and ring utility. *)
let shadow_registrations () =
  let zero = Some "0 0 #0000" in
  [ property "--tw-shadow" zero; property "--tw-shadow-color" None; property "--tw-inset-shadow" zero;
    property "--tw-inset-shadow-color" None; property "--tw-ring-color" None; property "--tw-ring-shadow" zero;
    property "--tw-inset-ring-color" None; property "--tw-inset-ring-shadow" zero; property "--tw-ring-inset" None;
    property "--tw-ring-offset-width" (Some "0px"); property "--tw-ring-offset-color" (Some "#fff");
    property "--tw-ring-offset-shadow" zero ]

let shadow_color_keys = [ "--box-shadow-color"; "--color" ]
let ring_color_keys = [ "--ring-color"; "--color" ]

(* Wraps every shadow color in [value] through [f]; None when a color cannot resolve. *)
let wrap_shadow_colors value (c : candidate) theme color_variable inset =
  let failed = ref false in
  let replaced =
    replace_shadow_colors value
      (fun color ->
        match as_color color c.cmodifier theme with
        | Some resolved -> "var(" ^ color_variable ^ ", " ^ resolved ^ ")"
        | None ->
            failed := true;
            color)
      inset
  in
  if !failed then None else Some replaced

(* Registers the shadow and ring utilities. *)
let register u theme =
  let with_box_shadow variable value = shadow_registrations () @ [ decl variable value; decl "box-shadow" box_shadow ] in
  let color_only variable value = shadow_registrations () @ [ decl variable value ] in
  let shadow root namespace variable color_variable inset =
    static u (root ^ "-none") (fun _ -> handled (with_box_shadow variable "0 0 #0000"));
    functional u root (fun c ->
        let wrap value inset_arbitrary =
          match wrap_shadow_colors value c theme color_variable inset_arbitrary with
          | Some replaced -> handled (with_box_shadow variable replaced)
          | None -> not_handled
        in
        match c.cvalue with
        | None -> (
            match Theme.resolve_value theme None [ namespace ] with Some v -> wrap v false | None -> not_handled)
        | Some { vkind = Arbitrary; value; _ } -> (
            if arbitrary_type c [ "color" ] = "color" then
              match as_color value c.cmodifier theme with
              | Some resolved -> handled (color_only color_variable resolved)
              | None -> not_handled
            else wrap value inset)
        | Some { value; _ } -> (
            match resolve_theme_color c shadow_color_keys theme with
            | Some color -> handled (color_only color_variable color)
            | None -> (
                match Theme.resolve_value theme (Some value) [ namespace ] with
                | Some v -> wrap v false
                | None -> not_handled)))
  in
  shadow "shadow" "--shadow" "--tw-shadow" "--tw-shadow-color" false;
  shadow "inset-shadow" "--inset-shadow" "--tw-inset-shadow" "--tw-inset-shadow-color" true;
  let ring root variable color_variable shadow_for =
    functional u root (fun c ->
        let width value = handled (with_box_shadow variable (shadow_for value)) in
        match c.cvalue with
        | None ->
            if c.cmodifier <> None then not_handled
            else width (match Theme.get theme [ "--default-ring-width" ] with Some w -> w | None -> "1px")
        | Some { vkind = Arbitrary; value; _ } -> (
            match arbitrary_type c [ "color"; "length"; "line-width" ] with
            | "color" -> (
                match as_color value c.cmodifier theme with
                | Some resolved -> handled (color_only color_variable resolved)
                | None -> not_handled)
            | "length" | "line-width" -> if c.cmodifier <> None then not_handled else width value
            | _ -> not_handled)
        | Some { value; _ } -> (
            match resolve_theme_color c ring_color_keys theme with
            | Some color -> handled (color_only color_variable color)
            | None -> (
                if c.cmodifier <> None then not_handled
                else
                  match Theme.resolve theme (Some value) [ "--ring-width" ] 0 with
                  | Some w -> width w
                  | None -> if is_positive_integer value then width (value ^ "px") else not_handled)))
  in
  ring "ring" "--tw-ring-shadow" "--tw-ring-color" (fun w ->
      "var(--tw-ring-inset,) 0 0 0 calc(" ^ w ^ " + var(--tw-ring-offset-width)) var(--tw-ring-color, currentcolor)");
  ring "inset-ring" "--tw-inset-ring-shadow" "--tw-inset-ring-color" (fun w ->
      "inset 0 0 0 " ^ w ^ " var(--tw-inset-ring-color, currentcolor)");
  static u "ring-inset" (fun _ -> handled (color_only "--tw-ring-inset" "inset"));
  functional u "ring-offset" (fun c ->
      let width value =
        handled
          (shadow_registrations ()
          @ [ decl "--tw-ring-offset-width" value;
              decl "--tw-ring-offset-shadow"
                "var(--tw-ring-inset,) 0 0 0 var(--tw-ring-offset-width) var(--tw-ring-offset-color)" ])
      in
      match c.cvalue with
      | None -> not_handled
      | Some { vkind = Arbitrary; value; _ } -> (
          match arbitrary_type c [ "color"; "length" ] with
          | "color" -> (
              match as_color value c.cmodifier theme with
              | Some resolved -> handled (color_only "--tw-ring-offset-color" resolved)
              | None -> not_handled)
          | "length" -> if c.cmodifier <> None then not_handled else width value
          | _ -> not_handled)
      | Some { value; _ } -> (
          match resolve_theme_color c [ "--ring-offset-color"; "--color" ] theme with
          | Some color -> handled (color_only "--tw-ring-offset-color" color)
          | None -> (
              if c.cmodifier <> None then not_handled
              else
                match Theme.resolve theme (Some value) [ "--ring-offset-width" ] 0 with
                | Some w -> width w
                | None -> if is_positive_integer value then width (value ^ "px") else not_handled)))
