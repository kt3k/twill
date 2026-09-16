package twill

import "testing"

func TestBuiltinVariantsOutput(t *testing.T) {
	ds := designSystemFor(t, "")
	cases := map[string]string{
		"hover:flex":                        block(`.hover\:flex`, block("&:hover", block("@media (hover: hover)", "display: flex;"))),
		"focus:flex":                        block(`.focus\:flex`, "&:focus {\n  display: flex;\n}"),
		"first:flex":                        block(`.first\:flex`, "&:first-child {\n  display: flex;\n}"),
		"odd:flex":                          block(`.odd\:flex`, "&:nth-child(odd) {\n  display: flex;\n}"),
		"open:flex":                         block(`.open\:flex`, "&:is([open], :popover-open, :open) {\n  display: flex;\n}"),
		"inert:flex":                        block(`.inert\:flex`, "&:is([inert], [inert] *) {\n  display: flex;\n}"),
		"ltr:flex":                          block(`.ltr\:flex`, `&:where(:dir(ltr), [dir="ltr"], [dir="ltr"] *) {`+"\n  display: flex;\n}"),
		"before:block":                      ".before\\:block {\n  &::before {\n    @property --tw-content {\n      syntax: \"*\";\n      inherits: false;\n      initial-value: \"\";\n    }\n    content: var(--tw-content);\n    display: block;\n  }\n}\n",
		"marker:flex":                       ".marker\\:flex {\n  & *::marker {\n    display: flex;\n  }\n  &::marker {\n    display: flex;\n  }\n  & *::-webkit-details-marker {\n    display: flex;\n  }\n  &::-webkit-details-marker {\n    display: flex;\n  }\n}\n",
		"placeholder:flex":                  block(`.placeholder\:flex`, "&::placeholder {\n  display: flex;\n}"),
		"*:flex":                            block(`.\*\:flex`, ":is(& > *) {\n  display: flex;\n}"),
		"**:flex":                           block(`.\*\*\:flex`, ":is(& *) {\n  display: flex;\n}"),
		"sm:flex":                           block(`.sm\:flex`, "@media (width >= 40rem) {\n  display: flex;\n}"),
		"max-md:flex":                       block(`.max-md\:flex`, "@media (width < 48rem) {\n  display: flex;\n}"),
		"min-[600px]:flex":                  block(`.min-\[600px\]\:flex`, "@media (width >= 600px) {\n  display: flex;\n}"),
		"min-md:flex":                       block(`.min-md\:flex`, "@media (width >= 48rem) {\n  display: flex;\n}"),
		"@md:flex":                          block(`.\@md\:flex`, "@container (width >= 28rem) {\n  display: flex;\n}"),
		"@md/main:flex":                     block(`.\@md\/main\:flex`, "@container main (width >= 28rem) {\n  display: flex;\n}"),
		"@max-md:flex":                      block(`.\@max-md\:flex`, "@container (width < 28rem) {\n  display: flex;\n}"),
		"@min-[300px]:flex":                 block(`.\@min-\[300px\]\:flex`, "@container (width >= 300px) {\n  display: flex;\n}"),
		"group-hover:flex":                  block(`.group-hover\:flex`, block("&:is(:where(.group):hover *)", block("@media (hover: hover)", "display: flex;"))),
		"group-hover/item:flex":             block(`.group-hover\/item\:flex`, block(`&:is(:where(.group\/item):hover *)`, block("@media (hover: hover)", "display: flex;"))),
		"peer-checked:flex":                 block(`.peer-checked\:flex`, "&:is(:where(.peer):checked ~ *) {\n  display: flex;\n}"),
		"has-[>img]:flex":                   block(`.has-\[\>img\]\:flex`, "&:has(> img) {\n  display: flex;\n}"),
		"has-checked:flex":                  block(`.has-checked\:flex`, "&:has(*:checked) {\n  display: flex;\n}"),
		"in-data-visible:flex":              block(`.in-data-visible\:flex`, ":where(*[data-visible]) & {\n  display: flex;\n}"),
		"not-hover:flex":                    ".not-hover\\:flex {\n  & {\n    &:not(:hover) {\n      display: flex;\n    }\n    @media not all and (hover: hover) {\n      display: flex;\n    }\n  }\n}\n",
		"not-supports-grid:flex":            block(`.not-supports-grid\:flex`, "@supports not (grid: var(--tw)) {\n  display: flex;\n}"),
		"not-sm:flex":                       block(`.not-sm\:flex`, "@media not all and (width >= 40rem) {\n  display: flex;\n}"),
		"not-print:flex":                    block(`.not-print\:flex`, "@media not print {\n  display: flex;\n}"),
		"not-@md:flex":                      block(`.not-\@md\:flex`, "@container not (width >= 28rem) {\n  display: flex;\n}"),
		"not-group-hover:flex":              ".not-group-hover\\:flex {\n  & {\n    &:not(:is(:where(.group):hover *)) {\n      display: flex;\n    }\n    @media not all and (hover: hover) {\n      display: flex;\n    }\n  }\n}\n",
		"aria-checked:flex":                 block(`.aria-checked\:flex`, `&[aria-checked="true"] {`+"\n  display: flex;\n}"),
		"aria-[label=foo]:flex":             block(`.aria-\[label\=foo\]\:flex`, `&[aria-label="foo"] {`+"\n  display: flex;\n}"),
		"data-[state=open]:flex":            block(`.data-\[state\=open\]\:flex`, `&[data-state="open"] {`+"\n  display: flex;\n}"),
		"data-[state=open_i]:flex":          block(`.data-\[state\=open_i\]\:flex`, `&[data-state="open" i] {`+"\n  display: flex;\n}"),
		"data-visible:flex":                 block(`.data-visible\:flex`, "&[data-visible] {\n  display: flex;\n}"),
		"nth-3:flex":                        block(`.nth-3\:flex`, "&:nth-child(3) {\n  display: flex;\n}"),
		"nth-last-[2n+1]:flex":              block(`.nth-last-\[2n\+1\]\:flex`, "&:nth-last-child(2n+1) {\n  display: flex;\n}"),
		"supports-[display:grid]:flex":      block(`.supports-\[display\:grid\]\:flex`, "@supports (display:grid) {\n  display: flex;\n}"),
		"supports-grid:flex":                block(`.supports-grid\:flex`, "@supports (grid: var(--tw)) {\n  display: flex;\n}"),
		"supports-[not(display:grid)]:flex": block(`.supports-\[not\(display\:grid\)\]\:flex`, "@supports not (display:grid) {\n  display: flex;\n}"),
		"dark:flex":                         block(`.dark\:flex`, "@media (prefers-color-scheme: dark) {\n  display: flex;\n}"),
		"print:flex":                        block(`.print\:flex`, "@media print {\n  display: flex;\n}"),
		"motion-reduce:flex":                block(`.motion-reduce\:flex`, "@media (prefers-reduced-motion: reduce) {\n  display: flex;\n}"),
		"starting:flex":                     block(`.starting\:flex`, "@starting-style {\n  display: flex;\n}"),
		"portrait:flex":                     block(`.portrait\:flex`, "@media (orientation: portrait) {\n  display: flex;\n}"),
		"pointer-coarse:flex":               block(`.pointer-coarse\:flex`, "@media (pointer: coarse) {\n  display: flex;\n}"),
		"noscript:flex":                     block(`.noscript\:flex`, "@media (scripting: none) {\n  display: flex;\n}"),
		"[&_p]:flex":                        block(`.\[\&_p\]\:flex`, "& p {\n  display: flex;\n}"),
		"[@media(width>=100px)]:flex":       block(`.\[\@media\(width\>\=100px\)\]\:flex`, "@media (width>=100px) {\n  display: flex;\n}"),
		"[p]:flex":                          block(`.\[p\]\:flex`, "&:is(p) {\n  display: flex;\n}"),
		"dark:hover:flex":                   block(`.dark\:hover\:flex`, block("@media (prefers-color-scheme: dark)", block("&:hover", block("@media (hover: hover)", "display: flex;")))),
	}
	for raw, want := range cases {
		if got := compileRaw(ds, raw); got != want {
			t.Errorf("%s:\n got: %q\nwant: %q", raw, got, want)
		}
	}
	for _, raw := range []string{
		"group-*:flex", "not-before:flex", "min-nope:flex", "sm/foo:flex", "min-[var(--x)]:flex",
		"[>img]:flex", "group-[>img]:flex", "group-sm:flex", "in-hover/x:flex", "not-marker:flex",
		"not-hover/x:flex", "nth-foo:flex", "aria-checked/x:flex",
	} {
		if got := compileRaw(ds, raw); got != "" {
			t.Errorf("%s should be invalid, got %q", raw, got)
		}
	}
}

func TestCustomDarkOverride(t *testing.T) {
	ds := designSystemFor(t, "")
	ds.Variants.Static("dark", func(node Node, _ *Variant) bool {
		children := Children(node)
		*children = []Node{StyleRule("&:where(.dark, .dark *)", *children...)}
		return true
	}, CompoundsStyleRules)
	assertEqual(t, compileRaw(ds, "dark:flex"), block(`.dark\:flex`, "&:where(.dark, .dark *) {\n  display: flex;\n}"))
}

func TestVariantOrdering(t *testing.T) {
	ds := designSystemFor(t, "")
	cmp := func(a, z string) int {
		c := ds.Variants.Compare(ds.ParseVariant(a), ds.ParseVariant(z))
		switch {
		case c < 0:
			return -1
		case c > 0:
			return 1
		}
		return 0
	}
	for _, c := range []struct {
		a, z string
		want int
	}{
		{"hover", "focus", -1}, {"focus", "hover", 1}, {"hover", "hover", 0}, {"sm", "md", -1}, {"md", "lg", -1},
		{"lg", "min-[600px]", 1}, {"max-md", "max-sm", -1}, {"max-lg", "sm", -1}, {"@md", "@lg", -1}, {"sm", "dark", -1},
		{"hover", "[&_p]", -1}, {"[&_a]", "[&_p]", -1}, {"group-hover", "group-focus", -1}, {"group-hover", "group-hover/item", -1},
		{"data-a", "data-b", -1}, {"data-a", "data-[b]", -1}, {"nth-3", "nth-last-3", -1}, {"not-hover", "not-focus", -1}, {"not-hover", "hover", -1},
	} {
		if got := cmp(c.a, c.z); got != c.want {
			t.Errorf("compare(%s, %s) = %d, want %d", c.a, c.z, got, c.want)
		}
	}

	ds = designSystemFor(t, "")
	hover, focus, sm := ds.ParseVariant("hover"), ds.ParseVariant("focus"), ds.ParseVariant("sm")
	order := ds.VariantOrder()
	if !(order[hover] < order[focus] && order[focus] < order[sm]) {
		t.Fatal("order")
	}
	dark := ds.ParseVariant("dark")
	if ds.VariantOrder()[dark] <= ds.VariantOrder()[sm] {
		t.Fatal("cache invalidation")
	}
}

func TestNegateAndQuote(t *testing.T) {
	check := func(got string, ok bool, want string, wantOk bool) {
		t.Helper()
		if ok != wantOk || got != want {
			t.Errorf("got %q %v, want %q %v", got, ok, want, wantOk)
		}
	}
	s, ok := NegateSelector("&:hover")
	check(s, ok, "&:not(:hover)", true)
	s, ok = NegateSelector("&[data-x]")
	check(s, ok, "&:not([data-x])", true)
	s, ok = NegateSelector(":where(.a) &")
	check(s, ok, "&:not(:where(.a) *)", true)
	s, ok = NegateSelector("&:is(:where(.group):hover *)")
	check(s, ok, "&:not(:is(:where(.group):hover *))", true)
	_, ok = NegateSelector("&::before")
	assertEqual(t, ok, false)
	_, ok = NegateSelector("&")
	assertEqual(t, ok, false)
	_, p, _ := NegateAtRule("@media", "(hover: hover)")
	assertEqual(t, p, "not all and (hover: hover)")
	_, p, _ = NegateAtRule("@media", "print")
	assertEqual(t, p, "not print")
	_, p, _ = NegateAtRule("@supports", "(display: grid)")
	assertEqual(t, p, "not (display: grid)")
	_, p, _ = NegateAtRule("@container", "main (width >= 1px)")
	assertEqual(t, p, "main not (width >= 1px)")
	_, _, ok = NegateAtRule("@starting-style", "")
	assertEqual(t, ok, false)

	for in, want := range map[string]string{
		"state": "state", "state=open": `state="open"`, `state="open"`: `state="open"`, "state='open'": "state='open'",
		"state=open i": `state="open" i`, "state^=op": `state^="op"`,
	} {
		got, ok := QuoteAttributeValue(in)
		check(got, ok, want, true)
	}
	for _, in := range []string{"state=", "st ate=open"} {
		if _, ok := QuoteAttributeValue(in); ok {
			t.Errorf("%q should fail", in)
		}
	}
}

func TestVariantsRegistry(t *testing.T) {
	v := NewVariants()
	noop := func(Node, *Variant) bool { return true }
	v.Static("a", noop, CompoundsStyleRules)
	v.Group(func() {
		v.Static("b", noop, CompoundsStyleRules)
		v.Functional("c", noop, CompoundsStyleRules)
	}, nil)
	v.Static("d", noop, CompoundsStyleRules)
	assertEqual(t, v.Get("a").Order, 1)
	assertEqual(t, v.Get("b").Order, 2)
	assertEqual(t, v.Get("c").Order, 2)
	assertEqual(t, v.Get("d").Order, 3)
	v.Functional("a", noop, CompoundsAtRules)
	assertEqual(t, v.Get("a").Order, 1)
	assertEqual(t, v.Get("a").Kind, VariantFunctional)
	assertEqual(t, v.Get("a").Compounds, CompoundsAtRules)
	assertEqual(t, v.Names(), []string{"a", "b", "c", "d"})

	assertEqual(t, CompoundsForSelectors([]string{"&:hover"}), CompoundsStyleRules)
	assertEqual(t, CompoundsForSelectors([]string{"@media (x)"}), CompoundsAtRules)
	assertEqual(t, CompoundsForSelectors([]string{"&:hover", "@supports (x)"}), CompoundsStyleRules|CompoundsAtRules)
	assertEqual(t, CompoundsForSelectors([]string{"@starting-style"}), CompoundsNever)
	assertEqual(t, CompoundsForSelectors([]string{"&::before"}), CompoundsNever)
}
