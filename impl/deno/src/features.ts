/** Feature flags reported by `compile` (SPEC §4.1.11). */
export const Features = {
  NONE: 0,
  AT_APPLY: 1 << 0,
  AT_IMPORT: 1 << 1,
  THEME_FUNCTION: 1 << 3,
  UTILITIES: 1 << 4,
  VARIANTS: 1 << 5,
  AT_THEME: 1 << 6,
} as const;

export type Features = number;
