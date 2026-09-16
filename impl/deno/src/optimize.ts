/**
 * Output optimization: property registration deduplication, root hoisting,
 * unused theme value removal, and nesting flattening (SPEC §12).
 *
 * @module
 */

import {
  type AstNode,
  type AtRule,
  atRule,
  type ContextValue,
  type Declaration,
  type Rule,
  styleRule,
} from "./ast.ts";
import type { DesignSystem } from "./design_system.ts";
import { ThemeOptions } from "./theme.ts";
import { segment, unescape } from "./utils.ts";

const KEEP_EMPTY_AT_RULES = new Set([
  "@layer",
  "@charset",
  "@custom-media",
  "@namespace",
  "@import",
  "@apply",
]);

/** At-rules whose child rules are not nested style rules. */
const OPAQUE_AT_RULES = new Set([
  "@keyframes",
  "@property",
  "@font-face",
  "@counter-style",
  "@page",
]);

const VAR_PATTERN = /var\(\s*(--[^\s,)]+)/g;

interface Tracked {
  parent: AstNode[];
  node: Declaration;
  key: string;
}

/**
 * Optimizes the whole document before serialization. The input is not
 * mutated; a new tree is returned.
 */
export function optimizeAst(ast: AstNode[], ds: DesignSystem): AstNode[] {
  const themeDeclarations: Tracked[] = [];
  const themeKeyframes: { parent: AstNode[]; node: AtRule }[] = [];
  const usedVariables = new Set<string>();
  const dependencies = new Map<string, Set<string>>();
  const usedKeyframes = new Set<string>();
  const seenProperties = new Set<string>();
  const roots: AstNode[] = [];

  const recordVariables = (value: string, ownerKey: string | null) => {
    if (!value.includes("var(")) return;
    for (const match of value.matchAll(VAR_PATTERN)) {
      const name = match[1];
      if (ownerKey === null) {
        usedVariables.add(name);
      } else {
        let deps = dependencies.get(ownerKey);
        if (deps === undefined) {
          deps = new Set();
          dependencies.set(ownerKey, deps);
        }
        deps.add(name);
      }
    }
  };

  const transform = (
    nodes: AstNode[],
    out: AstNode[],
    context: Record<string, ContextValue>,
    depth: number,
  ) => {
    for (const node of nodes) {
      switch (node.kind) {
        case "declaration": {
          if (node.value === undefined || node.property === "--tw-sort") {
            continue;
          }
          const copy: Declaration = { ...node };
          if (context.theme === true && node.property.startsWith("--")) {
            if (node.value === "initial") continue;
            const key = ds.theme.unprefixKey(unescape(node.property));
            recordVariables(node.value, key);
            themeDeclarations.push({ parent: out, node: copy, key });
          } else {
            recordVariables(node.value, null);
          }
          if (node.property === "animation") {
            for (const token of node.value.split(/[\s,]+/)) {
              if (token !== "") usedKeyframes.add(token);
            }
          }
          out.push(copy);
          break;
        }
        case "rule": {
          const children: AstNode[] = [];
          transform(node.nodes, children, context, depth + 1);
          if (children.length === 0) continue;
          out.push(styleRule(node.selector, children));
          break;
        }
        case "at-rule": {
          if (node.name === "@property" && depth === 0) {
            if (seenProperties.has(node.params)) continue;
            seenProperties.add(node.params);
          }
          const children: AstNode[] = [];
          transform(node.nodes, children, context, depth + 1);
          if (children.length === 0 && !KEEP_EMPTY_AT_RULES.has(node.name)) {
            continue;
          }
          const copy = atRule(node.name, node.params, children);
          if (node.name === "@keyframes" && context.theme === true) {
            themeKeyframes.push({ parent: out, node: copy });
          }
          out.push(copy);
          break;
        }
        case "at-root": {
          transform(node.nodes, roots, context, 0);
          break;
        }
        case "context": {
          if (node.context.reference === true) continue;
          transform(node.nodes, out, { ...context, ...node.context }, depth);
          break;
        }
        case "comment": {
          out.push({ ...node });
          break;
        }
      }
    }
  };

  const result: AstNode[] = [];
  transform(ast, result, {}, 0);

  // Prune unused theme variables (SPEC §7.6).
  for (const name of usedVariables) ds.theme.markUsedVariable(name);
  const keep = new Set<string>();
  for (const { key } of themeDeclarations) {
    const options = ds.theme.getOptions(key);
    if (options & (ThemeOptions.STATIC | ThemeOptions.USED)) keep.add(key);
  }
  let changed = true;
  while (changed) {
    changed = false;
    for (const key of [...keep]) {
      for (const dep of dependencies.get(key) ?? []) {
        const depKey = ds.theme.unprefixKey(unescape(dep));
        if (!keep.has(depKey) && ds.theme.has(depKey)) {
          keep.add(depKey);
          changed = true;
        }
      }
    }
  }
  const removed = new Set<AstNode>();
  for (const tracked of themeDeclarations) {
    if (!keep.has(tracked.key)) removed.add(tracked.node);
  }

  // Keyframes referenced by used `--animate-*` entries.
  for (const key of keep) {
    if (!key.startsWith("--animate")) continue;
    const value = ds.theme.get([key]);
    if (value === null) continue;
    for (const token of value.split(/[\s,]+/)) {
      if (token !== "") usedKeyframes.add(token);
    }
  }
  for (const { node } of themeKeyframes) {
    if (!usedKeyframes.has(node.params.trim())) removed.add(node);
  }

  const pruned = removed.size > 0 ? removeNodes(result, removed) : result;
  const prunedRoots = removed.size > 0 ? removeNodes(roots, removed) : roots;

  return flatten([...pruned, ...prunedRoots]);
}

/** Removes the given nodes and any rules or at-rules that become empty. */
function removeNodes(nodes: AstNode[], removed: Set<AstNode>): AstNode[] {
  const out: AstNode[] = [];
  for (const node of nodes) {
    if (removed.has(node)) continue;
    if (node.kind === "rule") {
      const children = removeNodes(node.nodes, removed);
      if (children.length === 0) continue;
      out.push(styleRule(node.selector, children));
    } else if (node.kind === "at-rule") {
      const children = removeNodes(node.nodes, removed);
      if (
        children.length === 0 && node.nodes.length > 0 &&
        !KEEP_EMPTY_AT_RULES.has(node.name)
      ) {
        continue;
      }
      if (
        children.length === 0 && node.name === "@layer" && node.nodes.length > 0
      ) continue;
      out.push(atRule(node.name, node.params, children));
    } else {
      out.push(node);
    }
  }
  return out;
}

/** Combines a parent selector with a nested selector (SPEC §12.3). */
export function combineSelectors(parent: string, child: string): string {
  const wrapped = segment(parent, ",").length > 1 ? `:is(${parent})` : parent;
  if (child.includes("&")) return child.replaceAll("&", wrapped);
  const first = child[0];
  if (first === ">" || first === "+" || first === "~") {
    return `${wrapped} ${child}`;
  }
  return `${wrapped}${child}`;
}

/** Flattens nested rules and at-rules into flat CSS (SPEC §12.3). */
export function flatten(nodes: AstNode[]): AstNode[] {
  const out: AstNode[] = [];
  for (const node of nodes) {
    if (node.kind === "rule") {
      out.push(...flattenRule(node, null));
    } else if (node.kind === "at-rule") {
      if (OPAQUE_AT_RULES.has(node.name)) out.push(node);
      else out.push(atRule(node.name, node.params, flatten(node.nodes)));
    } else if (node.kind === "context" || node.kind === "at-root") {
      out.push(...flatten(node.nodes));
    } else {
      out.push(node);
    }
  }
  return out;
}

function flattenRule(rule: Rule, parentSelector: string | null): AstNode[] {
  let selector: string;
  if (parentSelector === null) selector = rule.selector;
  else if (rule.selector === "&") selector = parentSelector;
  else selector = combineSelectors(parentSelector, rule.selector);
  return flattenChildren(rule.nodes, selector);
}

function flattenChildren(nodes: AstNode[], selector: string): AstNode[] {
  const declarations: AstNode[] = [];
  const hoisted: AstNode[] = [];
  for (const child of nodes) {
    switch (child.kind) {
      case "declaration":
      case "comment":
        declarations.push(child);
        break;
      case "rule":
        hoisted.push(...flattenRule(child, selector));
        break;
      case "at-rule":
        // Statement at-rules such as `@apply --mixin;` stay in place.
        if (child.nodes.length === 0) declarations.push(child);
        else hoisted.push(...flattenAtRule(child, selector));
        break;
      case "context":
      case "at-root":
        hoisted.push(...flattenChildren(child.nodes, selector));
        break;
    }
  }
  const out: AstNode[] = [];
  if (declarations.length > 0) out.push(styleRule(selector, declarations));
  out.push(...hoisted);
  return out;
}

function flattenAtRule(node: AtRule, selector: string): AstNode[] {
  if (OPAQUE_AT_RULES.has(node.name)) return [node];
  const children = flattenChildren(node.nodes, selector);
  if (children.length === 0) return [];
  return [atRule(node.name, node.params, children)];
}
