package twill

import "strings"

// CustomVariant is a parsed @custom-variant (SPEC §6.7).
type CustomVariant struct {
	Name string
	// Dependencies are the names referenced through nested @variant.
	Dependencies map[string]bool
	Register     func(ds *DesignSystem)
}

// IsCustomVariantCompat reports whether a top-level @variant is a
// compatibility form of @custom-variant.
func IsCustomVariantCompat(node *AtRule) bool {
	if len(node.Nodes) == 0 {
		return strings.Contains(node.Params, "(")
	}
	hasSlot := false
	Walk(&node.Nodes, func(n Node, _ *WalkUtils) WalkAction {
		if at, ok := n.(*AtRule); ok && at.Name == "@slot" {
			hasSlot = true
			return Stop
		}
		return Continue
	})
	return hasSlot
}

// ParseCustomVariant parses an @custom-variant node into a registration.
func ParseCustomVariant(node *AtRule) (*CustomVariant, error) {
	params := strings.TrimSpace(node.Params)
	name := params
	rest := ""
	if space := strings.IndexAny(params, " \t\n"); space != -1 {
		name = params[:space]
		rest = strings.TrimSpace(params[space:])
	}
	if !IsValidVariantName(name) {
		return nil, Errorf("`@custom-variant %s` defines an invalid variant name. Variants should only contain alphanumeric, dashes or underscore characters.", name)
	}
	if rest != "" && len(node.Nodes) > 0 {
		return nil, Errorf("`@custom-variant %s` cannot have both a selector and a body.", name)
	}
	if rest == "" && len(node.Nodes) == 0 {
		return nil, Errorf("`@custom-variant %s` has no selector or body.", name)
	}

	if rest != "" {
		if !strings.HasPrefix(rest, "(") || !strings.HasSuffix(rest, ")") {
			return nil, Errorf("`@custom-variant %s %s` has an invalid selector.", name, rest)
		}
		var selectors, atRuleSelectors, styleSelectors []string
		for _, s := range Segment(rest[1:len(rest)-1], ',') {
			s = strings.TrimSpace(s)
			if s == "" {
				return nil, Errorf("`@custom-variant %s %s` has an empty selector.", name, rest)
			}
			selectors = append(selectors, s)
			if strings.HasPrefix(s, "@") {
				atRuleSelectors = append(atRuleSelectors, s)
			} else {
				styleSelectors = append(styleSelectors, s)
			}
		}
		compounds := CompoundsForSelectors(selectors)
		return &CustomVariant{Name: name, Dependencies: map[string]bool{}, Register: func(ds *DesignSystem) {
			ds.Variants.Static(name, func(target Node, _ *Variant) bool {
				children := Children(target)
				first := true
				take := func() []Node {
					if first {
						first = false
						return *children
					}
					return CloneNodes(*children)
				}
				var nodes []Node
				if len(styleSelectors) > 0 {
					nodes = append(nodes, StyleRule(strings.Join(styleSelectors, ", "), take()...))
				}
				for _, s := range atRuleSelectors {
					nodes = append(nodes, NewRule(s, take()...))
				}
				*children = nodes
				return true
			}, compounds)
		}}, nil
	}

	body := CloneNodes(node.Nodes)
	dependencies := map[string]bool{}
	var selectors []string
	Walk(&body, func(n Node, _ *WalkUtils) WalkAction {
		switch n := n.(type) {
		case *Rule:
			selectors = append(selectors, n.Selector)
		case *AtRule:
			if n.Name == "@variant" {
				for _, group := range Segment(n.Params, ',') {
					for _, v := range Segment(group, ':') {
						if v = strings.TrimSpace(v); v != "" {
							dependencies[v] = true
						}
					}
				}
			} else if n.Name != "@slot" {
				selectors = append(selectors, strings.TrimSpace(n.Name+" "+n.Params))
			}
		}
		return Continue
	})
	compounds := CompoundsForSelectors(selectors)

	return &CustomVariant{Name: name, Dependencies: dependencies, Register: func(ds *DesignSystem) {
		ds.Variants.Static(name, func(target Node, _ *Variant) bool {
			children := Children(target)
			clone := CloneNodes(body)
			first := true
			Walk(&clone, func(n Node, u *WalkUtils) WalkAction {
				at, ok := n.(*AtRule)
				if !ok {
					return Continue
				}
				if _, ok := u.Parent.(*AtRoot); ok {
					return Continue
				}
				if at.Name == "@slot" {
					if first {
						first = false
						u.ReplaceWith(*children...)
					} else {
						u.ReplaceWith(CloneNodes(*children)...)
					}
					return Skip
				}
				if at.Name == "@keyframes" || at.Name == "@property" {
					u.ReplaceWith(NewAtRoot(at))
					return Skip
				}
				return Continue
			})
			*children = clone
			return true
		}, compounds)
	}}, nil
}
