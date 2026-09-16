/**
 * Candidate grammar and parsing (SPEC §8).
 *
 * @module
 */

import {
  decodeArbitraryValue,
  isValidArbitrary,
  NAMED_VALUE_PATTERN,
  segment,
} from "./utils.ts";

// Candidate entities (SPEC §4.1.3 through §4.1.6).

export interface NamedValue {
  kind: "named";
  value: string;
  /** For example `1/2` when a slash segment could be a fraction. */
  fraction: string | null;
}

export interface ArbitraryValue {
  kind: "arbitrary";
  /** The decoded value. */
  value: string;
  /** An explicit type hint such as `color`. */
  dataType: string | null;
}

export type CandidateValue = NamedValue | ArbitraryValue;

export interface NamedModifier {
  kind: "named";
  value: string;
}

export interface ArbitraryModifier {
  kind: "arbitrary";
  /** The decoded value; variable shorthands are stored as `var(--name)`. */
  value: string;
}

export type CandidateModifier = NamedModifier | ArbitraryModifier;

export interface StaticVariant {
  kind: "static";
  root: string;
}

export interface FunctionalVariant {
  kind: "functional";
  root: string;
  value: { kind: "named" | "arbitrary"; value: string } | null;
  modifier: CandidateModifier | null;
}

export interface CompoundVariant {
  kind: "compound";
  root: string;
  modifier: CandidateModifier | null;
  variant: Variant;
}

export interface ArbitraryVariant {
  kind: "arbitrary";
  selector: string;
  /** True when the selector starts with `>`, `+`, or `~`. */
  relative: boolean;
}

export type Variant =
  | StaticVariant
  | FunctionalVariant
  | CompoundVariant
  | ArbitraryVariant;

interface CandidateBase {
  /** The original class name including variants and the important marker. */
  raw: string;
  /** In application order: the rightmost variant in the source text is first. */
  variants: Variant[];
  important: boolean;
}

export interface StaticCandidate extends CandidateBase {
  kind: "static";
  root: string;
}

export interface FunctionalCandidate extends CandidateBase {
  kind: "functional";
  root: string;
  value: CandidateValue | null;
  modifier: CandidateModifier | null;
}

export interface ArbitraryCandidate extends CandidateBase {
  kind: "arbitrary";
  property: string;
  value: string;
  modifier: CandidateModifier | null;
}

export type Candidate =
  | StaticCandidate
  | FunctionalCandidate
  | ArbitraryCandidate;

export type UtilityKind = "static" | "functional";
export type VariantKind = "static" | "functional" | "compound";

/**
 * The part of the design system the parsers consult.
 */
export interface ParserContext {
  theme: { prefix: string | null };
  utilities: { has(name: string, kind: UtilityKind): boolean };
  variants: {
    has(name: string): boolean;
    kind(name: string): VariantKind | undefined;
    compoundsWith(parent: string, child: Variant): boolean;
  };
  /** Memoized variant parsing; must return the same object for the same input. */
  parseVariant(input: string): Variant | null;
}

/**
 * Yields zero or more interpretations of a raw class name (SPEC §8.2).
 */
export function* parseCandidate(
  input: string,
  ds: ParserContext,
): Generator<Candidate> {
  // 1. Split off the variants.
  const rawVariants = segment(input, ":");

  // 2. Prefix enforcement.
  if (ds.theme.prefix !== null) {
    if (rawVariants.length === 1) return;
    if (rawVariants[0] !== ds.theme.prefix) return;
    rawVariants.shift();
  }

  let base = rawVariants.pop()!;

  // 3. Parse the variants from right to left.
  const variants: Variant[] = [];
  for (let i = rawVariants.length - 1; i >= 0; i--) {
    const variant = ds.parseVariant(rawVariants[i]);
    if (variant === null) return;
    variants.push(variant);
  }

  // 4. Importance.
  let important = false;
  if (base.endsWith("!")) {
    important = true;
    base = base.slice(0, -1);
  } else if (base.startsWith("!")) {
    important = true;
    base = base.slice(1);
  }

  // 5. Static utility.
  if (ds.utilities.has(base, "static") && !base.includes("[")) {
    yield { kind: "static", root: base, variants, important, raw: input };
  }

  // 6. Modifier segment.
  const parts = segment(base, "/");
  if (parts.length > 2) return;
  const baseWithoutModifier = parts[0];
  const modifierSegment = parts.length === 2 ? parts[1] : null;

  // 7. Parse the modifier.
  const modifier = modifierSegment === null
    ? null
    : parseModifier(modifierSegment);
  if (modifierSegment !== null && modifier === null) return;

  // 8. Arbitrary property.
  if (baseWithoutModifier.charCodeAt(0) === 0x5b /* [ */) {
    if (!baseWithoutModifier.endsWith("]")) return;
    const second = baseWithoutModifier.charCodeAt(1);
    if (!(second === 0x2d || (second >= 0x61 && second <= 0x7a))) return;
    const inner = baseWithoutModifier.slice(1, -1);
    const colon = inner.indexOf(":");
    if (colon === -1 || colon === 0 || colon === inner.length - 1) return;
    const property = inner.slice(0, colon);
    const value = decodeArbitraryValue(inner.slice(colon + 1));
    if (!isValidArbitrary(value)) return;
    yield {
      kind: "arbitrary",
      property,
      value,
      modifier,
      variants,
      important,
      raw: input,
    };
    return;
  }

  let roots: Iterable<[string, string | null]>;

  if (baseWithoutModifier.endsWith("]")) {
    // 9. Arbitrary value.
    const idx = baseWithoutModifier.indexOf("-[");
    if (idx === -1) return;
    const root = baseWithoutModifier.slice(0, idx);
    if (!ds.utilities.has(root, "functional")) return;
    roots = [[root, baseWithoutModifier.slice(idx + 1)]];
  } else if (baseWithoutModifier.endsWith(")")) {
    // 10. Variable shorthand.
    const idx = baseWithoutModifier.indexOf("-(");
    if (idx === -1) return;
    const root = baseWithoutModifier.slice(0, idx);
    if (!ds.utilities.has(root, "functional")) return;
    const inner = baseWithoutModifier.slice(idx + 2, -1);
    const innerParts = segment(inner, ":");
    let dataType: string | null = null;
    let value = inner;
    if (innerParts.length === 2) {
      dataType = innerParts[0];
      value = innerParts[1];
    } else if (innerParts.length > 2) {
      return;
    }
    if (!value.startsWith("--") || !isValidArbitrary(value)) return;
    roots = [[
      root,
      dataType === null ? `[var(${value})]` : `[${dataType}:var(${value})]`,
    ]];
  } else {
    // 11. Named value.
    roots = findRoots(
      baseWithoutModifier,
      (root) => ds.utilities.has(root, "functional"),
    );
  }

  // 12. One candidate per root interpretation.
  for (const [root, value] of roots) {
    const candidate: FunctionalCandidate = {
      kind: "functional",
      root,
      modifier,
      value: null,
      variants,
      important,
      raw: input,
    };

    if (value === null) {
      yield candidate;
      continue;
    }

    const bracket = value.indexOf("[");
    if (bracket !== -1) {
      if (!value.endsWith("]")) return;
      const decoded = decodeArbitraryValue(value.slice(bracket + 1, -1));
      if (!isValidArbitrary(decoded)) continue;

      let dataType: string | null = null;
      let arbitraryValue = decoded;
      let i = 0;
      while (i < decoded.length) {
        const c = decoded.charCodeAt(i);
        if (c === 0x2d || (c >= 0x61 && c <= 0x7a)) i++;
        else break;
      }
      if (decoded.charCodeAt(i) === 0x3a /* : */) {
        dataType = decoded.slice(0, i);
        arbitraryValue = decoded.slice(i + 1);
        if (dataType === "") continue;
      }
      if (arbitraryValue.trim() === "") continue;
      candidate.value = { kind: "arbitrary", dataType, value: arbitraryValue };
    } else {
      const fraction = modifierSegment !== null && modifier?.kind === "named"
        ? `${value}/${modifierSegment}`
        : null;
      if (!NAMED_VALUE_PATTERN.test(value)) continue;
      candidate.value = { kind: "named", value, fraction };
    }

    yield candidate;
  }
}

/** Parses a modifier segment (SPEC §8.2.1). */
export function parseModifier(modifier: string): CandidateModifier | null {
  if (modifier.startsWith("[") && modifier.endsWith("]")) {
    const value = decodeArbitraryValue(modifier.slice(1, -1));
    if (!isValidArbitrary(value) || value.trim() === "") return null;
    return { kind: "arbitrary", value };
  }
  if (modifier.startsWith("(") && modifier.endsWith(")")) {
    const value = decodeArbitraryValue(modifier.slice(1, -1));
    if (!value.startsWith("--") || !isValidArbitrary(value)) return null;
    return { kind: "arbitrary", value: `var(${value})` };
  }
  if (NAMED_VALUE_PATTERN.test(modifier)) {
    return { kind: "named", value: modifier };
  }
  return null;
}

/**
 * Yields every `(root, value)` split of `input` whose root exists
 * (SPEC §8.2.2).
 */
export function* findRoots(
  input: string,
  exists: (root: string) => boolean,
): Generator<[string, string | null]> {
  if (exists(input)) yield [input, null];

  let idx = input.lastIndexOf("-");
  if (idx !== -1) {
    do {
      const root = input.slice(0, idx);
      if (exists(root)) {
        const value = input.slice(idx + 1);
        if (value === "") break;
        if (root === "@") break;
        yield [root, value];
      }
      idx = input.lastIndexOf("-", idx - 1);
    } while (idx > 0);
  }

  if (input.startsWith("@") && exists("@")) {
    yield ["@", input.slice(1)];
  }
}

/** Parses a variant segment (SPEC §8.3). */
export function parseVariant(input: string, ds: ParserContext): Variant | null {
  // 1. Arbitrary variant.
  if (input.startsWith("[") && input.endsWith("]")) {
    if (input.charCodeAt(1) === 0x40 /* @ */ && input.includes("&")) {
      return null;
    }
    const selector = decodeArbitraryValue(input.slice(1, -1));
    if (!isValidArbitrary(selector) || selector.trim() === "") return null;
    const first = selector[0];
    const relative = first === ">" || first === "+" || first === "~";
    let result = selector;
    if (!relative && first !== "@" && !selector.includes("&")) {
      result = `&:is(${selector})`;
    }
    return { kind: "arbitrary", selector: result, relative };
  }

  // 2. Modifier.
  const parts = segment(input, "/");
  if (parts.length > 2) return null;
  const name = parts[0];
  const modifierText = parts.length === 2 ? parts[1] : null;

  // 3. Roots.
  for (const [root, value] of findRoots(name, (r) => ds.variants.has(r))) {
    switch (ds.variants.kind(root)) {
      case "static": {
        if (value !== null || modifierText !== null) return null;
        return { kind: "static", root };
      }
      case "functional": {
        const modifier = modifierText === null
          ? null
          : parseModifier(modifierText);
        if (modifierText !== null && modifier === null) return null;
        if (value === null) {
          return { kind: "functional", root, value: null, modifier };
        }
        if (value.endsWith("]")) {
          if (!value.startsWith("[")) continue;
          const decoded = decodeArbitraryValue(value.slice(1, -1));
          if (!isValidArbitrary(decoded) || decoded.trim() === "") return null;
          return {
            kind: "functional",
            root,
            value: { kind: "arbitrary", value: decoded },
            modifier,
          };
        }
        if (value.endsWith(")")) {
          if (!value.startsWith("(")) continue;
          const decoded = decodeArbitraryValue(value.slice(1, -1));
          if (
            !isValidArbitrary(decoded) || decoded.trim() === "" ||
            !decoded.startsWith("--")
          ) {
            return null;
          }
          return {
            kind: "functional",
            root,
            value: { kind: "arbitrary", value: `var(${decoded})` },
            modifier,
          };
        }
        if (!NAMED_VALUE_PATTERN.test(value)) continue;
        return {
          kind: "functional",
          root,
          value: { kind: "named", value },
          modifier,
        };
      }
      case "compound": {
        if (value === null) return null;
        let innerText = value;
        let remainingModifier = modifierText;
        if (
          (root === "not" || root === "has" || root === "in") &&
          modifierText !== null
        ) {
          innerText = `${value}/${modifierText}`;
          remainingModifier = null;
        }
        const inner = ds.parseVariant(innerText);
        if (inner === null) return null;
        if (!ds.variants.compoundsWith(root, inner)) return null;
        const modifier = remainingModifier === null
          ? null
          : parseModifier(remainingModifier);
        if (remainingModifier !== null && modifier === null) return null;
        return { kind: "compound", root, modifier, variant: inner };
      }
    }
  }

  return null;
}
