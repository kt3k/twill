/**
 * Space, divider, and outline utilities (SPEC §10.8, "Space, dividers, and
 * outlines").
 *
 * @module
 */

import { type AstNode, atRule, decl, styleRule } from "./ast.ts";
import { propertyRegistration } from "./builtin_variants.ts";
import { inferDataType } from "./data_types.ts";
import type { Theme } from "./theme.ts";
import {
  asColor,
  colorUtility,
  functionalUtility,
  resolveThemeColor,
  spacingUtility,
  staticUtility,
  type Utilities,
} from "./utilities.ts";
import { isPositiveInteger } from "./utils.ts";

export const CHILDREN_SELECTOR = ":where(& > :not(:last-child))";

function children(nodes: AstNode[]): AstNode {
  return styleRule(CHILDREN_SELECTOR, nodes);
}

/** Registers the space, divide, and outline utilities. */
export function registerDividerUtilities(
  utilities: Utilities,
  theme: Theme,
): void {
  const stat = (name: string, nodes: () => AstNode[]) =>
    staticUtility(utilities, name, nodes);
  const pixels = (value: { value: string }) =>
    isPositiveInteger(value.value) ? `${value.value}px` : null;

  for (const axis of ["x", "y"]) {
    const reverse = `--tw-space-${axis}-reverse`;
    const [start, end] = axis === "x"
      ? ["margin-inline-start", "margin-inline-end"]
      : ["margin-block-start", "margin-block-end"];
    spacingUtility(
      utilities,
      theme,
      `space-${axis}`,
      ["--spacing"],
      (value) => [
        propertyRegistration(reverse, "0"),
        children([
          decl(reverse, "0"),
          decl(start, `calc(${value} * var(${reverse}))`),
          decl(end, `calc(${value} * calc(1 - var(${reverse})))`),
        ]),
      ],
      { supportsNegative: true },
    );
    stat(`space-${axis}-reverse`, () => [
      propertyRegistration(reverse, "0"),
      children([decl(reverse, "1")]),
    ]);
  }

  for (const axis of ["x", "y"]) {
    const reverse = `--tw-divide-${axis}-reverse`;
    const [style, start, end] = axis === "x"
      ? [
        "border-inline-style",
        "border-inline-start-width",
        "border-inline-end-width",
      ]
      : [
        "border-block-style",
        "border-block-start-width",
        "border-block-end-width",
      ];
    const width = (value: string): AstNode[] => [
      propertyRegistration(reverse, "0"),
      propertyRegistration("--tw-border-style", "solid"),
      children([
        decl(reverse, "0"),
        decl(style, "var(--tw-border-style)"),
        decl(start, `calc(${value} * var(${reverse}))`),
        decl(end, `calc(${value} * calc(1 - var(${reverse})))`),
      ]),
    ];
    utilities.functional(`divide-${axis}`, (candidate) => {
      if (candidate.modifier !== null) return;
      if (candidate.value === null) {
        return width(theme.get(["--default-border-width"]) ?? "1px");
      }
      if (candidate.value.kind === "arbitrary") {
        const value = candidate.value.value;
        const type = candidate.value.dataType ??
          inferDataType(value, ["length", "line-width"]);
        if (type === "length" || type === "line-width") return width(value);
        return;
      }
      const themeWidth = theme.resolve(candidate.value.value, [
        "--divide-width",
      ]);
      if (themeWidth !== null) return width(themeWidth);
      const bare = pixels(candidate.value);
      if (bare !== null) return width(bare);
    });
    stat(`divide-${axis}-reverse`, () => [
      propertyRegistration(reverse, "0"),
      children([decl(reverse, "1")]),
    ]);
  }
  colorUtility(utilities, theme, "divide", {
    themeKeys: ["--divide-color", "--color"],
    handle: (value) => [children([decl("border-color", value)])],
  });
  for (const style of ["solid", "dashed", "dotted", "double", "none"]) {
    stat(`divide-${style}`, () => [
      children([decl("--tw-border-style", style), decl("border-style", style)]),
    ]);
  }

  const outlineWidth = (value: string): AstNode[] => [
    propertyRegistration("--tw-outline-style", "solid"),
    decl("outline-style", "var(--tw-outline-style)"),
    decl("outline-width", value),
  ];
  utilities.functional("outline", (candidate) => {
    if (candidate.value === null) {
      if (candidate.modifier !== null) return;
      return outlineWidth("1px");
    }
    if (candidate.value.kind === "arbitrary") {
      const value = candidate.value.value;
      const type = candidate.value.dataType ??
        inferDataType(value, ["color", "length", "line-width"]);
      if (type === "color") {
        const resolved = asColor(value, candidate.modifier, theme);
        if (resolved === null) return;
        return [decl("outline-color", resolved)];
      }
      if (type === "length" || type === "line-width") {
        if (candidate.modifier !== null) return;
        return outlineWidth(value);
      }
      return;
    }
    const color = resolveThemeColor(candidate, [
      "--outline-color",
      "--color",
    ], theme);
    if (color !== null) return [decl("outline-color", color)];
    if (candidate.modifier !== null) return;
    const themeWidth = theme.resolve(candidate.value.value, [
      "--outline-width",
    ]);
    if (themeWidth !== null) return outlineWidth(themeWidth);
    const bare = pixels(candidate.value);
    if (bare !== null) return outlineWidth(bare);
  });
  stat("outline-none", () => [
    decl("--tw-outline-style", "none"),
    decl("outline-style", "none"),
  ]);
  stat("outline-hidden", () => [
    decl("--tw-outline-style", "none"),
    decl("outline-style", "none"),
    atRule("@media", "(forced-colors: active)", [
      decl("outline", "2px solid transparent"),
      decl("outline-offset", "2px"),
    ]),
  ]);
  for (const style of ["solid", "dashed", "dotted", "double"]) {
    stat(`outline-${style}`, () => [
      decl("--tw-outline-style", style),
      decl("outline-style", style),
    ]);
  }
  functionalUtility(utilities, theme, "outline-offset", {
    themeKeys: ["--outline-offset"],
    supportsNegative: true,
    handleBareValue: pixels,
    handle: (value) => [decl("outline-offset", value)],
  });
}
