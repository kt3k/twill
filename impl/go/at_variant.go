package twill

import "strings"

// SubstituteAtVariant expands every nested @variant in place (SPEC §6.9).
func SubstituteAtVariant(ast *[]Node, ds *DesignSystem) (Features, error) {
	var features Features
	var failure error
	Walk(ast, func(n Node, u *WalkUtils) WalkAction {
		at, ok := n.(*AtRule)
		if !ok || at.Name != "@variant" {
			return Continue
		}
		features |= FeatureVariants
		groups := Segment(at.Params, ',')
		var result []Node
		for i, group := range groups {
			names := Segment(strings.TrimSpace(group), ':')
			children := at.Nodes
			if i < len(groups)-1 {
				children = CloneNodes(at.Nodes)
			}
			rule := StyleRule("&", children...)
			for j := len(names) - 1; j >= 0; j-- {
				name := strings.TrimSpace(names[j])
				if name == "" {
					failure = Errorf("Cannot use `@variant` with an empty variant name in `@variant %s`.", at.Params)
					return Stop
				}
				variant := ds.ParseVariant(name)
				if variant == nil {
					failure = Errorf("Cannot use `@variant` with unknown variant: %s", name)
					return Stop
				}
				if !ApplyVariant(rule, variant, ds.Variants, 0) {
					failure = Errorf("Cannot use `@variant` with variant: %s", name)
					return Stop
				}
			}
			if rule.Selector == "&" {
				result = append(result, rule.Nodes...)
			} else {
				result = append(result, rule)
			}
		}
		u.ReplaceWith(result...)
		return Continue
	})
	return features, failure
}
