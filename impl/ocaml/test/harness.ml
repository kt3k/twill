(* A tiny test harness: every check records a failure instead of aborting. *)

let failures = ref 0
let count = ref 0

let fail name detail =
  incr failures;
  Printf.printf "FAIL %s\n%s\n" name detail

let check name cond =
  incr count;
  if not cond then fail name ""

let equal ?(show = fun s -> s) name got want =
  incr count;
  if got <> want then fail name (Printf.sprintf " got: %S\nwant: %S" (show got) (show want))

let equal_str name got want = equal name got want
let includes name output part =
  incr count;
  let n = String.length output and m = String.length part in
  let rec go i = i + m <= n && (String.sub output i m = part || go (i + 1)) in
  if not (go 0) then fail name (Printf.sprintf "expected to contain %S in:\n%s" part output)

let excludes name output part =
  incr count;
  let n = String.length output and m = String.length part in
  let rec go i = i + m <= n && (String.sub output i m = part || go (i + 1)) in
  if go 0 then fail name (Printf.sprintf "expected not to contain %S in:\n%s" part output)

let expect_error name f substring =
  incr count;
  match f () with
  | exception (Twill.Twill_error.Error _ as e) ->
      let msg = Twill.Twill_error.message e in
      let n = String.length msg and m = String.length substring in
      let rec go i = i + m <= n && (String.sub msg i m = substring || go (i + 1)) in
      if not (go 0) then fail name (Printf.sprintf "expected error containing %S, got %S" substring msg)
  | exception e -> fail name ("unexpected exception " ^ Printexc.to_string e)
  | _ -> fail name (Printf.sprintf "expected an error containing %S" substring)

let finish () =
  Printf.printf "%d checks, %d failures\n" !count !failures;
  if !failures > 0 then exit 1
