package twill

import (
	"math/big"
	"sort"
)

// CompileFlags control candidate compilation.
type CompileFlags int

const (
	// RespectImportant applies the design-system-wide important flag.
	RespectImportant CompileFlags = 1 << 0
)

// PropertySort is the sort key of a node list (SPEC §11.3).
type PropertySort struct {
	Order []int
	Count int
}

// CompiledRule is a compiled rule with its property sort key.
type CompiledRule struct {
	Node         *Rule
	PropertySort PropertySort
}

// GetPropertySort computes the property sort key of a node list.
func GetPropertySort(nodes []Node) PropertySort {
	seen := map[int]bool{}
	count := 0
	queue := append([]Node{}, nodes...)
outer:
	for len(queue) > 0 {
		n := queue[0]
		queue = queue[1:]
		switch n := n.(type) {
		case *Declaration:
			if n.NoValue {
				continue
			}
			count++
			if n.Property == "--tw-sort" {
				if index := PropertyIndex(n.Value); index != -1 {
					seen[index] = true
					break outer
				}
				continue
			}
			if index := PropertyIndex(n.Property); index != -1 {
				seen[index] = true
			}
		case *Rule:
			queue = append(queue, n.Nodes...)
		case *AtRule:
			queue = append(queue, n.Nodes...)
		case *Context:
			queue = append(queue, n.Nodes...)
		}
	}
	order := make([]int, 0, len(seen))
	for index := range seen {
		order = append(order, index)
	}
	sort.Ints(order)
	return PropertySort{Order: order, Count: count}
}

func compileBaseUtility(c *Candidate, ds *DesignSystem) [][]Node {
	if c.Kind == CandidateArbitrary {
		value, ok := AsColor(c.ArbitraryValue, c.Modifier, ds.Theme)
		if !ok {
			return nil
		}
		return [][]Node{{Decl(c.Property, value)}}
	}
	var defs []*UtilityDefinition
	for _, d := range ds.Utilities.Get(c.Root) {
		if (d.Kind == UtilityStatic) == (c.Kind == CandidateStatic) {
			defs = append(defs, d)
		}
	}
	var ordered []*UtilityDefinition
	for _, d := range defs {
		if !d.IsFallback() {
			ordered = append(ordered, d)
		}
	}
	for _, d := range defs {
		if d.IsFallback() {
			ordered = append(ordered, d)
		}
	}
	var results [][]Node
	for _, d := range ordered {
		nodes, status := d.Compile(c)
		switch status {
		case NotHandled:
			continue
		case Invalid:
			if d.Types != nil {
				return results
			}
			continue
		}
		results = append(results, nodes)
	}
	return results
}

func applyImportant(nodes []Node) {
	Walk(&nodes, func(n Node, _ *WalkUtils) WalkAction {
		switch n := n.(type) {
		case *AtRoot:
			return Skip
		case *Declaration:
			n.Important = true
		}
		return Continue
	})
}

// compileAstNodes compiles one candidate interpretation (SPEC §11.2).
func compileAstNodes(c *Candidate, flags CompileFlags, ds *DesignSystem) []CompiledRule {
	nodeLists := compileBaseUtility(c, ds)
	if len(nodeLists) == 0 {
		return nil
	}
	important := c.Important || (ds.Important && flags&RespectImportant != 0)
	var results []CompiledRule
	for _, nodes := range nodeLists {
		propertySort := GetPropertySort(nodes)
		if important {
			applyImportant(nodes)
		}
		rule := StyleRule("."+Escape(c.Raw), nodes...)
		for _, v := range c.Variants {
			if !ApplyVariant(rule, v, ds.Variants, 0) {
				return nil
			}
		}
		wrapped := []Node{rule}
		if _, err := SubstituteFunctions(&wrapped, ds); err != nil {
			return nil
		}
		if _, err := SubstituteAtVariant(&wrapped, ds); err != nil {
			return nil
		}
		results = append(results, CompiledRule{Node: rule, PropertySort: propertySort})
	}
	return results
}

// CompileAstNodes compiles a candidate interpretation, memoized per
// candidate object and flags. Cached results are cloned.
func (ds *DesignSystem) CompileAstNodes(c *Candidate, flags CompileFlags) []CompiledRule {
	byFlags, ok := ds.compileCache[c]
	if !ok {
		byFlags = map[int][]CompiledRule{}
		ds.compileCache[c] = byFlags
	}
	cached, ok := byFlags[int(flags)]
	if !ok {
		cached = compileAstNodes(c, flags, ds)
		byFlags[int(flags)] = cached
	}
	out := make([]CompiledRule, len(cached))
	for i, r := range cached {
		out[i] = CompiledRule{Node: CloneNode(r.Node).(*Rule), PropertySort: r.PropertySort}
	}
	return out
}

// CompileCandidatesOptions configure CompileCandidates.
type CompileCandidatesOptions struct {
	// IgnoreImportant disables the design-system-wide important flag.
	IgnoreImportant bool
	// OnInvalidCandidate receives each invalid raw candidate.
	OnInvalidCandidate func(candidate string)
}

type sortedRule struct {
	node         *Rule
	propertySort PropertySort
	variantOrder *big.Int
	candidate    string
}

// CompileCandidates compiles and sorts raw candidates (SPEC §11.1, §15.3).
func CompileCandidates(rawCandidates []string, ds *DesignSystem, options CompileCandidatesOptions) []*Rule {
	onInvalid := options.OnInvalidCandidate
	if onInvalid == nil {
		onInvalid = func(string) {}
	}
	flags := RespectImportant
	if options.IgnoreImportant {
		flags = 0
	}

	type match struct {
		raw        string
		candidates []*Candidate
	}
	var matches []match
	for _, raw := range rawCandidates {
		if ds.InvalidCandidates[raw] {
			onInvalid(raw)
			continue
		}
		parsed := ds.ParseCandidate(raw)
		if len(parsed) == 0 {
			onInvalid(raw)
			continue
		}
		matches = append(matches, match{raw, parsed})
	}

	order := ds.VariantOrder()
	var rules []sortedRule
	for _, m := range matches {
		found := false
		for _, c := range m.candidates {
			for _, compiled := range ds.CompileAstNodes(c, flags) {
				found = true
				variantOrder := new(big.Int)
				for _, v := range c.Variants {
					variantOrder.SetBit(variantOrder, order[v], 1)
				}
				rules = append(rules, sortedRule{compiled.Node, compiled.PropertySort, variantOrder, m.raw})
			}
		}
		if !found {
			onInvalid(m.raw)
		}
	}

	sort.SliceStable(rules, func(i, j int) bool { return compareSortedRules(rules[i], rules[j]) < 0 })
	out := make([]*Rule, len(rules))
	for i, r := range rules {
		out[i] = r.node
	}
	return out
}

func compareSortedRules(a, z sortedRule) int {
	if c := a.variantOrder.Cmp(z.variantOrder); c != 0 {
		return c
	}
	length := len(a.propertySort.Order)
	if len(z.propertySort.Order) > length {
		length = len(z.propertySort.Order)
	}
	for i := 0; i < length; i++ {
		ai, zi := -1, -1
		if i < len(a.propertySort.Order) {
			ai = a.propertySort.Order[i]
		}
		if i < len(z.propertySort.Order) {
			zi = z.propertySort.Order[i]
		}
		if ai == zi {
			continue
		}
		// A missing entry counts as infinity.
		if ai == -1 {
			return 1
		}
		if zi == -1 {
			return -1
		}
		return ai - zi
	}
	if a.propertySort.Count != z.propertySort.Count {
		return z.propertySort.Count - a.propertySort.Count
	}
	return Compare(a.candidate, z.candidate)
}
