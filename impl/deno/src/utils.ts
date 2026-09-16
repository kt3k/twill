/**
 * Stable identifiers and normalization rules (SPEC §4.2).
 *
 * @module
 */

import { TwillError } from "./error.ts";
import { parseValue, toCss, type ValueAstNode } from "./value_parser.ts";

/** Named values and named modifiers must match this pattern. */
export const NAMED_VALUE_PATTERN = /^[a-zA-Z0-9_.%-]+$/;

/** Custom variant names must match this pattern and must not end in `_` or `-`. */
export const VARIANT_NAME_PATTERN = /^@?[a-z0-9][a-zA-Z0-9_-]*$/;

/** A theme prefix must match this pattern. */
export const PREFIX_PATTERN = /^[a-z]+$/;

export function isValidVariantName(name: string): boolean {
  if (!VARIANT_NAME_PATTERN.test(name)) return false;
  const last = name[name.length - 1];
  return last !== "_" && last !== "-";
}

/**
 * Escapes a string for use in a class selector, matching `CSS.escape`.
 */
export function escape(value: string): string {
  let result = "";
  const length = value.length;
  const first = value.charCodeAt(0);
  for (let i = 0; i < length; i++) {
    const c = value.charCodeAt(i);
    if (c === 0) {
      result += "�";
      continue;
    }
    if (
      (c >= 0x01 && c <= 0x1f) || c === 0x7f ||
      (i === 0 && c >= 0x30 && c <= 0x39) ||
      (i === 1 && c >= 0x30 && c <= 0x39 && first === 0x2d)
    ) {
      result += `\\${c.toString(16)} `;
      continue;
    }
    if (i === 0 && length === 1 && c === 0x2d) {
      result += `\\${value[i]}`;
      continue;
    }
    if (
      c >= 0x80 || c === 0x2d || c === 0x5f ||
      (c >= 0x30 && c <= 0x39) || (c >= 0x41 && c <= 0x5a) ||
      (c >= 0x61 && c <= 0x7a)
    ) {
      result += value[i];
      continue;
    }
    result += `\\${value[i]}`;
  }
  return result;
}

/** Reverses `escape`. */
export function unescape(value: string): string {
  if (!value.includes("\\")) return value;
  return value.replace(/\\([0-9a-fA-F]{1,6}\s?|[\s\S])/g, (_, esc: string) => {
    if (/^[0-9a-fA-F]/.test(esc)) {
      return String.fromCodePoint(parseInt(esc.trim(), 16));
    }
    return esc;
  });
}

const BACKSLASH = 0x5c;
const DOUBLE_QUOTE = 0x22;
const SINGLE_QUOTE = 0x27;
const OPEN_PAREN = 0x28;
const CLOSE_PAREN = 0x29;
const OPEN_BRACKET = 0x5b;
const CLOSE_BRACKET = 0x5d;
const OPEN_CURLY = 0x7b;
const CLOSE_CURLY = 0x7d;
const SEMICOLON = 0x3b;

/**
 * Splits `input` on a single-character separator, ignoring separators inside
 * `(...)`, `[...]`, `{...}`, inside single or double quotes, and immediately
 * after a backslash.
 */
export function segment(input: string, separator: string): string[] {
  const sep = separator.charCodeAt(0);
  const parts: string[] = [];
  let stack = "";
  let last = 0;
  for (let i = 0; i < input.length; i++) {
    const c = input.charCodeAt(i);
    if (c === BACKSLASH) {
      i++;
      continue;
    }
    if (c === DOUBLE_QUOTE || c === SINGLE_QUOTE) {
      for (i++; i < input.length; i++) {
        const d = input.charCodeAt(i);
        if (d === BACKSLASH) {
          i++;
        } else if (d === c) {
          break;
        }
      }
      continue;
    }
    if (c === OPEN_PAREN) {
      stack += ")";
    } else if (c === OPEN_BRACKET) {
      stack += "]";
    } else if (c === OPEN_CURLY) {
      stack += "}";
    } else if (c === CLOSE_PAREN || c === CLOSE_BRACKET || c === CLOSE_CURLY) {
      if (stack !== "" && stack.charCodeAt(stack.length - 1) === c) {
        stack = stack.slice(0, -1);
      }
    } else if (c === sep && stack === "") {
      parts.push(input.slice(last, i));
      last = i + 1;
    }
  }
  parts.push(input.slice(last));
  return parts;
}

/**
 * Checks bracket balance of an arbitrary value. Returns false on a closing
 * `)`, `]`, or `}` with an empty stack, on a mismatched closer, or on a
 * top-level `;`.
 */
export function isValidArbitrary(input: string): boolean {
  let stack = "";
  for (let i = 0; i < input.length; i++) {
    const c = input.charCodeAt(i);
    if (c === BACKSLASH) {
      i++;
      continue;
    }
    if (c === DOUBLE_QUOTE || c === SINGLE_QUOTE) {
      for (i++; i < input.length; i++) {
        const d = input.charCodeAt(i);
        if (d === BACKSLASH) {
          i++;
        } else if (d === c) {
          break;
        }
      }
      continue;
    }
    if (c === OPEN_PAREN) {
      stack += ")";
    } else if (c === OPEN_BRACKET) {
      stack += "]";
    } else if (c === CLOSE_PAREN || c === CLOSE_BRACKET || c === CLOSE_CURLY) {
      if (stack === "" || stack.charCodeAt(stack.length - 1) !== c) {
        return false;
      }
      stack = stack.slice(0, -1);
    } else if (c === SEMICOLON && stack === "") {
      return false;
    }
  }
  return true;
}

function isDigit(c: number): boolean {
  return c >= 0x30 && c <= 0x39;
}

/**
 * Natural string comparison: characters compare one by one, but when both
 * strings have a digit at the current position the full digit runs compare
 * numerically (then lexically on tie).
 */
export function compare(a: string, z: string): number {
  let i = 0;
  let j = 0;
  while (i < a.length && j < z.length) {
    const ca = a.charCodeAt(i);
    const cz = z.charCodeAt(j);
    if (isDigit(ca) && isDigit(cz)) {
      let ei = i;
      while (ei < a.length && isDigit(a.charCodeAt(ei))) ei++;
      let ej = j;
      while (ej < z.length && isDigit(z.charCodeAt(ej))) ej++;
      const ra = a.slice(i, ei);
      const rz = z.slice(j, ej);
      // Compare numerically without overflow: strip leading zeros, then
      // compare by length, then lexically.
      const na = ra.replace(/^0+(?=\d)/, "");
      const nz = rz.replace(/^0+(?=\d)/, "");
      if (na.length !== nz.length) return na.length - nz.length;
      if (na !== nz) return na < nz ? -1 : 1;
      if (ra !== rz) return ra < rz ? -1 : 1;
      i = ei;
      j = ej;
      continue;
    }
    if (ca !== cz) return ca - cz;
    i++;
    j++;
  }
  return (a.length - i) - (z.length - j);
}

/** `Number(v)` is an integer, is >= 0, and `String(Number(v)) == v`. */
export function isPositiveInteger(value: string): boolean {
  const n = Number(value);
  return Number.isInteger(n) && n >= 0 && String(n) === value;
}

/** As `isPositiveInteger` with > 0. */
export function isStrictPositiveInteger(value: string): boolean {
  const n = Number(value);
  return Number.isInteger(n) && n > 0 && String(n) === value;
}

/**
 * `Number(v)` is a multiple of 0.25 with no redundant leading or trailing
 * zeros. Used for spacing multipliers and opacity values.
 */
export function isMultipleOfQuarter(value: string): boolean {
  if (!/^-?(0|[1-9]\d*)(\.\d*[1-9])?$/.test(value)) return false;
  const n = Number(value);
  return Number.isFinite(n) && n % 0.25 === 0;
}

const RANGE_PATTERN = /^(-?\d+)\.\.(-?\d+)(?:\.\.(-?\d+))?$/;

/**
 * Brace expansion: `{a,b,c}` enumerates; `{1..5}`, `{10..0}`, and
 * `{0..20..5}` produce integer ranges; nesting is allowed.
 */
export function expandBraces(pattern: string): string[] {
  // Find the first top-level `{`.
  let depth = 0;
  let open = -1;
  let close = -1;
  for (let i = 0; i < pattern.length; i++) {
    const c = pattern[i];
    if (c === "\\") {
      i++;
      continue;
    }
    if (c === "{") {
      if (depth === 0) open = i;
      depth++;
    } else if (c === "}") {
      if (depth === 0) {
        throw new TwillError(`Unbalanced braces in \`${pattern}\``);
      }
      depth--;
      if (depth === 0) {
        close = i;
        break;
      }
    }
  }
  if (depth !== 0) throw new TwillError(`Unbalanced braces in \`${pattern}\``);
  if (open === -1) return [pattern];

  const prefix = pattern.slice(0, open);
  const inner = pattern.slice(open + 1, close);
  const suffix = pattern.slice(close + 1);

  let items: string[];
  const range = RANGE_PATTERN.exec(inner);
  if (range !== null) {
    const start = Number(range[1]);
    const end = Number(range[2]);
    let step = range[3] === undefined ? 1 : Math.abs(Number(range[3]));
    if (step === 0) {
      throw new TwillError(`Step cannot be zero in \`${pattern}\``);
    }
    if (end < start) step = -step;
    items = [];
    if (step > 0) {
      for (let n = start; n <= end; n += step) items.push(String(n));
    } else {
      for (let n = start; n >= end; n += step) items.push(String(n));
    }
  } else {
    items = segment(inner, ",");
  }

  const result: string[] = [];
  for (const item of items) {
    for (const expanded of expandBraces(item + suffix)) {
      result.push(prefix + expanded);
    }
  }
  return result;
}

/** Strips one pair of surrounding single or double quotes. */
export function unquote(value: string): string | null {
  if (value.length < 2) return null;
  const first = value[0];
  const last = value[value.length - 1];
  if ((first === '"' || first === "'") && last === first) {
    return value.slice(1, -1);
  }
  return null;
}

const MATH_FUNCTIONS = new Set([
  "calc",
  "min",
  "max",
  "clamp",
  "round",
  "mod",
  "rem",
  "sin",
  "cos",
  "tan",
  "asin",
  "acos",
  "atan",
  "atan2",
  "pow",
  "sqrt",
  "hypot",
  "log",
  "exp",
  "abs",
  "sign",
]);

export function isMathFunction(name: string): boolean {
  return MATH_FUNCTIONS.has(name);
}

/**
 * Decodes an arbitrary value: `_` becomes a space except inside `url(...)`
 * (and `*_url(...)`), in the first argument of `var(...)` and `theme(...)`,
 * and when escaped as `\_`. Math operators inside math functions receive
 * surrounding spaces.
 */
export function decodeArbitraryValue(input: string): string {
  if (!input.includes("(")) {
    return replaceUnderscores(input);
  }
  const ast = parseValue(input);
  decodeNodes(ast);
  return addWhitespaceAroundMathOperators(toCss(ast));
}

function replaceUnderscores(input: string): string {
  if (!input.includes("_")) return input;
  let out = "";
  for (let i = 0; i < input.length; i++) {
    const c = input[i];
    if (c === "\\" && input[i + 1] === "_") {
      out += "_";
      i++;
    } else if (c === "_") {
      out += " ";
    } else {
      out += c;
    }
  }
  return out;
}

function unescapeUnderscores(input: string): string {
  return input.replaceAll("\\_", "_");
}

function decodeNodes(nodes: ValueAstNode[]): void {
  for (const node of nodes) {
    if (node.kind === "separator") continue;
    if (node.kind === "word") {
      node.value = replaceUnderscores(node.value);
      continue;
    }
    const name = node.value;
    if (name === "url" || name.endsWith("_url")) {
      continue;
    }
    if (name === "var" || name === "theme" || name === "--theme") {
      // The first argument keeps its underscores; the rest is decoded.
      let firstComma = node.nodes.findIndex((n) =>
        n.kind === "separator" && n.value === ","
      );
      if (firstComma === -1) firstComma = node.nodes.length;
      for (let j = 0; j < firstComma; j++) {
        const child = node.nodes[j];
        if (child.kind === "word") {
          child.value = unescapeUnderscores(child.value);
        }
      }
      decodeNodes(node.nodes.slice(firstComma));
      continue;
    }
    decodeNodes(node.nodes);
  }
}

function isIdentChar(c: string): boolean {
  return /[a-zA-Z0-9_-]/.test(c);
}

/**
 * Whether the `-` or `+` at `index` follows a value: a digit, `%`, `)`, or a
 * unit (letters immediately preceded by a digit or `.`).
 */
function followsValue(input: string, index: number): boolean {
  let i = index - 1;
  while (i >= 0 && input[i] === " ") i--;
  if (i < 0) return false;
  const c = input[i];
  if (/[0-9%)]/.test(c)) return true;
  if (/[a-zA-Z]/.test(c)) {
    while (i >= 0 && /[a-zA-Z]/.test(input[i])) i--;
    return i >= 0 && /[0-9.]/.test(input[i]);
  }
  return false;
}

/**
 * Inserts spaces around `+`, `-`, `*`, and `/` when they act as operators
 * inside math functions, so `calc(1px+2px)` becomes `calc(1px + 2px)`.
 * The contents of non-math functions such as `var(...)` are left untouched.
 */
export function addWhitespaceAroundMathOperators(input: string): string {
  if (
    !/(?:^|[^a-zA-Z0-9_-])(?:calc|min|max|clamp|round|mod|rem|sin|cos|tan|asin|acos|atan|atan2|pow|sqrt|hypot|log|exp|abs|sign)\(/
      .test(input)
  ) {
    return input;
  }
  let result = "";
  // Whether each open parenthesis level is a math context.
  const stack: boolean[] = [];
  const inMath = () => stack.length > 0 && stack[stack.length - 1];

  for (let i = 0; i < input.length; i++) {
    const c = input[i];

    if (c === "\\") {
      result += input.slice(i, i + 2);
      i++;
      continue;
    }

    if (c === '"' || c === "'") {
      const start = i;
      for (i++; i < input.length; i++) {
        if (input[i] === "\\") i++;
        else if (input[i] === c) break;
      }
      result += input.slice(start, i + 1);
      continue;
    }

    if (c === "(") {
      let start = i;
      while (start > 0 && isIdentChar(input[start - 1])) start--;
      let name = input.slice(start, i);
      if (name.startsWith("-") && !name.startsWith("--")) name = name.slice(1);
      if (name === "") stack.push(inMath());
      else stack.push(isMathFunction(name));
      result += c;
      continue;
    }

    if (c === ")") {
      stack.pop();
      result += c;
      continue;
    }

    if (!inMath()) {
      result += c;
      continue;
    }

    let isOperator = false;
    if (c === "*" || c === "/") {
      isOperator = true;
    } else if (c === "+" || c === "-") {
      let next = i + 1;
      while (next < input.length && input[next] === " ") next++;
      isOperator = followsValue(input, i) && next < input.length &&
        input[next] !== ")";
    }

    if (isOperator) {
      if (!result.endsWith(" ")) result += " ";
      result += `${c} `;
      while (i + 1 < input.length && input[i + 1] === " ") i++;
      continue;
    }

    result += c;
  }
  return result;
}
