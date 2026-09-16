/**
 * Typography, background, layout, table, scrolling, and interactivity
 * extensions (SPEC §10.8).
 *
 * @module
 */

import { type AstNode, atRule, decl } from "./ast.ts";
import {
  compareQueryValueStrings,
  propertyRegistration,
} from "./builtin_variants.ts";
import { inferDataType } from "./data_types.ts";
import { replaceShadowColors } from "./shadows.ts";
import type { Theme } from "./theme.ts";
import {
  asColor,
  bareInteger,
  functionalUtility,
  resolveThemeColor,
  spacingUtility,
  staticUtility,
  type Utilities,
} from "./utilities.ts";
import { isPositiveInteger } from "./utils.ts";

const BLEND_MODES = [
  "normal",
  "multiply",
  "screen",
  "overlay",
  "darken",
  "lighten",
  "color-dodge",
  "color-burn",
  "hard-light",
  "soft-light",
  "difference",
  "exclusion",
  "hue",
  "saturation",
  "color",
  "luminosity",
];

const NUMERIC_VARIABLES = [
  "--tw-ordinal",
  "--tw-slashed-zero",
  "--tw-numeric-figure",
  "--tw-numeric-spacing",
  "--tw-numeric-fraction",
];

/** Registers the extension utilities. */
export function registerExtraUtilities(
  utilities: Utilities,
  theme: Theme,
): void {
  const stat = (name: string, nodes: () => AstNode[]) =>
    staticUtility(utilities, name, nodes);
  const fn = (
    root: string,
    desc: Parameters<typeof functionalUtility>[3],
  ) => functionalUtility(utilities, theme, root, desc);
  const single = (property: string) => (value: string) => [
    decl(property, value),
  ];
  const pixels = (value: { value: string }) =>
    isPositiveInteger(value.value) ? `${value.value}px` : null;

  // Typography.
  stat("line-clamp-none", () => [
    decl("overflow", "visible"),
    decl("display", "block"),
    decl("-webkit-box-orient", "horizontal"),
    decl("-webkit-line-clamp", "unset"),
  ]);
  fn("line-clamp", {
    themeKeys: ["--line-clamp"],
    handleBareValue: bareInteger,
    handle: (value) => [
      decl("overflow", "hidden"),
      decl("display", "-webkit-box"),
      decl("-webkit-box-orient", "vertical"),
      decl("-webkit-line-clamp", value),
    ],
  });

  for (const style of ["solid", "double", "dotted", "dashed", "wavy"]) {
    stat(`decoration-${style}`, () => [decl("text-decoration-style", style)]);
  }
  stat("decoration-from-font", () => [
    decl("text-decoration-thickness", "from-font"),
  ]);
  stat("decoration-auto", () => [decl("text-decoration-thickness", "auto")]);
  utilities.functional("decoration", (candidate) => {
    if (candidate.value === null) return;
    if (candidate.value.kind === "arbitrary") {
      const value = candidate.value.value;
      const type = candidate.value.dataType ??
        inferDataType(value, ["color", "length", "percentage"]);
      if (type === "length" || type === "percentage") {
        if (candidate.modifier !== null) return;
        return [decl("text-decoration-thickness", value)];
      }
      const resolved = asColor(value, candidate.modifier, theme);
      if (resolved === null) return;
      return [decl("text-decoration-color", resolved)];
    }
    const color = resolveThemeColor(candidate, [
      "--text-decoration-color",
      "--color",
    ], theme);
    if (color !== null) return [decl("text-decoration-color", color)];
    if (candidate.modifier !== null) return;
    const thickness = theme.resolve(candidate.value.value, [
      "--text-decoration-thickness",
    ]);
    if (thickness !== null) {
      return [decl("text-decoration-thickness", thickness)];
    }
    const bare = pixels(candidate.value);
    if (bare !== null) return [decl("text-decoration-thickness", bare)];
  });

  for (const value of ["none", "manual", "auto"]) {
    stat(`hyphens-${value}`, () => [
      decl("-webkit-hyphens", value),
      decl("hyphens", value),
    ]);
  }

  stat("normal-nums", () => [decl("font-variant-numeric", "normal")]);
  const numeric = (name: string, variable: string) =>
    stat(name, () => [
      ...NUMERIC_VARIABLES.map((v) => propertyRegistration(v)),
      decl(variable, name),
      decl(
        "font-variant-numeric",
        NUMERIC_VARIABLES.map((v) => `var(${v},)`).join(" "),
      ),
    ]);
  numeric("ordinal", "--tw-ordinal");
  numeric("slashed-zero", "--tw-slashed-zero");
  numeric("lining-nums", "--tw-numeric-figure");
  numeric("oldstyle-nums", "--tw-numeric-figure");
  numeric("proportional-nums", "--tw-numeric-spacing");
  numeric("tabular-nums", "--tw-numeric-spacing");
  numeric("diagonal-fractions", "--tw-numeric-fraction");
  numeric("stacked-fractions", "--tw-numeric-fraction");

  for (
    const value of [
      "baseline",
      "top",
      "middle",
      "bottom",
      "text-top",
      "text-bottom",
      "sub",
      "super",
    ]
  ) {
    stat(`align-${value}`, () => [decl("vertical-align", value)]);
  }
  fn("align", { handle: single("vertical-align") });

  for (
    const value of [
      "ultra-condensed",
      "extra-condensed",
      "condensed",
      "semi-condensed",
      "normal",
      "semi-expanded",
      "expanded",
      "extra-expanded",
      "ultra-expanded",
    ]
  ) {
    stat(`font-stretch-${value}`, () => [decl("font-stretch", value)]);
  }
  fn("font-stretch", {
    themeKeys: ["--font-stretch"],
    handleBareValue: (value) =>
      value.value.endsWith("%") && isPositiveInteger(value.value.slice(0, -1))
        ? value.value
        : null,
    handle: single("font-stretch"),
  });

  stat("text-shadow-none", () => [decl("text-shadow", "none")]);
  utilities.functional("text-shadow", (candidate) => {
    const color = (value: string): AstNode[] => [
      propertyRegistration("--tw-text-shadow-color"),
      decl("--tw-text-shadow-color", value),
    ];
    const wrap = (value: string): AstNode[] | undefined => {
      let failed = false;
      const replaced = replaceShadowColors(value, (c) => {
        const resolved = asColor(c, candidate.modifier, theme);
        if (resolved === null) {
          failed = true;
          return c;
        }
        return `var(--tw-text-shadow-color, ${resolved})`;
      });
      if (failed) return;
      return [decl("text-shadow", replaced)];
    };
    if (candidate.value === null) {
      const value = theme.resolveValue(null, ["--text-shadow"]);
      if (value === null) return;
      return wrap(value);
    }
    if (candidate.value.kind === "arbitrary") {
      const value = candidate.value.value;
      const type = candidate.value.dataType ?? inferDataType(value, ["color"]);
      if (type === "color") {
        const resolved = asColor(value, candidate.modifier, theme);
        if (resolved === null) return;
        return color(resolved);
      }
      return wrap(value);
    }
    const themeColor = resolveThemeColor(candidate, [
      "--text-shadow-color",
      "--color",
    ], theme);
    if (themeColor !== null) return color(themeColor);
    const value = theme.resolveValue(candidate.value.value, ["--text-shadow"]);
    if (value === null) return;
    return wrap(value);
  });

  stat("wrap-break-word", () => [decl("overflow-wrap", "break-word")]);
  stat("wrap-anywhere", () => [decl("overflow-wrap", "anywhere")]);
  stat("wrap-normal", () => [decl("overflow-wrap", "normal")]);
  stat("list-image-none", () => [decl("list-style-image", "none")]);
  fn("list-image", { handle: single("list-style-image") });
  stat("content-none", () => [
    propertyRegistration("--tw-content", '""'),
    decl("--tw-content", "none"),
    decl("content", "none"),
  ]);

  // Backgrounds and blending.
  fn("bg-position", { handle: single("background-position") });
  fn("bg-size", { handle: single("background-size") });
  for (const mode of BLEND_MODES) {
    stat(`bg-blend-${mode}`, () => [decl("background-blend-mode", mode)]);
    stat(`mix-blend-${mode}`, () => [decl("mix-blend-mode", mode)]);
  }
  stat("mix-blend-plus-darker", () => [decl("mix-blend-mode", "plus-darker")]);
  stat("mix-blend-plus-lighter", () => [
    decl("mix-blend-mode", "plus-lighter"),
  ]);

  // Layout.
  stat("container", () => {
    const breakpoints: string[] = [];
    for (const [key, value] of theme.namespace("--breakpoint")) {
      if (key === null || key.startsWith("--")) continue;
      breakpoints.push(value);
    }
    breakpoints.sort(compareQueryValueStrings);
    return [
      decl("width", "100%"),
      ...breakpoints.map((value) =>
        atRule("@media", `(width >= ${value})`, [decl("max-width", value)])
      ),
    ];
  });
  for (
    const value of [
      "auto",
      "avoid",
      "all",
      "avoid-page",
      "page",
      "left",
      "right",
      "column",
    ]
  ) {
    stat(`break-after-${value}`, () => [decl("break-after", value)]);
    stat(`break-before-${value}`, () => [decl("break-before", value)]);
  }
  for (const value of ["auto", "avoid", "avoid-page", "avoid-column"]) {
    stat(`break-inside-${value}`, () => [decl("break-inside", value)]);
  }
  stat("box-decoration-clone", () => [decl("box-decoration-break", "clone")]);
  stat("box-decoration-slice", () => [decl("box-decoration-break", "slice")]);
  fn("object", { handle: single("object-position") });

  // Tables.
  stat("border-collapse", () => [decl("border-collapse", "collapse")]);
  stat("border-separate", () => [decl("border-collapse", "separate")]);
  const borderSpacing = (name: string, variables: string[]) =>
    spacingUtility(
      utilities,
      theme,
      name,
      ["--border-spacing", "--spacing"],
      (value) => [
        propertyRegistration("--tw-border-spacing-x", "0"),
        propertyRegistration("--tw-border-spacing-y", "0"),
        ...variables.map((v) => decl(v, value)),
        decl(
          "border-spacing",
          "var(--tw-border-spacing-x) var(--tw-border-spacing-y)",
        ),
      ],
    );
  borderSpacing("border-spacing", [
    "--tw-border-spacing-x",
    "--tw-border-spacing-y",
  ]);
  borderSpacing("border-spacing-x", ["--tw-border-spacing-x"]);
  borderSpacing("border-spacing-y", ["--tw-border-spacing-y"]);
  stat("table-auto", () => [decl("table-layout", "auto")]);
  stat("table-fixed", () => [decl("table-layout", "fixed")]);
  stat("caption-top", () => [decl("caption-side", "top")]);
  stat("caption-bottom", () => [decl("caption-side", "bottom")]);

  // Scrolling and touch.
  const scrollSides: [string, string][] = [
    ["", ""],
    ["x", "-inline"],
    ["y", "-block"],
    ["s", "-inline-start"],
    ["e", "-inline-end"],
    ["t", "-top"],
    ["r", "-right"],
    ["b", "-bottom"],
    ["l", "-left"],
  ];
  for (const [suffix, property] of scrollSides) {
    spacingUtility(
      utilities,
      theme,
      `scroll-m${suffix}`,
      ["--scroll-margin", "--spacing"],
      single(`scroll-margin${property}`),
      { supportsNegative: true },
    );
    spacingUtility(
      utilities,
      theme,
      `scroll-p${suffix}`,
      ["--scroll-padding", "--spacing"],
      single(`scroll-padding${property}`),
    );
  }
  stat("snap-none", () => [decl("scroll-snap-type", "none")]);
  for (const axis of ["x", "y", "both"]) {
    stat(`snap-${axis}`, () => [
      propertyRegistration("--tw-scroll-snap-strictness", "proximity"),
      decl("scroll-snap-type", `${axis} var(--tw-scroll-snap-strictness)`),
    ]);
  }
  for (const strictness of ["mandatory", "proximity"]) {
    stat(`snap-${strictness}`, () => [
      propertyRegistration("--tw-scroll-snap-strictness", "proximity"),
      decl("--tw-scroll-snap-strictness", strictness),
    ]);
  }
  for (const value of ["start", "end", "center"]) {
    stat(`snap-${value}`, () => [decl("scroll-snap-align", value)]);
  }
  stat("snap-align-none", () => [decl("scroll-snap-align", "none")]);
  stat("snap-normal", () => [decl("scroll-snap-stop", "normal")]);
  stat("snap-always", () => [decl("scroll-snap-stop", "always")]);
  for (const value of ["auto", "none", "manipulation"]) {
    stat(`touch-${value}`, () => [decl("touch-action", value)]);
  }
  const touch = (value: string, variable: string) =>
    stat(`touch-${value}`, () => [
      propertyRegistration("--tw-pan-x"),
      propertyRegistration("--tw-pan-y"),
      propertyRegistration("--tw-pinch-zoom"),
      decl(variable, value),
      decl(
        "touch-action",
        "var(--tw-pan-x,) var(--tw-pan-y,) var(--tw-pinch-zoom,)",
      ),
    ]);
  for (const value of ["pan-x", "pan-left", "pan-right"]) {
    touch(value, "--tw-pan-x");
  }
  for (const value of ["pan-y", "pan-up", "pan-down"]) {
    touch(value, "--tw-pan-y");
  }
  touch("pinch-zoom", "--tw-pinch-zoom");

  // Interactivity and SVG.
  utilities.functional("stroke", (candidate) => {
    if (candidate.value === null) return;
    if (candidate.value.kind === "arbitrary") {
      const value = candidate.value.value;
      const type = candidate.value.dataType ??
        inferDataType(value, ["color", "length", "number", "percentage"]);
      if (type === "length" || type === "number" || type === "percentage") {
        if (candidate.modifier !== null) return;
        return [decl("stroke-width", value)];
      }
      const resolved = asColor(value, candidate.modifier, theme);
      if (resolved === null) return;
      return [decl("stroke", resolved)];
    }
    const color = resolveThemeColor(candidate, ["--stroke", "--color"], theme);
    if (color !== null) return [decl("stroke", color)];
    if (candidate.modifier !== null) return;
    const width = theme.resolve(candidate.value.value, ["--stroke-width"]);
    if (width !== null) return [decl("stroke-width", width)];
    if (isPositiveInteger(candidate.value.value)) {
      return [decl("stroke-width", candidate.value.value)];
    }
  });
  const schemes: [string, string][] = [
    ["normal", "normal"],
    ["dark", "dark"],
    ["light", "light"],
    ["light-dark", "light dark"],
    ["only-dark", "only dark"],
    ["only-light", "only light"],
  ];
  for (const [name, value] of schemes) {
    stat(`scheme-${name}`, () => [decl("color-scheme", value)]);
  }
  stat("field-sizing-content", () => [decl("field-sizing", "content")]);
  stat("field-sizing-fixed", () => [decl("field-sizing", "fixed")]);
  stat("forced-color-adjust-auto", () => [decl("forced-color-adjust", "auto")]);
  stat("forced-color-adjust-none", () => [decl("forced-color-adjust", "none")]);
}
