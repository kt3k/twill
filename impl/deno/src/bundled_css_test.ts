import { assertEquals } from "@std/assert";
import { BUNDLED_CSS } from "./bundled_css.ts";
import { parse } from "./parser.ts";

Deno.test("bundled_css.ts is up to date with css/", async () => {
  for (const name of Object.keys(BUNDLED_CSS)) {
    const text = await Deno.readTextFile(
      new URL(`../css/${name}`, import.meta.url),
    );
    assertEquals(
      BUNDLED_CSS[name],
      text,
      `${name} is stale; run \`deno task gen\``,
    );
  }
});

Deno.test("built-in stylesheets parse", () => {
  assertEquals(Object.keys(BUNDLED_CSS), [
    "index.css",
    "theme.css",
    "preflight.css",
    "utilities.css",
  ]);
  for (const text of Object.values(BUNDLED_CSS)) {
    parse(text);
  }
});
