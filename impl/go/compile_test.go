package twill

import (
	"strings"
	"testing"
)

// run compiles and builds, returning the CSS.
func run(t *testing.T, css string, candidates []string, files map[string]string) string {
	t.Helper()
	compiler, err := Compile(css, CompileOptions{Base: "/root", LoadStylesheet: memoryLoader(files)})
	if err != nil {
		t.Fatalf("compile failed: %v", err)
	}
	return compiler.Build(candidates)
}

// runErr compiles and returns the compile error.
func runErr(t *testing.T, css string) error {
	t.Helper()
	_, err := Compile(css, CompileOptions{Base: "/root", LoadStylesheet: memoryLoader(nil)})
	return err
}

// expectErr asserts that compiling css fails with a message containing want.
func expectErr(t *testing.T, css string, want string) {
	t.Helper()
	err := runErr(t, css)
	if err == nil {
		t.Fatalf("expected an error containing %q for %q, got none", want, css)
	}
	if _, ok := err.(*Error); !ok {
		t.Fatalf("expected *Error, got %T: %v", err, err)
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("expected error containing %q, got %q", want, err.Error())
	}
}

// runUtilities is like run but drops the leading `:root, :host { ... }` block.
func runUtilities(t *testing.T, css string, candidates []string) string {
	t.Helper()
	output := run(t, css, candidates, nil)
	if strings.HasPrefix(output, ":root, :host {\n") {
		end := strings.Index(output, "\n}\n")
		if end != -1 {
			output = output[end+3:]
		}
	}
	return output
}

func includes(t *testing.T, output, want string, labels ...string) {
	t.Helper()
	if !strings.Contains(output, want) {
		label := ""
		if len(labels) > 0 {
			label = labels[0] + ": "
		}
		t.Fatalf("%sexpected output to contain:\n%s\n--- output ---\n%s", label, want, output)
	}
}

func excludes(t *testing.T, output, unwanted string) {
	t.Helper()
	if strings.Contains(output, unwanted) {
		t.Fatalf("expected output not to contain %q\n--- output ---\n%s", unwanted, output)
	}
}

const themeCSS = "@import \"twill/theme.css\";\n@twill utilities;\n"

func TestSpec161BasicDocument(t *testing.T) {
	output := run(t, `@theme {
  --color-black: #000;
  --breakpoint-md: 768px;
}
@layer utilities {
  @twill utilities;
}`, []string{"dark:bg-black", "hover:underline", "md:grid", "flex"}, nil)
	assertEqual(t, output, `:root, :host {
  --color-black: #000;
}
@layer utilities {
  .flex {
    display: flex;
  }
  @media (hover: hover) {
    .hover\:underline:hover {
      text-decoration-line: underline;
    }
  }
  @media (width >= 768px) {
    .md\:grid {
      display: grid;
    }
  }
  @media (prefers-color-scheme: dark) {
    .dark\:bg-black {
      background-color: var(--color-black);
    }
  }
}
`)
}

func escapeCandidate(c string) string {
	var b strings.Builder
	for i := 0; i < len(c); i++ {
		ch := c[i]
		if ch >= 'a' && ch <= 'z' || ch >= 'A' && ch <= 'Z' || ch >= '0' && ch <= '9' || ch == '_' || ch == '-' {
			b.WriteByte(ch)
		} else {
			b.WriteByte('\\')
			b.WriteByte(ch)
		}
	}
	return b.String()
}

func TestSpec162ValueForms(t *testing.T) {
	cases := [][2]string{
		{"p-4", "padding: calc(var(--spacing) * 4);"},
		{"p-1", "padding: var(--spacing);"},
		{"p-0", "padding: 0px;"},
		{"p-px", "padding: 1px;"},
		{"-mt-2", "margin-top: calc(var(--spacing) * -2);"},
		{"w-1/2", "width: calc(1 / 2 * 100%);"},
		{"w-[13px]", "width: 13px;"},
		{"w-(--my-w)", "width: var(--my-w);"},
		{"max-w-md", "max-width: var(--container-md);"},
		{"bg-red-500", "background-color: var(--color-red-500);"},
		{"bg-red-500/50", "background-color: color-mix(in oklab, var(--color-red-500) 50%, transparent);"},
		{"bg-[#0088cc]", "background-color: #0088cc;"},
		{"bg-[url(/a_b.png)]", "background-image: url(/a_b.png);"},
		{"bg-[length:10px_20px]", "background-size: 10px 20px;"},
		{"text-lg", "font-size: var(--text-lg);\n  line-height: var(--tw-leading, var(--text-lg--line-height));"},
		{"text-lg/8", "font-size: var(--text-lg);\n  line-height: calc(var(--spacing) * 8);"},
		{"text-red-500", "color: var(--color-red-500);"},
		{"font-bold", "--tw-font-weight: var(--font-weight-bold);\n  font-weight: var(--font-weight-bold);"},
		{"rounded-lg", "border-radius: var(--radius-lg);"},
		{"rounded-full", "border-radius: calc(infinity * 1px);"},
		{"border", "border-style: var(--tw-border-style);\n  border-width: 1px;"},
		{"border-2", "border-style: var(--tw-border-style);\n  border-width: 2px;"},
		{"z-10", "z-index: 10;"},
		{"-z-10", "z-index: calc(10 * -1);"},
		{"flex-1", "flex: 1;"},
		{"opacity-50", "opacity: 50%;"},
		{"[mask-type:luminance]", "mask-type: luminance;"},
		{"[--my-var:1px]", "--my-var: 1px;"},
		{"underline!", "text-decoration-line: underline !important;"},
	}
	for _, c := range cases {
		output := runUtilities(t, themeCSS, []string{c[0]})
		includes(t, output, "."+escapeCandidate(c[0])+" {\n  "+c[1]+"\n}\n", c[0])
	}

	fontBold := runUtilities(t, themeCSS, []string{"font-bold"})
	includes(t, fontBold, "@property --tw-font-weight {\n  syntax: \"*\";\n  inherits: false;\n}\n")
	border := runUtilities(t, themeCSS, []string{"border"})
	includes(t, border, "@property --tw-border-style {\n  syntax: \"*\";\n  inherits: false;\n  initial-value: solid;\n}\n")
}

func TestSpec163VariantForms(t *testing.T) {
	cases := [][2]string{
		{"hover:flex", "@media (hover: hover) {\n  .hover\\:flex:hover {\n    display: flex;\n  }\n}\n"},
		{"focus:flex", ".focus\\:flex:focus {\n  display: flex;\n}\n"},
		{"sm:flex", "@media (width >= 40rem) {\n  .sm\\:flex {\n    display: flex;\n  }\n}\n"},
		{"max-md:flex", "@media (width < 48rem) {\n  .max-md\\:flex {\n    display: flex;\n  }\n}\n"},
		{"min-[600px]:flex", "@media (width >= 600px) {\n  .min-\\[600px\\]\\:flex {\n    display: flex;\n  }\n}\n"},
		{"@md:flex", "@container (width >= 28rem) {\n  .\\@md\\:flex {\n    display: flex;\n  }\n}\n"},
		{"@md/main:flex", "@container main (width >= 28rem) {\n  .\\@md\\/main\\:flex {\n    display: flex;\n  }\n}\n"},
		{"group-hover:flex", "@media (hover: hover) {\n  .group-hover\\:flex:is(:where(.group):hover *) {\n    display: flex;\n  }\n}\n"},
		{"group-hover/item:flex", "@media (hover: hover) {\n  .group-hover\\/item\\:flex:is(:where(.group\\/item):hover *) {\n    display: flex;\n  }\n}\n"},
		{"peer-checked:flex", ".peer-checked\\:flex:is(:where(.peer):checked ~ *) {\n  display: flex;\n}\n"},
		{"has-[>img]:flex", ".has-\\[\\>img\\]\\:flex:has(> img) {\n  display: flex;\n}\n"},
		{"in-data-visible:flex", ":where(*[data-visible]) .in-data-visible\\:flex {\n  display: flex;\n}\n"},
		{"not-hover:flex", ".not-hover\\:flex:not(:hover) {\n  display: flex;\n}\n@media not all and (hover: hover) {\n  .not-hover\\:flex {\n    display: flex;\n  }\n}\n"},
		{"not-supports-grid:flex", "@supports not (grid: var(--tw)) {\n  .not-supports-grid\\:flex {\n    display: flex;\n  }\n}\n"},
		{"data-[state=open]:flex", ".data-\\[state\\=open\\]\\:flex[data-state=\"open\"] {\n  display: flex;\n}\n"},
		{"aria-checked:flex", ".aria-checked\\:flex[aria-checked=\"true\"] {\n  display: flex;\n}\n"},
		{"nth-3:flex", ".nth-3\\:flex:nth-child(3) {\n  display: flex;\n}\n"},
		{"[&_p]:flex", ".\\[\\&_p\\]\\:flex p {\n  display: flex;\n}\n"},
		{"[@media(width>=100px)]:flex", "@media (width>=100px) {\n  .\\[\\@media\\(width\\>\\=100px\\)\\]\\:flex {\n    display: flex;\n  }\n}\n"},
		{"*:flex", ":is(.\\*\\:flex > *) {\n  display: flex;\n}\n"},
		{"before:block", ".before\\:block::before {\n  content: var(--tw-content);\n  display: block;\n}\n"},
		{"dark:hover:flex", "@media (prefers-color-scheme: dark) {\n  @media (hover: hover) {\n    .dark\\:hover\\:flex:hover {\n      display: flex;\n    }\n  }\n}\n"},
	}
	for _, c := range cases {
		output := runUtilities(t, themeCSS, []string{c[0]})
		includes(t, output, c[1], c[0])
	}
}

func TestSpec164CustomUtilitiesAndApply(t *testing.T) {
	output := runUtilities(t, `@import "twill/theme.css";
@utility tab-* {
  tab-size: --value(integer);
  tab-size: --value(--tab-size-*);
  tab-size: --value([integer]);
}
@utility content-auto {
  content-visibility: auto;
}
.btn {
  @apply rounded-lg px-4 py-2 hover:bg-red-500;
}
@twill utilities;`, []string{"tab-4", "tab-[8]", "content-auto"})
	// tab-size has a position in the global property order while
	// content-visibility does not, so tab-* sorts first (SPEC §11.3).
	assertEqual(t, output, `.btn {
  border-radius: var(--radius-lg);
  padding-inline: calc(var(--spacing) * 4);
  padding-block: calc(var(--spacing) * 2);
}
@media (hover: hover) {
  .btn:hover {
    background-color: var(--color-red-500);
  }
}
.tab-4 {
  tab-size: 4;
}
.tab-\[8\] {
  tab-size: 8;
}
.content-auto {
  content-visibility: auto;
}
`)
}

func TestSpec165ThemeCustomization(t *testing.T) {
	css := `@import "twill";
@theme {
  --color-*: initial;
  --color-primary: oklch(0.6 0.2 250);
  --breakpoint-3xl: 120rem;
  --font-display: "Inter", sans-serif;
}
@custom-variant dark (&:where(.dark, .dark *));`
	output := run(t, css, []string{"bg-primary", "3xl:flex", "font-display", "bg-red-500", "dark:flex"}, nil)
	includes(t, output, ".bg-primary {\n    background-color: var(--color-primary);\n  }")
	includes(t, output, "@media (width >= 120rem) {\n    .\\33 xl\\:flex {\n      display: flex;\n    }\n  }")
	includes(t, output, ".font-display {\n    font-family: var(--font-display);\n  }")
	includes(t, output, ".dark\\:flex:where(.dark, .dark *) {\n    display: flex;\n  }")
	excludes(t, output, "bg-red-500")
	excludes(t, output, "--color-red-500")
	includes(t, output, "--color-primary: oklch(0.6 0.2 250);")
}

func TestBuildOrderIndependentAndStable(t *testing.T) {
	a := run(t, themeCSS, []string{"p-4", "hover:flex", "sm:p-2", "flex", "m-1"}, nil)
	b := run(t, themeCSS, []string{"m-1", "flex", "sm:p-2", "hover:flex", "p-4"}, nil)
	assertEqual(t, a, b)

	compiler, err := Compile(themeCSS, CompileOptions{})
	if err != nil {
		t.Fatal(err)
	}
	first := compiler.Build([]string{"flex", "p-4"})
	second := compiler.Build([]string{"p-4"})
	assertEqual(t, second, first)
	third := compiler.Build([]string{"flex", "nope"})
	assertEqual(t, third, first)
	fourth := compiler.Build([]string{"m-2"})
	if fourth == first {
		t.Fatal("expected a new candidate to change the output")
	}
	includes(t, fourth, ".m-2")
}

func TestBuildCandidatesAccumulate(t *testing.T) {
	compiler, err := Compile(themeCSS, CompileOptions{})
	if err != nil {
		t.Fatal(err)
	}
	compiler.Build([]string{"flex"})
	output := compiler.Build([]string{"hidden"})
	includes(t, output, ".flex {")
	includes(t, output, ".hidden {")
}

func TestBuildFeatures(t *testing.T) {
	compiler, err := Compile(".a { color: red; }", CompileOptions{})
	if err != nil {
		t.Fatal(err)
	}
	assertEqual(t, compiler.Features, Features(0))
	assertEqual(t, compiler.Build([]string{"flex"}), ".a { color: red; }")

	themed, err := Compile("@theme { --a: 1; } .a { color: var(--a); }", CompileOptions{})
	if err != nil {
		t.Fatal(err)
	}
	assertEqual(t, themed.Features&FeatureAtTheme, FeatureAtTheme)
	assertEqual(t, themed.Build([]string{"flex"}), ":root, :host {\n  --a: 1;\n}\n.a {\n  color: var(--a);\n}\n")

	full, err := Compile(`@import "twill"; .a { @apply flex; } .b { @variant hover { color: red; } }`, CompileOptions{})
	if err != nil {
		t.Fatal(err)
	}
	assertEqual(t, full.Features&FeatureAtImport, FeatureAtImport)
	assertEqual(t, full.Features&FeatureAtApply, FeatureAtApply)
	assertEqual(t, full.Features&FeatureVariants, FeatureVariants)
	assertEqual(t, full.Features&FeatureUtilities, FeatureUtilities)
	assertEqual(t, full.Features&FeatureThemeFunction, FeatureThemeFunction)
}

func TestImportantFlagAndMarker(t *testing.T) {
	output := run(t, "@import \"twill/theme.css\" important;\n@twill utilities;", []string{"flex", "font-bold"}, nil)
	includes(t, output, "display: flex !important;")
	includes(t, output, "font-weight: var(--font-weight-bold) !important;")
	includes(t, output, "inherits: false;\n")
	excludes(t, output, "inherits: false !important")

	applied := run(t, "@import \"twill/theme.css\" important;\n.a { @apply flex underline!; }\n@twill utilities;", nil, nil)
	includes(t, applied, ".a {\n  display: flex;\n  text-decoration-line: underline !important;\n}")
}

func TestThemeFunctions(t *testing.T) {
	output := run(t, `@import "twill/theme.css";
.a {
  padding: --spacing(4);
  margin: --spacing(0) --spacing(1);
  color: --alpha(var(--color-red-500) / 50%);
  width: --theme(--container-md);
  height: --theme(--container-md inline);
  gap: --theme(--nope, 1px, 2px);
  font: theme(--text-lg);
  border-color: --theme(--color-red-500/0.5);
}
@media (width >= --theme(--breakpoint-md)) {
  .b { color: red; }
}`, nil, nil)
	includes(t, output, "padding: calc(var(--spacing) * 4);")
	includes(t, output, "margin: 0px var(--spacing);")
	includes(t, output, "color: color-mix(in oklab, var(--color-red-500) 50%, transparent);")
	includes(t, output, "width: var(--container-md);")
	includes(t, output, "height: 28rem;")
	includes(t, output, "gap: 1px, 2px;")
	includes(t, output, "font: 1.125rem;")
	includes(t, output, "border-color: color-mix(in oklab, var(--color-red-500) 50%, transparent);")
	includes(t, output, "@media (width >= 48rem) {")
	includes(t, output, "--container-md: 28rem;")
}

func TestThemeFunctionsFallbackAndErrors(t *testing.T) {
	output := run(t, `@theme { --a: var(--b); --c: initial; --d: 1px; }
.x { color: --theme(--a, red); background: --theme(--a inline, red); width: --theme(--d, initial); height: --theme(--c, 2px); }`, nil, nil)
	includes(t, output, "color: var(--a, red);")
	includes(t, output, "background: var(--b, red);")
	includes(t, output, "width: var(--d);")
	includes(t, output, "height: 2px;")

	expectErr(t, ".a { padding: --spacing(); }", "--spacing")
	expectErr(t, ".a { padding: --spacing(1, 2); }", "--spacing")
	expectErr(t, ".a { padding: --spacing(4); }", "--spacing")
	expectErr(t, ".a { color: --alpha(red); }", "--alpha")
	expectErr(t, ".a { color: --theme(spacing); }", "--theme")
	expectErr(t, ".a { color: --theme(--nope); }", "resolve")
	expectErr(t, ".a { color: theme(--nope); }", "resolve")
}

func TestThemeFunctionFailureInvalidatesCandidate(t *testing.T) {
	output := run(t, "@theme { --color-a: red; }\n@twill utilities;", []string{"p-4", "bg-a"}, nil)
	excludes(t, output, "p-4")
	includes(t, output, ".bg-a {")
}

func TestApplyVariantsErrorsMixins(t *testing.T) {
	output := run(t, `@import "twill/theme.css";
.a { @apply flex hover:underline sm:p-2; }
.b { @apply --my-mixin; }
@apply flex;`, nil, nil)
	includes(t, output, `.a {
  display: flex;
}
@media (hover: hover) {
  .a:hover {
    text-decoration-line: underline;
  }
}
@media (width >= 40rem) {
  .a {
    padding: calc(var(--spacing) * 2);
  }
}
.b {
  @apply --my-mixin;
}
@apply flex;
`)

	expectErr(t, ".a { @apply --x flex; }", "mix")
	expectErr(t, ".a { @apply flex { color: red; } }", "body")
	expectErr(t, "@keyframes x { to { @apply flex; } }", "@keyframes")
	expectErr(t, ".a { @apply nope; }", "empty")
	expectErr(t, `@import "twill/theme.css"; .a { @apply nope; }`, "unknown utility")
	expectErr(t, `@import "twill/theme.css"; .a { @apply nope:flex; }`, "unknown variant")
	expectErr(t, `@import "twill/theme.css" prefix(tw); .a { @apply flex; }`, "prefix")
	expectErr(t, `@import "twill/theme.css"; @source not inline("flex"); .a { @apply flex; }`, "disabled")
}

func TestApplyCustomUtilitiesOrderAndCycles(t *testing.T) {
	output := run(t, `@import "twill/theme.css";
@utility foo { @apply bar p-1; }
@utility bar { color: red; }
.x { @apply foo; }
@twill utilities;`, []string{"foo"}, nil)
	includes(t, output, ".x {\n  padding: var(--spacing);\n  color: red;\n}")
	includes(t, output, ".foo {\n  padding: var(--spacing);\n  color: red;\n}")

	expectErr(t, "@utility foo { @apply bar; }\n@utility bar { @apply foo; }", "circular")
	expectErr(t, "@utility foo { @apply foo; }", "circular")
}

func TestUtilityStaticAndFunctionalForms(t *testing.T) {
	output := run(t, `@import "twill/theme.css";
@theme { --tab-size-github: 8; --leading-tight: 1.25; }
@utility tab-* {
  tab-size: --value(integer);
  tab-size: --value(--tab-size-*);
  tab-size: --value([integer]);
}
@utility aspect-* {
  aspect-ratio: --value(ratio, --aspect-*, [ratio]);
  width: --value(number);
}
@utility opacity-* {
  opacity: --value(percentage);
  opacity: --modifier(number);
}
@utility text-* {
  font-size: --value(--text-*);
  line-height: --value(--text-*--line-height);
  line-height: --modifier(--leading-*);
}
@utility lit-* {
  color: --value("red", 'blue');
}
@utility any-* {
  color: --value([*]);
}
@utility content-auto { content-visibility: auto; }
@utility foo-1\/2 { color: red; }
@twill utilities;`, []string{
		"tab-4", "tab-github", "tab-[8]", "tab-[foo]", "tab-x",
		"aspect-16/9", "aspect-video", "aspect-[4/3]", "aspect-3", "aspect-16/9/2",
		"opacity-50%", "opacity-50%/2", "opacity-50%/foo",
		"text-lg", "text-lg/tight",
		"lit-red", "lit-blue", "lit-green",
		"any-[foo]", "content-auto", "foo-1/2",
	}, nil)
	includes(t, output, ".tab-4 {\n  tab-size: 4;\n}")
	includes(t, output, ".tab-github {\n  tab-size: var(--tab-size-github);\n}")
	includes(t, output, ".tab-\\[8\\] {\n  tab-size: 8;\n}")
	excludes(t, output, "tab-\\[foo\\]")
	excludes(t, output, "tab-x")
	includes(t, output, ".aspect-16\\/9 {\n  aspect-ratio: 16 / 9;\n}")
	includes(t, output, ".aspect-video {\n  aspect-ratio: var(--aspect-video);\n}")
	includes(t, output, ".aspect-\\[4\\/3\\] {\n  aspect-ratio: 4/3;\n}")
	includes(t, output, ".aspect-3 {\n  width: 3;\n}")
	excludes(t, output, "aspect-16\\/9\\/2")
	includes(t, output, ".opacity-50\\% {\n  opacity: 50%;\n}")
	includes(t, output, ".opacity-50\\%\\/2 {\n  opacity: 50%;\n  opacity: 2;\n}")
	excludes(t, output, "opacity-50\\%\\/foo")
	includes(t, output, ".text-lg {\n  font-size: var(--text-lg);\n  line-height: var(--text-lg--line-height);\n}")
	includes(t, output, ".text-lg\\/tight {\n  font-size: var(--text-lg);\n  line-height: var(--text-lg--line-height);\n  line-height: var(--leading-tight);\n}")
	includes(t, output, ".lit-red {\n  color: red;\n}")
	includes(t, output, ".lit-blue {\n  color: blue;\n}")
	excludes(t, output, "lit-green")
	includes(t, output, ".any-\\[foo\\] {\n  color: foo;\n}")
	includes(t, output, ".content-auto {\n  content-visibility: auto;\n}")
	includes(t, output, ".foo-1\\/2 {\n  color: red;\n}")
}

func TestUtilityErrors(t *testing.T) {
	expectErr(t, "@utility foo {}", "empty")
	expectErr(t, "@utility foo* { color: red; }", "-*")
	expectErr(t, "@utility fo*o { color: red; }", "end")
	expectErr(t, "@utility Foo { color: red; }", "invalid utility name")
	expectErr(t, "@utility foo- { color: red; }", "invalid utility name")
	expectErr(t, ".a { @utility foo { color: red; } }", "nested")
}

func TestCustomVariantSelectorForm(t *testing.T) {
	output := run(t, `@import "twill/theme.css";
@custom-variant dark (&:where(.dark, .dark *));
@custom-variant hocus (&:hover, &:focus);
@custom-variant wide (@media (width >= 100px), @supports (display: grid));
@custom-variant mixed (&:hover, @media print);
@twill utilities;`, []string{"dark:flex", "hocus:flex", "wide:flex", "mixed:flex", "not-hocus:flex"}, nil)
	includes(t, output, ".dark\\:flex:where(.dark, .dark *) {\n  display: flex;\n}")
	includes(t, output, ".hocus\\:flex:hover, .hocus\\:flex:focus {\n  display: flex;\n}")
	includes(t, output, "@media (width >= 100px) {\n  .wide\\:flex {\n    display: flex;\n  }\n}\n@supports (display: grid) {\n  .wide\\:flex {\n    display: flex;\n  }\n}")
	includes(t, output, ".mixed\\:flex:hover {\n  display: flex;\n}\n@media print {\n  .mixed\\:flex {\n    display: flex;\n  }\n}")
	includes(t, output, ".not-hocus\\:flex:not(:hover), .not-hocus\\:flex:not(:focus) {")

	expectErr(t, "@custom-variant foo (&:hover) { color: red; }", "both")
	expectErr(t, "@custom-variant foo;", "no selector")
	expectErr(t, "@custom-variant Foo (&:hover);", "invalid variant name")
	expectErr(t, "@custom-variant foo (&:hover,);", "empty")
	expectErr(t, ".a { @custom-variant foo (&:hover); }", "nested")
}

func TestCustomVariantBodyFormDependenciesCycles(t *testing.T) {
	output := run(t, `@import "twill/theme.css";
@custom-variant theme-dark {
  &:where([data-theme="dark"], [data-theme="dark"] *) {
    @slot;
  }
}
@custom-variant dark-hover {
  @variant theme-dark {
    &:hover {
      @slot;
    }
  }
}
@custom-variant animated {
  @keyframes spin-custom { to { transform: rotate(1turn); } }
  animation: spin-custom 1s;
  @slot;
}
@twill utilities;`, []string{"theme-dark:flex", "dark-hover:flex", "animated:flex"}, nil)
	includes(t, output, ".theme-dark\\:flex:where([data-theme=\"dark\"], [data-theme=\"dark\"] *) {\n  display: flex;\n}")
	includes(t, output, ".dark-hover\\:flex:where([data-theme=\"dark\"], [data-theme=\"dark\"] *):hover {\n  display: flex;\n}")
	includes(t, output, ".animated\\:flex {\n  animation: spin-custom 1s;\n  display: flex;\n}")
	includes(t, output, "@keyframes spin-custom {\n  to {\n    transform: rotate(1turn);\n  }\n}")

	expectErr(t, "@custom-variant a { @variant b { @slot; } }\n@custom-variant b { @variant a { @slot; } }", "circular")
}

func TestVariantCompatibilityForms(t *testing.T) {
	output := run(t, `@import "twill/theme.css";
@variant hocus (&:hover, &:focus);
@variant dark { &:where(.dark, .dark *) { @slot; } }
@twill utilities;`, []string{"hocus:flex", "dark:flex"}, nil)
	includes(t, output, ".hocus\\:flex:hover, .hocus\\:flex:focus {")
	includes(t, output, ".dark\\:flex:where(.dark, .dark *) {")
	excludes(t, output, "@variant")
}

func TestNestedVariant(t *testing.T) {
	output := run(t, `@import "twill/theme.css";
.a {
  color: red;
  @variant hover { color: blue; }
  @variant dark:hover, sm { color: green; }
}`, nil, nil)
	assertEqual(t, output, `.a {
  color: red;
}
@media (hover: hover) {
  .a:hover {
    color: blue;
  }
}
@media (prefers-color-scheme: dark) {
  @media (hover: hover) {
    .a:hover {
      color: green;
    }
  }
}
@media (width >= 40rem) {
  .a {
    color: green;
  }
}
`)
	expectErr(t, ".a { @variant nope { color: red; } }", "unknown variant")
	expectErr(t, `@import "twill/theme.css"; .a { @variant hover: { color: red; } }`, "empty")
}

func TestOptimizerRegistrationsOnceAndHoisted(t *testing.T) {
	output := run(t, themeCSS, []string{"font-bold", "font-thin", "leading-6"}, nil)
	assertEqual(t, strings.Count(output, "@property --tw-font-weight"), 1)
	registration := strings.Index(output, "@property")
	lastRule := strings.LastIndex(output, ".leading-6")
	if registration <= lastRule {
		t.Fatalf("expected registrations after rules:\n%s", output)
	}
	if !strings.HasSuffix(strings.TrimRight(output, "\n "), "}") {
		t.Fatalf("expected output to end with a block:\n%s", output)
	}
}

func TestOptimizerPrunesThemeAndKeyframes(t *testing.T) {
	css := `@import "twill/theme.css";
@theme { --color-keep: red; --keep-static: 1px; --chain-a: var(--chain-b); --chain-b: 2px; }
@theme static { --always: 3px; }
@twill utilities;`
	output := run(t, css, []string{"bg-keep", "[width:var(--chain-a)]"}, nil)
	includes(t, output, "--color-keep: red;")
	includes(t, output, "--chain-a: var(--chain-b);")
	includes(t, output, "--chain-b: 2px;")
	includes(t, output, "--always: 3px;")
	excludes(t, output, "--keep-static")
	excludes(t, output, "--color-red-500")
	excludes(t, output, "@keyframes")

	spin := run(t, css, []string{"animate-spin"}, nil)
	includes(t, spin, "@keyframes spin {")
	excludes(t, spin, "@keyframes ping")
	includes(t, spin, "--animate-spin: spin 1s linear infinite;")

	marked := run(t, css, []string{"--color-blue-500"}, nil)
	includes(t, marked, "--color-blue-500:")

	empty := run(t, "@layer theme, base;\n@import \"twill/theme.css\" layer(theme);\n@twill utilities;", nil, nil)
	assertEqual(t, empty, "@layer theme, base;\n")
}

func TestOptimizerFlattensNesting(t *testing.T) {
	output := run(t, `@import "twill/theme.css";
.a, .b {
  color: red;
  &:hover { color: blue; }
  .c & { color: green; }
  > .d { color: purple; }
  @media (x) { color: orange; .e { color: black; } }
  @supports (y) { @media (z) { & .f { color: white; } } }
}
@keyframes k { 50% { opacity: 0; } }`, nil, nil)
	assertEqual(t, output, `.a, .b {
  color: red;
}
:is(.a, .b):hover {
  color: blue;
}
.c :is(.a, .b) {
  color: green;
}
:is(.a, .b) > .d {
  color: purple;
}
@media (x) {
  .a, .b {
    color: orange;
  }
  :is(.a, .b).e {
    color: black;
  }
}
@supports (y) {
  @media (z) {
    :is(.a, .b) .f {
      color: white;
    }
  }
}
@keyframes k {
  50% {
    opacity: 0;
  }
}
`)
}

func TestReferenceImportsProduceNoOutput(t *testing.T) {
	output := run(t, "@reference \"twill\";\n.a { @apply flex p-4; }", nil, nil)
	assertEqual(t, output, ".a {\n  display: flex;\n  padding: calc(var(--spacing, 0.25rem) * 4);\n}\n")
}

func TestPrefix(t *testing.T) {
	output := run(t, "@import \"twill/theme.css\" prefix(tw);\n@twill utilities;", []string{"tw:flex", "tw:bg-red-500", "tw:group-hover:flex", "flex"}, nil)
	includes(t, output, "--tw-color-red-500: oklch(63.7% 0.237 25.331);")
	includes(t, output, ".tw\\:flex {\n  display: flex;\n}")
	includes(t, output, ".tw\\:bg-red-500 {\n  background-color: var(--tw-color-red-500);\n}")
	includes(t, output, ".tw\\:group-hover\\:flex:is(:where(.tw\\:group):hover *)")
	excludes(t, output, "\n.flex")
}

func TestSourceInlineAndNotInline(t *testing.T) {
	output := run(t, `@import "twill/theme.css";
@source inline("p-{1,2}");
@source not inline("flex");
@twill utilities;`, []string{"flex", "hidden"}, nil)
	includes(t, output, ".p-1 {")
	includes(t, output, ".p-2 {")
	includes(t, output, ".hidden {")
	excludes(t, output, ".flex")
}

func TestLicenseCommentsAndExternalImportsSurvive(t *testing.T) {
	output := run(t, "/*! license */\n@import url(x.css);\n@import \"https://x/y.css\";\n@theme { --a: 1; }\n.a { color: var(--a); }", nil, nil)
	assertEqual(t, output, "/*! license */\n@import url(x.css);\n@import \"https://x/y.css\";\n:root, :host {\n  --a: 1;\n}\n.a {\n  color: var(--a);\n}\n")
}

func TestUserFilesThroughLoader(t *testing.T) {
	output := run(t, "@import \"./theme.css\";\n@import \"twill/theme.css\";\n@twill utilities;", []string{"bg-brand"}, map[string]string{
		"/root/theme.css": "@theme { --color-brand: blue; }",
	})
	includes(t, output, ".bg-brand {\n  background-color: var(--color-brand);\n}")
}
