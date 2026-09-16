package twill

import (
	"reflect"
	"strings"
	"testing"
)

func mustParse(t *testing.T, css string) []Node {
	t.Helper()
	ast, err := Parse(css)
	if err != nil {
		t.Fatalf("parse %q: %v", css, err)
	}
	return ast
}

func assertEqual(t *testing.T, got, want any) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("mismatch\n got: %#v\nwant: %#v", got, want)
	}
}

func TestParseDeclarationsAndRules(t *testing.T) {
	assertEqual(t, mustParse(t, `.a { color: red; background: blue }`), []Node{
		StyleRule(".a", Decl("color", "red"), Decl("background", "blue")),
	})
}

func TestParseNesting(t *testing.T) {
	ast := mustParse(t, `.a {
      color: red;
      &:hover { color: blue; }
      @media (hover: hover) {
        .b & { color: green; }
      }
    }`)
	assertEqual(t, ast, []Node{
		StyleRule(".a",
			Decl("color", "red"),
			StyleRule("&:hover", Decl("color", "blue")),
			NewAtRule("@media", "(hover: hover)", StyleRule(".b &", Decl("color", "green"))),
		),
	})
}

func TestParseStatementAtRules(t *testing.T) {
	ast := mustParse(t, "@import \"foo.css\" layer(base);\n@charset \"utf-8\";\n@twill utilities;")
	assertEqual(t, ast, []Node{
		NewAtRule("@import", `"foo.css" layer(base)`),
		NewAtRule("@charset", `"utf-8"`),
		NewAtRule("@twill", "utilities"),
	})
	assertEqual(t, mustParse(t, `@media(width>=1px){.a{x:y}}`), []Node{
		NewAtRule("@media", "(width>=1px)", StyleRule(".a", Decl("x", "y"))),
	})
}

func TestParseComments(t *testing.T) {
	ast := mustParse(t, "/* license? no */\n/*! license */\n.a { /* inner */ color: red; }")
	assertEqual(t, ast, []Node{NewComment("! license "), StyleRule(".a", Decl("color", "red"))})
}

func TestParseImportant(t *testing.T) {
	assertEqual(t, mustParse(t, `.a { color: red !important; width: 1px!IMPORTANT }`), []Node{
		StyleRule(".a", ImportantDecl("color", "red"), ImportantDecl("width", "1px")),
	})
}

func TestParseCustomProperties(t *testing.T) {
	assertEqual(t, mustParse(t, `:root { --a: 1px; --b:; --c: { foo: bar; }; --d: calc(1px + 2px) }`), []Node{
		StyleRule(":root", Decl("--a", "1px"), Decl("--b", ""), Decl("--c", "{ foo: bar; }"), Decl("--d", "calc(1px + 2px)")),
	})
}

func TestParseDelimitersInsideQuotesAndParens(t *testing.T) {
	assertEqual(t, mustParse(t, `.a { content: "a;b{c}"; background: url(x;y{z}); font-family: 'q}q' }`), []Node{
		StyleRule(".a", Decl("content", `"a;b{c}"`), Decl("background", "url(x;y{z})"), Decl("font-family", "'q}q'")),
	})
}

func TestParseEscapes(t *testing.T) {
	assertEqual(t, mustParse(t, `.a\:b { color: red }`), []Node{StyleRule(`.a\:b`, Decl("color", "red"))})
	assertEqual(t, mustParse(t, `.a { content: "\"" }`), []Node{StyleRule(".a", Decl("content", `"\""`))})
}

func TestParseMissingSemicolonAndCRLF(t *testing.T) {
	assertEqual(t, mustParse(t, `.a { color: red }`), []Node{StyleRule(".a", Decl("color", "red"))})
	assertEqual(t, mustParse(t, `@import "a"`), []Node{NewAtRule("@import", `"a"`)})
	assertEqual(t, mustParse(t, ".a {\r\n  color: red;\r\n}\r\n"), []Node{StyleRule(".a", Decl("color", "red"))})
	assertEqual(t, mustParse(t, ""), []Node{})
	assertEqual(t, mustParse(t, "  \n "), []Node{})
}

func TestParseErrors(t *testing.T) {
	_, err := Parse(".a {\n  color: red;\n")
	e, ok := err.(*Error)
	if !ok || e.Position == nil || *e.Position != (Position{Line: 3, Column: 1}) {
		t.Fatalf("unexpected error %v", err)
	}
	_, err = Parse(".a { color: red; }\n}")
	e = err.(*Error)
	if *e.Position != (Position{Line: 2, Column: 1}) {
		t.Fatalf("unexpected position %v", e.Position)
	}
	for input, message := range map[string]string{
		".a { color: 'red }":      "Unterminated string",
		".a { color: red; /* x":   "Unterminated comment",
		".a { color: calc(1px; }": "Missing closing",
		".a { color }":            "Invalid declaration",
	} {
		_, err := Parse(input)
		if err == nil || !strings.Contains(err.Error(), message) {
			t.Fatalf("%q: expected %q, got %v", input, message, err)
		}
	}
}

func TestSerialize(t *testing.T) {
	ast := []Node{
		NewComment("! license"),
		NewAtRule("@import", `"a.css"`),
		StyleRule(".a",
			Decl("color", "red"),
			ImportantDecl("width", "1px"),
			&Declaration{Property: "hidden", NoValue: true},
			NewAtRule("@media", "(hover: hover)", StyleRule("&:hover", Decl("color", "blue"))),
			NewAtRule("@starting-style", "", Decl("opacity", "0")),
		),
	}
	want := `/*! license*/
@import "a.css";
.a {
  color: red;
  width: 1px !important;
  @media (hover: hover) {
    &:hover {
      color: blue;
    }
  }
  @starting-style {
    opacity: 0;
  }
}
`
	assertEqual(t, Serialize(ast), want)

	ctx := []Node{
		NewContext(ContextMap{"base": "/x"}, StyleRule(".a", Decl("color", "red"))),
		NewAtRoot(StyleRule(".b", Decl("color", "blue"))),
	}
	assertEqual(t, Serialize(ctx), ".a {\n  color: red;\n}\n.b {\n  color: blue;\n}\n")

	css := "@layer theme, base;\n:root, :host {\n  --spacing: 0.25rem;\n}\n@media (width >= 40rem) {\n  .sm\\:flex {\n    display: flex;\n  }\n}\n"
	assertEqual(t, Serialize(mustParse(t, css)), css)

	compactAst := mustParse(t, ".a { color: red !important; @media (x) { &:hover { y: z } } }\n@import \"a\";")
	assertEqual(t, SerializeCompact(compactAst), `.a{color:red!important;@media (x){&:hover{y:z;}}}@import "a";`)
}

func TestWalkReplace(t *testing.T) {
	ast := mustParse(t, ".a { color: red; } .b { color: blue; }")
	Walk(&ast, func(n Node, u *WalkUtils) WalkAction {
		if r, ok := n.(*Rule); ok && r.Selector == ".a" {
			u.ReplaceWith(StyleRule(".c", Decl("x", "y")), StyleRule(".d"))
		}
		return Continue
	})
	assertEqual(t, Serialize(ast), ".c {\n  x: y;\n}\n.d {\n}\n.b {\n  color: blue;\n}\n")
	var seen []string
	Walk(&ast, func(n Node, u *WalkUtils) WalkAction {
		if d, ok := n.(*Declaration); ok {
			seen = append(seen, d.Property)
		}
		return Continue
	})
	assertEqual(t, seen, []string{"x", "color"})
}
