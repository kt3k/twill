import { assertEquals } from "@std/assert";
import {
  type Candidate,
  findRoots,
  parseCandidate,
  parseModifier,
  type ParserContext,
  parseVariant,
  type Variant,
  type VariantKind,
} from "./candidate.ts";

// A stand-in for the design system registries of §9 and §10.
const STATIC_UTILITIES = new Set([
  "flex",
  "block",
  "underline",
  "border",
  "rounded-full",
  "-mt-px",
]);
const FUNCTIONAL_UTILITIES = new Set([
  "w",
  "bg",
  "p",
  "mt",
  "-mt",
  "border",
  "border-t",
  "text",
  "z",
  "-z",
  "grid-cols",
  "tab",
  "@",
]);

const NEVER = 0;
const AT_RULES = 1;
const STYLE_RULES = 2;

const VARIANTS: Record<
  string,
  { kind: VariantKind; compounds: number; compoundsWith: number }
> = {
  "*": { kind: "static", compounds: NEVER, compoundsWith: NEVER },
  hover: { kind: "static", compounds: STYLE_RULES, compoundsWith: NEVER },
  first: { kind: "static", compounds: STYLE_RULES, compoundsWith: NEVER },
  dark: { kind: "static", compounds: AT_RULES, compoundsWith: NEVER },
  sm: { kind: "static", compounds: AT_RULES, compoundsWith: NEVER },
  "first-letter": { kind: "static", compounds: NEVER, compoundsWith: NEVER },
  data: { kind: "functional", compounds: STYLE_RULES, compoundsWith: NEVER },
  aria: { kind: "functional", compounds: STYLE_RULES, compoundsWith: NEVER },
  nth: { kind: "functional", compounds: STYLE_RULES, compoundsWith: NEVER },
  "nth-last": {
    kind: "functional",
    compounds: STYLE_RULES,
    compoundsWith: NEVER,
  },
  min: { kind: "functional", compounds: AT_RULES, compoundsWith: NEVER },
  max: { kind: "functional", compounds: AT_RULES, compoundsWith: NEVER },
  supports: { kind: "functional", compounds: AT_RULES, compoundsWith: NEVER },
  "@": { kind: "functional", compounds: AT_RULES, compoundsWith: NEVER },
  "@min": { kind: "functional", compounds: AT_RULES, compoundsWith: NEVER },
  not: {
    kind: "compound",
    compounds: STYLE_RULES | AT_RULES,
    compoundsWith: STYLE_RULES | AT_RULES,
  },
  group: {
    kind: "compound",
    compounds: STYLE_RULES,
    compoundsWith: STYLE_RULES,
  },
  peer: {
    kind: "compound",
    compounds: STYLE_RULES,
    compoundsWith: STYLE_RULES,
  },
  has: { kind: "compound", compounds: STYLE_RULES, compoundsWith: STYLE_RULES },
  in: { kind: "compound", compounds: STYLE_RULES, compoundsWith: STYLE_RULES },
};

function compoundsOf(variant: Variant): number {
  if (variant.kind === "arbitrary") {
    const s = variant.selector;
    if (s.startsWith("@")) {
      return s.startsWith("@media") || s.startsWith("@supports") ||
          s.startsWith("@container")
        ? AT_RULES
        : NEVER;
    }
    return s.includes("::") ? NEVER : STYLE_RULES;
  }
  return VARIANTS[variant.root].compounds;
}

function makeContext(prefix: string | null = null): ParserContext {
  const cache = new Map<string, Variant | null>();
  const ds: ParserContext = {
    theme: { prefix },
    utilities: {
      has: (name, kind) =>
        kind === "static"
          ? STATIC_UTILITIES.has(name)
          : FUNCTIONAL_UTILITIES.has(name),
    },
    variants: {
      has: (name) => name in VARIANTS,
      kind: (name) => VARIANTS[name]?.kind,
      compoundsWith: (parent, child) => {
        const p = VARIANTS[parent];
        if (p.kind !== "compound") return false;
        const c = compoundsOf(child);
        return c !== NEVER && p.compoundsWith !== NEVER &&
          (c & p.compoundsWith) !== 0;
      },
    },
    parseVariant(input) {
      if (cache.has(input)) return cache.get(input)!;
      const result = parseVariant(input, ds);
      cache.set(input, result);
      return result;
    },
  };
  return ds;
}

const ds = makeContext();

function candidates(input: string, ctx: ParserContext = ds): Candidate[] {
  return [...parseCandidate(input, ctx)];
}

function base(raw: string, variants: Variant[] = [], important = false) {
  return { raw, variants, important };
}

Deno.test("parseCandidate: static utility", () => {
  assertEquals(candidates("flex"), [{
    kind: "static",
    root: "flex",
    ...base("flex"),
  }]);
  assertEquals(candidates("rounded-full"), [
    { kind: "static", root: "rounded-full", ...base("rounded-full") },
  ]);
});

Deno.test("parseCandidate: functional utility with named value", () => {
  assertEquals(candidates("w-4"), [
    {
      kind: "functional",
      root: "w",
      value: { kind: "named", value: "4", fraction: null },
      modifier: null,
      ...base("w-4"),
    },
  ]);
  assertEquals(candidates("bg-red-500"), [
    {
      kind: "functional",
      root: "bg",
      value: { kind: "named", value: "red-500", fraction: null },
      modifier: null,
      ...base("bg-red-500"),
    },
  ]);
});

Deno.test("parseCandidate: functional utility without value", () => {
  assertEquals(candidates("border"), [
    { kind: "static", root: "border", ...base("border") },
    {
      kind: "functional",
      root: "border",
      value: null,
      modifier: null,
      ...base("border"),
    },
  ]);
});

Deno.test("parseCandidate: multiple root interpretations", () => {
  assertEquals(candidates("border-t-2"), [
    {
      kind: "functional",
      root: "border-t",
      value: { kind: "named", value: "2", fraction: null },
      modifier: null,
      ...base("border-t-2"),
    },
    {
      kind: "functional",
      root: "border",
      value: { kind: "named", value: "t-2", fraction: null },
      modifier: null,
      ...base("border-t-2"),
    },
  ]);
});

Deno.test("parseCandidate: negative root", () => {
  assertEquals(candidates("-mt-2"), [
    {
      kind: "functional",
      root: "-mt",
      value: { kind: "named", value: "2", fraction: null },
      modifier: null,
      ...base("-mt-2"),
    },
  ]);
});

Deno.test("parseCandidate: modifiers and fractions", () => {
  assertEquals(candidates("w-1/2"), [
    {
      kind: "functional",
      root: "w",
      value: { kind: "named", value: "1", fraction: "1/2" },
      modifier: { kind: "named", value: "2" },
      ...base("w-1/2"),
    },
  ]);
  assertEquals(candidates("bg-red-500/50"), [
    {
      kind: "functional",
      root: "bg",
      value: { kind: "named", value: "red-500", fraction: "red-500/50" },
      modifier: { kind: "named", value: "50" },
      ...base("bg-red-500/50"),
    },
  ]);
  assertEquals(candidates("bg-red-500/[0.5]"), [
    {
      kind: "functional",
      root: "bg",
      value: { kind: "named", value: "red-500", fraction: null },
      modifier: { kind: "arbitrary", value: "0.5" },
      ...base("bg-red-500/[0.5]"),
    },
  ]);
  assertEquals(candidates("bg-red-500/(--alpha)"), [
    {
      kind: "functional",
      root: "bg",
      value: { kind: "named", value: "red-500", fraction: null },
      modifier: { kind: "arbitrary", value: "var(--alpha)" },
      ...base("bg-red-500/(--alpha)"),
    },
  ]);
  // More than one modifier is invalid.
  assertEquals(candidates("bg-red-500/50/50"), []);
  // An invalid modifier is invalid.
  assertEquals(candidates("bg-red-500/[]"), []);
  assertEquals(candidates("bg-red-500/(foo)"), []);
  assertEquals(candidates("bg-red-500/a b"), []);
});

Deno.test("parseCandidate: arbitrary values", () => {
  assertEquals(candidates("w-[13px]"), [
    {
      kind: "functional",
      root: "w",
      value: { kind: "arbitrary", value: "13px", dataType: null },
      modifier: null,
      ...base("w-[13px]"),
    },
  ]);
  assertEquals(candidates("bg-[length:10px_20px]"), [
    {
      kind: "functional",
      root: "bg",
      value: { kind: "arbitrary", value: "10px 20px", dataType: "length" },
      modifier: null,
      ...base("bg-[length:10px_20px]"),
    },
  ]);
  assertEquals(candidates("bg-[url(/a_b.png)]"), [
    {
      kind: "functional",
      root: "bg",
      value: { kind: "arbitrary", value: "url(/a_b.png)", dataType: null },
      modifier: null,
      ...base("bg-[url(/a_b.png)]"),
    },
  ]);
  assertEquals(candidates("bg-[#0088cc]/50"), [
    {
      kind: "functional",
      root: "bg",
      value: { kind: "arbitrary", value: "#0088cc", dataType: null },
      modifier: { kind: "named", value: "50" },
      ...base("bg-[#0088cc]/50"),
    },
  ]);
  // Empty values, empty hints, and unbalanced values are invalid.
  assertEquals(candidates("w-[]"), []);
  assertEquals(candidates("w-[_]"), []);
  assertEquals(candidates("w-[:1px]"), []);
  assertEquals(candidates("w-[1px;]"), []);
  assertEquals(candidates("w-[1px}]"), []);
  assertEquals(candidates("w-[1px"), []);
  assertEquals(candidates("unknown-[1px]"), []);
  // `;` and `}` are allowed inside quotes or parentheses.
  assertEquals(candidates("w-['a;b']")[0].kind, "functional");
  assertEquals(
    (candidates("w-[calc(1px;)]")[0] as { value: { value: string } }).value
      .value,
    "calc(1px;)",
  );
});

Deno.test("parseCandidate: variable shorthand", () => {
  assertEquals(candidates("w-(--my-w)"), [
    {
      kind: "functional",
      root: "w",
      value: { kind: "arbitrary", value: "var(--my-w)", dataType: null },
      modifier: null,
      ...base("w-(--my-w)"),
    },
  ]);
  assertEquals(candidates("bg-(color:--my-color)"), [
    {
      kind: "functional",
      root: "bg",
      value: { kind: "arbitrary", value: "var(--my-color)", dataType: "color" },
      modifier: null,
      ...base("bg-(color:--my-color)"),
    },
  ]);
  assertEquals(candidates("w-(my-w)"), []);
  assertEquals(candidates("w-(--a:--b:--c)"), []);
  assertEquals(candidates("unknown-(--x)"), []);
});

Deno.test("parseCandidate: arbitrary properties", () => {
  assertEquals(candidates("[mask-type:luminance]"), [
    {
      kind: "arbitrary",
      property: "mask-type",
      value: "luminance",
      modifier: null,
      ...base("[mask-type:luminance]"),
    },
  ]);
  assertEquals(candidates("[--my-var:1px]"), [
    {
      kind: "arbitrary",
      property: "--my-var",
      value: "1px",
      modifier: null,
      ...base("[--my-var:1px]"),
    },
  ]);
  assertEquals(candidates("[color:red]/50"), [
    {
      kind: "arbitrary",
      property: "color",
      value: "red",
      modifier: { kind: "named", value: "50" },
      ...base("[color:red]/50"),
    },
  ]);
  assertEquals(candidates("[color:a_b]")[0], {
    kind: "arbitrary",
    property: "color",
    value: "a b",
    modifier: null,
    ...base("[color:a_b]"),
  });
  assertEquals(candidates("[Color:red]"), []);
  assertEquals(candidates("[color]"), []);
  assertEquals(candidates("[:red]"), []);
  assertEquals(candidates("[color:]"), []);
  assertEquals(candidates("[color:red"), []);
  assertEquals(candidates("[color:red;]"), []);
});

Deno.test("parseCandidate: importance", () => {
  assertEquals(candidates("underline!"), [
    { kind: "static", root: "underline", ...base("underline!", [], true) },
  ]);
  assertEquals(candidates("!underline"), [
    { kind: "static", root: "underline", ...base("!underline", [], true) },
  ]);
  assertEquals(candidates("hover:w-4!"), [
    {
      kind: "functional",
      root: "w",
      value: { kind: "named", value: "4", fraction: null },
      modifier: null,
      ...base("hover:w-4!", [{ kind: "static", root: "hover" }], true),
    },
  ]);
  assertEquals(candidates("!underline!"), []);
});

Deno.test("parseCandidate: variants are in application order", () => {
  assertEquals(candidates("dark:hover:flex"), [
    {
      kind: "static",
      root: "flex",
      ...base("dark:hover:flex", [
        { kind: "static", root: "hover" },
        { kind: "static", root: "dark" },
      ]),
    },
  ]);
  assertEquals(candidates("unknown:flex"), []);
  assertEquals(candidates("hover:unknown"), []);
});

Deno.test("parseCandidate: prefix enforcement", () => {
  const prefixed = makeContext("tw");
  assertEquals(candidates("flex", prefixed), []);
  assertEquals(candidates("foo:flex", prefixed), []);
  assertEquals(candidates("tw:flex", prefixed), [{
    kind: "static",
    root: "flex",
    ...base("tw:flex"),
  }]);
  assertEquals(candidates("tw:hover:flex", prefixed), [
    {
      kind: "static",
      root: "flex",
      ...base("tw:hover:flex", [{ kind: "static", root: "hover" }]),
    },
  ]);
});

Deno.test("parseCandidate: invalid named values", () => {
  assertEquals(candidates("w-a$b"), []);
  assertEquals(candidates("w-"), []);
  assertEquals(candidates("unknown"), []);
});

Deno.test("parseCandidate: @ root", () => {
  assertEquals(candidates("@container")[0]?.kind, "functional");
  assertEquals((candidates("@container")[0] as { root: string }).root, "@");
});

Deno.test("parseModifier", () => {
  assertEquals(parseModifier("50"), { kind: "named", value: "50" });
  assertEquals(parseModifier("[0.5]"), { kind: "arbitrary", value: "0.5" });
  assertEquals(parseModifier("[a_b]"), { kind: "arbitrary", value: "a b" });
  assertEquals(parseModifier("(--x)"), {
    kind: "arbitrary",
    value: "var(--x)",
  });
  assertEquals(parseModifier("[]"), null);
  assertEquals(parseModifier("(x)"), null);
  assertEquals(parseModifier("a b"), null);
});

Deno.test("findRoots", () => {
  const exists = (r: string) => ["border", "border-t", "@"].includes(r);
  assertEquals([...findRoots("border", exists)], [["border", null]]);
  assertEquals([...findRoots("border-t-2", exists)], [["border-t", "2"], [
    "border",
    "t-2",
  ]]);
  assertEquals([...findRoots("border-", exists)], []);
  assertEquals([...findRoots("border-t-", exists)], []);
  assertEquals([...findRoots("@md", exists)], [["@", "md"]]);
  assertEquals([...findRoots("@min-md", exists)], [["@", "min-md"]]);
  assertEquals([...findRoots("nope-1", exists)], []);
});

function variant(input: string): Variant | null {
  return parseVariant(input, ds);
}

Deno.test("parseVariant: static", () => {
  assertEquals(variant("hover"), { kind: "static", root: "hover" });
  assertEquals(variant("*"), { kind: "static", root: "*" });
  assertEquals(variant("hover-x"), null);
  assertEquals(variant("hover/x"), null);
  assertEquals(variant("unknown"), null);
});

Deno.test("parseVariant: functional", () => {
  assertEquals(variant("data-visible"), {
    kind: "functional",
    root: "data",
    value: { kind: "named", value: "visible" },
    modifier: null,
  });
  assertEquals(variant("data-[state=open]"), {
    kind: "functional",
    root: "data",
    value: { kind: "arbitrary", value: "state=open" },
    modifier: null,
  });
  assertEquals(variant("supports-(--x)"), {
    kind: "functional",
    root: "supports",
    value: { kind: "arbitrary", value: "var(--x)" },
    modifier: null,
  });
  assertEquals(variant("nth-3"), {
    kind: "functional",
    root: "nth",
    value: { kind: "named", value: "3" },
    modifier: null,
  });
  assertEquals(variant("nth-last-3"), {
    kind: "functional",
    root: "nth-last",
    value: { kind: "named", value: "3" },
    modifier: null,
  });
  assertEquals(variant("min-[600px]"), {
    kind: "functional",
    root: "min",
    value: { kind: "arbitrary", value: "600px" },
    modifier: null,
  });
  assertEquals(variant("@md/main"), {
    kind: "functional",
    root: "@",
    value: { kind: "named", value: "md" },
    modifier: { kind: "named", value: "main" },
  });
  assertEquals(variant("@min-md"), {
    kind: "functional",
    root: "@min",
    value: { kind: "named", value: "md" },
    modifier: null,
  });
  assertEquals(variant("data"), {
    kind: "functional",
    root: "data",
    value: null,
    modifier: null,
  });
  assertEquals(variant("data-[]"), null);
  assertEquals(variant("data-(x)"), null);
  assertEquals(variant("data-a$b"), null);
  assertEquals(variant("data-x/[]"), null);
});

Deno.test("parseVariant: compound", () => {
  assertEquals(variant("group-hover"), {
    kind: "compound",
    root: "group",
    modifier: null,
    variant: { kind: "static", root: "hover" },
  });
  assertEquals(variant("group-hover/item"), {
    kind: "compound",
    root: "group",
    modifier: { kind: "named", value: "item" },
    variant: { kind: "static", root: "hover" },
  });
  assertEquals(variant("not-hover"), {
    kind: "compound",
    root: "not",
    modifier: null,
    variant: { kind: "static", root: "hover" },
  });
  assertEquals(variant("not-sm"), {
    kind: "compound",
    root: "not",
    modifier: null,
    variant: { kind: "static", root: "sm" },
  });
  assertEquals(variant("in-data-visible"), {
    kind: "compound",
    root: "in",
    modifier: null,
    variant: {
      kind: "functional",
      root: "data",
      value: { kind: "named", value: "visible" },
      modifier: null,
    },
  });
  assertEquals(variant("has-[>img]"), {
    kind: "compound",
    root: "has",
    modifier: null,
    variant: { kind: "arbitrary", selector: ">img", relative: true },
  });
  // The modifier of not/has/in belongs to the inner variant.
  assertEquals(variant("not-group-hover/item"), {
    kind: "compound",
    root: "not",
    modifier: null,
    variant: {
      kind: "compound",
      root: "group",
      modifier: { kind: "named", value: "item" },
      variant: { kind: "static", root: "hover" },
    },
  });
  // Incompatible inner variants are rejected.
  assertEquals(variant("group-sm"), null);
  assertEquals(variant("group-*"), null);
  assertEquals(variant("group-first-letter"), null);
  assertEquals(variant("group"), null);
  assertEquals(variant("group-unknown"), null);
});

Deno.test("parseVariant: arbitrary", () => {
  assertEquals(variant("[&_p]"), {
    kind: "arbitrary",
    selector: "& p",
    relative: false,
  });
  assertEquals(variant("[p]"), {
    kind: "arbitrary",
    selector: "&:is(p)",
    relative: false,
  });
  assertEquals(variant("[>img]"), {
    kind: "arbitrary",
    selector: ">img",
    relative: true,
  });
  assertEquals(variant("[@media(width>=100px)]"), {
    kind: "arbitrary",
    selector: "@media(width>=100px)",
    relative: false,
  });
  assertEquals(variant("[@media(width>=100px)_&]"), null);
  assertEquals(variant("[]"), null);
  assertEquals(variant("[_]"), null);
  assertEquals(variant("[a;b]"), null);
});

Deno.test("parseVariant: memoization returns the same object", () => {
  assertEquals(ds.parseVariant("hover") === ds.parseVariant("hover"), true);
});
