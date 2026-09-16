(* The CSS AST (SPEC §4.1.1). Containers hold their children in a mutable
   reference so that walks can splice replacements in place. *)

module SMap = Map.Make (String)

(* Context metadata such as base, reference, theme, source, and sourceBase.
   Boolean flags are stored as "true". *)
type ctx = string SMap.t

type node =
  | Rule of rule
  | At_rule of at_rule
  | Declaration of declaration
  | Comment of comment
  | Context of context
  | At_root of at_root

and nodes = node list ref
and rule = { mutable selector : string; rule_nodes : nodes }
and at_rule = { mutable name : string; mutable params : string; at_nodes : nodes }

(* A declaration with [no_value] is never printed. *)
and declaration = {
  mutable property : string;
  mutable value : string;
  mutable important : bool;
  no_value : bool;
}

and comment = { comment : string }
and context = { context : ctx; ctx_nodes : nodes }
and at_root = { root_nodes : nodes }

let ctx_empty : ctx = SMap.empty
let ctx_of_list l : ctx = List.fold_left (fun m (k, v) -> SMap.add k v m) SMap.empty l
let ctx_get (c : ctx) key = SMap.find_opt key c
let ctx_bool (c : ctx) key = SMap.find_opt key c = Some "true"

(* Returns [c] with [other]'s entries applied. *)
let ctx_merge (c : ctx) (other : ctx) : ctx = SMap.union (fun _ _ v -> Some v) c other

let style_rule ?(nodes = []) selector = Rule { selector; rule_nodes = ref nodes }
let at_rule ?(nodes = []) name params = At_rule { name; params; at_nodes = ref nodes }

let is_space c = c = ' ' || c = '\t' || c = '\n' || c = '\r' || c = '\012'

(* Splits `@media (x)` into name and params. *)
let parse_at_rule_prelude ?(nodes = []) prelude =
  let text = String.trim prelude in
  let n = String.length text in
  let rec find i =
    if i >= n then at_rule ~nodes text ""
    else
      let c = text.[i] in
      if is_space c || c = '(' then
        at_rule ~nodes (String.sub text 0 i) (String.trim (String.sub text i (n - i)))
      else find (i + 1)
  in
  find 1

(* Creates an at-rule when [selector] starts with `@` and a style rule otherwise. *)
let rule ?(nodes = []) selector =
  if String.length selector > 0 && selector.[0] = '@' then parse_at_rule_prelude ~nodes selector
  else style_rule ~nodes selector

let decl ?(important = false) property value =
  Declaration { property; value; important; no_value = false }

let important_decl property value = decl ~important:true property value
let no_value_decl property = Declaration { property; value = ""; important = false; no_value = true }
let comment value = Comment { comment = value }
let context ?(nodes = []) c = Context { context = c; ctx_nodes = ref nodes }
let at_root nodes = At_root { root_nodes = ref nodes }

(* Returns the child list of a container node. *)
let children = function
  | Rule r -> Some r.rule_nodes
  | At_rule a -> Some a.at_nodes
  | Context c -> Some c.ctx_nodes
  | At_root r -> Some r.root_nodes
  | Declaration _ | Comment _ -> None

let rec clone_node = function
  | Rule r -> Rule { selector = r.selector; rule_nodes = ref (clone_nodes !(r.rule_nodes)) }
  | At_rule a -> At_rule { name = a.name; params = a.params; at_nodes = ref (clone_nodes !(a.at_nodes)) }
  | Declaration d -> Declaration { d with property = d.property }
  | Comment c -> Comment { comment = c.comment }
  | Context c -> Context { context = c.context; ctx_nodes = ref (clone_nodes !(c.ctx_nodes)) }
  | At_root r -> At_root { root_nodes = ref (clone_nodes !(r.root_nodes)) }

and clone_nodes nodes = List.map clone_node nodes

type action = Continue | Skip | Stop

(* Passed to a walk visitor. *)
type utils = {
  parent : node option;
  (* The merged context of every enclosing context node. *)
  wctx : ctx;
  (* The ancestors, outermost first. *)
  path : node list;
  mutable replacement : node list option;
}

(* Replaces the current node with [nodes], which are then walked in turn. *)
let replace_with u nodes = u.replacement <- Some nodes

type visitor = node -> utils -> action

(* Walks the tree depth first, parents before children. *)
let rec walk_from (nodes : nodes) (visit : visitor) parent ctx path =
  let rec loop before rest =
    match rest with
    | [] -> Continue
    | node :: tail -> (
        let node_ctx = match node with Context c -> ctx_merge ctx c.context | _ -> ctx in
        let u = { parent; wctx = node_ctx; path; replacement = None } in
        let action = visit node u in
        (match u.replacement with
        | Some repl -> nodes := List.rev_append before (repl @ tail)
        | None -> ());
        if action = Stop then Stop
        else
          match u.replacement with
          | Some repl -> loop before (repl @ tail)
          | None ->
              if action = Skip then loop (node :: before) tail
              else
                let stopped =
                  match children node with
                  | Some ch -> walk_from ch visit (Some node) node_ctx (path @ [ node ]) = Stop
                  | None -> false
                in
                if stopped then Stop else loop (node :: before) tail)
  in
  loop [] !nodes

let walk nodes visit = walk_from nodes visit None ctx_empty []

(* Walks the tree depth first, children before parents. *)
let rec walk_depth_from (nodes : nodes) (visit : node -> utils -> unit) parent ctx path =
  let rec loop before rest =
    match rest with
    | [] -> ()
    | node :: tail -> (
        let node_ctx = match node with Context c -> ctx_merge ctx c.context | _ -> ctx in
        (match children node with
        | Some ch -> walk_depth_from ch visit (Some node) node_ctx (path @ [ node ])
        | None -> ());
        let u = { parent; wctx = node_ctx; path; replacement = None } in
        visit node u;
        match u.replacement with
        | Some repl ->
            nodes := List.rev_append before (repl @ tail);
            loop (List.rev_append repl before) tail
        | None -> loop (node :: before) tail)
  in
  loop [] !nodes

let walk_depth nodes visit = walk_depth_from nodes visit None ctx_empty []
