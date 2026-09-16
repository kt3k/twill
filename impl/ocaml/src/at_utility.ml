(* @utility name validation (SPEC §6.8, §10.5). The bodies are compiled in
   the design system once the utility registry exists. *)

open Utils

type kind = Static | Functional

type t = { name : string; kind : kind; node : Ast.at_rule }

let functional_prefix_ok name =
  let body = trim_prefix name "-" in
  body <> "" && is_lower body.[0] && for_all_chars (fun c -> is_alnum c || c = '_' || c = '-') body

(* Whether [name] is a valid static utility name. *)
let is_valid_static_utility_name name =
  let n = String.length name in
  let start = if has_prefix name "-" then 1 else 0 in
  if start >= n || not (is_lower name.[start]) then false
  else begin
    let i = ref (start + 1) in
    while !i < n && (is_alnum name.[!i] || name.[!i] = '_' || name.[!i] = '-') do incr i done;
    let root = String.sub name 0 !i in
    let rest = after name !i in
    if has_suffix root "-" && rest = "" then false
    else if rest = "" then true
    else if not (for_all_chars (fun c -> is_alnum c || c = '_' || c = '.' || c = '/' || c = '%' || c = '-') rest) then false
    else if List.length (String.split_on_char '/' rest) > 2 || has_suffix rest "/" then false
    else begin
      let ok = ref true in
      String.iteri
        (fun i c ->
          if c = '.' then (if i = 0 || i = n - 1 || (not (is_digit name.[i - 1])) || not (is_digit name.[i + 1]) then ok := false)
          else if c = '%' then if i <> n - 1 || i = 0 || not (is_digit name.[i - 1]) then ok := false)
        name;
      !ok
    end
  end

let is_valid_functional_utility_name name =
  has_suffix name "-*" && functional_prefix_ok (String.sub name 0 (String.length name - 2))

(* Validates an @utility name. *)
let parse (node : Ast.at_rule) =
  let name = unescape (String.trim node.params) in
  if !(node.at_nodes) = [] then Twill_error.failf "`@utility %s` is empty. Utilities without a body are not supported." name;
  if is_valid_functional_utility_name name then { name = String.sub name 0 (String.length name - 2); kind = Functional; node }
  else if has_suffix name "*" then
    Twill_error.failf "`@utility %s` defines an invalid utility name. A functional utility must end in `-*`." name
  else if String.contains name '*' then
    Twill_error.failf "`@utility %s` defines an invalid utility name. The `*` must be at the end." name
  else if is_valid_static_utility_name name then { name; kind = Static; node }
  else
    Twill_error.failf
      "`@utility %s` defines an invalid utility name. Utilities should be alphabetic and start with a lowercase letter." name
