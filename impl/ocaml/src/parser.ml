(* The CSS parser (SPEC §5.1). *)

open Ast

let has_prefix s p = String.length s >= String.length p && String.sub s 0 (String.length p) = p

let has_suffix s p =
  let n = String.length s and m = String.length p in
  n >= m && String.sub s (n - m) m = p

let ends_with_important value =
  (* Matches `\s*!important\s*$` case-insensitively; returns the value before it. *)
  let trimmed = String.trim value in
  let n = String.length trimmed in
  if n >= 10 && String.lowercase_ascii (String.sub trimmed (n - 10) 10) = "!important" then
    Some (String.trim (String.sub trimmed 0 (n - 10)))
  else None

(* Parses `property: value [!important]`, or returns None when the text has no `:`. *)
let parse_declaration text =
  match String.index_opt text ':' with
  | None -> None
  | Some colon ->
      let property = String.trim (String.sub text 0 colon) in
      if property = "" then None
      else
        let value = String.trim (String.sub text (colon + 1) (String.length text - colon - 1)) in
        let value, important =
          match ends_with_important value with Some v -> (v, true) | None -> (value, false)
        in
        Some (Declaration { property; value; important; no_value = false })

let replace_crlf input =
  let b = Buffer.create (String.length input) in
  let n = String.length input in
  let i = ref 0 in
  while !i < n do
    if input.[!i] = '\r' && !i + 1 < n && input.[!i + 1] = '\n' then (Buffer.add_char b '\n'; i := !i + 2)
    else (Buffer.add_char b input.[!i]; incr i)
  done;
  Buffer.contents b

let index_from_opt input start sub =
  let n = String.length input and m = String.length sub in
  let rec go i = if i + m > n then None else if String.sub input i m = sub then Some i else go (i + 1) in
  go start

(* Parses CSS text into a list of nodes. *)
let parse input : node list =
  let input = replace_crlf input in
  let n = String.length input in
  let ast = ref [] in
  let stack = ref [] in
  let parent : node option ref = ref None in
  let buffer = Buffer.create 64 in
  let buffer_start = ref 0 in
  let brackets = Buffer.create 8 in
  let custom_property_braces = ref 0 in
  let fail message index = raise (Twill_error.Error (message, Some (Twill_error.position_at input index))) in
  let push node =
    match !parent with
    | Some p -> (
        match children p with Some ch -> ch := !ch @ [ node ] | None -> ())
    | None -> ast := !ast @ [ node ]
  in
  let finish end_ =
    let text = String.trim (Buffer.contents buffer) in
    Buffer.clear buffer;
    Buffer.clear brackets;
    custom_property_braces := 0;
    if text <> "" then
      if text.[0] = '@' then push (parse_at_rule_prelude text)
      else match parse_declaration text with Some d -> push d | None -> fail "Invalid declaration" end_
  in
  let i = ref 0 in
  while !i < n do
    let c = input.[!i] in
    if c = '\\' then begin
      if Buffer.length buffer = 0 then buffer_start := !i;
      let end_ = min (!i + 2) n in
      Buffer.add_string buffer (String.sub input !i (end_ - !i));
      i := !i + 2
    end
    else if c = '/' && !i + 1 < n && input.[!i + 1] = '*' then begin
      let start = !i in
      match index_from_opt input (!i + 2) "*/" with
      | None -> fail "Unterminated comment" start
      | Some end_ ->
          let text = String.sub input (start + 2) (end_ - start - 2) in
          if has_prefix text "!" && String.trim (Buffer.contents buffer) = "" then push (comment text);
          i := end_ + 2
    end
    else if c = '"' || c = '\'' then begin
      let start = !i in
      if Buffer.length buffer = 0 then buffer_start := !i;
      let j = ref (!i + 1) in
      let closed = ref false in
      while (not !closed) && !j < n do
        let d = input.[!j] in
        if d = '\\' then j := !j + 2
        else if d = c then closed := true
        else if d = '\n' then fail "Unterminated string" start
        else incr j
      done;
      if not !closed then fail "Unterminated string" start;
      Buffer.add_string buffer (String.sub input start (!j - start + 1));
      i := !j + 1
    end
    else if Buffer.length buffer = 0 && is_space c then incr i
    else begin
      if Buffer.length buffer = 0 then buffer_start := !i;
      let current = Buffer.contents buffer in
      let in_custom_property = has_prefix current "--" in
      if c = '(' || c = '[' then begin
        Buffer.add_char brackets (if c = '(' then ')' else ']');
        Buffer.add_char buffer c;
        incr i
      end
      else if c = ')' || c = ']' then begin
        let bl = Buffer.length brackets in
        if bl = 0 || Buffer.nth brackets (bl - 1) <> c then fail (Printf.sprintf "Unexpected `%c`" c) !i;
        Buffer.truncate brackets (bl - 1);
        Buffer.add_char buffer c;
        incr i
      end
      else if Buffer.length brackets > 0 then (Buffer.add_char buffer c; incr i)
      else if c = ';' then begin
        if !custom_property_braces > 0 then Buffer.add_char buffer c else finish !i;
        incr i
      end
      else if c = '{' then begin
        if in_custom_property && String.contains current ':' then begin
          incr custom_property_braces;
          Buffer.add_char buffer c
        end
        else begin
          let prelude = String.trim current in
          if prelude = "" then fail "Missing selector before `{`" !i;
          let node = if prelude.[0] = '@' then parse_at_rule_prelude prelude else style_rule prelude in
          push node;
          stack := !parent :: !stack;
          parent := Some node;
          Buffer.clear buffer
        end;
        incr i
      end
      else if c = '}' then begin
        if !custom_property_braces > 0 then begin
          decr custom_property_braces;
          Buffer.add_char buffer c
        end
        else begin
          if !parent = None then fail "Unexpected `}`" !i;
          finish !i;
          (match !stack with
          | p :: rest -> parent := p; stack := rest
          | [] -> parent := None)
        end;
        incr i
      end
      else (Buffer.add_char buffer c; incr i)
    end
  done;
  if !parent <> None then fail "Missing closing `}`" n;
  if Buffer.length brackets > 0 then
    fail (Printf.sprintf "Missing closing `%c`" (Buffer.nth brackets (Buffer.length brackets - 1))) n;
  if String.trim (Buffer.contents buffer) <> "" then finish !buffer_start;
  !ast
