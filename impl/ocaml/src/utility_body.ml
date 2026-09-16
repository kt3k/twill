(* Functional @utility bodies and custom utility registration (SPEC §6.8, §10.6). *)

open Ast
open Candidate
open Utils

let arg_space_re = Str.regexp "\\(--[a-zA-Z0-9_-]+?\\)\\(-\\*\\)?[ \t\n]+\\(--\\)"
let whitespace_re = Str.regexp "[ \t\n\r]+"
let repeated_star_re = Str.regexp "\\(-\\*\\)+"
let bare_namespace_re = Str.regexp "^--[a-zA-Z0-9_-]+$"

(* Joins a namespace, an optional star suffix, and a following `--` key with
   `-*`, so `--text-* --line-height` becomes `--text-*--line-height`. *)
let arg_space_replace a =
  let b = Buffer.create (String.length a) in
  let n = String.length a in
  let i = ref 0 in
  while !i < n do
    if Str.string_match arg_space_re a !i then begin
      let matched_end = Str.match_end () in
      let ns = Str.matched_group 1 a in
      let ns = if has_suffix ns "-*" then String.sub ns 0 (String.length ns - 2) else ns in
      Buffer.add_string b (ns ^ "-*--");
      i := matched_end
    end
    else begin
      Buffer.add_char b a.[!i];
      incr i
    end
  done;
  Buffer.contents b

(* Normalizes one --value(...) or --modifier(...) argument. *)
let normalize_argument arg =
  let a = replace_all arg "\\*" "*" in
  let a = arg_space_replace a in
  let a = Str.global_replace whitespace_re "" a in
  let a = Str.global_replace repeated_star_re "-*" a in
  if Str.string_match bare_namespace_re a 0 && not (has_suffix a "-*") then a ^ "-*" else a

type value_like = { lkind : value_kind; lvalue : string; lfraction : string option; ldata_type : string option }
type resolution = { rvalue : string; ratio : bool }

let plain v = Some { rvalue = v; ratio = false }

let resolve_argument arg target (ds : Design_system.t) =
  if has_prefix arg "--default(" && has_suffix arg ")" then
    match target with None -> plain (String.sub arg 10 (String.length arg - 11)) | Some _ -> None
  else
    match target with
    | None -> None
    | Some t -> (
        match unquote arg with
        | Some literal -> if t.lkind = Named && t.lvalue = literal then plain literal else None
        | None ->
            if has_prefix arg "--" then begin
              if t.lkind <> Named then None
              else
                match index_of arg "-*" with
                | -1 -> None
                | star -> (
                    let namespace = String.sub arg 0 star and sub = after arg (star + 2) in
                    if sub = "" then begin
                      let from_fraction =
                        match t.lfraction with Some f -> Theme.resolve ds.theme (Some f) [ namespace ] 0 | None -> None
                      in
                      match from_fraction with
                      | Some v -> plain v
                      | None -> ( match Theme.resolve ds.theme (Some t.lvalue) [ namespace ] 0 with Some v -> plain v | None -> None)
                    end
                    else if not (has_prefix sub "--") then None
                    else
                      match Theme.resolve_with ds.theme (Some t.lvalue) [ namespace ] [ sub ] with
                      | Some (_, extra) -> ( match List.assoc_opt sub extra with Some v -> plain v | None -> None)
                      | None -> None)
            end
            else if has_prefix arg "[" && has_suffix arg "]" then begin
              if t.lkind <> Arbitrary then None
              else
                let typ = String.sub arg 1 (String.length arg - 2) in
                if typ = "*" then plain t.lvalue
                else
                  match t.ldata_type with
                  | Some dt -> if dt = typ then plain t.lvalue else None
                  | None ->
                      if not (Data_types.is_data_type typ) then None
                      else if Data_types.infer_data_type t.lvalue [ typ ] = typ then plain t.lvalue
                      else None
            end
            else if t.lkind <> Named then None
            else
              match arg with
              | "number" -> if is_multiple_of_quarter t.lvalue then plain t.lvalue else None
              | "integer" -> if is_positive_integer t.lvalue then plain t.lvalue else None
              | "percentage" ->
                  let v = t.lvalue in
                  if String.length v > 1 && has_suffix v "%" && Data_types.is_integer (trim_suffix v "%") then plain v else None
              | "ratio" -> (
                  match t.lfraction with
                  | None -> None
                  | Some f -> (
                      match String.index_opt f '/' with
                      | None -> None
                      | Some i ->
                          let a = String.sub f 0 i and b = after f (i + 1) in
                          if is_positive_integer a && is_positive_integer b then Some { rvalue = a ^ " / " ^ b; ratio = true }
                          else None))
              | _ -> None)

(* Compiles a functional @utility body for a candidate (SPEC §10.6). *)
let compile_functional_body (c : candidate) body ds =
  let value_target =
    match c.cvalue with
    | Some v -> Some { lkind = v.vkind; lvalue = v.value; lfraction = v.fraction; ldata_type = v.data_type }
    | None -> None
  in
  let modifier_target =
    match c.cmodifier with Some m -> Some { lkind = m.mkind; lvalue = m.mvalue; lfraction = None; ldata_type = None } | None -> None
  in
  let nodes = ref (clone_nodes body) in
  let saw_value_fn = ref false and resolved_value = ref false and resolved_ratio = ref false in
  let saw_modifier_fn = ref false and resolved_modifier = ref false in
  let non_ratio = ref [] in
  ignore
    (walk nodes (fun n u ->
         match n with
         | At_root _ -> Skip
         | Declaration d when (not d.no_value) && (contains d.value "--value(" || contains d.value "--modifier(") ->
             let failed = ref false and used_ratio = ref false and used_non_ratio = ref false in
             let ast = ref (Value.parse d.value) in
             Value.walk ast (fun vn _ ->
                 match vn with
                 | Value.Fn f when f.fname = "--value" || f.fname = "--modifier" -> (
                     let is_value = f.fname = "--value" in
                     let target =
                       if is_value then (saw_value_fn := true; value_target) else (saw_modifier_fn := true; modifier_target)
                     in
                     let found =
                       List.fold_left
                         (fun acc arg ->
                           match acc with
                           | Some _ -> acc
                           | None ->
                               let arg = normalize_argument (String.trim arg) in
                               if arg = "" then None else resolve_argument arg target ds)
                         None
                         (segment (Value.to_css f.fnodes) ',')
                     in
                     match found with
                     | None ->
                         failed := true;
                         Value.VSkip
                     | Some res ->
                         if is_value then begin
                           resolved_value := true;
                           if res.ratio then (resolved_ratio := true; used_ratio := true) else used_non_ratio := true
                         end
                         else resolved_modifier := true;
                         Value.VReplace [ Value.word res.rvalue ])
                 | _ -> Value.VContinue);
             if !failed then replace_with u []
             else begin
               d.value <- Value.to_css !ast;
               if !used_non_ratio && not !used_ratio then non_ratio := d :: !non_ratio
             end;
             Continue
         | _ -> Continue));
  if (not !saw_value_fn) || not !resolved_value then None
  else if !saw_modifier_fn && (not !resolved_modifier) && c.cmodifier <> None then None
  else if !resolved_ratio && !resolved_modifier then None
  else if c.cmodifier <> None && (not !resolved_ratio) && not !resolved_modifier then None
  else begin
    if !resolved_ratio && !non_ratio <> [] then
      ignore
        (walk nodes (fun n u ->
             (match n with Declaration d when List.memq d !non_ratio -> replace_with u [] | _ -> ());
             Continue));
    Some !nodes
  end

(* Registers a parsed @utility on the design system. Its body is read
   lazily so @apply can expand first. *)
let register (ds : Design_system.t) (cu : At_utility.t) =
  match cu.kind with
  | Static -> Utilities.static ds.utilities cu.name (fun _ -> Utilities.handled (clone_nodes !(cu.node.at_nodes)))
  | Functional ->
      Utilities.functional ds.utilities cu.name (fun c ->
          match compile_functional_body c !(cu.node.at_nodes) ds with
          | Some nodes -> Utilities.handled nodes
          | None -> Utilities.not_handled)
