(* Helpers shared by the registry tests. *)

open Twill
open Ast
open Candidate

(* Builds a design system from a stylesheet without running the full
   compile pipeline. *)
let design_system_for css =
  let css = if css = "" then "@import \"twill\";" else css in
  let th, _, state = Test_directives.import_and_collect css in
  let ds = Design_system.build th in
  ds.important <- state.Directives.important;
  ds

(* Compiles one raw candidate into nested CSS through the registries,
   without theme function substitution. *)
let compile_raw (ds : Design_system.t) raw =
  let rules = ref [] in
  List.iter
    (fun (c : candidate) ->
      let results =
        if c.ckind = Arbitrary_candidate then
          match Utilities.as_color c.arbitrary_value c.cmodifier ds.theme with
          | Some v -> [ [ decl c.property v ] ]
          | None -> []
        else begin
          let defs =
            List.filter
              (fun (d : Utilities.definition) ->
                (d.ukind = Static && c.ckind = Static_candidate) || (d.ukind = Functional && c.ckind = Functional_candidate))
              (Utilities.get ds.utilities c.croot)
          in
          let ordered = List.filter (fun d -> not (Utilities.is_fallback d)) defs @ List.filter Utilities.is_fallback defs in
          let results = ref [] in
          (try
             List.iter
               (fun (d : Utilities.definition) ->
                 match d.compile c with
                 | _, Utilities.Not_handled -> ()
                 | _, Utilities.Invalid -> if d.types <> None then raise Exit
                 | nodes, Utilities.Handled -> results := !results @ [ nodes ])
               ordered
           with Exit -> ());
          !results
        end
      in
      List.iter
        (fun nodes ->
          let rule = style_rule ("." ^ Utils.escape raw) ~nodes in
          if List.for_all (fun v -> Variants.apply_variant rule v ds.variants 0) c.variants then rules := !rules @ [ rule ])
        results)
    (Design_system.parse_candidate ds raw);
  Serializer.serialize !rules

let rec trim_newlines s = if Utils.has_suffix s "\n" then trim_newlines (String.sub s 0 (String.length s - 1)) else s

let block selector body =
  let lines = List.map (fun l -> "  " ^ l) (String.split_on_char '\n' (trim_newlines body)) in
  selector ^ " {\n" ^ String.concat "\n" lines ^ "\n}\n"

let expect_decls ds raw declarations =
  let want = "." ^ Utils.escape raw ^ " {\n  " ^ String.concat "\n  " declarations ^ "\n}\n" in
  Harness.equal raw (compile_raw ds raw) want

let expect_invalid ds raws = List.iter (fun raw -> Harness.equal (raw ^ " invalid") (compile_raw ds raw) "") raws
