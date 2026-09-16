package twill

import (
	"regexp"
	"strings"
)

var functionalPrefixPattern = regexp.MustCompile(`^-?[a-z][a-zA-Z0-9_-]*$`)
var staticRootPattern = regexp.MustCompile(`^-?[a-z][a-zA-Z0-9_-]*`)
var staticRestPattern = regexp.MustCompile(`^[a-zA-Z0-9_./%-]*$`)

// IsValidStaticUtilityName reports whether name is a valid static utility name (SPEC §10.5).
func IsValidStaticUtilityName(name string) bool {
	root := staticRootPattern.FindString(name)
	if root == "" {
		return false
	}
	rest := name[len(root):]
	if strings.HasSuffix(root, "-") && rest == "" {
		return false
	}
	if rest == "" {
		return true
	}
	if !staticRestPattern.MatchString(rest) {
		return false
	}
	if strings.Count(rest, "/") > 1 || strings.HasSuffix(rest, "/") {
		return false
	}
	for i := 0; i < len(name); i++ {
		switch name[i] {
		case '.':
			if i == 0 || i == len(name)-1 || !isDigit(name[i-1]) || !isDigit(name[i+1]) {
				return false
			}
		case '%':
			if i != len(name)-1 || i == 0 || !isDigit(name[i-1]) {
				return false
			}
		}
	}
	return true
}

// IsValidFunctionalUtilityName reports whether name is `<root>-*` with a valid root.
func IsValidFunctionalUtilityName(name string) bool {
	return strings.HasSuffix(name, "-*") && functionalPrefixPattern.MatchString(name[:len(name)-2])
}

// CustomUtility is a parsed @utility (SPEC §6.8).
type CustomUtility struct {
	Name string
	Kind UtilityKind
	// Node is the @utility node; its body is read lazily so @apply can expand first.
	Node *AtRule
}

// ParseUtilityDefinition validates an @utility name.
func ParseUtilityDefinition(node *AtRule) (*CustomUtility, error) {
	name := Unescape(strings.TrimSpace(node.Params))
	if len(node.Nodes) == 0 {
		return nil, Errorf("`@utility %s` is empty. Utilities without a body are not supported.", name)
	}
	if IsValidFunctionalUtilityName(name) {
		return &CustomUtility{Name: name[:len(name)-2], Kind: UtilityFunctional, Node: node}, nil
	}
	if strings.HasSuffix(name, "*") {
		return nil, Errorf("`@utility %s` defines an invalid utility name. A functional utility must end in `-*`.", name)
	}
	if strings.Contains(name, "*") {
		return nil, Errorf("`@utility %s` defines an invalid utility name. The `*` must be at the end.", name)
	}
	if IsValidStaticUtilityName(name) {
		return &CustomUtility{Name: name, Kind: UtilityStatic, Node: node}, nil
	}
	return nil, Errorf("`@utility %s` defines an invalid utility name. Utilities should be alphabetic and start with a lowercase letter.", name)
}

// Register registers the utility on the design system.
func (cu *CustomUtility) Register(ds *DesignSystem) {
	if cu.Kind == UtilityStatic {
		ds.Utilities.Static(cu.Name, func(*Candidate) ([]Node, CompileStatus) {
			return CloneNodes(cu.Node.Nodes), Handled
		})
		return
	}
	ds.Utilities.Functional(cu.Name, func(c *Candidate) ([]Node, CompileStatus) {
		nodes, ok := CompileFunctionalBody(c, cu.Node.Nodes, ds)
		if !ok {
			return nil, NotHandled
		}
		return nodes, Handled
	}, nil)
}

var argSpacePattern = regexp.MustCompile(`(--[a-zA-Z0-9_-]+?)(?:-\*)?\s+(--)`)
var whitespacePattern = regexp.MustCompile(`\s+`)
var repeatedStarPattern = regexp.MustCompile(`(-\*)+`)
var bareNamespacePattern = regexp.MustCompile(`^--[a-zA-Z0-9_-]+$`)

// NormalizeArgument normalizes one --value(...) or --modifier(...) argument (SPEC §10.6).
func NormalizeArgument(arg string) string {
	a := strings.ReplaceAll(arg, `\*`, "*")
	a = argSpacePattern.ReplaceAllString(a, "$1-*$2")
	a = whitespacePattern.ReplaceAllString(a, "")
	a = repeatedStarPattern.ReplaceAllString(a, "-*")
	if bareNamespacePattern.MatchString(a) && !strings.HasSuffix(a, "-*") {
		a += "-*"
	}
	return a
}

type valueLike struct {
	kind     ValueKind
	value    string
	fraction *string
	dataType *string
}

type resolution struct {
	value string
	ratio bool
}

func resolveArgument(arg string, target *valueLike, ds *DesignSystem) (resolution, bool) {
	if strings.HasPrefix(arg, "--default(") && strings.HasSuffix(arg, ")") {
		if target == nil {
			return resolution{value: arg[len("--default(") : len(arg)-1]}, true
		}
		return resolution{}, false
	}
	if target == nil {
		return resolution{}, false
	}
	if literal, ok := Unquote(arg); ok {
		if target.kind == ValueNamed && target.value == literal {
			return resolution{value: literal}, true
		}
		return resolution{}, false
	}
	if strings.HasPrefix(arg, "--") {
		if target.kind != ValueNamed {
			return resolution{}, false
		}
		star := strings.Index(arg, "-*")
		if star == -1 {
			return resolution{}, false
		}
		namespace := arg[:star]
		sub := arg[star+2:]
		if sub == "" {
			if target.fraction != nil {
				if v, ok := ds.Theme.Resolve(target.fraction, []string{namespace}, 0); ok {
					return resolution{value: v}, true
				}
			}
			if v, ok := ds.Theme.Resolve(strPtr(target.value), []string{namespace}, 0); ok {
				return resolution{value: v}, true
			}
			return resolution{}, false
		}
		if !strings.HasPrefix(sub, "--") {
			return resolution{}, false
		}
		_, extra, ok := ds.Theme.ResolveWith(strPtr(target.value), []string{namespace}, []string{sub})
		if !ok {
			return resolution{}, false
		}
		if v, ok := extra[sub]; ok {
			return resolution{value: v}, true
		}
		return resolution{}, false
	}
	if strings.HasPrefix(arg, "[") && strings.HasSuffix(arg, "]") {
		if target.kind != ValueArbitrary {
			return resolution{}, false
		}
		typ := arg[1 : len(arg)-1]
		if typ == "*" {
			return resolution{value: target.value}, true
		}
		if target.dataType != nil {
			return resolution{value: target.value}, *target.dataType == typ
		}
		if !IsDataType(typ) {
			return resolution{}, false
		}
		return resolution{value: target.value}, InferDataType(target.value, []string{typ}) == typ
	}
	if target.kind != ValueNamed {
		return resolution{}, false
	}
	switch arg {
	case "number":
		return resolution{value: target.value}, IsMultipleOfQuarter(target.value)
	case "integer":
		return resolution{value: target.value}, IsPositiveInteger(target.value)
	case "percentage":
		if len(target.value) > 1 && strings.HasSuffix(target.value, "%") && integerPattern.MatchString(target.value[:len(target.value)-1]) {
			return resolution{value: target.value}, true
		}
		return resolution{}, false
	case "ratio":
		if target.fraction == nil {
			return resolution{}, false
		}
		parts := strings.SplitN(*target.fraction, "/", 2)
		if !IsPositiveInteger(parts[0]) || !IsPositiveInteger(parts[1]) {
			return resolution{}, false
		}
		return resolution{value: parts[0] + " / " + parts[1], ratio: true}, true
	}
	return resolution{}, false
}

// CompileFunctionalBody compiles a functional @utility body for a candidate (SPEC §10.6).
func CompileFunctionalBody(c *Candidate, body []Node, ds *DesignSystem) ([]Node, bool) {
	var valueTarget, modifierTarget *valueLike
	if c.Value != nil {
		valueTarget = &valueLike{kind: c.Value.Kind, value: c.Value.Value, fraction: c.Value.Fraction, dataType: c.Value.DataType}
	}
	if c.Modifier != nil {
		modifierTarget = &valueLike{kind: c.Modifier.Kind, value: c.Modifier.Value}
	}

	nodes := CloneNodes(body)
	sawValueFn, resolvedValue, resolvedRatio := false, false, false
	sawModifierFn, resolvedModifier := false, false
	nonRatio := map[Node]bool{}

	Walk(&nodes, func(n Node, u *WalkUtils) WalkAction {
		if _, ok := n.(*AtRoot); ok {
			return Skip
		}
		d, ok := n.(*Declaration)
		if !ok || d.NoValue {
			return Continue
		}
		if !strings.Contains(d.Value, "--value(") && !strings.Contains(d.Value, "--modifier(") {
			return Continue
		}
		failed, usedRatio, usedNonRatio := false, false, false
		ast := ParseValue(d.Value)
		WalkValue(&ast, func(vn ValueNode, _ *ValueFunction) ([]ValueNode, bool, bool) {
			f, ok := vn.(*ValueFunction)
			if !ok || (f.Name != "--value" && f.Name != "--modifier") {
				return nil, false, false
			}
			isValue := f.Name == "--value"
			target := modifierTarget
			if isValue {
				sawValueFn = true
				target = valueTarget
			} else {
				sawModifierFn = true
			}
			var res resolution
			found := false
			for _, arg := range Segment(ValueToCSS(f.Nodes), ',') {
				arg = NormalizeArgument(strings.TrimSpace(arg))
				if arg == "" {
					continue
				}
				if r, ok := resolveArgument(arg, target, ds); ok {
					res, found = r, true
					break
				}
			}
			if !found {
				failed = true
				return nil, false, true
			}
			if isValue {
				resolvedValue = true
				if res.ratio {
					resolvedRatio, usedRatio = true, true
				} else {
					usedNonRatio = true
				}
			} else {
				resolvedModifier = true
			}
			return []ValueNode{Word(res.value)}, true, true
		})
		if failed {
			u.ReplaceWith()
			return Continue
		}
		d.Value = ValueToCSS(ast)
		if usedNonRatio && !usedRatio {
			nonRatio[d] = true
		}
		return Continue
	})

	if !sawValueFn || !resolvedValue {
		return nil, false
	}
	if sawModifierFn && !resolvedModifier && c.Modifier != nil {
		return nil, false
	}
	if resolvedRatio && resolvedModifier {
		return nil, false
	}
	if c.Modifier != nil && !resolvedRatio && !resolvedModifier {
		return nil, false
	}
	if resolvedRatio && len(nonRatio) > 0 {
		Walk(&nodes, func(n Node, u *WalkUtils) WalkAction {
			if nonRatio[n] {
				u.ReplaceWith()
			}
			return Continue
		})
	}
	return nodes, true
}
