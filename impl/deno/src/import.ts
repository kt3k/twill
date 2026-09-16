/**
 * `@import` and `@reference` resolution (SPEC §6.2).
 *
 * @module
 */

import { type AstNode, atRule, context, walk, WalkAction } from "./ast.ts";
import type { StylesheetLoader } from "./builtin.ts";
import { TwillError } from "./error.ts";
import { Features } from "./features.ts";
import { parse } from "./parser.ts";
import { unquote } from "./utils.ts";
import { parseValue, toCss } from "./value_parser.ts";

const MAX_DEPTH = 100;

export interface ImportParams {
  uri: string;
  layer: string | null;
  media: string | null;
  supports: string | null;
}

/**
 * Parses the params of an `@import` rule. Returns `null` for imports that
 * must be left untouched (`url(...)`, `data:`, `http://`, `https://`).
 */
export function parseImportParams(params: string): ImportParams | null {
  const nodes = parseValue(params);
  // Skip leading whitespace.
  while (nodes.length > 0 && nodes[0].kind === "separator") nodes.shift();
  const first = nodes.shift();
  if (first === undefined || first.kind !== "word") return null;
  const uri = unquote(first.value);
  if (uri === null) return null;
  if (
    uri.startsWith("data:") || uri.startsWith("http://") ||
    uri.startsWith("https://")
  ) {
    return null;
  }

  let layer: string | null = null;
  let supports: string | null = null;
  const media: string[] = [];
  let sawMedia = false;
  for (const node of nodes) {
    if (node.kind === "separator") {
      if (media.length > 0) media.push(node.value);
      continue;
    }
    if (node.kind === "function" && node.value === "layer") {
      if (supports !== null || sawMedia) {
        throw new TwillError(
          `\`layer(...)\` must appear before \`supports(...)\` and media queries in \`@import ${params}\``,
        );
      }
      if (layer !== null) {
        throw new TwillError(
          `Duplicate \`layer(...)\` in \`@import ${params}\``,
        );
      }
      layer = toCss(node.nodes).trim();
      continue;
    }
    if (node.kind === "word" && node.value === "layer") {
      if (supports !== null || sawMedia) {
        throw new TwillError(
          `\`layer\` must appear before \`supports(...)\` and media queries in \`@import ${params}\``,
        );
      }
      layer = "";
      continue;
    }
    if (node.kind === "function" && node.value === "supports") {
      if (sawMedia) {
        throw new TwillError(
          `\`supports(...)\` must appear before media queries in \`@import ${params}\``,
        );
      }
      supports = toCss(node.nodes).trim();
      continue;
    }
    sawMedia = true;
    media.push(toCss([node]));
  }
  const mediaText = media.join("").trim();
  return { uri, layer, media: mediaText === "" ? null : mediaText, supports };
}

/** Wraps imported nodes per SPEC §6.2. */
export function buildImportNodes(
  nodes: AstNode[],
  layer: string | null,
  media: string | null,
  supports: string | null,
): AstNode[] {
  let root = nodes;
  if (layer !== null) root = [atRule("@layer", layer, root)];
  if (media !== null) root = [atRule("@media", media, root)];
  if (supports !== null) {
    const condition = supports.startsWith("(") ? supports : `(${supports})`;
    root = [atRule("@supports", condition, root)];
  }
  return root;
}

/**
 * Expands `@import` and `@reference` in place. Returns the feature flags
 * (`AT_IMPORT` whenever an import was resolved).
 */
export async function substituteAtImports(
  ast: AstNode[],
  base: string,
  load: StylesheetLoader,
  depth = 0,
): Promise<number> {
  let features = Features.NONE;
  const promises: Promise<void>[] = [];

  walk(ast, (node, { replaceWith }) => {
    if (node.kind !== "at-rule") return;
    if (node.name !== "@import" && node.name !== "@reference") return;
    const parsed = parseImportParams(node.params);
    if (parsed === null) return;
    if (node.name === "@reference") {
      parsed.media = "reference";
    }
    if (depth > MAX_DEPTH) {
      throw new TwillError(
        `Exceeded maximum recursion depth while resolving \`${node.name} ${node.params}\``,
      );
    }
    features |= Features.AT_IMPORT;
    const placeholder = context({}, []);
    promises.push((async () => {
      const loaded = await load(parsed.uri, base);
      const imported = parse(loaded.content);
      features |= await substituteAtImports(
        imported,
        loaded.base,
        load,
        depth + 1,
      );
      placeholder.nodes = buildImportNodes(
        [context({ base: loaded.base }, imported)],
        parsed.layer,
        parsed.media,
        parsed.supports,
      );
    })());
    replaceWith(placeholder);
    return WalkAction.Skip;
  });

  await Promise.all(promises);
  return features;
}
