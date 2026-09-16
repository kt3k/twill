package twill

import "strings"

// SubstituteAtApply expands every @apply in place (SPEC §6.10). @apply
// inside @utility bodies is expanded first, in dependency order.
func SubstituteAtApply(ast *[]Node, ds *DesignSystem) (Features, error) {
	var features Features

	utilityNodes := map[string]*AtRule{}
	var utilityOrder []string
	Walk(ast, func(n Node, _ *WalkUtils) WalkAction {
		if at, ok := n.(*AtRule); ok && at.Name == "@utility" {
			root := strings.TrimSuffix(strings.TrimSpace(at.Params), "-*")
			utilityNodes[root] = at
			utilityOrder = append(utilityOrder, root)
			return Skip
		}
		return Continue
	})

	if len(utilityNodes) > 0 {
		dependencies := map[string]map[string]bool{}
		var failure error
		for _, root := range utilityOrder {
			node := utilityNodes[root]
			deps := map[string]bool{}
			Walk(&node.Nodes, func(child Node, _ *WalkUtils) WalkAction {
				at, ok := child.(*AtRule)
				if !ok || at.Name != "@apply" {
					return Continue
				}
				for _, candidate := range strings.Fields(at.Params) {
					for _, parsed := range ds.ParseCandidate(candidate) {
						if parsed.Kind == CandidateArbitrary {
							continue
						}
						if _, ok := utilityNodes[parsed.Root]; !ok {
							continue
						}
						if parsed.Root == root {
							failure = Errorf("You cannot `@apply` the `%s` utility here because it creates a circular dependency.", candidate)
							return Stop
						}
						deps[parsed.Root] = true
					}
				}
				return Continue
			})
			if failure != nil {
				return 0, failure
			}
			dependencies[root] = deps
		}
		sorted, err := TopologicalSort(utilityOrder, dependencies, func(cycle string) error {
			return Errorf("You cannot `@apply` the `%s` utility here because it creates a circular dependency.", cycle)
		})
		if err != nil {
			return 0, err
		}
		for _, root := range sorted {
			node := utilityNodes[root]
			f, err := expandApplyIn(&node.Nodes, node, ds)
			if err != nil {
				return 0, err
			}
			features |= f
		}
	}

	f, err := expandApplyIn(ast, nil, ds)
	if err != nil {
		return 0, err
	}
	return features | f, nil
}

func expandApplyIn(nodes *[]Node, root Node, ds *DesignSystem) (Features, error) {
	var features Features
	var failure error
	var path []Node
	if root != nil {
		path = []Node{root}
	}
	WalkFrom(nodes, func(n Node, u *WalkUtils) WalkAction {
		at, ok := n.(*AtRule)
		if !ok {
			return Continue
		}
		if at.Name == "@utility" {
			return Skip
		}
		if at.Name != "@apply" {
			return Continue
		}
		parent := u.Parent
		if parent == nil {
			parent = root
		}
		// A top-level @apply is left untouched.
		if parent == nil {
			return Continue
		}
		if _, ok := parent.(*Context); ok && isTopLevelPath(u.Path) {
			return Continue
		}
		if len(at.Nodes) > 0 {
			failure = Errorf("`@apply` cannot have a body.")
			return Stop
		}
		for _, ancestor := range u.Path {
			if a, ok := ancestor.(*AtRule); ok && a.Name == "@keyframes" {
				failure = Errorf("You cannot use `@apply` inside `@keyframes`.")
				return Stop
			}
		}
		candidates := strings.Fields(at.Params)
		mixins := 0
		for _, c := range candidates {
			if strings.HasPrefix(c, "--") {
				mixins++
			}
		}
		if mixins == len(candidates) && len(candidates) > 0 {
			return Continue
		}
		if mixins > 0 {
			failure = Errorf("You cannot mix CSS mixins with utility classes in `@apply %s`.", at.Params)
			return Stop
		}
		var invalid error
		compiled := CompileCandidates(candidates, ds, CompileCandidatesOptions{
			IgnoreImportant: true,
			OnInvalidCandidate: func(candidate string) {
				if invalid == nil {
					invalid = Errorf("%s", ApplyErrorMessage(candidate, ds))
				}
			},
		})
		if invalid != nil {
			failure = invalid
			return Stop
		}
		var replacement []Node
		for _, rule := range compiled {
			replacement = append(replacement, rule.Nodes...)
		}
		features |= FeatureAtApply
		u.ReplaceWith(replacement...)
		return Continue
	}, root, ContextMap{}, path)
	return features, failure
}

// ApplyErrorMessage builds the error message for an unknown @apply candidate.
func ApplyErrorMessage(candidate string, ds *DesignSystem) string {
	prefix := ds.Theme.Prefix
	if prefix != "" && !strings.HasPrefix(candidate, prefix+":") {
		return "Cannot apply unknown utility class `" + candidate + "`. Did you forget the `" + prefix + ":` prefix?"
	}
	if ds.InvalidCandidates[candidate] {
		return "Cannot apply utility class `" + candidate + "` because it is explicitly disabled by `@source not inline(...)`."
	}
	parts := Segment(candidate, ':')
	variants := parts[:len(parts)-1]
	if prefix != "" && len(variants) > 0 {
		variants = variants[1:]
	}
	for _, v := range variants {
		if ds.ParseVariant(v) == nil {
			return "Cannot apply unknown variant `" + v + "` in `" + candidate + "`."
		}
	}
	if ds.Theme.Size() == 0 {
		return "Cannot apply unknown utility class `" + candidate + "`. The theme is empty; are you missing `@import \"twill\";` or `@reference \"twill\";`?"
	}
	return "Cannot apply unknown utility class `" + candidate + "`."
}

// TopologicalSort orders names so that every name comes after its
// dependencies. onCycle builds the error for a cycle.
func TopologicalSort(names []string, dependencies map[string]map[string]bool, onCycle func(string) error) ([]string, error) {
	var result []string
	state := map[string]int{}
	var failure error
	var visit func(name string)
	visit = func(name string) {
		if failure != nil {
			return
		}
		switch state[name] {
		case 2:
			return
		case 1:
			failure = onCycle(name)
			return
		}
		state[name] = 1
		var deps []string
		for dep := range dependencies[name] {
			if _, ok := dependencies[dep]; ok {
				deps = append(deps, dep)
			}
		}
		sortStrings(deps)
		for _, dep := range deps {
			visit(dep)
		}
		state[name] = 2
		result = append(result, name)
	}
	for _, name := range names {
		visit(name)
	}
	if failure != nil {
		return nil, failure
	}
	return result, nil
}

func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}
