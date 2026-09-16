(* Whole-document optimization before serialization (SPEC §12). *)

open Ast
open Utils

let keep_empty_at_rules = [ "@layer"; "@charset"; "@custom-media"; "@namespace"; "@import"; "@apply" ]

(* At-rules whose child rules are not nested style rules. *)
let opaque_at_rules = [ "@keyframes"; "@property"; "@font-face"; "@counter-style"; "@page" ]

let var_re = Str.regexp "var([ \t\n]*\\(--[^ \t\n,)]+\\)"

(* Every `--name` referenced through var(...) in a value. *)
let variables_in value =
  if not (contains value "var(") then []
  else begin
    let out = ref [] and pos = ref 0 in
    (try
       while true do
         let i = Str.search_forward var_re value !pos in
         out := Str.matched_group 1 value :: !out;
         pos := i + 1
       done
     with Not_found -> ());
    List.rev !out
  end

let split_tokens value seps =
  let out = ref [] and start = ref (-1) in
  let n = String.length value in
  for i = 0 to n do
    if i = n || String.contains seps value.[i] then begin
      if !start >= 0 then (out := String.sub value !start (i - !start) :: !out; start := -1)
    end
    else if !start < 0 then start := i
  done;
  List.rev !out

(* Combines a parent selector with a nested selector (SPEC §12.3). *)
let combine_selectors parent child =
  let wrapped = if List.length (segment parent ',') > 1 then ":is(" ^ parent ^ ")" else parent in
  if contains child "&" then replace_all child "&" wrapped
  else if String.length child > 0 && (child.[0] = '>' || child.[0] = '+' || child.[0] = '~') then wrapped ^ " " ^ child
  else wrapped ^ child

(* Flattens nested rules and at-rules into flat CSS (SPEC §12.3). *)
let rec flatten nodes =
  List.concat_map
    (fun n ->
      match n with
      | Rule r -> flatten_rule r None
      | At_rule a -> if List.mem a.name opaque_at_rules then [ n ] else [ at_rule a.name a.params ~nodes:(flatten !(a.at_nodes)) ]
      | Context c -> flatten !(c.ctx_nodes)
      | At_root r -> flatten !(r.root_nodes)
      | _ -> [ n ])
    nodes

and flatten_rule (r : rule) parent_selector =
  let selector =
    match parent_selector with
    | None -> r.selector
    | Some p -> if r.selector = "&" then p else combine_selectors p r.selector
  in
  flatten_children !(r.rule_nodes) selector

and flatten_children nodes selector =
  let declarations = ref [] and hoisted = ref [] in
  List.iter
    (fun child ->
      match child with
      | Declaration _ | Comment _ -> declarations := child :: !declarations
      | Rule r -> hoisted := List.rev_append (flatten_rule r (Some selector)) !hoisted
      | At_rule a ->
          if !(a.at_nodes) = [] then declarations := child :: !declarations
          else hoisted := List.rev_append (flatten_at_rule a selector) !hoisted
      | Context c -> hoisted := List.rev_append (flatten_children !(c.ctx_nodes) selector) !hoisted
      | At_root r -> hoisted := List.rev_append (flatten_children !(r.root_nodes) selector) !hoisted)
    nodes;
  let declarations = List.rev !declarations in
  (if declarations <> [] then [ style_rule selector ~nodes:declarations ] else []) @ List.rev !hoisted

and flatten_at_rule (a : at_rule) selector =
  if List.mem a.name opaque_at_rules then [ At_rule a ]
  else
    match flatten_children !(a.at_nodes) selector with
    | [] -> []
    | children -> [ at_rule a.name a.params ~nodes:children ]

let rec remove_nodes nodes (removed_decls : declaration list) (removed_keyframes : at_rule list) =
  List.concat_map
    (fun n ->
      match n with
      | Declaration d when List.memq d removed_decls -> []
      | At_rule a when List.memq a removed_keyframes -> []
      | Rule r -> (
          match remove_nodes !(r.rule_nodes) removed_decls removed_keyframes with
          | [] -> []
          | children -> [ style_rule r.selector ~nodes:children ])
      | At_rule a ->
          let children = remove_nodes !(a.at_nodes) removed_decls removed_keyframes in
          if children = [] && !(a.at_nodes) <> [] && ((not (List.mem a.name keep_empty_at_rules)) || a.name = "@layer") then []
          else [ at_rule a.name a.params ~nodes:children ]
      | _ -> [ n ])
    nodes

(* Optimizes the whole document before serialization. The input is not
   mutated; a new tree is returned. *)
let optimize_ast ast (ds : Design_system.t) =
  let theme = ds.theme in
  let theme_declarations = ref [] in
  let theme_keyframes = ref [] in
  let used_variables = Hashtbl.create 64 in
  let dependencies = Hashtbl.create 64 in
  let used_keyframes = Hashtbl.create 16 in
  let seen_properties = Hashtbl.create 64 in
  let roots = ref [] in
  let record_variables value owner_key is_theme =
    List.iter
      (fun name ->
        if not is_theme then Hashtbl.replace used_variables name ()
        else
          let existing = match Hashtbl.find_opt dependencies owner_key with Some l -> l | None -> [] in
          if not (List.mem name existing) then Hashtbl.replace dependencies owner_key (name :: existing))
      (variables_in value)
  in
  let rec transform nodes (out : node list ref) ctx depth =
    List.iter
      (fun n ->
        match n with
        | Declaration d when d.no_value || d.property = "--tw-sort" -> ()
        | Declaration d ->
            let copy = { property = d.property; value = d.value; important = d.important; no_value = d.no_value } in
            let skip =
              if ctx_bool ctx "theme" && has_prefix d.property "--" then
                if d.value = "initial" then true
                else begin
                  let key = Theme.unprefix_key theme (unescape d.property) in
                  record_variables d.value key true;
                  theme_declarations := (copy, key) :: !theme_declarations;
                  false
                end
              else (record_variables d.value "" false; false)
            in
            if not skip then begin
              if d.property = "animation" then
                List.iter (fun token -> Hashtbl.replace used_keyframes token ()) (split_tokens d.value " ,\t\n");
              out := Declaration copy :: !out
            end
        | Rule r ->
            let children = ref [] in
            transform !(r.rule_nodes) children ctx (depth + 1);
            if !children <> [] then out := style_rule r.selector ~nodes:(List.rev !children) :: !out
        | At_rule a ->
            let duplicate =
              a.name = "@property" && depth = 0
              && (if Hashtbl.mem seen_properties a.params then true else (Hashtbl.replace seen_properties a.params (); false))
            in
            if not duplicate then begin
              let children = ref [] in
              transform !(a.at_nodes) children ctx (depth + 1);
              if !children = [] && not (List.mem a.name keep_empty_at_rules) then ()
              else begin
                let copy = { name = a.name; params = a.params; at_nodes = ref (List.rev !children) } in
                if a.name = "@keyframes" && ctx_bool ctx "theme" then theme_keyframes := copy :: !theme_keyframes;
                out := At_rule copy :: !out
              end
            end
        | At_root r -> transform !(r.root_nodes) roots ctx 0
        | Context c -> if ctx_bool c.context "reference" then () else transform !(c.ctx_nodes) out (ctx_merge ctx c.context) depth
        | Comment c -> out := Comment { comment = c.comment } :: !out)
      nodes
  in
  let result = ref [] in
  transform ast result ctx_empty 0;
  let result = List.rev !result and roots = List.rev !roots in
  let theme_declarations = List.rev !theme_declarations and theme_keyframes = List.rev !theme_keyframes in
  Hashtbl.iter (fun name () -> ignore (Theme.mark_used_variable theme name)) used_variables;
  let keep = Hashtbl.create 64 in
  List.iter
    (fun (_, key) -> if Theme.options theme key land (Theme.static lor Theme.used) <> 0 then Hashtbl.replace keep key ())
    theme_declarations;
  let changed = ref true in
  while !changed do
    changed := false;
    List.iter
      (fun key ->
        List.iter
          (fun dep ->
            let dep_key = Theme.unprefix_key theme (unescape dep) in
            if (not (Hashtbl.mem keep dep_key)) && Theme.has theme dep_key then begin
              Hashtbl.replace keep dep_key ();
              changed := true
            end)
          (match Hashtbl.find_opt dependencies key with Some l -> l | None -> []))
      (Hashtbl.fold (fun k () acc -> k :: acc) keep [])
  done;
  let removed_decls = List.filter_map (fun (d, key) -> if Hashtbl.mem keep key then None else Some d) theme_declarations in
  Hashtbl.iter
    (fun key () ->
      if has_prefix key "--animate" then
        match Theme.get theme [ key ] with
        | Some value -> List.iter (fun token -> Hashtbl.replace used_keyframes token ()) (split_tokens value " ,")
        | None -> ())
    keep;
  let removed_keyframes = List.filter (fun (k : at_rule) -> not (Hashtbl.mem used_keyframes (String.trim k.params))) theme_keyframes in
  let result, roots =
    if removed_decls <> [] || removed_keyframes <> [] then
      (remove_nodes result removed_decls removed_keyframes, remove_nodes roots removed_decls removed_keyframes)
    else (result, roots)
  in
  flatten (result @ roots)
