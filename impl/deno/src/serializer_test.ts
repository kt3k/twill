import { assertEquals } from "@std/assert";
import { serialize, serializeCompact } from "./serializer.ts";
import { parse } from "./parser.ts";
import { atRoot, atRule, comment, context, decl, styleRule } from "./ast.ts";

Deno.test("serialize: format per SPEC §5.2", () => {
  const ast = [
    comment("! license"),
    atRule("@import", '"a.css"'),
    styleRule(".a", [
      decl("color", "red"),
      decl("width", "1px", true),
      decl("hidden", undefined),
      atRule("@media", "(hover: hover)", [
        styleRule("&:hover", [decl("color", "blue")]),
      ]),
      atRule("@starting-style", "", [decl("opacity", "0")]),
    ]),
  ];
  assertEquals(
    serialize(ast),
    `/*! license*/
@import "a.css";
.a {
  color: red;
  width: 1px !important;
  @media (hover: hover) {
    &:hover {
      color: blue;
    }
  }
  @starting-style {
    opacity: 0;
  }
}
`,
  );
});

Deno.test("serialize: context and at-root children print at the same depth", () => {
  const ast = [
    context({ base: "/x" }, [styleRule(".a", [decl("color", "red")])]),
    atRoot([styleRule(".b", [decl("color", "blue")])]),
  ];
  assertEquals(
    serialize(ast),
    `.a {\n  color: red;\n}\n.b {\n  color: blue;\n}\n`,
  );
});

Deno.test("serialize: round trip", () => {
  const css = `@layer theme, base;
:root, :host {
  --spacing: 0.25rem;
}
@media (width >= 40rem) {
  .sm\\:flex {
    display: flex;
  }
}
`;
  assertEquals(serialize(parse(css)), css);
});

Deno.test("serializeCompact", () => {
  const ast = parse(
    `.a { color: red !important; @media (x) { &:hover { y: z } } }\n@import "a";`,
  );
  assertEquals(
    serializeCompact(ast),
    `.a{color:red!important;@media (x){&:hover{y:z;}}}@import "a";`,
  );
});
