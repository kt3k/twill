/**
 * Color and opacity helpers (SPEC §10.3).
 *
 * @module
 */

/**
 * Applies an alpha value to a color. A numeric alpha is converted to a
 * percentage; `100%` returns the color unchanged; anything else produces a
 * `color-mix(...)`.
 */
export function withAlpha(color: string, alpha: string): string {
  if (alpha === "") return color;
  const n = Number(alpha);
  if (!Number.isNaN(n) && /^[+-]?(\d+\.?\d*|\.\d+)(e[+-]?\d+)?$/i.test(alpha)) {
    alpha = `${n * 100}%`;
  }
  if (alpha === "100%") return color;
  return `color-mix(in oklab, ${color} ${alpha}, transparent)`;
}
