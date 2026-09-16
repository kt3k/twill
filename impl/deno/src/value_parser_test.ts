import { assertEquals } from "@std/assert";
import {
  fn,
  parseValue,
  separator,
  toCss,
  walkValue,
  word,
} from "./value_parser.ts";

Deno.test("parseValue", () => {
  assertEquals(parseValue("1px solid red"), [
    word("1px"),
    separator(" "),
    word("solid"),
    separator(" "),
    word("red"),
  ]);
  assertEquals(parseValue("calc(1px + var(--x, 2px))"), [
    fn("calc", [
      word("1px"),
      separator(" "),
      word("+"),
      separator(" "),
      fn("var", [word("--x"), separator(","), separator(" "), word("2px")]),
    ]),
  ]);
  assertEquals(parseValue("a/b,c"), [
    word("a"),
    separator("/"),
    word("b"),
    separator(","),
    word("c"),
  ]);
  assertEquals(parseValue(`"a, b" c`), [
    word(`"a, b"`),
    separator(" "),
    word("c"),
  ]);
  assertEquals(parseValue("(a)"), [fn("", [word("a")])]);
  assertEquals(parseValue("a\\,b"), [word("a\\,b")]);
});

Deno.test("toCss round trips", () => {
  for (
    const input of [
      "1px solid red",
      "calc(1px + var(--x, 2px))",
      "url(/a.png)",
      "a/b,c",
      `"a, b" c`,
      "--spacing(4)",
    ]
  ) {
    assertEquals(toCss(parseValue(input)), input);
  }
});

Deno.test("walkValue with replacement", () => {
  const ast = parseValue("calc(--spacing(4) * 2)");
  walkValue(ast, (node, { replaceWith }) => {
    if (node.kind === "function" && node.value === "--spacing") {
      replaceWith(word("calc(var(--spacing) * 4)"));
    }
  });
  assertEquals(toCss(ast), "calc(calc(var(--spacing) * 4) * 2)");
});
