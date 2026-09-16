import { assertEquals, assertThrows } from "@std/assert";
import {
  compare,
  decodeArbitraryValue,
  escape,
  expandBraces,
  isMultipleOfQuarter,
  isPositiveInteger,
  isStrictPositiveInteger,
  isValidArbitrary,
  isValidVariantName,
  segment,
  unescape,
} from "./utils.ts";
import { TwillError } from "./error.ts";

Deno.test("escape matches CSS.escape", () => {
  assertEquals(escape("hover:flex"), "hover\\:flex");
  assertEquals(escape("w-1/2"), "w-1\\/2");
  assertEquals(escape("bg-[#0088cc]"), "bg-\\[\\#0088cc\\]");
  assertEquals(escape("2xl:p-4"), "\\32 xl\\:p-4");
  assertEquals(escape("-1"), "-\\31 ");
  assertEquals(escape("-"), "\\-");
  assertEquals(escape(`a${String.fromCharCode(1)}b`), "a\\1 b");
  assertEquals(escape("日本"), "日本");
  assertEquals(
    escape("data-[state=open]:flex"),
    "data-\\[state\\=open\\]\\:flex",
  );
  assertEquals(escape("*:flex"), "\\*\\:flex");
});

Deno.test("unescape", () => {
  assertEquals(unescape("hover\\:flex"), "hover:flex");
  assertEquals(unescape("\\32 xl"), "2xl");
  assertEquals(unescape("foo-1\\/2"), "foo-1/2");
  assertEquals(unescape("plain"), "plain");
});

Deno.test("segment", () => {
  assertEquals(segment("a:b:c", ":"), ["a", "b", "c"]);
  assertEquals(segment("a:[b:c]:d", ":"), ["a", "[b:c]", "d"]);
  assertEquals(segment("a:(b:c):d", ":"), ["a", "(b:c)", "d"]);
  assertEquals(segment("a:{b:c}:d", ":"), ["a", "{b:c}", "d"]);
  assertEquals(segment(`a:"b:c":d`, ":"), ["a", `"b:c"`, "d"]);
  assertEquals(segment("a\\:b:c", ":"), ["a\\:b", "c"]);
  assertEquals(segment("a:[b:(c]:d)]:e", ":"), ["a", "[b:(c]:d)]", "e"]);
  assertEquals(segment("", ":"), [""]);
  assertEquals(segment("a::b", ":"), ["a", "", "b"]);
});

Deno.test("isValidArbitrary", () => {
  assertEquals(isValidArbitrary("calc(1px + 2px)"), true);
  assertEquals(isValidArbitrary("a]"), false);
  assertEquals(isValidArbitrary("(a]"), false);
  assertEquals(isValidArbitrary("a;b"), false);
  assertEquals(isValidArbitrary("(a;b)"), true);
  assertEquals(isValidArbitrary("'a;b'"), true);
  assertEquals(isValidArbitrary("a}"), false);
  assertEquals(isValidArbitrary("a\\]"), true);
});

Deno.test("decodeArbitraryValue", () => {
  assertEquals(decodeArbitraryValue("10px_20px"), "10px 20px");
  assertEquals(decodeArbitraryValue("a\\_b_c"), "a_b c");
  assertEquals(decodeArbitraryValue("url(/a_b.png)"), "url(/a_b.png)");
  assertEquals(
    decodeArbitraryValue("image_url(/a_b.png)"),
    "image_url(/a_b.png)",
  );
  assertEquals(decodeArbitraryValue("var(--my_var)"), "var(--my_var)");
  assertEquals(
    decodeArbitraryValue("var(--my_var,1px_2px)"),
    "var(--my_var,1px 2px)",
  );
  assertEquals(decodeArbitraryValue("var(--my\\_var)"), "var(--my_var)");
  assertEquals(
    decodeArbitraryValue("theme(--spacing_x)"),
    "theme(--spacing_x)",
  );
  assertEquals(decodeArbitraryValue("calc(1px+2px)"), "calc(1px + 2px)");
  assertEquals(decodeArbitraryValue("calc(1px_+_2px)"), "calc(1px + 2px)");
  assertEquals(
    decodeArbitraryValue("calc(100%-var(--x))"),
    "calc(100% - var(--x))",
  );
  assertEquals(decodeArbitraryValue("calc(var(--x)*2)"), "calc(var(--x) * 2)");
  assertEquals(decodeArbitraryValue("calc(1px*-1)"), "calc(1px * -1)");
  assertEquals(decodeArbitraryValue("min(1px,2px)"), "min(1px,2px)");
  assertEquals(decodeArbitraryValue("rgb(0_0_0_/_0.5)"), "rgb(0 0 0 / 0.5)");
  assertEquals(
    decodeArbitraryValue("calc(-1*var(--x))"),
    "calc(-1 * var(--x))",
  );
  assertEquals(decodeArbitraryValue("'a_b'"), "'a b'");
  assertEquals(decodeArbitraryValue("calc(1rem-2px)"), "calc(1rem - 2px)");
  assertEquals(
    decodeArbitraryValue("min(100%,max-content)"),
    "min(100%,max-content)",
  );
  assertEquals(decodeArbitraryValue("calc(-1px)"), "calc(-1px)");
  assertEquals(decodeArbitraryValue("calc(1px_*_2)"), "calc(1px * 2)");
  assertEquals(
    decodeArbitraryValue("calc(var(--a)+var(--b))"),
    "calc(var(--a) + var(--b))",
  );
  assertEquals(
    decodeArbitraryValue("clamp(1rem,2vw+1rem,3rem)"),
    "clamp(1rem,2vw + 1rem,3rem)",
  );
  assertEquals(
    decodeArbitraryValue("calc((1px+2px)*3)"),
    "calc((1px + 2px) * 3)",
  );
  assertEquals(
    decodeArbitraryValue("env(safe-area-inset-top)"),
    "env(safe-area-inset-top)",
  );
});

Deno.test("compare: natural ordering", () => {
  assertEquals(compare("a", "b") < 0, true);
  assertEquals(compare("p-2", "p-10") < 0, true);
  assertEquals(compare("p-10", "p-2") > 0, true);
  assertEquals(compare("p-2", "p-2"), 0);
  assertEquals(compare("p-02", "p-2") < 0, true);
  assertEquals(compare("a", "ab") < 0, true);
  assertEquals(compare("ab", "a") > 0, true);
  assertEquals(
    ["p-10", "p-2", "p-1", "m-1"].sort(compare),
    ["m-1", "p-1", "p-2", "p-10"],
  );
});

Deno.test("numeric predicates", () => {
  assertEquals(isPositiveInteger("0"), true);
  assertEquals(isPositiveInteger("12"), true);
  assertEquals(isPositiveInteger("012"), false);
  assertEquals(isPositiveInteger("-1"), false);
  assertEquals(isPositiveInteger("1.5"), false);
  assertEquals(isStrictPositiveInteger("0"), false);
  assertEquals(isStrictPositiveInteger("1"), true);
  assertEquals(isMultipleOfQuarter("4"), true);
  assertEquals(isMultipleOfQuarter("0.25"), true);
  assertEquals(isMultipleOfQuarter("1.5"), true);
  assertEquals(isMultipleOfQuarter("-2.75"), true);
  assertEquals(isMultipleOfQuarter("0.3"), false);
  assertEquals(isMultipleOfQuarter("0.50"), false);
  assertEquals(isMultipleOfQuarter("04"), false);
  assertEquals(isMultipleOfQuarter(".5"), false);
});

Deno.test("expandBraces", () => {
  assertEquals(expandBraces("p-{1,2,3}"), ["p-1", "p-2", "p-3"]);
  assertEquals(expandBraces("p-{1..3}"), ["p-1", "p-2", "p-3"]);
  assertEquals(expandBraces("p-{3..1}"), ["p-3", "p-2", "p-1"]);
  assertEquals(expandBraces("p-{0..20..5}"), [
    "p-0",
    "p-5",
    "p-10",
    "p-15",
    "p-20",
  ]);
  assertEquals(expandBraces("m-{-2..0}"), ["m--2", "m--1", "m-0"]);
  assertEquals(expandBraces("{a,b}-{1,2}"), ["a-1", "a-2", "b-1", "b-2"]);
  assertEquals(expandBraces("{hover:,}flex"), ["hover:flex", "flex"]);
  assertEquals(expandBraces("{a,{b,c}}"), ["a", "b", "c"]);
  assertEquals(expandBraces("plain"), ["plain"]);
  assertThrows(() => expandBraces("p-{0..4..0}"), TwillError);
  assertThrows(() => expandBraces("p-{1,2"), TwillError);
  assertThrows(() => expandBraces("p-1}"), TwillError);
});

Deno.test("isValidVariantName", () => {
  assertEquals(isValidVariantName("dark"), true);
  assertEquals(isValidVariantName("@md"), true);
  assertEquals(isValidVariantName("foo-bar_1"), true);
  assertEquals(isValidVariantName("Foo"), false);
  assertEquals(isValidVariantName("foo-"), false);
  assertEquals(isValidVariantName("foo_"), false);
  assertEquals(isValidVariantName("-foo"), false);
});
