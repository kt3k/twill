/**
 * Data type inference for arbitrary values (SPEC §10.7).
 *
 * @module
 */

import { segment } from "./utils.ts";

export type DataType =
  | "color"
  | "length"
  | "percentage"
  | "number"
  | "integer"
  | "ratio"
  | "url"
  | "image"
  | "position"
  | "bg-size"
  | "line-width"
  | "absolute-size"
  | "relative-size"
  | "family-name"
  | "generic-name"
  | "angle"
  | "vector";

const NAMED_COLORS = new Set([
  "aliceblue",
  "antiquewhite",
  "aqua",
  "aquamarine",
  "azure",
  "beige",
  "bisque",
  "black",
  "blanchedalmond",
  "blue",
  "blueviolet",
  "brown",
  "burlywood",
  "cadetblue",
  "chartreuse",
  "chocolate",
  "coral",
  "cornflowerblue",
  "cornsilk",
  "crimson",
  "cyan",
  "darkblue",
  "darkcyan",
  "darkgoldenrod",
  "darkgray",
  "darkgreen",
  "darkgrey",
  "darkkhaki",
  "darkmagenta",
  "darkolivegreen",
  "darkorange",
  "darkorchid",
  "darkred",
  "darksalmon",
  "darkseagreen",
  "darkslateblue",
  "darkslategray",
  "darkslategrey",
  "darkturquoise",
  "darkviolet",
  "deeppink",
  "deepskyblue",
  "dimgray",
  "dimgrey",
  "dodgerblue",
  "firebrick",
  "floralwhite",
  "forestgreen",
  "fuchsia",
  "gainsboro",
  "ghostwhite",
  "gold",
  "goldenrod",
  "gray",
  "green",
  "greenyellow",
  "grey",
  "honeydew",
  "hotpink",
  "indianred",
  "indigo",
  "ivory",
  "khaki",
  "lavender",
  "lavenderblush",
  "lawngreen",
  "lemonchiffon",
  "lightblue",
  "lightcoral",
  "lightcyan",
  "lightgoldenrodyellow",
  "lightgray",
  "lightgreen",
  "lightgrey",
  "lightpink",
  "lightsalmon",
  "lightseagreen",
  "lightskyblue",
  "lightslategray",
  "lightslategrey",
  "lightsteelblue",
  "lightyellow",
  "lime",
  "limegreen",
  "linen",
  "magenta",
  "maroon",
  "mediumaquamarine",
  "mediumblue",
  "mediumorchid",
  "mediumpurple",
  "mediumseagreen",
  "mediumslateblue",
  "mediumspringgreen",
  "mediumturquoise",
  "mediumvioletred",
  "midnightblue",
  "mintcream",
  "mistyrose",
  "moccasin",
  "navajowhite",
  "navy",
  "oldlace",
  "olive",
  "olivedrab",
  "orange",
  "orangered",
  "orchid",
  "palegoldenrod",
  "palegreen",
  "paleturquoise",
  "palevioletred",
  "papayawhip",
  "peachpuff",
  "peru",
  "pink",
  "plum",
  "powderblue",
  "purple",
  "rebeccapurple",
  "red",
  "rosybrown",
  "royalblue",
  "saddlebrown",
  "salmon",
  "sandybrown",
  "seagreen",
  "seashell",
  "sienna",
  "silver",
  "skyblue",
  "slateblue",
  "slategray",
  "slategrey",
  "snow",
  "springgreen",
  "steelblue",
  "tan",
  "teal",
  "thistle",
  "tomato",
  "turquoise",
  "violet",
  "wheat",
  "white",
  "whitesmoke",
  "yellow",
  "yellowgreen",
  "transparent",
  "currentcolor",
]);

const COLOR_FUNCTIONS =
  /^(rgba?|hsla?|hwb|oklch|oklab|lab|lch|color|color-mix|light-dark)\(/;
const HEX_COLOR =
  /^#([0-9a-fA-F]{3}|[0-9a-fA-F]{4}|[0-9a-fA-F]{6}|[0-9a-fA-F]{8})$/;

export function isColor(value: string): boolean {
  const lower = value.toLowerCase();
  return NAMED_COLORS.has(lower) || HEX_COLOR.test(value) ||
    COLOR_FUNCTIONS.test(lower);
}

const LENGTH_UNITS = [
  "cm",
  "mm",
  "Q",
  "in",
  "pt",
  "pc",
  "px",
  "em",
  "rem",
  "ex",
  "rex",
  "cap",
  "rcap",
  "ch",
  "rch",
  "ic",
  "ric",
  "lh",
  "rlh",
  "vw",
  "svw",
  "lvw",
  "dvw",
  "vh",
  "svh",
  "lvh",
  "dvh",
  "vi",
  "svi",
  "lvi",
  "dvi",
  "vb",
  "svb",
  "lvb",
  "dvb",
  "vmin",
  "svmin",
  "lvmin",
  "dvmin",
  "vmax",
  "svmax",
  "lvmax",
  "dvmax",
  "cqw",
  "cqh",
  "cqi",
  "cqb",
  "cqmin",
  "cqmax",
];

const NUMBER = /^[+-]?(\d+\.?\d*|\.\d+)(e[+-]?\d+)?$/i;
const LENGTH = new RegExp(
  `^[+-]?(\\d+\\.?\\d*|\\.\\d+)(e[+-]?\\d+)?(${LENGTH_UNITS.join("|")})$`,
);
const MATH_FUNCTION =
  /^(calc|min|max|clamp|round|mod|rem|sin|cos|tan|asin|acos|atan|atan2|pow|sqrt|hypot|log|exp|abs|sign)\(/;

export function isLength(value: string): boolean {
  return value === "0" || LENGTH.test(value) || MATH_FUNCTION.test(value) ||
    value.startsWith("--spacing(");
}

export function isPercentage(value: string): boolean {
  return /^[+-]?(\d+\.?\d*|\.\d+)%$/.test(value) || MATH_FUNCTION.test(value);
}

export function isNumber(value: string): boolean {
  return NUMBER.test(value);
}

export function isInteger(value: string): boolean {
  return /^\d+$/.test(value);
}

export function isRatio(value: string): boolean {
  return /^(\d+\.?\d*|\.\d+)\s*\/\s*(\d+\.?\d*|\.\d+)$/.test(value);
}

export function isUrl(value: string): boolean {
  return value.startsWith("url(");
}

export function isImage(value: string): boolean {
  return /^(image|image-set|cross-fade|element)\(/.test(value) ||
    /^[a-z-]*gradient\(/.test(value);
}

const POSITION_KEYWORDS = new Set(["top", "right", "bottom", "left", "center"]);

export function isPosition(value: string): boolean {
  const parts = value.split(/\s+/).filter((p) => p !== "");
  if (parts.length === 0 || parts.length > 4) return false;
  return parts.every((p) =>
    POSITION_KEYWORDS.has(p) || isLength(p) || isPercentage(p)
  );
}

export function isBgSize(value: string): boolean {
  if (value === "cover" || value === "contain" || value === "auto") return true;
  const parts = value.split(/\s+/).filter((p) => p !== "");
  if (parts.length === 0 || parts.length > 2) return false;
  return parts.every((p) => p === "auto" || isLength(p) || isPercentage(p));
}

export function isLineWidth(value: string): boolean {
  return value === "thin" || value === "medium" || value === "thick" ||
    isLength(value);
}

const ABSOLUTE_SIZES = new Set([
  "xx-small",
  "x-small",
  "small",
  "medium",
  "large",
  "x-large",
  "xx-large",
  "xxx-large",
]);

export function isAbsoluteSize(value: string): boolean {
  return ABSOLUTE_SIZES.has(value);
}

export function isRelativeSize(value: string): boolean {
  return value === "larger" || value === "smaller";
}

const GENERIC_NAMES = new Set([
  "serif",
  "sans-serif",
  "monospace",
  "cursive",
  "fantasy",
  "system-ui",
  "ui-serif",
  "ui-sans-serif",
  "ui-monospace",
  "ui-rounded",
  "math",
  "emoji",
  "fangsong",
]);

export function isGenericName(value: string): boolean {
  return GENERIC_NAMES.has(value);
}

export function isFamilyName(value: string): boolean {
  const parts = segment(value, ",").map((p) => p.trim());
  if (parts.length === 0) return false;
  for (const part of parts) {
    if (part === "") return false;
    if (
      (part.startsWith('"') && part.endsWith('"')) ||
      (part.startsWith("'") && part.endsWith("'"))
    ) {
      continue;
    }
    if (!/^[a-zA-Z_][\w -]*$/.test(part)) return false;
  }
  return true;
}

export function isAngle(value: string): boolean {
  return /^[+-]?(\d+\.?\d*|\.\d+)(deg|rad|grad|turn)$/.test(value);
}

export function isVector(value: string): boolean {
  const parts = value.split(/\s+/).filter((p) => p !== "");
  return parts.length === 3 && parts.every(isNumber);
}

const CHECKS: Record<DataType, (value: string) => boolean> = {
  color: isColor,
  length: isLength,
  percentage: isPercentage,
  number: isNumber,
  integer: isInteger,
  ratio: isRatio,
  url: isUrl,
  image: isImage,
  position: isPosition,
  "bg-size": isBgSize,
  "line-width": isLineWidth,
  "absolute-size": isAbsoluteSize,
  "relative-size": isRelativeSize,
  "family-name": isFamilyName,
  "generic-name": isGenericName,
  angle: isAngle,
  vector: isVector,
};

export function isDataType(type: string): type is DataType {
  return type in CHECKS;
}

/**
 * Returns the first type in `types` whose predicate accepts `value`, or
 * null. A value starting with `var(` never matches.
 */
export function inferDataType(
  value: string,
  types: DataType[],
): DataType | null {
  if (value.startsWith("var(")) return null;
  for (const type of types) {
    if (CHECKS[type](value)) return type;
  }
  return null;
}
