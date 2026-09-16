import { assertEquals } from "@std/assert";
import {
  negateAtRule,
  negateSelector,
  quoteAttributeValue,
} from "./builtin_variants.ts";
import { compileRaw, designSystemFor } from "./testing.ts";
import type { DesignSystem } from "./design_system.ts";

let ds: DesignSystem;

async function setup(): Promise<DesignSystem> {
  ds ??= await designSystemFor();
  return ds;
}

function block(selector: string, body: string): string {
  const lines = body.trimEnd().split("\n").map((l) => `  ${l}`).join("\n");
  return `${selector} {\n${lines}\n}\n`;
}

Deno.test("variants: pseudo classes", async () => {
  const ds = await setup();
  assertEquals(
    compileRaw(ds, "hover:flex"),
    block(
      ".hover\\:flex",
      block("&:hover", block("@media (hover: hover)", "display: flex;"))
        .trimEnd(),
    ),
  );
  assertEquals(
    compileRaw(ds, "focus:flex"),
    block(".focus\\:flex", "&:focus {\n  display: flex;\n}"),
  );
  assertEquals(
    compileRaw(ds, "first:flex"),
    block(".first\\:flex", "&:first-child {\n  display: flex;\n}"),
  );
  assertEquals(
    compileRaw(ds, "odd:flex"),
    block(".odd\\:flex", "&:nth-child(odd) {\n  display: flex;\n}"),
  );
  assertEquals(
    compileRaw(ds, "open:flex"),
    block(
      ".open\\:flex",
      "&:is([open], :popover-open, :open) {\n  display: flex;\n}",
    ),
  );
  assertEquals(
    compileRaw(ds, "inert:flex"),
    block(".inert\\:flex", "&:is([inert], [inert] *) {\n  display: flex;\n}"),
  );
  assertEquals(
    compileRaw(ds, "ltr:flex"),
    block(
      ".ltr\\:flex",
      '&:where(:dir(ltr), [dir="ltr"], [dir="ltr"] *) {\n  display: flex;\n}',
    ),
  );
});

Deno.test("variants: pseudo elements", async () => {
  const ds = await setup();
  assertEquals(
    compileRaw(ds, "before:block"),
    `.before\\:block {
  &::before {
    @property --tw-content {
      syntax: "*";
      inherits: false;
      initial-value: "";
    }
    content: var(--tw-content);
    display: block;
  }
}
`,
  );
  assertEquals(
    compileRaw(ds, "marker:flex"),
    `.marker\\:flex {
  & *::marker {
    display: flex;
  }
  &::marker {
    display: flex;
  }
  & *::-webkit-details-marker {
    display: flex;
  }
  &::-webkit-details-marker {
    display: flex;
  }
}
`,
  );
  assertEquals(
    compileRaw(ds, "placeholder:flex"),
    block(".placeholder\\:flex", "&::placeholder {\n  display: flex;\n}"),
  );
});

Deno.test("variants: * and **", async () => {
  const ds = await setup();
  assertEquals(
    compileRaw(ds, "*:flex"),
    block(".\\*\\:flex", ":is(& > *) {\n  display: flex;\n}"),
  );
  assertEquals(
    compileRaw(ds, "**:flex"),
    block(".\\*\\*\\:flex", ":is(& *) {\n  display: flex;\n}"),
  );
  // Pseudo-element variants cannot be compounded.
  assertEquals(compileRaw(ds, "group-*:flex"), "");
  assertEquals(compileRaw(ds, "not-before:flex"), "");
});

Deno.test("variants: breakpoints and containers", async () => {
  const ds = await setup();
  assertEquals(
    compileRaw(ds, "sm:flex"),
    block(".sm\\:flex", "@media (width >= 40rem) {\n  display: flex;\n}"),
  );
  assertEquals(
    compileRaw(ds, "max-md:flex"),
    block(".max-md\\:flex", "@media (width < 48rem) {\n  display: flex;\n}"),
  );
  assertEquals(
    compileRaw(ds, "min-[600px]:flex"),
    block(
      ".min-\\[600px\\]\\:flex",
      "@media (width >= 600px) {\n  display: flex;\n}",
    ),
  );
  assertEquals(
    compileRaw(ds, "min-md:flex"),
    block(".min-md\\:flex", "@media (width >= 48rem) {\n  display: flex;\n}"),
  );
  assertEquals(
    compileRaw(ds, "@md:flex"),
    block(
      ".\\@md\\:flex",
      "@container (width >= 28rem) {\n  display: flex;\n}",
    ),
  );
  assertEquals(
    compileRaw(ds, "@md/main:flex"),
    block(
      ".\\@md\\/main\\:flex",
      "@container main (width >= 28rem) {\n  display: flex;\n}",
    ),
  );
  assertEquals(
    compileRaw(ds, "@max-md:flex"),
    block(
      ".\\@max-md\\:flex",
      "@container (width < 28rem) {\n  display: flex;\n}",
    ),
  );
  assertEquals(
    compileRaw(ds, "@min-[300px]:flex"),
    block(
      ".\\@min-\\[300px\\]\\:flex",
      "@container (width >= 300px) {\n  display: flex;\n}",
    ),
  );
  assertEquals(compileRaw(ds, "min-nope:flex"), "");
  assertEquals(compileRaw(ds, "sm/foo:flex"), "");
  assertEquals(compileRaw(ds, "min-[var(--x)]:flex"), "");
});

Deno.test("variants: group, peer, in, has", async () => {
  const ds = await setup();
  assertEquals(
    compileRaw(ds, "group-hover:flex"),
    block(
      ".group-hover\\:flex",
      block(
        "&:is(:where(.group):hover *)",
        block("@media (hover: hover)", "display: flex;"),
      ).trimEnd(),
    ),
  );
  assertEquals(
    compileRaw(ds, "group-hover/item:flex"),
    block(
      ".group-hover\\/item\\:flex",
      block(
        "&:is(:where(.group\\/item):hover *)",
        block("@media (hover: hover)", "display: flex;"),
      ).trimEnd(),
    ),
  );
  assertEquals(
    compileRaw(ds, "peer-checked:flex"),
    block(
      ".peer-checked\\:flex",
      "&:is(:where(.peer):checked ~ *) {\n  display: flex;\n}",
    ),
  );
  assertEquals(
    compileRaw(ds, "has-[>img]:flex"),
    block(".has-\\[\\>img\\]\\:flex", "&:has(> img) {\n  display: flex;\n}"),
  );
  assertEquals(
    compileRaw(ds, "has-checked:flex"),
    block(".has-checked\\:flex", "&:has(*:checked) {\n  display: flex;\n}"),
  );
  assertEquals(
    compileRaw(ds, "in-data-visible:flex"),
    block(
      ".in-data-visible\\:flex",
      ":where(*[data-visible]) & {\n  display: flex;\n}",
    ),
  );
  // A relative arbitrary selector at the top level is rejected.
  assertEquals(compileRaw(ds, "[>img]:flex"), "");
  assertEquals(compileRaw(ds, "group-[>img]:flex"), "");
  assertEquals(compileRaw(ds, "group-sm:flex"), "");
  assertEquals(compileRaw(ds, "in-hover/x:flex"), "");
});

Deno.test("variants: not", async () => {
  const ds = await setup();
  assertEquals(
    compileRaw(ds, "not-hover:flex"),
    `.not-hover\\:flex {
  & {
    &:not(:hover) {
      display: flex;
    }
    @media not all and (hover: hover) {
      display: flex;
    }
  }
}
`,
  );
  assertEquals(
    compileRaw(ds, "not-supports-grid:flex"),
    block(
      ".not-supports-grid\\:flex",
      "@supports not (grid: var(--tw)) {\n  display: flex;\n}",
    ),
  );
  assertEquals(
    compileRaw(ds, "not-sm:flex"),
    block(
      ".not-sm\\:flex",
      "@media not all and (width >= 40rem) {\n  display: flex;\n}",
    ),
  );
  assertEquals(
    compileRaw(ds, "not-print:flex"),
    block(".not-print\\:flex", "@media not print {\n  display: flex;\n}"),
  );
  assertEquals(
    compileRaw(ds, "not-@md:flex"),
    block(
      ".not-\\@md\\:flex",
      "@container not (width >= 28rem) {\n  display: flex;\n}",
    ),
  );
  assertEquals(
    compileRaw(ds, "not-group-hover:flex"),
    `.not-group-hover\\:flex {
  & {
    &:not(:is(:where(.group):hover *)) {
      display: flex;
    }
    @media not all and (hover: hover) {
      display: flex;
    }
  }
}
`,
  );
  // `marker` yields several sibling rules.
  assertEquals(compileRaw(ds, "not-marker:flex"), "");
  assertEquals(compileRaw(ds, "not-hover/x:flex"), "");
});

Deno.test("variants: aria, data, nth, supports", async () => {
  const ds = await setup();
  assertEquals(
    compileRaw(ds, "aria-checked:flex"),
    block(
      ".aria-checked\\:flex",
      '&[aria-checked="true"] {\n  display: flex;\n}',
    ),
  );
  assertEquals(
    compileRaw(ds, "aria-[label=foo]:flex"),
    block(
      ".aria-\\[label\\=foo\\]\\:flex",
      '&[aria-label="foo"] {\n  display: flex;\n}',
    ),
  );
  assertEquals(
    compileRaw(ds, "data-[state=open]:flex"),
    block(
      ".data-\\[state\\=open\\]\\:flex",
      '&[data-state="open"] {\n  display: flex;\n}',
    ),
  );
  assertEquals(
    compileRaw(ds, "data-[state=open_i]:flex"),
    block(
      ".data-\\[state\\=open_i\\]\\:flex",
      '&[data-state="open" i] {\n  display: flex;\n}',
    ),
  );
  assertEquals(
    compileRaw(ds, "data-visible:flex"),
    block(".data-visible\\:flex", "&[data-visible] {\n  display: flex;\n}"),
  );
  assertEquals(
    compileRaw(ds, "nth-3:flex"),
    block(".nth-3\\:flex", "&:nth-child(3) {\n  display: flex;\n}"),
  );
  assertEquals(
    compileRaw(ds, "nth-last-[2n+1]:flex"),
    block(
      ".nth-last-\\[2n\\+1\\]\\:flex",
      "&:nth-last-child(2n+1) {\n  display: flex;\n}",
    ),
  );
  assertEquals(compileRaw(ds, "nth-foo:flex"), "");
  assertEquals(
    compileRaw(ds, "supports-[display:grid]:flex"),
    block(
      ".supports-\\[display\\:grid\\]\\:flex",
      "@supports (display:grid) {\n  display: flex;\n}",
    ),
  );
  assertEquals(
    compileRaw(ds, "supports-grid:flex"),
    block(
      ".supports-grid\\:flex",
      "@supports (grid: var(--tw)) {\n  display: flex;\n}",
    ),
  );
  assertEquals(
    compileRaw(ds, "supports-[not(display:grid)]:flex"),
    block(
      ".supports-\\[not\\(display\\:grid\\)\\]\\:flex",
      "@supports not (display:grid) {\n  display: flex;\n}",
    ),
  );
  assertEquals(compileRaw(ds, "aria-checked/x:flex"), "");
});

Deno.test("variants: media queries", async () => {
  const ds = await setup();
  assertEquals(
    compileRaw(ds, "dark:flex"),
    block(
      ".dark\\:flex",
      "@media (prefers-color-scheme: dark) {\n  display: flex;\n}",
    ),
  );
  assertEquals(
    compileRaw(ds, "print:flex"),
    block(".print\\:flex", "@media print {\n  display: flex;\n}"),
  );
  assertEquals(
    compileRaw(ds, "motion-reduce:flex"),
    block(
      ".motion-reduce\\:flex",
      "@media (prefers-reduced-motion: reduce) {\n  display: flex;\n}",
    ),
  );
  assertEquals(
    compileRaw(ds, "starting:flex"),
    block(".starting\\:flex", "@starting-style {\n  display: flex;\n}"),
  );
  assertEquals(
    compileRaw(ds, "portrait:flex"),
    block(
      ".portrait\\:flex",
      "@media (orientation: portrait) {\n  display: flex;\n}",
    ),
  );
  assertEquals(
    compileRaw(ds, "pointer-coarse:flex"),
    block(
      ".pointer-coarse\\:flex",
      "@media (pointer: coarse) {\n  display: flex;\n}",
    ),
  );
  assertEquals(
    compileRaw(ds, "noscript:flex"),
    block(
      ".noscript\\:flex",
      "@media (scripting: none) {\n  display: flex;\n}",
    ),
  );
});

Deno.test("variants: arbitrary and stacking", async () => {
  const ds = await setup();
  assertEquals(
    compileRaw(ds, "[&_p]:flex"),
    block(".\\[\\&_p\\]\\:flex", "& p {\n  display: flex;\n}"),
  );
  assertEquals(
    compileRaw(ds, "[@media(width>=100px)]:flex"),
    block(
      ".\\[\\@media\\(width\\>\\=100px\\)\\]\\:flex",
      "@media (width>=100px) {\n  display: flex;\n}",
    ),
  );
  assertEquals(
    compileRaw(ds, "[p]:flex"),
    block(".\\[p\\]\\:flex", "&:is(p) {\n  display: flex;\n}"),
  );
  assertEquals(
    compileRaw(ds, "dark:hover:flex"),
    block(
      ".dark\\:hover\\:flex",
      block(
        "@media (prefers-color-scheme: dark)",
        block("&:hover", block("@media (hover: hover)", "display: flex;"))
          .trimEnd(),
      ).trimEnd(),
    ),
  );
});

Deno.test("variants: custom dark override", async () => {
  const ds = await designSystemFor('@import "twill";');
  ds.variants.static("dark", (node) => {
    node.nodes = [{
      kind: "rule",
      selector: "&:where(.dark, .dark *)",
      nodes: node.nodes,
    }];
  });
  assertEquals(
    compileRaw(ds, "dark:flex"),
    block(".dark\\:flex", "&:where(.dark, .dark *) {\n  display: flex;\n}"),
  );
});

Deno.test("variants: ordering", async () => {
  const ds = await setup();
  const v = (s: string) => ds.parseVariant(s)!;
  const cmp = (a: string, z: string) =>
    Math.sign(ds.variants.compare(v(a), v(z)));
  assertEquals(cmp("hover", "focus"), -1);
  assertEquals(cmp("focus", "hover"), 1);
  assertEquals(cmp("hover", "hover"), 0);
  assertEquals(cmp("sm", "md"), -1);
  assertEquals(cmp("md", "lg"), -1);
  assertEquals(cmp("lg", "min-[600px]"), 1);
  assertEquals(cmp("max-md", "max-sm"), -1);
  assertEquals(cmp("max-lg", "sm"), -1);
  assertEquals(cmp("@md", "@lg"), -1);
  assertEquals(cmp("sm", "dark"), -1);
  assertEquals(cmp("hover", "[&_p]"), -1);
  assertEquals(cmp("[&_a]", "[&_p]"), -1);
  assertEquals(cmp("group-hover", "group-focus"), -1);
  assertEquals(cmp("group-hover", "group-hover/item"), -1);
  assertEquals(cmp("data-a", "data-b"), -1);
  assertEquals(cmp("data-a", "data-[b]"), -1);
  assertEquals(cmp("nth-3", "nth-last-3"), -1);
  assertEquals(cmp("not-hover", "not-focus"), -1);
  assertEquals(cmp("not-hover", "hover"), -1);
});

Deno.test("variants: getVariantOrder", async () => {
  const ds = await designSystemFor();
  const hover = ds.parseVariant("hover")!;
  const focus = ds.parseVariant("focus")!;
  const sm = ds.parseVariant("sm")!;
  const order = ds.getVariantOrder();
  assertEquals(order.get(hover)! < order.get(focus)!, true);
  assertEquals(order.get(focus)! < order.get(sm)!, true);
  // Parsing a new variant invalidates the cache.
  const dark = ds.parseVariant("dark")!;
  assertEquals(
    ds.getVariantOrder().get(dark)! > ds.getVariantOrder().get(sm)!,
    true,
  );
});

Deno.test("negateSelector and negateAtRule", () => {
  assertEquals(negateSelector("&:hover"), "&:not(:hover)");
  assertEquals(negateSelector("&[data-x]"), "&:not([data-x])");
  assertEquals(negateSelector(":where(.a) &"), "&:not(:where(.a) *)");
  assertEquals(
    negateSelector("&:is(:where(.group):hover *)"),
    "&:not(:is(:where(.group):hover *))",
  );
  assertEquals(negateSelector("&::before"), null);
  assertEquals(negateSelector("&"), null);
  assertEquals(negateAtRule("@media", "(hover: hover)"), [
    "@media",
    "not all and (hover: hover)",
  ]);
  assertEquals(negateAtRule("@media", "print"), ["@media", "not print"]);
  assertEquals(negateAtRule("@supports", "(display: grid)"), [
    "@supports",
    "not (display: grid)",
  ]);
  assertEquals(negateAtRule("@container", "(width >= 1px)"), [
    "@container",
    "not (width >= 1px)",
  ]);
  assertEquals(negateAtRule("@container", "main (width >= 1px)"), [
    "@container",
    "main not (width >= 1px)",
  ]);
  assertEquals(negateAtRule("@starting-style", ""), null);
});

Deno.test("quoteAttributeValue", () => {
  assertEquals(quoteAttributeValue("state"), "state");
  assertEquals(quoteAttributeValue("state=open"), 'state="open"');
  assertEquals(quoteAttributeValue('state="open"'), 'state="open"');
  assertEquals(quoteAttributeValue("state='open'"), "state='open'");
  assertEquals(quoteAttributeValue("state=open i"), 'state="open" i');
  assertEquals(quoteAttributeValue("state^=op"), 'state^="op"');
  assertEquals(quoteAttributeValue("state="), null);
  assertEquals(quoteAttributeValue("st ate=open"), null);
});
