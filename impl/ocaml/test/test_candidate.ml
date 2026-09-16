open Twill
open Candidate
open Harness

(* A stand-in for the registries of §9 and §10. *)
let fake_never = 0 and fake_at_rules = 1 and fake_style_rules = 2

let fake_variants =
  [ ("*", (Static_variant, fake_never, fake_never)); ("hover", (Static_variant, fake_style_rules, fake_never));
    ("first", (Static_variant, fake_style_rules, fake_never)); ("dark", (Static_variant, fake_at_rules, fake_never));
    ("sm", (Static_variant, fake_at_rules, fake_never)); ("first-letter", (Static_variant, fake_never, fake_never));
    ("data", (Functional_variant, fake_style_rules, fake_never)); ("aria", (Functional_variant, fake_style_rules, fake_never));
    ("nth", (Functional_variant, fake_style_rules, fake_never)); ("nth-last", (Functional_variant, fake_style_rules, fake_never));
    ("min", (Functional_variant, fake_at_rules, fake_never)); ("max", (Functional_variant, fake_at_rules, fake_never));
    ("supports", (Functional_variant, fake_at_rules, fake_never)); ("@", (Functional_variant, fake_at_rules, fake_never));
    ("@min", (Functional_variant, fake_at_rules, fake_never));
    ("not", (Compound_variant, fake_style_rules lor fake_at_rules, fake_style_rules lor fake_at_rules));
    ("group", (Compound_variant, fake_style_rules, fake_style_rules)); ("peer", (Compound_variant, fake_style_rules, fake_style_rules));
    ("has", (Compound_variant, fake_style_rules, fake_style_rules)); ("in", (Compound_variant, fake_style_rules, fake_style_rules)) ]

let fake_context prefix =
  let statics = [ "flex"; "block"; "underline"; "border"; "rounded-full"; "-mt-px" ] in
  let fns = [ "w"; "bg"; "p"; "mt"; "-mt"; "border"; "border-t"; "text"; "z"; "-z"; "grid-cols"; "tab"; "@" ] in
  let cache = Hashtbl.create 16 in
  let compounds_of v =
    match v.kind with
    | Arbitrary_variant ->
        let s = v.selector in
        if String.length s > 0 && s.[0] = '@' then
          if List.exists (fun p -> Utils.has_prefix s p) [ "@media"; "@supports"; "@container" ] then fake_at_rules else fake_never
        else if Utils.contains s "::" then fake_never
        else fake_style_rules
    | _ -> ( match List.assoc_opt v.root fake_variants with Some (_, c, _) -> c | None -> fake_never)
  in
  let rec ctx =
    { theme_prefix = (fun () -> prefix);
      has_utility = (fun name kind -> if kind = Static then List.mem name statics else List.mem name fns);
      has_variant = (fun name -> List.mem_assoc name fake_variants);
      variant_kind_of = (fun name -> Option.map (fun (k, _, _) -> k) (List.assoc_opt name fake_variants));
      compounds_with =
        (fun parent child ->
          match List.assoc_opt parent fake_variants with
          | Some (Compound_variant, _, cw) ->
              let c = compounds_of child in
              c <> fake_never && cw <> fake_never && c land cw <> 0
          | _ -> false);
      parse_variant =
        (fun input ->
          match Hashtbl.find_opt cache input with
          | Some v -> v
          | None ->
              let v = parse_variant input ctx in
              Hashtbl.replace cache input v;
              v) }
  in
  ctx

let fake = fake_context ""
let candidates input = parse_candidate input fake
let strip v = { v with vraw = "" }
let rec strip_variant v = { v with vraw = ""; inner = Option.map strip_variant v.inner }
let strip_candidate c = { c with variants = List.map strip_variant c.variants }
let cands input = List.map strip_candidate (candidates input)
let variant input = Option.map strip_variant (parse_variant input fake)

let run () =
  check "flex" (cands "flex" = [ make_candidate ~croot:"flex" Static_candidate "flex" ]);
  check "w-4" (cands "w-4" = [ make_candidate ~croot:"w" ~cvalue:(Some (named_value "4")) Functional_candidate "w-4" ]);
  check "border"
    (cands "border" = [ make_candidate ~croot:"border" Static_candidate "border"; make_candidate ~croot:"border" Functional_candidate "border" ]);
  check "border-t-2"
    (cands "border-t-2"
    = [ make_candidate ~croot:"border-t" ~cvalue:(Some (named_value "2")) Functional_candidate "border-t-2";
        make_candidate ~croot:"border" ~cvalue:(Some (named_value "t-2")) Functional_candidate "border-t-2" ]);
  check "-mt-2" (cands "-mt-2" = [ make_candidate ~croot:"-mt" ~cvalue:(Some (named_value "2")) Functional_candidate "-mt-2" ]);
  check "unknown" (cands "unknown" = []);
  check "w-a$b" (cands "w-a$b" = []);
  check "w-" (cands "w-" = []);
  check "@container" ((List.hd (cands "@container")).croot = "@");
  check "w-1/2"
    (cands "w-1/2"
    = [ make_candidate ~croot:"w" ~cvalue:(Some (named_value ~fraction:(Some "1/2") "1")) ~cmodifier:(Some { mkind = Named; mvalue = "2" })
          Functional_candidate "w-1/2" ]);
  check "modifier arbitrary" ((List.hd (cands "bg-red-500/[0.5]")).cmodifier = Some { mkind = Arbitrary; mvalue = "0.5" });
  check "modifier value" ((List.hd (cands "bg-red-500/[0.5]")).cvalue = Some (named_value "red-500"));
  check "modifier var" ((List.hd (cands "bg-red-500/(--alpha)")).cmodifier = Some { mkind = Arbitrary; mvalue = "var(--alpha)" });
  List.iter (fun i -> check ("invalid " ^ i) (cands i = [])) [ "bg-red-500/50/50"; "bg-red-500/[]"; "bg-red-500/(foo)"; "bg-red-500/a b" ];
  check "w-[13px]" ((List.hd (cands "w-[13px]")).cvalue = Some (arbitrary_value "13px"));
  check "bg-[length:...]" ((List.hd (cands "bg-[length:10px_20px]")).cvalue = Some (arbitrary_value ~data_type:(Some "length") "10px 20px"));
  check "bg-[url]" ((List.hd (cands "bg-[url(/a_b.png)]")).cvalue = Some (arbitrary_value "url(/a_b.png)"));
  check "bg-[#0088cc]/50" ((List.hd (cands "bg-[#0088cc]/50")).cmodifier = Some { mkind = Named; mvalue = "50" });
  List.iter
    (fun i -> check ("invalid arbitrary " ^ i) (cands i = []))
    [ "w-[]"; "w-[_]"; "w-[:1px]"; "w-[1px;]"; "w-[1px}]"; "w-[1px"; "unknown-[1px]"; "w-(my-w)"; "w-(--a:--b:--c)"; "unknown-(--x)" ];
  check "w-['a;b']" ((List.hd (cands "w-['a;b']")).ckind = Functional_candidate);
  check "w-[calc(1px;)]" ((Option.get (List.hd (cands "w-[calc(1px;)]")).cvalue).value = "calc(1px;)");
  check "w-(--my-w)" ((List.hd (cands "w-(--my-w)")).cvalue = Some (arbitrary_value "var(--my-w)"));
  check "bg-(color:--my-color)" ((List.hd (cands "bg-(color:--my-color)")).cvalue = Some (arbitrary_value ~data_type:(Some "color") "var(--my-color)"));
  check "[mask-type:luminance]"
    (cands "[mask-type:luminance]"
    = [ make_candidate ~property:"mask-type" ~arbitrary_value:"luminance" Arbitrary_candidate "[mask-type:luminance]" ]);
  check "[--my-var:1px]" ((List.hd (cands "[--my-var:1px]")).property = "--my-var");
  check "[color:red]/50" ((List.hd (cands "[color:red]/50")).cmodifier = Some { mkind = Named; mvalue = "50" });
  check "[color:a_b]" ((List.hd (cands "[color:a_b]")).arbitrary_value = "a b");
  List.iter (fun i -> check ("invalid property " ^ i) (cands i = [])) [ "[Color:red]"; "[color]"; "[:red]"; "[color:]"; "[color:red"; "[color:red;]" ];
  check "underline!" (List.hd (cands "underline!")).important;
  check "!underline" (List.hd (cands "!underline")).important;
  let c = List.hd (cands "hover:w-4!") in
  check "hover:w-4!" (c.important && c.variants = [ make_variant Static_variant "hover" ]);
  check "!underline!" (cands "!underline!" = []);
  check "dark:hover:flex"
    ((List.hd (cands "dark:hover:flex")).variants = [ make_variant Static_variant "hover"; make_variant Static_variant "dark" ]);
  check "unknown:flex" (cands "unknown:flex" = []);
  check "hover:unknown" (cands "hover:unknown" = []);
  let prefixed = fake_context "tw" in
  check "prefix flex" (parse_candidate "flex" prefixed = []);
  check "prefix foo:flex" (parse_candidate "foo:flex" prefixed = []);
  check "prefix tw:flex" (List.length (parse_candidate "tw:flex" prefixed) = 1);
  check "prefix tw:hover:flex"
    (List.map strip_variant (List.hd (parse_candidate "tw:hover:flex" prefixed)).variants = [ make_variant Static_variant "hover" ]);
  check "modifier 50" (parse_modifier "50" = Some { mkind = Named; mvalue = "50" });
  check "modifier [a_b]" (parse_modifier "[a_b]" = Some { mkind = Arbitrary; mvalue = "a b" });
  check "modifier (--x)" (parse_modifier "(--x)" = Some { mkind = Arbitrary; mvalue = "var(--x)" });
  List.iter (fun i -> check ("modifier invalid " ^ i) (parse_modifier i = None)) [ "[]"; "(x)"; "a b" ];
  let exists r = r = "border" || r = "border-t" || r = "@" in
  check "roots border" (find_roots "border" exists = [ { rroot = "border"; rvalue = None } ]);
  check "roots border-t-2"
    (find_roots "border-t-2" exists = [ { rroot = "border-t"; rvalue = Some "2" }; { rroot = "border"; rvalue = Some "t-2" } ]);
  check "roots border-" (find_roots "border-" exists = []);
  check "roots border-t-" (find_roots "border-t-" exists = []);
  check "roots @md" (find_roots "@md" exists = [ { rroot = "@"; rvalue = Some "md" } ]);
  check "roots @min-md" (find_roots "@min-md" exists = [ { rroot = "@"; rvalue = Some "min-md" } ]);
  check "roots nope" (find_roots "nope-1" exists = []);
  check "variant hover" (variant "hover" = Some (make_variant Static_variant "hover"));
  check "variant *" (variant "*" = Some (make_variant Static_variant "*"));
  List.iter
    (fun i -> check ("variant invalid " ^ i) (variant i = None))
    [ "hover-x"; "hover/x"; "unknown"; "data-[]"; "data-(x)"; "data-a$b"; "data-x/[]"; "group-sm"; "group-*"; "group-first-letter";
      "group"; "group-unknown"; "[@media(width>=100px)_&]"; "[]"; "[_]"; "[a;b]" ];
  let fv ?(vmodifier = None) root kind value = make_variant ~vvalue:(Some { vvkind = kind; vvalue = value }) ~vmodifier Functional_variant root in
  check "data-visible" (variant "data-visible" = Some (fv "data" Named "visible"));
  check "data-[state=open]" (variant "data-[state=open]" = Some (fv "data" Arbitrary "state=open"));
  check "supports-(--x)" (variant "supports-(--x)" = Some (fv "supports" Arbitrary "var(--x)"));
  check "nth-last-3" (variant "nth-last-3" = Some (fv "nth-last" Named "3"));
  check "min-[600px]" (variant "min-[600px]" = Some (fv "min" Arbitrary "600px"));
  check "@md/main" (variant "@md/main" = Some (fv ~vmodifier:(Some { mkind = Named; mvalue = "main" }) "@" Named "md"));
  check "@min-md" (variant "@min-md" = Some (fv "@min" Named "md"));
  check "data" (variant "data" = Some (make_variant Functional_variant "data"));
  let cv ?(vmodifier = None) root inner = make_variant ~vmodifier ~inner:(Some inner) Compound_variant root in
  check "group-hover" (variant "group-hover" = Some (cv "group" (make_variant Static_variant "hover")));
  check "group-hover/item"
    (variant "group-hover/item" = Some (cv ~vmodifier:(Some { mkind = Named; mvalue = "item" }) "group" (make_variant Static_variant "hover")));
  check "not-sm" (variant "not-sm" = Some (cv "not" (make_variant Static_variant "sm")));
  check "in-data-visible" (variant "in-data-visible" = Some (cv "in" (fv "data" Named "visible")));
  check "has-[>img]" (variant "has-[>img]" = Some (cv "has" (make_variant ~selector:">img" ~relative:true Arbitrary_variant "")));
  check "not-group-hover/item"
    (variant "not-group-hover/item"
    = Some (cv "not" (cv ~vmodifier:(Some { mkind = Named; mvalue = "item" }) "group" (make_variant Static_variant "hover"))));
  check "[&_p]" (variant "[&_p]" = Some (make_variant ~selector:"& p" Arbitrary_variant ""));
  check "[p]" (variant "[p]" = Some (make_variant ~selector:"&:is(p)" Arbitrary_variant ""));
  check "[>img]" (variant "[>img]" = Some (make_variant ~selector:">img" ~relative:true Arbitrary_variant ""));
  check "[@media(width>=100px)]" (variant "[@media(width>=100px)]" = Some (make_variant ~selector:"@media(width>=100px)" Arbitrary_variant ""));
  check "memoization" (fake.parse_variant "hover" == fake.parse_variant "hover")
