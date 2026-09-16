/**
 * `@apply` expansion (SPEC §6.10).
 *
 * @module
 */

import {
  type AstNode,
  type AtRule,
  type ParentNode,
  walk,
  WalkAction,
} from "./ast.ts";
import { compileCandidates } from "./compile_candidates.ts";
import type { DesignSystem } from "./design_system.ts";
import { TwillError } from "./error.ts";
import { Features } from "./features.ts";
import { segment } from "./utils.ts";

/**
 * Expands every `@apply` in place. `@apply` inside `@utility` bodies is
 * expanded first, in dependency order, so that custom utilities may
 * reference each other. Returns the feature flags.
 */
export function substituteAtApply(ast: AstNode[], ds: DesignSystem): number {
  let features = Features.NONE;

  // Phase 1: `@utility` bodies in topological order.
  const utilityNodes = new Map<string, AtRule>();
  walk(ast, (node) => {
    if (node.kind === "at-rule" && node.name === "@utility") {
      utilityNodes.set(utilityRoot(node.params), node);
      return WalkAction.Skip;
    }
  });

  if (utilityNodes.size > 0) {
    const dependencies = new Map<string, Set<string>>();
    for (const [root, node] of utilityNodes) {
      const deps = new Set<string>();
      walk(node.nodes, (child) => {
        if (child.kind !== "at-rule" || child.name !== "@apply") return;
        for (const candidate of applyCandidates(child)) {
          for (const parsed of ds.parseCandidate(candidate)) {
            if (parsed.kind === "arbitrary") continue;
            if (utilityNodes.has(parsed.root) && parsed.root !== root) {
              deps.add(parsed.root);
            }
            if (utilityNodes.has(parsed.root) && parsed.root === root) {
              throw new TwillError(
                `You cannot \`@apply\` the \`${candidate}\` utility here because it creates a circular dependency.`,
              );
            }
          }
        }
      });
      dependencies.set(root, deps);
    }

    for (
      const root of topologicalSort(dependencies, (cycle) => {
        throw new TwillError(
          `You cannot \`@apply\` the \`${cycle}\` utility here because it creates a circular dependency.`,
        );
      })
    ) {
      const node = utilityNodes.get(root)!;
      features |= expandIn(node.nodes, node, ds);
    }
  }

  // Phase 2: everything else.
  features |= expandIn(ast, null, ds);
  return features;
}

function utilityRoot(params: string): string {
  const name = params.trim();
  return name.endsWith("-*") ? name.slice(0, -2) : name;
}

function applyCandidates(node: AtRule): string[] {
  return node.params.split(/\s+/).filter((c) => c !== "");
}

function expandIn(
  nodes: AstNode[],
  root: ParentNode | null,
  ds: DesignSystem,
): number {
  let features = Features.NONE;
  walk(
    nodes,
    (node, { parent, path, replaceWith }) => {
      if (
        node.kind === "at-rule" && node.name === "@utility"
      ) return WalkAction.Skip;
      if (node.kind !== "at-rule" || node.name !== "@apply") return;

      const effectiveParent = parent ?? root;
      // A top-level `@apply` is left untouched.
      if (effectiveParent === null) return;
      if (
        effectiveParent.kind === "context" &&
        path.every((p) => p.kind === "context")
      ) return;
      if (node.nodes.length > 0) {
        throw new TwillError("`@apply` cannot have a body.");
      }
      for (const ancestor of path) {
        if (ancestor.kind === "at-rule" && ancestor.name === "@keyframes") {
          throw new TwillError("You cannot use `@apply` inside `@keyframes`.");
        }
      }

      const candidates = applyCandidates(node);
      const mixins = candidates.filter((c) => c.startsWith("--"));
      if (mixins.length === candidates.length && candidates.length > 0) return;
      if (mixins.length > 0) {
        throw new TwillError(
          `You cannot mix CSS mixins with utility classes in \`@apply ${node.params}\`.`,
        );
      }

      const compiled = compileCandidates(candidates, ds, {
        respectImportant: false,
        onInvalidCandidate: (candidate) => {
          throw new TwillError(applyErrorMessage(candidate, ds));
        },
      });

      const replacement: AstNode[] = [];
      for (const rule of compiled.nodes) replacement.push(...rule.nodes);
      features |= Features.AT_APPLY;
      replaceWith(replacement);
    },
    root,
    {},
    root === null ? [] : [root],
  );
  return features;
}

/** Builds the error message for an unknown `@apply` candidate. */
export function applyErrorMessage(candidate: string, ds: DesignSystem): string {
  const prefix = ds.theme.prefix;
  if (prefix !== null && !candidate.startsWith(`${prefix}:`)) {
    return `Cannot apply unknown utility class \`${candidate}\`. Did you forget the \`${prefix}:\` prefix?`;
  }
  if (ds.invalidCandidates.has(candidate)) {
    return `Cannot apply utility class \`${candidate}\` because it is explicitly disabled by \`@source not inline(...)\`.`;
  }
  const parts = segment(candidate, ":");
  const variants = parts.slice(0, -1);
  if (prefix !== null) variants.shift();
  for (const variant of variants) {
    if (ds.parseVariant(variant) === null) {
      return `Cannot apply unknown variant \`${variant}\` in \`${candidate}\`.`;
    }
  }
  if (ds.theme.size === 0) {
    return `Cannot apply unknown utility class \`${candidate}\`. The theme is empty; are you missing \`@import "twill";\` or \`@reference "twill";\`?`;
  }
  return `Cannot apply unknown utility class \`${candidate}\`.`;
}

/**
 * Sorts nodes so that every node comes after its dependencies. Calls
 * `onCycle` with the offending node when a cycle exists.
 */
export function topologicalSort(
  dependencies: Map<string, Set<string>>,
  onCycle: (node: string) => never,
): string[] {
  const result: string[] = [];
  const state = new Map<string, "visiting" | "done">();
  const visit = (name: string) => {
    const current = state.get(name);
    if (current === "done") return;
    if (current === "visiting") onCycle(name);
    state.set(name, "visiting");
    for (const dep of dependencies.get(name) ?? []) {
      if (dependencies.has(dep)) visit(dep);
    }
    state.set(name, "done");
    result.push(name);
  };
  for (const name of dependencies.keys()) visit(name);
  return result;
}
