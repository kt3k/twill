(* Errors raised by the compiler (SPEC §4.1). *)

type position = { line : int; column : int }

exception Error of string * position option

let message = function
  | Error (message, Some p) -> Printf.sprintf "%s (%d:%d)" message p.line p.column
  | Error (message, None) -> message
  | e -> Printexc.to_string e

let fail msg = raise (Error (msg, None))
let failf fmt = Printf.ksprintf fail fmt

(* Computes the 1-based line and column of [index] in [input]. *)
let position_at input index =
  let index = min index (String.length input) in
  let line = ref 1 and column = ref 1 in
  for i = 0 to index - 1 do
    if input.[i] = '\n' then (incr line; column := 1) else incr column
  done;
  { line = !line; column = !column }
