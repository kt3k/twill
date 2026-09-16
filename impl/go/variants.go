package twill

import "strings"

// Compounds are the kinds of rules a variant generates or accepts (SPEC §4.1.8).
type Compounds int

const (
	// CompoundsNever generates or accepts nothing.
	CompoundsNever Compounds = 0
	// CompoundsAtRules covers at-rules.
	CompoundsAtRules Compounds = 1 << 0
	// CompoundsStyleRules covers style rules.
	CompoundsStyleRules Compounds = 1 << 1
)

// VariantApply mutates the children of node (a *Rule or *AtRule) in place.
// Returning false rejects the candidate.
type VariantApply func(node Node, v *Variant) bool

// VariantDefinition is a registered variant.
type VariantDefinition struct {
	Kind          VariantKind
	Order         int
	Apply         VariantApply
	Compounds     Compounds
	CompoundsWith Compounds
}

// VariantCompareFn compares two variants sharing one order.
type VariantCompareFn func(a, z *Variant) int

// Variants is the variant registry (SPEC §9.1).
type Variants struct {
	defs       map[string]*VariantDefinition
	names      []string
	compareFns map[int]VariantCompareFn
	lastOrder  int
	groupOrder int
}

// NewVariants creates an empty registry.
func NewVariants() *Variants {
	return &Variants{defs: map[string]*VariantDefinition{}, compareFns: map[int]VariantCompareFn{}}
}

func (v *Variants) register(name string, kind VariantKind, apply VariantApply, compounds, compoundsWith Compounds) {
	if existing, ok := v.defs[name]; ok {
		existing.Kind = kind
		existing.Apply = apply
		existing.Compounds = compounds
		return
	}
	order := v.groupOrder
	if order == 0 {
		v.lastOrder++
		order = v.lastOrder
	}
	v.defs[name] = &VariantDefinition{Kind: kind, Order: order, Apply: apply, Compounds: compounds, CompoundsWith: compoundsWith}
	v.names = append(v.names, name)
}

// Static registers a static variant.
func (v *Variants) Static(name string, apply VariantApply, compounds Compounds) {
	v.register(name, VariantStatic, apply, compounds, CompoundsNever)
}

// Functional registers a functional variant.
func (v *Variants) Functional(name string, apply VariantApply, compounds Compounds) {
	v.register(name, VariantFunctional, apply, compounds, CompoundsNever)
}

// Compound registers a compound variant.
func (v *Variants) Compound(name string, compoundsWith Compounds, apply VariantApply, compounds Compounds) {
	v.register(name, VariantCompound, apply, compounds, compoundsWith)
}

// Group registers every name inside fn with one shared order.
func (v *Variants) Group(fn func(), compare VariantCompareFn) {
	v.lastOrder++
	v.groupOrder = v.lastOrder
	if compare != nil {
		v.compareFns[v.groupOrder] = compare
	}
	fn()
	v.groupOrder = 0
}

// Has reports whether name is registered.
func (v *Variants) Has(name string) bool { _, ok := v.defs[name]; return ok }

// Get returns the definition of name.
func (v *Variants) Get(name string) *VariantDefinition { return v.defs[name] }

// Kind returns the kind of name.
func (v *Variants) Kind(name string) (VariantKind, bool) {
	d, ok := v.defs[name]
	if !ok {
		return 0, false
	}
	return d.Kind, true
}

// Names returns the registered names in registration order.
func (v *Variants) Names() []string { return append([]string{}, v.names...) }

// CompoundsForSelectors computes the compounds value of a selector list.
func CompoundsForSelectors(selectors []string) Compounds {
	result := CompoundsNever
	for _, s := range selectors {
		if strings.HasPrefix(s, "@") {
			if !strings.HasPrefix(s, "@media") && !strings.HasPrefix(s, "@supports") && !strings.HasPrefix(s, "@container") {
				return CompoundsNever
			}
			result |= CompoundsAtRules
			continue
		}
		if strings.Contains(s, "::") {
			return CompoundsNever
		}
		result |= CompoundsStyleRules
	}
	return result
}

// CompoundsOf returns the compounds value of a parsed variant.
func (v *Variants) CompoundsOf(variant *Variant) Compounds {
	if variant.Kind == VariantArbitrary {
		return CompoundsForSelectors([]string{variant.Selector})
	}
	if d, ok := v.defs[variant.Root]; ok {
		return d.Compounds
	}
	return CompoundsNever
}

// CompoundsWith reports whether the compound parent accepts child (SPEC §9.1).
func (v *Variants) CompoundsWith(parent string, child *Variant) bool {
	d, ok := v.defs[parent]
	if !ok || d.Kind != VariantCompound {
		return false
	}
	c := v.CompoundsOf(child)
	return c != CompoundsNever && d.CompoundsWith != CompoundsNever && c&d.CompoundsWith != 0
}

func compareStrings(a, z string) int {
	switch {
	case a < z:
		return -1
	case a > z:
		return 1
	}
	return 0
}

// Compare compares two parsed variants for output ordering (SPEC §9.4).
func (v *Variants) Compare(a, z *Variant) int {
	if a == z {
		return 0
	}
	if a == nil {
		return -1
	}
	if z == nil {
		return 1
	}
	if a.Kind == VariantArbitrary && z.Kind == VariantArbitrary {
		return compareStrings(a.Selector, z.Selector)
	}
	if a.Kind == VariantArbitrary {
		return 1
	}
	if z.Kind == VariantArbitrary {
		return -1
	}
	aOrder, zOrder := 1<<30, 1<<30
	if d, ok := v.defs[a.Root]; ok {
		aOrder = d.Order
	}
	if d, ok := v.defs[z.Root]; ok {
		zOrder = d.Order
	}
	if aOrder != zOrder {
		return aOrder - zOrder
	}
	if a.Kind == VariantCompound && z.Kind == VariantCompound {
		if inner := v.Compare(a.Inner, z.Inner); inner != 0 {
			return inner
		}
		switch {
		case a.Modifier != nil && z.Modifier != nil:
			return compareStrings(a.Modifier.Value, z.Modifier.Value)
		case a.Modifier != nil:
			return 1
		case z.Modifier != nil:
			return -1
		}
		return 0
	}
	if fn, ok := v.compareFns[aOrder]; ok {
		return fn(a, z)
	}
	if a.Root != z.Root {
		return compareStrings(a.Root, z.Root)
	}
	var aValue, zValue *VariantValue
	if a.Kind == VariantFunctional {
		aValue = a.Value
	}
	if z.Kind == VariantFunctional {
		zValue = z.Value
	}
	switch {
	case aValue == nil && zValue == nil:
		return 0
	case aValue == nil:
		return -1
	case zValue == nil:
		return 1
	case aValue.Kind != zValue.Kind:
		if aValue.Kind == ValueArbitrary {
			return 1
		}
		return -1
	}
	return compareStrings(aValue.Value, zValue.Value)
}

// StaticVariant registers a static variant that wraps the children in one
// rule per selector.
func StaticVariant(v *Variants, name string, selectors []string, compounds Compounds, useDefault bool) {
	if useDefault {
		compounds = CompoundsForSelectors(selectors)
	}
	v.Static(name, func(node Node, _ *Variant) bool {
		children := Children(node)
		nodes := make([]Node, 0, len(selectors))
		for i, selector := range selectors {
			inner := *children
			if i > 0 {
				inner = CloneNodes(*children)
			}
			nodes = append(nodes, NewRule(selector, inner...))
		}
		*children = nodes
		return true
	}, compounds)
}

// ApplyVariant applies a parsed variant to a rule node (SPEC §9.3).
func ApplyVariant(node Node, variant *Variant, variants *Variants, depth int) bool {
	children := Children(node)
	if variant.Kind == VariantArbitrary {
		if variant.Relative && depth == 0 {
			return false
		}
		*children = []Node{NewRule(variant.Selector, *children...)}
		return true
	}
	def := variants.Get(variant.Root)
	if def == nil {
		return false
	}
	if variant.Kind == VariantCompound {
		isolated := NewAtRule("@slot", "")
		if !ApplyVariant(isolated, variant.Inner, variants, depth+1) {
			return false
		}
		if variant.Root == "not" && len(isolated.Nodes) > 1 {
			return false
		}
		for _, child := range isolated.Nodes {
			switch child.(type) {
			case *Rule, *AtRule:
			default:
				return false
			}
			if !def.Apply(child, variant) {
				return false
			}
		}
		first := true
		var fill func(nodes []Node)
		fill = func(nodes []Node) {
			for _, child := range nodes {
				cc := Children(child)
				if cc == nil {
					continue
				}
				if _, ok := child.(*Rule); !ok {
					if _, ok := child.(*AtRule); !ok {
						continue
					}
				}
				if len(*cc) == 0 {
					if first {
						*cc = *children
					} else {
						*cc = CloneNodes(*children)
					}
					first = false
				} else {
					fill(*cc)
				}
			}
		}
		fill(isolated.Nodes)
		*children = isolated.Nodes
		return true
	}
	return def.Apply(node, variant)
}

// HasNestedStyleRules reports whether a tree contains nested style rules.
func HasNestedStyleRules(nodes []Node) bool {
	found := false
	Walk(&nodes, func(n Node, _ *WalkUtils) WalkAction {
		if _, ok := n.(*Rule); ok {
			found = true
			return Stop
		}
		return Continue
	})
	return found
}
