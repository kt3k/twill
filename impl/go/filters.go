package twill

import "strings"

// Filter and backdrop-filter utilities (SPEC §10.8, "Filters").

var filterNames = []string{"blur", "brightness", "contrast", "grayscale", "hue-rotate", "invert", "saturate", "sepia", "drop-shadow"}
var backdropNames = []string{"blur", "brightness", "contrast", "grayscale", "hue-rotate", "invert", "opacity", "saturate", "sepia"}

func composedFilter(prefix string, names []string) string {
	parts := make([]string, len(names))
	for i, n := range names {
		parts[i] = "var(" + prefix + n + ",)"
	}
	return strings.Join(parts, " ")
}

// Filter is the composed filter value.
var Filter = composedFilter("--tw-", filterNames)

// BackdropFilter is the composed backdrop-filter value.
var BackdropFilter = composedFilter("--tw-backdrop-", backdropNames)

func filterRegistrations(backdrop bool) []Node {
	prefix, names := "--tw-", filterNames
	if backdrop {
		prefix, names = "--tw-backdrop-", backdropNames
	}
	nodes := make([]Node, 0, len(names))
	for _, n := range names {
		nodes = append(nodes, Property(prefix+n, nil))
	}
	return nodes
}

func filterDeclarations(backdrop bool) []Node {
	if backdrop {
		return []Node{Decl("-webkit-backdrop-filter", BackdropFilter), Decl("backdrop-filter", BackdropFilter)}
	}
	return []Node{Decl("filter", Filter)}
}

// RegisterFilterUtilities registers the filter and backdrop-filter utilities.
func RegisterFilterUtilities(u *Utilities, theme *Theme) {
	statFn := func(name string, fn func() []Node) { StaticUtilityFn(u, name, fn) }
	fn := func(root string, desc FunctionalUtilityDescription) { FunctionalUtility(u, theme, root, desc) }
	percentage := func(v *CandidateValue) (string, bool) { return v.Value + "%", IsPositiveInteger(v.Value) }
	degrees := func(v *CandidateValue) (string, bool) { return v.Value + "deg", IsPositiveInteger(v.Value) }

	for _, backdrop := range []bool{false, true} {
		backdrop := backdrop
		prefix, variable := "", "--tw-"
		if backdrop {
			prefix, variable = "backdrop-", "--tw-backdrop-"
		}
		keys := func(name string) []string {
			if backdrop {
				return []string{"--backdrop-" + name, "--" + name}
			}
			return []string{"--" + name}
		}
		emit := func(name, value string) []Node {
			return append(append(filterRegistrations(backdrop), Decl(variable+name, value)), filterDeclarations(backdrop)...)
		}
		direct := func(value string) []Node {
			if backdrop {
				return []Node{Decl("-webkit-backdrop-filter", value), Decl("backdrop-filter", value)}
			}
			return []Node{Decl("filter", value)}
		}

		statFn(prefix+"filter-none", func() []Node { return direct("none") })
		statFn(prefix+"filter", func() []Node { return append(filterRegistrations(backdrop), filterDeclarations(backdrop)...) })
		fn(prefix+"filter", FunctionalUtilityDescription{Handle: func(value string, _ *string) []Node { return direct(value) }})

		fn(prefix+"blur", FunctionalUtilityDescription{ThemeKeys: keys("blur"), Handle: func(value string, _ *string) []Node { return emit("blur", "blur("+value+")") }})
		statFn(prefix+"blur-none", func() []Node { return emit("blur", "") })
		for _, name := range []string{"brightness", "contrast", "saturate"} {
			name := name
			fn(prefix+name, FunctionalUtilityDescription{ThemeKeys: keys(name), HandleBareValue: percentage,
				Handle: func(value string, _ *string) []Node { return emit(name, name+"("+value+")") }})
		}
		for _, name := range []string{"grayscale", "invert", "sepia"} {
			name := name
			fn(prefix+name, FunctionalUtilityDescription{ThemeKeys: keys(name), DefaultValue: strPtr("100%"), HandleBareValue: percentage,
				Handle: func(value string, _ *string) []Node { return emit(name, name+"("+value+")") }})
		}
		fn(prefix+"hue-rotate", FunctionalUtilityDescription{ThemeKeys: keys("hue-rotate"), SupportsNegative: true, HandleBareValue: degrees,
			Handle: func(value string, _ *string) []Node { return emit("hue-rotate", "hue-rotate("+value+")") }})
		if backdrop {
			fn("backdrop-opacity", FunctionalUtilityDescription{ThemeKeys: []string{"--backdrop-opacity", "--opacity"}, HandleBareValue: percentage,
				Handle: func(value string, _ *string) []Node { return emit("opacity", "opacity("+value+")") }})
			continue
		}

		statFn("drop-shadow-none", func() []Node { return emit("drop-shadow", "") })
		u.Functional("drop-shadow", func(c *Candidate) ([]Node, CompileStatus) {
			color := func(value string) ([]Node, CompileStatus) {
				return []Node{Property("--tw-drop-shadow-color", nil), Decl("--tw-drop-shadow-color", value)}, Handled
			}
			wrap := func(value string) ([]Node, CompileStatus) {
				failed := false
				replaced := ReplaceShadowColors(value, func(col string) string {
					resolved, ok := AsColor(col, c.Modifier, theme)
					if !ok {
						failed = true
						return col
					}
					return "var(--tw-drop-shadow-color, " + resolved + ")"
				}, false)
				if failed {
					return nil, NotHandled
				}
				var shadows []string
				for _, s := range Segment(replaced, ',') {
					shadows = append(shadows, "drop-shadow("+strings.TrimSpace(s)+")")
				}
				return emit("drop-shadow", strings.Join(shadows, " ")), Handled
			}
			if c.Value == nil {
				value, ok := theme.ResolveValue(nil, []string{"--drop-shadow"})
				if !ok {
					return nil, NotHandled
				}
				return wrap(value)
			}
			if c.Value.Kind == ValueArbitrary {
				value := c.Value.Value
				t := ""
				if c.Value.DataType != nil {
					t = *c.Value.DataType
				} else {
					t = InferDataType(value, []string{"color"})
				}
				if t == "color" {
					resolved, ok := AsColor(value, c.Modifier, theme)
					if !ok {
						return nil, NotHandled
					}
					return color(resolved)
				}
				return wrap(value)
			}
			if themeColor, ok := ResolveThemeColor(c, []string{"--drop-shadow-color", "--color"}, theme); ok {
				return color(themeColor)
			}
			value, ok := theme.ResolveValue(strPtr(c.Value.Value), []string{"--drop-shadow"})
			if !ok {
				return nil, NotHandled
			}
			return wrap(value)
		}, nil)
	}
}
