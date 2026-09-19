open Twill
open Utils
open Harness

let run () =
  List.iter
    (fun (input, want) -> equal ("escape " ^ input) (escape input) want)
    [ ("hover:flex", "hover\\:flex"); ("w-1/2", "w-1\\/2"); ("bg-[#0088cc]", "bg-\\[\\#0088cc\\]"); ("2xl:p-4", "\\32 xl\\:p-4");
      ("-1", "-\\31 "); ("-", "\\-"); ("a\001b", "a\\1 b"); ("日本", "日本");
      ("data-[state=open]:flex", "data-\\[state\\=open\\]\\:flex"); ("*:flex", "\\*\\:flex") ];
  equal "unescape" (unescape "hover\\:flex") "hover:flex";
  equal "unescape hex" (unescape "\\32 xl") "2xl";
  equal "unescape slash" (unescape "foo-1\\/2") "foo-1/2";
  equal "unescape plain" (unescape "plain") "plain";
  let seg name input sep want = check name (segment input sep = want) in
  seg "segment 1" "a:b:c" ':' [ "a"; "b"; "c" ];
  seg "segment 2" "a:[b:c]:d" ':' [ "a"; "[b:c]"; "d" ];
  seg "segment 3" "a:(b:c):d" ':' [ "a"; "(b:c)"; "d" ];
  seg "segment 4" "a:{b:c}:d" ':' [ "a"; "{b:c}"; "d" ];
  seg "segment 5" "a:\"b:c\":d" ':' [ "a"; "\"b:c\""; "d" ];
  seg "segment 6" "a\\:b:c" ':' [ "a\\:b"; "c" ];
  seg "segment 7" "a:[b:(c]:d)]:e" ':' [ "a"; "[b:(c]:d)]"; "e" ];
  seg "segment 8" "" ':' [ "" ];
  seg "segment 9" "a::b" ':' [ "a"; ""; "b" ];
  List.iter
    (fun (input, want) -> check ("arbitrary " ^ input) (is_valid_arbitrary input = want))
    [ ("calc(1px + 2px)", true); ("a]", false); ("(a]", false); ("a;b", false); ("(a;b)", true); ("'a;b'", true); ("a}", false);
      ("a\\]", true) ];
  List.iter
    (fun (input, want) -> equal ("decode " ^ input) (decode_arbitrary_value input) want)
    [ ("10px_20px", "10px 20px"); ("a\\_b_c", "a_b c"); ("url(/a_b.png)", "url(/a_b.png)"); ("image_url(/a_b.png)", "image_url(/a_b.png)");
      ("var(--my_var)", "var(--my_var)"); ("var(--my_var,1px_2px)", "var(--my_var,1px 2px)"); ("var(--my\\_var)", "var(--my_var)");
      ("theme(--spacing_x)", "theme(--spacing_x)"); ("&_svg", "& svg"); ("&_svg:not(.x)", "& svg:not(.x)");
      ("&_svg:not([class*='size-'])", "& svg:not([class*='size-'])"); ("&_p:is(.a_.b)", "& p:is(.a .b)");
      ("&\\_svg:not(.x)", "&_svg:not(.x)"); ("calc(1px+2px)", "calc(1px + 2px)"); ("calc(1px_+_2px)", "calc(1px + 2px)");
      ("calc(100%-var(--x))", "calc(100% - var(--x))"); ("calc(var(--x)*2)", "calc(var(--x) * 2)"); ("calc(1px*-1)", "calc(1px * -1)");
      ("min(1px,2px)", "min(1px,2px)"); ("rgb(0_0_0_/_0.5)", "rgb(0 0 0 / 0.5)"); ("calc(-1*var(--x))", "calc(-1 * var(--x))");
      ("'a_b'", "'a b'"); ("calc(1rem-2px)", "calc(1rem - 2px)"); ("min(100%,max-content)", "min(100%,max-content)");
      ("calc(-1px)", "calc(-1px)"); ("calc(var(--a)+var(--b))", "calc(var(--a) + var(--b))");
      ("clamp(1rem,2vw+1rem,3rem)", "clamp(1rem,2vw + 1rem,3rem)"); ("calc((1px+2px)*3)", "calc((1px + 2px) * 3)");
      ("env(safe-area-inset-top)", "env(safe-area-inset-top)") ];
  check "compare"
    (compare_natural "a" "b" < 0 && compare_natural "p-2" "p-10" < 0 && compare_natural "p-10" "p-2" > 0
    && compare_natural "p-2" "p-2" = 0 && compare_natural "p-02" "p-2" < 0 && compare_natural "a" "ab" < 0
    && compare_natural "ab" "a" > 0);
  check "sort" (List.sort compare_natural [ "p-10"; "p-2"; "p-1"; "m-1" ] = [ "m-1"; "p-1"; "p-2"; "p-10" ]);
  check "positive integer" (is_positive_integer "0" && (not (is_positive_integer "012")) && not (is_positive_integer "-1"));
  check "strict positive" ((not (is_strict_positive_integer "0")) && is_strict_positive_integer "1");
  List.iter
    (fun (input, want) -> check ("quarter " ^ input) (is_multiple_of_quarter input = want))
    [ ("4", true); ("0.25", true); ("1.5", true); ("-2.75", true); ("0.3", false); ("0.50", false); ("04", false); (".5", false) ];
  List.iter
    (fun (input, want) -> check ("expand " ^ input) (expand_braces input = want))
    [ ("p-{1,2,3}", [ "p-1"; "p-2"; "p-3" ]); ("p-{1..3}", [ "p-1"; "p-2"; "p-3" ]); ("p-{3..1}", [ "p-3"; "p-2"; "p-1" ]);
      ("p-{0..20..5}", [ "p-0"; "p-5"; "p-10"; "p-15"; "p-20" ]); ("m-{-2..0}", [ "m--2"; "m--1"; "m-0" ]);
      ("{a,b}-{1,2}", [ "a-1"; "a-2"; "b-1"; "b-2" ]); ("{hover:,}flex", [ "hover:flex"; "flex" ]); ("{a,{b,c}}", [ "a"; "b"; "c" ]);
      ("plain", [ "plain" ]) ];
  List.iter
    (fun input -> expect_error ("expand error " ^ input) (fun () -> expand_braces input) "")
    [ "p-{0..4..0}"; "p-{1,2"; "p-1}" ];
  check "variant names"
    (is_valid_variant_name "dark" && is_valid_variant_name "@md" && (not (is_valid_variant_name "Foo"))
    && (not (is_valid_variant_name "foo-")) && not (is_valid_variant_name "foo_"));
  let open Value in
  check "value parse 1" (parse "1px solid red" = [ word "1px"; sep " "; word "solid"; sep " "; word "red" ]);
  check "value parse 2"
    (parse "calc(1px + var(--x, 2px))"
    = [ fn "calc" [ word "1px"; sep " "; word "+"; sep " "; fn "var" [ word "--x"; sep ","; sep " "; word "2px" ] ] ]);
  check "value parse 3" (parse "a/b,c" = [ word "a"; sep "/"; word "b"; sep ","; word "c" ]);
  check "value parse 4" (parse "\"a, b\" c" = [ word "\"a, b\""; sep " "; word "c" ]);
  check "value parse 5" (parse "(a)" = [ fn "" [ word "a" ] ]);
  List.iter
    (fun input -> equal ("value roundtrip " ^ input) (to_css (parse input)) input)
    [ "1px solid red"; "calc(1px + var(--x, 2px))"; "url(/a.png)"; "a/b,c"; "\"a, b\" c"; "--spacing(4)" ];
  let ast = ref (parse "calc(--spacing(4) * 2)") in
  walk ast (fun n _ ->
      match n with Fn f when f.fname = "--spacing" -> VReplace [ word "calc(var(--spacing) * 4)" ] | _ -> VContinue);
  equal "value walk" (to_css !ast) "calc(calc(var(--spacing) * 4) * 2)";
  equal "alpha 0.5" (Theme.with_alpha "red" "0.5") "color-mix(in oklab, red 50%, transparent)";
  equal "alpha 50%" (Theme.with_alpha "red" "50%") "color-mix(in oklab, red 50%, transparent)";
  equal "alpha 1" (Theme.with_alpha "red" "1") "red";
  equal "alpha 100%" (Theme.with_alpha "red" "100%") "red";
  equal "alpha var" (Theme.with_alpha "red" "var(--a)") "color-mix(in oklab, red var(--a), transparent)";
  equal "alpha .25" (Theme.with_alpha "red" ".25") "color-mix(in oklab, red 25%, transparent)"
