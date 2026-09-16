/**
 * Built-in stylesheets as embedded resources (SPEC §6.3).
 *
 * @module
 */

import { BUNDLED_CSS } from "./bundled_css.ts";
import { TwillError } from "./error.ts";

/** The result of loading a stylesheet (SPEC §6.2). */
export interface LoadedStylesheet {
  /** A path identifying the stylesheet, used for full-rebuild tracking. */
  path: string;
  /** The base for resolving relative imports inside the stylesheet. */
  base: string;
  content: string;
}

/**
 * Loads a stylesheet by id (SPEC §6.2). Relative ids resolve against `base`.
 */
export type StylesheetLoader = (
  id: string,
  base: string,
) => LoadedStylesheet | Promise<LoadedStylesheet>;

/**
 * The `base` of every built-in stylesheet. It is distinct from any user
 * directory, so a relative import from user CSS never resolves to an
 * embedded resource.
 */
export const BUILTIN_BASE = "twill:";

/** The path of a built-in stylesheet, for example `twill:theme.css`. */
export function builtinPath(name: string): string {
  return `${BUILTIN_BASE}${name}`;
}

export function isBuiltinPath(path: string): boolean {
  return path.startsWith(BUILTIN_BASE);
}

/**
 * Resolves the id `twill`, the ids `twill/<name>.css`, and relative ids
 * requested from inside a built-in stylesheet. Returns `null` for any other
 * request so that the host loader can fall through to the filesystem.
 */
export function resolveBuiltin(
  id: string,
  base: string,
): LoadedStylesheet | null {
  let name: string;
  if (id === "twill") {
    name = "index.css";
  } else if (id.startsWith("twill/")) {
    name = id.slice("twill/".length);
  } else if (base === BUILTIN_BASE) {
    name = id.startsWith("./") ? id.slice(2) : id;
  } else {
    return null;
  }
  const content = BUNDLED_CSS[name];
  if (content === undefined) {
    throw new TwillError(`Unknown built-in stylesheet \`${id}\``);
  }
  return { path: builtinPath(name), base: BUILTIN_BASE, content };
}
