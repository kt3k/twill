package twill

import (
	"strings"
	"testing"
)

const borderStyleProperty = "@property --tw-border-style {\n    syntax: \"*\";\n    inherits: false;\n    initial-value: solid;\n  }"
const fontWeightProperty = "@property --tw-font-weight {\n    syntax: \"*\";\n    inherits: false;\n  }"
const leadingProperty = "@property --tw-leading {\n    syntax: \"*\";\n    inherits: false;\n  }"
const trackingProperty = "@property --tw-tracking {\n    syntax: \"*\";\n    inherits: false;\n  }"

func expectDecls(t *testing.T, ds *DesignSystem, raw string, declarations ...string) {
	t.Helper()
	want := "." + Escape(raw) + " {\n  " + strings.Join(declarations, "\n  ") + "\n}\n"
	if got := compileRaw(ds, raw); got != want {
		t.Errorf("%s:\n got: %q\nwant: %q", raw, got, want)
	}
}

func expectInvalid(t *testing.T, ds *DesignSystem, raws ...string) {
	t.Helper()
	for _, raw := range raws {
		if got := compileRaw(ds, raw); got != "" {
			t.Errorf("%s should be invalid, got %q", raw, got)
		}
	}
}

func TestUtilitiesSpec162(t *testing.T) {
	ds := designSystemFor(t, "")
	expectDecls(t, ds, "p-4", "padding: --spacing(4);")
	expectDecls(t, ds, "p-1", "padding: --spacing(1);")
	expectDecls(t, ds, "p-0", "padding: --spacing(0);")
	expectDecls(t, ds, "p-px", "padding: 1px;")
	expectDecls(t, ds, "-mt-2", "margin-top: --spacing(-2);")
	expectDecls(t, ds, "w-1/2", "width: calc(1 / 2 * 100%);")
	expectDecls(t, ds, "w-[13px]", "width: 13px;")
	expectDecls(t, ds, "w-(--my-w)", "width: var(--my-w);")
	expectDecls(t, ds, "max-w-md", "max-width: var(--container-md);")
	expectDecls(t, ds, "bg-red-500", "background-color: var(--color-red-500);")
	expectDecls(t, ds, "bg-red-500/50", "background-color: color-mix(in oklab, var(--color-red-500) 50%, transparent);")
	expectDecls(t, ds, "bg-[#0088cc]", "background-color: #0088cc;")
	expectDecls(t, ds, "bg-[url(/a_b.png)]", "background-image: url(/a_b.png);")
	expectDecls(t, ds, "bg-[length:10px_20px]", "background-size: 10px 20px;")
	expectDecls(t, ds, "text-lg", "font-size: var(--text-lg);", "line-height: var(--tw-leading, var(--text-lg--line-height));")
	expectDecls(t, ds, "text-lg/8", "font-size: var(--text-lg);", "line-height: --spacing(8);")
	expectDecls(t, ds, "text-red-500", "color: var(--color-red-500);")
	expectDecls(t, ds, "font-bold", fontWeightProperty, "--tw-font-weight: var(--font-weight-bold);", "font-weight: var(--font-weight-bold);")
	expectDecls(t, ds, "rounded-lg", "border-radius: var(--radius-lg);")
	expectDecls(t, ds, "rounded-full", "border-radius: calc(infinity * 1px);")
	expectDecls(t, ds, "border", borderStyleProperty, "border-style: var(--tw-border-style);", "border-width: 1px;")
	expectDecls(t, ds, "border-2", borderStyleProperty, "border-style: var(--tw-border-style);", "border-width: 2px;")
	expectDecls(t, ds, "z-10", "z-index: 10;")
	expectDecls(t, ds, "-z-10", "z-index: calc(10 * -1);")
	expectDecls(t, ds, "flex-1", "flex: 1;")
	expectDecls(t, ds, "opacity-50", "opacity: 50%;")
	expectDecls(t, ds, "[mask-type:luminance]", "mask-type: luminance;")
	expectDecls(t, ds, "[--my-var:1px]", "--my-var: 1px;")
	expectDecls(t, ds, "[color:red]/50", "color: color-mix(in oklab, red 50%, transparent);")
}

func TestUtilitiesLayoutFlexGrid(t *testing.T) {
	ds := designSystemFor(t, "")
	expectDecls(t, ds, "flex", "display: flex;")
	expectDecls(t, ds, "hidden", "display: none;")
	expectDecls(t, ds, "absolute", "position: absolute;")
	expectDecls(t, ds, "invisible", "visibility: hidden;")
	expectDecls(t, ds, "overflow-x-auto", "overflow-x: auto;")
	expectDecls(t, ds, "float-start", "float: inline-start;")
	expectDecls(t, ds, "inset-x-4", "inset-inline: --spacing(4);")
	expectDecls(t, ds, "-inset-1", "inset: --spacing(-1);")
	expectDecls(t, ds, "top-1/2", "top: calc(1 / 2 * 100%);")
	expectDecls(t, ds, "-top-full", "top: -100%;")
	expectDecls(t, ds, "left-[10%]", "left: 10%;")
	expectDecls(t, ds, "order-first", "order: -9999;")
	expectDecls(t, ds, "-order-2", "order: calc(2 * -1);")
	expectInvalid(t, ds, "z-1.5", "z-10/50", "p", "p-foo", "p-4/2", "inset-x", "overflow-nope")

	expectDecls(t, ds, "flex-col", "flex-direction: column;")
	expectDecls(t, ds, "flex-initial", "flex: 0 auto;")
	expectDecls(t, ds, "flex-1/2", "flex: calc(1 / 2 * 100%);")
	expectDecls(t, ds, "flex-[2_2_0%]", "flex: 2 2 0%;")
	expectDecls(t, ds, "grow", "flex-grow: 1;")
	expectDecls(t, ds, "grow-0", "flex-grow: 0;")
	expectDecls(t, ds, "basis-1/3", "flex-basis: calc(1 / 3 * 100%);")
	expectDecls(t, ds, "basis-md", "flex-basis: var(--container-md);")
	expectDecls(t, ds, "grid-cols-3", "grid-template-columns: repeat(3, minmax(0, 1fr));")
	expectDecls(t, ds, "grid-cols-[1fr_2fr]", "grid-template-columns: 1fr 2fr;")
	expectDecls(t, ds, "col-span-2", "grid-column: span 2 / span 2;")
	expectDecls(t, ds, "col-span-full", "grid-column: 1 / -1;")
	expectDecls(t, ds, "-col-end-1", "grid-column-end: calc(1 * -1);")
	expectDecls(t, ds, "row-3", "grid-row: 3;")
	expectDecls(t, ds, "auto-cols-fr", "grid-auto-columns: minmax(0, 1fr);")
	expectDecls(t, ds, "gap-x-2", "column-gap: --spacing(2);")
	expectDecls(t, ds, "gap-y-px", "row-gap: 1px;")
	expectDecls(t, ds, "justify-between", "justify-content: space-between;")
	expectDecls(t, ds, "items-start", "align-items: flex-start;")
	expectDecls(t, ds, "content-evenly", "align-content: space-evenly;")
	expectDecls(t, ds, "place-content-start", "place-content: start;")
	expectDecls(t, ds, "place-self-auto", "place-self: auto;")
	expectInvalid(t, ds, "flex-1.5", "grow-x", "col-span-x", "grid-cols-0.5")
}

func TestUtilitiesSpacingSizing(t *testing.T) {
	ds := designSystemFor(t, "")
	expectDecls(t, ds, "px-4", "padding-inline: --spacing(4);")
	expectDecls(t, ds, "pt-0.5", "padding-top: --spacing(0.5);")
	expectDecls(t, ds, "pl-[3px]", "padding-left: 3px;")
	expectDecls(t, ds, "mx-auto", "margin-inline: auto;")
	expectDecls(t, ds, "-mx-px", "margin-inline: -1px;")
	expectDecls(t, ds, "-m-[2px]", "margin: calc(2px * -1);")
	expectInvalid(t, ds, "-p-4", "p-0.3", "m-4/2")

	expectDecls(t, ds, "w-full", "width: 100%;")
	expectDecls(t, ds, "w-screen", "width: 100vw;")
	expectDecls(t, ds, "w-xl", "width: var(--container-xl);")
	expectDecls(t, ds, "max-w-none", "max-width: none;")
	expectDecls(t, ds, "h-dvh", "height: 100dvh;")
	expectDecls(t, ds, "h-1/3", "height: calc(1 / 3 * 100%);")
	expectDecls(t, ds, "max-h-[50vh]", "max-height: 50vh;")
	expectDecls(t, ds, "size-4", "--tw-sort: size;", "width: --spacing(4);", "height: --spacing(4);")
	expectDecls(t, ds, "size-full", "--tw-sort: size;", "width: 100%;", "height: 100%;")
	expectInvalid(t, ds, "w-1/1.5", "w-4/foo", "h-md")
}

func TestUtilitiesTypography(t *testing.T) {
	ds := designSystemFor(t, "")
	if !strings.Contains(compileRaw(ds, "font-sans"), "font-family: var(--font-sans);") {
		t.Fatal("font-sans")
	}
	expectDecls(t, ds, "font-[Inter]", "font-family: Inter;")
	expectDecls(t, ds, "font-[700]", fontWeightProperty, "--tw-font-weight: 700;", "font-weight: 700;")
	expectDecls(t, ds, "font-[family-name:foo]", "font-family: foo;")
	expectDecls(t, ds, "text-center", "text-align: center;")
	expectDecls(t, ds, "text-[14px]", "font-size: 14px;")
	expectDecls(t, ds, "text-[14px]/6", "font-size: 14px;", "line-height: --spacing(6);")
	expectDecls(t, ds, "text-[red]", "color: red;")
	expectDecls(t, ds, "text-[color:var(--x)]", "color: var(--x);")
	expectDecls(t, ds, "text-lg/tight", "font-size: var(--text-lg);", "line-height: var(--leading-tight);")
	expectDecls(t, ds, "text-lg/none", "font-size: var(--text-lg);", "line-height: 1;")
	expectDecls(t, ds, "text-lg/[1.2]", "font-size: var(--text-lg);", "line-height: 1.2;")
	expectDecls(t, ds, "text-current", "color: currentcolor;")
	expectDecls(t, ds, "leading-none", leadingProperty, "--tw-leading: 1;", "line-height: 1;")
	expectDecls(t, ds, "leading-6", leadingProperty, "--tw-leading: --spacing(6);", "line-height: --spacing(6);")
	expectDecls(t, ds, "tracking-tight", trackingProperty, "--tw-tracking: var(--tracking-tight);", "letter-spacing: var(--tracking-tight);")
	expectDecls(t, ds, "-tracking-tight", trackingProperty, "--tw-tracking: calc(var(--tracking-tight) * -1);", "letter-spacing: calc(var(--tracking-tight) * -1);")
	expectDecls(t, ds, "truncate", "overflow: hidden;", "text-overflow: ellipsis;", "white-space: nowrap;")
	expectDecls(t, ds, "break-all", "word-break: break-all;")
	expectDecls(t, ds, "antialiased", "-webkit-font-smoothing: antialiased;", "-moz-osx-font-smoothing: grayscale;")
	expectDecls(t, ds, "underline-offset-2", "text-underline-offset: 2px;")
	expectDecls(t, ds, "-underline-offset-2", "text-underline-offset: -2px;")
	expectDecls(t, ds, "-indent-4", "text-indent: --spacing(-4);")
	expectInvalid(t, ds, "text-lg/nope", "text-nope", "font-nope", "font-bold/50", "text-center/50")
}

func TestUtilitiesBackgroundsBorders(t *testing.T) {
	ds := designSystemFor(t, "")
	expectDecls(t, ds, "bg-[10px_20px]", "background-position: 10px 20px;")
	expectDecls(t, ds, "bg-[50%]", "background-position: 50%;")
	expectDecls(t, ds, "bg-[cover]", "background-size: cover;")
	expectDecls(t, ds, "bg-[linear-gradient(red,blue)]", "background-image: linear-gradient(red,blue);")
	expectDecls(t, ds, "bg-[var(--x)]/50", "background-color: color-mix(in oklab, var(--x) 50%, transparent);")
	expectDecls(t, ds, "bg-[#0088cc]/[0.3]", "background-color: color-mix(in oklab, #0088cc 30%, transparent);")
	expectDecls(t, ds, "bg-current", "background-color: currentcolor;")
	expectDecls(t, ds, "bg-no-repeat", "background-repeat: no-repeat;")
	expectDecls(t, ds, "bg-clip-text", "background-clip: text;")
	expectDecls(t, ds, "bg-origin-content", "background-origin: content-box;")
	expectInvalid(t, ds, "bg-[10px_20px]/50", "bg-red-500/30.3", "bg-nope", "bg")

	expectDecls(t, ds, "border-red-500", "border-color: var(--color-red-500);")
	expectDecls(t, ds, "border-t-red-500/50", "border-top-color: color-mix(in oklab, var(--color-red-500) 50%, transparent);")
	expectDecls(t, ds, "border-t-2", borderStyleProperty, "border-style: var(--tw-border-style);", "border-top-width: 2px;")
	expectDecls(t, ds, "border-x", borderStyleProperty, "border-style: var(--tw-border-style);", "border-inline-width: 1px;")
	expectDecls(t, ds, "border-[thin]", borderStyleProperty, "border-style: var(--tw-border-style);", "border-width: thin;")
	expectDecls(t, ds, "border-[#fff]", "border-color: #fff;")
	expectDecls(t, ds, "border-[length:var(--w)]", borderStyleProperty, "border-style: var(--tw-border-style);", "border-width: var(--w);")
	expectDecls(t, ds, "border-dashed", "--tw-border-style: dashed;", "border-style: dashed;")
	expectDecls(t, ds, "rounded-t-lg", "border-top-left-radius: var(--radius-lg);", "border-top-right-radius: var(--radius-lg);")
	expectDecls(t, ds, "rounded-ss-none", "border-start-start-radius: 0;")
	expectDecls(t, ds, "rounded", "border-radius: 0.25rem;")
	expectInvalid(t, ds, "border-2/50", "border-nope", "border/50")

	custom := designSystemFor(t, `@import "twill"; @theme { --default-border-width: 2px; --opacity-half: 50%; }`)
	expectDecls(t, custom, "border", borderStyleProperty, "border-style: var(--tw-border-style);", "border-width: 2px;")
	expectDecls(t, custom, "bg-red-500/half", "background-color: color-mix(in oklab, var(--color-red-500) var(--opacity-half), transparent);")
}

func TestUtilitiesEffects(t *testing.T) {
	ds := designSystemFor(t, "")
	expectDecls(t, ds, "opacity-[.5]", "opacity: .5;")
	expectDecls(t, ds, "opacity-75", "opacity: 75%;")
	expectInvalid(t, ds, "opacity-33.3", "opacity-50/50")
	if !strings.Contains(compileRaw(ds, "transition"), "transition-property: color, background-color, border-color, outline-color, text-decoration-color, fill, stroke, --tw-gradient-from, --tw-gradient-via, --tw-gradient-to, opacity, box-shadow, transform, translate, scale, rotate, filter, -webkit-backdrop-filter, backdrop-filter, display, content-visibility, overlay, pointer-events;") {
		t.Fatal("transition")
	}
	expectDecls(t, ds, "transition-opacity", "transition-property: opacity;", "transition-timing-function: var(--default-transition-timing-function);", "transition-duration: var(--default-transition-duration);")
	expectDecls(t, ds, "duration-300", "transition-duration: 300ms;")
	expectDecls(t, ds, "ease-in-out", "transition-timing-function: var(--ease-in-out);")
	expectDecls(t, ds, "animate-spin", "animation: var(--animate-spin);")
	expectDecls(t, ds, "cursor-pointer", "cursor: pointer;")
	expectDecls(t, ds, "select-none", "-webkit-user-select: none;", "user-select: none;")
	expectDecls(t, ds, "resize-y", "resize: vertical;")
	expectDecls(t, ds, "will-change-[top]", "will-change: top;")
	expectDecls(t, ds, "content-['hi']", "@property --tw-content {\n    syntax: \"*\";\n    inherits: false;\n    initial-value: \"\";\n  }", "--tw-content: 'hi';", "content: var(--tw-content);")
	expectDecls(t, ds, "aspect-video", "aspect-ratio: var(--aspect-video);")
	expectDecls(t, ds, "aspect-16/9", "aspect-ratio: 16 / 9;")
	expectDecls(t, ds, "columns-3", "columns: 3;")
	expectDecls(t, ds, "columns-md", "columns: var(--container-md);")
	expectDecls(t, ds, "object-cover", "object-fit: cover;")
	expectDecls(t, ds, "accent-red-500", "accent-color: var(--color-red-500);")
	expectDecls(t, ds, "fill-current", "fill: currentcolor;")
	expectDecls(t, ds, "stroke-red-500/50", "stroke: color-mix(in oklab, var(--color-red-500) 50%, transparent);")
	expectInvalid(t, ds, "content-hi", "aspect-16/x", "columns-1.5", "fill", "accent-nope")

	custom := designSystemFor(t, `@import "twill";
    @theme { --color-*: initial; --color-primary: oklch(0.6 0.2 250); --breakpoint-3xl: 120rem; --font-display: "Inter", sans-serif; }`)
	expectDecls(t, custom, "bg-primary", "background-color: var(--color-primary);")
	expectDecls(t, custom, "font-display", "font-family: var(--font-display);")
	assertEqual(t, compileRaw(custom, "3xl:flex"), ".\\33 xl\\:flex {\n  @media (width >= 120rem) {\n    display: flex;\n  }\n}\n")
	expectInvalid(t, custom, "bg-red-500")
}

func TestInferDataType(t *testing.T) {
	cases := []struct {
		value string
		types []string
		want  string
	}{
		{"red", []string{"color"}, "color"}, {"#ffff", []string{"color"}, "color"}, {"#ggg", []string{"color"}, ""},
		{"color-mix(in oklab, red, blue)", []string{"color"}, "color"}, {"var(--x)", []string{"color"}, ""},
		{"1.5rem", []string{"length"}, "length"}, {"0", []string{"length"}, "length"}, {"calc(1px + 2px)", []string{"length"}, "length"},
		{"--spacing(4)", []string{"length"}, "length"}, {"10", []string{"length"}, ""}, {"10%", []string{"percentage"}, "percentage"},
		{".5", []string{"number"}, "number"}, {"1.5", []string{"integer"}, ""}, {"16 / 9", []string{"ratio"}, "ratio"},
		{"url(/a.png)", []string{"url"}, "url"}, {"repeating-radial-gradient(red, blue)", []string{"image"}, "image"},
		{"url(/a.png)", []string{"image"}, ""}, {"left 10px top 20px", []string{"position"}, "position"}, {"foo", []string{"position"}, ""},
		{"10px auto", []string{"bg-size"}, "bg-size"}, {"thin", []string{"line-width"}, "line-width"}, {"x-large", []string{"absolute-size"}, "absolute-size"},
		{"'Segoe UI', Roboto", []string{"family-name"}, "family-name"}, {"700", []string{"family-name"}, ""}, {"sans-serif", []string{"generic-name"}, "generic-name"},
		{"0.5turn", []string{"angle"}, "angle"}, {"1 0 0", []string{"vector"}, "vector"}, {"1 0", []string{"vector"}, ""},
		{"10px", []string{"color", "length"}, "length"}, {"10px 20px", []string{"percentage", "position", "bg-size"}, "position"},
		{"700", []string{"number", "family-name"}, "number"},
	}
	for _, c := range cases {
		if got := InferDataType(c.value, c.types); got != c.want {
			t.Errorf("infer(%q, %v) = %q, want %q", c.value, c.types, got, c.want)
		}
	}
}

func TestDesignSystemMemoization(t *testing.T) {
	ds := designSystemFor(t, "")
	if len(ds.ParseCandidate("flex")) != 2 || len(ds.ParseCandidate("block")) != 1 || len(ds.ParseCandidate("nope")) != 0 {
		t.Fatal("parse counts")
	}
	if ds.ParseVariant("hover") != ds.ParseVariant("hover") || ds.ParseVariant("nope") != nil {
		t.Fatal("memoization")
	}
	prefixed := designSystemFor(t, `@import "twill" prefix(tw);`)
	assertEqual(t, len(prefixed.ParseCandidate("flex")), 0)
	assertEqual(t, len(prefixed.ParseCandidate("tw:block")), 1)
	assertEqual(t, designSystemFor(t, `@import "twill" important;`).Important, true)
}
