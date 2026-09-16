package twill

import "testing"

// A stand-in for the registries of §9 and §10.
type fakeContext struct {
	prefix   string
	static   map[string]bool
	fn       map[string]bool
	variants map[string]fakeVariant
	cache    map[string]*Variant
}

type fakeVariant struct {
	kind          VariantKind
	compounds     int
	compoundsWith int
}

const (
	fakeNever      = 0
	fakeAtRules    = 1
	fakeStyleRules = 2
)

func newFakeContext(prefix string) *fakeContext {
	set := func(names ...string) map[string]bool {
		m := map[string]bool{}
		for _, n := range names {
			m[n] = true
		}
		return m
	}
	return &fakeContext{
		prefix: prefix,
		static: set("flex", "block", "underline", "border", "rounded-full", "-mt-px"),
		fn:     set("w", "bg", "p", "mt", "-mt", "border", "border-t", "text", "z", "-z", "grid-cols", "tab", "@"),
		variants: map[string]fakeVariant{
			"*":            {VariantStatic, fakeNever, fakeNever},
			"hover":        {VariantStatic, fakeStyleRules, fakeNever},
			"first":        {VariantStatic, fakeStyleRules, fakeNever},
			"dark":         {VariantStatic, fakeAtRules, fakeNever},
			"sm":           {VariantStatic, fakeAtRules, fakeNever},
			"first-letter": {VariantStatic, fakeNever, fakeNever},
			"data":         {VariantFunctional, fakeStyleRules, fakeNever},
			"aria":         {VariantFunctional, fakeStyleRules, fakeNever},
			"nth":          {VariantFunctional, fakeStyleRules, fakeNever},
			"nth-last":     {VariantFunctional, fakeStyleRules, fakeNever},
			"min":          {VariantFunctional, fakeAtRules, fakeNever},
			"max":          {VariantFunctional, fakeAtRules, fakeNever},
			"supports":     {VariantFunctional, fakeAtRules, fakeNever},
			"@":            {VariantFunctional, fakeAtRules, fakeNever},
			"@min":         {VariantFunctional, fakeAtRules, fakeNever},
			"not":          {VariantCompound, fakeStyleRules | fakeAtRules, fakeStyleRules | fakeAtRules},
			"group":        {VariantCompound, fakeStyleRules, fakeStyleRules},
			"peer":         {VariantCompound, fakeStyleRules, fakeStyleRules},
			"has":          {VariantCompound, fakeStyleRules, fakeStyleRules},
			"in":           {VariantCompound, fakeStyleRules, fakeStyleRules},
		},
		cache: map[string]*Variant{},
	}
}

func (f *fakeContext) ThemePrefix() string { return f.prefix }
func (f *fakeContext) HasUtility(name string, kind UtilityKind) bool {
	if kind == UtilityStatic {
		return f.static[name]
	}
	return f.fn[name]
}
func (f *fakeContext) HasVariant(name string) bool { _, ok := f.variants[name]; return ok }
func (f *fakeContext) VariantKindOf(name string) (VariantKind, bool) {
	v, ok := f.variants[name]
	return v.kind, ok
}
func (f *fakeContext) compoundsOf(v *Variant) int {
	if v.Kind == VariantArbitrary {
		s := v.Selector
		if len(s) > 0 && s[0] == '@' {
			for _, p := range []string{"@media", "@supports", "@container"} {
				if len(s) >= len(p) && s[:len(p)] == p {
					return fakeAtRules
				}
			}
			return fakeNever
		}
		for i := 0; i+1 < len(s); i++ {
			if s[i] == ':' && s[i+1] == ':' {
				return fakeNever
			}
		}
		return fakeStyleRules
	}
	return f.variants[v.Root].compounds
}
func (f *fakeContext) CompoundsWith(parent string, child *Variant) bool {
	p := f.variants[parent]
	if p.kind != VariantCompound {
		return false
	}
	c := f.compoundsOf(child)
	return c != fakeNever && p.compoundsWith != fakeNever && c&p.compoundsWith != 0
}
func (f *fakeContext) ParseVariant(input string) *Variant {
	if v, ok := f.cache[input]; ok {
		return v
	}
	v := ParseVariant(input, f)
	f.cache[input] = v
	return v
}

var fake = newFakeContext("")

func candidates(input string) []*Candidate { return ParseCandidate(input, fake) }

func named(value string, fraction *string) *CandidateValue {
	return &CandidateValue{Kind: ValueNamed, Value: value, Fraction: fraction}
}

func arbitrary(value string, dataType *string) *CandidateValue {
	return &CandidateValue{Kind: ValueArbitrary, Value: value, DataType: dataType}
}

func TestParseCandidateBasics(t *testing.T) {
	assertEqual(t, candidates("flex"), []*Candidate{{Kind: CandidateStatic, Root: "flex", Variants: []*Variant{}, Raw: "flex"}})
	assertEqual(t, candidates("w-4"), []*Candidate{{Kind: CandidateFunctional, Root: "w", Value: named("4", nil), Variants: []*Variant{}, Raw: "w-4"}})
	assertEqual(t, candidates("border"), []*Candidate{
		{Kind: CandidateStatic, Root: "border", Variants: []*Variant{}, Raw: "border"},
		{Kind: CandidateFunctional, Root: "border", Variants: []*Variant{}, Raw: "border"},
	})
	assertEqual(t, candidates("border-t-2"), []*Candidate{
		{Kind: CandidateFunctional, Root: "border-t", Value: named("2", nil), Variants: []*Variant{}, Raw: "border-t-2"},
		{Kind: CandidateFunctional, Root: "border", Value: named("t-2", nil), Variants: []*Variant{}, Raw: "border-t-2"},
	})
	assertEqual(t, candidates("-mt-2"), []*Candidate{{Kind: CandidateFunctional, Root: "-mt", Value: named("2", nil), Variants: []*Variant{}, Raw: "-mt-2"}})
	assertEqual(t, len(candidates("unknown")), 0)
	assertEqual(t, len(candidates("w-a$b")), 0)
	assertEqual(t, len(candidates("w-")), 0)
	assertEqual(t, candidates("@container")[0].Root, "@")
}

func TestParseCandidateModifiers(t *testing.T) {
	assertEqual(t, candidates("w-1/2"), []*Candidate{{Kind: CandidateFunctional, Root: "w", Value: named("1", strPtr("1/2")), Modifier: &Modifier{ValueNamed, "2"}, Variants: []*Variant{}, Raw: "w-1/2"}})
	assertEqual(t, candidates("bg-red-500/[0.5]")[0].Modifier, &Modifier{ValueArbitrary, "0.5"})
	assertEqual(t, candidates("bg-red-500/[0.5]")[0].Value, named("red-500", nil))
	assertEqual(t, candidates("bg-red-500/(--alpha)")[0].Modifier, &Modifier{ValueArbitrary, "var(--alpha)"})
	for _, in := range []string{"bg-red-500/50/50", "bg-red-500/[]", "bg-red-500/(foo)", "bg-red-500/a b"} {
		assertEqual(t, len(candidates(in)), 0)
	}
}

func TestParseCandidateArbitrary(t *testing.T) {
	assertEqual(t, candidates("w-[13px]")[0].Value, arbitrary("13px", nil))
	assertEqual(t, candidates("bg-[length:10px_20px]")[0].Value, arbitrary("10px 20px", strPtr("length")))
	assertEqual(t, candidates("bg-[url(/a_b.png)]")[0].Value, arbitrary("url(/a_b.png)", nil))
	assertEqual(t, candidates("bg-[#0088cc]/50")[0].Modifier, &Modifier{ValueNamed, "50"})
	for _, in := range []string{"w-[]", "w-[_]", "w-[:1px]", "w-[1px;]", "w-[1px}]", "w-[1px", "unknown-[1px]", "w-(my-w)", "w-(--a:--b:--c)", "unknown-(--x)"} {
		if len(candidates(in)) != 0 {
			t.Errorf("%q should be invalid", in)
		}
	}
	assertEqual(t, candidates("w-['a;b']")[0].Kind, CandidateFunctional)
	assertEqual(t, candidates("w-[calc(1px;)]")[0].Value.Value, "calc(1px;)")
	assertEqual(t, candidates("w-(--my-w)")[0].Value, arbitrary("var(--my-w)", nil))
	assertEqual(t, candidates("bg-(color:--my-color)")[0].Value, arbitrary("var(--my-color)", strPtr("color")))

	assertEqual(t, candidates("[mask-type:luminance]"), []*Candidate{{Kind: CandidateArbitrary, Property: "mask-type", ArbitraryValue: "luminance", Variants: []*Variant{}, Raw: "[mask-type:luminance]"}})
	assertEqual(t, candidates("[--my-var:1px]")[0].Property, "--my-var")
	assertEqual(t, candidates("[color:red]/50")[0].Modifier, &Modifier{ValueNamed, "50"})
	assertEqual(t, candidates("[color:a_b]")[0].ArbitraryValue, "a b")
	for _, in := range []string{"[Color:red]", "[color]", "[:red]", "[color:]", "[color:red", "[color:red;]"} {
		if len(candidates(in)) != 0 {
			t.Errorf("%q should be invalid", in)
		}
	}
}

func TestParseCandidateVariantsAndImportance(t *testing.T) {
	assertEqual(t, candidates("underline!")[0].Important, true)
	assertEqual(t, candidates("!underline")[0].Important, true)
	c := candidates("hover:w-4!")[0]
	assertEqual(t, c.Important, true)
	assertEqual(t, c.Variants, []*Variant{{Kind: VariantStatic, Root: "hover"}})
	assertEqual(t, len(candidates("!underline!")), 0)
	c = candidates("dark:hover:flex")[0]
	assertEqual(t, c.Variants, []*Variant{{Kind: VariantStatic, Root: "hover"}, {Kind: VariantStatic, Root: "dark"}})
	assertEqual(t, len(candidates("unknown:flex")), 0)
	assertEqual(t, len(candidates("hover:unknown")), 0)

	prefixed := newFakeContext("tw")
	assertEqual(t, len(ParseCandidate("flex", prefixed)), 0)
	assertEqual(t, len(ParseCandidate("foo:flex", prefixed)), 0)
	assertEqual(t, len(ParseCandidate("tw:flex", prefixed)), 1)
	assertEqual(t, ParseCandidate("tw:hover:flex", prefixed)[0].Variants, []*Variant{{Kind: VariantStatic, Root: "hover"}})
}

func TestParseModifierAndFindRoots(t *testing.T) {
	assertEqual(t, ParseModifier("50"), &Modifier{ValueNamed, "50"})
	assertEqual(t, ParseModifier("[a_b]"), &Modifier{ValueArbitrary, "a b"})
	assertEqual(t, ParseModifier("(--x)"), &Modifier{ValueArbitrary, "var(--x)"})
	for _, in := range []string{"[]", "(x)", "a b"} {
		if ParseModifier(in) != nil {
			t.Errorf("%q should be nil", in)
		}
	}
	exists := func(r string) bool { return r == "border" || r == "border-t" || r == "@" }
	assertEqual(t, FindRoots("border", exists), []RootMatch{{Root: "border"}})
	assertEqual(t, FindRoots("border-t-2", exists), []RootMatch{{Root: "border-t", Value: strPtr("2")}, {Root: "border", Value: strPtr("t-2")}})
	assertEqual(t, len(FindRoots("border-", exists)), 0)
	assertEqual(t, len(FindRoots("border-t-", exists)), 0)
	assertEqual(t, FindRoots("@md", exists), []RootMatch{{Root: "@", Value: strPtr("md")}})
	assertEqual(t, FindRoots("@min-md", exists), []RootMatch{{Root: "@", Value: strPtr("min-md")}})
	assertEqual(t, len(FindRoots("nope-1", exists)), 0)
}

func variant(input string) *Variant { return ParseVariant(input, fake) }

func TestParseVariant(t *testing.T) {
	assertEqual(t, variant("hover"), &Variant{Kind: VariantStatic, Root: "hover"})
	assertEqual(t, variant("*"), &Variant{Kind: VariantStatic, Root: "*"})
	for _, in := range []string{"hover-x", "hover/x", "unknown", "data-[]", "data-(x)", "data-a$b", "data-x/[]", "group-sm", "group-*", "group-first-letter", "group", "group-unknown", "[@media(width>=100px)_&]", "[]", "[_]", "[a;b]"} {
		if variant(in) != nil {
			t.Errorf("%q should be nil", in)
		}
	}
	assertEqual(t, variant("data-visible"), &Variant{Kind: VariantFunctional, Root: "data", Value: &VariantValue{ValueNamed, "visible"}})
	assertEqual(t, variant("data-[state=open]"), &Variant{Kind: VariantFunctional, Root: "data", Value: &VariantValue{ValueArbitrary, "state=open"}})
	assertEqual(t, variant("supports-(--x)"), &Variant{Kind: VariantFunctional, Root: "supports", Value: &VariantValue{ValueArbitrary, "var(--x)"}})
	assertEqual(t, variant("nth-last-3"), &Variant{Kind: VariantFunctional, Root: "nth-last", Value: &VariantValue{ValueNamed, "3"}})
	assertEqual(t, variant("min-[600px]"), &Variant{Kind: VariantFunctional, Root: "min", Value: &VariantValue{ValueArbitrary, "600px"}})
	assertEqual(t, variant("@md/main"), &Variant{Kind: VariantFunctional, Root: "@", Value: &VariantValue{ValueNamed, "md"}, Modifier: &Modifier{ValueNamed, "main"}})
	assertEqual(t, variant("@min-md"), &Variant{Kind: VariantFunctional, Root: "@min", Value: &VariantValue{ValueNamed, "md"}})
	assertEqual(t, variant("data"), &Variant{Kind: VariantFunctional, Root: "data"})

	assertEqual(t, variant("group-hover"), &Variant{Kind: VariantCompound, Root: "group", Inner: &Variant{Kind: VariantStatic, Root: "hover"}})
	assertEqual(t, variant("group-hover/item"), &Variant{Kind: VariantCompound, Root: "group", Modifier: &Modifier{ValueNamed, "item"}, Inner: &Variant{Kind: VariantStatic, Root: "hover"}})
	assertEqual(t, variant("not-sm"), &Variant{Kind: VariantCompound, Root: "not", Inner: &Variant{Kind: VariantStatic, Root: "sm"}})
	assertEqual(t, variant("in-data-visible"), &Variant{Kind: VariantCompound, Root: "in", Inner: &Variant{Kind: VariantFunctional, Root: "data", Value: &VariantValue{ValueNamed, "visible"}}})
	assertEqual(t, variant("has-[>img]"), &Variant{Kind: VariantCompound, Root: "has", Inner: &Variant{Kind: VariantArbitrary, Selector: ">img", Relative: true}})
	assertEqual(t, variant("not-group-hover/item"), &Variant{Kind: VariantCompound, Root: "not", Inner: &Variant{Kind: VariantCompound, Root: "group", Modifier: &Modifier{ValueNamed, "item"}, Inner: &Variant{Kind: VariantStatic, Root: "hover"}}})

	assertEqual(t, variant("[&_p]"), &Variant{Kind: VariantArbitrary, Selector: "& p"})
	assertEqual(t, variant("[p]"), &Variant{Kind: VariantArbitrary, Selector: "&:is(p)"})
	assertEqual(t, variant("[>img]"), &Variant{Kind: VariantArbitrary, Selector: ">img", Relative: true})
	assertEqual(t, variant("[@media(width>=100px)]"), &Variant{Kind: VariantArbitrary, Selector: "@media(width>=100px)"})
	if fake.ParseVariant("hover") != fake.ParseVariant("hover") {
		t.Fatal("memoization")
	}
}
