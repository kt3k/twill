/**
 * The compiler entry point: `compile` and `build` (SPEC §6.1, §15.1, §15.2).
 *
 * @module
 */

import {
  type AstNode,
  atRoot,
  type Context,
  context,
  decl,
  walk,
  WalkAction,
} from "./ast.ts";
import { substituteAtApply, topologicalSort } from "./at_apply.ts";
import { registerCustomUtility } from "./at_utility.ts";
import { substituteAtVariant } from "./at_variant.ts";
import { resolveBuiltin, type StylesheetLoader } from "./builtin.ts";
import { compileCandidates } from "./compile_candidates.ts";
import { buildDesignSystem, type DesignSystem } from "./design_system.ts";
import {
  collectDirectives,
  type SourceEntry,
  type SourceRoot,
} from "./directives.ts";
import { TwillError } from "./error.ts";
import { Features } from "./features.ts";
import { substituteAtImports } from "./import.ts";
import { optimizeAst } from "./optimize.ts";
import { parse } from "./parser.ts";
import { serialize } from "./serializer.ts";
import { Theme, ThemeOptions } from "./theme.ts";
import { substituteFunctions } from "./theme_functions.ts";
import { escape } from "./utils.ts";

export interface CompileOptions {
  /** The base directory for resolving `@source` and relative imports. */
  base?: string;
  /**
   * Loads a stylesheet by id. When omitted, only the built-in stylesheets
   * can be imported.
   */
  loadStylesheet?: StylesheetLoader;
}

/** The compiler handle returned by `compile` (SPEC §4.1.11). */
export interface Compiler {
  /** Source entries collected from `@source`. */
  sources: SourceEntry[];
  /** The root from `@twill utilities source(...)`. */
  root: SourceRoot;
  /** Feature flags (`Features`). */
  features: number;
  /** The design system, exposed for inspection and tooling. */
  designSystem: DesignSystem;
  /** Builds the CSS for the accumulated candidates. */
  build(candidates: Iterable<string>): string;
}

/** The default loader: only the built-in stylesheets. */
export const builtinLoader: StylesheetLoader = (id, base) => {
  const builtin = resolveBuiltin(id, base);
  if (builtin === null) {
    throw new TwillError(
      `Cannot resolve \`${id}\`: no \`loadStylesheet\` option was provided to \`compile\`.`,
    );
  }
  return builtin;
};

/** Compiles a stylesheet (SPEC §15.1). */
export async function compile(
  css: string,
  options: CompileOptions = {},
): Promise<Compiler> {
  const base = options.base ?? "";
  const load = options.loadStylesheet ?? builtinLoader;

  let ast: AstNode[] = parse(css);
  ast = [context({ base }, ast)];

  let features = await substituteAtImports(ast, base, load);

  const theme = new Theme();
  const state = collectDirectives(ast, theme);
  features |= state.features;

  const ds = buildDesignSystem(theme);
  ds.important = state.important;
  for (const candidate of state.ignoredCandidates) {
    ds.invalidCandidates.add(candidate);
  }

  // Custom variants: reserve names in stylesheet order, then register in
  // dependency order.
  const customVariantNames = new Set(state.customVariants.map((v) => v.name));
  for (const variant of state.customVariants) {
    ds.variants.static(variant.name, () => {});
  }
  const dependencies = new Map<string, Set<string>>();
  const byName = new Map<string, (typeof state.customVariants)[number]>();
  for (const variant of state.customVariants) {
    byName.set(variant.name, variant);
    const deps = new Set<string>();
    for (const dep of variant.dependencies) {
      if (customVariantNames.has(dep) && dep !== variant.name) deps.add(dep);
      if (dep === variant.name) {
        throw new TwillError(
          `Custom variant \`${variant.name}\` depends on itself, creating a circular dependency.`,
        );
      }
    }
    dependencies.set(variant.name, deps);
  }
  for (
    const name of topologicalSort(dependencies, (cycle) => {
      throw new TwillError(
        `Custom variant \`${cycle}\` is part of a circular dependency between custom variants.`,
      );
    })
  ) {
    byName.get(name)!.register(ds);
  }

  for (const utility of state.customUtilities) {
    registerCustomUtility(utility, ds);
  }

  // Theme emission (SPEC §7.5).
  if (state.firstThemeRule !== null) {
    const declarations: AstNode[] = [];
    for (const [key, entry] of theme.entries()) {
      if (entry.options & ThemeOptions.REFERENCE) continue;
      declarations.push(decl(escape(theme.prefixKey(key)), entry.value));
    }
    state.firstThemeRule.nodes = [context({ theme: true }, declarations)];
  }
  for (const keyframes of theme.getKeyframes()) {
    ast.push(context({ theme: true }, [atRoot([keyframes])]));
  }

  features |= substituteAtVariant(ast, ds);
  features |= substituteFunctions(ast, ds);
  features |= substituteAtApply(ast, ds);

  // The `@twill utilities` node becomes an empty context that `build` fills.
  let utilitiesNode: Context | null = null;
  if (state.utilitiesNode !== null) {
    const node = state.utilitiesNode as unknown as Record<string, unknown>;
    delete node.name;
    delete node.params;
    node.kind = "context";
    node.context = {};
    node.nodes = [];
    utilitiesNode = state.utilitiesNode as unknown as Context;
  }

  walk(ast, (node, { replaceWith }) => {
    if (node.kind === "at-rule" && node.name === "@utility") {
      replaceWith([]);
      return WalkAction.Skip;
    }
  });

  // Build state (SPEC §15.2).
  const validCandidates = new Set<string>(state.inlineCandidates);
  let pendingInline = state.inlineCandidates.length > 0;
  let cached: string | null = null;
  let previousCount = -1;

  const build = (candidates: Iterable<string>): string => {
    if (features === Features.NONE) return css;
    if (utilitiesNode === null) {
      cached ??= serialize(optimizeAst(ast, ds));
      return cached;
    }

    let changed = pendingInline;
    pendingInline = false;
    let markedVariable = false;
    for (const candidate of candidates) {
      if (ds.invalidCandidates.has(candidate)) continue;
      if (candidate.startsWith("--")) {
        if (ds.theme.markUsedVariable(candidate)) {
          changed = true;
          markedVariable = true;
        }
        continue;
      }
      if (!validCandidates.has(candidate)) {
        validCandidates.add(candidate);
        changed = true;
      }
    }
    if (!changed && cached !== null) return cached;

    const { nodes } = compileCandidates(validCandidates, ds, {
      onInvalidCandidate(candidate) {
        ds.invalidCandidates.add(candidate);
        validCandidates.delete(candidate);
      },
    });

    if (nodes.length === previousCount && !markedVariable && cached !== null) {
      return cached;
    }
    previousCount = nodes.length;
    utilitiesNode.nodes = nodes;
    cached = serialize(optimizeAst(ast, ds));
    return cached;
  };

  return {
    sources: state.sources,
    root: state.root,
    features,
    designSystem: ds,
    build,
  };
}
