package twill

import (
	"sort"
	"testing"
)

func TestEscapeUnescape(t *testing.T) {
	cases := map[string]string{
		"hover:flex":             `hover\:flex`,
		"w-1/2":                  `w-1\/2`,
		"bg-[#0088cc]":           `bg-\[\#0088cc\]`,
		"2xl:p-4":                `\32 xl\:p-4`,
		"-1":                     `-\31 `,
		"-":                      `\-`,
		"a\x01b":                 `a\1 b`,
		"日本":                     "日本",
		"data-[state=open]:flex": `data-\[state\=open\]\:flex`,
		"*:flex":                 `\*\:flex`,
	}
	for in, want := range cases {
		assertEqual(t, Escape(in), want)
	}
	assertEqual(t, Unescape(`hover\:flex`), "hover:flex")
	assertEqual(t, Unescape(`\32 xl`), "2xl")
	assertEqual(t, Unescape(`foo-1\/2`), "foo-1/2")
	assertEqual(t, Unescape("plain"), "plain")
}

func TestSegmentAndArbitrary(t *testing.T) {
	assertEqual(t, Segment("a:b:c", ':'), []string{"a", "b", "c"})
	assertEqual(t, Segment("a:[b:c]:d", ':'), []string{"a", "[b:c]", "d"})
	assertEqual(t, Segment("a:(b:c):d", ':'), []string{"a", "(b:c)", "d"})
	assertEqual(t, Segment("a:{b:c}:d", ':'), []string{"a", "{b:c}", "d"})
	assertEqual(t, Segment(`a:"b:c":d`, ':'), []string{"a", `"b:c"`, "d"})
	assertEqual(t, Segment(`a\:b:c`, ':'), []string{`a\:b`, "c"})
	assertEqual(t, Segment("a:[b:(c]:d)]:e", ':'), []string{"a", "[b:(c]:d)]", "e"})
	assertEqual(t, Segment("", ':'), []string{""})
	assertEqual(t, Segment("a::b", ':'), []string{"a", "", "b"})

	for in, want := range map[string]bool{
		"calc(1px + 2px)": true, "a]": false, "(a]": false, "a;b": false, "(a;b)": true,
		"'a;b'": true, "a}": false, `a\]`: true,
	} {
		assertEqual(t, IsValidArbitrary(in), want)
	}
}

func TestDecodeArbitraryValue(t *testing.T) {
	cases := map[string]string{
		"10px_20px":                 "10px 20px",
		`a\_b_c`:                    "a_b c",
		"url(/a_b.png)":             "url(/a_b.png)",
		"image_url(/a_b.png)":       "image_url(/a_b.png)",
		"var(--my_var)":             "var(--my_var)",
		"var(--my_var,1px_2px)":     "var(--my_var,1px 2px)",
		`var(--my\_var)`:            "var(--my_var)",
		"theme(--spacing_x)":        "theme(--spacing_x)",
		"calc(1px+2px)":             "calc(1px + 2px)",
		"calc(1px_+_2px)":           "calc(1px + 2px)",
		"calc(100%-var(--x))":       "calc(100% - var(--x))",
		"calc(var(--x)*2)":          "calc(var(--x) * 2)",
		"calc(1px*-1)":              "calc(1px * -1)",
		"min(1px,2px)":              "min(1px,2px)",
		"rgb(0_0_0_/_0.5)":          "rgb(0 0 0 / 0.5)",
		"calc(-1*var(--x))":         "calc(-1 * var(--x))",
		"'a_b'":                     "'a b'",
		"calc(1rem-2px)":            "calc(1rem - 2px)",
		"min(100%,max-content)":     "min(100%,max-content)",
		"calc(-1px)":                "calc(-1px)",
		"calc(var(--a)+var(--b))":   "calc(var(--a) + var(--b))",
		"clamp(1rem,2vw+1rem,3rem)": "clamp(1rem,2vw + 1rem,3rem)",
		"calc((1px+2px)*3)":         "calc((1px + 2px) * 3)",
		"env(safe-area-inset-top)":  "env(safe-area-inset-top)",
	}
	for in, want := range cases {
		if got := DecodeArbitraryValue(in); got != want {
			t.Errorf("decode(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestCompareAndPredicates(t *testing.T) {
	if Compare("a", "b") >= 0 || Compare("p-2", "p-10") >= 0 || Compare("p-10", "p-2") <= 0 ||
		Compare("p-2", "p-2") != 0 || Compare("p-02", "p-2") >= 0 || Compare("a", "ab") >= 0 || Compare("ab", "a") <= 0 {
		t.Fatal("compare")
	}
	list := []string{"p-10", "p-2", "p-1", "m-1"}
	sort.Slice(list, func(i, j int) bool { return Compare(list[i], list[j]) < 0 })
	assertEqual(t, list, []string{"m-1", "p-1", "p-2", "p-10"})

	assertEqual(t, IsPositiveInteger("0"), true)
	assertEqual(t, IsPositiveInteger("012"), false)
	assertEqual(t, IsPositiveInteger("-1"), false)
	assertEqual(t, IsStrictPositiveInteger("0"), false)
	assertEqual(t, IsStrictPositiveInteger("1"), true)
	for in, want := range map[string]bool{"4": true, "0.25": true, "1.5": true, "-2.75": true, "0.3": false, "0.50": false, "04": false, ".5": false} {
		assertEqual(t, IsMultipleOfQuarter(in), want)
	}
}

func TestExpandBraces(t *testing.T) {
	cases := map[string][]string{
		"p-{1,2,3}":     {"p-1", "p-2", "p-3"},
		"p-{1..3}":      {"p-1", "p-2", "p-3"},
		"p-{3..1}":      {"p-3", "p-2", "p-1"},
		"p-{0..20..5}":  {"p-0", "p-5", "p-10", "p-15", "p-20"},
		"m-{-2..0}":     {"m--2", "m--1", "m-0"},
		"{a,b}-{1,2}":   {"a-1", "a-2", "b-1", "b-2"},
		"{hover:,}flex": {"hover:flex", "flex"},
		"{a,{b,c}}":     {"a", "b", "c"},
		"plain":         {"plain"},
	}
	for in, want := range cases {
		got, err := ExpandBraces(in)
		if err != nil {
			t.Fatal(err)
		}
		assertEqual(t, got, want)
	}
	for _, in := range []string{"p-{0..4..0}", "p-{1,2", "p-1}"} {
		if _, err := ExpandBraces(in); err == nil {
			t.Errorf("expected error for %q", in)
		}
	}
	assertEqual(t, IsValidVariantName("dark"), true)
	assertEqual(t, IsValidVariantName("@md"), true)
	assertEqual(t, IsValidVariantName("Foo"), false)
	assertEqual(t, IsValidVariantName("foo-"), false)
	assertEqual(t, IsValidVariantName("foo_"), false)
}

func TestValueParser(t *testing.T) {
	assertEqual(t, ParseValue("1px solid red"), []ValueNode{Word("1px"), Sep(" "), Word("solid"), Sep(" "), Word("red")})
	assertEqual(t, ParseValue("calc(1px + var(--x, 2px))"), []ValueNode{
		Fn("calc", Word("1px"), Sep(" "), Word("+"), Sep(" "), Fn("var", Word("--x"), Sep(","), Sep(" "), Word("2px"))),
	})
	assertEqual(t, ParseValue("a/b,c"), []ValueNode{Word("a"), Sep("/"), Word("b"), Sep(","), Word("c")})
	assertEqual(t, ParseValue(`"a, b" c`), []ValueNode{Word(`"a, b"`), Sep(" "), Word("c")})
	assertEqual(t, ParseValue("(a)"), []ValueNode{Fn("", Word("a"))})
	for _, in := range []string{"1px solid red", "calc(1px + var(--x, 2px))", "url(/a.png)", "a/b,c", `"a, b" c`, "--spacing(4)"} {
		assertEqual(t, ValueToCSS(ParseValue(in)), in)
	}
	ast := ParseValue("calc(--spacing(4) * 2)")
	WalkValue(&ast, func(n ValueNode, _ *ValueFunction) ([]ValueNode, bool, bool) {
		if f, ok := n.(*ValueFunction); ok && f.Name == "--spacing" {
			return []ValueNode{Word("calc(var(--spacing) * 4)")}, true, false
		}
		return nil, false, false
	})
	assertEqual(t, ValueToCSS(ast), "calc(calc(var(--spacing) * 4) * 2)")
}

func TestWithAlpha(t *testing.T) {
	assertEqual(t, WithAlpha("red", "0.5"), "color-mix(in oklab, red 50%, transparent)")
	assertEqual(t, WithAlpha("red", "50%"), "color-mix(in oklab, red 50%, transparent)")
	assertEqual(t, WithAlpha("red", "1"), "red")
	assertEqual(t, WithAlpha("red", "100%"), "red")
	assertEqual(t, WithAlpha("red", "var(--a)"), "color-mix(in oklab, red var(--a), transparent)")
	assertEqual(t, WithAlpha("red", ".25"), "color-mix(in oklab, red 25%, transparent)")
}
