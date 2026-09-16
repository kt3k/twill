package twill

import "strings"

// Gradient utilities (SPEC §10.8, "Gradients").

// GradientStops is the stops value for from-* and to-*.
const GradientStops = "var(--tw-gradient-via-stops, var(--tw-gradient-position), var(--tw-gradient-from) var(--tw-gradient-from-position), var(--tw-gradient-to) var(--tw-gradient-to-position))"

// GradientViaStops is the stops value set by via-*.
const GradientViaStops = "var(--tw-gradient-position), var(--tw-gradient-from) var(--tw-gradient-from-position), var(--tw-gradient-via) var(--tw-gradient-via-position), var(--tw-gradient-to) var(--tw-gradient-to-position)"

var interpolationMethods = map[string]bool{
	"srgb": true, "srgb-linear": true, "display-p3": true, "a98-rgb": true, "prophoto-rgb": true,
	"rec2020": true, "lab": true, "oklab": true, "xyz": true, "xyz-d50": true, "xyz-d65": true,
	"hsl": true, "hwb": true, "lch": true, "oklch": true,
}
var hueMethods = map[string]bool{"longer": true, "shorter": true, "increasing": true, "decreasing": true}

var gradientSides = map[string]string{
	"t": "top", "tr": "top right", "r": "right", "br": "bottom right",
	"b": "bottom", "bl": "bottom left", "l": "left", "tl": "top left",
}

// GradientRegistrations returns the registrations emitted by every stop utility.
func GradientRegistrations() []Node {
	transparent := strPtr("#0000")
	return []Node{
		Property("--tw-gradient-position", nil),
		PropertyRegistration("--tw-gradient-from", transparent, "<color>", false),
		PropertyRegistration("--tw-gradient-via", transparent, "<color>", false),
		PropertyRegistration("--tw-gradient-to", transparent, "<color>", false),
		Property("--tw-gradient-stops", nil),
		Property("--tw-gradient-via-stops", nil),
		PropertyRegistration("--tw-gradient-from-position", strPtr("0%"), "<length-percentage>", false),
		PropertyRegistration("--tw-gradient-via-position", strPtr("50%"), "<length-percentage>", false),
		PropertyRegistration("--tw-gradient-to-position", strPtr("100%"), "<length-percentage>", false),
	}
}

// Interpolation resolves an interpolation modifier.
func Interpolation(modifier *Modifier) (string, bool) {
	if modifier == nil {
		return "in oklab", true
	}
	if modifier.Kind == ValueArbitrary {
		return modifier.Value, true
	}
	if interpolationMethods[modifier.Value] {
		return "in " + modifier.Value, true
	}
	if hueMethods[modifier.Value] {
		return "in oklch " + modifier.Value + " hue", true
	}
	return "", false
}

// RegisterGradientUtilities registers the gradient utilities.
func RegisterGradientUtilities(u *Utilities, theme *Theme) {
	image := func(shape string) Node {
		return Decl("background-image", shape+"-gradient(var(--tw-gradient-stops))")
	}
	positioned := func(position string, modifier *Modifier, shape string) ([]Node, CompileStatus) {
		method, ok := Interpolation(modifier)
		if !ok {
			return nil, NotHandled
		}
		value := method
		if position != "" {
			value = position + " " + method
		}
		return []Node{Decl("--tw-gradient-position", value), image(shape)}, Handled
	}

	linear := func(c *Candidate, negative, legacy bool) ([]Node, CompileStatus) {
		if c.Value == nil {
			return nil, NotHandled
		}
		if c.Value.Kind == ValueArbitrary {
			if legacy || negative {
				return nil, NotHandled
			}
			value := c.Value.Value
			t := ""
			if c.Value.DataType != nil {
				t = *c.Value.DataType
			} else {
				t = InferDataType(value, []string{"angle"})
			}
			if t == "angle" {
				return positioned(value, c.Modifier, "linear")
			}
			if c.Modifier != nil {
				return nil, NotHandled
			}
			return []Node{Decl("background-image", "linear-gradient("+value+")")}, Handled
		}
		named := c.Value.Value
		if strings.HasPrefix(named, "to-") {
			if negative {
				return nil, NotHandled
			}
			side, ok := gradientSides[named[3:]]
			if !ok {
				return nil, NotHandled
			}
			return positioned("to "+side, c.Modifier, "linear")
		}
		if legacy || !IsPositiveInteger(named) {
			return nil, NotHandled
		}
		angle := named + "deg"
		if negative {
			angle = "calc(" + named + "deg * -1)"
		}
		return positioned(angle, c.Modifier, "linear")
	}
	u.Functional("bg-linear", func(c *Candidate) ([]Node, CompileStatus) { return linear(c, false, false) }, nil)
	u.Functional("-bg-linear", func(c *Candidate) ([]Node, CompileStatus) { return linear(c, true, false) }, nil)
	u.Functional("bg-gradient", func(c *Candidate) ([]Node, CompileStatus) { return linear(c, false, true) }, nil)

	u.Functional("bg-radial", func(c *Candidate) ([]Node, CompileStatus) {
		if c.Value == nil {
			return positioned("", c.Modifier, "radial")
		}
		if c.Value.Kind != ValueArbitrary || c.Modifier != nil {
			return nil, NotHandled
		}
		return []Node{Decl("--tw-gradient-position", c.Value.Value), image("radial")}, Handled
	}, nil)

	conic := func(c *Candidate, negative bool) ([]Node, CompileStatus) {
		if c.Value == nil {
			if negative {
				return nil, NotHandled
			}
			return positioned("", c.Modifier, "conic")
		}
		if c.Value.Kind == ValueArbitrary {
			if negative || c.Modifier != nil {
				return nil, NotHandled
			}
			return []Node{Decl("--tw-gradient-position", c.Value.Value), image("conic")}, Handled
		}
		named := c.Value.Value
		if !IsPositiveInteger(named) {
			return nil, NotHandled
		}
		angle := named + "deg"
		if negative {
			angle = "calc(" + named + "deg * -1)"
		}
		return positioned("from "+angle, c.Modifier, "conic")
	}
	u.Functional("bg-conic", func(c *Candidate) ([]Node, CompileStatus) { return conic(c, false) }, nil)
	u.Functional("-bg-conic", func(c *Candidate) ([]Node, CompileStatus) { return conic(c, true) }, nil)

	stop := func(name, colorVariable, positionVariable string, stops [][2]string) {
		color := func(value string) ([]Node, CompileStatus) {
			nodes := append(GradientRegistrations(), Decl(colorVariable, value))
			for _, s := range stops {
				nodes = append(nodes, Decl(s[0], s[1]))
			}
			return nodes, Handled
		}
		position := func(value string) ([]Node, CompileStatus) {
			return append(GradientRegistrations(), Decl(positionVariable, value)), Handled
		}
		u.Functional(name, func(c *Candidate) ([]Node, CompileStatus) {
			if c.Value == nil {
				return nil, NotHandled
			}
			if c.Value.Kind == ValueArbitrary {
				value := c.Value.Value
				t := ""
				if c.Value.DataType != nil {
					t = *c.Value.DataType
				} else {
					t = InferDataType(value, []string{"color", "length", "percentage"})
				}
				if t == "length" || t == "percentage" {
					if c.Modifier != nil {
						return nil, NotHandled
					}
					return position(value)
				}
				resolved, ok := AsColor(value, c.Modifier, theme)
				if !ok {
					return nil, NotHandled
				}
				return color(resolved)
			}
			if themeColor, ok := ResolveThemeColor(c, []string{"--background-color", "--color"}, theme); ok {
				return color(themeColor)
			}
			if c.Modifier != nil {
				return nil, NotHandled
			}
			named := c.Value.Value
			if strings.HasSuffix(named, "%") && IsPositiveInteger(named[:len(named)-1]) {
				return position(named)
			}
			return nil, NotHandled
		}, nil)
	}
	stop("from", "--tw-gradient-from", "--tw-gradient-from-position", [][2]string{{"--tw-gradient-stops", GradientStops}})
	stop("via", "--tw-gradient-via", "--tw-gradient-via-position", [][2]string{{"--tw-gradient-via-stops", GradientViaStops}, {"--tw-gradient-stops", "var(--tw-gradient-via-stops)"}})
	stop("to", "--tw-gradient-to", "--tw-gradient-to-position", [][2]string{{"--tw-gradient-stops", GradientStops}})
}
