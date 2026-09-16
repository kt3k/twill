(* Space, divider, and outline utilities (SPEC §10.8, "Space, dividers, and outlines"). *)

open Ast
open Candidate
open Utilities

let property = Builtin_variants.property

(* Targets every child but the last. *)
let children_selector = ":where(& > :not(:last-child))"

let children nodes = style_rule children_selector ~nodes

(* Registers the space, divide, and outline utilities. *)
let register u theme =
  let stat_fn name f = static_utility_fn u name f in
  let pixels = bare_suffix "px" in
  List.iter
    (fun axis ->
      let reverse = "--tw-space-" ^ axis ^ "-reverse" in
      let start, end_ = if axis = "y" then ("margin-block-start", "margin-block-end") else ("margin-inline-start", "margin-inline-end") in
      spacing_utility u theme ("space-" ^ axis) [ "--spacing" ]
        (fun value ->
          [ property reverse (Some "0");
            children
              [ decl reverse "0";
                decl start ("calc(" ^ value ^ " * var(" ^ reverse ^ "))");
                decl end_ ("calc(" ^ value ^ " * calc(1 - var(" ^ reverse ^ ")))") ] ])
        true false;
      stat_fn ("space-" ^ axis ^ "-reverse") (fun () -> [ property reverse (Some "0"); children [ decl reverse "1" ] ]))
    [ "x"; "y" ];
  List.iter
    (fun axis ->
      let reverse = "--tw-divide-" ^ axis ^ "-reverse" in
      let style, start, end_ =
        if axis = "y" then ("border-block-style", "border-block-start-width", "border-block-end-width")
        else ("border-inline-style", "border-inline-start-width", "border-inline-end-width")
      in
      let width value =
        handled
          [ property reverse (Some "0"); property "--tw-border-style" (Some "solid");
            children
              [ decl reverse "0"; decl style "var(--tw-border-style)";
                decl start ("calc(" ^ value ^ " * var(" ^ reverse ^ "))");
                decl end_ ("calc(" ^ value ^ " * calc(1 - var(" ^ reverse ^ ")))") ] ]
      in
      functional u ("divide-" ^ axis) (fun c ->
          if c.cmodifier <> None then not_handled
          else
            match c.cvalue with
            | None -> width (match Theme.get theme [ "--default-border-width" ] with Some w -> w | None -> "1px")
            | Some { vkind = Arbitrary; value; _ } -> (
                match arbitrary_type c [ "length"; "line-width" ] with "length" | "line-width" -> width value | _ -> not_handled)
            | Some named -> (
                match Theme.resolve theme (Some named.value) [ "--divide-width" ] 0 with
                | Some w -> width w
                | None -> ( match pixels named with Some bare -> width bare | None -> not_handled)));
      stat_fn ("divide-" ^ axis ^ "-reverse") (fun () -> [ property reverse (Some "0"); children [ decl reverse "1" ] ]))
    [ "x"; "y" ];
  color_utility u theme "divide" [ "--divide-color"; "--color" ] (fun value -> [ children [ decl "border-color" value ] ]);
  List.iter
    (fun style -> stat_fn ("divide-" ^ style) (fun () -> [ children [ decl "--tw-border-style" style; decl "border-style" style ] ]))
    [ "solid"; "dashed"; "dotted"; "double"; "none" ];
  let outline_width value =
    handled [ property "--tw-outline-style" (Some "solid"); decl "outline-style" "var(--tw-outline-style)"; decl "outline-width" value ]
  in
  functional u "outline" (fun c ->
      match c.cvalue with
      | None -> if c.cmodifier <> None then not_handled else outline_width "1px"
      | Some { vkind = Arbitrary; value; _ } -> (
          match arbitrary_type c [ "color"; "length"; "line-width" ] with
          | "color" -> ( match as_color value c.cmodifier theme with Some r -> handled [ decl "outline-color" r ] | None -> not_handled)
          | "length" | "line-width" -> if c.cmodifier <> None then not_handled else outline_width value
          | _ -> not_handled)
      | Some named -> (
          match resolve_theme_color c [ "--outline-color"; "--color" ] theme with
          | Some color -> handled [ decl "outline-color" color ]
          | None -> (
              if c.cmodifier <> None then not_handled
              else
                match Theme.resolve theme (Some named.value) [ "--outline-width" ] 0 with
                | Some w -> outline_width w
                | None -> ( match pixels named with Some bare -> outline_width bare | None -> not_handled))));
  stat_fn "outline-none" (fun () -> [ decl "--tw-outline-style" "none"; decl "outline-style" "none" ]);
  stat_fn "outline-hidden" (fun () ->
      [ decl "--tw-outline-style" "none"; decl "outline-style" "none";
        at_rule "@media" "(forced-colors: active)" ~nodes:[ decl "outline" "2px solid transparent"; decl "outline-offset" "2px" ] ]);
  List.iter
    (fun style -> stat_fn ("outline-" ^ style) (fun () -> [ decl "--tw-outline-style" style; decl "outline-style" style ]))
    [ "solid"; "dashed"; "dotted"; "double" ];
  functional_utility u theme "outline-offset"
    (describe ~theme_keys:[ "--outline-offset" ] ~supports_negative:true ~handle_bare_value:pixels (fun value _ ->
         [ decl "outline-offset" value ]))
