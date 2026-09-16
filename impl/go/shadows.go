package twill

import (
	"regexp"
	"strings"
)

// Shadow and ring utilities (SPEC §10.8, "Shadows and rings").

// BoxShadow is the composed box-shadow value.
const BoxShadow = "var(--tw-inset-shadow), var(--tw-inset-ring-shadow), var(--tw-ring-offset-shadow), var(--tw-ring-shadow), var(--tw-shadow)"

var shadowKeywords = map[string]bool{"inset": true, "inherit": true, "initial": true, "revert": true, "unset": true}
var shadowLengthToken = regexp.MustCompile(`^(?:\d|\.|-[\d.])`)
var shadowWhitespace = regexp.MustCompile(`\s+`)

// ReplaceShadowColors rewrites the color of every shadow in a box-shadow
// value. Shadows with fewer than two length tokens are left unchanged. With
// inset, shadows without the inset keyword are prefixed with it.
func ReplaceShadowColors(value string, fn func(color string) string, inset bool) string {
	shadows := Segment(value, ',')
	out := make([]string, 0, len(shadows))
	for _, shadow := range shadows {
		var tokens []string
		for _, t := range Segment(strings.TrimSpace(shadowWhitespace.ReplaceAllString(shadow, " ")), ' ') {
			if t != "" {
				tokens = append(tokens, t)
			}
		}
		lengths, colorIndex, hasInset := 0, -1, false
		for i, token := range tokens {
			switch {
			case shadowKeywords[token]:
				if token == "inset" {
					hasInset = true
				}
			case shadowLengthToken.MatchString(token):
				lengths++
			case colorIndex == -1:
				colorIndex = i
			}
		}
		if lengths < 2 {
			out = append(out, strings.TrimSpace(shadow))
			continue
		}
		if colorIndex == -1 {
			tokens = append(tokens, fn("currentcolor"))
		} else {
			tokens[colorIndex] = fn(tokens[colorIndex])
		}
		if inset && !hasInset {
			tokens = append([]string{"inset"}, tokens...)
		}
		out = append(out, strings.Join(tokens, " "))
	}
	return strings.Join(out, ", ")
}

// ShadowRegistrations returns the registrations emitted by every shadow and
// ring utility.
func ShadowRegistrations() []Node {
	zero := strPtr("0 0 #0000")
	return []Node{
		Property("--tw-shadow", zero),
		Property("--tw-shadow-color", nil),
		Property("--tw-inset-shadow", zero),
		Property("--tw-inset-shadow-color", nil),
		Property("--tw-ring-color", nil),
		Property("--tw-ring-shadow", zero),
		Property("--tw-inset-ring-color", nil),
		Property("--tw-inset-ring-shadow", zero),
		Property("--tw-ring-inset", nil),
		Property("--tw-ring-offset-width", strPtr("0px")),
		Property("--tw-ring-offset-color", strPtr("#fff")),
		Property("--tw-ring-offset-shadow", zero),
	}
}

var shadowColorKeys = []string{"--box-shadow-color", "--color"}
var ringColorKeys = []string{"--ring-color", "--color"}

// RegisterShadowUtilities registers the shadow and ring utilities.
func RegisterShadowUtilities(u *Utilities, theme *Theme) {
	withBoxShadow := func(variable, value string) []Node {
		return append(ShadowRegistrations(), Decl(variable, value), Decl("box-shadow", BoxShadow))
	}
	colorOnly := func(variable, value string) []Node {
		return append(ShadowRegistrations(), Decl(variable, value))
	}
	arbitraryType := func(c *Candidate, types []string) string {
		if c.Value.DataType != nil {
			return *c.Value.DataType
		}
		return InferDataType(c.Value.Value, types)
	}

	shadow := func(root, namespace, variable, colorVariable string, inset bool) {
		u.Static(root+"-none", func(*Candidate) ([]Node, CompileStatus) {
			return withBoxShadow(variable, "0 0 #0000"), Handled
		})
		u.Functional(root, func(c *Candidate) ([]Node, CompileStatus) {
			wrap := func(value string, insetArbitrary bool) ([]Node, CompileStatus) {
				failed := false
				replaced := ReplaceShadowColors(value, func(color string) string {
					resolved, ok := AsColor(color, c.Modifier, theme)
					if !ok {
						failed = true
						return color
					}
					return "var(" + colorVariable + ", " + resolved + ")"
				}, insetArbitrary)
				if failed {
					return nil, NotHandled
				}
				return withBoxShadow(variable, replaced), Handled
			}
			if c.Value == nil {
				value, ok := theme.ResolveValue(nil, []string{namespace})
				if !ok {
					return nil, NotHandled
				}
				return wrap(value, false)
			}
			if c.Value.Kind == ValueArbitrary {
				value := c.Value.Value
				if arbitraryType(c, []string{"color"}) == "color" {
					resolved, ok := AsColor(value, c.Modifier, theme)
					if !ok {
						return nil, NotHandled
					}
					return colorOnly(colorVariable, resolved), Handled
				}
				return wrap(value, inset)
			}
			if color, ok := ResolveThemeColor(c, shadowColorKeys, theme); ok {
				return colorOnly(colorVariable, color), Handled
			}
			value, ok := theme.ResolveValue(strPtr(c.Value.Value), []string{namespace})
			if !ok {
				return nil, NotHandled
			}
			return wrap(value, false)
		}, nil)
	}
	shadow("shadow", "--shadow", "--tw-shadow", "--tw-shadow-color", false)
	shadow("inset-shadow", "--inset-shadow", "--tw-inset-shadow", "--tw-inset-shadow-color", true)

	ring := func(root, variable, colorVariable string, shadowFor func(width string) string) {
		u.Functional(root, func(c *Candidate) ([]Node, CompileStatus) {
			width := func(value string) ([]Node, CompileStatus) {
				return withBoxShadow(variable, shadowFor(value)), Handled
			}
			if c.Value == nil {
				if c.Modifier != nil {
					return nil, NotHandled
				}
				w, ok := theme.Get("--default-ring-width")
				if !ok {
					w = "1px"
				}
				return width(w)
			}
			if c.Value.Kind == ValueArbitrary {
				value := c.Value.Value
				switch arbitraryType(c, []string{"color", "length", "line-width"}) {
				case "color":
					resolved, ok := AsColor(value, c.Modifier, theme)
					if !ok {
						return nil, NotHandled
					}
					return colorOnly(colorVariable, resolved), Handled
				case "length", "line-width":
					if c.Modifier != nil {
						return nil, NotHandled
					}
					return width(value)
				}
				return nil, NotHandled
			}
			if color, ok := ResolveThemeColor(c, ringColorKeys, theme); ok {
				return colorOnly(colorVariable, color), Handled
			}
			if c.Modifier != nil {
				return nil, NotHandled
			}
			if w, ok := theme.Resolve(strPtr(c.Value.Value), []string{"--ring-width"}, 0); ok {
				return width(w)
			}
			if IsPositiveInteger(c.Value.Value) {
				return width(c.Value.Value + "px")
			}
			return nil, NotHandled
		}, nil)
	}
	ring("ring", "--tw-ring-shadow", "--tw-ring-color", func(w string) string {
		return "var(--tw-ring-inset,) 0 0 0 calc(" + w + " + var(--tw-ring-offset-width)) var(--tw-ring-color, currentcolor)"
	})
	ring("inset-ring", "--tw-inset-ring-shadow", "--tw-inset-ring-color", func(w string) string {
		return "inset 0 0 0 " + w + " var(--tw-inset-ring-color, currentcolor)"
	})
	u.Static("ring-inset", func(*Candidate) ([]Node, CompileStatus) {
		return colorOnly("--tw-ring-inset", "inset"), Handled
	})

	u.Functional("ring-offset", func(c *Candidate) ([]Node, CompileStatus) {
		width := func(value string) ([]Node, CompileStatus) {
			return append(ShadowRegistrations(),
				Decl("--tw-ring-offset-width", value),
				Decl("--tw-ring-offset-shadow", "var(--tw-ring-inset,) 0 0 0 var(--tw-ring-offset-width) var(--tw-ring-offset-color)"),
			), Handled
		}
		if c.Value == nil {
			return nil, NotHandled
		}
		if c.Value.Kind == ValueArbitrary {
			value := c.Value.Value
			switch arbitraryType(c, []string{"color", "length"}) {
			case "color":
				resolved, ok := AsColor(value, c.Modifier, theme)
				if !ok {
					return nil, NotHandled
				}
				return colorOnly("--tw-ring-offset-color", resolved), Handled
			case "length":
				if c.Modifier != nil {
					return nil, NotHandled
				}
				return width(value)
			}
			return nil, NotHandled
		}
		if color, ok := ResolveThemeColor(c, []string{"--ring-offset-color", "--color"}, theme); ok {
			return colorOnly("--tw-ring-offset-color", color), Handled
		}
		if c.Modifier != nil {
			return nil, NotHandled
		}
		if w, ok := theme.Resolve(strPtr(c.Value.Value), []string{"--ring-offset-width"}, 0); ok {
			return width(w)
		}
		if IsPositiveInteger(c.Value.Value) {
			return width(c.Value.Value + "px")
		}
		return nil, NotHandled
	}, nil)
}
