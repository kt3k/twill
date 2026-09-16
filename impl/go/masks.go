package twill

// Mask utilities (SPEC §10.8, "Masks").

var maskEdges = []string{"top", "right", "bottom", "left"}

const maskWhite = "linear-gradient(#fff, #fff)"

func maskStopRegistrations(name string) []Node {
	return []Node{
		Property("--tw-mask-"+name+"-from-color", strPtr("black")),
		Property("--tw-mask-"+name+"-from-position", strPtr("0%")),
		Property("--tw-mask-"+name+"-to-color", strPtr("transparent")),
		Property("--tw-mask-"+name+"-to-position", strPtr("100%")),
	}
}

// MaskRegistrations returns the registrations emitted by every gradient mask utility.
func MaskRegistrations() []Node {
	var nodes []Node
	for _, edge := range maskEdges {
		nodes = append(nodes, Property("--tw-mask-"+edge, strPtr(maskWhite)))
		nodes = append(nodes, maskStopRegistrations(edge)...)
	}
	nodes = append(nodes, Property("--tw-mask-linear", strPtr(maskWhite)), Property("--tw-mask-linear-position", strPtr("0deg")))
	nodes = append(nodes, maskStopRegistrations("linear")...)
	nodes = append(nodes, Property("--tw-mask-radial", strPtr(maskWhite)), Property("--tw-mask-radial-shape", strPtr("ellipse")),
		Property("--tw-mask-radial-size", strPtr("farthest-corner")), Property("--tw-mask-radial-position", strPtr("center")))
	nodes = append(nodes, maskStopRegistrations("radial")...)
	nodes = append(nodes, Property("--tw-mask-conic", strPtr(maskWhite)), Property("--tw-mask-conic-position", strPtr("0deg")))
	nodes = append(nodes, maskStopRegistrations("conic")...)
	return nodes
}

const maskEdgeList = "var(--tw-mask-left), var(--tw-mask-right), var(--tw-mask-bottom), var(--tw-mask-top)"
const maskLinear = "linear-gradient(var(--tw-mask-linear-position), var(--tw-mask-linear-from-color) var(--tw-mask-linear-from-position), var(--tw-mask-linear-to-color) var(--tw-mask-linear-to-position))"
const maskRadial = "radial-gradient(var(--tw-mask-radial-shape) var(--tw-mask-radial-size) at var(--tw-mask-radial-position), var(--tw-mask-radial-from-color) var(--tw-mask-radial-from-position), var(--tw-mask-radial-to-color) var(--tw-mask-radial-to-position))"
const maskConic = "conic-gradient(from var(--tw-mask-conic-position), var(--tw-mask-conic-from-color) var(--tw-mask-conic-from-position), var(--tw-mask-conic-to-color) var(--tw-mask-conic-to-position))"

func maskEdgeGradient(edge string) string {
	return "linear-gradient(to " + edge + ", var(--tw-mask-" + edge + "-from-color) var(--tw-mask-" + edge + "-from-position), var(--tw-mask-" + edge + "-to-color) var(--tw-mask-" + edge + "-to-position))"
}

var maskPositions = [][2]string{
	{"top", "top"}, {"top-left", "top left"}, {"top-right", "top right"}, {"left", "left"}, {"center", "center"},
	{"right", "right"}, {"bottom", "bottom"}, {"bottom-left", "bottom left"}, {"bottom-right", "bottom right"},
}

type maskStop struct {
	kind  string
	value string
}

// RegisterMaskUtilities registers the mask utilities.
func RegisterMaskUtilities(u *Utilities, theme *Theme) {
	stat := func(name string, declarations ...[2]string) { StaticUtility(u, name, Declarations(declarations)) }
	fn := func(root string, desc FunctionalUtilityDescription) { FunctionalUtility(u, theme, root, desc) }
	single := func(property string) func(string, *string) []Node {
		return func(value string, _ *string) []Node { return []Node{Decl(property, value)} }
	}

	stat("mask-none", [2]string{"mask-image", "none"})
	fn("mask", FunctionalUtilityDescription{Handle: single("mask-image")})
	for _, v := range []string{"add", "subtract", "intersect", "exclude"} {
		stat("mask-"+v, [2]string{"mask-composite", v})
	}
	stat("mask-alpha", [2]string{"mask-mode", "alpha"})
	stat("mask-luminance", [2]string{"mask-mode", "luminance"})
	stat("mask-match", [2]string{"mask-mode", "match-source"})
	stat("mask-type-alpha", [2]string{"mask-type", "alpha"})
	stat("mask-type-luminance", [2]string{"mask-type", "luminance"})
	for _, v := range []string{"auto", "cover", "contain"} {
		stat("mask-"+v, [2]string{"mask-size", v})
	}
	fn("mask-size", FunctionalUtilityDescription{Handle: single("mask-size")})
	for _, box := range []string{"border", "padding", "content", "fill", "stroke", "view"} {
		stat("mask-clip-"+box, [2]string{"mask-clip", box + "-box"})
		stat("mask-origin-"+box, [2]string{"mask-origin", box + "-box"})
	}
	stat("mask-no-clip", [2]string{"mask-clip", "no-clip"})
	for _, p := range maskPositions {
		stat("mask-"+p[0], [2]string{"mask-position", p[1]})
	}
	fn("mask-position", FunctionalUtilityDescription{Handle: single("mask-position")})
	for _, r := range [][2]string{{"repeat", "repeat"}, {"no-repeat", "no-repeat"}, {"repeat-x", "repeat-x"}, {"repeat-y", "repeat-y"}, {"repeat-space", "space"}, {"repeat-round", "round"}} {
		stat("mask-"+r[0], [2]string{"mask-repeat", r[1]})
	}

	stopOf := func(c *Candidate) (maskStop, bool) {
		if c.Value == nil {
			return maskStop{}, false
		}
		if c.Value.Kind == ValueArbitrary {
			value := c.Value.Value
			t := ""
			if c.Value.DataType != nil {
				t = *c.Value.DataType
			} else {
				t = InferDataType(value, []string{"length", "percentage", "color"})
			}
			if t == "length" || t == "percentage" {
				if c.Modifier != nil {
					return maskStop{}, false
				}
				return maskStop{"position", value}, true
			}
			resolved, ok := AsColor(value, c.Modifier, theme)
			if !ok {
				return maskStop{}, false
			}
			return maskStop{"color", resolved}, true
		}
		if color, ok := ResolveThemeColor(c, []string{"--color"}, theme); ok {
			return maskStop{"color", color}, true
		}
		if c.Modifier != nil {
			return maskStop{}, false
		}
		named := c.Value.Value
		if len(named) > 1 && named[len(named)-1] == '%' && IsPositiveInteger(named[:len(named)-1]) {
			return maskStop{"position", named}, true
		}
		if IsPositiveInteger(named) {
			if _, ok := theme.Resolve(nil, []string{"--spacing"}, 0); !ok {
				return maskStop{}, false
			}
			return maskStop{"position", "--spacing(" + named + ")"}, true
		}
		return maskStop{}, false
	}
	composed := func(compositions [][2]string) []Node {
		nodes := append(MaskRegistrations(),
			Decl("mask-image", "var(--tw-mask-linear), var(--tw-mask-radial), var(--tw-mask-conic)"),
			Decl("mask-composite", "intersect"))
		for _, c := range compositions {
			nodes = append(nodes, Decl(c[0], c[1]))
		}
		return nodes
	}
	stopUtility := func(root, side string, compositions [][2]string, variables []string) {
		u.Functional(root+"-"+side, func(c *Candidate) ([]Node, CompileStatus) {
			stop, ok := stopOf(c)
			if !ok {
				return nil, NotHandled
			}
			nodes := composed(compositions)
			for _, v := range variables {
				nodes = append(nodes, Decl(v+"-"+side+"-"+stop.kind, stop.value))
			}
			return nodes, Handled
		}, nil)
	}

	for _, set := range []struct {
		name  string
		edges []string
	}{
		{"t", []string{"top"}}, {"r", []string{"right"}}, {"b", []string{"bottom"}}, {"l", []string{"left"}},
		{"x", []string{"left", "right"}}, {"y", []string{"top", "bottom"}},
	} {
		compositions := [][2]string{{"--tw-mask-linear", maskEdgeList}}
		var variables []string
		for _, e := range set.edges {
			compositions = append(compositions, [2]string{"--tw-mask-" + e, maskEdgeGradient(e)})
			variables = append(variables, "--tw-mask-"+e)
		}
		stopUtility("mask-"+set.name, "from", compositions, variables)
		stopUtility("mask-"+set.name, "to", compositions, variables)
	}

	angle := func(root, variable, composition string) {
		compile := func(c *Candidate, negative bool) ([]Node, CompileStatus) {
			if c.Value == nil || c.Modifier != nil {
				return nil, NotHandled
			}
			var value string
			if c.Value.Kind == ValueArbitrary {
				if negative {
					return nil, NotHandled
				}
				value = c.Value.Value
				t := ""
				if c.Value.DataType != nil {
					t = *c.Value.DataType
				} else {
					t = InferDataType(value, []string{"angle"})
				}
				if t != "angle" {
					return nil, NotHandled
				}
			} else {
				if !IsPositiveInteger(c.Value.Value) {
					return nil, NotHandled
				}
				value = c.Value.Value + "deg"
				if negative {
					value = "calc(" + c.Value.Value + "deg * -1)"
				}
			}
			return append(composed([][2]string{{variable, composition}}), Decl(variable+"-position", value)), Handled
		}
		u.Functional(root, func(c *Candidate) ([]Node, CompileStatus) { return compile(c, false) }, nil)
		u.Functional("-"+root, func(c *Candidate) ([]Node, CompileStatus) { return compile(c, true) }, nil)
	}
	angle("mask-linear", "--tw-mask-linear", maskLinear)
	stopUtility("mask-linear", "from", [][2]string{{"--tw-mask-linear", maskLinear}}, []string{"--tw-mask-linear"})
	stopUtility("mask-linear", "to", [][2]string{{"--tw-mask-linear", maskLinear}}, []string{"--tw-mask-linear"})

	u.Functional("mask-radial", func(c *Candidate) ([]Node, CompileStatus) {
		if c.Value == nil || c.Value.Kind != ValueArbitrary || c.Modifier != nil {
			return nil, NotHandled
		}
		return append(composed([][2]string{{"--tw-mask-radial", maskRadial}}), Decl("--tw-mask-radial-size", c.Value.Value)), Handled
	}, nil)
	stopUtility("mask-radial", "from", [][2]string{{"--tw-mask-radial", maskRadial}}, []string{"--tw-mask-radial"})
	stopUtility("mask-radial", "to", [][2]string{{"--tw-mask-radial", maskRadial}}, []string{"--tw-mask-radial"})
	stat("mask-circle", [2]string{"--tw-mask-radial-shape", "circle"})
	stat("mask-ellipse", [2]string{"--tw-mask-radial-shape", "ellipse"})
	for _, size := range []string{"closest-side", "farthest-side", "closest-corner", "farthest-corner"} {
		stat("mask-radial-"+size, [2]string{"--tw-mask-radial-size", size})
	}
	for _, p := range maskPositions {
		stat("mask-radial-at-"+p[0], [2]string{"--tw-mask-radial-position", p[1]})
	}
	fn("mask-radial-at", FunctionalUtilityDescription{Handle: single("--tw-mask-radial-position")})

	angle("mask-conic", "--tw-mask-conic", maskConic)
	stopUtility("mask-conic", "from", [][2]string{{"--tw-mask-conic", maskConic}}, []string{"--tw-mask-conic"})
	stopUtility("mask-conic", "to", [][2]string{{"--tw-mask-conic", maskConic}}, []string{"--tw-mask-conic"})
}
