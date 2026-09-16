/**
 * Nested `@variant` expansion (SPEC §6.9).
 *
 * @module
 */

import { type AstNode, cloneNodes, styleRule, walk } from "./ast.ts";
import type { DesignSystem } from "./design_system.ts";
import { TwillError } from "./error.ts";
import { Features } from "./features.ts";
import { segment } from "./utils.ts";
import { applyVariant } from "./variants.ts";

/**
 * Expands every nested `@variant` in place. Comma-separated groups are
 * independent alternatives; colon-separated names within a group stack and
 * are applied from right to left. Returns the feature flags.
 */
export function substituteAtVariant(ast: AstNode[], ds: DesignSystem): number {
  let features = Features.NONE;
  walk(ast, (node, { replaceWith }) => {
    if (node.kind !== "at-rule" || node.name !== "@variant") return;
    features |= Features.VARIANTS;

    const groups = segment(node.params, ",").map((g) => g.trim());
    const result: AstNode[] = [];
    for (let i = 0; i < groups.length; i++) {
      const names = segment(groups[i], ":").map((n) => n.trim());
      const children = i === groups.length - 1
        ? node.nodes
        : cloneNodes(node.nodes);
      const rule = styleRule("&", children);
      for (let j = names.length - 1; j >= 0; j--) {
        const name = names[j];
        if (name === "") {
          throw new TwillError(
            `Cannot use \`@variant\` with an empty variant name in \`@variant ${node.params}\`.`,
          );
        }
        const variant = ds.parseVariant(name);
        if (variant === null) {
          throw new TwillError(
            `Cannot use \`@variant\` with unknown variant: ${name}`,
          );
        }
        if (applyVariant(rule, variant, ds.variants) === null) {
          throw new TwillError(`Cannot use \`@variant\` with variant: ${name}`);
        }
      }
      if (rule.selector === "&") result.push(...rule.nodes);
      else result.push(rule);
    }
    replaceWith(result);
  });
  return features;
}
