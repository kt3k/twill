/**
 * Filter and backdrop-filter utilities (SPEC §10.8, "Filters").
 *
 * @module
 */

import { type AstNode, decl } from "./ast.ts";
import { propertyRegistration } from "./builtin_variants.ts";
import { inferDataType } from "./data_types.ts";
import { replaceShadowColors } from "./shadows.ts";
import type { Theme } from "./theme.ts";
import {
  asColor,
  functionalUtility,
  resolveThemeColor,
  staticUtility,
  type Utilities,
} from "./utilities.ts";
import { isPositiveInteger, segment } from "./utils.ts";

const FILTER_NAMES = [
  "blur",
  "brightness",
  "contrast",
  "grayscale",
  "hue-rotate",
  "invert",
  "saturate",
  "sepia",
  "drop-shadow",
];
const BACKDROP_NAMES = [
  "blur",
  "brightness",
  "contrast",
  "grayscale",
  "hue-rotate",
  "invert",
  "opacity",
  "saturate",
  "sepia",
];

export const FILTER = FILTER_NAMES.map((n) => `var(--tw-${n},)`).join(" ");
export const BACKDROP_FILTER = BACKDROP_NAMES.map((n) =>
  `var(--tw-backdrop-${n},)`
).join(" ");

function registrations(backdrop: boolean): AstNode[] {
  return backdrop
    ? BACKDROP_NAMES.map((n) => propertyRegistration(`--tw-backdrop-${n}`))
    : FILTER_NAMES.map((n) => propertyRegistration(`--tw-${n}`));
}

function filterDeclarations(backdrop: boolean): AstNode[] {
  return backdrop
    ? [
      decl("-webkit-backdrop-filter", BACKDROP_FILTER),
      decl("backdrop-filter", BACKDROP_FILTER),
    ]
    : [decl("filter", FILTER)];
}

/** Registers the filter and backdrop-filter utilities. */
export function registerFilterUtilities(
  utilities: Utilities,
  theme: Theme,
): void {
  const stat = (name: string, nodes: () => AstNode[]) =>
    staticUtility(utilities, name, nodes);
  const fn = (
    root: string,
    desc: Parameters<typeof functionalUtility>[3],
  ) => functionalUtility(utilities, theme, root, desc);
  const percentage = (value: { value: string }) =>
    isPositiveInteger(value.value) ? `${value.value}%` : null;
  const degrees = (value: { value: string }) =>
    isPositiveInteger(value.value) ? `${value.value}deg` : null;

  for (const backdrop of [false, true]) {
    const prefix = backdrop ? "backdrop-" : "";
    const variable = backdrop ? "--tw-backdrop-" : "--tw-";
    const keys = (name: string) =>
      backdrop ? [`--backdrop-${name}`, `--${name}`] : [`--${name}`];
    const emit = (name: string, value: string): AstNode[] => [
      ...registrations(backdrop),
      decl(`${variable}${name}`, value),
      ...filterDeclarations(backdrop),
    ];

    stat(
      `${prefix}filter-none`,
      () =>
        backdrop
          ? [
            decl("-webkit-backdrop-filter", "none"),
            decl("backdrop-filter", "none"),
          ]
          : [decl("filter", "none")],
    );
    stat(`${prefix}filter`, () => [
      ...registrations(backdrop),
      ...filterDeclarations(backdrop),
    ]);
    fn(`${prefix}filter`, {
      handle: (value) =>
        backdrop
          ? [
            decl("-webkit-backdrop-filter", value),
            decl("backdrop-filter", value),
          ]
          : [decl("filter", value)],
    });

    fn(`${prefix}blur`, {
      themeKeys: keys("blur"),
      handle: (value) => emit("blur", `blur(${value})`),
    });
    stat(`${prefix}blur-none`, () => emit("blur", ""));
    for (const name of ["brightness", "contrast", "saturate"]) {
      fn(`${prefix}${name}`, {
        themeKeys: keys(name),
        handleBareValue: percentage,
        handle: (value) => emit(name, `${name}(${value})`),
      });
    }
    for (const name of ["grayscale", "invert", "sepia"]) {
      fn(`${prefix}${name}`, {
        themeKeys: keys(name),
        defaultValue: "100%",
        handleBareValue: percentage,
        handle: (value) => emit(name, `${name}(${value})`),
      });
    }
    fn(`${prefix}hue-rotate`, {
      themeKeys: keys("hue-rotate"),
      supportsNegative: true,
      handleBareValue: degrees,
      handle: (value) => emit("hue-rotate", `hue-rotate(${value})`),
    });
    if (backdrop) {
      fn("backdrop-opacity", {
        themeKeys: ["--backdrop-opacity", "--opacity"],
        handleBareValue: percentage,
        handle: (value) => emit("opacity", `opacity(${value})`),
      });
      continue;
    }

    stat("drop-shadow-none", () => emit("drop-shadow", ""));
    utilities.functional("drop-shadow", (candidate) => {
      const color = (value: string): AstNode[] => [
        propertyRegistration("--tw-drop-shadow-color"),
        decl("--tw-drop-shadow-color", value),
      ];
      const wrap = (value: string): AstNode[] | undefined => {
        let failed = false;
        const replaced = replaceShadowColors(value, (c) => {
          const resolved = asColor(c, candidate.modifier, theme);
          if (resolved === null) {
            failed = true;
            return c;
          }
          return `var(--tw-drop-shadow-color, ${resolved})`;
        });
        if (failed) return;
        const shadows = segment(replaced, ",").map((s) =>
          `drop-shadow(${s.trim()})`
        );
        return emit("drop-shadow", shadows.join(" "));
      };
      if (candidate.value === null) {
        const value = theme.resolveValue(null, ["--drop-shadow"]);
        if (value === null) return;
        return wrap(value);
      }
      if (candidate.value.kind === "arbitrary") {
        const value = candidate.value.value;
        const type = candidate.value.dataType ??
          inferDataType(value, ["color"]);
        if (type === "color") {
          const resolved = asColor(value, candidate.modifier, theme);
          if (resolved === null) return;
          return color(resolved);
        }
        return wrap(value);
      }
      const themeColor = resolveThemeColor(candidate, [
        "--drop-shadow-color",
        "--color",
      ], theme);
      if (themeColor !== null) return color(themeColor);
      const value = theme.resolveValue(candidate.value.value, [
        "--drop-shadow",
      ]);
      if (value === null) return;
      return wrap(value);
    });
  }
}
