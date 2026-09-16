/**
 * Candidate compilation and ordering (SPEC §11).
 *
 * @module
 */

import {
  type AstNode,
  decl,
  type Rule,
  styleRule,
  walk,
  WalkAction,
} from "./ast.ts";
import { substituteAtVariant } from "./at_variant.ts";
import type { Candidate } from "./candidate.ts";
import type { DesignSystem } from "./design_system.ts";
import { TwillError } from "./error.ts";
import { propertyIndex } from "./property_order.ts";
import { substituteFunctions } from "./theme_functions.ts";
import { asColor, isFallbackDefinition } from "./utilities.ts";
import { compare, escape } from "./utils.ts";
import { applyVariant } from "./variants.ts";

export const CompileFlags = {
  NONE: 0,
  /** Apply the design-system-wide `important` flag. */
  RESPECT_IMPORTANT: 1 << 0,
} as const;

export interface PropertySort {
  /** Sorted indices into the global property order. */
  order: number[];
  /** The number of declarations with a defined value. */
  count: number;
}

export interface CompiledRule {
  node: Rule;
  propertySort: PropertySort;
}

/** Computes the property sort key of a node list (SPEC §11.3). */
export function getPropertySort(nodes: AstNode[]): PropertySort {
  const indices = new Set<number>();
  let count = 0;
  const queue: AstNode[] = [...nodes];
  let seenSort = false;
  outer: while (queue.length > 0) {
    const node = queue.shift()!;
    switch (node.kind) {
      case "declaration": {
        if (node.value === undefined) continue;
        count++;
        if (seenSort) continue;
        if (node.property === "--tw-sort") {
          const index = propertyIndex(node.value);
          if (index !== -1) {
            indices.add(index);
            seenSort = true;
            break outer;
          }
          continue;
        }
        const index = propertyIndex(node.property);
        if (index !== -1) indices.add(index);
        break;
      }
      case "rule":
      case "at-rule":
      case "context":
        queue.push(...node.nodes);
        break;
      default:
        break;
    }
  }
  return { order: [...indices].sort((a, z) => a - z), count };
}

/** Compiles one candidate interpretation (SPEC §11.2). */
export function compileAstNodes(
  candidate: Candidate,
  flags: number,
  ds: DesignSystem,
): CompiledRule[] {
  const nodeLists = compileBaseUtility(candidate, ds);
  if (nodeLists.length === 0) return [];

  const important = candidate.important ||
    (ds.important && (flags & CompileFlags.RESPECT_IMPORTANT) !== 0);

  const results: CompiledRule[] = [];
  for (const nodes of nodeLists) {
    const propertySort = getPropertySort(nodes);
    if (important) applyImportant(nodes);

    const node = styleRule(`.${escape(candidate.raw)}`, nodes);
    for (const variant of candidate.variants) {
      if (applyVariant(node, variant, ds.variants) === null) return [];
    }

    try {
      substituteFunctions([node], ds);
      substituteAtVariant([node], ds);
    } catch (error) {
      if (error instanceof TwillError) return [];
      throw error;
    }
    results.push({ node, propertySort });
  }
  return results;
}

function compileBaseUtility(
  candidate: Candidate,
  ds: DesignSystem,
): AstNode[][] {
  if (candidate.kind === "arbitrary") {
    const value = asColor(candidate.value, candidate.modifier, ds.theme);
    if (value === null) return [];
    return [[decl(candidate.property, value)]];
  }

  const definitions = ds.utilities.get(candidate.root).filter((d) =>
    d.kind === candidate.kind
  );
  const ordered = [
    ...definitions.filter((d) => !isFallbackDefinition(d)),
    ...definitions.filter(isFallbackDefinition),
  ];
  const results: AstNode[][] = [];
  for (const definition of ordered) {
    const result = definition.compile(candidate);
    if (result === undefined) continue;
    if (result === null) {
      if (definition.types !== undefined) break;
      continue;
    }
    results.push(result);
  }
  return results;
}

/** Marks every declaration `!important` except those inside `at-root` nodes. */
function applyImportant(nodes: AstNode[]): void {
  walk(nodes, (node) => {
    if (node.kind === "at-root") return WalkAction.Skip;
    if (node.kind === "declaration") node.important = true;
  });
}

export interface CompileCandidatesOptions {
  /** Whether the design-system-wide `important` flag applies. Default true. */
  respectImportant?: boolean;
  /** Receives each invalid raw candidate. */
  onInvalidCandidate?: (candidate: string) => void;
}

export interface CompiledCandidates {
  /** Sorted rule nodes. */
  nodes: Rule[];
  /** The sort key of every node, for callers that merge lists. */
  sorting: Map<
    Rule,
    { propertySort: PropertySort; variantOrder: bigint; candidate: string }
  >;
}

/** Compiles and sorts a list of raw candidates (SPEC §11.1, §15.3). */
export function compileCandidates(
  rawCandidates: Iterable<string>,
  ds: DesignSystem,
  options: CompileCandidatesOptions = {},
): CompiledCandidates {
  const onInvalid = options.onInvalidCandidate ?? (() => {});
  const flags = options.respectImportant === false
    ? CompileFlags.NONE
    : CompileFlags.RESPECT_IMPORTANT;

  const matches: [string, Candidate[]][] = [];
  for (const raw of rawCandidates) {
    if (ds.invalidCandidates.has(raw)) {
      onInvalid(raw);
      continue;
    }
    const parsed = ds.parseCandidate(raw);
    if (parsed.length === 0) {
      onInvalid(raw);
      continue;
    }
    matches.push([raw, parsed]);
  }

  const order = ds.getVariantOrder();
  const rules: {
    node: Rule;
    propertySort: PropertySort;
    variantOrder: bigint;
    candidate: string;
  }[] = [];

  for (const [raw, candidates] of matches) {
    let found = false;
    for (const candidate of candidates) {
      for (
        const { node, propertySort } of ds.compileAstNodes(candidate, flags)
      ) {
        found = true;
        let variantOrder = 0n;
        for (const variant of candidate.variants) {
          variantOrder |= 1n << BigInt(order.get(variant) ?? 0);
        }
        rules.push({ node, propertySort, variantOrder, candidate: raw });
      }
    }
    if (!found) onInvalid(raw);
  }

  rules.sort((a, z) => {
    if (a.variantOrder !== z.variantOrder) {
      return a.variantOrder < z.variantOrder ? -1 : 1;
    }
    const length = Math.max(
      a.propertySort.order.length,
      z.propertySort.order.length,
    );
    for (let i = 0; i < length; i++) {
      const ai = a.propertySort.order[i] ?? Infinity;
      const zi = z.propertySort.order[i] ?? Infinity;
      if (ai !== zi) return ai < zi ? -1 : 1;
    }
    if (a.propertySort.count !== z.propertySort.count) {
      return z.propertySort.count - a.propertySort.count;
    }
    return compare(a.candidate, z.candidate);
  });

  const sorting = new Map<
    Rule,
    { propertySort: PropertySort; variantOrder: bigint; candidate: string }
  >();
  for (const rule of rules) {
    sorting.set(rule.node, {
      propertySort: rule.propertySort,
      variantOrder: rule.variantOrder,
      candidate: rule.candidate,
    });
  }
  return { nodes: rules.map((r) => r.node), sorting };
}
