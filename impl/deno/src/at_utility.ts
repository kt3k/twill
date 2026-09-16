/**
 * `@utility` definitions (SPEC §6.8), utility name rules (§10.5), and
 * `--value(...)` / `--modifier(...)` resolution (§10.6).
 *
 * @module
 */

import {
  type AstNode,
  type AtRule,
  cloneNodes,
  walk,
  WalkAction,
} from "./ast.ts";
import type { FunctionalCandidate } from "./candidate.ts";
import type { DesignSystem } from "./design_system.ts";
import { inferDataType, isDataType } from "./data_types.ts";
import { TwillError } from "./error.ts";
import {
  isMultipleOfQuarter,
  isPositiveInteger,
  segment,
  unescape,
  unquote,
} from "./utils.ts";
import { parseValue, toCss, walkValue, word } from "./value_parser.ts";

const FUNCTIONAL_PREFIX = /^-?[a-z][a-zA-Z0-9_-]*$/;
const STATIC_ROOT = /^-?[a-z][a-zA-Z0-9_-]*/;

/** Whether `name` is a valid static utility name (SPEC §10.5). */
export function isValidStaticUtilityName(name: string): boolean {
  const match = STATIC_ROOT.exec(name);
  if (match === null) return false;
  const root = match[0];
  const rest = name.slice(root.length);
  if (root.endsWith("-") && rest === "") return false;
  if (rest === "") return true;
  if (!/^[a-zA-Z0-9_./%-]*$/.test(rest)) return false;
  const slashes = rest.split("/").length - 1;
  if (slashes > 1 || rest.endsWith("/")) return false;
  for (let i = 0; i < name.length; i++) {
    const c = name[i];
    if (c === ".") {
      if (
        !/[0-9]/.test(name[i - 1] ?? "") || !/[0-9]/.test(name[i + 1] ?? "")
      ) return false;
    } else if (c === "%") {
      if (i !== name.length - 1 || !/[0-9]/.test(name[i - 1] ?? "")) {
        return false;
      }
    }
  }
  return true;
}

/** Whether `name` is a valid functional utility name (`<root>-*`). */
export function isValidFunctionalUtilityName(name: string): boolean {
  return name.endsWith("-*") && FUNCTIONAL_PREFIX.test(name.slice(0, -2));
}

export interface CustomUtility {
  /** The utility name (for static) or root (for functional). */
  name: string;
  kind: "static" | "functional";
  /** The `@utility` node; its body is read lazily so `@apply` can expand first. */
  node: AtRule;
}

/** Validates an `@utility` name and returns its description. */
export function parseUtilityDefinition(node: AtRule): CustomUtility {
  const name = unescape(node.params.trim());
  if (node.nodes.length === 0) {
    throw new TwillError(
      `\`@utility ${name}\` is empty. Utilities without a body are not supported.`,
    );
  }
  if (isValidFunctionalUtilityName(name)) {
    return { name: name.slice(0, -2), kind: "functional", node };
  }
  if (name.endsWith("*")) {
    throw new TwillError(
      `\`@utility ${name}\` defines an invalid utility name. A functional utility must end in \`-*\`.`,
    );
  }
  if (name.includes("*")) {
    throw new TwillError(
      `\`@utility ${name}\` defines an invalid utility name. The \`*\` must be at the end.`,
    );
  }
  if (isValidStaticUtilityName(name)) {
    return { name, kind: "static", node };
  }
  throw new TwillError(
    `\`@utility ${name}\` defines an invalid utility name. Utilities should be alphabetic and start with a lowercase letter.`,
  );
}

/** Registers a parsed `@utility` on the design system. */
export function registerCustomUtility(
  utility: CustomUtility,
  ds: DesignSystem,
): void {
  if (utility.kind === "static") {
    ds.utilities.static(utility.name, () => cloneNodes(utility.node.nodes));
    return;
  }
  ds.utilities.functional(
    utility.name,
    (candidate) => compileFunctionalBody(candidate, utility.node.nodes, ds),
  );
}

/** Normalizes one `--value(...)` or `--modifier(...)` argument (SPEC §10.6). */
export function normalizeArgument(arg: string): string {
  let a = arg.replaceAll("\\*", "*");
  a = a.replace(/(--[a-zA-Z0-9_-]+?)(?:-\*)?\s+(?=--)/g, "$1-*");
  a = a.replace(/\s+/g, "");
  a = a.replace(/(-\*)+/g, "-*");
  if (/^--[a-zA-Z0-9_-]+$/.test(a) && !a.endsWith("-*")) a += "-*";
  return a;
}

interface ValueLike {
  kind: "named" | "arbitrary";
  value: string;
  fraction: string | null;
  dataType: string | null;
}

interface Resolution {
  value: string;
  ratio: boolean;
}

function resolveArgument(
  arg: string,
  target: ValueLike | null,
  ds: DesignSystem,
): Resolution | null {
  // --default(<v>)
  if (arg.startsWith("--default(") && arg.endsWith(")")) {
    if (target === null) {
      return { value: arg.slice("--default(".length, -1), ratio: false };
    }
    return null;
  }
  if (target === null) return null;

  // Quoted literal.
  const literal = unquote(arg);
  if (literal !== null) {
    if (target.kind === "named" && target.value === literal) {
      return { value: literal, ratio: false };
    }
    return null;
  }

  // Theme namespaces.
  if (arg.startsWith("--")) {
    if (target.kind !== "named") return null;
    const star = arg.indexOf("-*");
    if (star === -1) return null;
    const namespace = arg.slice(0, star);
    const sub = arg.slice(star + 2);
    if (sub === "") {
      let value: string | null = null;
      if (target.fraction !== null) {
        value = ds.theme.resolve(target.fraction, [namespace]);
      }
      if (value === null) value = ds.theme.resolve(target.value, [namespace]);
      return value === null ? null : { value, ratio: false };
    }
    if (!sub.startsWith("--")) return null;
    const resolved = ds.theme.resolveWith(target.value, [namespace], [sub]);
    if (resolved === null) return null;
    const value = resolved[1][sub];
    return value === undefined ? null : { value, ratio: false };
  }

  // Arbitrary value with a type.
  if (arg.startsWith("[") && arg.endsWith("]")) {
    if (target.kind !== "arbitrary") return null;
    const type = arg.slice(1, -1);
    if (type === "*") return { value: target.value, ratio: false };
    if (target.dataType !== null) {
      return target.dataType === type
        ? { value: target.value, ratio: false }
        : null;
    }
    if (!isDataType(type)) return null;
    return inferDataType(target.value, [type]) === type
      ? { value: target.value, ratio: false }
      : null;
  }

  // Bare data types for named values.
  if (target.kind !== "named") return null;
  switch (arg) {
    case "number":
      return isMultipleOfQuarter(target.value)
        ? { value: target.value, ratio: false }
        : null;
    case "integer":
      return isPositiveInteger(target.value)
        ? { value: target.value, ratio: false }
        : null;
    case "percentage":
      return /^\d+%$/.test(target.value)
        ? { value: target.value, ratio: false }
        : null;
    case "ratio": {
      if (target.fraction === null) return null;
      const [a, b] = target.fraction.split("/");
      if (!isPositiveInteger(a) || !isPositiveInteger(b)) return null;
      return { value: `${a} / ${b}`, ratio: true };
    }
    default:
      // Unsupported data type: ignored.
      return null;
  }
}

/** Compiles a functional `@utility` body for a candidate (SPEC §10.6). */
export function compileFunctionalBody(
  candidate: FunctionalCandidate,
  body: AstNode[],
  ds: DesignSystem,
): AstNode[] | undefined {
  const valueTarget: ValueLike | null = candidate.value === null
    ? null
    : candidate.value.kind === "named"
    ? {
      kind: "named",
      value: candidate.value.value,
      fraction: candidate.value.fraction,
      dataType: null,
    }
    : {
      kind: "arbitrary",
      value: candidate.value.value,
      fraction: null,
      dataType: candidate.value.dataType,
    };
  const modifierTarget: ValueLike | null = candidate.modifier === null
    ? null
    : {
      kind: candidate.modifier.kind,
      value: candidate.modifier.value,
      fraction: null,
      dataType: null,
    };

  const nodes = cloneNodes(body);
  let sawValueFn = false;
  let resolvedValue = false;
  let resolvedRatio = false;
  let sawModifierFn = false;
  let resolvedModifier = false;
  // Declarations that resolved a non-ratio `--value`.
  const nonRatioDeclarations = new Set<AstNode>();

  walk(nodes, (node, { replaceWith }) => {
    if (node.kind === "at-root") return WalkAction.Skip;
    if (node.kind !== "declaration" || node.value === undefined) return;
    if (
      !node.value.includes("--value(") && !node.value.includes("--modifier(")
    ) return;

    let failed = false;
    let usedRatio = false;
    let usedNonRatioValue = false;
    const ast = parseValue(node.value);
    walkValue(ast, (valueNode, { replaceWith: replaceValue }) => {
      if (valueNode.kind !== "function") return;
      if (valueNode.value !== "--value" && valueNode.value !== "--modifier") {
        return;
      }
      const isValue = valueNode.value === "--value";
      if (isValue) sawValueFn = true;
      else sawModifierFn = true;
      const target = isValue ? valueTarget : modifierTarget;
      const args = segment(toCss(valueNode.nodes), ",").map((a) =>
        normalizeArgument(a.trim())
      );
      let resolution: Resolution | null = null;
      for (const arg of args) {
        if (arg === "") continue;
        resolution = resolveArgument(arg, target, ds);
        if (resolution !== null) break;
      }
      if (resolution === null) {
        failed = true;
        return true;
      }
      if (isValue) {
        resolvedValue = true;
        if (resolution.ratio) {
          resolvedRatio = true;
          usedRatio = true;
        } else {
          usedNonRatioValue = true;
        }
      } else {
        resolvedModifier = true;
      }
      replaceValue(word(resolution.value));
      return true;
    });

    if (failed) {
      replaceWith([]);
      return;
    }
    node.value = toCss(ast);
    if (usedNonRatioValue && !usedRatio) nonRatioDeclarations.add(node);
  });

  if (!sawValueFn || !resolvedValue) return;
  if (sawModifierFn && !resolvedModifier && candidate.modifier !== null) return;
  if (resolvedRatio && resolvedModifier) return;
  if (candidate.modifier !== null && !resolvedRatio && !resolvedModifier) {
    return;
  }

  if (resolvedRatio && nonRatioDeclarations.size > 0) {
    walk(nodes, (node, { replaceWith }) => {
      if (nonRatioDeclarations.has(node)) replaceWith([]);
    });
  }

  return nodes;
}
