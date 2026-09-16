/**
 * Twill: a utility-first CSS compiler.
 *
 * @module
 */

export * from "./src/ast.ts";
export { TwillError } from "./src/error.ts";
export type { SourcePosition } from "./src/error.ts";
export { parse } from "./src/parser.ts";
export { serialize, serializeCompact } from "./src/serializer.ts";
export { Theme, type ThemeEntry, ThemeOptions } from "./src/theme.ts";
export { Features } from "./src/features.ts";
export {
  BUILTIN_BASE,
  isBuiltinPath,
  type LoadedStylesheet,
  resolveBuiltin,
  type StylesheetLoader,
} from "./src/builtin.ts";
export type { SourceEntry, SourceRoot } from "./src/directives.ts";
export * from "./src/candidate.ts";
export {
  applyVariant,
  Compounds,
  compoundsForSelectors,
  staticVariant,
  type VariantApply,
  type VariantDefinition,
  type VariantRule,
  Variants,
} from "./src/variants.ts";
export {
  asColor,
  colorUtility,
  type CompileResult,
  functionalUtility,
  type FunctionalUtilityDescription,
  propertyRegistration,
  resolveThemeColor,
  spacingUtility,
  staticUtility,
  Utilities,
  type UtilityDefinition,
} from "./src/utilities.ts";
export { type DataType, inferDataType } from "./src/data_types.ts";
export { buildDesignSystem, DesignSystem } from "./src/design_system.ts";
export { compile, type CompileOptions, type Compiler } from "./src/compile.ts";
export {
  compileAstNodes,
  compileCandidates,
  type CompileCandidatesOptions,
  CompileFlags,
} from "./src/compile_candidates.ts";
export { optimizeAst } from "./src/optimize.ts";
export { PROPERTY_ORDER } from "./src/property_order.ts";
export { Scanner } from "./src/scanner.ts";
export { extractCandidates } from "./src/extract.ts";
