(* The utility registry and the utility-definition helpers (SPEC §4.1.7, §10.1 through §10.3). *)

open Ast
open Candidate

type compile_status = Not_handled | Handled | Invalid
type compile_fn = candidate -> node list * compile_status

type definition = {
  ukind : utility_kind;
  compile : compile_fn;
  (* Marks a fallback definition when it has more than one entry including "any". *)
  types : string list option;
}

let is_fallback d = match d.types with Some ts when List.length ts > 1 -> List.mem "any" ts | _ -> false

type t = { defs : (string, definition list) Hashtbl.t; mutable names_rev : string list }

let create () = { defs = Hashtbl.create 1024; names_rev = [] }

let add u name d =
  match Hashtbl.find_opt u.defs name with
  | Some existing -> Hashtbl.replace u.defs name (existing @ [ d ])
  | None ->
      u.names_rev <- name :: u.names_rev;
      Hashtbl.replace u.defs name [ d ]

let static u name compile = add u name { ukind = Static; compile; types = None }
let functional u ?types name compile = add u name { ukind = Functional; compile; types }
let has u name kind = match Hashtbl.find_opt u.defs name with Some ds -> List.exists (fun d -> d.ukind = kind) ds | None -> false
let get u name = match Hashtbl.find_opt u.defs name with Some ds -> ds | None -> []
let names u = List.rev u.names_rev

(* Applies a modifier to a color value (SPEC §10.3). *)
let as_color value (modifier : modifier option) theme =
  match modifier with
  | None -> Some value
  | Some { mkind = Arbitrary; mvalue } -> Some (Theme.with_alpha value mvalue)
  | Some { mvalue; _ } -> (
      match Theme.resolve theme (Some mvalue) [ "--opacity" ] 0 with
      | Some opacity -> Some (Theme.with_alpha value opacity)
      | None -> if Utils.is_multiple_of_quarter mvalue then Some (Theme.with_alpha value (mvalue ^ "%")) else None)

(* Resolves a named color candidate value through the theme. *)
let resolve_theme_color (c : candidate) theme_keys theme =
  match c.cvalue with
  | Some { vkind = Named; value; _ } -> (
      let resolved =
        match value with
        | "inherit" -> Some "inherit"
        | "transparent" -> Some "transparent"
        | "current" -> Some "currentcolor"
        | _ -> Theme.resolve theme (Some value) theme_keys 0
      in
      match resolved with Some v -> as_color v c.cmodifier theme | None -> None)
  | _ -> None

let not_handled = ([], Not_handled)
let handled nodes = (nodes, Handled)

(* Registers a static definition returning the declarations. *)
let static_utility u name declarations =
  static u name (fun _ -> handled (List.map (fun (p, v) -> decl p v) declarations))

(* Registers a static definition backed by a function. *)
let static_utility_fn u name f = static u name (fun _ -> handled (f ()))

(* Describes a functional utility (SPEC §10.2). *)
type description = {
  supports_negative : bool;
  supports_fractions : bool;
  theme_keys : string list;
  (* The value for a candidate without a value segment; [no_default] marks the
     null case and when both are unset the first namespace itself resolves. *)
  default_value : string option;
  no_default : bool;
  static_values : (string * node list) list;
  handle_bare_value : (candidate_value -> string option) option;
  handle_negative_bare_value : (candidate_value -> string option) option;
  handle : string -> string option -> node list;
}

let describe ?(supports_negative = false) ?(supports_fractions = false) ?(theme_keys = []) ?default_value
    ?(no_default = false) ?(static_values = []) ?handle_bare_value ?handle_negative_bare_value handle =
  { supports_negative; supports_fractions; theme_keys; default_value; no_default; static_values; handle_bare_value;
    handle_negative_bare_value; handle }

exception Result of (node list * compile_status)

(* Registers a functional definition (SPEC §10.2). *)
let functional_utility u theme root desc =
  let compile (c : candidate) negative =
    try
      let resolved = ref None and data_type = ref None in
      (match c.cvalue with
      | None ->
          if c.cmodifier <> None then raise (Result not_handled);
          if desc.no_default then ()
          else if desc.default_value <> None then resolved := desc.default_value
          else resolved := Theme.resolve theme None desc.theme_keys 0
      | Some { vkind = Arbitrary; value; data_type = dt; _ } ->
          if c.cmodifier <> None then raise (Result not_handled);
          resolved := Some value;
          data_type := dt
      | Some named ->
          let fraction_consumed = ref false in
          (match named.fraction with
          | Some f -> (
              match Theme.resolve theme (Some f) desc.theme_keys 0 with
              | Some v ->
                  resolved := Some v;
                  fraction_consumed := true
              | None -> ())
          | None -> ());
          if !resolved = None then begin
            match Theme.resolve theme (Some named.value) desc.theme_keys 0 with
            | Some v ->
                if c.cmodifier <> None && not !fraction_consumed then raise (Result not_handled);
                resolved := Some v
            | None -> ()
          end;
          if !resolved = None && desc.supports_fractions then begin
            match named.fraction with
            | Some f ->
                let a, b = match String.index_opt f '/' with Some i -> (String.sub f 0 i, Utils.after f (i + 1)) | None -> (f, "") in
                if not (Utils.is_positive_integer a && Utils.is_positive_integer b) then raise (Result not_handled);
                resolved := Some ("calc(" ^ a ^ " / " ^ b ^ " * 100%)")
            | None -> ()
          end;
          if !resolved = None && negative then begin
            match desc.handle_negative_bare_value with
            | Some h -> (
                match h named with
                | Some bare ->
                    if (not (String.contains bare '/')) && c.cmodifier <> None then raise (Result not_handled);
                    raise (Result (handled (desc.handle bare None)))
                | None -> ())
            | None -> ()
          end;
          if !resolved = None then begin
            match desc.handle_bare_value with
            | Some h -> (
                match h named with
                | Some bare ->
                    if (not (String.contains bare '/')) && c.cmodifier <> None then raise (Result not_handled);
                    resolved := Some bare
                | None -> ())
            | None -> ()
          end;
          if !resolved = None && (not negative) && c.cmodifier = None then begin
            match List.assoc_opt named.value desc.static_values with
            | Some nodes -> raise (Result (handled (clone_nodes nodes)))
            | None -> ()
          end);
      match !resolved with
      | None -> not_handled
      | Some value ->
          let value = if negative then "calc(" ^ value ^ " * -1)" else value in
          let nodes = desc.handle value !data_type in
          if nodes = [] then not_handled else handled nodes
    with Result r -> r
  in
  functional u root (fun c -> compile c false);
  if desc.supports_negative then functional u ("-" ^ root) (fun c -> compile c true)

(* Registers a functional definition that resolves its value as a color. *)
let color_utility u theme root theme_keys handle =
  functional u root (fun (c : candidate) ->
      match c.cvalue with
      | None -> not_handled
      | Some { vkind = Arbitrary; value; _ } -> (
          match as_color value c.cmodifier theme with Some v -> handled (handle v) | None -> not_handled)
      | Some _ -> ( match resolve_theme_color c theme_keys theme with Some v -> handled (handle v) | None -> not_handled))

(* Registers `<name>-px`, optionally `-<name>-px`, and a functional utility
   whose bare values are spacing multipliers. *)
let spacing_utility u theme name theme_keys handle supports_negative supports_fractions =
  static_utility_fn u (name ^ "-px") (fun () -> handle "1px");
  if supports_negative then static_utility_fn u ("-" ^ name ^ "-px") (fun () -> handle "-1px");
  let spacing prefix (v : candidate_value) =
    if not (Utils.is_multiple_of_quarter v.value) then None
    else if Theme.resolve theme None [ "--spacing" ] 0 = None then None
    else Some ("--spacing(" ^ prefix ^ v.value ^ ")")
  in
  functional_utility u theme name
    (describe ~theme_keys ~no_default:true ~supports_negative ~supports_fractions ~handle_bare_value:(spacing "")
       ~handle_negative_bare_value:(spacing "-") (fun value _ -> handle value))

(* Resolves a bare value when it is a positive integer. *)
let bare_integer (v : candidate_value) = if Utils.is_positive_integer v.value then Some v.value else None

(* Resolves a bare positive integer with a unit suffix. *)
let bare_suffix suffix (v : candidate_value) = if Utils.is_positive_integer v.value then Some (v.value ^ suffix) else None

(* The explicit or inferred data type of an arbitrary candidate value. *)
let arbitrary_type (c : candidate) types =
  match c.cvalue with
  | Some { data_type = Some t; _ } -> t
  | Some { value; _ } -> Data_types.infer_data_type value types
  | None -> ""
