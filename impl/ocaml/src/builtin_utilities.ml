(* The built-in utility catalog (SPEC §10.8). *)

open Ast
open Candidate
open Utilities
open Utils

let property = Builtin_variants.property

(* Registers every built-in utility. *)
let register u theme =
  let stat name declarations = static_utility u name declarations in
  let stat_fn name f = static_utility_fn u name f in
  let fn root desc = functional_utility u theme root desc in
  let spacing name theme_keys handle negative fractions = spacing_utility u theme name theme_keys handle negative fractions in
  let color root theme_keys handle = color_utility u theme root theme_keys handle in
  let single property value = [ decl property value ] in
  let multi properties value = List.map (fun p -> decl p value) properties in
  let handle h = fun value _ -> h value in
  let decls properties value = List.map (fun p -> (p, value)) properties in

  (* Layout. *)
  List.iter
    (fun (name, value) -> stat name [ ("display", value) ])
    [ ("block", "block"); ("inline-block", "inline-block"); ("inline", "inline"); ("flex", "flex"); ("inline-flex", "inline-flex");
      ("grid", "grid"); ("inline-grid", "inline-grid"); ("hidden", "none"); ("contents", "contents"); ("flow-root", "flow-root");
      ("table", "table"); ("table-cell", "table-cell"); ("table-row", "table-row"); ("table-caption", "table-caption");
      ("table-column", "table-column"); ("table-column-group", "table-column-group"); ("table-footer-group", "table-footer-group");
      ("table-header-group", "table-header-group"); ("table-row-group", "table-row-group"); ("inline-table", "inline-table");
      ("list-item", "list-item") ];
  List.iter (fun name -> stat name [ ("position", name) ]) [ "static"; "fixed"; "absolute"; "relative"; "sticky" ];
  stat "visible" [ ("visibility", "visible") ];
  stat "invisible" [ ("visibility", "hidden") ];
  stat "collapse" [ ("visibility", "collapse") ];
  stat "isolate" [ ("isolation", "isolate") ];
  stat "isolation-auto" [ ("isolation", "auto") ];
  stat "box-border" [ ("box-sizing", "border-box") ];
  stat "box-content" [ ("box-sizing", "content-box") ];
  List.iter
    (fun v ->
      stat ("overflow-" ^ v) [ ("overflow", v) ];
      stat ("overflow-x-" ^ v) [ ("overflow-x", v) ];
      stat ("overflow-y-" ^ v) [ ("overflow-y", v) ])
    [ "auto"; "hidden"; "clip"; "visible"; "scroll" ];
  List.iter
    (fun v ->
      stat ("overscroll-" ^ v) [ ("overscroll-behavior", v) ];
      stat ("overscroll-x-" ^ v) [ ("overscroll-behavior-x", v) ];
      stat ("overscroll-y-" ^ v) [ ("overscroll-behavior-y", v) ])
    [ "auto"; "contain"; "none" ];
  stat "float-left" [ ("float", "left") ];
  stat "float-right" [ ("float", "right") ];
  stat "float-start" [ ("float", "inline-start") ];
  stat "float-end" [ ("float", "inline-end") ];
  stat "float-none" [ ("float", "none") ];
  stat "clear-left" [ ("clear", "left") ];
  stat "clear-right" [ ("clear", "right") ];
  stat "clear-start" [ ("clear", "inline-start") ];
  stat "clear-end" [ ("clear", "inline-end") ];
  stat "clear-both" [ ("clear", "both") ];
  stat "clear-none" [ ("clear", "none") ];
  stat "sr-only"
    [ ("position", "absolute"); ("width", "1px"); ("height", "1px"); ("padding", "0"); ("margin", "-1px"); ("overflow", "hidden");
      ("clip-path", "inset(50%)"); ("white-space", "nowrap"); ("border-width", "0") ];
  stat "not-sr-only"
    [ ("position", "static"); ("width", "auto"); ("height", "auto"); ("padding", "0"); ("margin", "0"); ("overflow", "visible");
      ("clip-path", "none"); ("white-space", "normal") ];
  List.iter
    (fun (name, properties) ->
      stat (name ^ "-auto") (decls properties "auto");
      stat (name ^ "-full") (decls properties "100%");
      stat ("-" ^ name ^ "-full") (decls properties "-100%");
      spacing name [ "--inset"; "--spacing" ] (multi properties) true true)
    [ ("inset", [ "inset" ]); ("inset-x", [ "inset-inline" ]); ("inset-y", [ "inset-block" ]); ("inset-s", [ "inset-inline-start" ]);
      ("inset-e", [ "inset-inline-end" ]); ("start", [ "inset-inline-start" ]); ("end", [ "inset-inline-end" ]); ("top", [ "top" ]);
      ("right", [ "right" ]); ("bottom", [ "bottom" ]); ("left", [ "left" ]) ];
  stat "z-auto" [ ("z-index", "auto") ];
  fn "z" (describe ~theme_keys:[ "--z-index" ] ~supports_negative:true ~handle_bare_value:bare_integer (handle (single "z-index")));
  stat "order-first" [ ("order", "-9999") ];
  stat "order-last" [ ("order", "9999") ];
  stat "order-none" [ ("order", "0") ];
  fn "order" (describe ~theme_keys:[ "--order" ] ~supports_negative:true ~handle_bare_value:bare_integer (handle (single "order")));

  (* Flexbox and grid. *)
  stat "flex-row" [ ("flex-direction", "row") ];
  stat "flex-row-reverse" [ ("flex-direction", "row-reverse") ];
  stat "flex-col" [ ("flex-direction", "column") ];
  stat "flex-col-reverse" [ ("flex-direction", "column-reverse") ];
  stat "flex-wrap" [ ("flex-wrap", "wrap") ];
  stat "flex-nowrap" [ ("flex-wrap", "nowrap") ];
  stat "flex-wrap-reverse" [ ("flex-wrap", "wrap-reverse") ];
  stat "flex-auto" [ ("flex", "auto") ];
  stat "flex-initial" [ ("flex", "0 auto") ];
  stat "flex-none" [ ("flex", "none") ];
  fn "flex"
    (describe ~theme_keys:[ "--flex" ] ~no_default:true ~supports_fractions:true ~handle_bare_value:bare_integer (handle (single "flex")));
  fn "grow" (describe ~theme_keys:[ "--flex-grow" ] ~default_value:"1" ~handle_bare_value:bare_integer (handle (single "flex-grow")));
  fn "shrink" (describe ~theme_keys:[ "--flex-shrink" ] ~default_value:"1" ~handle_bare_value:bare_integer (handle (single "flex-shrink")));
  stat "basis-auto" [ ("flex-basis", "auto") ];
  stat "basis-full" [ ("flex-basis", "100%") ];
  spacing "basis" [ "--flex-basis"; "--spacing"; "--container" ] (single "flex-basis") false true;
  List.iter
    (fun (name, property) ->
      stat (name ^ "-none") [ (property, "none") ];
      stat (name ^ "-subgrid") [ (property, "subgrid") ];
      fn name
        (describe ~theme_keys:[ "--" ^ property ] ~no_default:true
           ~handle_bare_value:(fun (v : candidate_value) ->
             if is_positive_integer v.value then Some ("repeat(" ^ v.value ^ ", minmax(0, 1fr))") else None)
           (handle (single property))))
    [ ("grid-cols", "grid-template-columns"); ("grid-rows", "grid-template-rows") ];
  List.iter
    (fun (prefix, axis) ->
      stat (prefix ^ "-span-full") [ ("grid-" ^ axis, "1 / -1") ];
      fn (prefix ^ "-span")
        (describe ~no_default:true
           ~handle_bare_value:(fun (v : candidate_value) ->
             if is_positive_integer v.value then Some ("span " ^ v.value ^ " / span " ^ v.value) else None)
           (handle (single ("grid-" ^ axis))));
      stat (prefix ^ "-start-auto") [ ("grid-" ^ axis ^ "-start", "auto") ];
      fn (prefix ^ "-start")
        (describe ~theme_keys:[ "--grid-" ^ axis ^ "-start" ] ~no_default:true ~supports_negative:true ~handle_bare_value:bare_integer
           (handle (single ("grid-" ^ axis ^ "-start"))));
      stat (prefix ^ "-end-auto") [ ("grid-" ^ axis ^ "-end", "auto") ];
      fn (prefix ^ "-end")
        (describe ~theme_keys:[ "--grid-" ^ axis ^ "-end" ] ~no_default:true ~supports_negative:true ~handle_bare_value:bare_integer
           (handle (single ("grid-" ^ axis ^ "-end"))));
      stat (prefix ^ "-auto") [ ("grid-" ^ axis, "auto") ];
      fn prefix
        (describe ~theme_keys:[ "--grid-" ^ axis ] ~no_default:true ~supports_negative:true ~handle_bare_value:bare_integer
           (handle (single ("grid-" ^ axis)))))
    [ ("col", "column"); ("row", "row") ];
  stat "grid-flow-row" [ ("grid-auto-flow", "row") ];
  stat "grid-flow-col" [ ("grid-auto-flow", "column") ];
  stat "grid-flow-dense" [ ("grid-auto-flow", "dense") ];
  stat "grid-flow-row-dense" [ ("grid-auto-flow", "row dense") ];
  stat "grid-flow-col-dense" [ ("grid-auto-flow", "column dense") ];
  List.iter
    (fun (name, property) ->
      stat (name ^ "-auto") [ (property, "auto") ];
      stat (name ^ "-min") [ (property, "min-content") ];
      stat (name ^ "-max") [ (property, "max-content") ];
      stat (name ^ "-fr") [ (property, "minmax(0, 1fr)") ];
      fn name (describe ~theme_keys:[ "--" ^ property ] ~no_default:true (handle (single property))))
    [ ("auto-cols", "grid-auto-columns"); ("auto-rows", "grid-auto-rows") ];
  spacing "gap" [ "--gap"; "--spacing" ] (single "gap") false false;
  spacing "gap-x" [ "--gap"; "--spacing" ] (single "column-gap") false false;
  spacing "gap-y" [ "--gap"; "--spacing" ] (single "row-gap") false false;
  let alignments =
    [ ("normal", "normal"); ("center", "center"); ("start", "flex-start"); ("end", "flex-end"); ("between", "space-between");
      ("around", "space-around"); ("evenly", "space-evenly"); ("stretch", "stretch"); ("baseline", "baseline") ]
  in
  List.iter (fun (name, value) -> stat ("justify-" ^ name) [ ("justify-content", value) ]) alignments;
  List.iter
    (fun (name, value) ->
      if name = "between" || name = "around" || name = "evenly" then ()
      else begin
        stat ("justify-items-" ^ name) [ ("justify-items", name) ];
        if name = "start" || name = "end" || name = "center" || name = "stretch" then
          stat ("justify-self-" ^ name) [ ("justify-self", name) ];
        stat ("items-" ^ name) [ ("align-items", value) ];
        if name <> "normal" then stat ("self-" ^ name) [ ("align-self", value) ]
      end)
    alignments;
  stat "justify-self-auto" [ ("justify-self", "auto") ];
  stat "self-auto" [ ("align-self", "auto") ];
  List.iter (fun (name, value) -> stat ("content-" ^ name) [ ("align-content", value) ]) alignments;
  List.iter
    (fun (name, value) ->
      let placed = if name = "start" || name = "end" then name else value in
      stat ("place-content-" ^ name) [ ("place-content", placed) ])
    alignments;
  List.iter (fun name -> stat ("place-items-" ^ name) [ ("place-items", name) ]) [ "start"; "end"; "center"; "baseline"; "stretch" ];
  List.iter (fun name -> stat ("place-self-" ^ name) [ ("place-self", name) ]) [ "auto"; "start"; "end"; "center"; "stretch" ];

  (* Spacing. *)
  List.iter
    (fun (name, property) -> spacing name [ "--padding"; "--spacing" ] (single property) false false)
    [ ("p", "padding"); ("px", "padding-inline"); ("py", "padding-block"); ("ps", "padding-inline-start"); ("pe", "padding-inline-end");
      ("pt", "padding-top"); ("pr", "padding-right"); ("pb", "padding-bottom"); ("pl", "padding-left") ];
  List.iter
    (fun (name, property) ->
      stat (name ^ "-auto") [ (property, "auto") ];
      spacing name [ "--margin"; "--spacing" ] (single property) true false)
    [ ("m", "margin"); ("mx", "margin-inline"); ("my", "margin-block"); ("ms", "margin-inline-start"); ("me", "margin-inline-end");
      ("mt", "margin-top"); ("mr", "margin-right"); ("mb", "margin-bottom"); ("ml", "margin-left") ];

  (* Sizing. *)
  let width_statics =
    [ ("auto", "auto"); ("full", "100%"); ("screen", "100vw"); ("svw", "100svw"); ("lvw", "100lvw"); ("dvw", "100dvw");
      ("min", "min-content"); ("max", "max-content"); ("fit", "fit-content") ]
  in
  let height_statics =
    [ ("auto", "auto"); ("full", "100%"); ("screen", "100vh"); ("svh", "100svh"); ("lvh", "100lvh"); ("dvh", "100dvh");
      ("min", "min-content"); ("max", "max-content"); ("fit", "fit-content") ]
  in
  List.iter
    (fun (name, property, key) ->
      List.iter (fun (s, value) -> if name <> "w" && s = "auto" then () else stat (name ^ "-" ^ s) [ (property, value) ]) width_statics;
      if name = "max-w" then stat "max-w-none" [ ("max-width", "none") ];
      spacing name [ key; "--spacing"; "--container" ] (single property) false true)
    [ ("w", "width", "--width"); ("min-w", "min-width", "--min-width"); ("max-w", "max-width", "--max-width") ];
  List.iter
    (fun (name, property, key) ->
      List.iter (fun (s, value) -> if name <> "h" && s = "auto" then () else stat (name ^ "-" ^ s) [ (property, value) ]) height_statics;
      if name = "max-h" then stat "max-h-none" [ ("max-height", "none") ];
      spacing name [ key; "--spacing" ] (single property) false true)
    [ ("h", "height", "--height"); ("min-h", "min-height", "--min-height"); ("max-h", "max-height", "--max-height") ];
  let size_handle value = [ decl "--tw-sort" "size"; decl "width" value; decl "height" value ] in
  List.iter
    (fun (s, value) ->
      if s = "screen" || s = "svw" || s = "lvw" || s = "dvw" then () else stat_fn ("size-" ^ s) (fun () -> size_handle value))
    width_statics;
  spacing "size" [ "--size"; "--spacing"; "--container" ] size_handle false true;

  (* Typography. *)
  let font_weight value = [ property "--tw-font-weight" None; decl "--tw-font-weight" value; decl "font-weight" value ] in
  functional u "font" (fun c ->
      match c.cvalue with
      | None -> not_handled
      | Some _ when c.cmodifier <> None -> not_handled
      | Some { vkind = Arbitrary; value; _ } -> (
          match arbitrary_type c [ "number"; "family-name"; "generic-name" ] with
          | "number" -> handled (font_weight value)
          | "family-name" | "generic-name" -> handled [ decl "font-family" value ]
          | _ -> not_handled)
      | Some named -> (
          match Theme.resolve_with theme (Some named.value) [ "--font" ] [ "--font-feature-settings"; "--font-variation-settings" ] with
          | Some (value, extra) ->
              let nodes = [ decl "font-family" value ] in
              let nodes =
                match List.assoc_opt "--font-feature-settings" extra with Some v -> nodes @ [ decl "font-feature-settings" v ] | None -> nodes
              in
              let nodes =
                match List.assoc_opt "--font-variation-settings" extra with
                | Some v -> nodes @ [ decl "font-variation-settings" v ]
                | None -> nodes
              in
              handled nodes
          | None -> (
              match Theme.resolve theme (Some named.value) [ "--font-weight" ] 0 with
              | Some weight -> handled (font_weight weight)
              | None -> not_handled)));
  stat "text-left" [ ("text-align", "left") ];
  stat "text-center" [ ("text-align", "center") ];
  stat "text-right" [ ("text-align", "right") ];
  stat "text-justify" [ ("text-align", "justify") ];
  stat "text-start" [ ("text-align", "start") ];
  stat "text-end" [ ("text-align", "end") ];
  stat "text-ellipsis" [ ("text-overflow", "ellipsis") ];
  stat "text-clip" [ ("text-overflow", "clip") ];
  stat "text-wrap" [ ("text-wrap", "wrap") ];
  stat "text-nowrap" [ ("text-wrap", "nowrap") ];
  stat "text-balance" [ ("text-wrap", "balance") ];
  stat "text-pretty" [ ("text-wrap", "pretty") ];
  (* [`Absent], [`Present v], or [`Failed] when a modifier exists but cannot resolve. *)
  let resolve_leading (c : candidate) =
    match c.cmodifier with
    | None -> `Absent
    | Some { mkind = Arbitrary; mvalue } -> `Present mvalue
    | Some { mvalue; _ } -> (
        match Theme.resolve theme (Some mvalue) [ "--leading" ] 0 with
        | Some leading -> `Present leading
        | None ->
            if is_multiple_of_quarter mvalue && Theme.resolve theme None [ "--spacing" ] 0 <> None then
              `Present ("--spacing(" ^ mvalue ^ ")")
            else if mvalue = "none" then `Present "1"
            else `Failed)
  in
  functional u "text" (fun c ->
      match c.cvalue with
      | None -> not_handled
      | Some { vkind = Arbitrary; value; _ } -> (
          match arbitrary_type c [ "color"; "length"; "percentage"; "absolute-size"; "relative-size" ] with
          | "color" | "" -> ( match as_color value c.cmodifier theme with Some r -> handled [ decl "color" r ] | None -> not_handled)
          | _ -> (
              match resolve_leading c with
              | `Failed -> not_handled
              | `Absent -> handled [ decl "font-size" value ]
              | `Present leading -> handled [ decl "font-size" value; decl "line-height" leading ]))
      | Some named -> (
          match resolve_theme_color c [ "--text-color"; "--color" ] theme with
          | Some color -> handled [ decl "color" color ]
          | None -> (
              match Theme.resolve_with theme (Some named.value) [ "--text" ] [ "--line-height"; "--letter-spacing"; "--font-weight" ] with
              | None -> not_handled
              | Some (value, extra) -> (
                  match resolve_leading c with
                  | `Failed -> not_handled
                  | leading ->
                      let nodes = [ decl "font-size" value ] in
                      let nodes =
                        match (leading, List.assoc_opt "--line-height" extra) with
                        | `Present l, _ -> nodes @ [ decl "line-height" l ]
                        | `Absent, Some lh -> nodes @ [ decl "line-height" ("var(--tw-leading, " ^ lh ^ ")") ]
                        | _ -> nodes
                      in
                      let nodes =
                        match List.assoc_opt "--letter-spacing" extra with
                        | Some ls -> nodes @ [ decl "letter-spacing" ("var(--tw-tracking, " ^ ls ^ ")") ]
                        | None -> nodes
                      in
                      let nodes =
                        match List.assoc_opt "--font-weight" extra with
                        | Some fw -> nodes @ [ decl "font-weight" ("var(--tw-font-weight, " ^ fw ^ ")") ]
                        | None -> nodes
                      in
                      handled nodes))));
  let leading_handle value = [ property "--tw-leading" None; decl "--tw-leading" value; decl "line-height" value ] in
  stat_fn "leading-none" (fun () -> leading_handle "1");
  spacing "leading" [ "--leading"; "--spacing" ] leading_handle false false;
  fn "tracking"
    (describe ~theme_keys:[ "--tracking" ] ~supports_negative:true (fun value _ ->
         [ property "--tw-tracking" None; decl "--tw-tracking" value; decl "letter-spacing" value ]));
  stat "uppercase" [ ("text-transform", "uppercase") ];
  stat "lowercase" [ ("text-transform", "lowercase") ];
  stat "capitalize" [ ("text-transform", "capitalize") ];
  stat "normal-case" [ ("text-transform", "none") ];
  stat "italic" [ ("font-style", "italic") ];
  stat "not-italic" [ ("font-style", "normal") ];
  stat "underline" [ ("text-decoration-line", "underline") ];
  stat "overline" [ ("text-decoration-line", "overline") ];
  stat "line-through" [ ("text-decoration-line", "line-through") ];
  stat "no-underline" [ ("text-decoration-line", "none") ];
  stat "truncate" [ ("overflow", "hidden"); ("text-overflow", "ellipsis"); ("white-space", "nowrap") ];
  List.iter (fun v -> stat ("whitespace-" ^ v) [ ("white-space", v) ]) [ "normal"; "nowrap"; "pre"; "pre-line"; "pre-wrap"; "break-spaces" ];
  stat "break-normal" [ ("overflow-wrap", "normal"); ("word-break", "normal") ];
  stat "break-words" [ ("overflow-wrap", "break-word") ];
  stat "break-all" [ ("word-break", "break-all") ];
  stat "break-keep" [ ("word-break", "keep-all") ];
  stat "list-none" [ ("list-style-type", "none") ];
  stat "list-disc" [ ("list-style-type", "disc") ];
  stat "list-decimal" [ ("list-style-type", "decimal") ];
  stat "list-inside" [ ("list-style-position", "inside") ];
  stat "list-outside" [ ("list-style-position", "outside") ];
  stat "antialiased" [ ("-webkit-font-smoothing", "antialiased"); ("-moz-osx-font-smoothing", "grayscale") ];
  stat "subpixel-antialiased" [ ("-webkit-font-smoothing", "auto"); ("-moz-osx-font-smoothing", "auto") ];
  stat "underline-offset-auto" [ ("text-underline-offset", "auto") ];
  fn "underline-offset"
    (describe ~theme_keys:[ "--text-underline-offset" ] ~supports_negative:true ~handle_bare_value:(bare_suffix "px")
       ~handle_negative_bare_value:(fun (v : candidate_value) -> if is_positive_integer v.value then Some ("-" ^ v.value ^ "px") else None)
       (handle (single "text-underline-offset")));
  spacing "indent" [ "--text-indent"; "--spacing" ] (single "text-indent") true false;

  (* Backgrounds and borders. *)
  functional u "bg" (fun c ->
      match c.cvalue with
      | None -> not_handled
      | Some { vkind = Arbitrary; value; _ } -> (
          let no_modifier property = if c.cmodifier <> None then not_handled else handled [ decl property value ] in
          match arbitrary_type c [ "image"; "color"; "percentage"; "position"; "bg-size"; "length"; "url" ] with
          | "percentage" | "position" -> no_modifier "background-position"
          | "bg-size" | "length" -> no_modifier "background-size"
          | "image" | "url" -> no_modifier "background-image"
          | _ -> ( match as_color value c.cmodifier theme with Some r -> handled [ decl "background-color" r ] | None -> not_handled))
      | Some named -> (
          match resolve_theme_color c [ "--background-color"; "--color" ] theme with
          | Some color -> handled [ decl "background-color" color ]
          | None -> (
              match Theme.resolve theme (Some named.value) [ "--background-image" ] 0 with
              | Some image -> if c.cmodifier <> None then not_handled else handled [ decl "background-image" image ]
              | None -> not_handled)));
  stat "bg-auto" [ ("background-size", "auto") ];
  stat "bg-cover" [ ("background-size", "cover") ];
  stat "bg-contain" [ ("background-size", "contain") ];
  stat "bg-fixed" [ ("background-attachment", "fixed") ];
  stat "bg-local" [ ("background-attachment", "local") ];
  stat "bg-scroll" [ ("background-attachment", "scroll") ];
  List.iter (fun p -> stat ("bg-" ^ p) [ ("background-position", p) ]) [ "top"; "center"; "bottom"; "left"; "right" ];
  stat "bg-top-left" [ ("background-position", "left top") ];
  stat "bg-top-right" [ ("background-position", "right top") ];
  stat "bg-bottom-left" [ ("background-position", "left bottom") ];
  stat "bg-bottom-right" [ ("background-position", "right bottom") ];
  stat "bg-repeat" [ ("background-repeat", "repeat") ];
  stat "bg-no-repeat" [ ("background-repeat", "no-repeat") ];
  stat "bg-repeat-x" [ ("background-repeat", "repeat-x") ];
  stat "bg-repeat-y" [ ("background-repeat", "repeat-y") ];
  stat "bg-repeat-round" [ ("background-repeat", "round") ];
  stat "bg-repeat-space" [ ("background-repeat", "space") ];
  stat "bg-none" [ ("background-image", "none") ];
  List.iter
    (fun v ->
      stat ("bg-clip-" ^ v) [ ("background-clip", v ^ "-box") ];
      stat ("bg-origin-" ^ v) [ ("background-origin", v ^ "-box") ])
    [ "border"; "padding"; "content" ];
  stat "bg-clip-text" [ ("background-clip", "text") ];
  List.iter
    (fun (root, width_property, color_property) ->
      let width value =
        [ property "--tw-border-style" (Some "solid"); decl "border-style" "var(--tw-border-style)"; decl width_property value ]
      in
      functional u root (fun c ->
          match c.cvalue with
          | None ->
              if c.cmodifier <> None then not_handled
              else handled (width (match Theme.get theme [ "--default-border-width" ] with Some w -> w | None -> "1px"))
          | Some { vkind = Arbitrary; value; _ } -> (
              match arbitrary_type c [ "color"; "line-width"; "length" ] with
              | "color" -> (
                  match as_color value c.cmodifier theme with Some r -> handled [ decl color_property r ] | None -> not_handled)
              | "line-width" | "length" -> if c.cmodifier <> None then not_handled else handled (width value)
              | _ -> not_handled)
          | Some named -> (
              match resolve_theme_color c [ "--border-color"; "--color" ] theme with
              | Some color -> handled [ decl color_property color ]
              | None -> (
                  if c.cmodifier <> None then not_handled
                  else
                    match Theme.resolve theme (Some named.value) [ "--border-width" ] 0 with
                    | Some w -> handled (width w)
                    | None -> if is_positive_integer named.value then handled (width (named.value ^ "px")) else not_handled))))
    [ ("border", "border-width", "border-color"); ("border-x", "border-inline-width", "border-inline-color");
      ("border-y", "border-block-width", "border-block-color"); ("border-s", "border-inline-start-width", "border-inline-start-color");
      ("border-e", "border-inline-end-width", "border-inline-end-color"); ("border-t", "border-top-width", "border-top-color");
      ("border-r", "border-right-width", "border-right-color"); ("border-b", "border-bottom-width", "border-bottom-color");
      ("border-l", "border-left-width", "border-left-color") ];
  List.iter
    (fun style -> stat ("border-" ^ style) [ ("--tw-border-style", style); ("border-style", style) ])
    [ "solid"; "dashed"; "dotted"; "double"; "hidden"; "none" ];
  List.iter
    (fun (root, properties) ->
      stat (root ^ "-none") (decls properties "0");
      stat (root ^ "-full") (decls properties "calc(infinity * 1px)");
      fn root (describe ~theme_keys:[ "--radius" ] (handle (multi properties))))
    [ ("rounded", [ "border-radius" ]); ("rounded-s", [ "border-start-start-radius"; "border-end-start-radius" ]);
      ("rounded-e", [ "border-start-end-radius"; "border-end-end-radius" ]);
      ("rounded-t", [ "border-top-left-radius"; "border-top-right-radius" ]);
      ("rounded-r", [ "border-top-right-radius"; "border-bottom-right-radius" ]);
      ("rounded-b", [ "border-bottom-right-radius"; "border-bottom-left-radius" ]);
      ("rounded-l", [ "border-top-left-radius"; "border-bottom-left-radius" ]); ("rounded-ss", [ "border-start-start-radius" ]);
      ("rounded-se", [ "border-start-end-radius" ]); ("rounded-ee", [ "border-end-end-radius" ]);
      ("rounded-es", [ "border-end-start-radius" ]); ("rounded-tl", [ "border-top-left-radius" ]);
      ("rounded-tr", [ "border-top-right-radius" ]); ("rounded-br", [ "border-bottom-right-radius" ]);
      ("rounded-bl", [ "border-bottom-left-radius" ]) ];

  (* Shadows and rings, transforms, filters, gradients, space and dividers,
     extensions, and masks. *)
  Shadows.register u theme;
  Transforms.register u theme;
  Filters.register u theme;
  Gradients.register u theme;
  Dividers.register u theme;
  Extras.register u theme;
  Masks.register u theme;

  (* Effects, transitions, interactivity. *)
  fn "opacity"
    (describe ~theme_keys:[ "--opacity" ]
       ~handle_bare_value:(fun (v : candidate_value) -> if is_multiple_of_quarter v.value then Some (v.value ^ "%") else None)
       (handle (single "opacity")));
  let with_tail property =
    [ decl "transition-property" property; decl "transition-timing-function" "var(--default-transition-timing-function)";
      decl "transition-duration" "var(--default-transition-duration)" ]
  in
  let transition_properties =
    String.concat ", "
      [ "color"; "background-color"; "border-color"; "outline-color"; "text-decoration-color"; "fill"; "stroke"; "--tw-gradient-from";
        "--tw-gradient-via"; "--tw-gradient-to"; "opacity"; "box-shadow"; "transform"; "translate"; "scale"; "rotate"; "filter";
        "-webkit-backdrop-filter"; "backdrop-filter"; "display"; "content-visibility"; "overlay"; "pointer-events" ]
  in
  stat_fn "transition" (fun () -> with_tail transition_properties);
  stat "transition-none" [ ("transition-property", "none") ];
  stat_fn "transition-all" (fun () -> with_tail "all");
  stat_fn "transition-colors" (fun () ->
      with_tail
        "color, background-color, border-color, outline-color, text-decoration-color, fill, stroke, --tw-gradient-from, --tw-gradient-via, --tw-gradient-to");
  stat_fn "transition-opacity" (fun () -> with_tail "opacity");
  stat_fn "transition-shadow" (fun () -> with_tail "box-shadow");
  stat_fn "transition-transform" (fun () -> with_tail "transform, translate, scale, rotate");
  stat "transition-discrete" [ ("transition-behavior", "allow-discrete") ];
  stat "transition-normal" [ ("transition-behavior", "normal") ];
  fn "duration" (describe ~theme_keys:[ "--transition-duration" ] ~handle_bare_value:(bare_suffix "ms") (handle (single "transition-duration")));
  fn "delay" (describe ~theme_keys:[ "--transition-delay" ] ~handle_bare_value:(bare_suffix "ms") (handle (single "transition-delay")));
  stat "ease-linear" [ ("transition-timing-function", "linear") ];
  stat "ease-initial" [ ("transition-timing-function", "initial") ];
  fn "ease" (describe ~theme_keys:[ "--ease" ] (handle (single "transition-timing-function")));
  stat "animate-none" [ ("animation", "none") ];
  fn "animate" (describe ~theme_keys:[ "--animate" ] (handle (single "animation")));
  List.iter
    (fun v -> stat ("cursor-" ^ v) [ ("cursor", v) ])
    (fields
       "auto default pointer wait text move help not-allowed none context-menu progress cell crosshair vertical-text alias copy \
        no-drop grab grabbing all-scroll col-resize row-resize n-resize e-resize s-resize w-resize ne-resize nw-resize se-resize \
        sw-resize ew-resize ns-resize nesw-resize nwse-resize zoom-in zoom-out");
  fn "cursor" (describe ~theme_keys:[ "--cursor" ] (handle (single "cursor")));
  List.iter (fun v -> stat ("select-" ^ v) [ ("-webkit-user-select", v); ("user-select", v) ]) [ "none"; "text"; "all"; "auto" ];
  stat "pointer-events-none" [ ("pointer-events", "none") ];
  stat "pointer-events-auto" [ ("pointer-events", "auto") ];
  stat "resize" [ ("resize", "both") ];
  stat "resize-none" [ ("resize", "none") ];
  stat "resize-x" [ ("resize", "horizontal") ];
  stat "resize-y" [ ("resize", "vertical") ];
  stat "appearance-none" [ ("appearance", "none") ];
  stat "appearance-auto" [ ("appearance", "auto") ];
  stat "scroll-auto" [ ("scroll-behavior", "auto") ];
  stat "scroll-smooth" [ ("scroll-behavior", "smooth") ];
  stat "will-change-auto" [ ("will-change", "auto") ];
  stat "will-change-scroll" [ ("will-change", "scroll-position") ];
  stat "will-change-contents" [ ("will-change", "contents") ];
  stat "will-change-transform" [ ("will-change", "transform") ];
  fn "will-change" (describe ~theme_keys:[ "--will-change" ] (handle (single "will-change")));
  functional u "content" (fun c ->
      match c.cvalue with
      | Some { vkind = Arbitrary; value; _ } when c.cmodifier = None ->
          handled [ property "--tw-content" (Some "\"\""); decl "--tw-content" value; decl "content" "var(--tw-content)" ]
      | _ -> not_handled);
  stat "aspect-square" [ ("aspect-ratio", "1 / 1") ];
  stat "aspect-auto" [ ("aspect-ratio", "auto") ];
  fn "aspect"
    (describe ~theme_keys:[ "--aspect" ]
       ~handle_bare_value:(fun (v : candidate_value) ->
         match v.fraction with
         | None -> None
         | Some f -> (
             match String.index_opt f '/' with
             | None -> None
             | Some i ->
                 let a = String.sub f 0 i and b = after f (i + 1) in
                 if is_positive_integer a && is_positive_integer b then Some (a ^ " / " ^ b) else None))
       (handle (single "aspect-ratio")));
  stat "columns-auto" [ ("columns", "auto") ];
  fn "columns" (describe ~theme_keys:[ "--columns"; "--container" ] ~handle_bare_value:bare_integer (handle (single "columns")));
  List.iter (fun v -> stat ("object-" ^ v) [ ("object-fit", v) ]) [ "contain"; "cover"; "fill"; "none"; "scale-down" ];
  List.iter (fun p -> stat ("object-" ^ p) [ ("object-position", p) ]) [ "top"; "center"; "bottom"; "left"; "right" ];
  stat "accent-auto" [ ("accent-color", "auto") ];
  color "accent" [ "--accent-color"; "--color" ] (single "accent-color");
  color "caret" [ "--caret-color"; "--color" ] (single "caret-color");
  stat "fill-none" [ ("fill", "none") ];
  color "fill" [ "--fill"; "--color" ] (single "fill");
  stat "stroke-none" [ ("stroke", "none") ]
