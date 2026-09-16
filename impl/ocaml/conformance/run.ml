(* Runs every conformance case through the OCaml implementation and writes
   the output to <root>/out/ocaml/<case>.css. *)

open Twill

let ( / ) = Filename.concat

let loader id base =
  match Builtin.resolve_builtin id base with
  | Some loaded -> loaded
  | None ->
      let path = if Filename.is_relative id then base / id else id in
      let content = In_channel.with_open_bin path In_channel.input_all in
      { Builtin.path; base = Filename.dirname path; content }

let () =
  let here = Sys.argv.(1) in
  let cases_dir = here / "cases" and out_dir = here / "out" / "ocaml" in
  List.iter (fun d -> if not (Sys.file_exists d) then Unix.mkdir d 0o755) [ here / "out"; out_dir ];
  let names = List.sort compare (List.filter (fun n -> Sys.is_directory (cases_dir / n)) (Array.to_list (Sys.readdir cases_dir))) in
  List.iter
    (fun name ->
      let dir = cases_dir / name in
      let css = In_channel.with_open_bin (dir / "input.css") In_channel.input_all in
      let raw = In_channel.with_open_bin (dir / "candidates.txt") In_channel.input_all in
      let candidates = List.filter (fun l -> l <> "") (String.split_on_char '\n' raw) in
      let output =
        match Compile.compile ~base:dir ~load:loader css with
        | exception (Twill_error.Error _ as e) -> "ERROR: " ^ Twill_error.message e ^ "\n"
        | compiler ->
            let output = Compile.build compiler candidates in
            let again = Compile.build compiler (List.rev candidates) in
            if again <> output then output ^ "\n/* UNSTABLE: second build differs */\n" ^ again else output
      in
      Out_channel.with_open_bin (out_dir / (name ^ ".css")) (fun oc -> output_string oc output))
    names;
  Printf.printf "ocaml: %d cases\n" (List.length names)
