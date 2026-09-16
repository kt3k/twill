package twill

import "strings"

// RegisterBuiltinUtilities registers every built-in utility (SPEC §10.8).
func RegisterBuiltinUtilities(u *Utilities, theme *Theme) {
	stat := func(name string, declarations ...[2]string) { StaticUtility(u, name, Declarations(declarations)) }
	fn := func(root string, desc FunctionalUtilityDescription) { FunctionalUtility(u, theme, root, desc) }
	spacing := func(name string, themeKeys []string, handle func(string) []Node, negative, fractions bool) {
		SpacingUtility(u, theme, name, themeKeys, handle, negative, fractions)
	}
	color := func(root string, themeKeys []string, handle func(string) []Node) {
		ColorUtility(u, theme, root, themeKeys, handle)
	}
	single := func(property string) func(string) []Node {
		return func(value string) []Node { return []Node{Decl(property, value)} }
	}
	multi := func(properties ...string) func(string) []Node {
		return func(value string) []Node {
			nodes := make([]Node, 0, len(properties))
			for _, p := range properties {
				nodes = append(nodes, Decl(p, value))
			}
			return nodes
		}
	}
	handle := func(h func(string) []Node) func(string, *string) []Node {
		return func(value string, _ *string) []Node { return h(value) }
	}
	bareSuffix := func(suffix string) func(*CandidateValue) (string, bool) {
		return func(v *CandidateValue) (string, bool) { return v.Value + suffix, IsPositiveInteger(v.Value) }
	}
	decls := func(properties []string, value string) [][2]string {
		out := make([][2]string, 0, len(properties))
		for _, p := range properties {
			out = append(out, [2]string{p, value})
		}
		return out
	}

	// Layout.
	for _, d := range [][2]string{
		{"block", "block"}, {"inline-block", "inline-block"}, {"inline", "inline"}, {"flex", "flex"},
		{"inline-flex", "inline-flex"}, {"grid", "grid"}, {"inline-grid", "inline-grid"}, {"hidden", "none"},
		{"contents", "contents"}, {"flow-root", "flow-root"}, {"table", "table"}, {"table-cell", "table-cell"},
		{"table-row", "table-row"}, {"table-caption", "table-caption"}, {"table-column", "table-column"},
		{"table-column-group", "table-column-group"}, {"table-footer-group", "table-footer-group"},
		{"table-header-group", "table-header-group"}, {"table-row-group", "table-row-group"},
		{"inline-table", "inline-table"}, {"list-item", "list-item"},
	} {
		stat(d[0], [2]string{"display", d[1]})
	}
	for _, name := range []string{"static", "fixed", "absolute", "relative", "sticky"} {
		stat(name, [2]string{"position", name})
	}
	stat("visible", [2]string{"visibility", "visible"})
	stat("invisible", [2]string{"visibility", "hidden"})
	stat("collapse", [2]string{"visibility", "collapse"})
	stat("isolate", [2]string{"isolation", "isolate"})
	stat("isolation-auto", [2]string{"isolation", "auto"})
	stat("box-border", [2]string{"box-sizing", "border-box"})
	stat("box-content", [2]string{"box-sizing", "content-box"})
	for _, v := range []string{"auto", "hidden", "clip", "visible", "scroll"} {
		stat("overflow-"+v, [2]string{"overflow", v})
		stat("overflow-x-"+v, [2]string{"overflow-x", v})
		stat("overflow-y-"+v, [2]string{"overflow-y", v})
	}
	for _, v := range []string{"auto", "contain", "none"} {
		stat("overscroll-"+v, [2]string{"overscroll-behavior", v})
		stat("overscroll-x-"+v, [2]string{"overscroll-behavior-x", v})
		stat("overscroll-y-"+v, [2]string{"overscroll-behavior-y", v})
	}
	stat("float-left", [2]string{"float", "left"})
	stat("float-right", [2]string{"float", "right"})
	stat("float-start", [2]string{"float", "inline-start"})
	stat("float-end", [2]string{"float", "inline-end"})
	stat("float-none", [2]string{"float", "none"})
	stat("clear-left", [2]string{"clear", "left"})
	stat("clear-right", [2]string{"clear", "right"})
	stat("clear-start", [2]string{"clear", "inline-start"})
	stat("clear-end", [2]string{"clear", "inline-end"})
	stat("clear-both", [2]string{"clear", "both"})
	stat("clear-none", [2]string{"clear", "none"})
	stat("sr-only", [2]string{"position", "absolute"}, [2]string{"width", "1px"}, [2]string{"height", "1px"},
		[2]string{"padding", "0"}, [2]string{"margin", "-1px"}, [2]string{"overflow", "hidden"},
		[2]string{"clip-path", "inset(50%)"}, [2]string{"white-space", "nowrap"}, [2]string{"border-width", "0"})
	stat("not-sr-only", [2]string{"position", "static"}, [2]string{"width", "auto"}, [2]string{"height", "auto"},
		[2]string{"padding", "0"}, [2]string{"margin", "0"}, [2]string{"overflow", "visible"},
		[2]string{"clip-path", "none"}, [2]string{"white-space", "normal"})

	for _, inset := range []struct {
		name       string
		properties []string
	}{
		{"inset", []string{"inset"}}, {"inset-x", []string{"inset-inline"}}, {"inset-y", []string{"inset-block"}},
		{"inset-s", []string{"inset-inline-start"}}, {"inset-e", []string{"inset-inline-end"}},
		{"top", []string{"top"}}, {"right", []string{"right"}}, {"bottom", []string{"bottom"}}, {"left", []string{"left"}},
	} {
		stat(inset.name+"-auto", decls(inset.properties, "auto")...)
		stat(inset.name+"-full", decls(inset.properties, "100%")...)
		stat("-"+inset.name+"-full", decls(inset.properties, "-100%")...)
		spacing(inset.name, []string{"--inset", "--spacing"}, multi(inset.properties...), true, true)
	}

	stat("z-auto", [2]string{"z-index", "auto"})
	fn("z", FunctionalUtilityDescription{ThemeKeys: []string{"--z-index"}, SupportsNegative: true, HandleBareValue: BareInteger, Handle: handle(single("z-index"))})
	stat("order-first", [2]string{"order", "-9999"})
	stat("order-last", [2]string{"order", "9999"})
	stat("order-none", [2]string{"order", "0"})
	fn("order", FunctionalUtilityDescription{ThemeKeys: []string{"--order"}, SupportsNegative: true, HandleBareValue: BareInteger, Handle: handle(single("order"))})

	// Flexbox and grid.
	stat("flex-row", [2]string{"flex-direction", "row"})
	stat("flex-row-reverse", [2]string{"flex-direction", "row-reverse"})
	stat("flex-col", [2]string{"flex-direction", "column"})
	stat("flex-col-reverse", [2]string{"flex-direction", "column-reverse"})
	stat("flex-wrap", [2]string{"flex-wrap", "wrap"})
	stat("flex-nowrap", [2]string{"flex-wrap", "nowrap"})
	stat("flex-wrap-reverse", [2]string{"flex-wrap", "wrap-reverse"})
	stat("flex-auto", [2]string{"flex", "auto"})
	stat("flex-initial", [2]string{"flex", "0 auto"})
	stat("flex-none", [2]string{"flex", "none"})
	fn("flex", FunctionalUtilityDescription{ThemeKeys: []string{"--flex"}, NoDefault: true, SupportsFractions: true, HandleBareValue: BareInteger, Handle: handle(single("flex"))})
	fn("grow", FunctionalUtilityDescription{ThemeKeys: []string{"--flex-grow"}, DefaultValue: strPtr("1"), HandleBareValue: BareInteger, Handle: handle(single("flex-grow"))})
	fn("shrink", FunctionalUtilityDescription{ThemeKeys: []string{"--flex-shrink"}, DefaultValue: strPtr("1"), HandleBareValue: BareInteger, Handle: handle(single("flex-shrink"))})
	stat("basis-auto", [2]string{"flex-basis", "auto"})
	stat("basis-full", [2]string{"flex-basis", "100%"})
	spacing("basis", []string{"--flex-basis", "--spacing", "--container"}, single("flex-basis"), false, true)

	for _, g := range [][2]string{{"grid-cols", "grid-template-columns"}, {"grid-rows", "grid-template-rows"}} {
		property := g[1]
		stat(g[0]+"-none", [2]string{property, "none"})
		stat(g[0]+"-subgrid", [2]string{property, "subgrid"})
		fn(g[0], FunctionalUtilityDescription{
			ThemeKeys: []string{"--" + property}, NoDefault: true,
			HandleBareValue: func(v *CandidateValue) (string, bool) {
				return "repeat(" + v.Value + ", minmax(0, 1fr))", IsPositiveInteger(v.Value)
			},
			Handle: handle(single(property)),
		})
	}
	for _, g := range [][2]string{{"col", "column"}, {"row", "row"}} {
		prefix, axis := g[0], g[1]
		stat(prefix+"-span-full", [2]string{"grid-" + axis, "1 / -1"})
		fn(prefix+"-span", FunctionalUtilityDescription{NoDefault: true,
			HandleBareValue: func(v *CandidateValue) (string, bool) {
				return "span " + v.Value + " / span " + v.Value, IsPositiveInteger(v.Value)
			},
			Handle: handle(single("grid-" + axis))})
		stat(prefix+"-start-auto", [2]string{"grid-" + axis + "-start", "auto"})
		fn(prefix+"-start", FunctionalUtilityDescription{ThemeKeys: []string{"--grid-" + axis + "-start"}, NoDefault: true, SupportsNegative: true, HandleBareValue: BareInteger, Handle: handle(single("grid-" + axis + "-start"))})
		stat(prefix+"-end-auto", [2]string{"grid-" + axis + "-end", "auto"})
		fn(prefix+"-end", FunctionalUtilityDescription{ThemeKeys: []string{"--grid-" + axis + "-end"}, NoDefault: true, SupportsNegative: true, HandleBareValue: BareInteger, Handle: handle(single("grid-" + axis + "-end"))})
		stat(prefix+"-auto", [2]string{"grid-" + axis, "auto"})
		fn(prefix, FunctionalUtilityDescription{ThemeKeys: []string{"--grid-" + axis}, NoDefault: true, SupportsNegative: true, HandleBareValue: BareInteger, Handle: handle(single("grid-" + axis))})
	}
	stat("grid-flow-row", [2]string{"grid-auto-flow", "row"})
	stat("grid-flow-col", [2]string{"grid-auto-flow", "column"})
	stat("grid-flow-dense", [2]string{"grid-auto-flow", "dense"})
	stat("grid-flow-row-dense", [2]string{"grid-auto-flow", "row dense"})
	stat("grid-flow-col-dense", [2]string{"grid-auto-flow", "column dense"})
	for _, g := range [][2]string{{"auto-cols", "grid-auto-columns"}, {"auto-rows", "grid-auto-rows"}} {
		property := g[1]
		stat(g[0]+"-auto", [2]string{property, "auto"})
		stat(g[0]+"-min", [2]string{property, "min-content"})
		stat(g[0]+"-max", [2]string{property, "max-content"})
		stat(g[0]+"-fr", [2]string{property, "minmax(0, 1fr)"})
		fn(g[0], FunctionalUtilityDescription{ThemeKeys: []string{"--" + property}, NoDefault: true, Handle: handle(single(property))})
	}
	spacing("gap", []string{"--gap", "--spacing"}, single("gap"), false, false)
	spacing("gap-x", []string{"--gap", "--spacing"}, single("column-gap"), false, false)
	spacing("gap-y", []string{"--gap", "--spacing"}, single("row-gap"), false, false)

	alignments := [][2]string{
		{"normal", "normal"}, {"center", "center"}, {"start", "flex-start"}, {"end", "flex-end"},
		{"between", "space-between"}, {"around", "space-around"}, {"evenly", "space-evenly"},
		{"stretch", "stretch"}, {"baseline", "baseline"},
	}
	for _, a := range alignments {
		stat("justify-"+a[0], [2]string{"justify-content", a[1]})
	}
	for _, a := range alignments {
		name, value := a[0], a[1]
		if name == "between" || name == "around" || name == "evenly" {
			continue
		}
		stat("justify-items-"+name, [2]string{"justify-items", name})
		if name == "start" || name == "end" || name == "center" || name == "stretch" {
			stat("justify-self-"+name, [2]string{"justify-self", name})
		}
		stat("items-"+name, [2]string{"align-items", value})
		if name != "normal" {
			stat("self-"+name, [2]string{"align-self", value})
		}
	}
	stat("justify-self-auto", [2]string{"justify-self", "auto"})
	stat("self-auto", [2]string{"align-self", "auto"})
	for _, a := range alignments {
		stat("content-"+a[0], [2]string{"align-content", a[1]})
	}
	for _, a := range alignments {
		placed := a[1]
		if a[0] == "start" || a[0] == "end" {
			placed = a[0]
		}
		stat("place-content-"+a[0], [2]string{"place-content", placed})
	}
	for _, name := range []string{"start", "end", "center", "baseline", "stretch"} {
		stat("place-items-"+name, [2]string{"place-items", name})
	}
	for _, name := range []string{"auto", "start", "end", "center", "stretch"} {
		stat("place-self-"+name, [2]string{"place-self", name})
	}

	// Spacing.
	for _, p := range [][2]string{
		{"p", "padding"}, {"px", "padding-inline"}, {"py", "padding-block"}, {"ps", "padding-inline-start"},
		{"pe", "padding-inline-end"}, {"pt", "padding-top"}, {"pr", "padding-right"}, {"pb", "padding-bottom"}, {"pl", "padding-left"},
	} {
		spacing(p[0], []string{"--padding", "--spacing"}, single(p[1]), false, false)
	}
	for _, m := range [][2]string{
		{"m", "margin"}, {"mx", "margin-inline"}, {"my", "margin-block"}, {"ms", "margin-inline-start"},
		{"me", "margin-inline-end"}, {"mt", "margin-top"}, {"mr", "margin-right"}, {"mb", "margin-bottom"}, {"ml", "margin-left"},
	} {
		stat(m[0]+"-auto", [2]string{m[1], "auto"})
		spacing(m[0], []string{"--margin", "--spacing"}, single(m[1]), true, false)
	}

	// Sizing.
	widthStatics := [][2]string{{"auto", "auto"}, {"full", "100%"}, {"screen", "100vw"}, {"svw", "100svw"}, {"lvw", "100lvw"}, {"dvw", "100dvw"}, {"min", "min-content"}, {"max", "max-content"}, {"fit", "fit-content"}}
	heightStatics := [][2]string{{"auto", "auto"}, {"full", "100%"}, {"screen", "100vh"}, {"svh", "100svh"}, {"lvh", "100lvh"}, {"dvh", "100dvh"}, {"min", "min-content"}, {"max", "max-content"}, {"fit", "fit-content"}}
	for _, w := range [][3]string{{"w", "width", "--width"}, {"min-w", "min-width", "--min-width"}, {"max-w", "max-width", "--max-width"}} {
		for _, s := range widthStatics {
			if w[0] != "w" && s[0] == "auto" {
				continue
			}
			stat(w[0]+"-"+s[0], [2]string{w[1], s[1]})
		}
		if w[0] == "max-w" {
			stat("max-w-none", [2]string{"max-width", "none"})
		}
		spacing(w[0], []string{w[2], "--spacing", "--container"}, single(w[1]), false, true)
	}
	for _, h := range [][3]string{{"h", "height", "--height"}, {"min-h", "min-height", "--min-height"}, {"max-h", "max-height", "--max-height"}} {
		for _, s := range heightStatics {
			if h[0] != "h" && s[0] == "auto" {
				continue
			}
			stat(h[0]+"-"+s[0], [2]string{h[1], s[1]})
		}
		if h[0] == "max-h" {
			stat("max-h-none", [2]string{"max-height", "none"})
		}
		spacing(h[0], []string{h[2], "--spacing"}, single(h[1]), false, true)
	}
	sizeHandle := func(value string) []Node {
		return []Node{Decl("--tw-sort", "size"), Decl("width", value), Decl("height", value)}
	}
	for _, s := range widthStatics {
		if s[0] == "screen" || s[0] == "svw" || s[0] == "lvw" || s[0] == "dvw" {
			continue
		}
		value := s[1]
		StaticUtilityFn(u, "size-"+s[0], func() []Node { return sizeHandle(value) })
	}
	spacing("size", []string{"--size", "--spacing", "--container"}, sizeHandle, false, true)

	// Typography.
	fontWeight := func(value string) []Node {
		return []Node{Property("--tw-font-weight", nil), Decl("--tw-font-weight", value), Decl("font-weight", value)}
	}
	u.Functional("font", func(c *Candidate) ([]Node, CompileStatus) {
		if c.Value == nil || c.Modifier != nil {
			return nil, NotHandled
		}
		if c.Value.Kind == ValueArbitrary {
			value := c.Value.Value
			t := ""
			if c.Value.DataType != nil {
				t = *c.Value.DataType
			} else {
				t = InferDataType(value, []string{"number", "family-name", "generic-name"})
			}
			switch t {
			case "number":
				return fontWeight(value), Handled
			case "family-name", "generic-name":
				return []Node{Decl("font-family", value)}, Handled
			}
			return nil, NotHandled
		}
		if value, extra, ok := theme.ResolveWith(strPtr(c.Value.Value), []string{"--font"}, []string{"--font-feature-settings", "--font-variation-settings"}); ok {
			nodes := []Node{Decl("font-family", value)}
			if v, ok := extra["--font-feature-settings"]; ok {
				nodes = append(nodes, Decl("font-feature-settings", v))
			}
			if v, ok := extra["--font-variation-settings"]; ok {
				nodes = append(nodes, Decl("font-variation-settings", v))
			}
			return nodes, Handled
		}
		if weight, ok := theme.Resolve(strPtr(c.Value.Value), []string{"--font-weight"}, 0); ok {
			return fontWeight(weight), Handled
		}
		return nil, NotHandled
	}, nil)

	stat("text-left", [2]string{"text-align", "left"})
	stat("text-center", [2]string{"text-align", "center"})
	stat("text-right", [2]string{"text-align", "right"})
	stat("text-justify", [2]string{"text-align", "justify"})
	stat("text-start", [2]string{"text-align", "start"})
	stat("text-end", [2]string{"text-align", "end"})
	stat("text-ellipsis", [2]string{"text-overflow", "ellipsis"})
	stat("text-clip", [2]string{"text-overflow", "clip"})
	stat("text-wrap", [2]string{"text-wrap", "wrap"})
	stat("text-nowrap", [2]string{"text-wrap", "nowrap"})
	stat("text-balance", [2]string{"text-wrap", "balance"})
	stat("text-pretty", [2]string{"text-wrap", "pretty"})

	// resolveLeading returns (value, present, ok): ok is false when a modifier exists but cannot resolve.
	resolveLeading := func(c *Candidate) (string, bool, bool) {
		if c.Modifier == nil {
			return "", false, true
		}
		if c.Modifier.Kind == ValueArbitrary {
			return c.Modifier.Value, true, true
		}
		if leading, ok := theme.Resolve(strPtr(c.Modifier.Value), []string{"--leading"}, 0); ok {
			return leading, true, true
		}
		if IsMultipleOfQuarter(c.Modifier.Value) {
			if _, ok := theme.Resolve(nil, []string{"--spacing"}, 0); ok {
				return "--spacing(" + c.Modifier.Value + ")", true, true
			}
		}
		if c.Modifier.Value == "none" {
			return "1", true, true
		}
		return "", false, false
	}

	u.Functional("text", func(c *Candidate) ([]Node, CompileStatus) {
		if c.Value == nil {
			return nil, NotHandled
		}
		if c.Value.Kind == ValueArbitrary {
			value := c.Value.Value
			t := ""
			if c.Value.DataType != nil {
				t = *c.Value.DataType
			} else {
				t = InferDataType(value, []string{"color", "length", "percentage", "absolute-size", "relative-size"})
			}
			if t == "color" || t == "" {
				resolved, ok := AsColor(value, c.Modifier, theme)
				if !ok {
					return nil, NotHandled
				}
				return []Node{Decl("color", resolved)}, Handled
			}
			nodes := []Node{Decl("font-size", value)}
			leading, present, ok := resolveLeading(c)
			if !ok {
				return nil, NotHandled
			}
			if present {
				nodes = append(nodes, Decl("line-height", leading))
			}
			return nodes, Handled
		}
		if asColor, ok := ResolveThemeColor(c, []string{"--text-color", "--color"}, theme); ok {
			return []Node{Decl("color", asColor)}, Handled
		}
		value, extra, ok := theme.ResolveWith(strPtr(c.Value.Value), []string{"--text"}, []string{"--line-height", "--letter-spacing", "--font-weight"})
		if !ok {
			return nil, NotHandled
		}
		nodes := []Node{Decl("font-size", value)}
		leading, present, ok := resolveLeading(c)
		if !ok {
			return nil, NotHandled
		}
		if present {
			nodes = append(nodes, Decl("line-height", leading))
		} else if lh, ok := extra["--line-height"]; ok {
			nodes = append(nodes, Decl("line-height", "var(--tw-leading, "+lh+")"))
		}
		if ls, ok := extra["--letter-spacing"]; ok {
			nodes = append(nodes, Decl("letter-spacing", "var(--tw-tracking, "+ls+")"))
		}
		if fw, ok := extra["--font-weight"]; ok {
			nodes = append(nodes, Decl("font-weight", "var(--tw-font-weight, "+fw+")"))
		}
		return nodes, Handled
	}, nil)

	leadingHandle := func(value string) []Node {
		return []Node{Property("--tw-leading", nil), Decl("--tw-leading", value), Decl("line-height", value)}
	}
	StaticUtilityFn(u, "leading-none", func() []Node { return leadingHandle("1") })
	spacing("leading", []string{"--leading", "--spacing"}, leadingHandle, false, false)
	fn("tracking", FunctionalUtilityDescription{ThemeKeys: []string{"--tracking"}, SupportsNegative: true, Handle: handle(func(value string) []Node {
		return []Node{Property("--tw-tracking", nil), Decl("--tw-tracking", value), Decl("letter-spacing", value)}
	})})

	stat("uppercase", [2]string{"text-transform", "uppercase"})
	stat("lowercase", [2]string{"text-transform", "lowercase"})
	stat("capitalize", [2]string{"text-transform", "capitalize"})
	stat("normal-case", [2]string{"text-transform", "none"})
	stat("italic", [2]string{"font-style", "italic"})
	stat("not-italic", [2]string{"font-style", "normal"})
	stat("underline", [2]string{"text-decoration-line", "underline"})
	stat("overline", [2]string{"text-decoration-line", "overline"})
	stat("line-through", [2]string{"text-decoration-line", "line-through"})
	stat("no-underline", [2]string{"text-decoration-line", "none"})
	stat("truncate", [2]string{"overflow", "hidden"}, [2]string{"text-overflow", "ellipsis"}, [2]string{"white-space", "nowrap"})
	for _, v := range []string{"normal", "nowrap", "pre", "pre-line", "pre-wrap", "break-spaces"} {
		stat("whitespace-"+v, [2]string{"white-space", v})
	}
	stat("break-normal", [2]string{"overflow-wrap", "normal"}, [2]string{"word-break", "normal"})
	stat("break-words", [2]string{"overflow-wrap", "break-word"})
	stat("break-all", [2]string{"word-break", "break-all"})
	stat("break-keep", [2]string{"word-break", "keep-all"})
	stat("list-none", [2]string{"list-style-type", "none"})
	stat("list-disc", [2]string{"list-style-type", "disc"})
	stat("list-decimal", [2]string{"list-style-type", "decimal"})
	stat("list-inside", [2]string{"list-style-position", "inside"})
	stat("list-outside", [2]string{"list-style-position", "outside"})
	stat("antialiased", [2]string{"-webkit-font-smoothing", "antialiased"}, [2]string{"-moz-osx-font-smoothing", "grayscale"})
	stat("subpixel-antialiased", [2]string{"-webkit-font-smoothing", "auto"}, [2]string{"-moz-osx-font-smoothing", "auto"})
	stat("underline-offset-auto", [2]string{"text-underline-offset", "auto"})
	fn("underline-offset", FunctionalUtilityDescription{
		ThemeKeys: []string{"--text-underline-offset"}, SupportsNegative: true,
		HandleBareValue: bareSuffix("px"),
		HandleNegativeBareValue: func(v *CandidateValue) (string, bool) {
			return "-" + v.Value + "px", IsPositiveInteger(v.Value)
		},
		Handle: handle(single("text-underline-offset")),
	})
	spacing("indent", []string{"--text-indent", "--spacing"}, single("text-indent"), true, false)

	// Backgrounds and borders.
	u.Functional("bg", func(c *Candidate) ([]Node, CompileStatus) {
		if c.Value == nil {
			return nil, NotHandled
		}
		if c.Value.Kind == ValueArbitrary {
			value := c.Value.Value
			t := ""
			if c.Value.DataType != nil {
				t = *c.Value.DataType
			} else {
				t = InferDataType(value, []string{"image", "color", "percentage", "position", "bg-size", "length", "url"})
			}
			switch t {
			case "percentage", "position":
				if c.Modifier != nil {
					return nil, NotHandled
				}
				return []Node{Decl("background-position", value)}, Handled
			case "bg-size", "length":
				if c.Modifier != nil {
					return nil, NotHandled
				}
				return []Node{Decl("background-size", value)}, Handled
			case "image", "url":
				if c.Modifier != nil {
					return nil, NotHandled
				}
				return []Node{Decl("background-image", value)}, Handled
			}
			resolved, ok := AsColor(value, c.Modifier, theme)
			if !ok {
				return nil, NotHandled
			}
			return []Node{Decl("background-color", resolved)}, Handled
		}
		if asColor, ok := ResolveThemeColor(c, []string{"--background-color", "--color"}, theme); ok {
			return []Node{Decl("background-color", asColor)}, Handled
		}
		if image, ok := theme.Resolve(strPtr(c.Value.Value), []string{"--background-image"}, 0); ok {
			if c.Modifier != nil {
				return nil, NotHandled
			}
			return []Node{Decl("background-image", image)}, Handled
		}
		return nil, NotHandled
	}, nil)
	stat("bg-auto", [2]string{"background-size", "auto"})
	stat("bg-cover", [2]string{"background-size", "cover"})
	stat("bg-contain", [2]string{"background-size", "contain"})
	stat("bg-fixed", [2]string{"background-attachment", "fixed"})
	stat("bg-local", [2]string{"background-attachment", "local"})
	stat("bg-scroll", [2]string{"background-attachment", "scroll"})
	for _, p := range []string{"top", "center", "bottom", "left", "right"} {
		stat("bg-"+p, [2]string{"background-position", p})
	}
	stat("bg-top-left", [2]string{"background-position", "left top"})
	stat("bg-top-right", [2]string{"background-position", "right top"})
	stat("bg-bottom-left", [2]string{"background-position", "left bottom"})
	stat("bg-bottom-right", [2]string{"background-position", "right bottom"})
	stat("bg-repeat", [2]string{"background-repeat", "repeat"})
	stat("bg-no-repeat", [2]string{"background-repeat", "no-repeat"})
	stat("bg-repeat-x", [2]string{"background-repeat", "repeat-x"})
	stat("bg-repeat-y", [2]string{"background-repeat", "repeat-y"})
	stat("bg-repeat-round", [2]string{"background-repeat", "round"})
	stat("bg-repeat-space", [2]string{"background-repeat", "space"})
	stat("bg-none", [2]string{"background-image", "none"})
	for _, v := range []string{"border", "padding", "content"} {
		stat("bg-clip-"+v, [2]string{"background-clip", v + "-box"})
		stat("bg-origin-"+v, [2]string{"background-origin", v + "-box"})
	}
	stat("bg-clip-text", [2]string{"background-clip", "text"})

	for _, b := range [][3]string{
		{"border", "border-width", "border-color"}, {"border-x", "border-inline-width", "border-inline-color"},
		{"border-y", "border-block-width", "border-block-color"}, {"border-s", "border-inline-start-width", "border-inline-start-color"},
		{"border-e", "border-inline-end-width", "border-inline-end-color"}, {"border-t", "border-top-width", "border-top-color"},
		{"border-r", "border-right-width", "border-right-color"}, {"border-b", "border-bottom-width", "border-bottom-color"},
		{"border-l", "border-left-width", "border-left-color"},
	} {
		widthProperty, colorProperty := b[1], b[2]
		width := func(value string) []Node {
			return []Node{Property("--tw-border-style", strPtr("solid")), Decl("border-style", "var(--tw-border-style)"), Decl(widthProperty, value)}
		}
		u.Functional(b[0], func(c *Candidate) ([]Node, CompileStatus) {
			if c.Value == nil {
				if c.Modifier != nil {
					return nil, NotHandled
				}
				w, ok := theme.Get("--default-border-width")
				if !ok {
					w = "1px"
				}
				return width(w), Handled
			}
			if c.Value.Kind == ValueArbitrary {
				value := c.Value.Value
				t := ""
				if c.Value.DataType != nil {
					t = *c.Value.DataType
				} else {
					t = InferDataType(value, []string{"color", "line-width", "length"})
				}
				switch t {
				case "color":
					resolved, ok := AsColor(value, c.Modifier, theme)
					if !ok {
						return nil, NotHandled
					}
					return []Node{Decl(colorProperty, resolved)}, Handled
				case "line-width", "length":
					if c.Modifier != nil {
						return nil, NotHandled
					}
					return width(value), Handled
				}
				return nil, NotHandled
			}
			if asColor, ok := ResolveThemeColor(c, []string{"--border-color", "--color"}, theme); ok {
				return []Node{Decl(colorProperty, asColor)}, Handled
			}
			if c.Modifier != nil {
				return nil, NotHandled
			}
			if w, ok := theme.Resolve(strPtr(c.Value.Value), []string{"--border-width"}, 0); ok {
				return width(w), Handled
			}
			if IsPositiveInteger(c.Value.Value) {
				return width(c.Value.Value + "px"), Handled
			}
			return nil, NotHandled
		}, nil)
	}
	for _, style := range []string{"solid", "dashed", "dotted", "double", "hidden", "none"} {
		stat("border-"+style, [2]string{"--tw-border-style", style}, [2]string{"border-style", style})
	}

	for _, r := range []struct {
		root       string
		properties []string
	}{
		{"rounded", []string{"border-radius"}},
		{"rounded-s", []string{"border-start-start-radius", "border-end-start-radius"}},
		{"rounded-e", []string{"border-start-end-radius", "border-end-end-radius"}},
		{"rounded-t", []string{"border-top-left-radius", "border-top-right-radius"}},
		{"rounded-r", []string{"border-top-right-radius", "border-bottom-right-radius"}},
		{"rounded-b", []string{"border-bottom-right-radius", "border-bottom-left-radius"}},
		{"rounded-l", []string{"border-top-left-radius", "border-bottom-left-radius"}},
		{"rounded-ss", []string{"border-start-start-radius"}}, {"rounded-se", []string{"border-start-end-radius"}},
		{"rounded-ee", []string{"border-end-end-radius"}}, {"rounded-es", []string{"border-end-start-radius"}},
		{"rounded-tl", []string{"border-top-left-radius"}}, {"rounded-tr", []string{"border-top-right-radius"}},
		{"rounded-br", []string{"border-bottom-right-radius"}}, {"rounded-bl", []string{"border-bottom-left-radius"}},
	} {
		stat(r.root+"-none", decls(r.properties, "0")...)
		stat(r.root+"-full", decls(r.properties, "calc(infinity * 1px)")...)
		fn(r.root, FunctionalUtilityDescription{ThemeKeys: []string{"--radius"}, Handle: handle(multi(r.properties...))})
	}

	// Shadows and rings.
	RegisterShadowUtilities(u, theme)

	// Transforms.
	RegisterTransformUtilities(u, theme)

	// Filters.
	RegisterFilterUtilities(u, theme)

	// Gradients.
	RegisterGradientUtilities(u, theme)

	// Effects, transitions, interactivity.
	fn("opacity", FunctionalUtilityDescription{ThemeKeys: []string{"--opacity"},
		HandleBareValue: func(v *CandidateValue) (string, bool) { return v.Value + "%", IsMultipleOfQuarter(v.Value) },
		Handle:          handle(single("opacity"))})
	transitionTail := [][2]string{
		{"transition-timing-function", "var(--default-transition-timing-function)"},
		{"transition-duration", "var(--default-transition-duration)"},
	}
	withTail := func(property string) []Node {
		return []Node{Decl("transition-property", property), Decl(transitionTail[0][0], transitionTail[0][1]), Decl(transitionTail[1][0], transitionTail[1][1])}
	}
	transitionProperties := strings.Join([]string{
		"color", "background-color", "border-color", "outline-color", "text-decoration-color", "fill", "stroke",
		"--tw-gradient-from", "--tw-gradient-via", "--tw-gradient-to", "opacity", "box-shadow", "transform",
		"translate", "scale", "rotate", "filter", "-webkit-backdrop-filter", "backdrop-filter", "display",
		"content-visibility", "overlay", "pointer-events",
	}, ", ")
	StaticUtilityFn(u, "transition", func() []Node { return withTail(transitionProperties) })
	stat("transition-none", [2]string{"transition-property", "none"})
	StaticUtilityFn(u, "transition-all", func() []Node { return withTail("all") })
	StaticUtilityFn(u, "transition-colors", func() []Node {
		return withTail("color, background-color, border-color, outline-color, text-decoration-color, fill, stroke, --tw-gradient-from, --tw-gradient-via, --tw-gradient-to")
	})
	StaticUtilityFn(u, "transition-opacity", func() []Node { return withTail("opacity") })
	StaticUtilityFn(u, "transition-shadow", func() []Node { return withTail("box-shadow") })
	StaticUtilityFn(u, "transition-transform", func() []Node { return withTail("transform, translate, scale, rotate") })
	stat("transition-discrete", [2]string{"transition-behavior", "allow-discrete"})
	stat("transition-normal", [2]string{"transition-behavior", "normal"})
	fn("duration", FunctionalUtilityDescription{ThemeKeys: []string{"--transition-duration"}, HandleBareValue: bareSuffix("ms"), Handle: handle(single("transition-duration"))})
	fn("delay", FunctionalUtilityDescription{ThemeKeys: []string{"--transition-delay"}, HandleBareValue: bareSuffix("ms"), Handle: handle(single("transition-delay"))})
	stat("ease-linear", [2]string{"transition-timing-function", "linear"})
	stat("ease-initial", [2]string{"transition-timing-function", "initial"})
	fn("ease", FunctionalUtilityDescription{ThemeKeys: []string{"--ease"}, Handle: handle(single("transition-timing-function"))})
	stat("animate-none", [2]string{"animation", "none"})
	fn("animate", FunctionalUtilityDescription{ThemeKeys: []string{"--animate"}, Handle: handle(single("animation"))})

	for _, v := range strings.Fields(`auto default pointer wait text move help not-allowed none context-menu progress cell crosshair
vertical-text alias copy no-drop grab grabbing all-scroll col-resize row-resize n-resize e-resize s-resize w-resize
ne-resize nw-resize se-resize sw-resize ew-resize ns-resize nesw-resize nwse-resize zoom-in zoom-out`) {
		stat("cursor-"+v, [2]string{"cursor", v})
	}
	fn("cursor", FunctionalUtilityDescription{ThemeKeys: []string{"--cursor"}, Handle: handle(single("cursor"))})
	for _, v := range []string{"none", "text", "all", "auto"} {
		stat("select-"+v, [2]string{"-webkit-user-select", v}, [2]string{"user-select", v})
	}
	stat("pointer-events-none", [2]string{"pointer-events", "none"})
	stat("pointer-events-auto", [2]string{"pointer-events", "auto"})
	stat("resize", [2]string{"resize", "both"})
	stat("resize-none", [2]string{"resize", "none"})
	stat("resize-x", [2]string{"resize", "horizontal"})
	stat("resize-y", [2]string{"resize", "vertical"})
	stat("appearance-none", [2]string{"appearance", "none"})
	stat("appearance-auto", [2]string{"appearance", "auto"})
	stat("scroll-auto", [2]string{"scroll-behavior", "auto"})
	stat("scroll-smooth", [2]string{"scroll-behavior", "smooth"})
	stat("will-change-auto", [2]string{"will-change", "auto"})
	stat("will-change-scroll", [2]string{"will-change", "scroll-position"})
	stat("will-change-contents", [2]string{"will-change", "contents"})
	stat("will-change-transform", [2]string{"will-change", "transform"})
	fn("will-change", FunctionalUtilityDescription{ThemeKeys: []string{"--will-change"}, Handle: handle(single("will-change"))})

	u.Functional("content", func(c *Candidate) ([]Node, CompileStatus) {
		if c.Value == nil || c.Value.Kind != ValueArbitrary || c.Modifier != nil {
			return nil, NotHandled
		}
		return []Node{Property("--tw-content", strPtr(`""`)), Decl("--tw-content", c.Value.Value), Decl("content", "var(--tw-content)")}, Handled
	}, nil)

	stat("aspect-square", [2]string{"aspect-ratio", "1 / 1"})
	stat("aspect-auto", [2]string{"aspect-ratio", "auto"})
	fn("aspect", FunctionalUtilityDescription{ThemeKeys: []string{"--aspect"},
		HandleBareValue: func(v *CandidateValue) (string, bool) {
			if v.Fraction == nil {
				return "", false
			}
			parts := strings.SplitN(*v.Fraction, "/", 2)
			if !IsPositiveInteger(parts[0]) || !IsPositiveInteger(parts[1]) {
				return "", false
			}
			return parts[0] + " / " + parts[1], true
		},
		Handle: handle(single("aspect-ratio"))})
	stat("columns-auto", [2]string{"columns", "auto"})
	fn("columns", FunctionalUtilityDescription{ThemeKeys: []string{"--columns", "--container"}, HandleBareValue: BareInteger, Handle: handle(single("columns"))})
	for _, v := range []string{"contain", "cover", "fill", "none", "scale-down"} {
		stat("object-"+v, [2]string{"object-fit", v})
	}
	for _, p := range []string{"top", "center", "bottom", "left", "right"} {
		stat("object-"+p, [2]string{"object-position", p})
	}
	stat("accent-auto", [2]string{"accent-color", "auto"})
	color("accent", []string{"--accent-color", "--color"}, single("accent-color"))
	color("caret", []string{"--caret-color", "--color"}, single("caret-color"))
	stat("fill-none", [2]string{"fill", "none"})
	color("fill", []string{"--fill", "--color"}, single("fill"))
	stat("stroke-none", [2]string{"stroke", "none"})
	color("stroke", []string{"--stroke", "--color"}, single("stroke"))
}
