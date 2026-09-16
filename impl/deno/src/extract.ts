/**
 * Candidate extraction from arbitrary text (SPEC §13.4).
 *
 * @module
 */

const SPACE = 0x20;
const TAB = 0x09;
const LINE_BREAK = 0x0a;
const CARRIAGE_RETURN = 0x0d;
const FORM_FEED = 0x0c;
const DOUBLE_QUOTE = 0x22;
const SINGLE_QUOTE = 0x27;
const BACKTICK = 0x60;
const DOT = 0x2e;
const CLOSE_CURLY = 0x7d;
const OPEN_CURLY = 0x7b;
const GREATER = 0x3e;
const LESS = 0x3c;
const CLOSE_BRACKET = 0x5d;
const OPEN_BRACKET = 0x5b;
const OPEN_PAREN = 0x28;
const CLOSE_PAREN = 0x29;
const EQUALS = 0x3d;
const BACKSLASH = 0x5c;
const COLON = 0x3a;
const SLASH = 0x2f;
const DASH = 0x2d;
const UNDERSCORE = 0x5f;
const PERCENT = 0x25;
const AT_SIGN = 0x40;
const EXCLAMATION = 0x21;

function isWhitespace(c: number): boolean {
  return c === SPACE || c === TAB || c === LINE_BREAK ||
    c === CARRIAGE_RETURN || c === FORM_FEED;
}

function isQuote(c: number): boolean {
  return c === DOUBLE_QUOTE || c === SINGLE_QUOTE || c === BACKTICK;
}

function isStartBoundary(c: number): boolean {
  return isWhitespace(c) || isQuote(c) || c === DOT || c === CLOSE_CURLY ||
    c === GREATER;
}

function isEndBoundary(c: number): boolean {
  return isWhitespace(c) || isQuote(c) || c === CLOSE_BRACKET ||
    c === OPEN_CURLY ||
    c === EQUALS || c === BACKSLASH || c === LESS;
}

function isLetter(c: number): boolean {
  return (c >= 0x61 && c <= 0x7a) || (c >= 0x41 && c <= 0x5a);
}

function isDigit(c: number): boolean {
  return c >= 0x30 && c <= 0x39;
}

function isNameChar(c: number): boolean {
  return isLetter(c) || isDigit(c) || c === UNDERSCORE || c === DASH;
}

/**
 * Consumes a balanced bracket group starting at `i` (which must be the
 * opener). Returns the index after the closer, or -1 when unbalanced or
 * interrupted by a line break.
 */
function consumeBalanced(
  text: string,
  i: number,
  open: number,
  close: number,
): number {
  let depth = 0;
  for (let j = i; j < text.length; j++) {
    const c = text.charCodeAt(j);
    if (c === LINE_BREAK) return -1;
    if (c === BACKSLASH) {
      j++;
      continue;
    }
    if (c === open) depth++;
    else if (c === close) {
      depth--;
      if (depth === 0) return j + 1;
    }
  }
  return -1;
}

/** Consumes `-[...]` or `-(...)` when present at `i - 1`/`i`. */
function consumeBracketSuffix(text: string, i: number): number {
  const c = text.charCodeAt(i);
  if (c === OPEN_BRACKET) {
    return consumeBalanced(text, i, OPEN_BRACKET, CLOSE_BRACKET);
  }
  if (c === OPEN_PAREN) {
    return consumeBalanced(text, i, OPEN_PAREN, CLOSE_PAREN);
  }
  return -1;
}

/** Consumes `/modifier` at `i` (which must be the slash). Returns the end. */
function consumeModifier(text: string, i: number): number {
  const c = text.charCodeAt(i + 1);
  if (c === OPEN_BRACKET || c === OPEN_PAREN) {
    const end = consumeBracketSuffix(text, i + 1);
    return end === -1 ? i : end;
  }
  let j = i + 1;
  while (j < text.length) {
    const d = text.charCodeAt(j);
    if (isNameChar(d) || d === DOT || d === PERCENT) j++;
    else break;
  }
  if (j === i + 1) return i;
  const last = text.charCodeAt(j - 1);
  if (last === DASH || last === UNDERSCORE) return i;
  return j;
}

/**
 * Consumes a variant at `i`. Returns the index after the variant (before
 * the `:`), or -1.
 */
function consumeVariant(text: string, i: number): number {
  const c = text.charCodeAt(i);
  if (c === OPEN_BRACKET) {
    return consumeBalanced(text, i, OPEN_BRACKET, CLOSE_BRACKET);
  }
  if (!isLetter(c) && c !== AT_SIGN) return -1;
  let j = i + 1;
  while (j < text.length) {
    const d = text.charCodeAt(j);
    if (isNameChar(d)) {
      j++;
      continue;
    }
    if (
      (d === OPEN_BRACKET || d === OPEN_PAREN) &&
      text.charCodeAt(j - 1) === DASH
    ) {
      const end = consumeBracketSuffix(text, j);
      if (end === -1) return -1;
      j = end;
      continue;
    }
    break;
  }
  if (j === i + 1 && c === AT_SIGN) return -1;
  const last = text.charCodeAt(j - 1);
  if (last === DASH || last === UNDERSCORE) return -1;
  if (text.charCodeAt(j) === SLASH) j = consumeModifier(text, j);
  return j;
}

/**
 * Consumes a utility at `i`. Returns the index after it, or -1.
 */
function consumeUtility(text: string, i: number): number {
  let j = i;
  if (text.charCodeAt(j) === EXCLAMATION) j++;
  const c = text.charCodeAt(j);

  if (c === OPEN_BRACKET) {
    const end = consumeBalanced(text, j, OPEN_BRACKET, CLOSE_BRACKET);
    if (end === -1) return -1;
    if (text.slice(j, end).indexOf(":") === -1) return -1;
    j = end;
  } else {
    if (c === DASH) {
      const next = text.charCodeAt(j + 1);
      if (!isLetter(next) && !isDigit(next)) return -1;
      j++;
    } else if (!isLetter(c) && c !== AT_SIGN) {
      return -1;
    }
    j++;
    while (j < text.length) {
      const d = text.charCodeAt(j);
      if (isNameChar(d) || d === PERCENT) {
        j++;
        continue;
      }
      if (d === DOT) {
        if (
          isDigit(text.charCodeAt(j - 1)) && isDigit(text.charCodeAt(j + 1))
        ) {
          j++;
          continue;
        }
        break;
      }
      if (
        (d === OPEN_BRACKET || d === OPEN_PAREN) &&
        text.charCodeAt(j - 1) === DASH
      ) {
        const end = consumeBracketSuffix(text, j);
        if (end === -1) return -1;
        j = end;
        continue;
      }
      break;
    }
    const last = text.charCodeAt(j - 1);
    if (last === DASH || last === UNDERSCORE) return -1;
  }

  if (text.charCodeAt(j) === SLASH) j = consumeModifier(text, j);
  if (text.charCodeAt(j) === EXCLAMATION) j++;
  return j;
}

/** Consumes a full candidate `(variant ":")* utility` at `i`. */
function consumeCandidate(text: string, i: number): number {
  let j = i;
  while (true) {
    const end = consumeVariant(text, j);
    if (end !== -1 && text.charCodeAt(end) === COLON) {
      j = end + 1;
      continue;
    }
    break;
  }
  return consumeUtility(text, j);
}

/** Consumes a `--name` custom property reference at `i`. */
function consumeVariable(text: string, i: number): number {
  let j = i + 2;
  while (j < text.length && isNameChar(text.charCodeAt(j))) j++;
  return j === i + 2 ? -1 : j;
}

/**
 * Extracts every candidate-like span from `text` (SPEC §13.4). Recall is
 * favored over precision: unknown candidates are harmless.
 */
export function extractCandidates(
  text: string,
  into: Set<string> = new Set(),
): Set<string> {
  const length = text.length;
  let i = 0;
  while (i < length) {
    const c = text.charCodeAt(i);
    if (i > 0 && !isStartBoundary(text.charCodeAt(i - 1))) {
      i++;
      continue;
    }

    let end = -1;
    if (c === DASH && text.charCodeAt(i + 1) === DASH) {
      end = consumeVariable(text, i);
    } else if (
      isLetter(c) || c === AT_SIGN || c === DASH || c === EXCLAMATION ||
      c === OPEN_BRACKET
    ) {
      end = consumeCandidate(text, i);
    }

    if (
      end !== -1 && end > i &&
      (end === length || isEndBoundary(text.charCodeAt(end)))
    ) {
      into.add(text.slice(i, end));
      i = end;
      continue;
    }

    // Skip to the next boundary so that substrings are never emitted.
    i++;
    while (i < length && !isStartBoundary(text.charCodeAt(i - 1))) i++;
  }
  return into;
}
