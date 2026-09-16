/**
 * CSS serializer (SPEC §5.2).
 *
 * @module
 */

import type { AstNode } from "./ast.ts";

/**
 * Prints nodes recursively with two-space indentation per depth and `\n`
 * after every line. `context` children are printed at the same depth;
 * `at-root` nodes are never printed in place (the optimizer hoists them), so
 * they are printed like `context` here for debuggability.
 */
export function serialize(nodes: AstNode[], depth = 0): string {
  let out = "";
  for (const node of nodes) out += serializeNode(node, depth);
  return out;
}

function serializeNode(node: AstNode, depth: number): string {
  const indent = "  ".repeat(depth);
  switch (node.kind) {
    case "declaration": {
      if (node.value === undefined) return "";
      return `${indent}${node.property}: ${node.value}${
        node.important ? " !important" : ""
      };\n`;
    }
    case "rule": {
      return `${indent}${node.selector} {\n${
        serialize(node.nodes, depth + 1)
      }${indent}}\n`;
    }
    case "at-rule": {
      const head = node.params === ""
        ? node.name
        : `${node.name} ${node.params}`;
      if (node.nodes.length === 0) return `${indent}${head};\n`;
      return `${indent}${head} {\n${
        serialize(node.nodes, depth + 1)
      }${indent}}\n`;
    }
    case "comment": {
      return `${indent}/*${node.value}*/\n`;
    }
    case "context":
    case "at-root": {
      return serialize(node.nodes, depth);
    }
  }
}

/**
 * Prints nodes without indentation or line breaks. Used for `--minify`; the
 * output is semantically identical to `serialize`.
 */
export function serializeCompact(nodes: AstNode[]): string {
  let out = "";
  for (const node of nodes) out += compactNode(node);
  return out;
}

function compactNode(node: AstNode): string {
  switch (node.kind) {
    case "declaration": {
      if (node.value === undefined) return "";
      return `${node.property}:${node.value}${
        node.important ? "!important" : ""
      };`;
    }
    case "rule": {
      return `${node.selector}{${serializeCompact(node.nodes)}}`;
    }
    case "at-rule": {
      const head = node.params === ""
        ? node.name
        : `${node.name} ${node.params}`;
      if (node.nodes.length === 0) return `${head};`;
      return `${head}{${serializeCompact(node.nodes)}}`;
    }
    case "comment": {
      return `/*${node.value}*/`;
    }
    case "context":
    case "at-root": {
      return serializeCompact(node.nodes);
    }
  }
}
