(* The built-in variants (SPEC §9.5) and the helpers they share. *)

open Ast
open Candidate
open Utils

(* Builds `@property <name> { ... }` inside an at-root. *)
let property_registration name initial_value syntax inherits =
  let nodes = [ decl "syntax" ("\"" ^ syntax ^ "\""); decl "inherits" (if inherits then "true" else "false") ] in
  let nodes = match initial_value with Some v -> nodes @ [ decl "initial-value" v ] | None -> nodes in
  at_root [ at_rule "@property" name ~nodes ]

(* A `syntax: "*"; inherits: false` registration. *)
let property name initial_value = property_registration name initial_value "*" false

let replace_ampersand selector replacement = replace_all selector "&" replacement

(* Negates a style selector produced by a variant. *)
let negate_selector selector =
  if contains selector "::" then None
  else
    try
      let parts =
        List.map
          (fun part ->
            let part = String.trim part in
            if part = "&" then raise Exit;
            if has_prefix part "&" then begin
              let rest = after part 1 in
              if (not (contains rest "&")) && (has_prefix rest ":" || has_prefix rest "[") then "&:not(" ^ rest ^ ")"
              else "&:not(" ^ replace_ampersand part "*" ^ ")"
            end
            else "&:not(" ^ replace_ampersand part "*" ^ ")")
          (segment selector ',')
      in
      Some (String.concat ", " parts)
    with Exit -> None

(* Negates an at-rule produced by a variant. *)
let negate_at_rule name params =
  match name with
  | "@media" ->
      if has_prefix params "not " then Some (name, after params 4)
      else if has_prefix params "(" then Some (name, "not all and " ^ params)
      else Some (name, "not " ^ params)
  | "@supports" -> if has_prefix params "not " then Some (name, after params 4) else Some (name, "not " ^ params)
  | "@container" -> (
      if has_prefix params "(" then Some (name, "not " ^ params)
      else
        match String.index_opt params ' ' with
        | None -> None
        | Some space ->
            let container_name = String.sub params 0 space in
            let rest = String.trim (after params (space + 1)) in
            if has_prefix rest "not " then Some (name, container_name ^ " " ^ after rest 4)
            else Some (name, container_name ^ " not " ^ rest))
  | _ -> None

let attribute_name_re = Str.regexp "^[a-zA-Z_:][-a-zA-Z0-9_:.]*$"
let attribute_flag_re = Str.regexp "[ \t]+\\([isIS]\\)$"

(* Wraps an unquoted attribute value in double quotes, keeping a trailing
   ` i` or ` s` flag outside the quotes. *)
let quote_attribute_value value =
  match String.index_opt value '=' with
  | None -> if Str.string_match attribute_name_re value 0 then Some value else None
  | Some eq ->
      let name = String.sub value 0 eq and rest = after value (eq + 1) in
      let name, operator =
        if name <> "" then
          let last = name.[String.length name - 1] in
          if last = '~' || last = '|' || last = '^' || last = '$' || last = '*' then
            (String.sub name 0 (String.length name - 1), String.make 1 last ^ "=")
          else (name, "=")
        else (name, "=")
      in
      if not (Str.string_match attribute_name_re name 0) then None
      else
        let rest, flag =
          match Str.search_forward attribute_flag_re rest 0 with
          | i ->
              let flag = " " ^ Str.matched_group 1 rest in
              (String.sub rest 0 i, flag)
          | exception Not_found -> (rest, "")
        in
        let rest = String.trim rest in
        if rest = "" then None
        else
          let quoted = (has_prefix rest "\"" && has_suffix rest "\"") || (has_prefix rest "'" && has_suffix rest "'") in
          if (not quoted) && contains rest "\"" then None
          else
            let rest = if quoted then rest else "\"" ^ rest ^ "\"" in
            Some (name ^ operator ^ rest ^ flag)

let resolve_query_value theme (variant : variant) namespace =
  match variant.kind with
  | Static_variant -> Theme.resolve_value theme (Some variant.root) [ namespace ]
  | Functional_variant -> (
      match variant.vvalue with
      | None -> None
      | Some vv -> (
          let value =
            if vv.vvkind = Arbitrary then Some vv.vvalue else Theme.resolve_value theme (Some vv.vvalue) [ namespace ]
          in
          match value with Some v when not (contains v "var(") -> Some v | _ -> None))
  | _ -> None

let number_prefix_re = Str.regexp "^-?[0-9.]+"
let number_prefix value = if Str.string_match number_prefix_re value 0 then Str.matched_string value else ""

let bucket_of value =
  match String.index_opt value '(' with
  | Some paren when has_suffix value ")" -> String.sub value 0 paren
  | _ ->
      let prefix = number_prefix value in
      after value (String.length prefix)

(* Compares two breakpoint or container values: by unit bucket, then
   numerically (SPEC §9.5). *)
let compare_query_values a_value z_value =
  if a_value = z_value then 0
  else
    let a_bucket = bucket_of a_value and z_bucket = bucket_of z_value in
    if a_bucket <> z_bucket then compare a_bucket z_bucket
    else
      match (float_of_string_opt (number_prefix a_value), float_of_string_opt (number_prefix z_value)) with
      | Some a, Some z -> compare a z
      | _ -> compare a_value z_value

let compare_query_values_fn theme namespace ascending (a : variant) (z : variant) =
  if a == z then 0
  else
    match (resolve_query_value theme a namespace, resolve_query_value theme z namespace) with
    | None, None -> 0
    | None, Some _ -> if ascending then -1 else 1
    | Some _, None -> if ascending then 1 else -1
    | Some av, Some zv ->
        if av = zv then 0
        else
          let result = compare_query_values av zv in
          if ascending then result else -result

let with_children node f = match children node with Some ch -> f ch | None -> false
let is_rule_like = Variants.is_rule_like

(* Negates each rule produced by an inner variant. *)
let negate node =
  let result = ref None and failed = ref false in
  ignore
    (walk (ref [ node ]) (fun child u ->
         match children child with
         | Some cc when is_rule_like child -> (
             if !cc <> [] then Continue
             else
               let style_rules = ref [] and at_rules = ref [] in
               List.iter
                 (function
                   | Rule r -> style_rules := !style_rules @ [ r.selector ]
                   | At_rule a -> at_rules := !at_rules @ [ (a.name, a.params) ]
                   | _ -> ())
                 (u.path @ [ child ]);
               if List.length !style_rules > 1 || List.length !at_rules > 1 || !result <> None then begin
                 failed := true;
                 Stop
               end
               else
                 try
                   let rules =
                     List.map
                       (fun selector ->
                         match negate_selector selector with Some s -> style_rule s | None -> raise Exit)
                       !style_rules
                     @ List.map
                         (fun (name, params) ->
                           match negate_at_rule name params with Some (n, p) -> at_rule n p | None -> raise Exit)
                         !at_rules
                   in
                   result := Some rules;
                   Skip
                 with Exit ->
                   failed := true;
                   Stop)
         | _ -> Continue));
  match !result with
  | None -> false
  | Some _ when !failed -> false
  | Some rules ->
      let replacement = match rules with [ r ] -> r | _ -> style_rule "&" ~nodes:rules in
      (match (node, replacement) with
      | Rule target, Rule r ->
          target.selector <- r.selector;
          target.rule_nodes := !(r.rule_nodes)
      | Rule target, r ->
          (* A style rule cannot become an at-rule in place; wrap instead. *)
          target.selector <- "&";
          target.rule_nodes := [ r ]
      | At_rule target, At_rule r ->
          target.name <- r.name;
          target.params <- r.params;
          target.at_nodes := !(r.at_nodes)
      | At_rule target, r ->
          target.name <- "@media";
          target.params <- "all";
          target.at_nodes := [ r ]
      | _ -> ());
      true

let group_like node (variant : variant) class_name combinator (theme : Theme.t) =
  match node with
  | Rule r -> (
      match variant.vmodifier with
      | Some m when m.mkind <> Named -> false
      | _ -> (
          match variant.inner with
          | Some { kind = Arbitrary_variant; relative = true; _ } -> false
          | _ ->
              if Variants.has_nested_style_rules !(r.rule_nodes) then false
              else begin
                let marker = match variant.vmodifier with Some m -> class_name ^ "/" ^ m.mvalue | None -> class_name in
                let marker = if theme.prefix <> "" then theme.prefix ^ ":" ^ marker else marker in
                let selector = replace_ampersand r.selector (":where(." ^ escape marker ^ ")") in
                let selector = if List.length (segment selector ',') > 1 then ":is(" ^ selector ^ ")" else selector in
                r.selector <- "&:is(" ^ selector ^ " " ^ combinator ^ ")";
                true
              end))
  | _ -> false

let relative_re = Str.regexp "^[>+~][^ ]"
let supports_call_re = Str.regexp "^[a-zA-Z0-9_-]*[ \t]*("
let supports_keyword_re = Str.regexp "\\b\\(and\\|or\\|not\\)("

(* Registers every built-in variant in the order of SPEC §9.5. *)
let register (v : Variants.t) (theme : Theme.t) =
  let open Variants in
  let static name selectors = static_variant v name selectors 0 true in
  let never name selectors = static_variant v name selectors compounds_never false in
  let wrap node make = with_children node (fun ch -> ch := [ make !ch ]; true) in
  never "*" [ ":is(& > *)" ];
  never "**" [ ":is(& *)" ];
  compound v "not" (compounds_style_rules lor compounds_at_rules)
    (fun node variant -> if variant.vmodifier <> None then false else negate node)
    (compounds_style_rules lor compounds_at_rules);
  compound v "group" compounds_style_rules (fun node variant -> group_like node variant "group" "*" theme) compounds_style_rules;
  compound v "peer" compounds_style_rules (fun node variant -> group_like node variant "peer" "~ *" theme) compounds_style_rules;
  never "first-letter" [ "&::first-letter" ];
  never "first-line" [ "&::first-line" ];
  never "marker" [ "& *::marker"; "&::marker"; "& *::-webkit-details-marker"; "&::-webkit-details-marker" ];
  never "selection" [ "& *::selection"; "&::selection" ];
  never "file" [ "&::file-selector-button" ];
  never "placeholder" [ "&::placeholder" ];
  never "backdrop" [ "&::backdrop" ];
  never "details-content" [ "&::details-content" ];
  List.iter
    (fun name ->
      let pseudo = "&::" ^ name in
      Variants.static v name
        (fun node _ ->
          wrap node (fun nodes ->
              style_rule pseudo ~nodes:([ property "--tw-content" (Some "\"\""); decl "content" "var(--tw-content)" ] @ nodes)))
        compounds_never)
    [ "before"; "after" ];
  List.iter
    (fun (name, selector) -> static name [ selector ])
    [ ("first", "&:first-child"); ("last", "&:last-child"); ("only", "&:only-child"); ("odd", "&:nth-child(odd)");
      ("even", "&:nth-child(even)"); ("first-of-type", "&:first-of-type"); ("last-of-type", "&:last-of-type");
      ("only-of-type", "&:only-of-type"); ("visited", "&:visited"); ("target", "&:target");
      ("open", "&:is([open], :popover-open, :open)"); ("default", "&:default"); ("checked", "&:checked");
      ("indeterminate", "&:indeterminate"); ("placeholder-shown", "&:placeholder-shown"); ("autofill", "&:autofill");
      ("optional", "&:optional"); ("required", "&:required"); ("valid", "&:valid"); ("invalid", "&:invalid");
      ("user-valid", "&:user-valid"); ("user-invalid", "&:user-invalid"); ("in-range", "&:in-range");
      ("out-of-range", "&:out-of-range"); ("read-only", "&:read-only"); ("empty", "&:empty");
      ("focus-within", "&:focus-within") ];
  Variants.static v "hover"
    (fun node _ -> wrap node (fun nodes -> style_rule "&:hover" ~nodes:[ at_rule "@media" "(hover: hover)" ~nodes ]))
    compounds_style_rules;
  List.iter (fun name -> static name [ "&:" ^ name ]) [ "focus"; "focus-visible"; "active"; "enabled"; "disabled" ];
  static "inert" [ "&:is([inert], [inert] *)" ];
  compound v "in" compounds_style_rules
    (fun node variant ->
      match node with
      | Rule r when variant.vmodifier = None ->
          r.selector <- ":where(" ^ replace_ampersand r.selector "*" ^ ") &";
          true
      | _ -> false)
    compounds_style_rules;
  compound v "has" compounds_style_rules
    (fun node variant ->
      match node with
      | Rule r when variant.vmodifier = None ->
          let selector = replace_ampersand r.selector "*" in
          let selector =
            if Str.string_match relative_re selector 0 then String.sub selector 0 1 ^ " " ^ after selector 1 else selector
          in
          r.selector <- "&:has(" ^ selector ^ ")";
          true
      | _ -> false)
    compounds_style_rules;
  let valued (variant : variant) f =
    match (variant.vvalue, variant.vmodifier) with Some vv, None -> f vv | _ -> false
  in
  functional v "aria"
    (fun node variant ->
      valued variant (fun vv ->
          if vv.vvkind = Named then wrap node (fun nodes -> style_rule ("&[aria-" ^ vv.vvalue ^ "=\"true\"]") ~nodes)
          else
            match quote_attribute_value vv.vvalue with
            | Some attribute -> wrap node (fun nodes -> style_rule ("&[aria-" ^ attribute ^ "]") ~nodes)
            | None -> false))
    compounds_style_rules;
  functional v "data"
    (fun node variant ->
      valued variant (fun vv ->
          match quote_attribute_value vv.vvalue with
          | Some attribute -> wrap node (fun nodes -> style_rule ("&[data-" ^ attribute ^ "]") ~nodes)
          | None -> false))
    compounds_style_rules;
  List.iter
    (fun (name, pseudo) ->
      functional v name
        (fun node variant ->
          valued variant (fun vv ->
              if vv.vvkind = Named && not (is_positive_integer vv.vvalue) then false
              else wrap node (fun nodes -> style_rule ("&:" ^ pseudo ^ "(" ^ vv.vvalue ^ ")") ~nodes)))
        compounds_style_rules)
    [ ("nth", "nth-child"); ("nth-last", "nth-last-child"); ("nth-of-type", "nth-of-type"); ("nth-last-of-type", "nth-last-of-type") ];
  functional v "supports"
    (fun node variant ->
      valued variant (fun vv ->
          let value = vv.vvalue in
          let condition =
            if Str.string_match supports_call_re value 0 then Str.global_replace supports_keyword_re "\\1 (" value
            else if not (String.contains value ':') then "(" ^ value ^ ": var(--tw))"
            else if has_prefix value "(" && has_suffix value ")" then value
            else "(" ^ value ^ ")"
          in
          wrap node (fun nodes -> at_rule "@supports" condition ~nodes)))
    compounds_at_rules;
  List.iter
    (fun (name, query) -> static name [ "@media " ^ query ])
    [ ("motion-safe", "(prefers-reduced-motion: no-preference)"); ("motion-reduce", "(prefers-reduced-motion: reduce)");
      ("contrast-more", "(prefers-contrast: more)"); ("contrast-less", "(prefers-contrast: less)") ];
  let media_query namespace operator node (variant : variant) =
    valued variant (fun _ ->
        match resolve_query_value theme variant namespace with
        | Some value -> wrap node (fun nodes -> at_rule "@media" ("(width " ^ operator ^ " " ^ value ^ ")") ~nodes)
        | None -> false)
  in
  group v ~compare:(compare_query_values_fn theme "--breakpoint" false) (fun () ->
      functional v "max" (media_query "--breakpoint" "<") compounds_at_rules);
  group v ~compare:(compare_query_values_fn theme "--breakpoint" true) (fun () ->
      List.iter
        (fun (entry : Theme.namespace_entry) ->
          if entry.self || contains entry.key "--" then ()
          else
            let value = entry.value in
            Variants.static v entry.key
              (fun node _ -> wrap node (fun nodes -> at_rule "@media" ("(width >= " ^ value ^ ")") ~nodes))
              compounds_at_rules)
        (Theme.namespace theme "--breakpoint");
      functional v "min" (media_query "--breakpoint" ">=") compounds_at_rules);
  let container_query operator node (variant : variant) =
    match variant.vvalue with
    | None -> false
    | Some _ -> (
        match variant.vmodifier with
        | Some m when m.mkind <> Named -> false
        | modifier -> (
            match resolve_query_value theme variant "--container" with
            | None -> false
            | Some value ->
                let name = match modifier with Some m -> m.mvalue ^ " " | None -> "" in
                wrap node (fun nodes -> at_rule "@container" (name ^ "(width " ^ operator ^ " " ^ value ^ ")") ~nodes)))
  in
  group v ~compare:(compare_query_values_fn theme "--container" false) (fun () ->
      functional v "@max" (container_query "<") compounds_at_rules);
  group v ~compare:(compare_query_values_fn theme "--container" true) (fun () ->
      functional v "@" (container_query ">=") compounds_at_rules;
      functional v "@min" (container_query ">=") compounds_at_rules);
  static "portrait" [ "@media (orientation: portrait)" ];
  static "landscape" [ "@media (orientation: landscape)" ];
  static "ltr" [ "&:where(:dir(ltr), [dir=\"ltr\"], [dir=\"ltr\"] *)" ];
  static "rtl" [ "&:where(:dir(rtl), [dir=\"rtl\"], [dir=\"rtl\"] *)" ];
  static "dark" [ "@media (prefers-color-scheme: dark)" ];
  never "starting" [ "@starting-style" ];
  static "print" [ "@media print" ];
  static "forced-colors" [ "@media (forced-colors: active)" ];
  static "inverted-colors" [ "@media (inverted-colors: inverted)" ];
  static "pointer-none" [ "@media (pointer: none)" ];
  static "pointer-coarse" [ "@media (pointer: coarse)" ];
  static "pointer-fine" [ "@media (pointer: fine)" ];
  static "any-pointer-none" [ "@media (any-pointer: none)" ];
  static "any-pointer-coarse" [ "@media (any-pointer: coarse)" ];
  static "any-pointer-fine" [ "@media (any-pointer: fine)" ];
  static "noscript" [ "@media (scripting: none)" ]
