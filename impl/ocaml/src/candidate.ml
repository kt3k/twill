(* Candidate and variant parsing (SPEC §4.1.3 through §4.1.6, §8). *)

open Utils

type value_kind = Named | Arbitrary

(* The value segment of a functional candidate. [fraction] is set for named
   values when a slash segment could be a fraction; [data_type] is an
   explicit type hint of an arbitrary value. *)
type candidate_value = { vkind : value_kind; value : string; fraction : string option; data_type : string option }

type modifier = { mkind : value_kind; mvalue : string }
type variant_kind = Static_variant | Functional_variant | Compound_variant | Arbitrary_variant
type variant_value = { vvkind : value_kind; vvalue : string }

type variant = {
  kind : variant_kind;
  root : string;
  vvalue : variant_value option;
  vmodifier : modifier option;
  inner : variant option;
  (* [selector] and [relative] belong to arbitrary variants. *)
  selector : string;
  relative : bool;
  (* The text this variant was parsed from; memoized parsers key on it. *)
  vraw : string;
}

type candidate_kind = Static_candidate | Functional_candidate | Arbitrary_candidate

type candidate = {
  ckind : candidate_kind;
  raw : string;
  (* In application order: the rightmost in the source is first. *)
  variants : variant list;
  important : bool;
  croot : string;
  cvalue : candidate_value option;
  cmodifier : modifier option;
  property : string;
  arbitrary_value : string;
}

type utility_kind = At_utility.kind = Static | Functional

(* The part of the design system the parsers consult. *)
type context = {
  theme_prefix : unit -> string;
  has_utility : string -> utility_kind -> bool;
  has_variant : string -> bool;
  variant_kind_of : string -> variant_kind option;
  compounds_with : string -> variant -> bool;
  (* Memoized: returns the same object for the same input. *)
  parse_variant : string -> variant option;
}

let make_variant ?(vvalue = None) ?(vmodifier = None) ?(inner = None) ?(selector = "") ?(relative = false) ?(vraw = "") kind root =
  { kind; root; vvalue; vmodifier; inner; selector; relative; vraw }

let named_value ?(fraction = None) value = { vkind = Named; value; fraction; data_type = None }
let arbitrary_value ?(data_type = None) value = { vkind = Arbitrary; value; fraction = None; data_type }

let make_candidate ?(variants = []) ?(important = false) ?(croot = "") ?(cvalue = None) ?(cmodifier = None) ?(property = "")
    ?(arbitrary_value = "") ckind raw =
  { ckind; raw; variants; important; croot; cvalue; cmodifier; property; arbitrary_value }

type root_match = { rroot : string; rvalue : string option }

(* Every (root, value) split whose root exists (SPEC §8.2.2). *)
let find_roots input exists =
  let result = ref [] in
  if exists input then result := [ { rroot = input; rvalue = None } ];
  (match String.rindex_opt input '-' with
  | None -> ()
  | Some idx ->
      let idx = ref idx in
      let continue = ref true in
      while !continue do
        let root = String.sub input 0 !idx in
        if exists root then begin
          let value = after input (!idx + 1) in
          if value = "" || root = "@" then continue := false
          else result := !result @ [ { rroot = root; rvalue = Some value } ]
        end;
        if !continue then
          if !idx = 0 then continue := false
          else
            match String.rindex_opt (String.sub input 0 !idx) '-' with
            | Some j when j > 0 -> idx := j
            | _ -> continue := false
      done);
  if has_prefix input "@" && exists "@" then result := !result @ [ { rroot = "@"; rvalue = Some (after input 1) } ];
  !result

(* Parses a modifier segment (SPEC §8.2.1). *)
let parse_modifier modifier =
  let n = String.length modifier in
  if has_prefix modifier "[" && has_suffix modifier "]" then begin
    let value = decode_arbitrary_value (String.sub modifier 1 (n - 2)) in
    if (not (is_valid_arbitrary value)) || String.trim value = "" then None else Some { mkind = Arbitrary; mvalue = value }
  end
  else if has_prefix modifier "(" && has_suffix modifier ")" then begin
    let value = decode_arbitrary_value (String.sub modifier 1 (n - 2)) in
    if (not (has_prefix value "--")) || not (is_valid_arbitrary value) then None
    else Some { mkind = Arbitrary; mvalue = "var(" ^ value ^ ")" }
  end
  else if is_named_value modifier then Some { mkind = Named; mvalue = modifier }
  else None

exception Done

(* Returns every interpretation of a raw class name (SPEC §8.2). *)
let parse_candidate input (ds : context) : candidate list =
  let results = ref [] in
  let raw_variants = segment input ':' in
  let prefix = ds.theme_prefix () in
  let raw_variants =
    if prefix <> "" then
      match raw_variants with [ _ ] -> raise Done | p :: rest when p = prefix -> rest | _ -> raise Done
    else raw_variants
  in
  let base = List.nth raw_variants (List.length raw_variants - 1) in
  let raw_variants = List.filteri (fun i _ -> i < List.length raw_variants - 1) raw_variants in
  let variants =
    List.map
      (fun v -> match ds.parse_variant v with Some p -> p | None -> raise Done)
      (List.rev raw_variants)
  in
  let important, base =
    if has_suffix base "!" then (true, String.sub base 0 (String.length base - 1))
    else if has_prefix base "!" then (true, after base 1)
    else (false, base)
  in
  if ds.has_utility base Static && not (String.contains base '[') then
    results := [ make_candidate ~variants ~important ~croot:base Static_candidate input ];
  let finish () = !results in
  let parts = segment base '/' in
  if List.length parts > 2 then finish ()
  else begin
    let base_without_modifier = List.hd parts in
    let modifier_segment = match parts with [ _; m ] -> Some m | _ -> None in
    let modifier = match modifier_segment with Some m -> parse_modifier m | None -> None in
    if modifier_segment <> None && modifier = None then finish ()
    else if has_prefix base_without_modifier "[" then begin
      let b = base_without_modifier in
      let n = String.length b in
      if (not (has_suffix b "]")) || n < 2 then finish ()
      else
        let second = b.[1] in
        if not (second = '-' || is_lower second) then finish ()
        else
          let inner = String.sub b 1 (n - 2) in
          match String.index_opt inner ':' with
          | None -> finish ()
          | Some colon when colon = 0 || colon = String.length inner - 1 -> finish ()
          | Some colon ->
              let value = decode_arbitrary_value (after inner (colon + 1)) in
              if not (is_valid_arbitrary value) then finish ()
              else
                !results
                @ [ make_candidate ~variants ~important ~cmodifier:modifier ~property:(String.sub inner 0 colon)
                      ~arbitrary_value:value Arbitrary_candidate input ]
    end
    else begin
      let b = base_without_modifier in
      let roots =
        if has_suffix b "]" then begin
          match index_of b "-[" with
          | -1 -> raise Done
          | idx ->
              let root = String.sub b 0 idx in
              if not (ds.has_utility root Functional) then raise Done;
              [ { rroot = root; rvalue = Some (after b (idx + 1)) } ]
        end
        else if has_suffix b ")" then begin
          match index_of b "-(" with
          | -1 -> raise Done
          | idx ->
              let root = String.sub b 0 idx in
              if not (ds.has_utility root Functional) then raise Done;
              let inner = String.sub b (idx + 2) (String.length b - idx - 3) in
              let data_type, value =
                match segment inner ':' with
                | [ v ] -> ("", v)
                | [ t; v ] -> (t, v)
                | _ -> raise Done
              in
              if (not (has_prefix value "--")) || not (is_valid_arbitrary value) then raise Done;
              let rewritten = if data_type = "" then "[var(" ^ value ^ ")]" else "[" ^ data_type ^ ":var(" ^ value ^ ")]" in
              [ { rroot = root; rvalue = Some rewritten } ]
        end
        else find_roots b (fun r -> ds.has_utility r Functional)
      in
      let rec loop = function
        | [] -> ()
        | m :: rest -> (
            let base_candidate = make_candidate ~variants ~important ~croot:m.rroot ~cmodifier:modifier Functional_candidate input in
            match m.rvalue with
            | None ->
                results := !results @ [ base_candidate ];
                loop rest
            | Some value ->
                let skip = ref false in
                let cvalue =
                  match String.index_opt value '[' with
                  | Some bracket ->
                      if not (has_suffix value "]") then raise Done;
                      let decoded = decode_arbitrary_value (String.sub value (bracket + 1) (String.length value - bracket - 2)) in
                      if not (is_valid_arbitrary decoded) then (skip := true; None)
                      else begin
                        let i = ref 0 in
                        let dn = String.length decoded in
                        while !i < dn && (decoded.[!i] = '-' || is_lower decoded.[!i]) do incr i done;
                        let data_type, arbitrary =
                          if !i < dn && decoded.[!i] = ':' then
                            let hint = String.sub decoded 0 !i in
                            if hint = "" then (skip := true; (None, decoded)) else (Some hint, after decoded (!i + 1))
                          else (None, decoded)
                        in
                        if String.trim arbitrary = "" then skip := true;
                        Some (arbitrary_value ~data_type arbitrary)
                      end
                  | None ->
                      let fraction =
                        match (modifier_segment, modifier) with
                        | Some seg, Some { mkind = Named; _ } -> Some (value ^ "/" ^ seg)
                        | _ -> None
                      in
                      if not (is_named_value value) then (skip := true; None) else Some (named_value ~fraction value)
                in
                if not !skip then results := !results @ [ { base_candidate with cvalue } ];
                loop rest)
      in
      loop roots;
      finish ()
    end
  end

let parse_candidate input ds = try parse_candidate input ds with Done -> []

(* Parses a variant segment (SPEC §8.3). *)
let parse_variant input (ds : context) : variant option =
  let n = String.length input in
  if has_prefix input "[" && has_suffix input "]" then begin
    if n > 1 && input.[1] = '@' && String.contains input '&' then None
    else
      let selector = decode_arbitrary_value (String.sub input 1 (n - 2)) in
      if (not (is_valid_arbitrary selector)) || String.trim selector = "" then None
      else
        let first = selector.[0] in
        let relative = first = '>' || first = '+' || first = '~' in
        let result = if (not relative) && first <> '@' && not (String.contains selector '&') then "&:is(" ^ selector ^ ")" else selector in
        Some (make_variant ~selector:result ~relative ~vraw:input Arbitrary_variant "")
  end
  else
    let parts = segment input '/' in
    if List.length parts > 2 then None
    else
      let name = List.hd parts in
      let modifier_text = match parts with [ _; m ] -> Some m | _ -> None in
      let rec try_roots = function
        | [] -> None
        | m :: rest -> (
            match ds.variant_kind_of m.rroot with
            | None -> try_roots rest
            | Some Static_variant -> if m.rvalue <> None || modifier_text <> None then None else Some (make_variant ~vraw:input Static_variant m.rroot)
            | Some Functional_variant -> (
                let modifier =
                  match modifier_text with
                  | Some t -> ( match parse_modifier t with Some md -> Some (Some md) | None -> None)
                  | None -> Some None
                in
                match modifier with
                | None -> None
                | Some vmodifier -> (
                    match m.rvalue with
                    | None -> Some (make_variant ~vmodifier ~vraw:input Functional_variant m.rroot)
                    | Some value ->
                        if has_suffix value "]" then
                          if not (has_prefix value "[") then try_roots rest
                          else
                            let decoded = decode_arbitrary_value (String.sub value 1 (String.length value - 2)) in
                            if (not (is_valid_arbitrary decoded)) || String.trim decoded = "" then None
                            else
                              Some
                                (make_variant ~vvalue:(Some { vvkind = Arbitrary; vvalue = decoded }) ~vmodifier ~vraw:input
                                   Functional_variant m.rroot)
                        else if has_suffix value ")" then
                          if not (has_prefix value "(") then try_roots rest
                          else
                            let decoded = decode_arbitrary_value (String.sub value 1 (String.length value - 2)) in
                            if (not (is_valid_arbitrary decoded)) || String.trim decoded = "" || not (has_prefix decoded "--") then None
                            else
                              Some
                                (make_variant ~vvalue:(Some { vvkind = Arbitrary; vvalue = "var(" ^ decoded ^ ")" }) ~vmodifier
                                   ~vraw:input Functional_variant m.rroot)
                        else if not (is_named_value value) then try_roots rest
                        else Some (make_variant ~vvalue:(Some { vvkind = Named; vvalue = value }) ~vmodifier ~vraw:input Functional_variant m.rroot)))
            | Some Compound_variant -> (
                match m.rvalue with
                | None -> None
                | Some inner_text ->
                    let inner_text, remaining =
                      if (m.rroot = "not" || m.rroot = "has" || m.rroot = "in") && modifier_text <> None then
                        (inner_text ^ "/" ^ Option.get modifier_text, None)
                      else (inner_text, modifier_text)
                    in
                    match ds.parse_variant inner_text with
                    | None -> None
                    | Some inner ->
                        if not (ds.compounds_with m.rroot inner) then None
                        else
                          let modifier =
                            match remaining with
                            | Some t -> ( match parse_modifier t with Some md -> Some (Some md) | None -> None)
                            | None -> Some None
                          in
                          (match modifier with
                          | None -> None
                          | Some vmodifier -> Some (make_variant ~vmodifier ~inner:(Some inner) ~vraw:input Compound_variant m.rroot)))
            | Some Arbitrary_variant -> try_roots rest)
      in
      try_roots (find_roots name ds.has_variant)
