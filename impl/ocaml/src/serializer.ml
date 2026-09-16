(* Serialization (SPEC §5.2). *)

open Ast

let rec serialize_nodes b nodes depth = List.iter (fun n -> serialize_node b n depth) nodes

and serialize_node b n depth =
  let indent = String.make (2 * depth) ' ' in
  match n with
  | Declaration d ->
      if not d.no_value then begin
        Buffer.add_string b indent;
        Buffer.add_string b d.property;
        Buffer.add_string b ": ";
        Buffer.add_string b d.value;
        if d.important then Buffer.add_string b " !important";
        Buffer.add_string b ";\n"
      end
  | Rule r ->
      Buffer.add_string b indent;
      Buffer.add_string b r.selector;
      Buffer.add_string b " {\n";
      serialize_nodes b !(r.rule_nodes) (depth + 1);
      Buffer.add_string b indent;
      Buffer.add_string b "}\n"
  | At_rule a ->
      Buffer.add_string b indent;
      Buffer.add_string b a.name;
      if a.params <> "" then (Buffer.add_string b " "; Buffer.add_string b a.params);
      if !(a.at_nodes) = [] then Buffer.add_string b ";\n"
      else begin
        Buffer.add_string b " {\n";
        serialize_nodes b !(a.at_nodes) (depth + 1);
        Buffer.add_string b indent;
        Buffer.add_string b "}\n"
      end
  | Comment c ->
      Buffer.add_string b indent;
      Buffer.add_string b "/*";
      Buffer.add_string b c.comment;
      Buffer.add_string b "*/\n"
  | Context c -> serialize_nodes b !(c.ctx_nodes) depth
  | At_root r -> serialize_nodes b !(r.root_nodes) depth

(* Prints nodes with two-space indentation per depth. *)
let serialize nodes =
  let b = Buffer.create 1024 in
  serialize_nodes b nodes 0;
  Buffer.contents b

let rec compact b nodes =
  List.iter
    (fun n ->
      match n with
      | Declaration d ->
          if not d.no_value then begin
            Buffer.add_string b d.property;
            Buffer.add_string b ":";
            Buffer.add_string b d.value;
            if d.important then Buffer.add_string b "!important";
            Buffer.add_string b ";"
          end
      | Rule r ->
          Buffer.add_string b r.selector;
          Buffer.add_string b "{";
          compact b !(r.rule_nodes);
          Buffer.add_string b "}"
      | At_rule a ->
          Buffer.add_string b a.name;
          if a.params <> "" then (Buffer.add_string b " "; Buffer.add_string b a.params);
          if !(a.at_nodes) = [] then Buffer.add_string b ";"
          else begin
            Buffer.add_string b "{";
            compact b !(a.at_nodes);
            Buffer.add_string b "}"
          end
      | Comment c ->
          Buffer.add_string b "/*";
          Buffer.add_string b c.comment;
          Buffer.add_string b "*/"
      | Context c -> compact b !(c.ctx_nodes)
      | At_root r -> compact b !(r.root_nodes))
    nodes

(* Prints nodes without indentation or line breaks; used for `--minify`. *)
let serialize_compact nodes =
  let b = Buffer.create 1024 in
  compact b nodes;
  Buffer.contents b
