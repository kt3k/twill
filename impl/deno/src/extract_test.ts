import { assertEquals } from "@std/assert";
import { extractCandidates } from "./extract.ts";

function extract(text: string): string[] {
  return [...extractCandidates(text)].sort();
}

Deno.test("extract: HTML attributes", () => {
  assertEquals(
    extract(
      `<div class="flex p-4 hover:underline md:grid-cols-2 bg-red-500/50"></div>`,
    ),
    [
      "bg-red-500/50",
      "class",
      "flex",
      "hover:underline",
      "md:grid-cols-2",
      "p-4",
    ],
  );
});

Deno.test("extract: template literals and object keys", () => {
  assertEquals(
    extract(
      "const c = `w-[13px] ${x} text-lg/8`; const o = { 'sm:flex': true, \"dark:bg-black\": 1 };",
    ),
    [
      "c",
      "const",
      "dark:bg-black",
      "o",
      "sm:flex",
      "text-lg/8",
      "w-[13px]",
    ],
  );
});

Deno.test("extract: arbitrary values, properties, and variants", () => {
  assertEquals(
    extract(
      `"[mask-type:luminance] bg-[url(/a_b.png)] [&_p]:flex has-[>img]:block data-[state=open]:flex w-(--my-w) supports-[display:grid]:grid"`,
    ),
    [
      "[&_p]:flex",
      "[mask-type:luminance]",
      "bg-[url(/a_b.png)]",
      "data-[state=open]:flex",
      "has-[>img]:block",
      "supports-[display:grid]:grid",
      "w-(--my-w)",
    ],
  );
});

Deno.test("extract: importance, negatives, containers, fractions", () => {
  assertEquals(
    extract(
      `'underline! !flex -mt-2 @md:flex @md/main:grid w-1/2 p-1.5 opacity-50% aria-[label=foo_i]:hidden'`,
    ),
    [
      "!flex",
      "-mt-2",
      "@md/main:grid",
      "@md:flex",
      "aria-[label=foo_i]:hidden",
      "opacity-50%",
      "p-1.5",
      "underline!",
      "w-1/2",
    ],
  );
});

Deno.test("extract: boundaries", () => {
  // `.` and `}` and `>` are start boundaries; `=`, `<`, `{`, `]` end.
  assertEquals(extract(`.flex{}</div>p-4=x`), ["flex", "p-4"]);
  // Substrings are never emitted.
  assertEquals(extract(`xflex yhover:flex`), ["xflex", "yhover:flex"]);
  // `.` starts a span, so a harmless `com/path` survives; `https` and
  // `example` are followed by non-boundary characters and do not.
  assertEquals(extract(`https://example.com/path`), ["com/path"]);
  assertEquals(extract(`foo- bar_ -@x`), []);
  // `.` is not an end boundary, so `a` and `c` are not spans.
  assertEquals(extract(`a.b c.d`), ["b", "d"]);
  assertEquals(extract(`class:list={["flex"]}`), ["class:list", "flex"]);
});

Deno.test("extract: custom property references", () => {
  assertEquals(extract(`"--color-red-500" '--spacing' --x --`), [
    "--color-red-500",
    "--spacing",
    "--x",
  ]);
  // `(` is neither a start nor an end boundary.
  assertEquals(extract(`var(--color-red-500)`), []);
});

Deno.test("extract: unbalanced brackets are not candidates", () => {
  assertEquals(extract(`w-[13px`), []);
  assertEquals(extract(`w-[a\nb]`), ["b"]);
});
