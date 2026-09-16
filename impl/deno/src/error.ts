/** Position of a character in a source text (1-based line and column). */
export interface SourcePosition {
  line: number;
  column: number;
}

/**
 * Error raised by the compiler.
 *
 * Syntax errors carry the position of the offending character in the input;
 * semantic errors (invalid directives, unknown candidates in `@apply`, ...)
 * usually do not.
 */
export class TwillError extends Error {
  readonly position: SourcePosition | undefined;

  constructor(message: string, position?: SourcePosition) {
    super(
      position ? `${message} (${position.line}:${position.column})` : message,
    );
    this.name = "TwillError";
    this.position = position;
  }
}

/** Computes the 1-based line and column of `index` in `input`. */
export function positionAt(input: string, index: number): SourcePosition {
  let line = 1;
  let column = 1;
  const end = Math.min(index, input.length);
  for (let i = 0; i < end; i++) {
    if (input.charCodeAt(i) === 10) {
      line++;
      column = 1;
    } else {
      column++;
    }
  }
  return { line, column };
}
