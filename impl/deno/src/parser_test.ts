import { assertEquals, assertThrows } from "@std/assert";
import { parse } from "./parser.ts";
import { atRule, comment, decl, styleRule } from "./ast.ts";
import { TwillError } from "./error.ts";

Deno.test("parse: declarations and rules", () => {
  assertEquals(
    parse(`.a { color: red; background: blue }`),
    [styleRule(".a", [decl("color", "red"), decl("background", "blue")])],
  );
});

Deno.test("parse: nested rules, nested at-rules, and &", () => {
  assertEquals(
    parse(`.a {
      color: red;
      &:hover { color: blue; }
      @media (hover: hover) {
        .b & { color: green; }
      }
    }`),
    [
      styleRule(".a", [
        decl("color", "red"),
        styleRule("&:hover", [decl("color", "blue")]),
        atRule("@media", "(hover: hover)", [
          styleRule(".b &", [decl("color", "green")]),
        ]),
      ]),
    ],
  );
});

Deno.test("parse: statement at-rules have no children", () => {
  assertEquals(
    parse(
      `@import "foo.css" layer(base);\n@charset "utf-8";\n@twill utilities;`,
    ),
    [
      atRule("@import", '"foo.css" layer(base)'),
      atRule("@charset", '"utf-8"'),
      atRule("@twill", "utilities"),
    ],
  );
});

Deno.test("parse: at-rule directly followed by parenthesis", () => {
  assertEquals(parse(`@media(width>=1px){.a{x:y}}`), [
    atRule("@media", "(width>=1px)", [styleRule(".a", [decl("x", "y")])]),
  ]);
});

Deno.test("parse: ordinary comments are dropped, /*! comments are kept", () => {
  assertEquals(
    parse(`/* license? no */\n/*! license */\n.a { /* inner */ color: red; }`),
    [comment("! license "), styleRule(".a", [decl("color", "red")])],
  );
});

Deno.test("parse: !important is split from values", () => {
  assertEquals(parse(`.a { color: red !important; width: 1px!IMPORTANT }`), [
    styleRule(".a", [decl("color", "red", true), decl("width", "1px", true)]),
  ]);
});

Deno.test("parse: custom properties", () => {
  assertEquals(
    parse(`:root { --a: 1px; --b:; --c: { foo: bar; }; --d: calc(1px + 2px) }`),
    [
      styleRule(":root", [
        decl("--a", "1px"),
        decl("--b", ""),
        decl("--c", "{ foo: bar; }"),
        decl("--d", "calc(1px + 2px)"),
      ]),
    ],
  );
});

Deno.test("parse: delimiters inside quotes and parentheses are literal", () => {
  assertEquals(
    parse(
      `.a { content: "a;b{c}"; background: url(x;y{z}); font-family: 'q}q' }`,
    ),
    [
      styleRule(".a", [
        decl("content", '"a;b{c}"'),
        decl("background", "url(x;y{z})"),
        decl("font-family", "'q}q'"),
      ]),
    ],
  );
});

Deno.test("parse: escaped characters are preserved", () => {
  assertEquals(parse(`.a\\:b { color: red }`), [
    styleRule(".a\\:b", [decl("color", "red")]),
  ]);
  assertEquals(parse(`.a { content: "\\"" }`), [
    styleRule(".a", [decl("content", '"\\""')]),
  ]);
});

Deno.test("parse: missing trailing ; is accepted", () => {
  assertEquals(parse(`.a { color: red }`), [
    styleRule(".a", [decl("color", "red")]),
  ]);
  assertEquals(parse(`@import "a"`), [atRule("@import", '"a"')]);
});

Deno.test("parse: CRLF line endings", () => {
  assertEquals(parse(".a {\r\n  color: red;\r\n}\r\n"), [
    styleRule(".a", [decl("color", "red")]),
  ]);
});

Deno.test("parse: syntax errors carry positions", () => {
  const err = assertThrows(() => parse(".a {\n  color: red;\n"), TwillError);
  assertEquals(err.position, { line: 3, column: 1 });

  const err2 = assertThrows(() => parse(".a { color: red; }\n}"), TwillError);
  assertEquals(err2.position, { line: 2, column: 1 });

  assertThrows(
    () => parse(".a { color: 'red }"),
    TwillError,
    "Unterminated string",
  );
  assertThrows(
    () => parse(".a { color: red; /* x"),
    TwillError,
    "Unterminated comment",
  );
  assertThrows(() => parse(".a { color: calc(1px; }"), TwillError);
  assertThrows(() => parse(".a { color }"), TwillError, "Invalid declaration");
});

Deno.test("parse: empty input", () => {
  assertEquals(parse(""), []);
  assertEquals(parse("  \n  "), []);
});
