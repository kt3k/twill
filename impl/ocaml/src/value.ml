(* Declaration value parsing (SPEC §4.1.8). *)

type vnode = Word of word | Sep of string | Fn of fn
and word = { mutable w : string }
and fn = { mutable fname : string; mutable fnodes : vnode list }

let word s = Word { w = s }
let sep s = Sep s
let fn name nodes = Fn { fname = name; fnodes = nodes }

(* Parses a value into words, separators, and function calls. *)
let parse input : vnode list =
  let n = String.length input in
  let ast = ref [] in
  let stack = ref [] in
  let parent : fn option ref = ref None in
  let buffer = Buffer.create 32 in
  let push node =
    match !parent with Some f -> f.fnodes <- f.fnodes @ [ node ] | None -> ast := !ast @ [ node ]
  in
  let flush () =
    if Buffer.length buffer > 0 then begin
      push (word (Buffer.contents buffer));
      Buffer.clear buffer
    end
  in
  let i = ref 0 in
  while !i < n do
    let c = input.[!i] in
    if c = '\\' then begin
      let end_ = min (!i + 2) n in
      Buffer.add_string buffer (String.sub input !i (end_ - !i));
      i := !i + 2
    end
    else if c = '"' || c = '\'' then begin
      let start = !i in
      let j = ref (!i + 1) in
      let closed = ref false in
      while (not !closed) && !j < n do
        let d = input.[!j] in
        if d = '\\' then j := !j + 2 else if d = c then closed := true else incr j
      done;
      let end_ = min (!j + 1) n in
      Buffer.add_string buffer (String.sub input start (end_ - start));
      i := end_
    end
    else if c = '(' then begin
      let node = { fname = Buffer.contents buffer; fnodes = [] } in
      Buffer.clear buffer;
      push (Fn node);
      stack := !parent :: !stack;
      parent := Some node;
      incr i
    end
    else if c = ')' then begin
      flush ();
      (match !parent with
      | Some _ -> (
          match !stack with
          | p :: rest -> parent := p; stack := rest
          | [] -> parent := None)
      | None -> Buffer.add_char buffer ')');
      incr i
    end
    else if c = ',' || c = '/' then begin
      flush ();
      push (sep (String.make 1 c));
      incr i
    end
    else if Ast.is_space c then begin
      flush ();
      let j = ref !i in
      while !j < n && Ast.is_space input.[!j] do incr j done;
      push (sep (String.sub input !i (!j - !i)));
      i := !j
    end
    else (Buffer.add_char buffer c; incr i)
  done;
  flush ();
  !ast

let rec to_css_buf b nodes =
  List.iter
    (function
      | Word w -> Buffer.add_string b w.w
      | Sep s -> Buffer.add_string b s
      | Fn f ->
          Buffer.add_string b f.fname;
          Buffer.add_char b '(';
          to_css_buf b f.fnodes;
          Buffer.add_char b ')')
    nodes

(* Prints value nodes back to CSS text. *)
let to_css nodes =
  let b = Buffer.create 64 in
  to_css_buf b nodes;
  Buffer.contents b

type vaction = VContinue | VSkip | VReplace of vnode list

(* Walks value nodes depth first; replacements are not revisited. *)
let rec walk_from (nodes : vnode list ref) visit (parent : fn option) =
  let rec loop before rest =
    match rest with
    | [] -> ()
    | node :: tail -> (
        match visit node parent with
        | VReplace repl ->
            nodes := List.rev_append before (repl @ tail);
            loop (List.rev_append repl before) tail
        | VSkip -> loop (node :: before) tail
        | VContinue ->
            (match node with
            | Fn f ->
                let r = ref f.fnodes in
                walk_from r visit (Some f);
                f.fnodes <- !r
            | _ -> ());
            loop (node :: before) tail)
  in
  loop [] !nodes

let walk nodes visit = walk_from nodes visit None
