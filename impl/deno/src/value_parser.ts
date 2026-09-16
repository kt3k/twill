/**
 * A small parser for CSS declaration values: words, separators, and function
 * calls. Used by arbitrary value decoding (SPEC §4.2), theme functions
 * (§6.11), and `--value(...)` resolution (§10.6).
 *
 * @module
 */

export interface ValueWord {
  kind: "word";
  value: string;
}

export interface ValueSeparator {
  kind: "separator";
  value: string;
}

export interface ValueFunction {
  kind: "function";
  /** The function name without the parenthesis; empty for a bare `( ... )`. */
  value: string;
  nodes: ValueAstNode[];
}

export type ValueAstNode = ValueWord | ValueSeparator | ValueFunction;

export function word(value: string): ValueWord {
  return { kind: "word", value };
}

export function separator(value: string): ValueSeparator {
  return { kind: "separator", value };
}

export function fn(value: string, nodes: ValueAstNode[] = []): ValueFunction {
  return { kind: "function", value, nodes };
}

const BACKSLASH = 0x5c;
const DOUBLE_QUOTE = 0x22;
const SINGLE_QUOTE = 0x27;
const OPEN_PAREN = 0x28;
const CLOSE_PAREN = 0x29;
const COMMA = 0x2c;
const SLASH = 0x2f;
const SPACE = 0x20;
const TAB = 0x09;
const LINE_BREAK = 0x0a;

function isWhitespace(c: number): boolean {
  return c === SPACE || c === TAB || c === LINE_BREAK || c === 0x0d ||
    c === 0x0c;
}

/** Parses a value into words, separators, and function calls. */
export function parseValue(input: string): ValueAstNode[] {
  const ast: ValueAstNode[] = [];
  const stack: (ValueFunction | null)[] = [];
  let parent: ValueFunction | null = null;
  let buffer = "";

  const push = (node: ValueAstNode) => {
    if (parent) parent.nodes.push(node);
    else ast.push(node);
  };
  const flush = () => {
    if (buffer !== "") {
      push(word(buffer));
      buffer = "";
    }
  };

  for (let i = 0; i < input.length; i++) {
    const c = input.charCodeAt(i);
    if (c === BACKSLASH) {
      buffer += input.slice(i, i + 2);
      i++;
      continue;
    }
    if (c === DOUBLE_QUOTE || c === SINGLE_QUOTE) {
      const start = i;
      for (i++; i < input.length; i++) {
        const d = input.charCodeAt(i);
        if (d === BACKSLASH) {
          i++;
        } else if (d === c) {
          break;
        }
      }
      buffer += input.slice(start, i + 1);
      continue;
    }
    if (c === OPEN_PAREN) {
      const node = fn(buffer, []);
      buffer = "";
      push(node);
      stack.push(parent);
      parent = node;
      continue;
    }
    if (c === CLOSE_PAREN) {
      flush();
      if (parent !== null) parent = stack.pop() ?? null;
      else buffer += ")";
      continue;
    }
    if (c === COMMA || c === SLASH) {
      flush();
      push(separator(input[i]));
      continue;
    }
    if (isWhitespace(c)) {
      flush();
      let j = i;
      while (j < input.length && isWhitespace(input.charCodeAt(j))) j++;
      push(separator(input.slice(i, j)));
      i = j - 1;
      continue;
    }
    buffer += input[i];
  }
  flush();
  return ast;
}

/** Prints value nodes back to CSS text. */
export function toCss(nodes: ValueAstNode[]): string {
  let out = "";
  for (const node of nodes) {
    switch (node.kind) {
      case "word":
      case "separator":
        out += node.value;
        break;
      case "function":
        out += `${node.value}(${toCss(node.nodes)})`;
        break;
    }
  }
  return out;
}

/**
 * Walks value nodes depth first. The visitor may replace the current node
 * with one or more nodes; replacement nodes are not revisited.
 */
export function walkValue(
  nodes: ValueAstNode[],
  visit: (
    node: ValueAstNode,
    utils: {
      parent: ValueFunction | null;
      replaceWith(nodes: ValueAstNode | ValueAstNode[]): void;
    },
  ) => boolean | void,
  parent: ValueFunction | null = null,
): void {
  for (let i = 0; i < nodes.length; i++) {
    const node = nodes[i];
    let replaced = false;
    const skip = visit(node, {
      parent,
      replaceWith(replacement) {
        const list = Array.isArray(replacement) ? replacement : [replacement];
        nodes.splice(i, 1, ...list);
        i += list.length - 1;
        replaced = true;
      },
    });
    if (replaced || skip === true) continue;
    if (node.kind === "function") walkValue(node.nodes, visit, node);
  }
}
