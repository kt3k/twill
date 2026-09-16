import { assertEquals } from "@std/assert";
import { withAlpha } from "./color.ts";

Deno.test("withAlpha", () => {
  assertEquals(
    withAlpha("red", "0.5"),
    "color-mix(in oklab, red 50%, transparent)",
  );
  assertEquals(
    withAlpha("red", "50%"),
    "color-mix(in oklab, red 50%, transparent)",
  );
  assertEquals(withAlpha("red", "1"), "red");
  assertEquals(withAlpha("red", "100%"), "red");
  assertEquals(
    withAlpha("red", "var(--a)"),
    "color-mix(in oklab, red var(--a), transparent)",
  );
  assertEquals(
    withAlpha("red", ".25"),
    "color-mix(in oklab, red 25%, transparent)",
  );
});
