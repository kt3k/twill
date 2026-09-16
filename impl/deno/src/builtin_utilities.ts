/**
 * Built-in utility catalog (SPEC §10.8).
 *
 * @module
 */

import { type AstNode, decl } from "./ast.ts";
import type { FunctionalCandidate } from "./candidate.ts";
import { inferDataType } from "./data_types.ts";
import type { Theme } from "./theme.ts";
import { registerDividerUtilities } from "./dividers.ts";
import { registerFilterUtilities } from "./filters.ts";
import { registerGradientUtilities } from "./gradients.ts";
import { registerShadowUtilities } from "./shadows.ts";
import { registerTransformUtilities } from "./transforms.ts";
import { isMultipleOfQuarter, isPositiveInteger } from "./utils.ts";
import {
  asColor,
  bareInteger,
  colorUtility,
  type CompileResult,
  type Declarations,
  functionalUtility,
  propertyRegistration,
  resolveThemeColor,
  spacingUtility,
  staticUtility,
  type Utilities,
} from "./utilities.ts";

/** Registers every built-in utility. */
export function registerBuiltinUtilities(
  utilities: Utilities,
  theme: Theme,
): void {
  const stat = (name: string, declarations: Declarations | (() => AstNode[])) =>
    staticUtility(utilities, name, declarations);
  const fn = (
    root: string,
    desc: Parameters<typeof functionalUtility>[3],
  ) => functionalUtility(utilities, theme, root, desc);
  const spacing = (
    name: string,
    themeKeys: string[],
    handle: (value: string) => AstNode[],
    options: { supportsNegative?: boolean; supportsFractions?: boolean } = {},
  ) => spacingUtility(utilities, theme, name, themeKeys, handle, options);
  const color = (
    root: string,
    themeKeys: string[],
    handle: (value: string) => AstNode[],
  ) => colorUtility(utilities, theme, root, { themeKeys, handle });
  const single =
    (property: string) => (value: string) => [decl(property, value)];
  const multi = (...properties: string[]) => (value: string) =>
    properties.map((property) => decl(property, value));

  // ---------------------------------------------------------------------
  // Layout
  // ---------------------------------------------------------------------

  const displays: [string, string][] = [
    ["block", "block"],
    ["inline-block", "inline-block"],
    ["inline", "inline"],
    ["flex", "flex"],
    ["inline-flex", "inline-flex"],
    ["grid", "grid"],
    ["inline-grid", "inline-grid"],
    ["hidden", "none"],
    ["contents", "contents"],
    ["flow-root", "flow-root"],
    ["table", "table"],
    ["table-cell", "table-cell"],
    ["table-row", "table-row"],
    ["table-caption", "table-caption"],
    ["table-column", "table-column"],
    ["table-column-group", "table-column-group"],
    ["table-footer-group", "table-footer-group"],
    ["table-header-group", "table-header-group"],
    ["table-row-group", "table-row-group"],
    ["inline-table", "inline-table"],
    ["list-item", "list-item"],
  ];
  for (const [name, value] of displays) stat(name, [["display", value]]);

  for (const name of ["static", "fixed", "absolute", "relative", "sticky"]) {
    stat(name, [["position", name]]);
  }

  stat("visible", [["visibility", "visible"]]);
  stat("invisible", [["visibility", "hidden"]]);
  stat("collapse", [["visibility", "collapse"]]);

  stat("isolate", [["isolation", "isolate"]]);
  stat("isolation-auto", [["isolation", "auto"]]);

  stat("box-border", [["box-sizing", "border-box"]]);
  stat("box-content", [["box-sizing", "content-box"]]);

  for (const value of ["auto", "hidden", "clip", "visible", "scroll"]) {
    stat(`overflow-${value}`, [["overflow", value]]);
    stat(`overflow-x-${value}`, [["overflow-x", value]]);
    stat(`overflow-y-${value}`, [["overflow-y", value]]);
  }

  for (const value of ["auto", "contain", "none"]) {
    stat(`overscroll-${value}`, [["overscroll-behavior", value]]);
    stat(`overscroll-x-${value}`, [["overscroll-behavior-x", value]]);
    stat(`overscroll-y-${value}`, [["overscroll-behavior-y", value]]);
  }

  stat("float-left", [["float", "left"]]);
  stat("float-right", [["float", "right"]]);
  stat("float-start", [["float", "inline-start"]]);
  stat("float-end", [["float", "inline-end"]]);
  stat("float-none", [["float", "none"]]);
  stat("clear-left", [["clear", "left"]]);
  stat("clear-right", [["clear", "right"]]);
  stat("clear-start", [["clear", "inline-start"]]);
  stat("clear-end", [["clear", "inline-end"]]);
  stat("clear-both", [["clear", "both"]]);
  stat("clear-none", [["clear", "none"]]);

  stat("sr-only", [
    ["position", "absolute"],
    ["width", "1px"],
    ["height", "1px"],
    ["padding", "0"],
    ["margin", "-1px"],
    ["overflow", "hidden"],
    ["clip-path", "inset(50%)"],
    ["white-space", "nowrap"],
    ["border-width", "0"],
  ]);
  stat("not-sr-only", [
    ["position", "static"],
    ["width", "auto"],
    ["height", "auto"],
    ["padding", "0"],
    ["margin", "0"],
    ["overflow", "visible"],
    ["clip-path", "none"],
    ["white-space", "normal"],
  ]);

  const insets: [string, string[]][] = [
    ["inset", ["inset"]],
    ["inset-x", ["inset-inline"]],
    ["inset-y", ["inset-block"]],
    ["inset-s", ["inset-inline-start"]],
    ["inset-e", ["inset-inline-end"]],
    ["top", ["top"]],
    ["right", ["right"]],
    ["bottom", ["bottom"]],
    ["left", ["left"]],
  ];
  for (const [name, properties] of insets) {
    stat(
      `${name}-auto`,
      properties.map((p) => [p, "auto"] as [string, string]),
    );
    stat(
      `${name}-full`,
      properties.map((p) => [p, "100%"] as [string, string]),
    );
    stat(
      `-${name}-full`,
      properties.map((p) => [p, "-100%"] as [string, string]),
    );
    spacing(name, ["--inset", "--spacing"], multi(...properties), {
      supportsNegative: true,
      supportsFractions: true,
    });
  }

  stat("z-auto", [["z-index", "auto"]]);
  fn("z", {
    themeKeys: ["--z-index"],
    supportsNegative: true,
    handleBareValue: bareInteger,
    handle: single("z-index"),
  });

  stat("order-first", [["order", "-9999"]]);
  stat("order-last", [["order", "9999"]]);
  stat("order-none", [["order", "0"]]);
  fn("order", {
    themeKeys: ["--order"],
    supportsNegative: true,
    handleBareValue: bareInteger,
    handle: single("order"),
  });

  // ---------------------------------------------------------------------
  // Flexbox and grid
  // ---------------------------------------------------------------------

  stat("flex-row", [["flex-direction", "row"]]);
  stat("flex-row-reverse", [["flex-direction", "row-reverse"]]);
  stat("flex-col", [["flex-direction", "column"]]);
  stat("flex-col-reverse", [["flex-direction", "column-reverse"]]);
  stat("flex-wrap", [["flex-wrap", "wrap"]]);
  stat("flex-nowrap", [["flex-wrap", "nowrap"]]);
  stat("flex-wrap-reverse", [["flex-wrap", "wrap-reverse"]]);

  stat("flex-auto", [["flex", "auto"]]);
  stat("flex-initial", [["flex", "0 auto"]]);
  stat("flex-none", [["flex", "none"]]);
  fn("flex", {
    themeKeys: ["--flex"],
    defaultValue: null,
    supportsFractions: true,
    handleBareValue: bareInteger,
    handle: single("flex"),
  });

  fn("grow", {
    themeKeys: ["--flex-grow"],
    defaultValue: "1",
    handleBareValue: bareInteger,
    handle: single("flex-grow"),
  });
  fn("shrink", {
    themeKeys: ["--flex-shrink"],
    defaultValue: "1",
    handleBareValue: bareInteger,
    handle: single("flex-shrink"),
  });

  stat("basis-auto", [["flex-basis", "auto"]]);
  stat("basis-full", [["flex-basis", "100%"]]);
  spacing(
    "basis",
    ["--flex-basis", "--spacing", "--container"],
    single("flex-basis"),
    {
      supportsFractions: true,
    },
  );

  for (
    const [name, property] of [["grid-cols", "grid-template-columns"], [
      "grid-rows",
      "grid-template-rows",
    ]] as const
  ) {
    stat(`${name}-none`, [[property, "none"]]);
    stat(`${name}-subgrid`, [[property, "subgrid"]]);
    fn(name, {
      themeKeys: [`--${property}`],
      defaultValue: null,
      handleBareValue: (value) =>
        isPositiveInteger(value.value)
          ? `repeat(${value.value}, minmax(0, 1fr))`
          : null,
      handle: single(property),
    });
  }

  for (const [prefix, axis] of [["col", "column"], ["row", "row"]] as const) {
    stat(`${prefix}-span-full`, [[`grid-${axis}`, "1 / -1"]]);
    fn(`${prefix}-span`, {
      defaultValue: null,
      handleBareValue: (value) =>
        isPositiveInteger(value.value)
          ? `span ${value.value} / span ${value.value}`
          : null,
      handle: single(`grid-${axis}`),
    });
    stat(`${prefix}-start-auto`, [[`grid-${axis}-start`, "auto"]]);
    fn(`${prefix}-start`, {
      themeKeys: [`--grid-${axis}-start`],
      defaultValue: null,
      supportsNegative: true,
      handleBareValue: bareInteger,
      handle: single(`grid-${axis}-start`),
    });
    stat(`${prefix}-end-auto`, [[`grid-${axis}-end`, "auto"]]);
    fn(`${prefix}-end`, {
      themeKeys: [`--grid-${axis}-end`],
      defaultValue: null,
      supportsNegative: true,
      handleBareValue: bareInteger,
      handle: single(`grid-${axis}-end`),
    });
    stat(`${prefix}-auto`, [[`grid-${axis}`, "auto"]]);
    fn(prefix, {
      themeKeys: [`--grid-${axis}`],
      defaultValue: null,
      supportsNegative: true,
      handleBareValue: bareInteger,
      handle: single(`grid-${axis}`),
    });
  }

  stat("grid-flow-row", [["grid-auto-flow", "row"]]);
  stat("grid-flow-col", [["grid-auto-flow", "column"]]);
  stat("grid-flow-dense", [["grid-auto-flow", "dense"]]);
  stat("grid-flow-row-dense", [["grid-auto-flow", "row dense"]]);
  stat("grid-flow-col-dense", [["grid-auto-flow", "column dense"]]);

  for (
    const [name, property] of [["auto-cols", "grid-auto-columns"], [
      "auto-rows",
      "grid-auto-rows",
    ]] as const
  ) {
    stat(`${name}-auto`, [[property, "auto"]]);
    stat(`${name}-min`, [[property, "min-content"]]);
    stat(`${name}-max`, [[property, "max-content"]]);
    stat(`${name}-fr`, [[property, "minmax(0, 1fr)"]]);
    fn(name, {
      themeKeys: [`--${property}`],
      defaultValue: null,
      handle: single(property),
    });
  }

  spacing("gap", ["--gap", "--spacing"], single("gap"));
  spacing("gap-x", ["--gap", "--spacing"], single("column-gap"));
  spacing("gap-y", ["--gap", "--spacing"], single("row-gap"));

  const alignments: [string, string][] = [
    ["normal", "normal"],
    ["center", "center"],
    ["start", "flex-start"],
    ["end", "flex-end"],
    ["between", "space-between"],
    ["around", "space-around"],
    ["evenly", "space-evenly"],
    ["stretch", "stretch"],
    ["baseline", "baseline"],
  ];
  for (const [name, value] of alignments) {
    stat(`justify-${name}`, [["justify-content", value]]);
  }
  for (const [name, value] of alignments) {
    if (name === "between" || name === "around" || name === "evenly") continue;
    stat(`justify-items-${name}`, [["justify-items", name]]);
    if (
      name === "start" || name === "end" || name === "center" ||
      name === "stretch"
    ) {
      stat(`justify-self-${name}`, [["justify-self", name]]);
    }
    stat(`items-${name}`, [["align-items", value]]);
    if (name !== "normal") stat(`self-${name}`, [["align-self", value]]);
  }
  stat("justify-self-auto", [["justify-self", "auto"]]);
  stat("self-auto", [["align-self", "auto"]]);
  for (const [name, value] of alignments) {
    stat(`content-${name}`, [["align-content", value]]);
  }
  for (const [name, value] of alignments) {
    stat(`place-content-${name}`, [[
      "place-content",
      value === "flex-start" || value === "flex-end" ? name : value,
    ]]);
  }
  for (const name of ["start", "end", "center", "baseline", "stretch"]) {
    stat(`place-items-${name}`, [["place-items", name]]);
  }
  for (const name of ["auto", "start", "end", "center", "stretch"]) {
    stat(`place-self-${name}`, [["place-self", name]]);
  }

  // ---------------------------------------------------------------------
  // Spacing
  // ---------------------------------------------------------------------

  const paddings: [string, string][] = [
    ["p", "padding"],
    ["px", "padding-inline"],
    ["py", "padding-block"],
    ["ps", "padding-inline-start"],
    ["pe", "padding-inline-end"],
    ["pt", "padding-top"],
    ["pr", "padding-right"],
    ["pb", "padding-bottom"],
    ["pl", "padding-left"],
  ];
  for (const [name, property] of paddings) {
    spacing(name, ["--padding", "--spacing"], single(property));
  }

  const margins: [string, string][] = [
    ["m", "margin"],
    ["mx", "margin-inline"],
    ["my", "margin-block"],
    ["ms", "margin-inline-start"],
    ["me", "margin-inline-end"],
    ["mt", "margin-top"],
    ["mr", "margin-right"],
    ["mb", "margin-bottom"],
    ["ml", "margin-left"],
  ];
  for (const [name, property] of margins) {
    stat(`${name}-auto`, [[property, "auto"]]);
    spacing(name, ["--margin", "--spacing"], single(property), {
      supportsNegative: true,
    });
  }

  // ---------------------------------------------------------------------
  // Sizing
  // ---------------------------------------------------------------------

  const widthStatics: [string, string][] = [
    ["auto", "auto"],
    ["full", "100%"],
    ["screen", "100vw"],
    ["svw", "100svw"],
    ["lvw", "100lvw"],
    ["dvw", "100dvw"],
    ["min", "min-content"],
    ["max", "max-content"],
    ["fit", "fit-content"],
  ];
  const heightStatics: [string, string][] = [
    ["auto", "auto"],
    ["full", "100%"],
    ["screen", "100vh"],
    ["svh", "100svh"],
    ["lvh", "100lvh"],
    ["dvh", "100dvh"],
    ["min", "min-content"],
    ["max", "max-content"],
    ["fit", "fit-content"],
  ];

  for (
    const [name, property, key] of [
      ["w", "width", "--width"],
      ["min-w", "min-width", "--min-width"],
      ["max-w", "max-width", "--max-width"],
    ] as const
  ) {
    for (const [suffix, value] of widthStatics) {
      if (name !== "w" && suffix === "auto") continue;
      stat(`${name}-${suffix}`, [[property, value]]);
    }
    if (name === "max-w") stat("max-w-none", [["max-width", "none"]]);
    spacing(name, [key, "--spacing", "--container"], single(property), {
      supportsFractions: true,
    });
  }

  for (
    const [name, property, key] of [
      ["h", "height", "--height"],
      ["min-h", "min-height", "--min-height"],
      ["max-h", "max-height", "--max-height"],
    ] as const
  ) {
    for (const [suffix, value] of heightStatics) {
      if (name !== "h" && suffix === "auto") continue;
      stat(`${name}-${suffix}`, [[property, value]]);
    }
    if (name === "max-h") stat("max-h-none", [["max-height", "none"]]);
    spacing(name, [key, "--spacing"], single(property), {
      supportsFractions: true,
    });
  }

  const sizeHandle = (value: string) => [
    decl("--tw-sort", "size"),
    decl("width", value),
    decl("height", value),
  ];
  for (const [suffix, value] of widthStatics) {
    if (
      suffix === "screen" || suffix === "svw" || suffix === "lvw" ||
      suffix === "dvw"
    ) continue;
    stat(`size-${suffix}`, () => sizeHandle(value));
  }
  spacing("size", ["--size", "--spacing", "--container"], sizeHandle, {
    supportsFractions: true,
  });

  // ---------------------------------------------------------------------
  // Typography
  // ---------------------------------------------------------------------

  const fontWeight = (value: string): AstNode[] => [
    propertyRegistration("--tw-font-weight"),
    decl("--tw-font-weight", value),
    decl("font-weight", value),
  ];

  utilities.functional("font", (candidate) => {
    if (candidate.value === null) return;
    if (candidate.value.kind === "arbitrary") {
      if (candidate.modifier !== null) return;
      const value = candidate.value.value;
      const type = candidate.value.dataType ??
        inferDataType(value, ["number", "family-name", "generic-name"]);
      if (type === "number") return fontWeight(value);
      if (type === "family-name" || type === "generic-name") {
        return [decl("font-family", value)];
      }
      return;
    }
    if (candidate.modifier !== null) return;
    const family = theme.resolveWith(candidate.value.value, ["--font"], [
      "--font-feature-settings",
      "--font-variation-settings",
    ]);
    if (family !== null) {
      const [value, extra] = family;
      const nodes: AstNode[] = [decl("font-family", value)];
      if (extra["--font-feature-settings"] !== undefined) {
        nodes.push(
          decl("font-feature-settings", extra["--font-feature-settings"]),
        );
      }
      if (extra["--font-variation-settings"] !== undefined) {
        nodes.push(
          decl("font-variation-settings", extra["--font-variation-settings"]),
        );
      }
      return nodes;
    }
    const weight = theme.resolve(candidate.value.value, ["--font-weight"]);
    if (weight !== null) return fontWeight(weight);
  });

  stat("text-left", [["text-align", "left"]]);
  stat("text-center", [["text-align", "center"]]);
  stat("text-right", [["text-align", "right"]]);
  stat("text-justify", [["text-align", "justify"]]);
  stat("text-start", [["text-align", "start"]]);
  stat("text-end", [["text-align", "end"]]);
  stat("text-ellipsis", [["text-overflow", "ellipsis"]]);
  stat("text-clip", [["text-overflow", "clip"]]);
  stat("text-wrap", [["text-wrap", "wrap"]]);
  stat("text-nowrap", [["text-wrap", "nowrap"]]);
  stat("text-balance", [["text-wrap", "balance"]]);
  stat("text-pretty", [["text-wrap", "pretty"]]);

  const resolveLeadingModifier = (
    candidate: FunctionalCandidate,
  ): string | null | undefined => {
    const modifier = candidate.modifier;
    if (modifier === null) return undefined;
    if (modifier.kind === "arbitrary") return modifier.value;
    const leading = theme.resolve(modifier.value, ["--leading"]);
    if (leading !== null) return leading;
    if (
      isMultipleOfQuarter(modifier.value) &&
      theme.resolve(null, ["--spacing"]) !== null
    ) {
      return `--spacing(${modifier.value})`;
    }
    if (modifier.value === "none") return "1";
    return null;
  };

  utilities.functional("text", (candidate) => {
    if (candidate.value === null) return;
    if (candidate.value.kind === "arbitrary") {
      const value = candidate.value.value;
      const type = candidate.value.dataType ??
        inferDataType(value, [
          "color",
          "length",
          "percentage",
          "absolute-size",
          "relative-size",
        ]);
      if (type === "color" || type === null) {
        const resolved = asColor(value, candidate.modifier, theme);
        if (resolved === null) return;
        return [decl("color", resolved)];
      }
      const nodes: AstNode[] = [decl("font-size", value)];
      const leading = resolveLeadingModifier(candidate);
      if (leading === null) return;
      if (leading !== undefined) nodes.push(decl("line-height", leading));
      return nodes;
    }

    const asThemeColor = resolveThemeColor(candidate, [
      "--text-color",
      "--color",
    ], theme);
    if (asThemeColor !== null) return [decl("color", asThemeColor)];

    const size = theme.resolveWith(candidate.value.value, ["--text"], [
      "--line-height",
      "--letter-spacing",
      "--font-weight",
    ]);
    if (size === null) return;
    const [value, extra] = size;
    const nodes: AstNode[] = [decl("font-size", value)];
    const leading = resolveLeadingModifier(candidate);
    if (leading === null) return;
    if (leading !== undefined) {
      nodes.push(decl("line-height", leading));
    } else if (extra["--line-height"] !== undefined) {
      nodes.push(
        decl("line-height", `var(--tw-leading, ${extra["--line-height"]})`),
      );
    }
    if (extra["--letter-spacing"] !== undefined) {
      nodes.push(
        decl(
          "letter-spacing",
          `var(--tw-tracking, ${extra["--letter-spacing"]})`,
        ),
      );
    }
    if (extra["--font-weight"] !== undefined) {
      nodes.push(
        decl("font-weight", `var(--tw-font-weight, ${extra["--font-weight"]})`),
      );
    }
    return nodes;
  });

  const leadingHandle = (value: string): AstNode[] => [
    propertyRegistration("--tw-leading"),
    decl("--tw-leading", value),
    decl("line-height", value),
  ];
  stat("leading-none", () => leadingHandle("1"));
  spacing("leading", ["--leading", "--spacing"], leadingHandle);

  fn("tracking", {
    themeKeys: ["--tracking"],
    supportsNegative: true,
    handle: (value) => [
      propertyRegistration("--tw-tracking"),
      decl("--tw-tracking", value),
      decl("letter-spacing", value),
    ],
  });

  stat("uppercase", [["text-transform", "uppercase"]]);
  stat("lowercase", [["text-transform", "lowercase"]]);
  stat("capitalize", [["text-transform", "capitalize"]]);
  stat("normal-case", [["text-transform", "none"]]);

  stat("italic", [["font-style", "italic"]]);
  stat("not-italic", [["font-style", "normal"]]);

  stat("underline", [["text-decoration-line", "underline"]]);
  stat("overline", [["text-decoration-line", "overline"]]);
  stat("line-through", [["text-decoration-line", "line-through"]]);
  stat("no-underline", [["text-decoration-line", "none"]]);

  stat("truncate", [
    ["overflow", "hidden"],
    ["text-overflow", "ellipsis"],
    ["white-space", "nowrap"],
  ]);

  for (
    const value of [
      "normal",
      "nowrap",
      "pre",
      "pre-line",
      "pre-wrap",
      "break-spaces",
    ]
  ) {
    stat(`whitespace-${value}`, [["white-space", value]]);
  }

  stat("break-normal", [["overflow-wrap", "normal"], ["word-break", "normal"]]);
  stat("break-words", [["overflow-wrap", "break-word"]]);
  stat("break-all", [["word-break", "break-all"]]);
  stat("break-keep", [["word-break", "keep-all"]]);

  stat("list-none", [["list-style-type", "none"]]);
  stat("list-disc", [["list-style-type", "disc"]]);
  stat("list-decimal", [["list-style-type", "decimal"]]);
  stat("list-inside", [["list-style-position", "inside"]]);
  stat("list-outside", [["list-style-position", "outside"]]);

  stat("antialiased", [
    ["-webkit-font-smoothing", "antialiased"],
    ["-moz-osx-font-smoothing", "grayscale"],
  ]);
  stat("subpixel-antialiased", [
    ["-webkit-font-smoothing", "auto"],
    ["-moz-osx-font-smoothing", "auto"],
  ]);

  stat("underline-offset-auto", [["text-underline-offset", "auto"]]);
  fn("underline-offset", {
    themeKeys: ["--text-underline-offset"],
    supportsNegative: true,
    handleBareValue: (
      value,
    ) => (isPositiveInteger(value.value) ? `${value.value}px` : null),
    handleNegativeBareValue: (value) =>
      isPositiveInteger(value.value) ? `-${value.value}px` : null,
    handle: single("text-underline-offset"),
  });

  spacing("indent", ["--text-indent", "--spacing"], single("text-indent"), {
    supportsNegative: true,
  });

  // ---------------------------------------------------------------------
  // Backgrounds and borders
  // ---------------------------------------------------------------------

  utilities.functional("bg", (candidate) => {
    if (candidate.value === null) return;
    if (candidate.value.kind === "arbitrary") {
      const value = candidate.value.value;
      const type = candidate.value.dataType ??
        inferDataType(value, [
          "image",
          "color",
          "percentage",
          "position",
          "bg-size",
          "length",
          "url",
        ]);
      switch (type) {
        case "percentage":
        case "position":
          if (candidate.modifier !== null) return;
          return [decl("background-position", value)];
        case "bg-size":
        case "length":
          if (candidate.modifier !== null) return;
          return [decl("background-size", value)];
        case "image":
        case "url":
          if (candidate.modifier !== null) return;
          return [decl("background-image", value)];
        default: {
          const resolved = asColor(value, candidate.modifier, theme);
          if (resolved === null) return;
          return [decl("background-color", resolved)];
        }
      }
    }
    const asThemeColor = resolveThemeColor(candidate, [
      "--background-color",
      "--color",
    ], theme);
    if (asThemeColor !== null) return [decl("background-color", asThemeColor)];
    const image = theme.resolve(candidate.value.value, ["--background-image"]);
    if (image !== null) {
      if (candidate.modifier !== null) return;
      return [decl("background-image", image)];
    }
  });

  stat("bg-auto", [["background-size", "auto"]]);
  stat("bg-cover", [["background-size", "cover"]]);
  stat("bg-contain", [["background-size", "contain"]]);
  stat("bg-fixed", [["background-attachment", "fixed"]]);
  stat("bg-local", [["background-attachment", "local"]]);
  stat("bg-scroll", [["background-attachment", "scroll"]]);
  for (const position of ["top", "center", "bottom", "left", "right"]) {
    stat(`bg-${position}`, [["background-position", position]]);
  }
  stat("bg-top-left", [["background-position", "left top"]]);
  stat("bg-top-right", [["background-position", "right top"]]);
  stat("bg-bottom-left", [["background-position", "left bottom"]]);
  stat("bg-bottom-right", [["background-position", "right bottom"]]);
  stat("bg-repeat", [["background-repeat", "repeat"]]);
  stat("bg-no-repeat", [["background-repeat", "no-repeat"]]);
  stat("bg-repeat-x", [["background-repeat", "repeat-x"]]);
  stat("bg-repeat-y", [["background-repeat", "repeat-y"]]);
  stat("bg-repeat-round", [["background-repeat", "round"]]);
  stat("bg-repeat-space", [["background-repeat", "space"]]);
  stat("bg-none", [["background-image", "none"]]);
  for (const value of ["border", "padding", "content"]) {
    stat(`bg-clip-${value}`, [["background-clip", `${value}-box`]]);
  }
  stat("bg-clip-text", [["background-clip", "text"]]);
  for (const value of ["border", "padding", "content"]) {
    stat(`bg-origin-${value}`, [["background-origin", `${value}-box`]]);
  }

  const borders: [string, string, string][] = [
    ["border", "border-width", "border-color"],
    ["border-x", "border-inline-width", "border-inline-color"],
    ["border-y", "border-block-width", "border-block-color"],
    ["border-s", "border-inline-start-width", "border-inline-start-color"],
    ["border-e", "border-inline-end-width", "border-inline-end-color"],
    ["border-t", "border-top-width", "border-top-color"],
    ["border-r", "border-right-width", "border-right-color"],
    ["border-b", "border-bottom-width", "border-bottom-color"],
    ["border-l", "border-left-width", "border-left-color"],
  ];
  for (const [root, widthProperty, colorProperty] of borders) {
    const width = (value: string): AstNode[] => [
      propertyRegistration("--tw-border-style", "solid"),
      decl("border-style", "var(--tw-border-style)"),
      decl(widthProperty, value),
    ];
    utilities.functional(root, (candidate): CompileResult => {
      if (candidate.value === null) {
        if (candidate.modifier !== null) return;
        return width(theme.get(["--default-border-width"]) ?? "1px");
      }
      if (candidate.value.kind === "arbitrary") {
        const value = candidate.value.value;
        const type = candidate.value.dataType ??
          inferDataType(value, ["color", "line-width", "length"]);
        if (type === "color") {
          const resolved = asColor(value, candidate.modifier, theme);
          if (resolved === null) return;
          return [decl(colorProperty, resolved)];
        }
        if (type === "line-width" || type === "length") {
          if (candidate.modifier !== null) return;
          return width(value);
        }
        return;
      }
      const asThemeColor = resolveThemeColor(candidate, [
        "--border-color",
        "--color",
      ], theme);
      if (asThemeColor !== null) return [decl(colorProperty, asThemeColor)];
      if (candidate.modifier !== null) return;
      const themeWidth = theme.resolve(candidate.value.value, [
        "--border-width",
      ]);
      if (themeWidth !== null) return width(themeWidth);
      if (isPositiveInteger(candidate.value.value)) {
        return width(`${candidate.value.value}px`);
      }
    });
  }

  for (
    const style of ["solid", "dashed", "dotted", "double", "hidden", "none"]
  ) {
    stat(`border-${style}`, [["--tw-border-style", style], [
      "border-style",
      style,
    ]]);
  }

  const radii: [string, string[]][] = [
    ["rounded", ["border-radius"]],
    ["rounded-s", ["border-start-start-radius", "border-end-start-radius"]],
    ["rounded-e", ["border-start-end-radius", "border-end-end-radius"]],
    ["rounded-t", ["border-top-left-radius", "border-top-right-radius"]],
    ["rounded-r", ["border-top-right-radius", "border-bottom-right-radius"]],
    ["rounded-b", ["border-bottom-right-radius", "border-bottom-left-radius"]],
    ["rounded-l", ["border-top-left-radius", "border-bottom-left-radius"]],
    ["rounded-ss", ["border-start-start-radius"]],
    ["rounded-se", ["border-start-end-radius"]],
    ["rounded-ee", ["border-end-end-radius"]],
    ["rounded-es", ["border-end-start-radius"]],
    ["rounded-tl", ["border-top-left-radius"]],
    ["rounded-tr", ["border-top-right-radius"]],
    ["rounded-br", ["border-bottom-right-radius"]],
    ["rounded-bl", ["border-bottom-left-radius"]],
  ];
  for (const [root, properties] of radii) {
    stat(`${root}-none`, properties.map((p) => [p, "0"] as [string, string]));
    stat(
      `${root}-full`,
      properties.map((p) => [p, "calc(infinity * 1px)"] as [string, string]),
    );
    fn(root, { themeKeys: ["--radius"], handle: multi(...properties) });
  }

  // ---------------------------------------------------------------------
  // Shadows and rings
  // ---------------------------------------------------------------------

  registerShadowUtilities(utilities, theme);

  // ---------------------------------------------------------------------
  // Transforms
  // ---------------------------------------------------------------------

  registerTransformUtilities(utilities, theme);

  // ---------------------------------------------------------------------
  // Filters
  // ---------------------------------------------------------------------

  registerFilterUtilities(utilities, theme);

  // ---------------------------------------------------------------------
  // Gradients
  // ---------------------------------------------------------------------

  registerGradientUtilities(utilities, theme);

  // ---------------------------------------------------------------------
  // Space, dividers, and outlines
  // ---------------------------------------------------------------------

  registerDividerUtilities(utilities, theme);

  // ---------------------------------------------------------------------
  // Effects, transitions, interactivity
  // ---------------------------------------------------------------------

  fn("opacity", {
    themeKeys: ["--opacity"],
    handleBareValue: (
      value,
    ) => (isMultipleOfQuarter(value.value) ? `${value.value}%` : null),
    handle: single("opacity"),
  });

  const transitionTail: Declarations = [
    ["transition-timing-function", "var(--default-transition-timing-function)"],
    ["transition-duration", "var(--default-transition-duration)"],
  ];
  const transitionProperties = [
    "color",
    "background-color",
    "border-color",
    "outline-color",
    "text-decoration-color",
    "fill",
    "stroke",
    "--tw-gradient-from",
    "--tw-gradient-via",
    "--tw-gradient-to",
    "opacity",
    "box-shadow",
    "transform",
    "translate",
    "scale",
    "rotate",
    "filter",
    "-webkit-backdrop-filter",
    "backdrop-filter",
    "display",
    "content-visibility",
    "overlay",
    "pointer-events",
  ];
  stat("transition", [
    ["transition-property", transitionProperties.join(", ")],
    ...transitionTail,
  ]);
  stat("transition-none", [["transition-property", "none"]]);
  stat("transition-all", [["transition-property", "all"], ...transitionTail]);
  stat("transition-colors", [
    [
      "transition-property",
      "color, background-color, border-color, outline-color, text-decoration-color, fill, stroke, --tw-gradient-from, --tw-gradient-via, --tw-gradient-to",
    ],
    ...transitionTail,
  ]);
  stat("transition-opacity", [
    ["transition-property", "opacity"],
    ...transitionTail,
  ]);
  stat("transition-shadow", [
    ["transition-property", "box-shadow"],
    ...transitionTail,
  ]);
  stat("transition-transform", [
    ["transition-property", "transform, translate, scale, rotate"],
    ...transitionTail,
  ]);
  stat("transition-discrete", [["transition-behavior", "allow-discrete"]]);
  stat("transition-normal", [["transition-behavior", "normal"]]);

  fn("duration", {
    themeKeys: ["--transition-duration"],
    handleBareValue: (
      value,
    ) => (isPositiveInteger(value.value) ? `${value.value}ms` : null),
    handle: single("transition-duration"),
  });
  fn("delay", {
    themeKeys: ["--transition-delay"],
    handleBareValue: (
      value,
    ) => (isPositiveInteger(value.value) ? `${value.value}ms` : null),
    handle: single("transition-delay"),
  });
  stat("ease-linear", [["transition-timing-function", "linear"]]);
  stat("ease-initial", [["transition-timing-function", "initial"]]);
  fn("ease", {
    themeKeys: ["--ease"],
    handle: single("transition-timing-function"),
  });

  stat("animate-none", [["animation", "none"]]);
  fn("animate", { themeKeys: ["--animate"], handle: single("animation") });

  const cursors = [
    "auto",
    "default",
    "pointer",
    "wait",
    "text",
    "move",
    "help",
    "not-allowed",
    "none",
    "context-menu",
    "progress",
    "cell",
    "crosshair",
    "vertical-text",
    "alias",
    "copy",
    "no-drop",
    "grab",
    "grabbing",
    "all-scroll",
    "col-resize",
    "row-resize",
    "n-resize",
    "e-resize",
    "s-resize",
    "w-resize",
    "ne-resize",
    "nw-resize",
    "se-resize",
    "sw-resize",
    "ew-resize",
    "ns-resize",
    "nesw-resize",
    "nwse-resize",
    "zoom-in",
    "zoom-out",
  ];
  for (const value of cursors) stat(`cursor-${value}`, [["cursor", value]]);
  fn("cursor", { themeKeys: ["--cursor"], handle: single("cursor") });

  for (const value of ["none", "text", "all", "auto"]) {
    stat(`select-${value}`, [["-webkit-user-select", value], [
      "user-select",
      value,
    ]]);
  }
  stat("pointer-events-none", [["pointer-events", "none"]]);
  stat("pointer-events-auto", [["pointer-events", "auto"]]);
  stat("resize", [["resize", "both"]]);
  stat("resize-none", [["resize", "none"]]);
  stat("resize-x", [["resize", "horizontal"]]);
  stat("resize-y", [["resize", "vertical"]]);
  stat("appearance-none", [["appearance", "none"]]);
  stat("appearance-auto", [["appearance", "auto"]]);
  stat("scroll-auto", [["scroll-behavior", "auto"]]);
  stat("scroll-smooth", [["scroll-behavior", "smooth"]]);

  stat("will-change-auto", [["will-change", "auto"]]);
  stat("will-change-scroll", [["will-change", "scroll-position"]]);
  stat("will-change-contents", [["will-change", "contents"]]);
  stat("will-change-transform", [["will-change", "transform"]]);
  fn("will-change", {
    themeKeys: ["--will-change"],
    handle: single("will-change"),
  });

  utilities.functional("content", (candidate) => {
    if (candidate.value === null || candidate.value.kind !== "arbitrary") {
      return;
    }
    if (candidate.modifier !== null) return;
    return [
      propertyRegistration("--tw-content", '""'),
      decl("--tw-content", candidate.value.value),
      decl("content", "var(--tw-content)"),
    ];
  });

  stat("aspect-square", [["aspect-ratio", "1 / 1"]]);
  stat("aspect-auto", [["aspect-ratio", "auto"]]);
  fn("aspect", {
    themeKeys: ["--aspect"],
    handleBareValue: (value) => {
      if (value.fraction === null) return null;
      const [a, b] = value.fraction.split("/");
      if (!isPositiveInteger(a) || !isPositiveInteger(b)) return null;
      return `${a} / ${b}`;
    },
    handle: single("aspect-ratio"),
  });

  stat("columns-auto", [["columns", "auto"]]);
  fn("columns", {
    themeKeys: ["--columns", "--container"],
    handleBareValue: bareInteger,
    handle: single("columns"),
  });

  for (const value of ["contain", "cover", "fill", "none", "scale-down"]) {
    stat(`object-${value}`, [["object-fit", value]]);
  }
  for (const position of ["top", "center", "bottom", "left", "right"]) {
    stat(`object-${position}`, [["object-position", position]]);
  }

  stat("accent-auto", [["accent-color", "auto"]]);
  color("accent", ["--accent-color", "--color"], single("accent-color"));
  color("caret", ["--caret-color", "--color"], single("caret-color"));
  stat("fill-none", [["fill", "none"]]);
  color("fill", ["--fill", "--color"], single("fill"));
  stat("stroke-none", [["stroke", "none"]]);
  color("stroke", ["--stroke", "--color"], single("stroke"));
}
