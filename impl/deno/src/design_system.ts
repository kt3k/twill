/**
 * The design system: theme, registries, memoization caches, and the set of
 * known-invalid candidates (SPEC §4.1.9).
 *
 * @module
 */

import { registerBuiltinUtilities } from "./builtin_utilities.ts";
import { registerBuiltinVariants } from "./builtin_variants.ts";
import { cloneNode } from "./ast.ts";
import {
  type Candidate,
  parseCandidate as parseCandidateImpl,
  type ParserContext,
  parseVariant as parseVariantImpl,
  type Variant,
} from "./candidate.ts";
import {
  compileAstNodes as compileAstNodesImpl,
  type CompiledRule,
} from "./compile_candidates.ts";
import type { Theme } from "./theme.ts";
import { Utilities } from "./utilities.ts";
import { Variants } from "./variants.ts";

export class DesignSystem implements ParserContext {
  readonly theme: Theme;
  readonly utilities: Utilities = new Utilities();
  readonly variants: Variants = new Variants();
  readonly invalidCandidates: Set<string> = new Set();
  /** When true every generated declaration is marked `!important`. */
  important = false;

  readonly #candidateCache = new Map<string, Candidate[]>();
  readonly #variantCache = new Map<string, Variant | null>();
  readonly #compileCache = new Map<Candidate, Map<number, CompiledRule[]>>();
  #variantOrder: Map<Variant, number> | null = null;

  constructor(theme: Theme) {
    this.theme = theme;
  }

  /** Every interpretation of a raw class name, memoized. */
  parseCandidate(raw: string): Candidate[] {
    let cached = this.#candidateCache.get(raw);
    if (cached === undefined) {
      cached = [...parseCandidateImpl(raw, this)];
      this.#candidateCache.set(raw, cached);
    }
    return cached;
  }

  /** Parses a variant, memoized so that equal inputs share one object. */
  parseVariant(raw: string): Variant | null {
    if (this.#variantCache.has(raw)) return this.#variantCache.get(raw)!;
    const variant = parseVariantImpl(raw, this);
    this.#variantCache.set(raw, variant);
    if (variant !== null) this.#variantOrder = null;
    return variant;
  }

  /**
   * Sorts every parsed variant and assigns an index that increases each time
   * `compare` reports a difference between neighbors (SPEC §9.4).
   */
  getVariantOrder(): Map<Variant, number> {
    if (this.#variantOrder !== null) return this.#variantOrder;
    const parsed: Variant[] = [];
    for (const variant of this.#variantCache.values()) {
      if (variant !== null) parsed.push(variant);
    }
    parsed.sort((a, z) => this.variants.compare(a, z));
    const order = new Map<Variant, number>();
    let index = 0;
    let previous: Variant | null = null;
    for (const variant of parsed) {
      if (previous !== null && this.variants.compare(previous, variant) !== 0) {
        index++;
      }
      order.set(variant, index);
      previous = variant;
    }
    this.#variantOrder = order;
    return order;
  }

  /**
   * Compiles a candidate interpretation, memoized per candidate object and
   * flags. Cached results are cloned so callers may mutate them.
   */
  compileAstNodes(candidate: Candidate, flags: number): CompiledRule[] {
    let byFlags = this.#compileCache.get(candidate);
    if (byFlags === undefined) {
      byFlags = new Map();
      this.#compileCache.set(candidate, byFlags);
    }
    let cached = byFlags.get(flags);
    if (cached === undefined) {
      cached = compileAstNodesImpl(candidate, flags, this);
      byFlags.set(flags, cached);
    }
    return cached.map((r) => ({
      node: cloneNode(r.node),
      propertySort: r.propertySort,
    }));
  }

  /** Drops memoized candidates and compiled results; used after registries change. */
  clearCandidateCache(): void {
    this.#candidateCache.clear();
    this.#compileCache.clear();
  }
}

/** Creates a design system with the built-in variants and utilities. */
export function buildDesignSystem(theme: Theme): DesignSystem {
  const ds = new DesignSystem(theme);
  registerBuiltinVariants(ds.variants, theme);
  registerBuiltinUtilities(ds.utilities, theme);
  return ds;
}
