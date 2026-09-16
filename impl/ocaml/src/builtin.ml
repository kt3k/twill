(* The built-in stylesheets (SPEC §6.3) as embedded resources. theme.css
   and preflight.css are ported from Tailwind CSS (MIT License, Copyright
   (c) Tailwind Labs, Inc.). *)

(* The base of every built-in stylesheet; distinct from any user directory. *)
let builtin_base = "twill:"

type loaded = { path : string; base : string; content : string }

(* Loads a stylesheet by id; relative ids resolve against the base. *)
type loader = string -> string -> loaded

let is_builtin_path path = Utils.has_prefix path builtin_base
let builtin_css name = List.assoc_opt name Bundled_css.files

(* Resolves `twill`, `twill/<name>`, and relative ids requested from inside
   a built-in stylesheet; None for anything else. *)
let resolve_builtin id base =
  let name =
    if id = "twill" then Some "index.css"
    else if Utils.has_prefix id "twill/" then Some (Utils.after id 6)
    else if base = builtin_base then Some (Utils.trim_prefix id "./")
    else None
  in
  match name with
  | None -> None
  | Some name -> (
      match builtin_css name with
      | Some content -> Some { path = builtin_base ^ name; base = builtin_base; content }
      | None -> Twill_error.failf "Unknown built-in stylesheet `%s`" id)

(* Resolves only the built-in stylesheets. *)
let builtin_loader id base =
  match resolve_builtin id base with
  | Some l -> l
  | None -> Twill_error.failf "Cannot resolve `%s`: no stylesheet loader was provided" id
