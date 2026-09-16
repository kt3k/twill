package twill

import (
	"regexp"
	"strings"
)

var themeFunctionPattern = regexp.MustCompile(`(^|[^\w-])(?:--spacing|--alpha|--theme|theme)\(`)

// HasThemeFunction reports whether a value contains a theme function call.
func HasThemeFunction(value string) bool { return themeFunctionPattern.MatchString(value) }

var atRulesWithParams = map[string]bool{"@media": true, "@custom-media": true, "@container": true, "@supports": true}

// SubstituteFunctions substitutes theme functions in declaration values and
// in the params of @media, @custom-media, @container, and @supports
// (SPEC §6.11).
func SubstituteFunctions(ast *[]Node, ds *DesignSystem) (Features, error) {
	var features Features
	var failure error
	Walk(ast, func(n Node, _ *WalkUtils) WalkAction {
		switch n := n.(type) {
		case *Declaration:
			if !n.NoValue && HasThemeFunction(n.Value) {
				value, err := SubstituteFunctionsInValue(n.Value, ds, false)
				if err != nil {
					failure = err
					return Stop
				}
				n.Value = value
				features |= FeatureThemeFunction
			}
		case *AtRule:
			if atRulesWithParams[n.Name] && HasThemeFunction(n.Params) {
				params, err := SubstituteFunctionsInValue(n.Params, ds, true)
				if err != nil {
					failure = err
					return Stop
				}
				n.Params = params
				features |= FeatureThemeFunction
			}
		}
		return Continue
	})
	return features, failure
}

// SubstituteFunctionsInValue substitutes theme functions in one value.
func SubstituteFunctionsInValue(value string, ds *DesignSystem, inAtRule bool) (string, error) {
	ast := ParseValue(value)
	var failure error
	WalkValue(&ast, func(n ValueNode, _ *ValueFunction) ([]ValueNode, bool, bool) {
		f, ok := n.(*ValueFunction)
		if !ok || failure != nil {
			return nil, false, false
		}
		var result string
		var err error
		switch f.Name {
		case "--spacing":
			result, err = spacingFunction(f.Nodes, ds)
		case "--alpha":
			result, err = alphaFunction(f.Nodes)
		case "--theme":
			result, err = themeFunction(f.Nodes, ds, inAtRule)
		case "theme":
			result, err = legacyThemeFunction(f.Nodes, ds)
		default:
			return nil, false, false
		}
		if err != nil {
			failure = err
			return nil, false, true
		}
		return []ValueNode{Word(result)}, true, true
	})
	if failure != nil {
		return "", failure
	}
	return ValueToCSS(ast), nil
}

func functionArgs(nodes []ValueNode) []string {
	parts := Segment(ValueToCSS(nodes), ',')
	for i, p := range parts {
		parts[i] = strings.TrimSpace(p)
	}
	return parts
}

func spacingFunction(nodes []ValueNode, ds *DesignSystem) (string, error) {
	args := functionArgs(nodes)
	if len(args) != 1 || args[0] == "" {
		count := 0
		for _, a := range args {
			if a != "" {
				count++
			}
		}
		return "", Errorf("The --spacing(…) function requires exactly one argument, but received %d.", count)
	}
	multiplier, ok := ds.Theme.Resolve(nil, []string{"--spacing"}, 0)
	if !ok {
		return "", Errorf("The --spacing(…) function requires that the `--spacing` theme variable exists, but it was not found.")
	}
	switch args[0] {
	case "0":
		return "0px", nil
	case "1":
		return multiplier, nil
	}
	return "calc(" + multiplier + " * " + args[0] + ")", nil
}

func alphaFunction(nodes []ValueNode) (string, error) {
	text := ValueToCSS(nodes)
	if len(Segment(text, ',')) != 1 {
		return "", Errorf("The --alpha(…) function requires exactly one argument in the form `<color> / <alpha>`, but received `%s`.", text)
	}
	parts := Segment(text, '/')
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
		return "", Errorf("The --alpha(…) function requires a color and an alpha value separated by `/`, but received `%s`.", text)
	}
	return WithAlpha(strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])), nil
}

func themeFunction(nodes []ValueNode, ds *DesignSystem, inAtRule bool) (string, error) {
	args := functionArgs(nodes)
	key := ""
	if len(args) > 0 {
		key = args[0]
	}
	inline := inAtRule
	if strings.HasSuffix(key, " inline") {
		inline = true
		key = strings.TrimSpace(key[:len(key)-len(" inline")])
	}
	fallback := ""
	hasFallback := len(args) > 1
	if hasFallback {
		fallback = strings.Join(args[1:], ", ")
	}
	if !strings.HasPrefix(key, "--") {
		return "", Errorf("The --theme(…) function can only be used with CSS variables from your theme, but received `%s`.", key)
	}
	resolved, ok := ds.Theme.ResolveThemeValue(key, inline)
	if !ok {
		if hasFallback {
			return fallback, nil
		}
		return "", Errorf("Could not resolve value for theme function: `--theme(%s)`.", key)
	}
	if !hasFallback || fallback == "initial" {
		return resolved, nil
	}
	if resolved == "initial" {
		return fallback, nil
	}
	if strings.HasPrefix(resolved, "var(") || strings.HasPrefix(resolved, "theme(") || strings.HasPrefix(resolved, "--theme(") {
		return InjectFallback(resolved, fallback), nil
	}
	return resolved, nil
}

func legacyThemeFunction(nodes []ValueNode, ds *DesignSystem) (string, error) {
	args := functionArgs(nodes)
	key := ""
	if len(args) > 0 {
		key = args[0]
	}
	if unquoted, ok := Unquote(key); ok {
		key = unquoted
	}
	resolved, ok := ds.Theme.ResolveThemeValue(key, true)
	if !ok {
		if len(args) > 1 {
			return strings.Join(args[1:], ", "), nil
		}
		return "", Errorf("Could not resolve value for theme function: `theme(%s)`.", key)
	}
	return resolved, nil
}

func isThemeReference(name string) bool { return name == "var" || name == "theme" || name == "--theme" }

// InjectFallback injects fallback into the innermost var(...), theme(...),
// or --theme(...) call that has no fallback or whose fallback is `initial`.
func InjectFallback(value, fallback string) string {
	ast := ParseValue(value)
	injectFallback(ast, fallback)
	return ValueToCSS(ast)
}

func injectFallback(nodes []ValueNode, fallback string) bool {
	for _, n := range nodes {
		f, ok := n.(*ValueFunction)
		if !ok {
			continue
		}
		if injectFallback(f.Nodes, fallback) {
			return true
		}
		if !isThemeReference(f.Name) {
			continue
		}
		comma := -1
		for i, child := range f.Nodes {
			if s, ok := child.(*ValueSeparator); ok && s.Value == "," {
				comma = i
				break
			}
		}
		if comma == -1 {
			f.Nodes = append(f.Nodes, Sep(","), Sep(" "), Word(fallback))
			return true
		}
		if strings.TrimSpace(ValueToCSS(f.Nodes[comma+1:])) == "initial" {
			f.Nodes = append(append([]ValueNode{}, f.Nodes[:comma+1]...), Sep(" "), Word(fallback))
			return true
		}
	}
	return false
}
