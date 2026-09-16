open Twill
open Harness

let extract text = List.sort compare (Extract.extract_candidates text)
let same name got want = equal name (String.concat " " got) (String.concat " " want)

let run () =
  same "html attributes"
    (extract "<div class=\"flex p-4 hover:underline md:grid-cols-2 bg-red-500/50\"></div>")
    [ "bg-red-500/50"; "class"; "flex"; "hover:underline"; "md:grid-cols-2"; "p-4" ];
  same "template literals"
    (extract "const c = `w-[13px] ${x} text-lg/8`; const o = { 'sm:flex': true, \"dark:bg-black\": 1 };")
    [ "c"; "const"; "dark:bg-black"; "o"; "sm:flex"; "text-lg/8"; "w-[13px]" ];
  same "arbitrary"
    (extract
       "\"[mask-type:luminance] bg-[url(/a_b.png)] [&_p]:flex has-[>img]:block data-[state=open]:flex w-(--my-w) supports-[display:grid]:grid\"")
    [ "[&_p]:flex"; "[mask-type:luminance]"; "bg-[url(/a_b.png)]"; "data-[state=open]:flex"; "has-[>img]:block";
      "supports-[display:grid]:grid"; "w-(--my-w)" ];
  same "importance and fractions"
    (extract "'underline! !flex -mt-2 @md:flex @md/main:grid w-1/2 p-1.5 opacity-50% aria-[label=foo_i]:hidden'")
    [ "!flex"; "-mt-2"; "@md/main:grid"; "@md:flex"; "aria-[label=foo_i]:hidden"; "opacity-50%"; "p-1.5"; "underline!"; "w-1/2" ];
  same "boundaries" (extract ".flex{}</div>p-4=x") [ "flex"; "p-4" ];
  same "boundaries 2" (extract "xflex yhover:flex") [ "xflex"; "yhover:flex" ];
  same "url" (extract "https://example.com/path") [ "com/path" ];
  same "trailing" (extract "foo- bar_ -@x") [];
  same "dots" (extract "a.b c.d") [ "b"; "d" ];
  same "class list" (extract "class:list={[\"flex\"]}") [ "class:list"; "flex" ];
  same "variables" (extract "\"--color-red-500\" '--spacing' --x --") [ "--color-red-500"; "--spacing"; "--x" ];
  same "var call" (extract "var(--color-red-500)") [];
  same "unbalanced" (extract "w-[13px") [];
  same "unbalanced newline" (extract "w-[a\nb]") [ "b" ];
  same "order and dedupe" (Extract.extract_candidates "b a b c") [ "b"; "a"; "c" ]
