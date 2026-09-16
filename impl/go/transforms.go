package twill

import "strings"

// Transform utilities (SPEC §10.8, "Transforms").

// Transform is the composed transform value.
const Transform = "var(--tw-rotate-x,) var(--tw-rotate-y,) var(--tw-rotate-z,) var(--tw-skew-x,) var(--tw-skew-y,)"

const translate2D = "var(--tw-translate-x) var(--tw-translate-y)"
const translate3D = translate2D + " var(--tw-translate-z)"
const scale2D = "var(--tw-scale-x) var(--tw-scale-y)"
const scale3D = scale2D + " var(--tw-scale-z)"

func transformRegistrations() []Node {
	return []Node{
		Property("--tw-rotate-x", nil), Property("--tw-rotate-y", nil), Property("--tw-rotate-z", nil),
		Property("--tw-skew-x", nil), Property("--tw-skew-y", nil),
	}
}

func translateRegistrations() []Node {
	zero := strPtr("0")
	return []Node{Property("--tw-translate-x", zero), Property("--tw-translate-y", zero), Property("--tw-translate-z", zero)}
}

func scaleRegistrations() []Node {
	one := strPtr("1")
	return []Node{Property("--tw-scale-x", one), Property("--tw-scale-y", one), Property("--tw-scale-z", one)}
}

var transformOrigins = [][2]string{
	{"center", "center"}, {"top", "top"}, {"top-right", "top right"}, {"right", "right"},
	{"bottom-right", "bottom right"}, {"bottom", "bottom"}, {"bottom-left", "bottom left"},
	{"left", "left"}, {"top-left", "top left"},
}

// RegisterTransformUtilities registers the transform utilities.
func RegisterTransformUtilities(u *Utilities, theme *Theme) {
	stat := func(name string, declarations ...[2]string) { StaticUtility(u, name, Declarations(declarations)) }
	statFn := func(name string, fn func() []Node) { StaticUtilityFn(u, name, fn) }
	fn := func(root string, desc FunctionalUtilityDescription) { FunctionalUtility(u, theme, root, desc) }
	degrees := func(v *CandidateValue) (string, bool) { return v.Value + "deg", IsPositiveInteger(v.Value) }
	percentage := func(v *CandidateValue) (string, bool) { return v.Value + "%", IsPositiveInteger(v.Value) }
	single := func(property string) func(string, *string) []Node {
		return func(value string, _ *string) []Node { return []Node{Decl(property, value)} }
	}

	stat("transform-none", [2]string{"transform", "none"})
	statFn("transform-gpu", func() []Node {
		return append(transformRegistrations(), Decl("transform", "translateZ(0) "+Transform))
	})
	statFn("transform-cpu", func() []Node { return append(transformRegistrations(), Decl("transform", Transform)) })
	fn("transform", FunctionalUtilityDescription{Handle: single("transform")})
	stat("transform-flat", [2]string{"transform-style", "flat"})
	stat("transform-3d", [2]string{"transform-style", "preserve-3d"})
	stat("backface-visible", [2]string{"backface-visibility", "visible"})
	stat("backface-hidden", [2]string{"backface-visibility", "hidden"})

	stat("perspective-none", [2]string{"perspective", "none"})
	fn("perspective", FunctionalUtilityDescription{ThemeKeys: []string{"--perspective"}, Handle: single("perspective")})
	for _, o := range transformOrigins {
		stat("perspective-origin-"+o[0], [2]string{"perspective-origin", o[1]})
		stat("origin-"+o[0], [2]string{"transform-origin", o[1]})
	}
	fn("perspective-origin", FunctionalUtilityDescription{Handle: single("perspective-origin")})
	fn("origin", FunctionalUtilityDescription{Handle: single("transform-origin")})

	translate := func(name string, variables []string, composed string, fractions bool) {
		handle := func(value string) []Node {
			nodes := translateRegistrations()
			for _, v := range variables {
				nodes = append(nodes, Decl(v, value))
			}
			return append(nodes, Decl("translate", composed))
		}
		SpacingUtility(u, theme, name, []string{"--translate", "--spacing"}, handle, true, fractions)
		if fractions {
			statFn(name+"-full", func() []Node { return handle("100%") })
			statFn("-"+name+"-full", func() []Node { return handle("-100%") })
		}
	}
	translate("translate", []string{"--tw-translate-x", "--tw-translate-y"}, translate2D, true)
	translate("translate-x", []string{"--tw-translate-x"}, translate2D, true)
	translate("translate-y", []string{"--tw-translate-y"}, translate2D, true)
	translate("translate-z", []string{"--tw-translate-z"}, translate3D, false)
	statFn("translate-3d", func() []Node { return append(translateRegistrations(), Decl("translate", translate3D)) })
	stat("translate-none", [2]string{"translate", "none"})

	scaleCompile := func(c *Candidate, negative bool) ([]Node, CompileStatus) {
		if c.Value == nil || c.Modifier != nil {
			return nil, NotHandled
		}
		if c.Value.Kind == ValueArbitrary {
			if negative {
				return nil, NotHandled
			}
			return []Node{Decl("scale", c.Value.Value)}, Handled
		}
		value, ok := theme.Resolve(strPtr(c.Value.Value), []string{"--scale"}, 0)
		if !ok {
			if value, ok = percentage(c.Value); !ok {
				return nil, NotHandled
			}
		}
		if negative {
			value = "calc(" + value + " * -1)"
		}
		return append(scaleRegistrations(),
			Decl("--tw-scale-x", value), Decl("--tw-scale-y", value), Decl("--tw-scale-z", value),
			Decl("scale", scale2D)), Handled
	}
	u.Functional("scale", func(c *Candidate) ([]Node, CompileStatus) { return scaleCompile(c, false) }, nil)
	u.Functional("-scale", func(c *Candidate) ([]Node, CompileStatus) { return scaleCompile(c, true) }, nil)
	for _, axis := range []string{"x", "y", "z"} {
		axis := axis
		composed := scale2D
		if axis == "z" {
			composed = scale3D
		}
		fn("scale-"+axis, FunctionalUtilityDescription{
			ThemeKeys: []string{"--scale"}, SupportsNegative: true, HandleBareValue: percentage,
			Handle: func(value string, _ *string) []Node {
				return append(scaleRegistrations(), Decl("--tw-scale-"+axis, value), Decl("scale", composed))
			},
		})
	}
	statFn("scale-3d", func() []Node { return append(scaleRegistrations(), Decl("scale", scale3D)) })
	stat("scale-none", [2]string{"scale", "none"})

	fn("rotate", FunctionalUtilityDescription{ThemeKeys: []string{"--rotate"}, SupportsNegative: true, HandleBareValue: degrees, Handle: single("rotate")})
	stat("rotate-none", [2]string{"rotate", "none"})
	for _, axis := range []string{"x", "y", "z"} {
		axis := axis
		fn("rotate-"+axis, FunctionalUtilityDescription{
			ThemeKeys: []string{"--rotate"}, SupportsNegative: true, HandleBareValue: degrees,
			Handle: func(value string, _ *string) []Node {
				return append(transformRegistrations(),
					Decl("--tw-rotate-"+axis, "rotate"+strings.ToUpper(axis)+"("+value+")"),
					Decl("transform", Transform))
			},
		})
	}

	skew := func(name string, axes []string) {
		fn(name, FunctionalUtilityDescription{
			ThemeKeys: []string{"--skew"}, SupportsNegative: true, HandleBareValue: degrees,
			Handle: func(value string, _ *string) []Node {
				nodes := transformRegistrations()
				for _, axis := range axes {
					nodes = append(nodes, Decl("--tw-skew-"+axis, "skew"+strings.ToUpper(axis)+"("+value+")"))
				}
				return append(nodes, Decl("transform", Transform))
			},
		})
	}
	skew("skew", []string{"x", "y"})
	skew("skew-x", []string{"x"})
	skew("skew-y", []string{"y"})
}
