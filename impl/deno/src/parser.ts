/**
 * CSS parser (SPEC §5.1).
 *
 * Produces the AST of `./ast.ts`. Ordinary comments are dropped, `/*!`
 * comments are kept, `!important` is split from declaration values, a missing
 * `;` before `}` is accepted, and `;`, `{`, `}` inside quotes or parentheses
 * are not treated as delimiters.
 *
 * @module
 */

import {
  type AstNode,
  type AtRule,
  comment,
  decl,
  type Declaration,
  parseAtRule,
  type Rule,
  styleRule,
} from "./ast.ts";
import { positionAt, TwillError } from "./error.ts";

const BACKSLASH = 0x5c;
const SLASH = 0x2f;
const ASTERISK = 0x2a;
const DOUBLE_QUOTE = 0x22;
const SINGLE_QUOTE = 0x27;
const SEMICOLON = 0x3b;
const LINE_BREAK = 0x0a;
const SPACE = 0x20;
const TAB = 0x09;
const CARRIAGE_RETURN = 0x0d;
const FORM_FEED = 0x0c;
const OPEN_CURLY = 0x7b;
const CLOSE_CURLY = 0x7d;
const OPEN_PAREN = 0x28;
const CLOSE_PAREN = 0x29;
const OPEN_BRACKET = 0x5b;
const CLOSE_BRACKET = 0x5d;
const DASH = 0x2d;
const AT_SIGN = 0x40;
const EXCLAMATION = 0x21;

function isWhitespace(c: number): boolean {
  return c === SPACE || c === TAB || c === LINE_BREAK ||
    c === CARRIAGE_RETURN || c === FORM_FEED;
}

/** Parses CSS text into a list of nodes. */
export function parse(input: string): AstNode[] {
  if (input.includes("\r\n")) input = input.replaceAll("\r\n", "\n");

  const ast: AstNode[] = [];
  const stack: (Rule | AtRule | null)[] = [];
  let parent: Rule | AtRule | null = null;

  // The text of the prelude or declaration currently being read.
  let buffer = "";
  // Index in `input` where `buffer` started, for error positions.
  let bufferStart = 0;
  // Stack of expected closing brackets for `(` and `[` inside `buffer`.
  // While non-empty, `;`, `{`, and `}` are literal characters.
  let brackets = "";
  // Depth of `{` inside a custom property value.
  let customPropertyBraces = 0;

  const error = (message: string, index: number): never => {
    throw new TwillError(message, positionAt(input, index));
  };

  const push = (node: AstNode) => {
    if (parent) parent.nodes.push(node);
    else ast.push(node);
  };

  const finishBuffer = (end: number) => {
    const text = buffer.trim();
    buffer = "";
    brackets = "";
    customPropertyBraces = 0;
    if (text === "") return;
    if (text.charCodeAt(0) === AT_SIGN) {
      push(parseAtRule(text));
    } else {
      const node = parseDeclaration(text);
      if (node === null) return error("Invalid declaration", end);
      push(node);
    }
  };

  for (let i = 0; i < input.length; i++) {
    const c = input.charCodeAt(i);

    // Escaped characters are copied verbatim, including the escaped character.
    if (c === BACKSLASH) {
      if (buffer === "") bufferStart = i;
      buffer += input.slice(i, i + 2);
      i++;
      continue;
    }

    // Comments: `/* ... */`. Only `/*!` comments are kept.
    if (c === SLASH && input.charCodeAt(i + 1) === ASTERISK) {
      const start = i;
      const end = input.indexOf("*/", i + 2);
      if (end === -1) error("Unterminated comment", start);
      const text = input.slice(start + 2, end);
      if (text.charCodeAt(0) === EXCLAMATION) {
        // A license comment inside a declaration is dropped; at statement
        // level it is kept.
        if (buffer.trim() === "") push(comment(text));
      }
      i = end + 1;
      continue;
    }

    // Strings are copied verbatim.
    if (c === DOUBLE_QUOTE || c === SINGLE_QUOTE) {
      const start = i;
      if (buffer === "") bufferStart = i;
      let j = i + 1;
      for (; j < input.length; j++) {
        const d = input.charCodeAt(j);
        if (d === BACKSLASH) {
          j++;
        } else if (d === c) {
          break;
        } else if (d === LINE_BREAK) {
          error("Unterminated string", start);
        }
      }
      if (j >= input.length) error("Unterminated string", start);
      buffer += input.slice(start, j + 1);
      i = j;
      continue;
    }

    // Skip leading whitespace of a prelude or declaration.
    if (buffer === "" && isWhitespace(c)) continue;

    if (buffer === "") bufferStart = i;

    const inCustomProperty = buffer.charCodeAt(0) === DASH &&
      buffer.charCodeAt(1) === DASH;

    if (c === OPEN_PAREN || c === OPEN_BRACKET) {
      brackets += c === OPEN_PAREN ? ")" : "]";
      buffer += input[i];
      continue;
    }

    if (c === CLOSE_PAREN || c === CLOSE_BRACKET) {
      if (brackets === "" || brackets.charCodeAt(brackets.length - 1) !== c) {
        error(`Unexpected \`${input[i]}\``, i);
      }
      brackets = brackets.slice(0, -1);
      buffer += input[i];
      continue;
    }

    if (brackets !== "") {
      buffer += input[i];
      continue;
    }

    if (c === SEMICOLON) {
      if (customPropertyBraces > 0) {
        buffer += input[i];
        continue;
      }
      finishBuffer(i);
      continue;
    }

    if (c === OPEN_CURLY) {
      // `{` inside a custom property value is part of the value.
      if (inCustomProperty && buffer.includes(":")) {
        customPropertyBraces++;
        buffer += input[i];
        continue;
      }
      const prelude = buffer.trim();
      if (prelude === "") error("Missing selector before `{`", i);
      const node: Rule | AtRule = prelude.charCodeAt(0) === AT_SIGN
        ? parseAtRule(prelude)
        : styleRule(prelude, []);
      push(node);
      stack.push(parent);
      parent = node;
      buffer = "";
      continue;
    }

    if (c === CLOSE_CURLY) {
      if (customPropertyBraces > 0) {
        customPropertyBraces--;
        buffer += input[i];
        continue;
      }
      if (parent === null) error("Unexpected `}`", i);
      // A declaration or statement at-rule without a trailing `;`.
      finishBuffer(i);
      parent = stack.pop() ?? null;
      continue;
    }

    buffer += input[i];
  }

  if (parent !== null) {
    error("Missing closing `}`", input.length);
  }
  if (brackets !== "") {
    error(`Missing closing \`${brackets[brackets.length - 1]}\``, input.length);
  }
  if (buffer.trim() !== "") {
    // Trailing text without `;` at the top level.
    finishBuffer(bufferStart);
  }

  return ast;
}

/**
 * Parses `property: value [!important]` into a declaration, or returns
 * `null` when the text has no `:`.
 */
export function parseDeclaration(text: string): Declaration | null {
  const colon = text.indexOf(":");
  if (colon === -1) return null;
  const property = text.slice(0, colon).trim();
  if (property === "") return null;
  let value = text.slice(colon + 1).trim();
  let important = false;
  const match = /\s*!important\s*$/i.exec(value);
  if (match !== null) {
    important = true;
    value = value.slice(0, match.index).trimEnd();
  }
  return decl(property, value, important);
}
