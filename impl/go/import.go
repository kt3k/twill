package twill

import "strings"

const maxImportDepth = 100

// ImportParams are the parsed params of an @import rule.
type ImportParams struct {
	URI      string
	Layer    *string
	Media    *string
	Supports *string
}

func strPtr(s string) *string { return &s }

// ParseImportParams parses @import params. It returns nil for imports that
// must be left untouched (url(), data:, http(s)).
func ParseImportParams(params string) (*ImportParams, error) {
	nodes := ParseValue(params)
	for len(nodes) > 0 {
		if _, ok := nodes[0].(*ValueSeparator); ok {
			nodes = nodes[1:]
			continue
		}
		break
	}
	if len(nodes) == 0 {
		return nil, nil
	}
	first, ok := nodes[0].(*ValueWord)
	if !ok {
		return nil, nil
	}
	uri, ok := Unquote(first.Value)
	if !ok {
		return nil, nil
	}
	if strings.HasPrefix(uri, "data:") || strings.HasPrefix(uri, "http://") || strings.HasPrefix(uri, "https://") {
		return nil, nil
	}
	nodes = nodes[1:]

	result := &ImportParams{URI: uri}
	var media []string
	sawMedia := false
	for _, n := range nodes {
		switch n := n.(type) {
		case *ValueSeparator:
			if len(media) > 0 {
				media = append(media, n.Value)
			}
		case *ValueFunction:
			if n.Name == "layer" {
				if result.Supports != nil || sawMedia {
					return nil, Errorf("`layer(...)` must appear before `supports(...)` and media queries in `@import %s`", params)
				}
				if result.Layer != nil {
					return nil, Errorf("Duplicate `layer(...)` in `@import %s`", params)
				}
				result.Layer = strPtr(strings.TrimSpace(ValueToCSS(n.Nodes)))
				continue
			}
			if n.Name == "supports" {
				if sawMedia {
					return nil, Errorf("`supports(...)` must appear before media queries in `@import %s`", params)
				}
				result.Supports = strPtr(strings.TrimSpace(ValueToCSS(n.Nodes)))
				continue
			}
			sawMedia = true
			media = append(media, ValueToCSS([]ValueNode{n}))
		case *ValueWord:
			if n.Value == "layer" {
				if result.Supports != nil || sawMedia {
					return nil, Errorf("`layer` must appear before `supports(...)` and media queries in `@import %s`", params)
				}
				result.Layer = strPtr("")
				continue
			}
			sawMedia = true
			media = append(media, n.Value)
		}
	}
	if text := strings.TrimSpace(strings.Join(media, "")); text != "" {
		result.Media = strPtr(text)
	}
	return result, nil
}

// BuildImportNodes wraps imported nodes per SPEC §6.2.
func BuildImportNodes(nodes []Node, layer, media, supports *string) []Node {
	root := nodes
	if layer != nil {
		root = []Node{NewAtRule("@layer", *layer, root...)}
	}
	if media != nil {
		root = []Node{NewAtRule("@media", *media, root...)}
	}
	if supports != nil {
		condition := *supports
		if !strings.HasPrefix(condition, "(") {
			condition = "(" + condition + ")"
		}
		root = []Node{NewAtRule("@supports", condition, root...)}
	}
	return root
}

// SubstituteAtImports expands @import and @reference in place. It returns
// FeatureAtImport whenever an import was resolved.
func SubstituteAtImports(ast *[]Node, base string, load StylesheetLoader) (Features, error) {
	return substituteAtImports(ast, base, load, 0)
}

func substituteAtImports(ast *[]Node, base string, load StylesheetLoader, depth int) (Features, error) {
	var features Features
	var failure error
	Walk(ast, func(n Node, u *WalkUtils) WalkAction {
		at, ok := n.(*AtRule)
		if !ok || (at.Name != "@import" && at.Name != "@reference") {
			return Continue
		}
		parsed, err := ParseImportParams(at.Params)
		if err != nil {
			failure = err
			return Stop
		}
		if parsed == nil {
			return Continue
		}
		if at.Name == "@reference" {
			parsed.Media = strPtr("reference")
		}
		if depth > maxImportDepth {
			failure = Errorf("Exceeded maximum recursion depth while resolving `%s %s`", at.Name, at.Params)
			return Stop
		}
		features |= FeatureAtImport
		loaded, err := load(parsed.URI, base)
		if err != nil {
			failure = err
			return Stop
		}
		imported, err := Parse(loaded.Content)
		if err != nil {
			failure = err
			return Stop
		}
		nested, err := substituteAtImports(&imported, loaded.Base, load, depth+1)
		if err != nil {
			failure = err
			return Stop
		}
		features |= nested
		wrapped := BuildImportNodes([]Node{NewContext(ContextMap{"base": loaded.Base}, imported...)}, parsed.Layer, parsed.Media, parsed.Supports)
		u.ReplaceWith(NewContext(ContextMap{}, wrapped...))
		return Skip
	})
	if failure != nil {
		return 0, failure
	}
	return features, nil
}
