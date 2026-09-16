package twill

import "strings"

// SourceEntry is a source collected from @source (SPEC §4.1.10).
type SourceEntry struct {
	Base    string
	Pattern string
	Negated bool
}

// SourceRoot is the root of a compiler handle: nil, "none", or an entry.
type SourceRoot struct {
	None    bool
	Base    string
	Pattern string
}

// CollectedDirectives is the result of collecting directives.
type CollectedDirectives struct {
	Features          Features
	Important         bool
	Sources           []SourceEntry
	Root              *SourceRoot
	InlineCandidates  []string
	IgnoredCandidates []string
	UtilitiesNode     *AtRule
	FirstThemeRule    *Rule
	CustomVariants    []*CustomVariant
	CustomUtilities   []*CustomUtility
}

func isTopLevelPath(path []Node) bool {
	for _, p := range path {
		if _, ok := p.(*Context); !ok {
			return false
		}
	}
	return true
}

// CollectDirectives walks the AST once and registers @theme, @source,
// @twill utilities, @custom-variant, and @utility, interpreting the
// media-position import parameters (SPEC §6.1 step 2, §6.4 through §6.8).
func CollectDirectives(ast *[]Node, theme *Theme) (*CollectedDirectives, error) {
	state := &CollectedDirectives{Sources: []SourceEntry{}}
	var failure error
	fail := func(err error) WalkAction {
		failure = err
		return Stop
	}

	Walk(ast, func(n Node, u *WalkUtils) WalkAction {
		at, ok := n.(*AtRule)
		if !ok {
			return Continue
		}
		switch {
		case at.Name == "@media":
			action, err := handleImportMedia(at, u, state)
			if err != nil {
				return fail(err)
			}
			return action

		case at.Name == "@custom-variant" || (at.Name == "@variant" && isTopLevelPath(u.Path) && IsCustomVariantCompat(at)):
			if !isTopLevelPath(u.Path) {
				return fail(Errorf("`%s %s` cannot be nested.", at.Name, at.Params))
			}
			variant, err := ParseCustomVariant(at)
			if err != nil {
				return fail(err)
			}
			state.CustomVariants = append(state.CustomVariants, variant)
			u.ReplaceWith()
			return Continue

		case at.Name == "@utility":
			if !isTopLevelPath(u.Path) {
				return fail(Errorf("`@utility %s` cannot be nested.", at.Params))
			}
			utility, err := ParseUtilityDefinition(at)
			if err != nil {
				return fail(err)
			}
			state.CustomUtilities = append(state.CustomUtilities, utility)
			return Skip

		case at.Name == "@twill":
			if !strings.HasPrefix(at.Params, "utilities") {
				return Continue
			}
			if u.Context.Bool("reference") || state.UtilitiesNode != nil {
				u.ReplaceWith()
				return Continue
			}
			source, isNone, err := parseUtilitiesSource(at.Params)
			if err != nil {
				return fail(err)
			}
			if isNone {
				state.Root = &SourceRoot{None: true}
			} else if source != "" {
				base := u.Context["sourceBase"]
				if base == "" {
					base = u.Context["base"]
				}
				state.Root = &SourceRoot{Base: base, Pattern: source}
			}
			state.UtilitiesNode = at
			state.Features |= FeatureUtilities
			return Skip

		case at.Name == "@theme":
			options, err := parseThemeOptions(at.Params, u.Context.Bool("reference"), theme)
			if err != nil {
				return fail(err)
			}
			for _, child := range at.Nodes {
				switch child := child.(type) {
				case *Comment:
					continue
				case *Declaration:
					if strings.HasPrefix(child.Property, "--") {
						if err := theme.Add(Unescape(child.Property), child.Value, options); err != nil {
							return fail(err)
						}
						continue
					}
				case *AtRule:
					if child.Name == "@keyframes" {
						if options&Reference == 0 {
							theme.AddKeyframes(child)
						}
						continue
					}
				}
				lines := strings.SplitN(Serialize([]Node{child}), "\n", 4)
				if len(lines) > 3 {
					lines = lines[:3]
				}
				return fail(Errorf("`@theme` blocks must only contain custom properties or `@keyframes`.\n\n%s", strings.Join(lines, "\n")))
			}
			state.Features |= FeatureAtTheme
			if state.FirstThemeRule == nil && options&Reference == 0 {
				state.FirstThemeRule = StyleRule(":root, :host")
				u.ReplaceWith(state.FirstThemeRule)
			} else {
				u.ReplaceWith()
			}
			return Continue

		case at.Name == "@source":
			if len(at.Nodes) > 0 {
				return fail(Errorf("`@source` cannot have a body."))
			}
			for _, ancestor := range u.Path {
				switch ancestor.(type) {
				case *Rule, *AtRule:
					return fail(Errorf("`@source` cannot be nested."))
				}
			}
			if err := handleSource(at, u.Context["base"], state); err != nil {
				return fail(err)
			}
			u.ReplaceWith()
			return Continue
		}
		return Continue
	})
	if failure != nil {
		return nil, failure
	}
	return state, nil
}

func parseUtilitiesSource(params string) (path string, none bool, err error) {
	rest := strings.TrimSpace(params[len("utilities"):])
	if rest == "" {
		return "", false, nil
	}
	if !strings.HasPrefix(rest, "source(") || !strings.HasSuffix(rest, ")") {
		return "", false, Errorf("Invalid `@twill %s`", params)
	}
	inner := strings.TrimSpace(rest[len("source(") : len(rest)-1])
	if inner == "none" {
		return "", true, nil
	}
	quoted, ok := Unquote(inner)
	if !ok {
		return "", false, Errorf("`source(%s)` paths must be quoted.\n\nInstead use:\n@twill utilities source(\"%s\");", inner, inner)
	}
	return quoted, false, nil
}

func parseThemeOptions(params string, inReference bool, theme *Theme) (ThemeOptions, error) {
	var options ThemeOptions
	if inReference {
		options |= Reference
	}
	for _, option := range Segment(params, ' ') {
		switch {
		case option == "":
		case option == "reference":
			options |= Reference
		case option == "inline":
			options |= Inline
		case option == "default":
			options |= Default
		case option == "static":
			options |= Static
		case strings.HasPrefix(option, "prefix(") && strings.HasSuffix(option, ")"):
			prefix := option[len("prefix(") : len(option)-1]
			if !IsValidPrefix(prefix) {
				return 0, Errorf("The prefix \"%s\" is invalid. Prefixes must be alphabetic and lowercase.", prefix)
			}
			theme.Prefix = prefix
		default:
			return 0, Errorf("Unknown `@theme` option `%s`", option)
		}
	}
	return options, nil
}

func handleSource(node *AtRule, base string, state *CollectedDirectives) error {
	params := strings.TrimSpace(node.Params)
	negated := false
	if strings.HasPrefix(params, "not ") {
		negated = true
		params = strings.TrimSpace(params[4:])
	}
	if strings.HasPrefix(params, "inline(") {
		if !strings.HasSuffix(params, ")") {
			return Errorf("Invalid `@source %s`", node.Params)
		}
		inner, ok := Unquote(strings.TrimSpace(params[len("inline(") : len(params)-1]))
		if !ok {
			return Errorf("`@source inline(...)` patterns must be quoted.")
		}
		for _, item := range strings.Fields(inner) {
			expanded, err := ExpandBraces(item)
			if err != nil {
				return err
			}
			if negated {
				state.IgnoredCandidates = append(state.IgnoredCandidates, expanded...)
			} else {
				state.InlineCandidates = append(state.InlineCandidates, expanded...)
			}
		}
		return nil
	}
	pattern, ok := Unquote(params)
	if !ok {
		prefix := ""
		if negated {
			prefix = "not "
		}
		return Errorf("`@source` paths must be quoted.\n\nInstead use:\n@source %s\"%s\";", prefix, params)
	}
	state.Sources = append(state.Sources, SourceEntry{Base: base, Pattern: pattern, Negated: negated})
	return nil
}

// handleImportMedia interprets the media-position import parameters.
func handleImportMedia(node *AtRule, u *WalkUtils, state *CollectedDirectives) (WalkAction, error) {
	var remaining []string
	consumed := false
	for _, param := range Segment(node.Params, ' ') {
		switch {
		case param == "":
		case param == "reference":
			node.Nodes = []Node{NewContext(ContextMap{"reference": "true"}, node.Nodes...)}
			consumed = true
		case strings.HasPrefix(param, "theme(") && strings.HasSuffix(param, ")"):
			opts := strings.TrimSpace(param[len("theme(") : len(param)-1])
			isReference := false
			for _, o := range Segment(opts, ' ') {
				if o == "reference" {
					isReference = true
				}
			}
			var failure error
			Walk(&node.Nodes, func(child Node, _ *WalkUtils) WalkAction {
				switch child := child.(type) {
				case *AtRule:
					if child.Name == "@theme" {
						child.Params = strings.TrimSpace(child.Params + " " + opts)
						return Skip
					}
					if child.Name == "@layer" || child.Name == "@media" {
						return Continue
					}
				case *Context, *Comment:
					return Continue
				}
				if isReference {
					failure = Errorf("Importing a stylesheet with `theme(reference)` is only allowed for stylesheets that contain only `@theme` blocks.")
					return Stop
				}
				return Skip
			})
			if failure != nil {
				return Stop, failure
			}
			consumed = true
		case strings.HasPrefix(param, "prefix(") && strings.HasSuffix(param, ")"):
			p := param
			Walk(&node.Nodes, func(child Node, _ *WalkUtils) WalkAction {
				if at, ok := child.(*AtRule); ok && at.Name == "@theme" {
					at.Params = strings.TrimSpace(at.Params + " " + p)
					return Skip
				}
				return Continue
			})
			consumed = true
		case param == "important":
			state.Important = true
			consumed = true
		case strings.HasPrefix(param, "source(") && strings.HasSuffix(param, ")"):
			sourceBase := u.Context["base"]
			p := param
			Walk(&node.Nodes, func(child Node, cu *WalkUtils) WalkAction {
				if at, ok := child.(*AtRule); ok && at.Name == "@twill" && at.Params == "utilities" {
					at.Params = "utilities " + p
					cu.ReplaceWith(NewContext(ContextMap{"sourceBase": sourceBase}, at))
					return Stop
				}
				return Continue
			})
			consumed = true
		default:
			remaining = append(remaining, param)
		}
	}
	if !consumed {
		return Continue, nil
	}
	if len(remaining) == 0 {
		u.ReplaceWith(node.Nodes...)
	} else {
		node.Params = strings.Join(remaining, " ")
	}
	return Continue, nil
}
