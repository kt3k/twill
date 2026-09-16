/**
 * Shadow and ring utilities (SPEC §10.8, "Shadows and rings").
 *
 * @module
 */

import { type AstNode, decl } from "./ast.ts";
import { propertyRegistration } from "./builtin_variants.ts";
import { inferDataType } from "./data_types.ts";
import type { Theme } from "./theme.ts";
import { asColor, resolveThemeColor, type Utilities } from "./utilities.ts";
import { isPositiveInteger, segment } from "./utils.ts";

/** The composed `box-shadow` value. */
export const BOX_SHADOW =
  "var(--tw-inset-shadow), var(--tw-inset-ring-shadow), var(--tw-ring-offset-shadow), var(--tw-ring-shadow), var(--tw-shadow)";

const SHADOW_KEYWORDS = new Set([
  "inset",
  "inherit",
  "initial",
  "revert",
  "unset",
]);
const LENGTH_TOKEN = /^(?:\d|\.|-[\d.])/;

/**
 * Rewrites the color of every shadow in a `box-shadow` value. Shadows with
 * fewer than two length tokens are left unchanged. With `inset`, shadows
 * without the `inset` keyword are prefixed with it.
 */
export function replaceShadowColors(
  value: string,
  fn: (color: string) => string,
  inset = false,
): string {
  return segment(value, ",").map((shadow) => {
    const tokens = segment(shadow.replace(/\s+/g, " ").trim(), " ").filter((
      t,
    ) => t !== "");
    let lengths = 0;
    let colorIndex = -1;
    let hasInset = false;
    for (let i = 0; i < tokens.length; i++) {
      const token = tokens[i];
      if (SHADOW_KEYWORDS.has(token)) {
        if (token === "inset") hasInset = true;
      } else if (LENGTH_TOKEN.test(token)) {
        lengths++;
      } else if (colorIndex === -1) {
        colorIndex = i;
      }
    }
    if (lengths < 2) return shadow.trim();
    if (colorIndex === -1) tokens.push(fn("currentcolor"));
    else tokens[colorIndex] = fn(tokens[colorIndex]);
    if (inset && !hasInset) tokens.unshift("inset");
    return tokens.join(" ");
  }).join(", ");
}

/** The registrations emitted by every shadow and ring utility. */
export function shadowRegistrations(): AstNode[] {
  return [
    propertyRegistration("--tw-shadow", "0 0 #0000"),
    propertyRegistration("--tw-shadow-color"),
    propertyRegistration("--tw-inset-shadow", "0 0 #0000"),
    propertyRegistration("--tw-inset-shadow-color"),
    propertyRegistration("--tw-ring-color"),
    propertyRegistration("--tw-ring-shadow", "0 0 #0000"),
    propertyRegistration("--tw-inset-ring-color"),
    propertyRegistration("--tw-inset-ring-shadow", "0 0 #0000"),
    propertyRegistration("--tw-ring-inset"),
    propertyRegistration("--tw-ring-offset-width", "0px"),
    propertyRegistration("--tw-ring-offset-color", "#fff"),
    propertyRegistration("--tw-ring-offset-shadow", "0 0 #0000"),
  ];
}

const COLOR_KEYS = ["--box-shadow-color", "--color"];
const RING_COLOR_KEYS = ["--ring-color", "--color"];

/** Registers the shadow and ring utilities. */
export function registerShadowUtilities(
  utilities: Utilities,
  theme: Theme,
): void {
  const withBoxShadow = (variable: string, value: string): AstNode[] => [
    ...shadowRegistrations(),
    decl(variable, value),
    decl("box-shadow", BOX_SHADOW),
  ];
  const colorOnly = (variable: string, value: string): AstNode[] => [
    ...shadowRegistrations(),
    decl(variable, value),
  ];

  const shadow = (
    root: string,
    namespace: string,
    variable: string,
    colorVariable: string,
    inset: boolean,
  ) => {
    utilities.static(
      `${root}-none`,
      () => withBoxShadow(variable, "0 0 #0000"),
    );
    utilities.functional(root, (candidate) => {
      const wrap = (value: string, insetArbitrary: boolean) => {
        let failed = false;
        const replaced = replaceShadowColors(value, (color) => {
          const resolved = asColor(color, candidate.modifier, theme);
          if (resolved === null) {
            failed = true;
            return color;
          }
          return `var(${colorVariable}, ${resolved})`;
        }, insetArbitrary);
        if (failed) return;
        return withBoxShadow(variable, replaced);
      };
      if (candidate.value === null) {
        const value = theme.resolveValue(null, [namespace]);
        if (value === null) return;
        return wrap(value, false);
      }
      if (candidate.value.kind === "arbitrary") {
        const value = candidate.value.value;
        const type = candidate.value.dataType ??
          inferDataType(value, ["color"]);
        if (type === "color") {
          const resolved = asColor(value, candidate.modifier, theme);
          if (resolved === null) return;
          return colorOnly(colorVariable, resolved);
        }
        return wrap(value, inset);
      }
      const color = resolveThemeColor(candidate, COLOR_KEYS, theme);
      if (color !== null) return colorOnly(colorVariable, color);
      const value = theme.resolveValue(candidate.value.value, [namespace]);
      if (value === null) return;
      return wrap(value, false);
    });
  };
  shadow("shadow", "--shadow", "--tw-shadow", "--tw-shadow-color", false);
  shadow(
    "inset-shadow",
    "--inset-shadow",
    "--tw-inset-shadow",
    "--tw-inset-shadow-color",
    true,
  );

  const ring = (
    root: string,
    variable: string,
    colorVariable: string,
    shadowFor: (width: string) => string,
  ) => {
    utilities.functional(root, (candidate) => {
      const width = (value: string) =>
        withBoxShadow(variable, shadowFor(value));
      if (candidate.value === null) {
        if (candidate.modifier !== null) return;
        return width(theme.get(["--default-ring-width"]) ?? "1px");
      }
      if (candidate.value.kind === "arbitrary") {
        const value = candidate.value.value;
        const type = candidate.value.dataType ??
          inferDataType(value, ["color", "length", "line-width"]);
        if (type === "color") {
          const resolved = asColor(value, candidate.modifier, theme);
          if (resolved === null) return;
          return colorOnly(colorVariable, resolved);
        }
        if (type === "length" || type === "line-width") {
          if (candidate.modifier !== null) return;
          return width(value);
        }
        return;
      }
      const color = resolveThemeColor(candidate, RING_COLOR_KEYS, theme);
      if (color !== null) return colorOnly(colorVariable, color);
      if (candidate.modifier !== null) return;
      const themeWidth = theme.resolve(candidate.value.value, ["--ring-width"]);
      if (themeWidth !== null) return width(themeWidth);
      if (isPositiveInteger(candidate.value.value)) {
        return width(`${candidate.value.value}px`);
      }
    });
  };
  ring(
    "ring",
    "--tw-ring-shadow",
    "--tw-ring-color",
    (w) =>
      `var(--tw-ring-inset,) 0 0 0 calc(${w} + var(--tw-ring-offset-width)) var(--tw-ring-color, currentcolor)`,
  );
  ring(
    "inset-ring",
    "--tw-inset-ring-shadow",
    "--tw-inset-ring-color",
    (w) => `inset 0 0 0 ${w} var(--tw-inset-ring-color, currentcolor)`,
  );
  utilities.static("ring-inset", () => colorOnly("--tw-ring-inset", "inset"));

  utilities.functional("ring-offset", (candidate) => {
    const width = (value: string): AstNode[] => [
      ...shadowRegistrations(),
      decl("--tw-ring-offset-width", value),
      decl(
        "--tw-ring-offset-shadow",
        "var(--tw-ring-inset,) 0 0 0 var(--tw-ring-offset-width) var(--tw-ring-offset-color)",
      ),
    ];
    if (candidate.value === null) return;
    if (candidate.value.kind === "arbitrary") {
      const value = candidate.value.value;
      const type = candidate.value.dataType ??
        inferDataType(value, ["color", "length"]);
      if (type === "color") {
        const resolved = asColor(value, candidate.modifier, theme);
        if (resolved === null) return;
        return colorOnly("--tw-ring-offset-color", resolved);
      }
      if (type === "length") {
        if (candidate.modifier !== null) return;
        return width(value);
      }
      return;
    }
    const color = resolveThemeColor(candidate, [
      "--ring-offset-color",
      "--color",
    ], theme);
    if (color !== null) return colorOnly("--tw-ring-offset-color", color);
    if (candidate.modifier !== null) return;
    const themeWidth = theme.resolve(candidate.value.value, [
      "--ring-offset-width",
    ]);
    if (themeWidth !== null) return width(themeWidth);
    if (isPositiveInteger(candidate.value.value)) {
      return width(`${candidate.value.value}px`);
    }
  });
}
