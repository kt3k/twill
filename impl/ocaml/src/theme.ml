(* The theme (SPEC §7). *)

open Utils

(* Option bits (SPEC §4.1.2). *)
let inline = 1
let reference = 2
let default = 4
let static = 8
let used = 16

type entry = { mutable value : string; mutable options : int }

type t = {
  mutable keys : string list;
  values : (string, entry) Hashtbl.t;
  mutable keyframes : Ast.at_rule list;
  mutable prefix : string;
}

let create () = { keys = []; values = Hashtbl.create 512; keyframes = []; prefix = "" }
let size t = Hashtbl.length t.values

let ignored_namespaces =
  [ ("--font", [ "--font-weight"; "--font-size" ]);
    ("--inset", [ "--inset-shadow"; "--inset-ring" ]);
    ( "--text",
      [ "--text-color"; "--text-decoration-color"; "--text-decoration-thickness"; "--text-indent"; "--text-shadow";
        "--text-underline-offset" ] );
    ("--grid-column", [ "--grid-column-start"; "--grid-column-end" ]);
    ("--grid-row", [ "--grid-row-start"; "--grid-row-end" ]) ]

let is_ignored_key namespace key =
  match List.assoc_opt namespace ignored_namespaces with
  | Some list -> List.exists (fun ns -> key = ns || has_prefix key (ns ^ "-")) list
  | None -> false

let delete t key =
  if Hashtbl.mem t.values key then begin
    Hashtbl.remove t.values key;
    t.keys <- List.filter (fun k -> k <> key) t.keys
  end

let has t key = Hashtbl.mem t.values key

let clear_namespace t namespace =
  List.iter
    (fun key -> if has_prefix key namespace && not (is_ignored_key namespace key) then delete t key)
    t.keys

(* Registers a value (SPEC §7.2). *)
let add t key value options =
  if has_suffix key "-*" then begin
    if value <> "initial" then Twill_error.failf "Invalid theme value `%s` for namespace `%s`" value key;
    if key = "--*" then (t.keys <- []; Hashtbl.reset t.values)
    else clear_namespace t (String.sub key 0 (String.length key - 2))
  end
  else
    let skip =
      options land default <> 0
      && match Hashtbl.find_opt t.values key with Some e -> e.options land default = 0 | None -> false
    in
    if not skip then
      if value = "initial" then delete t key
      else
        match Hashtbl.find_opt t.values key with
        | Some e -> e.value <- value; e.options <- options
        | None ->
            t.keys <- t.keys @ [ key ];
            Hashtbl.replace t.values key { value; options }

let add_keyframes t (node : Ast.at_rule) =
  if not (List.exists (fun k -> k == node) t.keyframes) then t.keyframes <- t.keyframes @ [ node ]

let keyframes t = t.keyframes
let keys t = t.keys
let entry t key = Hashtbl.find_opt t.values key
let options t key = match entry t key with Some e -> e.options | None -> 0

(* Finds the key for a candidate value in the namespaces (SPEC §7.3). *)
let resolve_key t (candidate : string option) namespaces =
  let rec go = function
    | [] -> None
    | ns :: rest -> (
        let key = match candidate with None -> ns | Some v -> ns ^ "-" ^ v in
        let key =
          if has t key then Some key
          else
            match candidate with
            | Some v when String.contains v '.' ->
                let k = ns ^ "-" ^ String.map (fun c -> if c = '.' then '_' else c) v in
                if has t k then Some k else None
            | _ -> None
        in
        match key with
        | Some k when not (is_ignored_key ns k) -> Some k
        | _ -> go rest)
  in
  go namespaces

let prefix_key t key = if t.prefix = "" then key else "--" ^ t.prefix ^ "-" ^ after key 2

let unprefix_key t key =
  if t.prefix = "" then key
  else
    let p = "--" ^ t.prefix ^ "-" in
    if has_prefix key p then "--" ^ after key (String.length p) else key

let reference_of t key opts =
  let e = Hashtbl.find t.values key in
  if (opts lor e.options) land inline <> 0 then e.value
  else
    let name = escape (prefix_key t key) in
    if e.options land reference <> 0 then "var(" ^ name ^ ", " ^ e.value ^ ")" else "var(" ^ name ^ ")"

(* Returns a var(...) reference or inline value for a candidate value. *)
let resolve t candidate namespaces opts =
  match resolve_key t candidate namespaces with Some key -> Some (reference_of t key opts) | None -> None

(* Returns the raw value for a candidate value. *)
let resolve_value t candidate namespaces =
  match resolve_key t candidate namespaces with Some key -> Some (Hashtbl.find t.values key).value | None -> None

(* Resolves a key and its nested sub-keys. *)
let resolve_with t candidate namespaces nested_keys =
  match resolve_key t candidate namespaces with
  | None -> None
  | Some key ->
      let extra =
        List.filter_map
          (fun nested ->
            let k = key ^ nested in
            if has t k then Some (nested, reference_of t k 0) else None)
          nested_keys
      in
      Some (reference_of t key 0, extra)

(* Returns the raw value of the first key that exists. *)
let rec get t = function
  | [] -> None
  | key :: rest -> ( match entry t key with Some e -> Some e.value | None -> get t rest)

type namespace_entry = { key : string; self : bool; value : string }

let namespace t ns =
  let prefix = ns ^ "-" in
  List.filter_map
    (fun key ->
      let e = Hashtbl.find t.values key in
      if key = ns then Some { key = ""; self = true; value = e.value }
      else if has_prefix key (ns ^ "--") then Some { key = after key (String.length ns); self = false; value = e.value }
      else if has_prefix key prefix then Some { key = after key (String.length prefix); self = false; value = e.value }
      else None)
    t.keys

let keys_in_namespaces t namespaces =
  List.concat_map
    (fun ns ->
      let prefix = ns ^ "-" in
      List.filter_map
        (fun key ->
          if has_prefix key prefix && (not (contains (after key 2) "--")) && not (is_ignored_key ns key) then
            Some (after key (String.length prefix))
          else None)
        t.keys)
    namespaces

(* Sets [used] on the unprefixed, unescaped key; true when newly set. *)
let mark_used_variable t key =
  match entry t (unprefix_key t (unescape key)) with
  | Some e when e.options land used = 0 -> e.options <- e.options lor used; true
  | _ -> false

let number_re = Str.regexp "^[+-]?\\([0-9]+\\.?[0-9]*\\|\\.[0-9]+\\)\\([eE][+-]?[0-9]+\\)?$"

(* Formats a float the way Go's strconv.FormatFloat(f, 'f', -1, 64) does. *)
let format_float x =
  if Float.is_integer x && Float.abs x < 1e15 then Printf.sprintf "%.0f" x
  else
    let rec go p =
      let s = Printf.sprintf "%.*f" p x in
      if p >= 20 || float_of_string s = x then s else go (p + 1)
    in
    go 1

(* Applies an alpha value to a color (SPEC §10.3). *)
let with_alpha color alpha =
  if alpha = "" then color
  else
    let alpha =
      if Str.string_match number_re alpha 0 then
        match float_of_string_opt alpha with Some n -> format_float (n *. 100.) ^ "%" | None -> alpha
      else alpha
    in
    if alpha = "100%" then color else "color-mix(in oklab, " ^ color ^ " " ^ alpha ^ ", transparent)"

(* Resolves a path such as `--color-red-500/50`. *)
let resolve_theme_value t path force_inline =
  let trimmed = String.trim path in
  let key, modifier =
    match String.rindex_opt trimmed '/' with
    | Some slash -> (String.trim (String.sub trimmed 0 slash), Some (String.trim (after trimmed (slash + 1))))
    | None -> (trimmed, None)
  in
  match resolve t None [ key ] (if force_inline then inline else 0) with
  | None -> None
  | Some value -> ( match modifier with Some m -> Some (with_alpha value m) | None -> Some value)
