/**
 * Theme storage and resolution (SPEC §7).
 *
 * @module
 */

import type { AtRule } from "./ast.ts";
import { withAlpha } from "./color.ts";
import { TwillError } from "./error.ts";
import { escape, unescape } from "./utils.ts";

/** Theme entry option bits (SPEC §4.1.2). */
export const ThemeOptions = {
  NONE: 0,
  /** Consumers embed the raw value instead of `var(...)`. */
  INLINE: 1 << 0,
  /** The variable is not printed; consumers embed `var(key, value)`. */
  REFERENCE: 1 << 1,
  /** A later non-default entry with the same key wins even if it was added first. */
  DEFAULT: 1 << 2,
  /** The variable is printed even when unused. */
  STATIC: 1 << 3,
  /** Something referenced the variable. */
  USED: 1 << 4,
} as const;

export type ThemeOptions = number;

export interface ThemeEntry {
  value: string;
  options: ThemeOptions;
}

/**
 * Sub-namespaces that namespace clearing and resolution skip (SPEC §7.2.1).
 * Each entry also covers keys with a further `-` suffix.
 */
const IGNORED_NAMESPACES: Record<string, string[]> = {
  "--font": ["--font-weight", "--font-size"],
  "--inset": ["--inset-shadow", "--inset-ring"],
  "--text": [
    "--text-color",
    "--text-decoration-color",
    "--text-decoration-thickness",
    "--text-indent",
    "--text-shadow",
    "--text-underline-offset",
  ],
  "--grid-column": ["--grid-column-start", "--grid-column-end"],
  "--grid-row": ["--grid-row-start", "--grid-row-end"],
};

function isIgnoredKey(namespace: string, key: string): boolean {
  const ignored = IGNORED_NAMESPACES[namespace];
  if (ignored === undefined) return false;
  for (const ns of ignored) {
    if (key === ns || key.startsWith(`${ns}-`)) return true;
  }
  return false;
}

export class Theme {
  readonly #values = new Map<string, ThemeEntry>();
  readonly #keyframes = new Set<AtRule>();
  prefix: string | null = null;

  constructor(
    entries?: Iterable<[string, ThemeEntry]>,
    keyframes?: Iterable<AtRule>,
  ) {
    if (entries) { for (const [k, v] of entries) this.#values.set(k, v); }
    if (keyframes) { for (const k of keyframes) this.#keyframes.add(k); }
  }

  get size(): number {
    return this.#values.size;
  }

  /** Registers a value (SPEC §7.2). */
  add(
    key: string,
    value: string,
    options: ThemeOptions = ThemeOptions.NONE,
  ): void {
    if (key.endsWith("-*")) {
      if (value !== "initial") {
        throw new TwillError(
          `Invalid theme value \`${value}\` for namespace \`${key}\``,
        );
      }
      if (key === "--*") {
        this.#values.clear();
      } else {
        this.#clearNamespace(key.slice(0, -2));
      }
      return;
    }

    if (options & ThemeOptions.DEFAULT) {
      const existing = this.#values.get(key);
      if (
        existing !== undefined && !(existing.options & ThemeOptions.DEFAULT)
      ) return;
    }

    if (value === "initial") {
      this.#values.delete(key);
    } else {
      this.#values.set(key, { value, options });
    }
  }

  #clearNamespace(namespace: string): void {
    for (const key of this.#values.keys()) {
      if (key.startsWith(namespace) && !isIgnoredKey(namespace, key)) {
        this.#values.delete(key);
      }
    }
  }

  addKeyframes(node: AtRule): void {
    this.#keyframes.add(node);
  }

  getKeyframes(): AtRule[] {
    return [...this.#keyframes];
  }

  /** Every stored entry in insertion order. */
  entries(): [string, ThemeEntry][] {
    return [...this.#values.entries()];
  }

  has(key: string): boolean {
    return this.#values.has(key);
  }

  getOptions(key: string): ThemeOptions {
    return this.#values.get(key)?.options ?? ThemeOptions.NONE;
  }

  /** Finds the key for a candidate value in the given namespaces (SPEC §7.3). */
  resolveKey(
    candidateValue: string | null,
    namespaces: string[],
  ): string | null {
    for (const namespace of namespaces) {
      let key = candidateValue === null
        ? namespace
        : `${namespace}-${candidateValue}`;
      if (!this.#values.has(key)) {
        if (candidateValue !== null && candidateValue.includes(".")) {
          key = `${namespace}-${candidateValue.replaceAll(".", "_")}`;
          if (!this.#values.has(key)) continue;
        } else {
          continue;
        }
      }
      if (isIgnoredKey(namespace, key)) continue;
      return key;
    }
    return null;
  }

  /** Returns the value of `key` as a `var(...)` reference or an inline value. */
  #reference(key: string, options: ThemeOptions): string {
    const entry = this.#values.get(key)!;
    if ((options | entry.options) & ThemeOptions.INLINE) {
      return entry.value;
    }
    const name = escape(this.prefixKey(key));
    if (entry.options & ThemeOptions.REFERENCE) {
      return `var(${name}, ${entry.value})`;
    }
    return `var(${name})`;
  }

  resolve(
    candidateValue: string | null,
    namespaces: string[],
    options: ThemeOptions = ThemeOptions.NONE,
  ): string | null {
    const key = this.resolveKey(candidateValue, namespaces);
    if (key === null) return null;
    return this.#reference(key, options);
  }

  resolveValue(
    candidateValue: string | null,
    namespaces: string[],
  ): string | null {
    const key = this.resolveKey(candidateValue, namespaces);
    if (key === null) return null;
    return this.#values.get(key)!.value;
  }

  resolveWith(
    candidateValue: string | null,
    namespaces: string[],
    nestedKeys: string[] = [],
  ): [string, Record<string, string>] | null {
    const key = this.resolveKey(candidateValue, namespaces);
    if (key === null) return null;
    const extra: Record<string, string> = {};
    for (const nested of nestedKeys) {
      const nestedKey = `${key}${nested}`;
      if (this.#values.has(nestedKey)) {
        extra[nested] = this.#reference(nestedKey, ThemeOptions.NONE);
      }
    }
    return [this.#reference(key, ThemeOptions.NONE), extra];
  }

  /** Returns the raw value of the first key that exists, or null. */
  get(keys: string[]): string | null {
    for (const key of keys) {
      const entry = this.#values.get(key);
      if (entry !== undefined) return entry.value;
    }
    return null;
  }

  /**
   * Returns a map with a null key for `ns` itself, keys with the `<ns>-`
   * prefix removed, and keys starting with `<ns>--` with only `<ns>` removed.
   */
  namespace(ns: string): Map<string | null, string> {
    const result = new Map<string | null, string>();
    const prefix = `${ns}-`;
    for (const [key, entry] of this.#values) {
      if (key === ns) {
        result.set(null, entry.value);
      } else if (key.startsWith(`${ns}--`)) {
        result.set(key.slice(ns.length), entry.value);
      } else if (key.startsWith(prefix)) {
        result.set(key.slice(prefix.length), entry.value);
      }
    }
    return result;
  }

  /**
   * Every key under each namespace with the prefix removed, excluding keys
   * that contain a second `--` and excluding ignored keys.
   */
  keysInNamespaces(namespaces: string[]): string[] {
    const keys: string[] = [];
    for (const namespace of namespaces) {
      const prefix = `${namespace}-`;
      for (const key of this.#values.keys()) {
        if (!key.startsWith(prefix)) continue;
        if (key.indexOf("--", 2) !== -1) continue;
        if (isIgnoredKey(namespace, key)) continue;
        keys.push(key.slice(prefix.length));
      }
    }
    return keys;
  }

  prefixKey(key: string): string {
    if (this.prefix === null) return key;
    return `--${this.prefix}-${key.slice(2)}`;
  }

  /** Removes the prefix from a prefixed key. */
  unprefixKey(key: string): string {
    if (this.prefix === null) return key;
    const prefix = `--${this.prefix}-`;
    if (key.startsWith(prefix)) return `--${key.slice(prefix.length)}`;
    return key;
  }

  /**
   * Sets `USED` on the (unprefixed, unescaped) key. Returns true when the
   * flag was not set before.
   */
  markUsedVariable(key: string): boolean {
    const entry = this.#values.get(this.unprefixKey(unescape(key)));
    if (entry === undefined) return false;
    if (entry.options & ThemeOptions.USED) return false;
    entry.options |= ThemeOptions.USED;
    return true;
  }

  /**
   * Resolves a theme path such as `--color-red-500/50` (SPEC §7.3).
   */
  resolveThemeValue(path: string, forceInline = true): string | null {
    let modifier: string | null = null;
    const trimmed = path.trim();
    let key = trimmed;
    const slash = trimmed.lastIndexOf("/");
    if (slash !== -1) {
      key = trimmed.slice(0, slash).trim();
      modifier = trimmed.slice(slash + 1).trim();
    }
    const value = this.resolve(
      null,
      [key],
      forceInline ? ThemeOptions.INLINE : ThemeOptions.NONE,
    );
    if (value === null) return null;
    if (modifier !== null) return withAlpha(value, modifier);
    return value;
  }
}
