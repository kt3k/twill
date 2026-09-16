/**
 * Theme functions: `--spacing(...)`, `--alpha(...)`, `--theme(...)`, and the
 * legacy `theme(...)` (SPEC §6.11).
 *
 * @module
 */

import { type AstNode, walk } from "./ast.ts";
import { withAlpha } from "./color.ts";
import type { DesignSystem } from "./design_system.ts";
import { TwillError } from "./error.ts";
import { Features } from "./features.ts";
import { segment, unquote } from "./utils.ts";
import {
  parseValue,
  separator,
  toCss,
  type ValueAstNode,
  walkValue,
  word,
} from "./value_parser.ts";

const THEME_FUNCTION_PATTERN =
  /(^|[^\w-])(?:--spacing|--alpha|--theme|theme)\(/;

/** Whether a value or params string contains a theme function call. */
export function hasThemeFunction(value: string): boolean {
  return THEME_FUNCTION_PATTERN.test(value);
}

const AT_RULES_WITH_PARAMS = new Set([
  "@media",
  "@custom-media",
  "@container",
  "@supports",
]);

/**
 * Substitutes theme functions in declaration values and in the params of
 * `@media`, `@custom-media`, `@container`, and `@supports`. Returns the
 * feature flags.
 */
export function substituteFunctions(ast: AstNode[], ds: DesignSystem): number {
  let features = Features.NONE;
  walk(ast, (node) => {
    if (node.kind === "declaration") {
      if (node.value !== undefined && hasThemeFunction(node.value)) {
        node.value = substituteFunctionsInValue(node.value, ds, false);
        features |= Features.THEME_FUNCTION;
      }
      return;
    }
    if (node.kind === "at-rule" && AT_RULES_WITH_PARAMS.has(node.name)) {
      if (hasThemeFunction(node.params)) {
        node.params = substituteFunctionsInValue(node.params, ds, true);
        features |= Features.THEME_FUNCTION;
      }
    }
  });
  return features;
}

/**
 * Substitutes theme functions in a single value. `inAtRule` forces inline
 * resolution for `--theme(...)`.
 */
export function substituteFunctionsInValue(
  value: string,
  ds: DesignSystem,
  inAtRule: boolean,
): string {
  const ast = parseValue(value);
  walkValue(ast, (node, { replaceWith }) => {
    if (node.kind !== "function") return;
    switch (node.value) {
      case "--spacing": {
        replaceWith(word(spacing(node.nodes, ds)));
        return true;
      }
      case "--alpha": {
        replaceWith(word(alpha(node.nodes)));
        return true;
      }
      case "--theme": {
        replaceWith(word(themeFunction(node.nodes, ds, inAtRule)));
        return true;
      }
      case "theme": {
        replaceWith(word(legacyTheme(node.nodes, ds)));
        return true;
      }
    }
  });
  return toCss(ast);
}

function args(nodes: ValueAstNode[]): string[] {
  return segment(toCss(nodes), ",").map((a) => a.trim());
}

function spacing(nodes: ValueAstNode[], ds: DesignSystem): string {
  const list = args(nodes);
  if (list.length !== 1 || list[0] === "") {
    throw new TwillError(
      `The --spacing(…) function requires exactly one argument, but received ${
        list.filter((a) => a !== "").length
      }.`,
    );
  }
  const multiplier = ds.theme.resolve(null, ["--spacing"]);
  if (multiplier === null) {
    throw new TwillError(
      "The --spacing(…) function requires that the `--spacing` theme variable exists, but it was not found.",
    );
  }
  const n = list[0];
  if (n === "0") return "0px";
  if (n === "1") return multiplier;
  return `calc(${multiplier} * ${n})`;
}

function alpha(nodes: ValueAstNode[]): string {
  const text = toCss(nodes);
  const list = segment(text, ",");
  if (list.length !== 1) {
    throw new TwillError(
      `The --alpha(…) function requires exactly one argument in the form \`<color> / <alpha>\`, but received \`${text}\`.`,
    );
  }
  const parts = segment(text, "/").map((p) => p.trim());
  if (parts.length !== 2 || parts[0] === "" || parts[1] === "") {
    throw new TwillError(
      `The --alpha(…) function requires a color and an alpha value separated by \`/\`, but received \`${text}\`.`,
    );
  }
  return withAlpha(parts[0], parts[1]);
}

function themeFunction(
  nodes: ValueAstNode[],
  ds: DesignSystem,
  inAtRule: boolean,
): string {
  const list = args(nodes);
  let key = list[0] ?? "";
  let inline = inAtRule;
  if (key.endsWith(" inline")) {
    inline = true;
    key = key.slice(0, -" inline".length).trim();
  }
  const fallback = list.length > 1 ? list.slice(1).join(", ") : null;
  if (!key.startsWith("--")) {
    throw new TwillError(
      `The --theme(…) function can only be used with CSS variables from your theme, but received \`${key}\`.`,
    );
  }
  const resolved = ds.theme.resolveThemeValue(key, inline);
  if (resolved === null) {
    if (fallback !== null) return fallback;
    throw new TwillError(
      `Could not resolve value for theme function: \`--theme(${key})\`.`,
    );
  }
  if (fallback === null) return resolved;
  if (fallback === "initial") return resolved;
  if (resolved === "initial") return fallback;
  if (
    resolved.startsWith("var(") || resolved.startsWith("theme(") ||
    resolved.startsWith("--theme(")
  ) {
    return injectFallback(resolved, fallback);
  }
  return resolved;
}

function legacyTheme(nodes: ValueAstNode[], ds: DesignSystem): string {
  const list = args(nodes);
  let key = list[0] ?? "";
  key = unquote(key) ?? key;
  const fallback = list.length > 1 ? list.slice(1).join(", ") : null;
  const resolved = ds.theme.resolveThemeValue(key, true);
  if (resolved === null) {
    if (fallback !== null) return fallback;
    throw new TwillError(
      `Could not resolve value for theme function: \`theme(${key})\`.`,
    );
  }
  return resolved;
}

function isThemeReference(name: string): boolean {
  return name === "var" || name === "theme" || name === "--theme";
}

/**
 * Injects `fallback` into the innermost `var(...)`, `theme(...)`, or
 * `--theme(...)` call that has no fallback or whose fallback is `initial`.
 */
export function injectFallback(value: string, fallback: string): string {
  const ast = parseValue(value);
  inject(ast, fallback);
  return toCss(ast);
}

function inject(nodes: ValueAstNode[], fallback: string): boolean {
  for (const node of nodes) {
    if (node.kind !== "function") continue;
    if (inject(node.nodes, fallback)) return true;
    if (!isThemeReference(node.value)) continue;
    const comma = node.nodes.findIndex((n) =>
      n.kind === "separator" && n.value === ","
    );
    if (comma === -1) {
      node.nodes.push(separator(","), separator(" "), word(fallback));
      return true;
    }
    const existing = toCss(node.nodes.slice(comma + 1)).trim();
    if (existing === "initial") {
      node.nodes = [
        ...node.nodes.slice(0, comma + 1),
        separator(" "),
        word(fallback),
      ];
      return true;
    }
  }
  return false;
}
