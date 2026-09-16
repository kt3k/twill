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

func shadowRegistrations() []string {
	props := [][2]string{
		{"--tw-shadow", "0 0 #0000"}, {"--tw-shadow-color", ""}, {"--tw-inset-shadow", "0 0 #0000"},
		{"--tw-inset-shadow-color", ""}, {"--tw-ring-color", ""}, {"--tw-ring-shadow", "0 0 #0000"},
		{"--tw-inset-ring-color", ""}, {"--tw-inset-ring-shadow", "0 0 #0000"}, {"--tw-ring-inset", ""},
		{"--tw-ring-offset-width", "0px"}, {"--tw-ring-offset-color", "#fff"}, {"--tw-ring-offset-shadow", "0 0 #0000"},
	}
	var out []string
	for _, p := range props {
		reg := "@property " + p[0] + " {\n    syntax: \"*\";\n    inherits: false;\n"
		if p[1] != "" {
			reg += "    initial-value: " + p[1] + ";\n"
		}
		out = append(out, reg+"  }")
	}
	return out
}

const boxShadowDecl = "box-shadow: var(--tw-inset-shadow), var(--tw-inset-ring-shadow), var(--tw-ring-offset-shadow), var(--tw-ring-shadow), var(--tw-shadow);"

func TestUtilitiesShadowsAndRings(t *testing.T) {
	ds := designSystemFor(t, `@import "twill";
    @theme { --ring-width-thick: 4px; --box-shadow-color-glow: #ff0; }`)
	shadow := func(raw string, declarations ...string) {
		t.Helper()
		expectDecls(t, ds, raw, append(shadowRegistrations(), declarations...)...)
	}
	shadow("shadow", "--tw-shadow: 0 1px 3px 0 var(--tw-shadow-color, rgb(0 0 0 / 0.1)), 0 1px 2px -1px var(--tw-shadow-color, rgb(0 0 0 / 0.1));", boxShadowDecl)
	shadow("shadow-lg", "--tw-shadow: 0 10px 15px -3px var(--tw-shadow-color, rgb(0 0 0 / 0.1)), 0 4px 6px -4px var(--tw-shadow-color, rgb(0 0 0 / 0.1));", boxShadowDecl)
	shadow("shadow-2xl/50", "--tw-shadow: 0 25px 50px -12px var(--tw-shadow-color, color-mix(in oklab, rgb(0 0 0 / 0.25) 50%, transparent));", boxShadowDecl)
	shadow("shadow-none", "--tw-shadow: 0 0 #0000;", boxShadowDecl)
	shadow("shadow-red-500", "--tw-shadow-color: var(--color-red-500);")
	shadow("shadow-red-500/50", "--tw-shadow-color: color-mix(in oklab, var(--color-red-500) 50%, transparent);")
	shadow("shadow-glow", "--tw-shadow-color: var(--box-shadow-color-glow);")
	shadow("shadow-current", "--tw-shadow-color: currentcolor;")
	shadow("shadow-[0_0_3px_red,0_0_6px]", "--tw-shadow: 0 0 3px var(--tw-shadow-color, red), 0 0 6px var(--tw-shadow-color, currentcolor);", boxShadowDecl)
	shadow("shadow-[#fff]", "--tw-shadow-color: #fff;")
	shadow("shadow-[color:var(--c)]", "--tw-shadow-color: var(--c);")
	shadow("shadow-[var(--s)]", "--tw-shadow: var(--s);", boxShadowDecl)
	shadow("inset-shadow-sm", "--tw-inset-shadow: inset 0 2px 4px var(--tw-inset-shadow-color, rgb(0 0 0 / 0.05));", boxShadowDecl)
	shadow("inset-shadow-[0_2px_4px_red]", "--tw-inset-shadow: inset 0 2px 4px var(--tw-inset-shadow-color, red);", boxShadowDecl)
	shadow("inset-shadow-[inset_0_2px_red]", "--tw-inset-shadow: inset 0 2px var(--tw-inset-shadow-color, red);", boxShadowDecl)
	shadow("inset-shadow-none", "--tw-inset-shadow: 0 0 #0000;", boxShadowDecl)
	shadow("inset-shadow-red-500", "--tw-inset-shadow-color: var(--color-red-500);")
	shadow("ring", "--tw-ring-shadow: var(--tw-ring-inset,) 0 0 0 calc(1px + var(--tw-ring-offset-width)) var(--tw-ring-color, currentcolor);", boxShadowDecl)
	shadow("ring-2", "--tw-ring-shadow: var(--tw-ring-inset,) 0 0 0 calc(2px + var(--tw-ring-offset-width)) var(--tw-ring-color, currentcolor);", boxShadowDecl)
	shadow("ring-thick", "--tw-ring-shadow: var(--tw-ring-inset,) 0 0 0 calc(var(--ring-width-thick) + var(--tw-ring-offset-width)) var(--tw-ring-color, currentcolor);", boxShadowDecl)
	shadow("ring-[3px]", "--tw-ring-shadow: var(--tw-ring-inset,) 0 0 0 calc(3px + var(--tw-ring-offset-width)) var(--tw-ring-color, currentcolor);", boxShadowDecl)
	shadow("ring-red-500/30", "--tw-ring-color: color-mix(in oklab, var(--color-red-500) 30%, transparent);")
	shadow("ring-[#000]", "--tw-ring-color: #000;")
	shadow("ring-inset", "--tw-ring-inset: inset;")
	shadow("inset-ring", "--tw-inset-ring-shadow: inset 0 0 0 1px var(--tw-inset-ring-color, currentcolor);", boxShadowDecl)
	shadow("inset-ring-2", "--tw-inset-ring-shadow: inset 0 0 0 2px var(--tw-inset-ring-color, currentcolor);", boxShadowDecl)
	shadow("inset-ring-blue-500", "--tw-inset-ring-color: var(--color-blue-500);")
	shadow("ring-offset-2", "--tw-ring-offset-width: 2px;", "--tw-ring-offset-shadow: var(--tw-ring-inset,) 0 0 0 var(--tw-ring-offset-width) var(--tw-ring-offset-color);")
	shadow("ring-offset-[3px]", "--tw-ring-offset-width: 3px;", "--tw-ring-offset-shadow: var(--tw-ring-inset,) 0 0 0 var(--tw-ring-offset-width) var(--tw-ring-offset-color);")
	shadow("ring-offset-white", "--tw-ring-offset-color: var(--color-white);")
	expectInvalid(t, ds, "shadow-nope", "shadow-lg/foo", "inset-shadow", "ring-x", "ring-2/50", "ring-[3px]/50", "ring-offset", "ring-offset-2/50", "inset-ring-x")
}

func TestReplaceShadowColors(t *testing.T) {
	wrap := func(c string) string { return "<" + c + ">" }
	assertEqual(t, ReplaceShadowColors("0 0 3px red, inset 0 1px", wrap, false), "0 0 3px <red>, inset 0 1px <currentcolor>")
	assertEqual(t, ReplaceShadowColors("none", wrap, false), "none")
	assertEqual(t, ReplaceShadowColors("var(--x)", wrap, false), "var(--x)")
	assertEqual(t, ReplaceShadowColors("0 2px 4px rgb(0 0 0 / 0.1)", wrap, true), "inset 0 2px 4px <rgb(0 0 0 / 0.1)>")
	assertEqual(t, ReplaceShadowColors("inset 0 2px 4px #000", wrap, true), "inset 0 2px 4px <#000>")
	assertEqual(t, ReplaceShadowColors("0px  1px\n  0px #000", wrap, false), "0px 1px 0px <#000>")
}

func registrations(names []string, initial string) []string {
	var out []string
	for _, n := range names {
		reg := "@property " + n + " {\n    syntax: \"*\";\n    inherits: false;\n"
		if initial != "" {
			reg += "    initial-value: " + initial + ";\n"
		}
		out = append(out, reg+"  }")
	}
	return out
}

func TestUtilitiesTransforms(t *testing.T) {
	ds := designSystemFor(t, "")
	transformRegs := registrations([]string{"--tw-rotate-x", "--tw-rotate-y", "--tw-rotate-z", "--tw-skew-x", "--tw-skew-y"}, "")
	translateRegs := registrations([]string{"--tw-translate-x", "--tw-translate-y", "--tw-translate-z"}, "0")
	scaleRegs := registrations([]string{"--tw-scale-x", "--tw-scale-y", "--tw-scale-z"}, "1")
	transformDecl := "transform: var(--tw-rotate-x,) var(--tw-rotate-y,) var(--tw-rotate-z,) var(--tw-skew-x,) var(--tw-skew-y,);"
	with := func(regs []string, decls ...string) []string { return append(append([]string{}, regs...), decls...) }

	expectDecls(t, ds, "transform-none", "transform: none;")
	expectDecls(t, ds, "transform-gpu", with(transformRegs, "transform: translateZ(0) var(--tw-rotate-x,) var(--tw-rotate-y,) var(--tw-rotate-z,) var(--tw-skew-x,) var(--tw-skew-y,);")...)
	expectDecls(t, ds, "transform-[matrix(1,0,0,1,0,0)]", "transform: matrix(1,0,0,1,0,0);")
	expectDecls(t, ds, "transform-3d", "transform-style: preserve-3d;")
	expectDecls(t, ds, "backface-hidden", "backface-visibility: hidden;")
	expectDecls(t, ds, "perspective-near", "perspective: var(--perspective-near);")
	expectDecls(t, ds, "perspective-origin-top-left", "perspective-origin: top left;")
	expectDecls(t, ds, "origin-[10px_20px]", "transform-origin: 10px 20px;")
	expectDecls(t, ds, "translate-4", with(translateRegs, "--tw-translate-x: --spacing(4);", "--tw-translate-y: --spacing(4);", "translate: var(--tw-translate-x) var(--tw-translate-y);")...)
	expectDecls(t, ds, "-translate-x-1/2", with(translateRegs, "--tw-translate-x: calc(calc(1 / 2 * 100%) * -1);", "translate: var(--tw-translate-x) var(--tw-translate-y);")...)
	expectDecls(t, ds, "-translate-y-full", with(translateRegs, "--tw-translate-y: -100%;", "translate: var(--tw-translate-x) var(--tw-translate-y);")...)
	expectDecls(t, ds, "translate-z-px", with(translateRegs, "--tw-translate-z: 1px;", "translate: var(--tw-translate-x) var(--tw-translate-y) var(--tw-translate-z);")...)
	expectDecls(t, ds, "translate-none", "translate: none;")
	expectDecls(t, ds, "scale-50", with(scaleRegs, "--tw-scale-x: 50%;", "--tw-scale-y: 50%;", "--tw-scale-z: 50%;", "scale: var(--tw-scale-x) var(--tw-scale-y);")...)
	expectDecls(t, ds, "-scale-x-75", with(scaleRegs, "--tw-scale-x: calc(75% * -1);", "scale: var(--tw-scale-x) var(--tw-scale-y);")...)
	expectDecls(t, ds, "scale-z-150", with(scaleRegs, "--tw-scale-z: 150%;", "scale: var(--tw-scale-x) var(--tw-scale-y) var(--tw-scale-z);")...)
	expectDecls(t, ds, "scale-[1.5]", "scale: 1.5;")
	expectDecls(t, ds, "rotate-45", "rotate: 45deg;")
	expectDecls(t, ds, "-rotate-45", "rotate: calc(45deg * -1);")
	expectDecls(t, ds, "rotate-[30deg]", "rotate: 30deg;")
	expectDecls(t, ds, "rotate-x-30", with(transformRegs, "--tw-rotate-x: rotateX(30deg);", transformDecl)...)
	expectDecls(t, ds, "-skew-6", with(transformRegs, "--tw-skew-x: skewX(calc(6deg * -1));", "--tw-skew-y: skewY(calc(6deg * -1));", transformDecl)...)
	expectDecls(t, ds, "skew-y-[10deg]", with(transformRegs, "--tw-skew-y: skewY(10deg);", transformDecl)...)
	expectInvalid(t, ds, "transform", "origin-nope", "translate-z-1/2", "scale-1.5", "scale-50/2", "-scale-[1.5]", "rotate-1.5", "rotate-45/2", "skew-x", "perspective-500")
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
