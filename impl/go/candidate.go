package twill

import "strings"

// ValueKind distinguishes named and arbitrary values and modifiers.
type ValueKind int

const (
	// ValueNamed is a named value such as `red-500`.
	ValueNamed ValueKind = iota
	// ValueArbitrary is a decoded arbitrary value.
	ValueArbitrary
)

// CandidateValue is the value segment of a functional candidate (SPEC §4.1.4).
type CandidateValue struct {
	Kind  ValueKind
	Value string
	// Fraction is set for named values when a slash segment could be a
	// fraction, for example `1/2`.
	Fraction *string
	// DataType is an explicit type hint of an arbitrary value.
	DataType *string
}

// Modifier is a candidate or variant modifier (SPEC §4.1.5).
type Modifier struct {
	Kind  ValueKind
	Value string
}

// VariantKind is the kind of a parsed variant.
type VariantKind int

const (
	// VariantStatic is a static variant.
	VariantStatic VariantKind = iota
	// VariantFunctional is a functional variant.
	VariantFunctional
	// VariantCompound is a compound variant.
	VariantCompound
	// VariantArbitrary is an arbitrary variant.
	VariantArbitrary
)

// VariantValue is the value of a functional variant.
type VariantValue struct {
	Kind  ValueKind
	Value string
}

// Variant is a parsed variant (SPEC §4.1.6).
type Variant struct {
	Kind     VariantKind
	Root     string
	Value    *VariantValue
	Modifier *Modifier
	// Inner is the inner variant of a compound variant.
	Inner *Variant
	// Selector and Relative belong to arbitrary variants.
	Selector string
	Relative bool
}

// CandidateKind is the kind of a parsed candidate.
type CandidateKind int

const (
	// CandidateStatic is a static utility.
	CandidateStatic CandidateKind = iota
	// CandidateFunctional is a functional utility.
	CandidateFunctional
	// CandidateArbitrary is an arbitrary property.
	CandidateArbitrary
)

// Candidate is a parsed class name (SPEC §4.1.3).
type Candidate struct {
	Kind CandidateKind
	// Raw is the original class name including variants and the important marker.
	Raw string
	// Variants are in application order: the rightmost in the source is first.
	Variants  []*Variant
	Important bool
	// Root is set for static and functional candidates.
	Root string
	// Value and Modifier are set for functional candidates.
	Value    *CandidateValue
	Modifier *Modifier
	// Property and ArbitraryValue are set for arbitrary candidates.
	Property       string
	ArbitraryValue string
}

// UtilityKind is the kind of a utility definition.
type UtilityKind int

const (
	// UtilityStatic is a static utility.
	UtilityStatic UtilityKind = iota
	// UtilityFunctional is a functional utility.
	UtilityFunctional
)

// ParserContext is the part of the design system the parsers consult.
type ParserContext interface {
	ThemePrefix() string
	HasUtility(name string, kind UtilityKind) bool
	HasVariant(name string) bool
	VariantKindOf(name string) (VariantKind, bool)
	CompoundsWith(parent string, child *Variant) bool
	// ParseVariant is memoized; it returns the same object for the same input.
	ParseVariant(input string) *Variant
}

// RootMatch is a split of an input into a root and an optional value.
type RootMatch struct {
	Root  string
	Value *string
}

// FindRoots yields every (root, value) split whose root exists (SPEC §8.2.2).
func FindRoots(input string, exists func(string) bool) []RootMatch {
	var result []RootMatch
	if exists(input) {
		result = append(result, RootMatch{Root: input})
	}
	idx := strings.LastIndex(input, "-")
	if idx != -1 {
		for {
			root := input[:idx]
			if exists(root) {
				value := input[idx+1:]
				if value == "" {
					break
				}
				if root == "@" {
					break
				}
				result = append(result, RootMatch{Root: root, Value: strPtr(value)})
			}
			if idx == 0 {
				break
			}
			idx = strings.LastIndex(input[:idx], "-")
			if idx <= 0 {
				break
			}
		}
	}
	if strings.HasPrefix(input, "@") && exists("@") {
		result = append(result, RootMatch{Root: "@", Value: strPtr(input[1:])})
	}
	return result
}

// ParseModifier parses a modifier segment (SPEC §8.2.1).
func ParseModifier(modifier string) *Modifier {
	if strings.HasPrefix(modifier, "[") && strings.HasSuffix(modifier, "]") {
		value := DecodeArbitraryValue(modifier[1 : len(modifier)-1])
		if !IsValidArbitrary(value) || strings.TrimSpace(value) == "" {
			return nil
		}
		return &Modifier{Kind: ValueArbitrary, Value: value}
	}
	if strings.HasPrefix(modifier, "(") && strings.HasSuffix(modifier, ")") {
		value := DecodeArbitraryValue(modifier[1 : len(modifier)-1])
		if !strings.HasPrefix(value, "--") || !IsValidArbitrary(value) {
			return nil
		}
		return &Modifier{Kind: ValueArbitrary, Value: "var(" + value + ")"}
	}
	if IsNamedValue(modifier) {
		return &Modifier{Kind: ValueNamed, Value: modifier}
	}
	return nil
}

// ParseCandidate returns every interpretation of a raw class name (SPEC §8.2).
func ParseCandidate(input string, ds ParserContext) []*Candidate {
	var results []*Candidate

	rawVariants := Segment(input, ':')
	if prefix := ds.ThemePrefix(); prefix != "" {
		if len(rawVariants) == 1 || rawVariants[0] != prefix {
			return nil
		}
		rawVariants = rawVariants[1:]
	}
	base := rawVariants[len(rawVariants)-1]
	rawVariants = rawVariants[:len(rawVariants)-1]

	variants := []*Variant{}
	for i := len(rawVariants) - 1; i >= 0; i-- {
		v := ds.ParseVariant(rawVariants[i])
		if v == nil {
			return nil
		}
		variants = append(variants, v)
	}

	important := false
	if strings.HasSuffix(base, "!") {
		important = true
		base = base[:len(base)-1]
	} else if strings.HasPrefix(base, "!") {
		important = true
		base = base[1:]
	}

	if ds.HasUtility(base, UtilityStatic) && !strings.Contains(base, "[") {
		results = append(results, &Candidate{Kind: CandidateStatic, Root: base, Variants: variants, Important: important, Raw: input})
	}

	parts := Segment(base, '/')
	if len(parts) > 2 {
		return results
	}
	baseWithoutModifier := parts[0]
	var modifierSegment *string
	if len(parts) == 2 {
		modifierSegment = strPtr(parts[1])
	}
	var modifier *Modifier
	if modifierSegment != nil {
		modifier = ParseModifier(*modifierSegment)
		if modifier == nil {
			return results
		}
	}

	if strings.HasPrefix(baseWithoutModifier, "[") {
		if !strings.HasSuffix(baseWithoutModifier, "]") || len(baseWithoutModifier) < 2 {
			return results
		}
		second := baseWithoutModifier[1]
		if !(second == '-' || (second >= 'a' && second <= 'z')) {
			return results
		}
		inner := baseWithoutModifier[1 : len(baseWithoutModifier)-1]
		colon := strings.Index(inner, ":")
		if colon == -1 || colon == 0 || colon == len(inner)-1 {
			return results
		}
		value := DecodeArbitraryValue(inner[colon+1:])
		if !IsValidArbitrary(value) {
			return results
		}
		return append(results, &Candidate{
			Kind: CandidateArbitrary, Property: inner[:colon], ArbitraryValue: value,
			Modifier: modifier, Variants: variants, Important: important, Raw: input,
		})
	}

	var roots []RootMatch
	switch {
	case strings.HasSuffix(baseWithoutModifier, "]"):
		idx := strings.Index(baseWithoutModifier, "-[")
		if idx == -1 {
			return results
		}
		root := baseWithoutModifier[:idx]
		if !ds.HasUtility(root, UtilityFunctional) {
			return results
		}
		roots = []RootMatch{{Root: root, Value: strPtr(baseWithoutModifier[idx+1:])}}
	case strings.HasSuffix(baseWithoutModifier, ")"):
		idx := strings.Index(baseWithoutModifier, "-(")
		if idx == -1 {
			return results
		}
		root := baseWithoutModifier[:idx]
		if !ds.HasUtility(root, UtilityFunctional) {
			return results
		}
		inner := baseWithoutModifier[idx+2 : len(baseWithoutModifier)-1]
		innerParts := Segment(inner, ':')
		dataType := ""
		value := inner
		if len(innerParts) == 2 {
			dataType, value = innerParts[0], innerParts[1]
		} else if len(innerParts) > 2 {
			return results
		}
		if !strings.HasPrefix(value, "--") || !IsValidArbitrary(value) {
			return results
		}
		rewritten := "[var(" + value + ")]"
		if dataType != "" {
			rewritten = "[" + dataType + ":var(" + value + ")]"
		}
		roots = []RootMatch{{Root: root, Value: strPtr(rewritten)}}
	default:
		roots = FindRoots(baseWithoutModifier, func(r string) bool { return ds.HasUtility(r, UtilityFunctional) })
	}

	for _, match := range roots {
		candidate := &Candidate{Kind: CandidateFunctional, Root: match.Root, Modifier: modifier, Variants: variants, Important: important, Raw: input}
		if match.Value == nil {
			results = append(results, candidate)
			continue
		}
		value := *match.Value
		if bracket := strings.Index(value, "["); bracket != -1 {
			if !strings.HasSuffix(value, "]") {
				return results
			}
			decoded := DecodeArbitraryValue(value[bracket+1 : len(value)-1])
			if !IsValidArbitrary(decoded) {
				continue
			}
			var dataType *string
			arbitraryValue := decoded
			i := 0
			for i < len(decoded) {
				c := decoded[i]
				if c == '-' || (c >= 'a' && c <= 'z') {
					i++
				} else {
					break
				}
			}
			if i < len(decoded) && decoded[i] == ':' {
				hint := decoded[:i]
				if hint == "" {
					continue
				}
				dataType = strPtr(hint)
				arbitraryValue = decoded[i+1:]
			}
			if strings.TrimSpace(arbitraryValue) == "" {
				continue
			}
			candidate.Value = &CandidateValue{Kind: ValueArbitrary, Value: arbitraryValue, DataType: dataType}
		} else {
			var fraction *string
			if modifierSegment != nil && modifier != nil && modifier.Kind == ValueNamed {
				fraction = strPtr(value + "/" + *modifierSegment)
			}
			if !IsNamedValue(value) {
				continue
			}
			candidate.Value = &CandidateValue{Kind: ValueNamed, Value: value, Fraction: fraction}
		}
		results = append(results, candidate)
	}
	return results
}

// ParseVariant parses a variant segment (SPEC §8.3).
func ParseVariant(input string, ds ParserContext) *Variant {
	if strings.HasPrefix(input, "[") && strings.HasSuffix(input, "]") {
		if len(input) > 1 && input[1] == '@' && strings.Contains(input, "&") {
			return nil
		}
		selector := DecodeArbitraryValue(input[1 : len(input)-1])
		if !IsValidArbitrary(selector) || strings.TrimSpace(selector) == "" {
			return nil
		}
		first := selector[0]
		relative := first == '>' || first == '+' || first == '~'
		result := selector
		if !relative && first != '@' && !strings.Contains(selector, "&") {
			result = "&:is(" + selector + ")"
		}
		return &Variant{Kind: VariantArbitrary, Selector: result, Relative: relative}
	}

	parts := Segment(input, '/')
	if len(parts) > 2 {
		return nil
	}
	name := parts[0]
	var modifierText *string
	if len(parts) == 2 {
		modifierText = strPtr(parts[1])
	}

	for _, match := range FindRoots(name, ds.HasVariant) {
		kind, ok := ds.VariantKindOf(match.Root)
		if !ok {
			continue
		}
		switch kind {
		case VariantStatic:
			if match.Value != nil || modifierText != nil {
				return nil
			}
			return &Variant{Kind: VariantStatic, Root: match.Root}
		case VariantFunctional:
			var modifier *Modifier
			if modifierText != nil {
				modifier = ParseModifier(*modifierText)
				if modifier == nil {
					return nil
				}
			}
			if match.Value == nil {
				return &Variant{Kind: VariantFunctional, Root: match.Root, Modifier: modifier}
			}
			value := *match.Value
			if strings.HasSuffix(value, "]") {
				if !strings.HasPrefix(value, "[") {
					continue
				}
				decoded := DecodeArbitraryValue(value[1 : len(value)-1])
				if !IsValidArbitrary(decoded) || strings.TrimSpace(decoded) == "" {
					return nil
				}
				return &Variant{Kind: VariantFunctional, Root: match.Root, Value: &VariantValue{Kind: ValueArbitrary, Value: decoded}, Modifier: modifier}
			}
			if strings.HasSuffix(value, ")") {
				if !strings.HasPrefix(value, "(") {
					continue
				}
				decoded := DecodeArbitraryValue(value[1 : len(value)-1])
				if !IsValidArbitrary(decoded) || strings.TrimSpace(decoded) == "" || !strings.HasPrefix(decoded, "--") {
					return nil
				}
				return &Variant{Kind: VariantFunctional, Root: match.Root, Value: &VariantValue{Kind: ValueArbitrary, Value: "var(" + decoded + ")"}, Modifier: modifier}
			}
			if !IsNamedValue(value) {
				continue
			}
			return &Variant{Kind: VariantFunctional, Root: match.Root, Value: &VariantValue{Kind: ValueNamed, Value: value}, Modifier: modifier}
		case VariantCompound:
			if match.Value == nil {
				return nil
			}
			innerText := *match.Value
			remaining := modifierText
			if (match.Root == "not" || match.Root == "has" || match.Root == "in") && modifierText != nil {
				innerText = innerText + "/" + *modifierText
				remaining = nil
			}
			inner := ds.ParseVariant(innerText)
			if inner == nil {
				return nil
			}
			if !ds.CompoundsWith(match.Root, inner) {
				return nil
			}
			var modifier *Modifier
			if remaining != nil {
				modifier = ParseModifier(*remaining)
				if modifier == nil {
					return nil
				}
			}
			return &Variant{Kind: VariantCompound, Root: match.Root, Modifier: modifier, Inner: inner}
		}
	}
	return nil
}
