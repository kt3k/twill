import { assertEquals } from "@std/assert";
import { replaceShadowColors } from "./shadows.ts";

Deno.test("replaceShadowColors", () => {
  const wrap = (c: string) => `<${c}>`;
  assertEquals(
    replaceShadowColors("0 0 3px red, inset 0 1px", wrap),
    "0 0 3px <red>, inset 0 1px <currentcolor>",
  );
  assertEquals(replaceShadowColors("none", wrap), "none");
  assertEquals(replaceShadowColors("var(--x)", wrap), "var(--x)");
  assertEquals(
    replaceShadowColors("0 2px 4px rgb(0 0 0 / 0.1)", wrap, true),
    "inset 0 2px 4px <rgb(0 0 0 / 0.1)>",
  );
  assertEquals(
    replaceShadowColors("inset 0 2px 4px #000", wrap, true),
    "inset 0 2px 4px <#000>",
  );
  assertEquals(
    replaceShadowColors("0px  1px\n  0px #000", wrap),
    "0px 1px 0px <#000>",
  );
});
