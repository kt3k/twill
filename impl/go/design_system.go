package twill

import "sort"

// DesignSystem owns the theme, the registries, memoization caches, and the
// set of known-invalid candidates (SPEC §4.1.9).
type DesignSystem struct {
	Theme             *Theme
	Utilities         *Utilities
	Variants          *Variants
	InvalidCandidates map[string]bool
	// Important marks every generated declaration !important.
	Important bool

	candidateCache map[string][]*Candidate
	variantCache   map[string]*Variant
	variantMissing map[string]bool
	compileCache   map[*Candidate]map[int][]CompiledRule
	variantOrder   map[*Variant]int
}

// NewDesignSystem creates an empty design system.
func NewDesignSystem(theme *Theme) *DesignSystem {
	return &DesignSystem{
		Theme:             theme,
		Utilities:         NewUtilities(),
		Variants:          NewVariants(),
		InvalidCandidates: map[string]bool{},
		candidateCache:    map[string][]*Candidate{},
		variantCache:      map[string]*Variant{},
		variantMissing:    map[string]bool{},
		compileCache:      map[*Candidate]map[int][]CompiledRule{},
	}
}

// BuildDesignSystem creates a design system with the built-in variants and utilities.
func BuildDesignSystem(theme *Theme) *DesignSystem {
	ds := NewDesignSystem(theme)
	RegisterBuiltinVariants(ds.Variants, theme)
	RegisterBuiltinUtilities(ds.Utilities, theme)
	return ds
}

// ThemePrefix implements ParserContext.
func (ds *DesignSystem) ThemePrefix() string { return ds.Theme.Prefix }

// HasUtility implements ParserContext.
func (ds *DesignSystem) HasUtility(name string, kind UtilityKind) bool {
	return ds.Utilities.Has(name, kind)
}

// HasVariant implements ParserContext.
func (ds *DesignSystem) HasVariant(name string) bool { return ds.Variants.Has(name) }

// VariantKindOf implements ParserContext.
func (ds *DesignSystem) VariantKindOf(name string) (VariantKind, bool) { return ds.Variants.Kind(name) }

// CompoundsWith implements ParserContext.
func (ds *DesignSystem) CompoundsWith(parent string, child *Variant) bool {
	return ds.Variants.CompoundsWith(parent, child)
}

// ParseCandidate returns every interpretation of a raw class name, memoized.
func (ds *DesignSystem) ParseCandidate(raw string) []*Candidate {
	if cached, ok := ds.candidateCache[raw]; ok {
		return cached
	}
	parsed := ParseCandidate(raw, ds)
	ds.candidateCache[raw] = parsed
	return parsed
}

// ParseVariant parses a variant, memoized so equal inputs share one object.
func (ds *DesignSystem) ParseVariant(raw string) *Variant {
	if v, ok := ds.variantCache[raw]; ok {
		return v
	}
	if ds.variantMissing[raw] {
		return nil
	}
	v := ParseVariant(raw, ds)
	if v == nil {
		ds.variantMissing[raw] = true
		return nil
	}
	ds.variantCache[raw] = v
	ds.variantOrder = nil
	return v
}

// VariantOrder sorts every parsed variant and assigns an index that
// increases each time Compare reports a difference (SPEC §9.4).
func (ds *DesignSystem) VariantOrder() map[*Variant]int {
	if ds.variantOrder != nil {
		return ds.variantOrder
	}
	parsed := make([]*Variant, 0, len(ds.variantCache))
	for _, v := range ds.variantCache {
		parsed = append(parsed, v)
	}
	sort.SliceStable(parsed, func(i, j int) bool { return ds.Variants.Compare(parsed[i], parsed[j]) < 0 })
	order := map[*Variant]int{}
	index := 0
	var previous *Variant
	for _, v := range parsed {
		if previous != nil && ds.Variants.Compare(previous, v) != 0 {
			index++
		}
		order[v] = index
		previous = v
	}
	ds.variantOrder = order
	return order
}

// ClearCandidateCache drops memoized candidates and compiled results.
func (ds *DesignSystem) ClearCandidateCache() {
	ds.candidateCache = map[string][]*Candidate{}
	ds.compileCache = map[*Candidate]map[int][]CompiledRule{}
}
