package twill

import (
	"regexp"
	"strconv"
	"strings"
)

// PropertyRegistration builds `@property --tw-<name> { ... }` inside an at-root.
func PropertyRegistration(name string, initialValue *string, syntax string, inherits bool) Node {
	nodes := []Node{Decl("syntax", `"`+syntax+`"`)}
	if inherits {
		nodes = append(nodes, Decl("inherits", "true"))
	} else {
		nodes = append(nodes, Decl("inherits", "false"))
	}
	if initialValue != nil {
		nodes = append(nodes, Decl("initial-value", *initialValue))
	}
	return NewAtRoot(NewAtRule("@property", name, nodes...))
}

// Property is a shorthand for a `syntax: "*"; inherits: false` registration.
func Property(name string, initialValue *string) Node {
	return PropertyRegistration(name, initialValue, "*", false)
}

func replaceAmpersand(selector, replacement string) string {
	return strings.ReplaceAll(selector, "&", replacement)
}

// NegateSelector negates a style selector produced by a variant.
func NegateSelector(selector string) (string, bool) {
	if strings.Contains(selector, "::") {
		return "", false
	}
	var parts []string
	for _, part := range Segment(selector, ',') {
		part = strings.TrimSpace(part)
		if part == "&" {
			return "", false
		}
		if strings.HasPrefix(part, "&") {
			rest := part[1:]
			if !strings.Contains(rest, "&") && (strings.HasPrefix(rest, ":") || strings.HasPrefix(rest, "[")) {
				parts = append(parts, "&:not("+rest+")")
				continue
			}
		}
		parts = append(parts, "&:not("+replaceAmpersand(part, "*")+")")
	}
	return strings.Join(parts, ", "), true
}

// NegateAtRule negates an at-rule produced by a variant.
func NegateAtRule(name, params string) (string, string, bool) {
	switch name {
	case "@media":
		if strings.HasPrefix(params, "not ") {
			return name, params[4:], true
		}
		if strings.HasPrefix(params, "(") {
			return name, "not all and " + params, true
		}
		return name, "not " + params, true
	case "@supports":
		if strings.HasPrefix(params, "not ") {
			return name, params[4:], true
		}
		return name, "not " + params, true
	case "@container":
		if strings.HasPrefix(params, "(") {
			return name, "not " + params, true
		}
		space := strings.Index(params, " ")
		if space == -1 {
			return "", "", false
		}
		containerName := params[:space]
		rest := strings.TrimSpace(params[space+1:])
		if strings.HasPrefix(rest, "not ") {
			return name, containerName + " " + rest[4:], true
		}
		return name, containerName + " not " + rest, true
	}
	return "", "", false
}

var attributeNamePattern = regexp.MustCompile(`^[a-zA-Z_:][-a-zA-Z0-9_:.]*$`)
var attributeFlagPattern = regexp.MustCompile(`\s+([isIS])$`)

// QuoteAttributeValue wraps an unquoted attribute value in double quotes,
// keeping a trailing ` i` or ` s` flag outside the quotes.
func QuoteAttributeValue(value string) (string, bool) {
	eq := strings.Index(value, "=")
	if eq == -1 {
		return value, attributeNamePattern.MatchString(value)
	}
	name := value[:eq]
	rest := value[eq+1:]
	operator := "="
	if name != "" {
		last := name[len(name)-1]
		if last == '~' || last == '|' || last == '^' || last == '$' || last == '*' {
			operator = string(last) + "="
			name = name[:len(name)-1]
		}
	}
	if !attributeNamePattern.MatchString(name) {
		return "", false
	}
	flag := ""
	if loc := attributeFlagPattern.FindStringSubmatchIndex(rest); loc != nil {
		flag = " " + rest[loc[2]:loc[3]]
		rest = rest[:loc[0]]
	}
	rest = strings.TrimSpace(rest)
	if rest == "" {
		return "", false
	}
	quoted := (strings.HasPrefix(rest, `"`) && strings.HasSuffix(rest, `"`)) || (strings.HasPrefix(rest, "'") && strings.HasSuffix(rest, "'"))
	if !quoted {
		if strings.Contains(rest, `"`) {
			return "", false
		}
		rest = `"` + rest + `"`
	}
	return name + operator + rest + flag, true
}

func resolveQueryValue(theme *Theme, variant *Variant, namespace string) (string, bool) {
	if variant.Kind == VariantStatic {
		return theme.ResolveValue(strPtr(variant.Root), []string{namespace})
	}
	if variant.Kind != VariantFunctional || variant.Value == nil {
		return "", false
	}
	var value string
	if variant.Value.Kind == ValueArbitrary {
		value = variant.Value.Value
	} else {
		v, ok := theme.ResolveValue(strPtr(variant.Value.Value), []string{namespace})
		if !ok {
			return "", false
		}
		value = v
	}
	if strings.Contains(value, "var(") {
		return "", false
	}
	return value, true
}

var numberPrefixPattern = regexp.MustCompile(`^-?[0-9.]+`)

func bucketOf(value string) string {
	if paren := strings.Index(value, "("); paren != -1 && strings.HasSuffix(value, ")") {
		return value[:paren]
	}
	if loc := numberPrefixPattern.FindStringIndex(value); loc != nil {
		return value[loc[1]:]
	}
	return value
}

func compareQueryValues(theme *Theme, namespace string, ascending bool) VariantCompareFn {
	return func(a, z *Variant) int {
		if a == z {
			return 0
		}
		aValue, aOk := resolveQueryValue(theme, a, namespace)
		zValue, zOk := resolveQueryValue(theme, z, namespace)
		switch {
		case !aOk && !zOk:
			return 0
		case !aOk:
			if ascending {
				return -1
			}
			return 1
		case !zOk:
			if ascending {
				return 1
			}
			return -1
		case aValue == zValue:
			return 0
		}
		aBucket, zBucket := bucketOf(aValue), bucketOf(zValue)
		if aBucket != zBucket {
			return compareStrings(aBucket, zBucket)
		}
		aNumber, aErr := strconv.ParseFloat(numberPrefixPattern.FindString(aValue), 64)
		zNumber, zErr := strconv.ParseFloat(numberPrefixPattern.FindString(zValue), 64)
		if aErr != nil || zErr != nil {
			return compareStrings(aValue, zValue)
		}
		diff := aNumber - zNumber
		if !ascending {
			diff = -diff
		}
		switch {
		case diff < 0:
			return -1
		case diff > 0:
			return 1
		}
		return 0
	}
}

// RegisterBuiltinVariants registers every built-in variant in the order of SPEC §9.5.
func RegisterBuiltinVariants(v *Variants, theme *Theme) {
	static := func(name string, selectors ...string) { StaticVariant(v, name, selectors, 0, true) }
	never := func(name string, selectors ...string) { StaticVariant(v, name, selectors, CompoundsNever, false) }

	never("*", ":is(& > *)")
	never("**", ":is(& *)")

	v.Compound("not", CompoundsStyleRules|CompoundsAtRules, func(node Node, variant *Variant) bool {
		if variant.Modifier != nil {
			return false
		}
		return negate(node)
	}, CompoundsStyleRules|CompoundsAtRules)

	v.Compound("group", CompoundsStyleRules, func(node Node, variant *Variant) bool {
		return groupLike(node, variant, "group", "*", theme)
	}, CompoundsStyleRules)
	v.Compound("peer", CompoundsStyleRules, func(node Node, variant *Variant) bool {
		return groupLike(node, variant, "peer", "~ *", theme)
	}, CompoundsStyleRules)

	never("first-letter", "&::first-letter")
	never("first-line", "&::first-line")
	never("marker", "& *::marker", "&::marker", "& *::-webkit-details-marker", "&::-webkit-details-marker")
	never("selection", "& *::selection", "&::selection")
	never("file", "&::file-selector-button")
	never("placeholder", "&::placeholder")
	never("backdrop", "&::backdrop")
	never("details-content", "&::details-content")

	for _, name := range []string{"before", "after"} {
		pseudo := "&::" + name
		v.Static(name, func(node Node, _ *Variant) bool {
			children := Children(node)
			nodes := []Node{Property("--tw-content", strPtr(`""`)), Decl("content", "var(--tw-content)")}
			nodes = append(nodes, *children...)
			*children = []Node{StyleRule(pseudo, nodes...)}
			return true
		}, CompoundsNever)
	}

	pseudos := [][2]string{
		{"first", "&:first-child"}, {"last", "&:last-child"}, {"only", "&:only-child"},
		{"odd", "&:nth-child(odd)"}, {"even", "&:nth-child(even)"}, {"first-of-type", "&:first-of-type"},
		{"last-of-type", "&:last-of-type"}, {"only-of-type", "&:only-of-type"}, {"visited", "&:visited"},
		{"target", "&:target"}, {"open", "&:is([open], :popover-open, :open)"}, {"default", "&:default"},
		{"checked", "&:checked"}, {"indeterminate", "&:indeterminate"}, {"placeholder-shown", "&:placeholder-shown"},
		{"autofill", "&:autofill"}, {"optional", "&:optional"}, {"required", "&:required"}, {"valid", "&:valid"},
		{"invalid", "&:invalid"}, {"user-valid", "&:user-valid"}, {"user-invalid", "&:user-invalid"},
		{"in-range", "&:in-range"}, {"out-of-range", "&:out-of-range"}, {"read-only", "&:read-only"},
		{"empty", "&:empty"}, {"focus-within", "&:focus-within"},
	}
	for _, p := range pseudos {
		static(p[0], p[1])
	}

	v.Static("hover", func(node Node, _ *Variant) bool {
		children := Children(node)
		*children = []Node{StyleRule("&:hover", NewAtRule("@media", "(hover: hover)", *children...))}
		return true
	}, CompoundsStyleRules)

	for _, name := range []string{"focus", "focus-visible", "active", "enabled", "disabled"} {
		static(name, "&:"+name)
	}
	static("inert", "&:is([inert], [inert] *)")

	v.Compound("in", CompoundsStyleRules, func(node Node, variant *Variant) bool {
		rule, ok := node.(*Rule)
		if variant.Modifier != nil || !ok {
			return false
		}
		rule.Selector = ":where(" + replaceAmpersand(rule.Selector, "*") + ") &"
		return true
	}, CompoundsStyleRules)

	relativePattern := regexp.MustCompile(`^[>+~][^ ]`)
	v.Compound("has", CompoundsStyleRules, func(node Node, variant *Variant) bool {
		rule, ok := node.(*Rule)
		if variant.Modifier != nil || !ok {
			return false
		}
		selector := replaceAmpersand(rule.Selector, "*")
		if relativePattern.MatchString(selector) {
			selector = selector[:1] + " " + selector[1:]
		}
		rule.Selector = "&:has(" + selector + ")"
		return true
	}, CompoundsStyleRules)

	v.Functional("aria", func(node Node, variant *Variant) bool {
		if variant.Value == nil || variant.Modifier != nil {
			return false
		}
		children := Children(node)
		if variant.Value.Kind == ValueNamed {
			*children = []Node{StyleRule(`&[aria-`+variant.Value.Value+`="true"]`, *children...)}
			return true
		}
		attribute, ok := QuoteAttributeValue(variant.Value.Value)
		if !ok {
			return false
		}
		*children = []Node{StyleRule("&[aria-"+attribute+"]", *children...)}
		return true
	}, CompoundsStyleRules)

	v.Functional("data", func(node Node, variant *Variant) bool {
		if variant.Value == nil || variant.Modifier != nil {
			return false
		}
		attribute, ok := QuoteAttributeValue(variant.Value.Value)
		if !ok {
			return false
		}
		children := Children(node)
		*children = []Node{StyleRule("&[data-"+attribute+"]", *children...)}
		return true
	}, CompoundsStyleRules)

	for _, nth := range [][2]string{{"nth", "nth-child"}, {"nth-last", "nth-last-child"}, {"nth-of-type", "nth-of-type"}, {"nth-last-of-type", "nth-last-of-type"}} {
		pseudo := nth[1]
		v.Functional(nth[0], func(node Node, variant *Variant) bool {
			if variant.Value == nil || variant.Modifier != nil {
				return false
			}
			if variant.Value.Kind == ValueNamed && !IsPositiveInteger(variant.Value.Value) {
				return false
			}
			children := Children(node)
			*children = []Node{StyleRule("&:"+pseudo+"("+variant.Value.Value+")", *children...)}
			return true
		}, CompoundsStyleRules)
	}

	supportsCall := regexp.MustCompile(`^[\w-]*\s*\(`)
	supportsKeyword := regexp.MustCompile(`\b(and|or|not)\(`)
	v.Functional("supports", func(node Node, variant *Variant) bool {
		if variant.Value == nil || variant.Modifier != nil {
			return false
		}
		value := variant.Value.Value
		var condition string
		switch {
		case supportsCall.MatchString(value):
			condition = supportsKeyword.ReplaceAllString(value, "$1 (")
		case !strings.Contains(value, ":"):
			condition = "(" + value + ": var(--tw))"
		case strings.HasPrefix(value, "(") && strings.HasSuffix(value, ")"):
			condition = value
		default:
			condition = "(" + value + ")"
		}
		children := Children(node)
		*children = []Node{NewAtRule("@supports", condition, *children...)}
		return true
	}, CompoundsAtRules)

	for _, m := range [][2]string{
		{"motion-safe", "(prefers-reduced-motion: no-preference)"}, {"motion-reduce", "(prefers-reduced-motion: reduce)"},
		{"contrast-more", "(prefers-contrast: more)"}, {"contrast-less", "(prefers-contrast: less)"},
	} {
		static(m[0], "@media "+m[1])
	}

	mediaQuery := func(namespace, operator string) VariantApply {
		return func(node Node, variant *Variant) bool {
			if variant.Value == nil || variant.Modifier != nil {
				return false
			}
			value, ok := resolveQueryValue(theme, variant, namespace)
			if !ok {
				return false
			}
			children := Children(node)
			*children = []Node{NewAtRule("@media", "(width "+operator+" "+value+")", *children...)}
			return true
		}
	}

	v.Group(func() {
		v.Functional("max", mediaQuery("--breakpoint", "<"), CompoundsAtRules)
	}, compareQueryValues(theme, "--breakpoint", false))

	v.Group(func() {
		for _, entry := range theme.Namespace("--breakpoint") {
			if entry.Self || strings.Contains(entry.Key, "--") {
				continue
			}
			value := entry.Value
			v.Static(entry.Key, func(node Node, _ *Variant) bool {
				children := Children(node)
				*children = []Node{NewAtRule("@media", "(width >= "+value+")", *children...)}
				return true
			}, CompoundsAtRules)
		}
		v.Functional("min", mediaQuery("--breakpoint", ">="), CompoundsAtRules)
	}, compareQueryValues(theme, "--breakpoint", true))

	containerQuery := func(operator string) VariantApply {
		return func(node Node, variant *Variant) bool {
			if variant.Value == nil {
				return false
			}
			if variant.Modifier != nil && variant.Modifier.Kind != ValueNamed {
				return false
			}
			value, ok := resolveQueryValue(theme, variant, "--container")
			if !ok {
				return false
			}
			name := ""
			if variant.Modifier != nil {
				name = variant.Modifier.Value + " "
			}
			children := Children(node)
			*children = []Node{NewAtRule("@container", name+"(width "+operator+" "+value+")", *children...)}
			return true
		}
	}

	v.Group(func() {
		v.Functional("@max", containerQuery("<"), CompoundsAtRules)
	}, compareQueryValues(theme, "--container", false))
	v.Group(func() {
		v.Functional("@", containerQuery(">="), CompoundsAtRules)
		v.Functional("@min", containerQuery(">="), CompoundsAtRules)
	}, compareQueryValues(theme, "--container", true))

	static("portrait", "@media (orientation: portrait)")
	static("landscape", "@media (orientation: landscape)")
	static("ltr", `&:where(:dir(ltr), [dir="ltr"], [dir="ltr"] *)`)
	static("rtl", `&:where(:dir(rtl), [dir="rtl"], [dir="rtl"] *)`)
	static("dark", "@media (prefers-color-scheme: dark)")
	never("starting", "@starting-style")
	static("print", "@media print")
	static("forced-colors", "@media (forced-colors: active)")
	static("inverted-colors", "@media (inverted-colors: inverted)")
	static("pointer-none", "@media (pointer: none)")
	static("pointer-coarse", "@media (pointer: coarse)")
	static("pointer-fine", "@media (pointer: fine)")
	static("any-pointer-none", "@media (any-pointer: none)")
	static("any-pointer-coarse", "@media (any-pointer: coarse)")
	static("any-pointer-fine", "@media (any-pointer: fine)")
	static("noscript", "@media (scripting: none)")
}

// negate negates each rule produced by an inner variant.
func negate(node Node) bool {
	var result []Node
	failed := false
	nodes := []Node{node}
	Walk(&nodes, func(child Node, u *WalkUtils) WalkAction {
		cc := Children(child)
		switch child.(type) {
		case *Rule, *AtRule:
		default:
			return Continue
		}
		if len(*cc) > 0 {
			return Continue
		}
		var styleRules []string
		var atRules [][2]string
		for _, ancestor := range append(append([]Node{}, u.Path...), child) {
			switch a := ancestor.(type) {
			case *Rule:
				styleRules = append(styleRules, a.Selector)
			case *AtRule:
				atRules = append(atRules, [2]string{a.Name, a.Params})
			}
		}
		if len(styleRules) > 1 || len(atRules) > 1 || result != nil {
			failed = true
			return Stop
		}
		var rules []Node
		for _, selector := range styleRules {
			negated, ok := NegateSelector(selector)
			if !ok {
				failed = true
				return Stop
			}
			rules = append(rules, StyleRule(negated))
		}
		for _, at := range atRules {
			name, params, ok := NegateAtRule(at[0], at[1])
			if !ok {
				failed = true
				return Stop
			}
			rules = append(rules, NewAtRule(name, params))
		}
		result = rules
		return Skip
	})
	if failed || result == nil {
		return false
	}
	var replacement Node
	if len(result) == 1 {
		replacement = result[0]
	} else {
		replacement = StyleRule("&", result...)
	}
	switch target := node.(type) {
	case *Rule:
		switch r := replacement.(type) {
		case *Rule:
			*target = *r
		case *AtRule:
			// A style rule cannot become an at-rule in place; wrap instead.
			*target = Rule{Selector: "&", Nodes: []Node{r}}
		}
	case *AtRule:
		switch r := replacement.(type) {
		case *AtRule:
			*target = *r
		case *Rule:
			*target = AtRule{Name: "@media", Params: "all", Nodes: []Node{r}}
		}
	}
	return true
}

func groupLike(node Node, variant *Variant, className, combinator string, theme *Theme) bool {
	rule, ok := node.(*Rule)
	if !ok {
		return false
	}
	if variant.Modifier != nil && variant.Modifier.Kind != ValueNamed {
		return false
	}
	if variant.Inner.Kind == VariantArbitrary && variant.Inner.Relative {
		return false
	}
	if HasNestedStyleRules(rule.Nodes) {
		return false
	}
	marker := className
	if variant.Modifier != nil {
		marker = className + "/" + variant.Modifier.Value
	}
	if theme.Prefix != "" {
		marker = theme.Prefix + ":" + marker
	}
	selector := replaceAmpersand(rule.Selector, ":where(."+Escape(marker)+")")
	if len(Segment(selector, ',')) > 1 {
		selector = ":is(" + selector + ")"
	}
	rule.Selector = "&:is(" + selector + " " + combinator + ")"
	return true
}
