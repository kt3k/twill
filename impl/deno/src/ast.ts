/**
 * The AST the compiler operates on (SPEC §4.1.1).
 *
 * @module
 */

export interface Rule {
  kind: "rule";
  selector: string;
  nodes: AstNode[];
}

export interface AtRule {
  kind: "at-rule";
  /** Includes the leading `@`, for example `@media`. */
  name: string;
  params: string;
  /** Empty for statement at-rules such as `@import`. */
  nodes: AstNode[];
}

export interface Declaration {
  kind: "declaration";
  property: string;
  /** Declarations whose value is `undefined` are never printed. */
  value: string | undefined;
  important: boolean;
}

export interface Comment {
  kind: "comment";
  /** The text between `/*` and `*\/`. */
  value: string;
}

export type ContextValue = string | boolean;

/**
 * Never printed. Carries metadata such as `base`, `reference`, `theme`,
 * `source`, and `sourceBase` to its subtree.
 */
export interface Context {
  kind: "context";
  context: Record<string, ContextValue>;
  nodes: AstNode[];
}

/**
 * Never printed in place. Its children are hoisted to the end of the document
 * during optimization (SPEC §12).
 */
export interface AtRoot {
  kind: "at-root";
  nodes: AstNode[];
}

export type AstNode = Rule | AtRule | Declaration | Comment | Context | AtRoot;

/** A node that can contain children. */
export type ParentNode = Rule | AtRule | Context | AtRoot;

export function styleRule(selector: string, nodes: AstNode[] = []): Rule {
  return { kind: "rule", selector, nodes };
}

export function atRule(
  name: string,
  params = "",
  nodes: AstNode[] = [],
): AtRule {
  return { kind: "at-rule", name, params, nodes };
}

/**
 * Creates an `at-rule` when `selector` starts with `@` (splitting on the first
 * whitespace into `name` and `params`) and a `rule` otherwise.
 */
export function rule(selector: string, nodes: AstNode[] = []): Rule | AtRule {
  if (selector.charCodeAt(0) === 64 /* @ */) {
    return parseAtRule(selector, nodes);
  }
  return styleRule(selector, nodes);
}

/** Splits an at-rule prelude such as `@media (width >= 1px)` into a node. */
export function parseAtRule(prelude: string, nodes: AstNode[] = []): AtRule {
  const text = prelude.trim();
  for (let i = 0; i < text.length; i++) {
    const c = text.charCodeAt(i);
    // whitespace or `(`
    if (c === 32 || c === 9 || c === 10 || c === 12 || c === 13 || c === 40) {
      return atRule(text.slice(0, i), text.slice(i).trim(), nodes);
    }
  }
  return atRule(text, "", nodes);
}

export function decl(
  property: string,
  value: string | undefined,
  important = false,
): Declaration {
  return { kind: "declaration", property, value, important };
}

export function comment(value: string): Comment {
  return { kind: "comment", value };
}

export function context(
  context: Record<string, ContextValue>,
  nodes: AstNode[],
): Context {
  return { kind: "context", context, nodes };
}

export function atRoot(nodes: AstNode[]): AtRoot {
  return { kind: "at-root", nodes };
}

export function isParent(node: AstNode): node is ParentNode {
  return node.kind === "rule" || node.kind === "at-rule" ||
    node.kind === "context" ||
    node.kind === "at-root";
}

/** Deep-clones a list of nodes. */
export function cloneNodes<T extends AstNode>(nodes: T[]): T[] {
  return nodes.map(cloneNode);
}

export function cloneNode<T extends AstNode>(node: T): T {
  switch (node.kind) {
    case "rule":
      return {
        kind: "rule",
        selector: node.selector,
        nodes: cloneNodes(node.nodes),
      } as T;
    case "at-rule":
      return {
        kind: "at-rule",
        name: node.name,
        params: node.params,
        nodes: cloneNodes(node.nodes),
      } as T;
    case "declaration":
      return { ...node } as T;
    case "comment":
      return { ...node } as T;
    case "context":
      return {
        kind: "context",
        context: { ...node.context },
        nodes: cloneNodes(node.nodes),
      } as T;
    case "at-root":
      return { kind: "at-root", nodes: cloneNodes(node.nodes) } as T;
  }
}

export const WalkAction = {
  /** Continue walking, descending into children. */
  Continue: 0,
  /** Do not descend into this node's children. */
  Skip: 1,
  /** Stop the whole walk. */
  Stop: 2,
} as const;

export type WalkAction = typeof WalkAction[keyof typeof WalkAction];

export interface WalkUtils {
  /** The parent node, or `null` at the top level. */
  parent: ParentNode | null;
  /** The merged `context` values of every enclosing `context` node. */
  context: Record<string, ContextValue>;
  /** The path of ancestors, outermost first. */
  path: ParentNode[];
  /** Replaces the current node with the given nodes (which are then walked). */
  replaceWith(nodes: AstNode | AstNode[]): void;
}

export type WalkVisitor = (
  node: AstNode,
  utils: WalkUtils,
) => WalkAction | undefined | void;

/**
 * Walks the tree depth first, visiting parents before children. The visitor
 * may replace the current node; replacement nodes are visited in turn.
 */
export function walk(
  nodes: AstNode[],
  visit: WalkVisitor,
  parent: ParentNode | null = null,
  ctx: Record<string, ContextValue> = {},
  path: ParentNode[] = [],
): WalkAction | undefined {
  for (let i = 0; i < nodes.length; i++) {
    const node = nodes[i];
    let replaced = false;
    const nodeCtx = node.kind === "context" ? { ...ctx, ...node.context } : ctx;
    const utils: WalkUtils = {
      parent,
      context: node.kind === "context" ? nodeCtx : ctx,
      path,
      replaceWith(replacement) {
        const list = Array.isArray(replacement) ? replacement : [replacement];
        nodes.splice(i, 1, ...list);
        replaced = true;
        // Re-visit from the same index so the replacement is walked.
        i--;
      },
    };
    const action = visit(node, utils) ?? WalkAction.Continue;
    if (action === WalkAction.Stop) return WalkAction.Stop;
    if (replaced) continue;
    if (action === WalkAction.Skip) continue;
    if (isParent(node)) {
      path.push(node);
      const result = walk(node.nodes, visit, node, nodeCtx, path);
      path.pop();
      if (result === WalkAction.Stop) return WalkAction.Stop;
    }
  }
}

/**
 * Walks the tree depth first, visiting children before their parents.
 */
export function walkDepth(
  nodes: AstNode[],
  visit: (
    node: AstNode,
    utils: Omit<WalkUtils, "replaceWith"> & {
      replaceWith(nodes: AstNode | AstNode[]): void;
    },
  ) => void,
  parent: ParentNode | null = null,
  ctx: Record<string, ContextValue> = {},
  path: ParentNode[] = [],
): void {
  for (let i = 0; i < nodes.length; i++) {
    const node = nodes[i];
    const nodeCtx = node.kind === "context" ? { ...ctx, ...node.context } : ctx;
    if (isParent(node)) {
      path.push(node);
      walkDepth(node.nodes, visit, node, nodeCtx, path);
      path.pop();
    }
    visit(node, {
      parent,
      context: nodeCtx,
      path,
      replaceWith(replacement) {
        const list = Array.isArray(replacement) ? replacement : [replacement];
        nodes.splice(i, 1, ...list);
        i += list.length - 1;
      },
    });
  }
}
