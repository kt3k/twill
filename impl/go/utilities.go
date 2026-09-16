package twill

import "strings"

// CompileStatus is the outcome of a utility definition (SPEC §4.1.7).
type CompileStatus int

const (
	// NotHandled means this definition does not handle the candidate.
	NotHandled CompileStatus = iota
	// Handled means success.
	Handled
	// Invalid means the candidate is invalid for this definition.
	Invalid
)

// CompileFn compiles a candidate.
type CompileFn func(c *Candidate) ([]Node, CompileStatus)

// UtilityDefinition is a registered utility.
type UtilityDefinition struct {
	Kind    UtilityKind
	Compile CompileFn
	// Types marks a fallback definition when it has more than one entry and
	// includes "any".
	Types []string
}

// IsFallback reports whether the definition is a fallback definition.
func (d *UtilityDefinition) IsFallback() bool {
	if len(d.Types) <= 1 {
		return false
	}
	for _, t := range d.Types {
		if t == "any" {
			return true
		}
	}
	return false
}

// Utilities is the utility registry (SPEC §10.1).
type Utilities struct {
	defs  map[string][]*UtilityDefinition
	names []string
}

// NewUtilities creates an empty registry.
func NewUtilities() *Utilities { return &Utilities{defs: map[string][]*UtilityDefinition{}} }

func (u *Utilities) add(name string, d *UtilityDefinition) {
	if _, ok := u.defs[name]; !ok {
		u.names = append(u.names, name)
	}
	u.defs[name] = append(u.defs[name], d)
}

// Static registers a static definition.
func (u *Utilities) Static(name string, compile CompileFn) {
	u.add(name, &UtilityDefinition{Kind: UtilityStatic, Compile: compile})
}

// Functional registers a functional definition.
func (u *Utilities) Functional(name string, compile CompileFn, types []string) {
	u.add(name, &UtilityDefinition{Kind: UtilityFunctional, Compile: compile, Types: types})
}

// Has reports whether a definition of the kind exists for name.
func (u *Utilities) Has(name string, kind UtilityKind) bool {
	for _, d := range u.defs[name] {
		if d.Kind == kind {
			return true
		}
	}
	return false
}

// Get returns the definitions for name.
func (u *Utilities) Get(name string) []*UtilityDefinition { return u.defs[name] }

// Names returns the registered names.
func (u *Utilities) Names() []string { return append([]string{}, u.names...) }

// AsColor applies a modifier to a color value (SPEC §10.3).
func AsColor(value string, modifier *Modifier, theme *Theme) (string, bool) {
	if modifier == nil {
		return value, true
	}
	if modifier.Kind == ValueArbitrary {
		return WithAlpha(value, modifier.Value), true
	}
	if opacity, ok := theme.Resolve(strPtr(modifier.Value), []string{"--opacity"}, 0); ok {
		return WithAlpha(value, opacity), true
	}
	if !IsMultipleOfQuarter(modifier.Value) {
		return "", false
	}
	return WithAlpha(value, modifier.Value+"%"), true
}

// ResolveThemeColor resolves a named color candidate value through the theme.
func ResolveThemeColor(c *Candidate, themeKeys []string, theme *Theme) (string, bool) {
	if c.Value == nil || c.Value.Kind != ValueNamed {
		return "", false
	}
	var value string
	switch c.Value.Value {
	case "inherit":
		value = "inherit"
	case "transparent":
		value = "transparent"
	case "current":
		value = "currentcolor"
	default:
		v, ok := theme.Resolve(strPtr(c.Value.Value), themeKeys, 0)
		if !ok {
			return "", false
		}
		value = v
	}
	return AsColor(value, c.Modifier, theme)
}

// Declarations is a list of (property, value) pairs.
type Declarations [][2]string

// StaticUtility registers a static definition returning the declarations.
func StaticUtility(u *Utilities, name string, declarations Declarations) {
	u.Static(name, func(*Candidate) ([]Node, CompileStatus) {
		nodes := make([]Node, 0, len(declarations))
		for _, d := range declarations {
			nodes = append(nodes, Decl(d[0], d[1]))
		}
		return nodes, Handled
	})
}

// StaticUtilityFn registers a static definition backed by a function.
func StaticUtilityFn(u *Utilities, name string, fn func() []Node) {
	u.Static(name, func(*Candidate) ([]Node, CompileStatus) { return fn(), Handled })
}

// FunctionalUtilityDescription describes a functional utility (SPEC §10.2).
type FunctionalUtilityDescription struct {
	SupportsNegative  bool
	SupportsFractions bool
	ThemeKeys         []string
	// DefaultValue is the value for a candidate without a value segment.
	// NoDefault marks the null case (no output); when both are unset the
	// first namespace itself is resolved.
	DefaultValue            *string
	NoDefault               bool
	StaticValues            map[string][]Node
	HandleBareValue         func(v *CandidateValue) (string, bool)
	HandleNegativeBareValue func(v *CandidateValue) (string, bool)
	Handle                  func(value string, dataType *string) []Node
}

// FunctionalUtility registers a functional definition (SPEC §10.2).
func FunctionalUtility(u *Utilities, theme *Theme, root string, desc FunctionalUtilityDescription) {
	compile := func(c *Candidate, negative bool) ([]Node, CompileStatus) {
		var value string
		resolved := false
		var dataType *string

		switch {
		case c.Value == nil:
			if c.Modifier != nil {
				return nil, NotHandled
			}
			switch {
			case desc.NoDefault:
			case desc.DefaultValue != nil:
				value, resolved = *desc.DefaultValue, true
			default:
				value, resolved = theme.Resolve(nil, desc.ThemeKeys, 0)
			}
		case c.Value.Kind == ValueArbitrary:
			if c.Modifier != nil {
				return nil, NotHandled
			}
			value, resolved = c.Value.Value, true
			dataType = c.Value.DataType
		default:
			named := c.Value
			fractionConsumed := false
			if named.Fraction != nil {
				if v, ok := theme.Resolve(named.Fraction, desc.ThemeKeys, 0); ok {
					value, resolved, fractionConsumed = v, true, true
				}
			}
			if !resolved {
				if v, ok := theme.Resolve(strPtr(named.Value), desc.ThemeKeys, 0); ok {
					if c.Modifier != nil && !fractionConsumed {
						return nil, NotHandled
					}
					value, resolved = v, true
				}
			}
			if !resolved && desc.SupportsFractions && named.Fraction != nil {
				parts := strings.SplitN(*named.Fraction, "/", 2)
				if !IsPositiveInteger(parts[0]) || !IsPositiveInteger(parts[1]) {
					return nil, NotHandled
				}
				value, resolved = "calc("+parts[0]+" / "+parts[1]+" * 100%)", true
			}
			if !resolved && negative && desc.HandleNegativeBareValue != nil {
				if bare, ok := desc.HandleNegativeBareValue(named); ok {
					if !strings.Contains(bare, "/") && c.Modifier != nil {
						return nil, NotHandled
					}
					return desc.Handle(bare, nil), Handled
				}
			}
			if !resolved && desc.HandleBareValue != nil {
				if bare, ok := desc.HandleBareValue(named); ok {
					if !strings.Contains(bare, "/") && c.Modifier != nil {
						return nil, NotHandled
					}
					value, resolved = bare, true
				}
			}
			if !resolved && !negative && c.Modifier == nil && desc.StaticValues != nil {
				if nodes, ok := desc.StaticValues[named.Value]; ok {
					return CloneNodes(nodes), Handled
				}
			}
		}

		if !resolved {
			return nil, NotHandled
		}
		if negative {
			value = "calc(" + value + " * -1)"
		}
		nodes := desc.Handle(value, dataType)
		if nodes == nil {
			return nil, NotHandled
		}
		return nodes, Handled
	}

	u.Functional(root, func(c *Candidate) ([]Node, CompileStatus) { return compile(c, false) }, nil)
	if desc.SupportsNegative {
		u.Functional("-"+root, func(c *Candidate) ([]Node, CompileStatus) { return compile(c, true) }, nil)
	}
}

// ColorUtility registers a functional definition that resolves its value as a color.
func ColorUtility(u *Utilities, theme *Theme, root string, themeKeys []string, handle func(value string) []Node) {
	u.Functional(root, func(c *Candidate) ([]Node, CompileStatus) {
		if c.Value == nil {
			return nil, NotHandled
		}
		var value string
		var ok bool
		if c.Value.Kind == ValueArbitrary {
			value, ok = AsColor(c.Value.Value, c.Modifier, theme)
		} else {
			value, ok = ResolveThemeColor(c, themeKeys, theme)
		}
		if !ok {
			return nil, NotHandled
		}
		return handle(value), Handled
	}, nil)
}

// SpacingUtility registers `<name>-px`, optionally `-<name>-px`, and a
// functional utility whose bare values are spacing multipliers.
func SpacingUtility(u *Utilities, theme *Theme, name string, themeKeys []string, handle func(value string) []Node, supportsNegative, supportsFractions bool) {
	StaticUtilityFn(u, name+"-px", func() []Node { return handle("1px") })
	if supportsNegative {
		StaticUtilityFn(u, "-"+name+"-px", func() []Node { return handle("-1px") })
	}
	spacing := func(prefix string) func(v *CandidateValue) (string, bool) {
		return func(v *CandidateValue) (string, bool) {
			if !IsMultipleOfQuarter(v.Value) {
				return "", false
			}
			if _, ok := theme.Resolve(nil, []string{"--spacing"}, 0); !ok {
				return "", false
			}
			return "--spacing(" + prefix + v.Value + ")", true
		}
	}
	FunctionalUtility(u, theme, name, FunctionalUtilityDescription{
		ThemeKeys:               themeKeys,
		NoDefault:               true,
		SupportsNegative:        supportsNegative,
		SupportsFractions:       supportsFractions,
		HandleBareValue:         spacing(""),
		HandleNegativeBareValue: spacing("-"),
		Handle:                  func(value string, _ *string) []Node { return handle(value) },
	})
}

// BareInteger resolves a bare value when it is a positive integer.
func BareInteger(v *CandidateValue) (string, bool) {
	return v.Value, IsPositiveInteger(v.Value)
}
