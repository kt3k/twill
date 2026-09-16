/**
 * Transform utilities (SPEC §10.8, "Transforms").
 *
 * @module
 */

import { type AstNode, decl } from "./ast.ts";
import { propertyRegistration } from "./builtin_variants.ts";
import type { FunctionalCandidate } from "./candidate.ts";
import type { Theme } from "./theme.ts";
import {
  functionalUtility,
  spacingUtility,
  staticUtility,
  type Utilities,
} from "./utilities.ts";
import { isPositiveInteger } from "./utils.ts";

export const TRANSFORM =
  "var(--tw-rotate-x,) var(--tw-rotate-y,) var(--tw-rotate-z,) var(--tw-skew-x,) var(--tw-skew-y,)";
const TRANSLATE_2D = "var(--tw-translate-x) var(--tw-translate-y)";
const TRANSLATE_3D = `${TRANSLATE_2D} var(--tw-translate-z)`;
const SCALE_2D = "var(--tw-scale-x) var(--tw-scale-y)";
const SCALE_3D = `${SCALE_2D} var(--tw-scale-z)`;

function transformRegistrations(): AstNode[] {
  return [
    propertyRegistration("--tw-rotate-x"),
    propertyRegistration("--tw-rotate-y"),
    propertyRegistration("--tw-rotate-z"),
    propertyRegistration("--tw-skew-x"),
    propertyRegistration("--tw-skew-y"),
  ];
}

function translateRegistrations(): AstNode[] {
  return ["x", "y", "z"].map((axis) =>
    propertyRegistration(`--tw-translate-${axis}`, "0")
  );
}

function scaleRegistrations(): AstNode[] {
  return ["x", "y", "z"].map((axis) =>
    propertyRegistration(`--tw-scale-${axis}`, "1")
  );
}

const ORIGINS: [string, string][] = [
  ["center", "center"],
  ["top", "top"],
  ["top-right", "top right"],
  ["right", "right"],
  ["bottom-right", "bottom right"],
  ["bottom", "bottom"],
  ["bottom-left", "bottom left"],
  ["left", "left"],
  ["top-left", "top left"],
];

/** Registers the transform utilities. */
export function registerTransformUtilities(
  utilities: Utilities,
  theme: Theme,
): void {
  const stat = (name: string, nodes: () => AstNode[]) =>
    staticUtility(utilities, name, nodes);
  const fn = (
    root: string,
    desc: Parameters<typeof functionalUtility>[3],
  ) => functionalUtility(utilities, theme, root, desc);
  const degrees = (value: { value: string }) =>
    isPositiveInteger(value.value) ? `${value.value}deg` : null;

  stat("transform-none", () => [decl("transform", "none")]);
  stat("transform-gpu", () => [
    ...transformRegistrations(),
    decl("transform", `translateZ(0) ${TRANSFORM}`),
  ]);
  stat("transform-cpu", () => [
    ...transformRegistrations(),
    decl("transform", TRANSFORM),
  ]);
  fn("transform", { handle: (value) => [decl("transform", value)] });
  stat("transform-flat", () => [decl("transform-style", "flat")]);
  stat("transform-3d", () => [decl("transform-style", "preserve-3d")]);
  stat("backface-visible", () => [decl("backface-visibility", "visible")]);
  stat("backface-hidden", () => [decl("backface-visibility", "hidden")]);

  stat("perspective-none", () => [decl("perspective", "none")]);
  fn("perspective", {
    themeKeys: ["--perspective"],
    handle: (value) => [decl("perspective", value)],
  });
  for (const [name, value] of ORIGINS) {
    stat(`perspective-origin-${name}`, () => [
      decl("perspective-origin", value),
    ]);
    stat(`origin-${name}`, () => [decl("transform-origin", value)]);
  }
  fn("perspective-origin", {
    handle: (value) => [decl("perspective-origin", value)],
  });
  fn("origin", { handle: (value) => [decl("transform-origin", value)] });

  const translate = (
    name: string,
    variables: string[],
    composed: string,
    fractions: boolean,
  ) => {
    const handle = (value: string): AstNode[] => [
      ...translateRegistrations(),
      ...variables.map((v) => decl(v, value)),
      decl("translate", composed),
    ];
    spacingUtility(
      utilities,
      theme,
      name,
      ["--translate", "--spacing"],
      handle,
      {
        supportsNegative: true,
        supportsFractions: fractions,
      },
    );
    if (fractions) {
      stat(`${name}-full`, () => handle("100%"));
      stat(`-${name}-full`, () => handle("-100%"));
    }
  };
  translate(
    "translate",
    ["--tw-translate-x", "--tw-translate-y"],
    TRANSLATE_2D,
    true,
  );
  translate("translate-x", ["--tw-translate-x"], TRANSLATE_2D, true);
  translate("translate-y", ["--tw-translate-y"], TRANSLATE_2D, true);
  translate("translate-z", ["--tw-translate-z"], TRANSLATE_3D, false);
  stat("translate-3d", () => [
    ...translateRegistrations(),
    decl("translate", TRANSLATE_3D),
  ]);
  stat("translate-none", () => [decl("translate", "none")]);

  const percentage = (value: { value: string }) =>
    isPositiveInteger(value.value) ? `${value.value}%` : null;
  const scaleCompile = (
    candidate: FunctionalCandidate,
    negative: boolean,
  ): AstNode[] | undefined => {
    if (candidate.value === null || candidate.modifier !== null) return;
    if (candidate.value.kind === "arbitrary") {
      if (negative) return;
      return [decl("scale", candidate.value.value)];
    }
    let value = theme.resolve(candidate.value.value, ["--scale"]) ??
      percentage(candidate.value);
    if (value === null) return;
    if (negative) value = `calc(${value} * -1)`;
    return [
      ...scaleRegistrations(),
      decl("--tw-scale-x", value),
      decl("--tw-scale-y", value),
      decl("--tw-scale-z", value),
      decl("scale", SCALE_2D),
    ];
  };
  utilities.functional("scale", (candidate) => scaleCompile(candidate, false));
  utilities.functional("-scale", (candidate) => scaleCompile(candidate, true));
  for (const axis of ["x", "y", "z"]) {
    fn(`scale-${axis}`, {
      themeKeys: ["--scale"],
      supportsNegative: true,
      handleBareValue: percentage,
      handle: (value) => [
        ...scaleRegistrations(),
        decl(`--tw-scale-${axis}`, value),
        decl("scale", axis === "z" ? SCALE_3D : SCALE_2D),
      ],
    });
  }
  stat("scale-3d", () => [...scaleRegistrations(), decl("scale", SCALE_3D)]);
  stat("scale-none", () => [decl("scale", "none")]);

  fn("rotate", {
    themeKeys: ["--rotate"],
    supportsNegative: true,
    handleBareValue: degrees,
    handle: (value) => [decl("rotate", value)],
  });
  stat("rotate-none", () => [decl("rotate", "none")]);
  for (const axis of ["x", "y", "z"]) {
    fn(`rotate-${axis}`, {
      themeKeys: ["--rotate"],
      supportsNegative: true,
      handleBareValue: degrees,
      handle: (value) => [
        ...transformRegistrations(),
        decl(`--tw-rotate-${axis}`, `rotate${axis.toUpperCase()}(${value})`),
        decl("transform", TRANSFORM),
      ],
    });
  }

  const skew = (name: string, axes: string[]) =>
    fn(name, {
      themeKeys: ["--skew"],
      supportsNegative: true,
      handleBareValue: degrees,
      handle: (value) => [
        ...transformRegistrations(),
        ...axes.map((axis) =>
          decl(`--tw-skew-${axis}`, `skew${axis.toUpperCase()}(${value})`)
        ),
        decl("transform", TRANSFORM),
      ],
    });
  skew("skew", ["x", "y"]);
  skew("skew-x", ["x"]);
  skew("skew-y", ["y"]);
}
