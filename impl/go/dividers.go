package twill

// Space, divider, and outline utilities (SPEC §10.8, "Space, dividers, and
// outlines").

// ChildrenSelector targets every child but the last.
const ChildrenSelector = ":where(& > :not(:last-child))"

func children(nodes ...Node) Node { return StyleRule(ChildrenSelector, nodes...) }

// RegisterDividerUtilities registers the space, divide, and outline utilities.
func RegisterDividerUtilities(u *Utilities, theme *Theme) {
	statFn := func(name string, fn func() []Node) { StaticUtilityFn(u, name, fn) }
	pixels := func(v *CandidateValue) (string, bool) { return v.Value + "px", IsPositiveInteger(v.Value) }
	arbitraryType := func(c *Candidate, types []string) string {
		if c.Value.DataType != nil {
			return *c.Value.DataType
		}
		return InferDataType(c.Value.Value, types)
	}

	for _, axis := range []string{"x", "y"} {
		reverse := "--tw-space-" + axis + "-reverse"
		start, end := "margin-inline-start", "margin-inline-end"
		if axis == "y" {
			start, end = "margin-block-start", "margin-block-end"
		}
		SpacingUtility(u, theme, "space-"+axis, []string{"--spacing"}, func(value string) []Node {
			return []Node{
				Property(reverse, strPtr("0")),
				children(
					Decl(reverse, "0"),
					Decl(start, "calc("+value+" * var("+reverse+"))"),
					Decl(end, "calc("+value+" * calc(1 - var("+reverse+")))"),
				),
			}
		}, true, false)
		statFn("space-"+axis+"-reverse", func() []Node {
			return []Node{Property(reverse, strPtr("0")), children(Decl(reverse, "1"))}
		})
	}

	for _, axis := range []string{"x", "y"} {
		reverse := "--tw-divide-" + axis + "-reverse"
		style, start, end := "border-inline-style", "border-inline-start-width", "border-inline-end-width"
		if axis == "y" {
			style, start, end = "border-block-style", "border-block-start-width", "border-block-end-width"
		}
		width := func(value string) ([]Node, CompileStatus) {
			return []Node{
				Property(reverse, strPtr("0")),
				Property("--tw-border-style", strPtr("solid")),
				children(
					Decl(reverse, "0"),
					Decl(style, "var(--tw-border-style)"),
					Decl(start, "calc("+value+" * var("+reverse+"))"),
					Decl(end, "calc("+value+" * calc(1 - var("+reverse+")))"),
				),
			}, Handled
		}
		u.Functional("divide-"+axis, func(c *Candidate) ([]Node, CompileStatus) {
			if c.Modifier != nil {
				return nil, NotHandled
			}
			if c.Value == nil {
				w, ok := theme.Get("--default-border-width")
				if !ok {
					w = "1px"
				}
				return width(w)
			}
			if c.Value.Kind == ValueArbitrary {
				switch arbitraryType(c, []string{"length", "line-width"}) {
				case "length", "line-width":
					return width(c.Value.Value)
				}
				return nil, NotHandled
			}
			if w, ok := theme.Resolve(strPtr(c.Value.Value), []string{"--divide-width"}, 0); ok {
				return width(w)
			}
			if bare, ok := pixels(c.Value); ok {
				return width(bare)
			}
			return nil, NotHandled
		}, nil)
		statFn("divide-"+axis+"-reverse", func() []Node {
			return []Node{Property(reverse, strPtr("0")), children(Decl(reverse, "1"))}
		})
	}
	ColorUtility(u, theme, "divide", []string{"--divide-color", "--color"}, func(value string) []Node {
		return []Node{children(Decl("border-color", value))}
	})
	for _, style := range []string{"solid", "dashed", "dotted", "double", "none"} {
		style := style
		statFn("divide-"+style, func() []Node {
			return []Node{children(Decl("--tw-border-style", style), Decl("border-style", style))}
		})
	}

	outlineWidth := func(value string) ([]Node, CompileStatus) {
		return []Node{
			Property("--tw-outline-style", strPtr("solid")),
			Decl("outline-style", "var(--tw-outline-style)"),
			Decl("outline-width", value),
		}, Handled
	}
	u.Functional("outline", func(c *Candidate) ([]Node, CompileStatus) {
		if c.Value == nil {
			if c.Modifier != nil {
				return nil, NotHandled
			}
			return outlineWidth("1px")
		}
		if c.Value.Kind == ValueArbitrary {
			value := c.Value.Value
			switch arbitraryType(c, []string{"color", "length", "line-width"}) {
			case "color":
				resolved, ok := AsColor(value, c.Modifier, theme)
				if !ok {
					return nil, NotHandled
				}
				return []Node{Decl("outline-color", resolved)}, Handled
			case "length", "line-width":
				if c.Modifier != nil {
					return nil, NotHandled
				}
				return outlineWidth(value)
			}
			return nil, NotHandled
		}
		if color, ok := ResolveThemeColor(c, []string{"--outline-color", "--color"}, theme); ok {
			return []Node{Decl("outline-color", color)}, Handled
		}
		if c.Modifier != nil {
			return nil, NotHandled
		}
		if w, ok := theme.Resolve(strPtr(c.Value.Value), []string{"--outline-width"}, 0); ok {
			return outlineWidth(w)
		}
		if bare, ok := pixels(c.Value); ok {
			return outlineWidth(bare)
		}
		return nil, NotHandled
	}, nil)
	statFn("outline-none", func() []Node {
		return []Node{Decl("--tw-outline-style", "none"), Decl("outline-style", "none")}
	})
	statFn("outline-hidden", func() []Node {
		return []Node{
			Decl("--tw-outline-style", "none"), Decl("outline-style", "none"),
			NewAtRule("@media", "(forced-colors: active)", Decl("outline", "2px solid transparent"), Decl("outline-offset", "2px")),
		}
	})
	for _, style := range []string{"solid", "dashed", "dotted", "double"} {
		style := style
		statFn("outline-"+style, func() []Node {
			return []Node{Decl("--tw-outline-style", style), Decl("outline-style", style)}
		})
	}
	FunctionalUtility(u, theme, "outline-offset", FunctionalUtilityDescription{
		ThemeKeys: []string{"--outline-offset"}, SupportsNegative: true, HandleBareValue: pixels,
		Handle: func(value string, _ *string) []Node { return []Node{Decl("outline-offset", value)} },
	})
}
