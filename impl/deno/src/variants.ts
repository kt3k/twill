/**
 * Variant registry, application, and ordering (SPEC §9.1 through §9.4).
 *
 * @module
 */

import {
  type AstNode,
  type AtRule,
  atRule,
  cloneNodes,
  type Rule,
  rule,
  walk,
  WalkAction,
} from "./ast.ts";
import type {
  ArbitraryVariant,
  CompoundVariant,
  FunctionalVariant,
  StaticVariant,
  Variant,
  VariantKind,
} from "./candidate.ts";

/** The kinds of rules a variant generates or accepts (SPEC §4.1.8). */
export const Compounds = {
  NEVER: 0,
  AT_RULES: 1 << 0,
  STYLE_RULES: 1 << 1,
} as const;

export type Compounds = number;

/** A rule node handed to a variant's `apply`. */
export type VariantRule = Rule | AtRule;

/**
 * Mutates `node.nodes` in place, wrapping the existing children in new rules
 * or at-rules. Returning `null` rejects the candidate.
 */
export type VariantApply<T extends Variant = Variant> = (
  node: VariantRule,
  variant: T,
) => null | undefined | void;

export interface VariantDefinition {
  kind: VariantKind;
  /** Registration order. */
  order: number;
  // deno-lint-ignore no-explicit-any
  apply: VariantApply<any>;
  compounds: Compounds;
  compoundsWith: Compounds;
}

export type VariantCompareFn = (a: Variant, z: Variant) => number;

/**
 * Returns `NEVER` if any selector starts with `@` but not with `@media`,
 * `@supports`, or `@container`, or if any selector contains `::`. Otherwise
 * ORs `AT_RULES` for at-rule selectors and `STYLE_RULES` for the rest.
 */
export function compoundsForSelectors(selectors: string[]): Compounds {
  let result: Compounds = Compounds.NEVER;
  for (const selector of selectors) {
    if (selector.startsWith("@")) {
      if (
        !selector.startsWith("@media") && !selector.startsWith("@supports") &&
        !selector.startsWith("@container")
      ) {
        return Compounds.NEVER;
      }
      result |= Compounds.AT_RULES;
      continue;
    }
    if (selector.includes("::")) return Compounds.NEVER;
    result |= Compounds.STYLE_RULES;
  }
  return result;
}

export class Variants {
  readonly #variants = new Map<string, VariantDefinition>();
  readonly #compareFns = new Map<number, VariantCompareFn>();
  #lastOrder = 0;
  #groupOrder: number | null = null;

  #register(
    name: string,
    kind: VariantKind,
    // deno-lint-ignore no-explicit-any
    apply: VariantApply<any>,
    compounds: Compounds,
    compoundsWith: Compounds,
  ): void {
    const existing = this.#variants.get(name);
    if (existing !== undefined) {
      // Re-registering keeps the order.
      existing.kind = kind;
      existing.apply = apply;
      existing.compounds = compounds;
      return;
    }
    let order: number;
    if (this.#groupOrder !== null) {
      order = this.#groupOrder;
    } else {
      order = ++this.#lastOrder;
    }
    this.#variants.set(name, { kind, order, apply, compounds, compoundsWith });
  }

  static(
    name: string,
    apply: VariantApply<StaticVariant>,
    options: { compounds?: Compounds } = {},
  ): void {
    this.#register(
      name,
      "static",
      apply,
      options.compounds ?? Compounds.STYLE_RULES,
      Compounds.NEVER,
    );
  }

  functional(
    name: string,
    apply: VariantApply<FunctionalVariant>,
    options: { compounds?: Compounds } = {},
  ): void {
    this.#register(
      name,
      "functional",
      apply,
      options.compounds ?? Compounds.STYLE_RULES,
      Compounds.NEVER,
    );
  }

  compound(
    name: string,
    compoundsWith: Compounds,
    apply: VariantApply<CompoundVariant>,
    options: { compounds?: Compounds } = {},
  ): void {
    this.#register(
      name,
      "compound",
      apply,
      options.compounds ?? Compounds.STYLE_RULES,
      compoundsWith,
    );
  }

  /**
   * Every name registered by `fn` shares one order; `compareFn` is stored
   * for that order.
   */
  group(fn: () => void, compareFn?: VariantCompareFn): void {
    this.#groupOrder = ++this.#lastOrder;
    if (compareFn) this.#compareFns.set(this.#groupOrder, compareFn);
    try {
      fn();
    } finally {
      this.#groupOrder = null;
    }
  }

  has(name: string): boolean {
    return this.#variants.has(name);
  }

  get(name: string): VariantDefinition | undefined {
    return this.#variants.get(name);
  }

  kind(name: string): VariantKind | undefined {
    return this.#variants.get(name)?.kind;
  }

  keys(): string[] {
    return [...this.#variants.keys()];
  }

  entries(): [string, VariantDefinition][] {
    return [...this.#variants.entries()];
  }

  /** The `compounds` value of a parsed variant. */
  compoundsOf(variant: Variant): Compounds {
    if (variant.kind === "arbitrary") {
      return compoundsForSelectors([variant.selector]);
    }
    return this.#variants.get(variant.root)?.compounds ?? Compounds.NEVER;
  }

  /** Whether the compound variant `parent` accepts `child` (SPEC §9.1). */
  compoundsWith(parent: string, child: Variant): boolean {
    const definition = this.#variants.get(parent);
    if (definition === undefined || definition.kind !== "compound") {
      return false;
    }
    const childCompounds = this.compoundsOf(child);
    if (childCompounds === Compounds.NEVER) return false;
    if (definition.compoundsWith === Compounds.NEVER) return false;
    return (childCompounds & definition.compoundsWith) !== 0;
  }

  /** Compares two parsed variants for output ordering (SPEC §9.4). */
  compare(a: Variant | null, z: Variant | null): number {
    if (a === z) return 0;
    if (a === null) return -1;
    if (z === null) return 1;

    if (a.kind === "arbitrary" && z.kind === "arbitrary") {
      return a.selector < z.selector ? -1 : a.selector > z.selector ? 1 : 0;
    }
    if (a.kind === "arbitrary") return 1;
    if (z.kind === "arbitrary") return -1;

    const aOrder = this.#variants.get(a.root)?.order ?? Number.MAX_SAFE_INTEGER;
    const zOrder = this.#variants.get(z.root)?.order ?? Number.MAX_SAFE_INTEGER;
    if (aOrder !== zOrder) return aOrder - zOrder;

    if (a.kind === "compound" && z.kind === "compound") {
      const inner = this.compare(a.variant, z.variant);
      if (inner !== 0) return inner;
      if (a.modifier && z.modifier) {
        return a.modifier.value < z.modifier.value
          ? -1
          : a.modifier.value > z.modifier.value
          ? 1
          : 0;
      }
      if (a.modifier) return 1;
      if (z.modifier) return -1;
      return 0;
    }

    const compareFn = this.#compareFns.get(aOrder);
    if (compareFn !== undefined) return compareFn(a, z);

    if (a.root !== z.root) return a.root < z.root ? -1 : 1;

    const aValue = a.kind === "functional" ? a.value : null;
    const zValue = z.kind === "functional" ? z.value : null;
    if (aValue === null && zValue === null) return 0;
    if (aValue === null) return -1;
    if (zValue === null) return 1;
    if (aValue.kind !== zValue.kind) {
      return aValue.kind === "arbitrary" ? 1 : -1;
    }
    return aValue.value < zValue.value
      ? -1
      : aValue.value > zValue.value
      ? 1
      : 0;
  }
}

/**
 * The standard static helper: sets `node.nodes` to one `rule(selector,
 * children)` per selector.
 */
export function staticVariant(
  variants: Variants,
  name: string,
  selectors: string[],
  options: { compounds?: Compounds } = {},
): void {
  variants.static(
    name,
    (node) => {
      node.nodes = selectors.map((selector) => rule(selector, node.nodes));
    },
    { compounds: options.compounds ?? compoundsForSelectors(selectors) },
  );
}

/** Applies a parsed variant to a rule node (SPEC §9.3). */
export function applyVariant(
  node: VariantRule,
  variant: Variant,
  variants: Variants,
  depth = 0,
): null | undefined {
  if (variant.kind === "arbitrary") {
    if (variant.relative && depth === 0) return null;
    node.nodes = [rule((variant as ArbitraryVariant).selector, node.nodes)];
    return;
  }

  const definition = variants.get(variant.root);
  if (definition === undefined) return null;

  if (variant.kind === "compound") {
    const isolated = atRule("@slot", "", []);
    if (applyVariant(isolated, variant.variant, variants, depth + 1) === null) {
      return null;
    }
    if (variant.root === "not" && isolated.nodes.length > 1) return null;

    for (const child of isolated.nodes) {
      if (child.kind !== "rule" && child.kind !== "at-rule") return null;
      if (definition.apply(child, variant) === null) return null;
    }

    // Give every leaf rule the original children. Leaves after the first
    // receive a clone so that no two leaves share one node list.
    let first = true;
    const fill = (nodes: AstNode[]) => {
      for (const child of nodes) {
        if (child.kind !== "rule" && child.kind !== "at-rule") continue;
        if (child.nodes.length === 0) {
          child.nodes = first ? node.nodes : cloneNodes(node.nodes);
          first = false;
        } else {
          fill(child.nodes);
        }
      }
    };
    fill(isolated.nodes);
    node.nodes = isolated.nodes;
    return;
  }

  return definition.apply(node, variant) === null ? null : undefined;
}

/** Whether a rule tree contains nested style rules. */
export function hasNestedStyleRules(nodes: AstNode[]): boolean {
  let found = false;
  walk(nodes, (child) => {
    if (child.kind === "rule") {
      found = true;
      return WalkAction.Stop;
    }
  });
  return found;
}
