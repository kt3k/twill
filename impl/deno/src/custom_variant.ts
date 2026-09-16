/**
 * `@custom-variant` definitions (SPEC §6.7).
 *
 * @module
 */

import {
  type AstNode,
  atRoot,
  type AtRule,
  cloneNodes,
  rule,
  styleRule,
  walk,
  WalkAction,
} from "./ast.ts";
import type { DesignSystem } from "./design_system.ts";
import { TwillError } from "./error.ts";
import { isValidVariantName, segment } from "./utils.ts";
import { compoundsForSelectors } from "./variants.ts";

export interface CustomVariant {
  name: string;
  /** Names of other custom variants referenced through nested `@variant`. */
  dependencies: Set<string>;
  register: (ds: DesignSystem) => void;
}

/**
 * Whether a top-level `@variant` node is a compatibility form of
 * `@custom-variant`: a selector form without a body, or a body containing
 * `@slot`.
 */
export function isCustomVariantCompat(node: AtRule): boolean {
  if (node.nodes.length === 0) return node.params.includes("(");
  let hasSlot = false;
  walk(node.nodes, (child) => {
    if (child.kind === "at-rule" && child.name === "@slot") {
      hasSlot = true;
      return WalkAction.Stop;
    }
  });
  return hasSlot;
}

/** Parses an `@custom-variant` node into a registration. */
export function parseCustomVariant(node: AtRule): CustomVariant {
  const params = node.params.trim();
  const space = params.search(/\s/);
  const name = space === -1 ? params : params.slice(0, space);
  const rest = space === -1 ? "" : params.slice(space).trim();

  if (!isValidVariantName(name)) {
    throw new TwillError(
      `\`@custom-variant ${name}\` defines an invalid variant name. Variants should only contain alphanumeric, dashes or underscore characters.`,
    );
  }
  if (rest !== "" && node.nodes.length > 0) {
    throw new TwillError(
      `\`@custom-variant ${name}\` cannot have both a selector and a body.`,
    );
  }
  if (rest === "" && node.nodes.length === 0) {
    throw new TwillError(
      `\`@custom-variant ${name}\` has no selector or body.`,
    );
  }

  if (rest !== "") {
    if (!rest.startsWith("(") || !rest.endsWith(")")) {
      throw new TwillError(
        `\`@custom-variant ${name} ${rest}\` has an invalid selector.`,
      );
    }
    const selectors = segment(rest.slice(1, -1), ",").map((s) => s.trim());
    if (selectors.some((s) => s === "")) {
      throw new TwillError(
        `\`@custom-variant ${name} ${rest}\` has an empty selector.`,
      );
    }
    const atRuleSelectors = selectors.filter((s) => s.startsWith("@"));
    const styleSelectors = selectors.filter((s) => !s.startsWith("@"));
    const compounds = compoundsForSelectors(selectors);
    return {
      name,
      dependencies: new Set(),
      register(ds) {
        ds.variants.static(name, (target) => {
          const nodes: AstNode[] = [];
          let first = true;
          const children = () => {
            const result = first ? target.nodes : cloneNodes(target.nodes);
            first = false;
            return result;
          };
          if (styleSelectors.length > 0) {
            nodes.push(styleRule(styleSelectors.join(", "), children()));
          }
          for (const selector of atRuleSelectors) {
            nodes.push(rule(selector, children()));
          }
          target.nodes = nodes;
        }, { compounds });
      },
    };
  }

  const body = cloneNodes(node.nodes);
  const dependencies = new Set<string>();
  const selectors: string[] = [];
  walk(body, (child) => {
    if (child.kind === "rule") {
      selectors.push(child.selector);
    } else if (child.kind === "at-rule") {
      if (child.name === "@variant") {
        for (const group of segment(child.params, ",")) {
          for (const variantName of segment(group, ":")) {
            const trimmed = variantName.trim();
            if (trimmed !== "") dependencies.add(trimmed);
          }
        }
      } else if (child.name !== "@slot") {
        selectors.push(`${child.name} ${child.params}`.trim());
      }
    }
  });
  const compounds = compoundsForSelectors(selectors);

  return {
    name,
    dependencies,
    register(ds) {
      ds.variants.static(name, (target) => {
        const clone = cloneNodes(body);
        let first = true;
        walk(clone, (child, { replaceWith, parent }) => {
          if (child.kind !== "at-rule") return;
          if (parent?.kind === "at-root") return;
          if (child.name === "@slot") {
            replaceWith(first ? target.nodes : cloneNodes(target.nodes));
            first = false;
            return WalkAction.Skip;
          }
          if (child.name === "@keyframes" || child.name === "@property") {
            replaceWith(atRoot([child]));
            return WalkAction.Skip;
          }
        });
        target.nodes = clone;
      }, { compounds });
    },
  };
}
