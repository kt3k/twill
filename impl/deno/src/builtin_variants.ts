/**
 * Built-in variants (SPEC §9.5). Registration order determines output order.
 *
 * @module
 */

import {
  type AstNode,
  atRoot,
  atRule,
  decl,
  styleRule,
  walk,
  WalkAction,
} from "./ast.ts";
import type {
  CompoundVariant,
  FunctionalVariant,
  Variant,
} from "./candidate.ts";
import type { Theme } from "./theme.ts";
import { escape, isPositiveInteger, segment } from "./utils.ts";
import {
  Compounds,
  hasNestedStyleRules,
  staticVariant,
  type VariantRule,
  type Variants,
} from "./variants.ts";

/** `@property --tw-<name> { syntax: "*"; inherits: false; [initial-value: <v>;] }` inside `at-root`. */
export function propertyRegistration(
  name: string,
  initialValue?: string,
  syntax = "*",
  inherits = false,
): AstNode {
  const nodes: AstNode[] = [
    decl("syntax", `"${syntax}"`),
    decl("inherits", inherits ? "true" : "false"),
  ];
  if (initialValue !== undefined) {
    nodes.push(decl("initial-value", initialValue));
  }
  return atRoot([atRule("@property", name, nodes)]);
}

function replaceAmpersand(selector: string, replacement: string): string {
  return selector.replaceAll("&", replacement);
}

/** Negates a style selector produced by a variant, or returns null. */
export function negateSelector(selector: string): string | null {
  if (selector.includes("::")) return null;
  const parts = segment(selector, ",").map((part) => {
    part = part.trim();
    if (part === "&") return null;
    if (part.startsWith("&")) {
      const rest = part.slice(1);
      if (
        !rest.includes("&") && (rest.startsWith(":") || rest.startsWith("["))
      ) {
        return `&:not(${rest})`;
      }
    }
    return `&:not(${replaceAmpersand(part, "*")})`;
  });
  if (parts.some((p) => p === null)) return null;
  return parts.join(", ");
}

/** Negates an at-rule produced by a variant, or returns null. */
export function negateAtRule(
  name: string,
  params: string,
): [string, string] | null {
  if (name === "@media") {
    if (params.startsWith("not ")) return [name, params.slice(4)];
    if (params.startsWith("(")) return [name, `not all and ${params}`];
    return [name, `not ${params}`];
  }
  if (name === "@supports") {
    if (params.startsWith("not ")) return [name, params.slice(4)];
    return [name, `not ${params}`];
  }
  if (name === "@container") {
    if (params.startsWith("(")) return [name, `not ${params}`];
    const space = params.indexOf(" ");
    if (space === -1) return null;
    const containerName = params.slice(0, space);
    const rest = params.slice(space + 1).trim();
    if (rest.startsWith("not ")) {
      return [name, `${containerName} ${rest.slice(4)}`];
    }
    return [name, `${containerName} not ${rest}`];
  }
  return null;
}

/**
 * Wraps an unquoted attribute value in double quotes, keeping a trailing
 * ` i` or ` s` flag outside the quotes.
 */
export function quoteAttributeValue(value: string): string | null {
  const eq = value.indexOf("=");
  if (eq === -1) {
    return /^[a-zA-Z_:][-a-zA-Z0-9_:.]*$/.test(value) ? value : null;
  }
  let name = value.slice(0, eq);
  let rest = value.slice(eq + 1);
  // Attribute operators such as `~=`, `|=`, `^=`, `$=`, `*=`.
  const last = name[name.length - 1];
  let operator = "=";
  if (
    last === "~" || last === "|" || last === "^" || last === "$" || last === "*"
  ) {
    operator = `${last}=`;
    name = name.slice(0, -1);
  }
  if (!/^[a-zA-Z_:][-a-zA-Z0-9_:.]*$/.test(name)) return null;
  let flag = "";
  const flagMatch = /\s+([isIS])$/.exec(rest);
  if (flagMatch !== null) {
    flag = ` ${flagMatch[1]}`;
    rest = rest.slice(0, flagMatch.index);
  }
  rest = rest.trim();
  if (rest === "") return null;
  if (
    !(rest.startsWith('"') && rest.endsWith('"')) &&
    !(rest.startsWith("'") && rest.endsWith("'"))
  ) {
    if (rest.includes('"')) return null;
    rest = `"${rest}"`;
  }
  return `${name}${operator}${rest}${flag}`;
}

/**
 * Resolves the value used by a breakpoint or container variant, or returns
 * null when it cannot be resolved.
 */
function resolveQueryValue(
  theme: Theme,
  variant: Variant,
  namespace: string,
): string | null {
  if (variant.kind === "static") {
    return theme.resolveValue(variant.root, [namespace]);
  }
  if (variant.kind !== "functional" || variant.value === null) return null;
  const value = variant.value.kind === "arbitrary"
    ? variant.value.value
    : theme.resolveValue(variant.value.value, [namespace]);
  if (value === null || value.includes("var(")) return null;
  return value;
}

function compareQueryValues(
  theme: Theme,
  namespace: string,
  direction: "asc" | "desc",
): (a: Variant, z: Variant) => number {
  return (a, z) => {
    if (a === z) return 0;
    const aValue = resolveQueryValue(theme, a, namespace);
    const zValue = resolveQueryValue(theme, z, namespace);
    if (aValue === null && zValue === null) return 0;
    if (aValue === null) return direction === "asc" ? -1 : 1;
    if (zValue === null) return direction === "asc" ? 1 : -1;
    const result = compareQueryValueStrings(aValue, zValue);
    return direction === "asc" ? result : -result;
  };
}

/**
 * Compares two breakpoint or container values: by unit bucket, then
 * numerically (SPEC §9.5).
 */
export function compareQueryValueStrings(
  aValue: string,
  zValue: string,
): number {
  if (aValue === zValue) return 0;
  const aBucket = bucketOf(aValue);
  const zBucket = bucketOf(zValue);
  if (aBucket !== zBucket) return aBucket < zBucket ? -1 : 1;
  const aNumber = parseFloat(aValue);
  const zNumber = parseFloat(zValue);
  if (Number.isNaN(aNumber) || Number.isNaN(zNumber)) {
    return aValue < zValue ? -1 : 1;
  }
  return aNumber - zNumber;
}

function bucketOf(value: string): string {
  const paren = value.indexOf("(");
  if (paren !== -1 && value.endsWith(")")) return value.slice(0, paren);
  const match = /^-?[0-9.]+(.*)$/.exec(value);
  return match === null ? value : match[1];
}

/** Registers every built-in variant in the order of SPEC §9.5. */
export function registerBuiltinVariants(
  variants: Variants,
  theme: Theme,
): void {
  staticVariant(variants, "*", [":is(& > *)"], { compounds: Compounds.NEVER });
  staticVariant(variants, "**", [":is(& *)"], { compounds: Compounds.NEVER });

  variants.compound(
    "not",
    Compounds.STYLE_RULES | Compounds.AT_RULES,
    (node, variant) => {
      if (variant.modifier) return null;
      return negate(node);
    },
    { compounds: Compounds.STYLE_RULES | Compounds.AT_RULES },
  );

  variants.compound("group", Compounds.STYLE_RULES, (node, variant) => {
    return groupLike(node, variant, "group", "*", theme);
  });
  variants.compound("peer", Compounds.STYLE_RULES, (node, variant) => {
    return groupLike(node, variant, "peer", "~ *", theme);
  });

  staticVariant(variants, "first-letter", ["&::first-letter"], {
    compounds: Compounds.NEVER,
  });
  staticVariant(variants, "first-line", ["&::first-line"], {
    compounds: Compounds.NEVER,
  });
  staticVariant(variants, "marker", [
    "& *::marker",
    "&::marker",
    "& *::-webkit-details-marker",
    "&::-webkit-details-marker",
  ], { compounds: Compounds.NEVER });
  staticVariant(variants, "selection", ["& *::selection", "&::selection"], {
    compounds: Compounds.NEVER,
  });
  staticVariant(variants, "file", ["&::file-selector-button"], {
    compounds: Compounds.NEVER,
  });
  staticVariant(variants, "placeholder", ["&::placeholder"], {
    compounds: Compounds.NEVER,
  });
  staticVariant(variants, "backdrop", ["&::backdrop"], {
    compounds: Compounds.NEVER,
  });
  staticVariant(variants, "details-content", ["&::details-content"], {
    compounds: Compounds.NEVER,
  });

  for (const name of ["before", "after"]) {
    variants.static(name, (node) => {
      node.nodes = [
        styleRule(`&::${name}`, [
          propertyRegistration("--tw-content", '""'),
          decl("content", "var(--tw-content)"),
          ...node.nodes,
        ]),
      ];
    }, { compounds: Compounds.NEVER });
  }

  const pseudos: [string, string][] = [
    ["first", "&:first-child"],
    ["last", "&:last-child"],
    ["only", "&:only-child"],
    ["odd", "&:nth-child(odd)"],
    ["even", "&:nth-child(even)"],
    ["first-of-type", "&:first-of-type"],
    ["last-of-type", "&:last-of-type"],
    ["only-of-type", "&:only-of-type"],
    ["visited", "&:visited"],
    ["target", "&:target"],
    ["open", "&:is([open], :popover-open, :open)"],
    ["default", "&:default"],
    ["checked", "&:checked"],
    ["indeterminate", "&:indeterminate"],
    ["placeholder-shown", "&:placeholder-shown"],
    ["autofill", "&:autofill"],
    ["optional", "&:optional"],
    ["required", "&:required"],
    ["valid", "&:valid"],
    ["invalid", "&:invalid"],
    ["user-valid", "&:user-valid"],
    ["user-invalid", "&:user-invalid"],
    ["in-range", "&:in-range"],
    ["out-of-range", "&:out-of-range"],
    ["read-only", "&:read-only"],
    ["empty", "&:empty"],
    ["focus-within", "&:focus-within"],
  ];
  for (const [name, selector] of pseudos) {
    staticVariant(variants, name, [selector]);
  }

  variants.static("hover", (node) => {
    node.nodes = [
      styleRule("&:hover", [atRule("@media", "(hover: hover)", node.nodes)]),
    ];
  });

  for (
    const name of ["focus", "focus-visible", "active", "enabled", "disabled"]
  ) {
    staticVariant(variants, name, [`&:${name}`]);
  }
  staticVariant(variants, "inert", ["&:is([inert], [inert] *)"]);

  variants.compound("in", Compounds.STYLE_RULES, (node, variant) => {
    if (variant.modifier) return null;
    if (node.kind !== "rule") return null;
    node.selector = `:where(${replaceAmpersand(node.selector, "*")}) &`;
  });

  variants.compound("has", Compounds.STYLE_RULES, (node, variant) => {
    if (variant.modifier) return null;
    if (node.kind !== "rule") return null;
    let selector = replaceAmpersand(node.selector, "*");
    if (/^[>+~][^ ]/.test(selector)) {
      selector = `${selector[0]} ${selector.slice(1)}`;
    }
    node.selector = `&:has(${selector})`;
  });

  variants.functional("aria", (node, variant) => {
    if (variant.value === null || variant.modifier) return null;
    if (variant.value.kind === "named") {
      node.nodes = [
        styleRule(`&[aria-${variant.value.value}="true"]`, node.nodes),
      ];
      return;
    }
    const attribute = quoteAttributeValue(variant.value.value);
    if (attribute === null) return null;
    node.nodes = [styleRule(`&[aria-${attribute}]`, node.nodes)];
  });

  variants.functional("data", (node, variant) => {
    if (variant.value === null || variant.modifier) return null;
    const attribute = quoteAttributeValue(variant.value.value);
    if (attribute === null) return null;
    node.nodes = [styleRule(`&[data-${attribute}]`, node.nodes)];
  });

  const nths: [string, string][] = [
    ["nth", "nth-child"],
    ["nth-last", "nth-last-child"],
    ["nth-of-type", "nth-of-type"],
    ["nth-last-of-type", "nth-last-of-type"],
  ];
  for (const [name, pseudo] of nths) {
    variants.functional(name, (node, variant) => {
      if (variant.value === null || variant.modifier) return null;
      if (
        variant.value.kind === "named" &&
        !isPositiveInteger(variant.value.value)
      ) return null;
      node.nodes = [
        styleRule(`&:${pseudo}(${variant.value.value})`, node.nodes),
      ];
    });
  }

  variants.functional("supports", (node, variant) => {
    if (variant.value === null || variant.modifier) return null;
    const value = variant.value.value;
    let condition: string;
    if (/^[\w-]*\s*\(/.test(value)) {
      condition = value.replace(/\b(and|or|not)\(/g, "$1 (");
    } else if (!value.includes(":")) {
      condition = `(${value}: var(--tw))`;
    } else if (value.startsWith("(") && value.endsWith(")")) {
      condition = value;
    } else {
      condition = `(${value})`;
    }
    node.nodes = [atRule("@supports", condition, node.nodes)];
  }, { compounds: Compounds.AT_RULES });

  const medias: [string, string][] = [
    ["motion-safe", "(prefers-reduced-motion: no-preference)"],
    ["motion-reduce", "(prefers-reduced-motion: reduce)"],
    ["contrast-more", "(prefers-contrast: more)"],
    ["contrast-less", "(prefers-contrast: less)"],
  ];
  for (const [name, query] of medias) {
    staticVariant(variants, name, [`@media ${query}`]);
  }

  // Breakpoints.
  variants.group(() => {
    variants.functional("max", (node, variant) => {
      if (variant.value === null || variant.modifier) return null;
      const value = resolveQueryValue(theme, variant, "--breakpoint");
      if (value === null) return null;
      node.nodes = [atRule("@media", `(width < ${value})`, node.nodes)];
    }, { compounds: Compounds.AT_RULES });
  }, compareQueryValues(theme, "--breakpoint", "desc"));

  variants.group(() => {
    for (const [key, value] of theme.namespace("--breakpoint")) {
      if (key === null || key.includes("--")) continue;
      variants.static(key, (node) => {
        node.nodes = [atRule("@media", `(width >= ${value})`, node.nodes)];
      }, { compounds: Compounds.AT_RULES });
    }
    variants.functional("min", (node, variant) => {
      if (variant.value === null || variant.modifier) return null;
      const value = resolveQueryValue(theme, variant, "--breakpoint");
      if (value === null) return null;
      node.nodes = [atRule("@media", `(width >= ${value})`, node.nodes)];
    }, { compounds: Compounds.AT_RULES });
  }, compareQueryValues(theme, "--breakpoint", "asc"));

  // Containers.
  const containerQuery = (
    node: VariantRule,
    variant: FunctionalVariant,
    operator: string,
  ): null | undefined => {
    if (variant.value === null) return null;
    if (variant.modifier && variant.modifier.kind !== "named") return null;
    const value = resolveQueryValue(theme, variant, "--container");
    if (value === null) return null;
    const name = variant.modifier ? `${variant.modifier.value} ` : "";
    node.nodes = [
      atRule("@container", `${name}(width ${operator} ${value})`, node.nodes),
    ];
  };

  variants.group(() => {
    variants.functional(
      "@max",
      (node, variant) => containerQuery(node, variant, "<"),
      {
        compounds: Compounds.AT_RULES,
      },
    );
  }, compareQueryValues(theme, "--container", "desc"));

  variants.group(() => {
    variants.functional(
      "@",
      (node, variant) => containerQuery(node, variant, ">="),
      {
        compounds: Compounds.AT_RULES,
      },
    );
    variants.functional(
      "@min",
      (node, variant) => containerQuery(node, variant, ">="),
      {
        compounds: Compounds.AT_RULES,
      },
    );
  }, compareQueryValues(theme, "--container", "asc"));

  staticVariant(variants, "portrait", ["@media (orientation: portrait)"]);
  staticVariant(variants, "landscape", ["@media (orientation: landscape)"]);
  staticVariant(variants, "ltr", [
    '&:where(:dir(ltr), [dir="ltr"], [dir="ltr"] *)',
  ]);
  staticVariant(variants, "rtl", [
    '&:where(:dir(rtl), [dir="rtl"], [dir="rtl"] *)',
  ]);
  staticVariant(variants, "dark", ["@media (prefers-color-scheme: dark)"]);
  staticVariant(variants, "starting", ["@starting-style"], {
    compounds: Compounds.NEVER,
  });
  staticVariant(variants, "print", ["@media print"]);
  staticVariant(variants, "forced-colors", ["@media (forced-colors: active)"]);
  staticVariant(variants, "inverted-colors", [
    "@media (inverted-colors: inverted)",
  ]);
  staticVariant(variants, "pointer-none", ["@media (pointer: none)"]);
  staticVariant(variants, "pointer-coarse", ["@media (pointer: coarse)"]);
  staticVariant(variants, "pointer-fine", ["@media (pointer: fine)"]);
  staticVariant(variants, "any-pointer-none", ["@media (any-pointer: none)"]);
  staticVariant(variants, "any-pointer-coarse", [
    "@media (any-pointer: coarse)",
  ]);
  staticVariant(variants, "any-pointer-fine", ["@media (any-pointer: fine)"]);
  staticVariant(variants, "noscript", ["@media (scripting: none)"]);
}

/**
 * Negates each rule produced by an inner variant: the leaf's enclosing style
 * rule and at-rule are negated independently and emitted as siblings.
 */
function negate(node: VariantRule): null | undefined {
  let result: AstNode[] | null = null;
  let failed = false;

  walk([node], (child, { path }) => {
    if (child.kind !== "rule" && child.kind !== "at-rule") return;
    if (child.nodes.length > 0) return;

    const styleRules: string[] = [];
    const atRules: [string, string][] = [];
    for (const ancestor of [...path, child]) {
      if (ancestor.kind === "rule") styleRules.push(ancestor.selector);
      else if (ancestor.kind === "at-rule") {
        atRules.push([ancestor.name, ancestor.params]);
      }
    }
    if (styleRules.length > 1 || atRules.length > 1 || result !== null) {
      failed = true;
      return WalkAction.Stop;
    }

    const rules: AstNode[] = [];
    for (const selector of styleRules) {
      const negated = negateSelector(selector);
      if (negated === null) {
        failed = true;
        return WalkAction.Stop;
      }
      rules.push(styleRule(negated, []));
    }
    for (const [name, params] of atRules) {
      const negated = negateAtRule(name, params);
      if (negated === null) {
        failed = true;
        return WalkAction.Stop;
      }
      rules.push(atRule(negated[0], negated[1], []));
    }
    result = rules;
    return WalkAction.Skip;
  });

  if (failed || result === null) return null;
  const rules: AstNode[] = result;
  if (rules.length === 1) {
    Object.assign(node, rules[0]);
  } else {
    Object.assign(node, styleRule("&", rules));
  }
}

function groupLike(
  node: VariantRule,
  variant: CompoundVariant,
  className: string,
  combinator: string,
  theme: Theme,
): null | undefined {
  if (node.kind !== "rule") return null;
  if (variant.modifier && variant.modifier.kind !== "named") return null;
  if (variant.variant.kind === "arbitrary" && variant.variant.relative) {
    return null;
  }
  if (hasNestedStyleRules(node.nodes)) return null;

  let marker = variant.modifier
    ? `${className}/${variant.modifier.value}`
    : className;
  if (theme.prefix !== null) marker = `${theme.prefix}:${marker}`;
  let selector = replaceAmpersand(node.selector, `:where(.${escape(marker)})`);
  if (segment(selector, ",").length > 1) selector = `:is(${selector})`;
  node.selector = `&:is(${selector} ${combinator})`;
}
