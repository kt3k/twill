/**
 * Helpers shared by the test files. Not part of the public API.
 *
 * @module
 */

import { type AstNode, context, decl, styleRule } from "./ast.ts";
import { resolveBuiltin, type StylesheetLoader } from "./builtin.ts";
import { collectDirectives } from "./directives.ts";
import { buildDesignSystem, type DesignSystem } from "./design_system.ts";
import { TwillError } from "./error.ts";
import { substituteAtImports } from "./import.ts";
import { parse } from "./parser.ts";
import { serialize } from "./serializer.ts";
import { Theme } from "./theme.ts";
import { asColor, isFallbackDefinition } from "./utilities.ts";
import { escape } from "./utils.ts";
import { applyVariant } from "./variants.ts";

/** A loader serving the built-in stylesheets and an in-memory file map. */
export function memoryLoader(
  files: Record<string, string> = {},
): StylesheetLoader {
  return (id, base) => {
    const builtin = resolveBuiltin(id, base);
    if (builtin !== null) return builtin;
    const path = `${base}/${id.replace(/^\.\//, "")}`;
    if (!(path in files)) throw new TwillError(`Cannot find ${path}`);
    return {
      path,
      base: path.slice(0, path.lastIndexOf("/")),
      content: files[path],
    };
  };
}

/**
 * Builds a design system from a stylesheet (defaulting to the built-in
 * stylesheet) without running the full compile pipeline.
 */
export async function designSystemFor(
  css = '@import "twill";',
  files: Record<string, string> = {},
): Promise<DesignSystem> {
  const theme = new Theme();
  const ast: AstNode[] = [context({ base: "/root" }, parse(css))];
  await substituteAtImports(ast, "/root", memoryLoader(files));
  const state = collectDirectives(ast, theme);
  const ds = buildDesignSystem(theme);
  ds.important = state.important;
  return ds;
}

/**
 * Compiles one raw candidate into nested (unflattened) CSS through the
 * registries, without theme function substitution.
 */
export function compileRaw(ds: DesignSystem, raw: string): string {
  const rules: AstNode[] = [];
  for (const candidate of ds.parseCandidate(raw)) {
    const results: AstNode[][] = [];
    if (candidate.kind === "arbitrary") {
      const value = asColor(candidate.value, candidate.modifier, ds.theme);
      if (value !== null) results.push([decl(candidate.property, value)]);
    } else {
      const definitions = ds.utilities.get(candidate.root).filter((d) =>
        d.kind === candidate.kind
      );
      const ordered = [
        ...definitions.filter((d) => !isFallbackDefinition(d)),
        ...definitions.filter(isFallbackDefinition),
      ];
      for (const definition of ordered) {
        const result = definition.compile(candidate);
        if (result === undefined) continue;
        if (result === null) {
          if (definition.types !== undefined) break;
          continue;
        }
        results.push(result);
      }
    }
    for (const nodes of results) {
      const node = styleRule(`.${escape(raw)}`, nodes);
      let ok = true;
      for (const variant of candidate.variants) {
        if (applyVariant(node, variant, ds.variants) === null) {
          ok = false;
          break;
        }
      }
      if (ok) rules.push(node);
    }
  }
  return serialize(rules);
}
