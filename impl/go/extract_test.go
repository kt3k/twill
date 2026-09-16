package twill

import (
	"sort"
	"testing"
)

func extract(text string) []string {
	out := ExtractCandidates(text)
	sort.Strings(out)
	if out == nil {
		out = []string{}
	}
	return out
}

func TestExtractHTMLAttributes(t *testing.T) {
	assertEqual(t, extract(`<div class="flex p-4 hover:underline md:grid-cols-2 bg-red-500/50"></div>`),
		[]string{"bg-red-500/50", "class", "flex", "hover:underline", "md:grid-cols-2", "p-4"})
}

func TestExtractTemplateLiteralsAndObjectKeys(t *testing.T) {
	assertEqual(t, extract("const c = `w-[13px] ${x} text-lg/8`; const o = { 'sm:flex': true, \"dark:bg-black\": 1 };"),
		[]string{"c", "const", "dark:bg-black", "o", "sm:flex", "text-lg/8", "w-[13px]"})
}

func TestExtractArbitraryValuesPropertiesVariants(t *testing.T) {
	assertEqual(t, extract(`"[mask-type:luminance] bg-[url(/a_b.png)] [&_p]:flex has-[>img]:block data-[state=open]:flex w-(--my-w) supports-[display:grid]:grid"`),
		[]string{"[&_p]:flex", "[mask-type:luminance]", "bg-[url(/a_b.png)]", "data-[state=open]:flex", "has-[>img]:block", "supports-[display:grid]:grid", "w-(--my-w)"})
}

func TestExtractImportanceNegativesContainersFractions(t *testing.T) {
	assertEqual(t, extract(`'underline! !flex -mt-2 @md:flex @md/main:grid w-1/2 p-1.5 opacity-50% aria-[label=foo_i]:hidden'`),
		[]string{"!flex", "-mt-2", "@md/main:grid", "@md:flex", "aria-[label=foo_i]:hidden", "opacity-50%", "p-1.5", "underline!", "w-1/2"})
}

func TestExtractBoundaries(t *testing.T) {
	assertEqual(t, extract(`.flex{}</div>p-4=x`), []string{"flex", "p-4"})
	assertEqual(t, extract(`xflex yhover:flex`), []string{"xflex", "yhover:flex"})
	assertEqual(t, extract(`https://example.com/path`), []string{"com/path"})
	assertEqual(t, extract(`foo- bar_ -@x`), []string{})
	assertEqual(t, extract(`a.b c.d`), []string{"b", "d"})
	assertEqual(t, extract(`class:list={["flex"]}`), []string{"class:list", "flex"})
}

func TestExtractCustomPropertyReferences(t *testing.T) {
	assertEqual(t, extract(`"--color-red-500" '--spacing' --x --`), []string{"--color-red-500", "--spacing", "--x"})
	assertEqual(t, extract(`var(--color-red-500)`), []string{})
}

func TestExtractUnbalancedBrackets(t *testing.T) {
	assertEqual(t, extract(`w-[13px`), []string{})
	assertEqual(t, extract("w-[a\nb]"), []string{"b"})
}

func TestExtractOrderAndDedupe(t *testing.T) {
	assertEqual(t, ExtractCandidates("b a b c"), []string{"b", "a", "c"})
}
