package twill

import (
	"strings"
	"testing"
)

// memoryLoader serves the built-in stylesheets and an in-memory file map.
func memoryLoader(files map[string]string) StylesheetLoader {
	return func(id, base string) (*LoadedStylesheet, error) {
		builtin, err := ResolveBuiltin(id, base)
		if err != nil || builtin != nil {
			return builtin, err
		}
		path := base + "/" + strings.TrimPrefix(id, "./")
		content, ok := files[path]
		if !ok {
			return nil, Errorf("Cannot find %s", path)
		}
		return &LoadedStylesheet{Path: path, Base: path[:strings.LastIndex(path, "/")], Content: content}, nil
	}
}

func importAndCollect(t *testing.T, css string, files map[string]string) (*Theme, []Node, *CollectedDirectives) {
	t.Helper()
	th := NewTheme()
	parsed := mustParse(t, css)
	ast := []Node{NewContext(ContextMap{"base": "/root"}, parsed...)}
	if _, err := SubstituteAtImports(&ast, "/root", memoryLoader(files)); err != nil {
		t.Fatal(err)
	}
	state, err := CollectDirectives(&ast, th)
	if err != nil {
		t.Fatal(err)
	}
	return th, ast, state
}

func collectErr(t *testing.T, css string) error {
	t.Helper()
	th := NewTheme()
	ast := []Node{NewContext(ContextMap{"base": "/root"}, mustParse(t, css)...)}
	if _, err := SubstituteAtImports(&ast, "/root", memoryLoader(nil)); err != nil {
		return err
	}
	_, err := CollectDirectives(&ast, th)
	return err
}

func TestImportParams(t *testing.T) {
	p, _ := ParseImportParams(`"a.css" layer(base) supports(display: grid) screen and (min-width: 1px)`)
	assertEqual(t, p, &ImportParams{URI: "a.css", Layer: strPtr("base"), Media: strPtr("screen and (min-width: 1px)"), Supports: strPtr("display: grid")})
	p, _ = ParseImportParams(`"twill" important theme(reference) prefix(tw)`)
	assertEqual(t, p, &ImportParams{URI: "twill", Media: strPtr("important theme(reference) prefix(tw)")})
	for _, in := range []string{`url(a.css)`, `"data:text/css,a"`, `"https://example.com/a.css"`, `a.css`} {
		p, err := ParseImportParams(in)
		if p != nil || err != nil {
			t.Errorf("%q: expected untouched", in)
		}
	}
	for _, in := range []string{`"a.css" supports(x) layer(base)`, `"a.css" screen layer(base)`} {
		if _, err := ParseImportParams(in); err == nil {
			t.Errorf("%q: expected error", in)
		}
	}
}

func TestImportWrapping(t *testing.T) {
	_, ast, _ := importAndCollect(t, `@import "./a.css" layer(base) supports(display: grid) print;`, map[string]string{"/root/a.css": ".a { color: red }"})
	assertEqual(t, Serialize(ast), "@supports (display: grid) {\n  @media print {\n    @layer base {\n      .a {\n        color: red;\n      }\n    }\n  }\n}\n")

	_, ast, _ = importAndCollect(t, `@import "./a.css";`, map[string]string{
		"/root/a.css": `@import "./sub/b.css";`, "/root/sub/b.css": `@import "./c.css";`, "/root/sub/c.css": ".c { color: red }",
	})
	assertEqual(t, Serialize(ast), ".c {\n  color: red;\n}\n")

	css := "@import url(a.css);\n@import \"https://example.com/a.css\";\n@import \"data:text/css,a\";\n"
	_, ast, _ = importAndCollect(t, css, nil)
	assertEqual(t, Serialize(ast), css)

	th := NewTheme()
	loop := []Node{NewContext(ContextMap{"base": "/root"}, mustParse(t, `@import "./a.css";`)...)}
	_, err := SubstituteAtImports(&loop, "/root", memoryLoader(map[string]string{"/root/a.css": `@import "./a.css";`}))
	if err == nil || !strings.Contains(err.Error(), "recursion") {
		t.Fatalf("expected recursion error, got %v", err)
	}
	_ = th
}

func TestBuiltinStylesheets(t *testing.T) {
	_, ast, state := importAndCollect(t, `@import "twill";`, nil)
	css := Serialize(ast)
	if !strings.HasPrefix(css, "@layer theme, base, components, utilities;\n@layer theme {\n") {
		t.Fatal(css[:80])
	}
	if !strings.Contains(css, "@layer utilities {\n  @twill utilities;\n}\n") || state.UtilitiesNode == nil {
		t.Fatal("utilities node missing")
	}
	l, err := ResolveBuiltin("./theme.css", "/root")
	if l != nil || err != nil {
		t.Fatal("user relative import resolved to a builtin")
	}
	l, _ = ResolveBuiltin("./theme.css", "twill:")
	assertEqual(t, l.Path, "twill:theme.css")
	l, _ = ResolveBuiltin("twill/preflight.css", "/root")
	assertEqual(t, l.Path, "twill:preflight.css")
	if _, err := ResolveBuiltin("./nope.css", "twill:"); err == nil {
		t.Fatal("expected error")
	}
	for _, name := range []string{"index.css", "theme.css", "preflight.css", "utilities.css"} {
		content, ok := BuiltinCSS(name)
		if !ok {
			t.Fatal(name)
		}
		if _, err := Parse(content); err != nil {
			t.Fatal(name, err)
		}
	}
}

func TestThemeDirective(t *testing.T) {
	th, ast, state := importAndCollect(t, `
    @theme {
      --color-black: #000;
      /* comment */
      --breakpoint-md: 768px;
      @keyframes spin { to { transform: rotate(360deg) } }
    }
    @theme { --color-white: #fff; }
  `, nil)
	assertEqual(t, th.Entry("--color-black").Value, "#000")
	assertEqual(t, th.Entry("--color-white").Value, "#fff")
	assertEqual(t, len(th.Keyframes()), 1)
	assertEqual(t, Serialize(ast), ":root, :host {\n}\n")
	assertEqual(t, state.FirstThemeRule.Selector, ":root, :host")
	assertEqual(t, state.Features&FeatureAtTheme != 0, true)

	th, _, _ = importAndCollect(t, `
    @theme reference { --a: 1; }
    @theme inline { --b: 2; }
    @theme default { --c: 3; }
    @theme static { --d: 4; }
    @theme prefix(tw) { --e: 5; }
    @theme { --container-1\.5: 1.5rem; }
  `, nil)
	assertEqual(t, th.Options("--a"), Reference)
	assertEqual(t, th.Options("--b"), Inline)
	assertEqual(t, th.Options("--c"), Default)
	assertEqual(t, th.Options("--d"), Static)
	assertEqual(t, th.Prefix, "tw")
	assertEqual(t, th.Has("--container-1.5"), true)

	_, ast, _ = importAndCollect(t, "@theme reference { --a: 1; }\n@theme { --b: 2; }", nil)
	assertEqual(t, Serialize(ast), ":root, :host {\n}\n")

	for css, msg := range map[string]string{
		`@theme prefix(Tw) {}`:             "prefix",
		`@theme { .foo { color: red; } }`:  ".foo {",
		`@theme { color: red; }`:           "@theme",
		`@source "./a" { color: red; }`:    "body",
		`.a { @source "./a"; }`:            "nested",
		`@media x { @source "./a"; }`:      "nested",
		`@source ./a;`:                     "quoted",
		`@source inline(p-4);`:             "quoted",
		`@twill utilities source(../app);`: "quoted",
	} {
		err := collectErr(t, css)
		if err == nil || !strings.Contains(err.Error(), msg) {
			t.Errorf("%q: expected %q, got %v", css, msg, err)
		}
	}
}

func TestSourceAndUtilitiesDirectives(t *testing.T) {
	_, ast, state := importAndCollect(t, `
    @source "./src/**/*.html";
    @source not '../vendor/**';
    @source inline("p-{1..3} {hover:,}flex");
    @source not inline("bg-red-{100,200}");
  `, nil)
	assertEqual(t, state.Sources, []SourceEntry{
		{Base: "/root", Pattern: "./src/**/*.html"}, {Base: "/root", Pattern: "../vendor/**", Negated: true},
	})
	assertEqual(t, state.InlineCandidates, []string{"p-1", "p-2", "p-3", "hover:flex", "flex"})
	assertEqual(t, state.IgnoredCandidates, []string{"bg-red-100", "bg-red-200"})
	assertEqual(t, Serialize(ast), "")

	_, ast, state = importAndCollect(t, "@twill utilities;\n@twill utilities;", nil)
	assertEqual(t, Serialize(ast), "@twill utilities;\n")
	assertEqual(t, state.Root == nil, true)
	assertEqual(t, state.Features&FeatureUtilities != 0, true)
	_, _, state = importAndCollect(t, `@twill utilities source(none);`, nil)
	assertEqual(t, state.Root, &SourceRoot{None: true})
	_, _, state = importAndCollect(t, `@twill utilities source("../app");`, nil)
	assertEqual(t, state.Root, &SourceRoot{Base: "/root", Pattern: "../app"})
	_, ast, state = importAndCollect(t, `@media reference { @twill utilities; }`, nil)
	assertEqual(t, state.UtilitiesNode == nil, true)
	assertEqual(t, Serialize(ast), "")
}

func TestImportParameters(t *testing.T) {
	th, ast, _ := importAndCollect(t, `@reference "./a.css";`, map[string]string{"/root/a.css": `@theme { --a: 1; } .x { color: red; }`})
	assertEqual(t, th.Options("--a")&Reference != 0, true)
	assertEqual(t, ast, []Node{
		NewContext(ContextMap{"base": "/root"}, NewContext(ContextMap{}, NewContext(ContextMap{"reference": "true"},
			NewContext(ContextMap{"base": "/root"}, StyleRule(".x", Decl("color", "red")))))),
	})

	th, _, state := importAndCollect(t, `@import "./a.css" theme(static) prefix(tw) important;`, map[string]string{"/root/a.css": `@theme { --a: 1; }`})
	assertEqual(t, th.Options("--a"), Static)
	assertEqual(t, th.Prefix, "tw")
	assertEqual(t, state.Important, true)

	th2 := NewTheme()
	bad := []Node{NewContext(ContextMap{"base": "/root"}, mustParse(t, `@import "./a.css" theme(reference);`)...)}
	_, _ = SubstituteAtImports(&bad, "/root", memoryLoader(map[string]string{"/root/a.css": `@theme { --a: 1; } .x { color: red; }`}))
	if _, err := CollectDirectives(&bad, th2); err == nil {
		t.Fatal("expected theme(reference) error")
	}

	_, _, state = importAndCollect(t, `@import "./sub/a.css" source("../app");`, map[string]string{"/root/sub/a.css": `@twill utilities;`})
	assertEqual(t, state.Root, &SourceRoot{Base: "/root", Pattern: "../app"})
	assertEqual(t, state.UtilitiesNode.Params, `utilities source("../app")`)

	_, ast, _ = importAndCollect(t, `@import "./a.css" print important;`, map[string]string{"/root/a.css": `.x { color: red; }`})
	assertEqual(t, Serialize(ast), "@media print {\n  .x {\n    color: red;\n  }\n}\n")
}

func TestBuiltinTheme(t *testing.T) {
	th, ast, state := importAndCollect(t, `
    @import "twill";
    @theme {
      --color-*: initial;
      --color-primary: oklch(0.6 0.2 250);
      --breakpoint-3xl: 120rem;
    }
  `, nil)
	get := func(value string, ns ...string) string {
		var v *string
		if value != "" {
			v = &value
		}
		got, ok := th.ResolveValue(v, ns)
		if !ok {
			return "<nil>"
		}
		return got
	}
	assertEqual(t, get("", "--spacing"), "0.25rem")
	assertEqual(t, get("md", "--breakpoint"), "48rem")
	assertEqual(t, get("3xl", "--breakpoint"), "120rem")
	assertEqual(t, get("md", "--container"), "28rem")
	assertEqual(t, get("lg", "--text"), "1.125rem")
	assertEqual(t, get("bold", "--font-weight"), "700")
	assertEqual(t, get("lg", "--radius"), "0.5rem")
	assertEqual(t, get("", "--default-transition-duration"), "150ms")
	assertEqual(t, get("red-500", "--color"), "<nil>")
	assertEqual(t, get("primary", "--color"), "oklch(0.6 0.2 250)")
	var names []string
	for _, k := range th.Keyframes() {
		names = append(names, k.Params)
	}
	assertEqual(t, names, []string{"spin", "ping", "pulse", "bounce"})
	assertEqual(t, state.UtilitiesNode != nil, true)
	css := Serialize(ast)
	if strings.Contains(css, "@theme") || !strings.Contains(css, "@layer theme {\n  :root, :host {\n  }\n}") {
		t.Fatal("unexpected output")
	}
}
