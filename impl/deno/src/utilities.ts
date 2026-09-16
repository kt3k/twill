/**
 * Utility registry, definition helpers, and color handling (SPEC §10.1
 * through §10.4).
 *
 * @module
 */

import { type AstNode, cloneNodes, decl } from "./ast.ts";
import type {
  Candidate,
  CandidateModifier,
  FunctionalCandidate,
  NamedValue,
  UtilityKind,
} from "./candidate.ts";
import { withAlpha } from "./color.ts";
import type { DataType } from "./data_types.ts";
import type { Theme } from "./theme.ts";
import { isMultipleOfQuarter, isPositiveInteger } from "./utils.ts";

export { propertyRegistration } from "./builtin_variants.ts";

/**
 * A list of nodes means success, `undefined` means this definition does not
 * handle the candidate, and `null` means the candidate is invalid for this
 * definition (SPEC §4.1.7).
 */
export type CompileResult = AstNode[] | undefined | null;

export interface UtilityDefinition {
  kind: UtilityKind;
  // deno-lint-ignore no-explicit-any
  compile: (candidate: any) => CompileResult;
  /**
   * A definition whose `types` has more than one entry and includes `any` is
   * a fallback definition tried only after all others fail.
   */
  types?: string[];
}

export class Utilities {
  readonly #utilities = new Map<string, UtilityDefinition[]>();

  static(name: string, compile: (candidate: Candidate) => CompileResult): void {
    this.#add(name, { kind: "static", compile });
  }

  functional(
    name: string,
    compile: (candidate: FunctionalCandidate) => CompileResult,
    options: { types?: string[] } = {},
  ): void {
    this.#add(name, { kind: "functional", compile, types: options.types });
  }

  #add(name: string, definition: UtilityDefinition): void {
    const list = this.#utilities.get(name);
    if (list === undefined) this.#utilities.set(name, [definition]);
    else list.push(definition);
  }

  has(name: string, kind: UtilityKind): boolean {
    const list = this.#utilities.get(name);
    if (list === undefined) return false;
    return list.some((d) => d.kind === kind);
  }

  get(name: string): UtilityDefinition[] {
    return this.#utilities.get(name) ?? [];
  }

  keys(kind?: UtilityKind): string[] {
    const keys: string[] = [];
    for (const [name, list] of this.#utilities) {
      if (kind === undefined || list.some((d) => d.kind === kind)) {
        keys.push(name);
      }
    }
    return keys;
  }
}

/** Whether a definition is a fallback definition (SPEC §4.1.7). */
export function isFallbackDefinition(definition: UtilityDefinition): boolean {
  return definition.types !== undefined && definition.types.length > 1 &&
    definition.types.includes("any");
}

// Colors and opacity (SPEC §10.3).

/** Applies a candidate modifier to a color value. */
export function asColor(
  value: string,
  modifier: CandidateModifier | null,
  theme: Theme,
): string | null {
  if (modifier === null) return value;
  if (modifier.kind === "arbitrary") return withAlpha(value, modifier.value);
  const opacity = theme.resolve(modifier.value, ["--opacity"]);
  if (opacity !== null) return withAlpha(value, opacity);
  if (!isMultipleOfQuarter(modifier.value)) return null;
  return withAlpha(value, `${modifier.value}%`);
}

/** Resolves a named color candidate value through the theme. */
export function resolveThemeColor(
  candidate: FunctionalCandidate,
  themeKeys: string[],
  theme: Theme,
): string | null {
  if (candidate.value === null || candidate.value.kind !== "named") return null;
  let value: string | null;
  switch (candidate.value.value) {
    case "inherit":
      value = "inherit";
      break;
    case "transparent":
      value = "transparent";
      break;
    case "current":
      value = "currentcolor";
      break;
    default:
      value = theme.resolve(candidate.value.value, themeKeys);
  }
  if (value === null) return null;
  return asColor(value, candidate.modifier, theme);
}

// Definition helpers (SPEC §10.2).

export type Declarations = [string, string][];

/**
 * Registers a static definition returning the given `(property, value)`
 * pairs as declarations, or the result of calling a supplied function.
 */
export function staticUtility(
  utilities: Utilities,
  name: string,
  declarations: Declarations | (() => AstNode[]),
): void {
  utilities.static(name, () => {
    if (typeof declarations === "function") return declarations();
    return declarations.map(([property, value]) => decl(property, value));
  });
}

export interface FunctionalUtilityDescription {
  /** Also register `-<root>`; its values are wrapped as `calc(<value> * -1)`. */
  supportsNegative?: boolean;
  /** A named value with a fraction resolves to `calc(<a> / <b> * 100%)`. */
  supportsFractions?: boolean;
  /** Namespaces used to resolve named values and the default value. */
  themeKeys?: string[];
  /**
   * The value for a candidate without a value segment. When absent, the
   * first namespace itself is resolved.
   */
  defaultValue?: string | null;
  /** Consulted last, only for non-negative candidates without a modifier. */
  staticValues?: Record<string, AstNode[]>;
  /** A value string for a named value that is not in the theme, or null. */
  handleBareValue?: (value: NamedValue) => string | null;
  handleNegativeBareValue?: (value: NamedValue) => string | null;
  /** The declarations for a final value. */
  handle: (
    value: string,
    dataType: string | null,
  ) => AstNode[] | undefined | null;
}

/** Registers a functional definition (SPEC §10.2). */
export function functionalUtility(
  utilities: Utilities,
  theme: Theme,
  root: string,
  desc: FunctionalUtilityDescription,
): void {
  const themeKeys = desc.themeKeys ?? [];

  const compile = (
    candidate: FunctionalCandidate,
    negative: boolean,
  ): CompileResult => {
    let value: string | null = null;
    let dataType: string | null = null;

    if (candidate.value === null) {
      if (candidate.modifier !== null) return;
      value = desc.defaultValue !== undefined
        ? desc.defaultValue
        : theme.resolve(null, themeKeys);
    } else if (candidate.value.kind === "arbitrary") {
      if (candidate.modifier !== null) return;
      value = candidate.value.value;
      dataType = candidate.value.dataType;
    } else {
      const named = candidate.value;
      let fractionConsumed = false;
      if (named.fraction !== null) {
        value = theme.resolve(named.fraction, themeKeys);
        if (value !== null) fractionConsumed = true;
      }
      if (value === null) {
        value = theme.resolve(named.value, themeKeys);
        if (
          value !== null && candidate.modifier !== null && !fractionConsumed
        ) return;
      }

      if (value === null && desc.supportsFractions && named.fraction !== null) {
        const [a, b] = named.fraction.split("/");
        if (!isPositiveInteger(a) || !isPositiveInteger(b)) return;
        value = `calc(${a} / ${b} * 100%)`;
      }

      if (value === null && negative && desc.handleNegativeBareValue) {
        const bare = desc.handleNegativeBareValue(named);
        if (bare !== null) {
          if (!bare.includes("/") && candidate.modifier !== null) return;
          return desc.handle(bare, null);
        }
      }

      if (value === null && desc.handleBareValue) {
        value = desc.handleBareValue(named);
        if (
          value !== null && !value.includes("/") && candidate.modifier !== null
        ) return;
      }

      if (
        value === null && !negative && candidate.modifier === null &&
        desc.staticValues !== undefined && named.value in desc.staticValues
      ) {
        return cloneNodes(desc.staticValues[named.value]);
      }
    }

    if (value === null) return;
    if (negative) value = `calc(${value} * -1)`;
    return desc.handle(value, dataType);
  };

  utilities.functional(root, (candidate) => compile(candidate, false));
  if (desc.supportsNegative) {
    utilities.functional(`-${root}`, (candidate) => compile(candidate, true));
  }
}

/**
 * Registers a functional definition that requires a value and resolves it as
 * a color (SPEC §10.2).
 */
export function colorUtility(
  utilities: Utilities,
  theme: Theme,
  root: string,
  options: { themeKeys: string[]; handle: (value: string) => AstNode[] },
): void {
  utilities.functional(root, (candidate) => {
    if (candidate.value === null) return;
    let value: string | null;
    if (candidate.value.kind === "arbitrary") {
      value = asColor(candidate.value.value, candidate.modifier, theme);
    } else {
      value = resolveThemeColor(candidate, options.themeKeys, theme);
    }
    if (value === null) return;
    return options.handle(value);
  });
}

/**
 * Registers `<name>-px`, optionally `-<name>-px`, and a functional utility
 * whose bare values are spacing multipliers (SPEC §10.2).
 */
export function spacingUtility(
  utilities: Utilities,
  theme: Theme,
  name: string,
  themeKeys: string[],
  handle: (value: string) => AstNode[],
  options: { supportsNegative?: boolean; supportsFractions?: boolean } = {},
): void {
  staticUtility(utilities, `${name}-px`, () => handle("1px"));
  if (options.supportsNegative) {
    staticUtility(utilities, `-${name}-px`, () => handle("-1px"));
  }
  functionalUtility(utilities, theme, name, {
    themeKeys,
    defaultValue: null,
    supportsNegative: options.supportsNegative,
    supportsFractions: options.supportsFractions,
    handleBareValue: (value) => {
      if (!isMultipleOfQuarter(value.value)) return null;
      if (theme.resolve(null, ["--spacing"]) === null) return null;
      return `--spacing(${value.value})`;
    },
    handleNegativeBareValue: (value) => {
      if (!isMultipleOfQuarter(value.value)) return null;
      if (theme.resolve(null, ["--spacing"]) === null) return null;
      return `--spacing(-${value.value})`;
    },
    handle: (value) => handle(value),
  });
}

/** Resolves a bare value as `<n>` when it is a positive integer. */
export function bareInteger(value: NamedValue): string | null {
  return isPositiveInteger(value.value) ? value.value : null;
}

export type { DataType };
