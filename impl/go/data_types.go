package twill

import (
	"regexp"
	"strings"
)

var namedColors = map[string]bool{}

func init() {
	for _, c := range strings.Fields(`aliceblue antiquewhite aqua aquamarine azure beige bisque black
blanchedalmond blue blueviolet brown burlywood cadetblue chartreuse chocolate coral cornflowerblue
cornsilk crimson cyan darkblue darkcyan darkgoldenrod darkgray darkgreen darkgrey darkkhaki
darkmagenta darkolivegreen darkorange darkorchid darkred darksalmon darkseagreen darkslateblue
darkslategray darkslategrey darkturquoise darkviolet deeppink deepskyblue dimgray dimgrey dodgerblue
firebrick floralwhite forestgreen fuchsia gainsboro ghostwhite gold goldenrod gray green greenyellow
grey honeydew hotpink indianred indigo ivory khaki lavender lavenderblush lawngreen lemonchiffon
lightblue lightcoral lightcyan lightgoldenrodyellow lightgray lightgreen lightgrey lightpink
lightsalmon lightseagreen lightskyblue lightslategray lightslategrey lightsteelblue lightyellow lime
limegreen linen magenta maroon mediumaquamarine mediumblue mediumorchid mediumpurple mediumseagreen
mediumslateblue mediumspringgreen mediumturquoise mediumvioletred midnightblue mintcream mistyrose
moccasin navajowhite navy oldlace olive olivedrab orange orangered orchid palegoldenrod palegreen
paleturquoise palevioletred papayawhip peachpuff peru pink plum powderblue purple rebeccapurple red
rosybrown royalblue saddlebrown salmon sandybrown seagreen seashell sienna silver skyblue slateblue
slategray slategrey snow springgreen steelblue tan teal thistle tomato turquoise violet wheat white
whitesmoke yellow yellowgreen transparent currentcolor`) {
		namedColors[c] = true
	}
}

var (
	colorFunctionPattern = regexp.MustCompile(`^(rgba?|hsla?|hwb|oklch|oklab|lab|lch|color|color-mix|light-dark)\(`)
	hexColorPattern      = regexp.MustCompile(`^#([0-9a-fA-F]{3}|[0-9a-fA-F]{4}|[0-9a-fA-F]{6}|[0-9a-fA-F]{8})$`)
	lengthPattern        = regexp.MustCompile(`^[+-]?(\d+\.?\d*|\.\d+)([eE][+-]?\d+)?(cm|mm|Q|in|pt|pc|px|em|rem|ex|rex|cap|rcap|ch|rch|ic|ric|lh|rlh|vw|svw|lvw|dvw|vh|svh|lvh|dvh|vi|svi|lvi|dvi|vb|svb|lvb|dvb|vmin|svmin|lvmin|dvmin|vmax|svmax|lvmax|dvmax|cqw|cqh|cqi|cqb|cqmin|cqmax)$`)
	mathFunctionPattern  = regexp.MustCompile(`^(calc|min|max|clamp|round|mod|rem|sin|cos|tan|asin|acos|atan|atan2|pow|sqrt|hypot|log|exp|abs|sign)\(`)
	percentagePattern    = regexp.MustCompile(`^[+-]?(\d+\.?\d*|\.\d+)%$`)
	integerPattern       = regexp.MustCompile(`^\d+$`)
	ratioPattern         = regexp.MustCompile(`^(\d+\.?\d*|\.\d+)\s*/\s*(\d+\.?\d*|\.\d+)$`)
	imagePattern         = regexp.MustCompile(`^(image|image-set|cross-fade|element)\(`)
	gradientPattern      = regexp.MustCompile(`^[a-z-]*gradient\(`)
	familyNamePattern    = regexp.MustCompile(`^[a-zA-Z_][\w -]*$`)
	anglePattern         = regexp.MustCompile(`^[+-]?(\d+\.?\d*|\.\d+)(deg|rad|grad|turn)$`)
)

var positionKeywords = map[string]bool{"top": true, "right": true, "bottom": true, "left": true, "center": true}
var absoluteSizes = map[string]bool{"xx-small": true, "x-small": true, "small": true, "medium": true, "large": true, "x-large": true, "xx-large": true, "xxx-large": true}
var genericNames = map[string]bool{"serif": true, "sans-serif": true, "monospace": true, "cursive": true, "fantasy": true, "system-ui": true, "ui-serif": true, "ui-sans-serif": true, "ui-monospace": true, "ui-rounded": true, "math": true, "emoji": true, "fangsong": true}

func isColor(v string) bool {
	lower := strings.ToLower(v)
	return namedColors[lower] || hexColorPattern.MatchString(v) || colorFunctionPattern.MatchString(lower)
}

func isLength(v string) bool {
	return v == "0" || lengthPattern.MatchString(v) || mathFunctionPattern.MatchString(v) || strings.HasPrefix(v, "--spacing(")
}

func isPercentage(v string) bool {
	return percentagePattern.MatchString(v) || mathFunctionPattern.MatchString(v)
}
func isNumber(v string) bool  { return numberPattern.MatchString(v) }
func isInteger(v string) bool { return integerPattern.MatchString(v) }
func isRatio(v string) bool   { return ratioPattern.MatchString(v) }
func isURL(v string) bool     { return strings.HasPrefix(v, "url(") }
func isImage(v string) bool   { return imagePattern.MatchString(v) || gradientPattern.MatchString(v) }

func isPosition(v string) bool {
	parts := strings.Fields(v)
	if len(parts) == 0 || len(parts) > 4 {
		return false
	}
	for _, p := range parts {
		if !positionKeywords[p] && !isLength(p) && !isPercentage(p) {
			return false
		}
	}
	return true
}

func isBgSize(v string) bool {
	if v == "cover" || v == "contain" || v == "auto" {
		return true
	}
	parts := strings.Fields(v)
	if len(parts) == 0 || len(parts) > 2 {
		return false
	}
	for _, p := range parts {
		if p != "auto" && !isLength(p) && !isPercentage(p) {
			return false
		}
	}
	return true
}

func isLineWidth(v string) bool { return v == "thin" || v == "medium" || v == "thick" || isLength(v) }

func isFamilyName(v string) bool {
	parts := Segment(v, ',')
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			return false
		}
		if (strings.HasPrefix(part, `"`) && strings.HasSuffix(part, `"`)) || (strings.HasPrefix(part, "'") && strings.HasSuffix(part, "'")) {
			continue
		}
		if !familyNamePattern.MatchString(part) {
			return false
		}
	}
	return len(parts) > 0
}

func isVector(v string) bool {
	parts := strings.Fields(v)
	if len(parts) != 3 {
		return false
	}
	for _, p := range parts {
		if !isNumber(p) {
			return false
		}
	}
	return true
}

var dataTypeChecks = map[string]func(string) bool{
	"color": isColor, "length": isLength, "percentage": isPercentage, "number": isNumber,
	"integer": isInteger, "ratio": isRatio, "url": isURL, "image": isImage, "position": isPosition,
	"bg-size": isBgSize, "line-width": isLineWidth,
	"absolute-size": func(v string) bool { return absoluteSizes[v] },
	"relative-size": func(v string) bool { return v == "larger" || v == "smaller" },
	"family-name":   isFamilyName,
	"generic-name":  func(v string) bool { return genericNames[v] },
	"angle":         func(v string) bool { return anglePattern.MatchString(v) },
	"vector":        isVector,
}

// IsDataType reports whether name is a known data type.
func IsDataType(name string) bool { _, ok := dataTypeChecks[name]; return ok }

// InferDataType returns the first type in types whose predicate accepts
// value, or "" (SPEC §10.7). A value starting with var( never matches.
func InferDataType(value string, types []string) string {
	if strings.HasPrefix(value, "var(") {
		return ""
	}
	for _, t := range types {
		if check, ok := dataTypeChecks[t]; ok && check(value) {
			return t
		}
	}
	return ""
}
