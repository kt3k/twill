(* Command twill is the Twill utility-first CSS compiler (SPEC §14).

   twill -i input.css -o output.css --watch *)

open Twill

let stdin_eof () =
  (* Drain whatever is readable without blocking; end of file is a zero-length read. *)
  let buf = Bytes.create 4096 in
  let rec drain () =
    match Unix.select [ Unix.stdin ] [] [] 0.0 with
    | [], _, _ -> false
    | _ -> ( match Unix.read Unix.stdin buf 0 (Bytes.length buf) with 0 -> true | _ -> drain () | exception Unix.Unix_error _ -> true)
    | exception Unix.Unix_error _ -> true
  in
  drain ()

let () =
  let io =
    { Cli.read_stdin = (fun () -> In_channel.input_all stdin);
      is_tty = Unix.isatty Unix.stdin;
      stdin_eof;
      out = (fun s -> print_string s; flush stdout);
      err = (fun s -> prerr_string s; flush stderr);
      executable = Sys.executable_name;
      stop = (fun () -> false) }
  in
  exit (Cli.main (List.tl (Array.to_list Sys.argv)) io)
