import { assertEquals } from "@std/assert";
import { designSystemFor } from "./testing.ts";
import { Compounds, compoundsForSelectors, Variants } from "./variants.ts";

Deno.test("DesignSystem: memoized parsing", async () => {
  const ds = await designSystemFor();
  assertEquals(ds.parseCandidate("flex") === ds.parseCandidate("flex"), true);
  // `flex` is both the static display utility and the functional `flex-*` root.
  assertEquals(ds.parseCandidate("flex").length, 2);
  assertEquals(ds.parseCandidate("block").length, 1);
  assertEquals(ds.parseCandidate("nope").length, 0);
  assertEquals(ds.parseVariant("hover") === ds.parseVariant("hover"), true);
  assertEquals(ds.parseVariant("nope"), null);
});

Deno.test("DesignSystem: prefix is enforced through the theme", async () => {
  const ds = await designSystemFor('@import "twill" prefix(tw);');
  assertEquals(ds.parseCandidate("flex").length, 0);
  assertEquals(ds.parseCandidate("tw:block").length, 1);
});

Deno.test("DesignSystem: important flag from the import", async () => {
  const ds = await designSystemFor('@import "twill" important;');
  assertEquals(ds.important, true);
});

Deno.test("Variants registry: orders, groups, and re-registration", () => {
  const variants = new Variants();
  variants.static("a", () => {});
  variants.group(() => {
    variants.static("b", () => {});
    variants.functional("c", () => {});
  });
  variants.static("d", () => {});
  assertEquals(variants.get("a")!.order, 1);
  assertEquals(variants.get("b")!.order, 2);
  assertEquals(variants.get("c")!.order, 2);
  assertEquals(variants.get("d")!.order, 3);
  variants.functional("a", () => {}, { compounds: Compounds.AT_RULES });
  assertEquals(variants.get("a")!.order, 1);
  assertEquals(variants.get("a")!.kind, "functional");
  assertEquals(variants.get("a")!.compounds, Compounds.AT_RULES);
  assertEquals(variants.keys(), ["a", "b", "c", "d"]);
});

Deno.test("compoundsForSelectors", () => {
  assertEquals(compoundsForSelectors(["&:hover"]), Compounds.STYLE_RULES);
  assertEquals(compoundsForSelectors(["@media (x)"]), Compounds.AT_RULES);
  assertEquals(
    compoundsForSelectors(["&:hover", "@supports (x)"]),
    Compounds.STYLE_RULES | Compounds.AT_RULES,
  );
  assertEquals(compoundsForSelectors(["@starting-style"]), Compounds.NEVER);
  assertEquals(compoundsForSelectors(["&::before"]), Compounds.NEVER);
  assertEquals(compoundsForSelectors([]), Compounds.NEVER);
});
