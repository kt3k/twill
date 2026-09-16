package twill

import (
	"sort"
	"strings"
)

// Typography, background, layout, table, scrolling, and interactivity
// extensions (SPEC §10.8).

var blendModes = []string{
	"normal", "multiply", "screen", "overlay", "darken", "lighten", "color-dodge", "color-burn",
	"hard-light", "soft-light", "difference", "exclusion", "hue", "saturation", "color", "luminosity",
}

var numericVariables = []string{"--tw-ordinal", "--tw-slashed-zero", "--tw-numeric-figure", "--tw-numeric-spacing", "--tw-numeric-fraction"}

// RegisterExtraUtilities registers the extension utilities.
func RegisterExtraUtilities(u *Utilities, theme *Theme) {
	stat := func(name string, declarations ...[2]string) { StaticUtility(u, name, Declarations(declarations)) }
	statFn := func(name string, fn func() []Node) { StaticUtilityFn(u, name, fn) }
	fn := func(root string, desc FunctionalUtilityDescription) { FunctionalUtility(u, theme, root, desc) }
	single := func(property string) func(string, *string) []Node {
		return func(value string, _ *string) []Node { return []Node{Decl(property, value)} }
	}
	pixels := func(v *CandidateValue) (string, bool) { return v.Value + "px", IsPositiveInteger(v.Value) }
	arbitraryType := func(c *Candidate, types []string) string {
		if c.Value.DataType != nil {
			return *c.Value.DataType
		}
		return InferDataType(c.Value.Value, types)
	}

	// Typography.
	stat("line-clamp-none", [2]string{"overflow", "visible"}, [2]string{"display", "block"},
		[2]string{"-webkit-box-orient", "horizontal"}, [2]string{"-webkit-line-clamp", "unset"})
	fn("line-clamp", FunctionalUtilityDescription{ThemeKeys: []string{"--line-clamp"}, HandleBareValue: BareInteger,
		Handle: func(value string, _ *string) []Node {
			return []Node{Decl("overflow", "hidden"), Decl("display", "-webkit-box"), Decl("-webkit-box-orient", "vertical"), Decl("-webkit-line-clamp", value)}
		}})

	for _, style := range []string{"solid", "double", "dotted", "dashed", "wavy"} {
		stat("decoration-"+style, [2]string{"text-decoration-style", style})
	}
	stat("decoration-from-font", [2]string{"text-decoration-thickness", "from-font"})
	stat("decoration-auto", [2]string{"text-decoration-thickness", "auto"})
	u.Functional("decoration", func(c *Candidate) ([]Node, CompileStatus) {
		if c.Value == nil {
			return nil, NotHandled
		}
		if c.Value.Kind == ValueArbitrary {
			value := c.Value.Value
			switch arbitraryType(c, []string{"color", "length", "percentage"}) {
			case "length", "percentage":
				if c.Modifier != nil {
					return nil, NotHandled
				}
				return []Node{Decl("text-decoration-thickness", value)}, Handled
			}
			resolved, ok := AsColor(value, c.Modifier, theme)
			if !ok {
				return nil, NotHandled
			}
			return []Node{Decl("text-decoration-color", resolved)}, Handled
		}
		if color, ok := ResolveThemeColor(c, []string{"--text-decoration-color", "--color"}, theme); ok {
			return []Node{Decl("text-decoration-color", color)}, Handled
		}
		if c.Modifier != nil {
			return nil, NotHandled
		}
		if thickness, ok := theme.Resolve(strPtr(c.Value.Value), []string{"--text-decoration-thickness"}, 0); ok {
			return []Node{Decl("text-decoration-thickness", thickness)}, Handled
		}
		if bare, ok := pixels(c.Value); ok {
			return []Node{Decl("text-decoration-thickness", bare)}, Handled
		}
		return nil, NotHandled
	}, nil)

	for _, v := range []string{"none", "manual", "auto"} {
		stat("hyphens-"+v, [2]string{"-webkit-hyphens", v}, [2]string{"hyphens", v})
	}

	stat("normal-nums", [2]string{"font-variant-numeric", "normal"})
	numericValue := make([]string, len(numericVariables))
	for i, v := range numericVariables {
		numericValue[i] = "var(" + v + ",)"
	}
	numeric := func(name, variable string) {
		statFn(name, func() []Node {
			nodes := make([]Node, 0, len(numericVariables)+2)
			for _, v := range numericVariables {
				nodes = append(nodes, Property(v, nil))
			}
			return append(nodes, Decl(variable, name), Decl("font-variant-numeric", strings.Join(numericValue, " ")))
		})
	}
	numeric("ordinal", "--tw-ordinal")
	numeric("slashed-zero", "--tw-slashed-zero")
	numeric("lining-nums", "--tw-numeric-figure")
	numeric("oldstyle-nums", "--tw-numeric-figure")
	numeric("proportional-nums", "--tw-numeric-spacing")
	numeric("tabular-nums", "--tw-numeric-spacing")
	numeric("diagonal-fractions", "--tw-numeric-fraction")
	numeric("stacked-fractions", "--tw-numeric-fraction")

	for _, v := range []string{"baseline", "top", "middle", "bottom", "text-top", "text-bottom", "sub", "super"} {
		stat("align-"+v, [2]string{"vertical-align", v})
	}
	fn("align", FunctionalUtilityDescription{Handle: single("vertical-align")})

	for _, v := range []string{"ultra-condensed", "extra-condensed", "condensed", "semi-condensed", "normal", "semi-expanded", "expanded", "extra-expanded", "ultra-expanded"} {
		stat("font-stretch-"+v, [2]string{"font-stretch", v})
	}
	fn("font-stretch", FunctionalUtilityDescription{ThemeKeys: []string{"--font-stretch"},
		HandleBareValue: func(v *CandidateValue) (string, bool) {
			return v.Value, strings.HasSuffix(v.Value, "%") && IsPositiveInteger(v.Value[:len(v.Value)-1])
		},
		Handle: single("font-stretch")})

	stat("text-shadow-none", [2]string{"text-shadow", "none"})
	u.Functional("text-shadow", func(c *Candidate) ([]Node, CompileStatus) {
		color := func(value string) ([]Node, CompileStatus) {
			return []Node{Property("--tw-text-shadow-color", nil), Decl("--tw-text-shadow-color", value)}, Handled
		}
		wrap := func(value string) ([]Node, CompileStatus) {
			failed := false
			replaced := ReplaceShadowColors(value, func(col string) string {
				resolved, ok := AsColor(col, c.Modifier, theme)
				if !ok {
					failed = true
					return col
				}
				return "var(--tw-text-shadow-color, " + resolved + ")"
			}, false)
			if failed {
				return nil, NotHandled
			}
			return []Node{Decl("text-shadow", replaced)}, Handled
		}
		if c.Value == nil {
			value, ok := theme.ResolveValue(nil, []string{"--text-shadow"})
			if !ok {
				return nil, NotHandled
			}
			return wrap(value)
		}
		if c.Value.Kind == ValueArbitrary {
			value := c.Value.Value
			if arbitraryType(c, []string{"color"}) == "color" {
				resolved, ok := AsColor(value, c.Modifier, theme)
				if !ok {
					return nil, NotHandled
				}
				return color(resolved)
			}
			return wrap(value)
		}
		if themeColor, ok := ResolveThemeColor(c, []string{"--text-shadow-color", "--color"}, theme); ok {
			return color(themeColor)
		}
		value, ok := theme.ResolveValue(strPtr(c.Value.Value), []string{"--text-shadow"})
		if !ok {
			return nil, NotHandled
		}
		return wrap(value)
	}, nil)

	stat("wrap-break-word", [2]string{"overflow-wrap", "break-word"})
	stat("wrap-anywhere", [2]string{"overflow-wrap", "anywhere"})
	stat("wrap-normal", [2]string{"overflow-wrap", "normal"})
	stat("list-image-none", [2]string{"list-style-image", "none"})
	fn("list-image", FunctionalUtilityDescription{Handle: single("list-style-image")})
	statFn("content-none", func() []Node {
		return []Node{Property("--tw-content", strPtr(`""`)), Decl("--tw-content", "none"), Decl("content", "none")}
	})

	// Backgrounds and blending.
	fn("bg-position", FunctionalUtilityDescription{Handle: single("background-position")})
	fn("bg-size", FunctionalUtilityDescription{Handle: single("background-size")})
	for _, mode := range blendModes {
		stat("bg-blend-"+mode, [2]string{"background-blend-mode", mode})
		stat("mix-blend-"+mode, [2]string{"mix-blend-mode", mode})
	}
	stat("mix-blend-plus-darker", [2]string{"mix-blend-mode", "plus-darker"})
	stat("mix-blend-plus-lighter", [2]string{"mix-blend-mode", "plus-lighter"})

	// Layout.
	statFn("container", func() []Node {
		var breakpoints []string
		for _, entry := range theme.Namespace("--breakpoint") {
			if entry.Self || strings.HasPrefix(entry.Key, "--") {
				continue
			}
			breakpoints = append(breakpoints, entry.Value)
		}
		sort.SliceStable(breakpoints, func(i, j int) bool { return CompareQueryValues(breakpoints[i], breakpoints[j]) < 0 })
		nodes := []Node{Decl("width", "100%")}
		for _, value := range breakpoints {
			nodes = append(nodes, NewAtRule("@media", "(width >= "+value+")", Decl("max-width", value)))
		}
		return nodes
	})
	for _, v := range []string{"auto", "avoid", "all", "avoid-page", "page", "left", "right", "column"} {
		stat("break-after-"+v, [2]string{"break-after", v})
		stat("break-before-"+v, [2]string{"break-before", v})
	}
	for _, v := range []string{"auto", "avoid", "avoid-page", "avoid-column"} {
		stat("break-inside-"+v, [2]string{"break-inside", v})
	}
	stat("box-decoration-clone", [2]string{"box-decoration-break", "clone"})
	stat("box-decoration-slice", [2]string{"box-decoration-break", "slice"})
	fn("object", FunctionalUtilityDescription{Handle: single("object-position")})

	// Tables.
	stat("border-collapse", [2]string{"border-collapse", "collapse"})
	stat("border-separate", [2]string{"border-collapse", "separate"})
	borderSpacing := func(name string, variables []string) {
		SpacingUtility(u, theme, name, []string{"--border-spacing", "--spacing"}, func(value string) []Node {
			nodes := []Node{Property("--tw-border-spacing-x", strPtr("0")), Property("--tw-border-spacing-y", strPtr("0"))}
			for _, v := range variables {
				nodes = append(nodes, Decl(v, value))
			}
			return append(nodes, Decl("border-spacing", "var(--tw-border-spacing-x) var(--tw-border-spacing-y)"))
		}, false, false)
	}
	borderSpacing("border-spacing", []string{"--tw-border-spacing-x", "--tw-border-spacing-y"})
	borderSpacing("border-spacing-x", []string{"--tw-border-spacing-x"})
	borderSpacing("border-spacing-y", []string{"--tw-border-spacing-y"})
	stat("table-auto", [2]string{"table-layout", "auto"})
	stat("table-fixed", [2]string{"table-layout", "fixed"})
	stat("caption-top", [2]string{"caption-side", "top"})
	stat("caption-bottom", [2]string{"caption-side", "bottom"})

	// Scrolling and touch.
	for _, side := range [][2]string{{"", ""}, {"x", "-inline"}, {"y", "-block"}, {"s", "-inline-start"}, {"e", "-inline-end"}, {"t", "-top"}, {"r", "-right"}, {"b", "-bottom"}, {"l", "-left"}} {
		margin, padding := "scroll-margin"+side[1], "scroll-padding"+side[1]
		SpacingUtility(u, theme, "scroll-m"+side[0], []string{"--scroll-margin", "--spacing"}, func(value string) []Node { return []Node{Decl(margin, value)} }, true, false)
		SpacingUtility(u, theme, "scroll-p"+side[0], []string{"--scroll-padding", "--spacing"}, func(value string) []Node { return []Node{Decl(padding, value)} }, false, false)
	}
	stat("snap-none", [2]string{"scroll-snap-type", "none"})
	for _, axis := range []string{"x", "y", "both"} {
		axis := axis
		statFn("snap-"+axis, func() []Node {
			return []Node{Property("--tw-scroll-snap-strictness", strPtr("proximity")), Decl("scroll-snap-type", axis+" var(--tw-scroll-snap-strictness)")}
		})
	}
	for _, strictness := range []string{"mandatory", "proximity"} {
		strictness := strictness
		statFn("snap-"+strictness, func() []Node {
			return []Node{Property("--tw-scroll-snap-strictness", strPtr("proximity")), Decl("--tw-scroll-snap-strictness", strictness)}
		})
	}
	for _, v := range []string{"start", "end", "center"} {
		stat("snap-"+v, [2]string{"scroll-snap-align", v})
	}
	stat("snap-align-none", [2]string{"scroll-snap-align", "none"})
	stat("snap-normal", [2]string{"scroll-snap-stop", "normal"})
	stat("snap-always", [2]string{"scroll-snap-stop", "always"})
	for _, v := range []string{"auto", "none", "manipulation"} {
		stat("touch-"+v, [2]string{"touch-action", v})
	}
	touch := func(value, variable string) {
		statFn("touch-"+value, func() []Node {
			return []Node{
				Property("--tw-pan-x", nil), Property("--tw-pan-y", nil), Property("--tw-pinch-zoom", nil),
				Decl(variable, value), Decl("touch-action", "var(--tw-pan-x,) var(--tw-pan-y,) var(--tw-pinch-zoom,)"),
			}
		})
	}
	for _, v := range []string{"pan-x", "pan-left", "pan-right"} {
		touch(v, "--tw-pan-x")
	}
	for _, v := range []string{"pan-y", "pan-up", "pan-down"} {
		touch(v, "--tw-pan-y")
	}
	touch("pinch-zoom", "--tw-pinch-zoom")

	// Interactivity and SVG.
	u.Functional("stroke", func(c *Candidate) ([]Node, CompileStatus) {
		if c.Value == nil {
			return nil, NotHandled
		}
		if c.Value.Kind == ValueArbitrary {
			value := c.Value.Value
			switch arbitraryType(c, []string{"color", "length", "number", "percentage"}) {
			case "length", "number", "percentage":
				if c.Modifier != nil {
					return nil, NotHandled
				}
				return []Node{Decl("stroke-width", value)}, Handled
			}
			resolved, ok := AsColor(value, c.Modifier, theme)
			if !ok {
				return nil, NotHandled
			}
			return []Node{Decl("stroke", resolved)}, Handled
		}
		if color, ok := ResolveThemeColor(c, []string{"--stroke", "--color"}, theme); ok {
			return []Node{Decl("stroke", color)}, Handled
		}
		if c.Modifier != nil {
			return nil, NotHandled
		}
		if width, ok := theme.Resolve(strPtr(c.Value.Value), []string{"--stroke-width"}, 0); ok {
			return []Node{Decl("stroke-width", width)}, Handled
		}
		if IsPositiveInteger(c.Value.Value) {
			return []Node{Decl("stroke-width", c.Value.Value)}, Handled
		}
		return nil, NotHandled
	}, nil)
	for _, s := range [][2]string{{"normal", "normal"}, {"dark", "dark"}, {"light", "light"}, {"light-dark", "light dark"}, {"only-dark", "only dark"}, {"only-light", "only light"}} {
		stat("scheme-"+s[0], [2]string{"color-scheme", s[1]})
	}
	stat("field-sizing-content", [2]string{"field-sizing", "content"})
	stat("field-sizing-fixed", [2]string{"field-sizing", "fixed"})
	stat("forced-color-adjust-auto", [2]string{"forced-color-adjust", "auto"})
	stat("forced-color-adjust-none", [2]string{"forced-color-adjust", "none"})
}
