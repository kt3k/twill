/**
 * Directive collection (SPEC §6.1 step 2, §6.4 through §6.6, and the
 * media-position import parameters of §6.2).
 *
 * @module
 */

import {
  type AstNode,
  type AtRule,
  context,
  type ParentNode,
  type Rule,
  styleRule,
  walk,
  WalkAction,
} from "./ast.ts";
import { type CustomUtility, parseUtilityDefinition } from "./at_utility.ts";
import {
  type CustomVariant,
  isCustomVariantCompat,
  parseCustomVariant,
} from "./custom_variant.ts";
import { TwillError } from "./error.ts";
import { Features } from "./features.ts";
import { serialize } from "./serializer.ts";
import { type Theme, ThemeOptions } from "./theme.ts";
import {
  expandBraces,
  PREFIX_PATTERN,
  segment,
  unescape,
  unquote,
} from "./utils.ts";

/** A source entry collected from `@source` (SPEC §4.1.10). */
export interface SourceEntry {
  base: string;
  pattern: string;
  negated: boolean;
}

/** The `root` of a compiler handle (SPEC §4.1.11). */
export type SourceRoot = null | "none" | { base: string; pattern: string };

export interface CollectedDirectives {
  features: number;
  /** The design-system-wide `important` flag. */
  important: boolean;
  sources: SourceEntry[];
  root: SourceRoot;
  /** Candidates from `@source inline(...)`. */
  inlineCandidates: string[];
  /** Candidates from `@source not inline(...)`. */
  ignoredCandidates: string[];
  /** The kept `@twill utilities` node, or null. */
  utilitiesNode: AtRule | null;
  /** The `:root, :host` rule that replaced the first `@theme`, or null. */
  firstThemeRule: Rule | null;
  /** Custom variants in stylesheet order. */
  customVariants: CustomVariant[];
  /** Custom utilities in stylesheet order. */
  customUtilities: CustomUtility[];
}

/**
 * Walks the AST once and registers `@theme`, `@source`, and
 * `@twill utilities`, interpreting the media-position import parameters.
 * Directives that must not appear in the output are removed.
 */
export function collectDirectives(
  ast: AstNode[],
  theme: Theme,
): CollectedDirectives {
  const state: CollectedDirectives = {
    features: Features.NONE,
    important: false,
    sources: [],
    root: null,
    inlineCandidates: [],
    ignoredCandidates: [],
    utilitiesNode: null,
    firstThemeRule: null,
    customVariants: [],
    customUtilities: [],
  };

  const assertTopLevel = (node: AtRule, path: ParentNode[]) => {
    for (const ancestor of path) {
      if (ancestor.kind === "rule" || ancestor.kind === "at-rule") {
        throw new TwillError(
          `\`${node.name} ${node.params}\` cannot be nested.`,
        );
      }
    }
  };

  walk(ast, (node, { context: ctx, path, replaceWith }) => {
    if (node.kind !== "at-rule") return;

    if (
      node.name === "@custom-variant" ||
      (node.name === "@variant" && isTopLevel(path) &&
        isCustomVariantCompat(node))
    ) {
      assertTopLevel(node, path);
      state.customVariants.push(parseCustomVariant(node));
      replaceWith([]);
      return;
    }

    if (node.name === "@utility") {
      assertTopLevel(node, path);
      state.customUtilities.push(parseUtilityDefinition(node));
      return WalkAction.Skip;
    }

    if (node.name === "@media") {
      return handleImportMedia(node, ctx, state, replaceWith);
    }

    if (node.name === "@twill") {
      if (!node.params.startsWith("utilities")) return;
      if (ctx.reference === true || state.utilitiesNode !== null) {
        replaceWith([]);
        return;
      }
      const source = parseUtilitiesSource(node.params);
      if (source !== null) {
        if (source === "none") {
          state.root = "none";
        } else {
          const base = typeof ctx.sourceBase === "string"
            ? ctx.sourceBase
            : String(ctx.base ?? "");
          state.root = { base, pattern: source };
        }
      }
      state.utilitiesNode = node;
      state.features |= Features.UTILITIES;
      return WalkAction.Skip;
    }

    if (node.name === "@theme") {
      const options = parseThemeOptions(
        node.params,
        ctx.reference === true,
        theme,
      );
      for (const child of node.nodes) {
        if (child.kind === "comment") continue;
        if (child.kind === "declaration" && child.property.startsWith("--")) {
          theme.add(unescape(child.property), child.value ?? "", options);
          continue;
        }
        if (child.kind === "at-rule" && child.name === "@keyframes") {
          if (options & ThemeOptions.REFERENCE) continue;
          theme.addKeyframes(child);
          continue;
        }
        const snippet = serialize([child]).split("\n").slice(0, 3).join("\n");
        throw new TwillError(
          `\`@theme\` blocks must only contain custom properties or \`@keyframes\`.\n\n${snippet}`,
        );
      }
      state.features |= Features.AT_THEME;
      if (
        state.firstThemeRule === null && !(options & ThemeOptions.REFERENCE)
      ) {
        state.firstThemeRule = styleRule(":root, :host", []);
        replaceWith(state.firstThemeRule);
      } else {
        replaceWith([]);
      }
      return WalkAction.Skip;
    }

    if (node.name === "@source") {
      if (node.nodes.length > 0) {
        throw new TwillError("`@source` cannot have a body.");
      }
      for (const ancestor of path) {
        if (ancestor.kind === "rule" || ancestor.kind === "at-rule") {
          throw new TwillError("`@source` cannot be nested.");
        }
      }
      handleSource(node, String(ctx.base ?? ""), state);
      replaceWith([]);
      return;
    }
  });

  return state;
}

function isTopLevel(path: ParentNode[]): boolean {
  return path.every((p) => p.kind === "context");
}

function parseUtilitiesSource(params: string): string | null {
  const rest = params.slice("utilities".length).trim();
  if (rest === "") return null;
  if (!rest.startsWith("source(") || !rest.endsWith(")")) {
    throw new TwillError(`Invalid \`@twill ${params}\``);
  }
  const inner = rest.slice("source(".length, -1).trim();
  if (inner === "none") return "none";
  const path = unquote(inner);
  if (path === null) {
    throw new TwillError(
      `\`source(${inner})\` paths must be quoted.\n\nInstead use:\n@twill utilities source("${inner}");`,
    );
  }
  return path;
}

function parseThemeOptions(
  params: string,
  inReference: boolean,
  theme: Theme,
): number {
  let options = ThemeOptions.NONE;
  if (inReference) options |= ThemeOptions.REFERENCE;
  for (const option of segment(params, " ")) {
    if (option === "") continue;
    if (option === "reference") {
      options |= ThemeOptions.REFERENCE;
    } else if (option === "inline") {
      options |= ThemeOptions.INLINE;
    } else if (option === "default") {
      options |= ThemeOptions.DEFAULT;
    } else if (option === "static") {
      options |= ThemeOptions.STATIC;
    } else if (option.startsWith("prefix(") && option.endsWith(")")) {
      const prefix = option.slice("prefix(".length, -1);
      if (!PREFIX_PATTERN.test(prefix)) {
        throw new TwillError(
          `The prefix "${prefix}" is invalid. Prefixes must be alphabetic and lowercase.`,
        );
      }
      theme.prefix = prefix;
    } else {
      throw new TwillError(`Unknown \`@theme\` option \`${option}\``);
    }
  }
  return options;
}

function handleSource(
  node: AtRule,
  base: string,
  state: CollectedDirectives,
): void {
  let params = node.params.trim();
  let negated = false;
  if (params.startsWith("not ")) {
    negated = true;
    params = params.slice(4).trim();
  }

  if (params.startsWith("inline(")) {
    if (!params.endsWith(")")) {
      throw new TwillError(`Invalid \`@source ${node.params}\``);
    }
    const inner = unquote(params.slice("inline(".length, -1).trim());
    if (inner === null) {
      throw new TwillError(`\`@source inline(...)\` patterns must be quoted.`);
    }
    const target = negated ? state.ignoredCandidates : state.inlineCandidates;
    for (const item of inner.split(/\s+/)) {
      if (item === "") continue;
      target.push(...expandBraces(item));
    }
    return;
  }

  const pattern = unquote(params);
  if (pattern === null) {
    throw new TwillError(
      `\`@source\` paths must be quoted.\n\nInstead use:\n@source ${
        negated ? "not " : ""
      }"${params}";`,
    );
  }
  state.sources.push({ base, pattern, negated });
}

/**
 * Interprets the media-position import parameters (`reference`,
 * `theme(...)`, `prefix(...)`, `important`, `source(...)`) on an `@media`
 * produced by the import resolver.
 */
function handleImportMedia(
  node: AtRule,
  ctx: Record<string, string | boolean>,
  state: CollectedDirectives,
  replaceWith: (nodes: AstNode | AstNode[]) => void,
): WalkAction | undefined {
  const params = segment(node.params, " ").filter((p) => p !== "");
  const remaining: string[] = [];
  let consumed = false;

  for (const param of params) {
    if (param === "reference") {
      node.nodes = [context({ reference: true }, node.nodes)];
      consumed = true;
    } else if (param.startsWith("theme(") && param.endsWith(")")) {
      const opts = param.slice("theme(".length, -1).trim();
      const isReference = segment(opts, " ").includes("reference");
      walk(node.nodes, (child) => {
        if (child.kind === "at-rule" && child.name === "@theme") {
          child.params = `${child.params} ${opts}`.trim();
          return WalkAction.Skip;
        }
        if (child.kind === "context" || child.kind === "comment") return;
        if (
          child.kind === "at-rule" &&
          (child.name === "@layer" || child.name === "@media")
        ) {
          return;
        }
        if (isReference) {
          throw new TwillError(
            "Importing a stylesheet with `theme(reference)` is only allowed for stylesheets that contain only `@theme` blocks.",
          );
        }
        return WalkAction.Skip;
      });
      consumed = true;
    } else if (param.startsWith("prefix(") && param.endsWith(")")) {
      walk(node.nodes, (child) => {
        if (child.kind === "at-rule" && child.name === "@theme") {
          child.params = `${child.params} ${param}`.trim();
          return WalkAction.Skip;
        }
      });
      consumed = true;
    } else if (param === "important") {
      state.important = true;
      consumed = true;
    } else if (param.startsWith("source(") && param.endsWith(")")) {
      const sourceBase = String(ctx.base ?? "");
      walk(node.nodes, (child, { replaceWith: replaceChild }) => {
        if (
          child.kind === "at-rule" && child.name === "@twill" &&
          child.params === "utilities"
        ) {
          child.params = `utilities ${param}`;
          replaceChild(context({ sourceBase }, [child]));
          return WalkAction.Stop;
        }
      });
      consumed = true;
    } else {
      remaining.push(param);
    }
  }

  if (!consumed) return;
  if (remaining.length === 0) {
    replaceWith(node.nodes);
  } else {
    node.params = remaining.join(" ");
  }
}
