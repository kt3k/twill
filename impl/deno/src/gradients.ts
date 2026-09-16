/**
 * Gradient utilities (SPEC §10.8, "Gradients").
 *
 * @module
 */

import { type AstNode, decl } from "./ast.ts";
import { propertyRegistration } from "./builtin_variants.ts";
import type { CandidateModifier, FunctionalCandidate } from "./candidate.ts";
import { inferDataType } from "./data_types.ts";
import type { Theme } from "./theme.ts";
import { asColor, resolveThemeColor, type Utilities } from "./utilities.ts";
import { isPositiveInteger } from "./utils.ts";

export const GRADIENT_STOPS =
  "var(--tw-gradient-via-stops, var(--tw-gradient-position), var(--tw-gradient-from) var(--tw-gradient-from-position), var(--tw-gradient-to) var(--tw-gradient-to-position))";
export const GRADIENT_VIA_STOPS =
  "var(--tw-gradient-position), var(--tw-gradient-from) var(--tw-gradient-from-position), var(--tw-gradient-via) var(--tw-gradient-via-position), var(--tw-gradient-to) var(--tw-gradient-to-position)";

const INTERPOLATION_METHODS = new Set([
  "srgb",
  "srgb-linear",
  "display-p3",
  "a98-rgb",
  "prophoto-rgb",
  "rec2020",
  "lab",
  "oklab",
  "xyz",
  "xyz-d50",
  "xyz-d65",
  "hsl",
  "hwb",
  "lch",
  "oklch",
]);
const HUE_METHODS = new Set(["longer", "shorter", "increasing", "decreasing"]);

const SIDES: Record<string, string> = {
  t: "top",
  tr: "top right",
  r: "right",
  br: "bottom right",
  b: "bottom",
  bl: "bottom left",
  l: "left",
  tl: "top left",
};

/** Registrations emitted by every stop utility. */
export function gradientRegistrations(): AstNode[] {
  return [
    propertyRegistration("--tw-gradient-position"),
    propertyRegistration("--tw-gradient-from", "#0000", "<color>"),
    propertyRegistration("--tw-gradient-via", "#0000", "<color>"),
    propertyRegistration("--tw-gradient-to", "#0000", "<color>"),
    propertyRegistration("--tw-gradient-stops"),
    propertyRegistration("--tw-gradient-via-stops"),
    propertyRegistration(
      "--tw-gradient-from-position",
      "0%",
      "<length-percentage>",
    ),
    propertyRegistration(
      "--tw-gradient-via-position",
      "50%",
      "<length-percentage>",
    ),
    propertyRegistration(
      "--tw-gradient-to-position",
      "100%",
      "<length-percentage>",
    ),
  ];
}

/** Resolves an interpolation modifier, or null when invalid. */
export function interpolation(
  modifier: CandidateModifier | null,
): string | null {
  if (modifier === null) return "in oklab";
  if (modifier.kind === "arbitrary") return modifier.value;
  if (INTERPOLATION_METHODS.has(modifier.value)) return `in ${modifier.value}`;
  if (HUE_METHODS.has(modifier.value)) return `in oklch ${modifier.value} hue`;
  return null;
}

/** Registers the gradient utilities. */
export function registerGradientUtilities(
  utilities: Utilities,
  theme: Theme,
): void {
  const image = (shape: string) =>
    decl("background-image", `${shape}-gradient(var(--tw-gradient-stops))`);
  const positioned = (
    position: string,
    modifier: CandidateModifier | null,
    shape: string,
  ): AstNode[] | undefined => {
    const method = interpolation(modifier);
    if (method === null) return;
    return [
      decl(
        "--tw-gradient-position",
        position === "" ? method : `${position} ${method}`,
      ),
      image(shape),
    ];
  };

  const linear = (
    candidate: FunctionalCandidate,
    negative: boolean,
    legacy: boolean,
  ): AstNode[] | undefined => {
    if (candidate.value === null) return;
    if (candidate.value.kind === "arbitrary") {
      if (legacy || negative) return;
      const value = candidate.value.value;
      const type = candidate.value.dataType ?? inferDataType(value, ["angle"]);
      if (type === "angle") {
        return positioned(value, candidate.modifier, "linear");
      }
      if (candidate.modifier !== null) return;
      return [decl("background-image", `linear-gradient(${value})`)];
    }
    const named = candidate.value.value;
    if (named.startsWith("to-")) {
      if (negative) return;
      const side = SIDES[named.slice(3)];
      if (side === undefined) return;
      return positioned(`to ${side}`, candidate.modifier, "linear");
    }
    if (legacy || !isPositiveInteger(named)) return;
    const angle = negative ? `calc(${named}deg * -1)` : `${named}deg`;
    return positioned(angle, candidate.modifier, "linear");
  };
  utilities.functional("bg-linear", (c) => linear(c, false, false));
  utilities.functional("-bg-linear", (c) => linear(c, true, false));
  utilities.functional("bg-gradient", (c) => linear(c, false, true));

  utilities.functional("bg-radial", (candidate) => {
    if (candidate.value === null) {
      return positioned("", candidate.modifier, "radial");
    }
    if (candidate.value.kind !== "arbitrary" || candidate.modifier !== null) {
      return;
    }
    return [
      decl("--tw-gradient-position", candidate.value.value),
      image("radial"),
    ];
  });

  const conic = (
    candidate: FunctionalCandidate,
    negative: boolean,
  ): AstNode[] | undefined => {
    if (candidate.value === null) {
      if (negative) return;
      return positioned("", candidate.modifier, "conic");
    }
    if (candidate.value.kind === "arbitrary") {
      if (negative || candidate.modifier !== null) return;
      return [
        decl("--tw-gradient-position", candidate.value.value),
        image("conic"),
      ];
    }
    const named = candidate.value.value;
    if (!isPositiveInteger(named)) return;
    const angle = negative ? `calc(${named}deg * -1)` : `${named}deg`;
    return positioned(`from ${angle}`, candidate.modifier, "conic");
  };
  utilities.functional("bg-conic", (c) => conic(c, false));
  utilities.functional("-bg-conic", (c) => conic(c, true));

  const stop = (
    name: string,
    colorVariable: string,
    positionVariable: string,
    stops: [string, string][],
  ) => {
    const color = (value: string): AstNode[] => [
      ...gradientRegistrations(),
      decl(colorVariable, value),
      ...stops.map(([property, v]) => decl(property, v)),
    ];
    const position = (value: string): AstNode[] => [
      ...gradientRegistrations(),
      decl(positionVariable, value),
    ];
    utilities.functional(name, (candidate) => {
      if (candidate.value === null) return;
      if (candidate.value.kind === "arbitrary") {
        const value = candidate.value.value;
        const type = candidate.value.dataType ??
          inferDataType(value, ["color", "length", "percentage"]);
        if (type === "length" || type === "percentage") {
          if (candidate.modifier !== null) return;
          return position(value);
        }
        const resolved = asColor(value, candidate.modifier, theme);
        if (resolved === null) return;
        return color(resolved);
      }
      const themeColor = resolveThemeColor(candidate, [
        "--background-color",
        "--color",
      ], theme);
      if (themeColor !== null) return color(themeColor);
      if (candidate.modifier !== null) return;
      const named = candidate.value.value;
      if (named.endsWith("%") && isPositiveInteger(named.slice(0, -1))) {
        return position(named);
      }
    });
  };
  stop("from", "--tw-gradient-from", "--tw-gradient-from-position", [
    ["--tw-gradient-stops", GRADIENT_STOPS],
  ]);
  stop("via", "--tw-gradient-via", "--tw-gradient-via-position", [
    ["--tw-gradient-via-stops", GRADIENT_VIA_STOPS],
    ["--tw-gradient-stops", "var(--tw-gradient-via-stops)"],
  ]);
  stop("to", "--tw-gradient-to", "--tw-gradient-to-position", [
    ["--tw-gradient-stops", GRADIENT_STOPS],
  ]);
}
