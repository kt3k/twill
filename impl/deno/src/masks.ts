/**
 * Mask utilities (SPEC §10.8, "Masks").
 *
 * @module
 */

import { type AstNode, decl } from "./ast.ts";
import { propertyRegistration } from "./builtin_variants.ts";
import type { FunctionalCandidate } from "./candidate.ts";
import { inferDataType } from "./data_types.ts";
import type { Theme } from "./theme.ts";
import {
  asColor,
  functionalUtility,
  resolveThemeColor,
  staticUtility,
  type Utilities,
} from "./utilities.ts";
import { isPositiveInteger } from "./utils.ts";

const EDGES = ["top", "right", "bottom", "left"];
const WHITE = "linear-gradient(#fff, #fff)";

function stopRegistrations(name: string): AstNode[] {
  return [
    propertyRegistration(`--tw-mask-${name}-from-color`, "black"),
    propertyRegistration(`--tw-mask-${name}-from-position`, "0%"),
    propertyRegistration(`--tw-mask-${name}-to-color`, "transparent"),
    propertyRegistration(`--tw-mask-${name}-to-position`, "100%"),
  ];
}

/** The registrations emitted by every gradient mask utility. */
export function maskRegistrations(): AstNode[] {
  const nodes: AstNode[] = [];
  for (const edge of EDGES) {
    nodes.push(
      propertyRegistration(`--tw-mask-${edge}`, WHITE),
      ...stopRegistrations(edge),
    );
  }
  nodes.push(
    propertyRegistration("--tw-mask-linear", WHITE),
    propertyRegistration("--tw-mask-linear-position", "0deg"),
    ...stopRegistrations("linear"),
    propertyRegistration("--tw-mask-radial", WHITE),
    propertyRegistration("--tw-mask-radial-shape", "ellipse"),
    propertyRegistration("--tw-mask-radial-size", "farthest-corner"),
    propertyRegistration("--tw-mask-radial-position", "center"),
    ...stopRegistrations("radial"),
    propertyRegistration("--tw-mask-conic", WHITE),
    propertyRegistration("--tw-mask-conic-position", "0deg"),
    ...stopRegistrations("conic"),
  );
  return nodes;
}

const MASK_IMAGE: [string, string][] = [
  [
    "mask-image",
    "var(--tw-mask-linear), var(--tw-mask-radial), var(--tw-mask-conic)",
  ],
  ["mask-composite", "intersect"],
];

const EDGE_LIST =
  "var(--tw-mask-left), var(--tw-mask-right), var(--tw-mask-bottom), var(--tw-mask-top)";
const LINEAR =
  "linear-gradient(var(--tw-mask-linear-position), var(--tw-mask-linear-from-color) var(--tw-mask-linear-from-position), var(--tw-mask-linear-to-color) var(--tw-mask-linear-to-position))";
const RADIAL =
  "radial-gradient(var(--tw-mask-radial-shape) var(--tw-mask-radial-size) at var(--tw-mask-radial-position), var(--tw-mask-radial-from-color) var(--tw-mask-radial-from-position), var(--tw-mask-radial-to-color) var(--tw-mask-radial-to-position))";
const CONIC =
  "conic-gradient(from var(--tw-mask-conic-position), var(--tw-mask-conic-from-color) var(--tw-mask-conic-from-position), var(--tw-mask-conic-to-color) var(--tw-mask-conic-to-position))";

function edgeGradient(edge: string): string {
  return `linear-gradient(to ${edge}, var(--tw-mask-${edge}-from-color) var(--tw-mask-${edge}-from-position), var(--tw-mask-${edge}-to-color) var(--tw-mask-${edge}-to-position))`;
}

const POSITIONS: [string, string][] = [
  ["top", "top"],
  ["top-left", "top left"],
  ["top-right", "top right"],
  ["left", "left"],
  ["center", "center"],
  ["right", "right"],
  ["bottom", "bottom"],
  ["bottom-left", "bottom left"],
  ["bottom-right", "bottom right"],
];

interface MaskStop {
  kind: "color" | "position";
  value: string;
}

/** Registers the mask utilities. */
export function registerMaskUtilities(
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

  stat("mask-none", () => [decl("mask-image", "none")]);
  fn("mask", { handle: single("mask-image") });
  for (const value of ["add", "subtract", "intersect", "exclude"]) {
    stat(`mask-${value}`, () => [decl("mask-composite", value)]);
  }
  stat("mask-alpha", () => [decl("mask-mode", "alpha")]);
  stat("mask-luminance", () => [decl("mask-mode", "luminance")]);
  stat("mask-match", () => [decl("mask-mode", "match-source")]);
  stat("mask-type-alpha", () => [decl("mask-type", "alpha")]);
  stat("mask-type-luminance", () => [decl("mask-type", "luminance")]);
  for (const value of ["auto", "cover", "contain"]) {
    stat(`mask-${value}`, () => [decl("mask-size", value)]);
  }
  fn("mask-size", { handle: single("mask-size") });
  for (
    const box of ["border", "padding", "content", "fill", "stroke", "view"]
  ) {
    stat(`mask-clip-${box}`, () => [decl("mask-clip", `${box}-box`)]);
    stat(`mask-origin-${box}`, () => [decl("mask-origin", `${box}-box`)]);
  }
  stat("mask-no-clip", () => [decl("mask-clip", "no-clip")]);
  for (const [name, value] of POSITIONS) {
    stat(`mask-${name}`, () => [decl("mask-position", value)]);
  }
  fn("mask-position", { handle: single("mask-position") });
  const repeats: [string, string][] = [
    ["repeat", "repeat"],
    ["no-repeat", "no-repeat"],
    ["repeat-x", "repeat-x"],
    ["repeat-y", "repeat-y"],
    ["repeat-space", "space"],
    ["repeat-round", "round"],
  ];
  for (const [name, value] of repeats) {
    stat(`mask-${name}`, () => [decl("mask-repeat", value)]);
  }

  const maskStop = (candidate: FunctionalCandidate): MaskStop | null => {
    if (candidate.value === null) return null;
    if (candidate.value.kind === "arbitrary") {
      const value = candidate.value.value;
      const type = candidate.value.dataType ??
        inferDataType(value, ["length", "percentage", "color"]);
      if (type === "length" || type === "percentage") {
        if (candidate.modifier !== null) return null;
        return { kind: "position", value };
      }
      const resolved = asColor(value, candidate.modifier, theme);
      return resolved === null ? null : { kind: "color", value: resolved };
    }
    const color = resolveThemeColor(candidate, ["--color"], theme);
    if (color !== null) return { kind: "color", value: color };
    if (candidate.modifier !== null) return null;
    const named = candidate.value.value;
    if (named.endsWith("%") && isPositiveInteger(named.slice(0, -1))) {
      return { kind: "position", value: named };
    }
    if (isPositiveInteger(named)) {
      if (theme.resolve(null, ["--spacing"]) === null) return null;
      return { kind: "position", value: `--spacing(${named})` };
    }
    return null;
  };
  const composed = (compositions: [string, string][]): AstNode[] => [
    ...maskRegistrations(),
    ...MASK_IMAGE.map(([p, v]) => decl(p, v)),
    ...compositions.map(([p, v]) => decl(p, v)),
  ];
  const stopUtility = (
    root: string,
    side: "from" | "to",
    compositions: [string, string][],
    variables: string[],
  ) =>
    utilities.functional(`${root}-${side}`, (candidate) => {
      const stop = maskStop(candidate);
      if (stop === null) return;
      return [
        ...composed(compositions),
        ...variables.map((v) => decl(`${v}-${side}-${stop.kind}`, stop.value)),
      ];
    });

  const edgeSets: [string, string[]][] = [
    ["t", ["top"]],
    ["r", ["right"]],
    ["b", ["bottom"]],
    ["l", ["left"]],
    ["x", ["left", "right"]],
    ["y", ["top", "bottom"]],
  ];
  for (const [name, edges] of edgeSets) {
    const compositions: [string, string][] = [
      ["--tw-mask-linear", EDGE_LIST],
      ...edges.map((e) =>
        [`--tw-mask-${e}`, edgeGradient(e)] as [string, string]
      ),
    ];
    const variables = edges.map((e) => `--tw-mask-${e}`);
    stopUtility(`mask-${name}`, "from", compositions, variables);
    stopUtility(`mask-${name}`, "to", compositions, variables);
  }

  const angle = (root: string, variable: string, composition: string) => {
    const compile = (
      candidate: FunctionalCandidate,
      negative: boolean,
    ): AstNode[] | undefined => {
      if (candidate.value === null || candidate.modifier !== null) return;
      let value: string;
      if (candidate.value.kind === "arbitrary") {
        if (negative) return;
        value = candidate.value.value;
        const type = candidate.value.dataType ??
          inferDataType(value, ["angle"]);
        if (type !== "angle") return;
      } else {
        if (!isPositiveInteger(candidate.value.value)) return;
        value = negative
          ? `calc(${candidate.value.value}deg * -1)`
          : `${candidate.value.value}deg`;
      }
      return [
        ...composed([[variable, composition]]),
        decl(`${variable}-position`, value),
      ];
    };
    utilities.functional(root, (c) => compile(c, false));
    utilities.functional(`-${root}`, (c) => compile(c, true));
  };
  angle("mask-linear", "--tw-mask-linear", LINEAR);
  stopUtility("mask-linear", "from", [["--tw-mask-linear", LINEAR]], [
    "--tw-mask-linear",
  ]);
  stopUtility("mask-linear", "to", [["--tw-mask-linear", LINEAR]], [
    "--tw-mask-linear",
  ]);

  utilities.functional("mask-radial", (candidate) => {
    if (
      candidate.value === null || candidate.value.kind !== "arbitrary" ||
      candidate.modifier !== null
    ) return;
    return [
      ...composed([["--tw-mask-radial", RADIAL]]),
      decl("--tw-mask-radial-size", candidate.value.value),
    ];
  });
  stopUtility("mask-radial", "from", [["--tw-mask-radial", RADIAL]], [
    "--tw-mask-radial",
  ]);
  stopUtility("mask-radial", "to", [["--tw-mask-radial", RADIAL]], [
    "--tw-mask-radial",
  ]);
  stat("mask-circle", () => [decl("--tw-mask-radial-shape", "circle")]);
  stat("mask-ellipse", () => [decl("--tw-mask-radial-shape", "ellipse")]);
  for (
    const size of [
      "closest-side",
      "farthest-side",
      "closest-corner",
      "farthest-corner",
    ]
  ) {
    stat(`mask-radial-${size}`, () => [decl("--tw-mask-radial-size", size)]);
  }
  for (const [name, value] of POSITIONS) {
    stat(`mask-radial-at-${name}`, () => [
      decl("--tw-mask-radial-position", value),
    ]);
  }
  fn("mask-radial-at", { handle: single("--tw-mask-radial-position") });

  angle("mask-conic", "--tw-mask-conic", CONIC);
  stopUtility("mask-conic", "from", [["--tw-mask-conic", CONIC]], [
    "--tw-mask-conic",
  ]);
  stopUtility("mask-conic", "to", [["--tw-mask-conic", CONIC]], [
    "--tw-mask-conic",
  ]);
}
