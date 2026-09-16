package twill

import "testing"

func makeTheme() *Theme {
	th := NewTheme()
	for _, kv := range [][2]string{
		{"--color-red-500", "red"}, {"--color-blue-500", "blue"}, {"--spacing", "0.25rem"},
		{"--text-lg", "1.125rem"}, {"--text-lg--line-height", "calc(1.75 / 1.125)"},
		{"--text-shadow-sm", "0 1px 1px black"}, {"--font-sans", "ui-sans-serif"},
		{"--font-weight-bold", "700"}, {"--breakpoint-md", "48rem"}, {"--container-1_5", "1.5rem"},
	} {
		_ = th.Add(kv[0], kv[1], 0)
	}
	return th
}

func resolve(t *testing.T, th *Theme, value string, namespaces ...string) string {
	t.Helper()
	var v *string
	if value != "" {
		v = &value
	}
	got, ok := th.Resolve(v, namespaces, 0)
	if !ok {
		return "<nil>"
	}
	return got
}

func TestThemeResolve(t *testing.T) {
	th := makeTheme()
	assertEqual(t, resolve(t, th, "red-500", "--color"), "var(--color-red-500)")
	assertEqual(t, resolve(t, th, "", "--spacing"), "var(--spacing)")
	assertEqual(t, resolve(t, th, "missing", "--color"), "<nil>")
	assertEqual(t, resolve(t, th, "red-500", "--text-color", "--color"), "var(--color-red-500)")
	assertEqual(t, resolve(t, th, "1.5", "--container"), "var(--container-1_5)")
	assertEqual(t, resolve(t, th, "shadow-sm", "--text"), "<nil>")
	assertEqual(t, resolve(t, th, "weight-bold", "--font"), "<nil>")
	assertEqual(t, resolve(t, th, "bold", "--font-weight"), "var(--font-weight-bold)")

	th2 := NewTheme()
	_ = th2.Add("--a", "1", Inline)
	_ = th2.Add("--b", "2", Reference)
	_ = th2.Add("--c", "3", 0)
	assertEqual(t, resolve(t, th2, "", "--a"), "1")
	assertEqual(t, resolve(t, th2, "", "--b"), "var(--b, 2)")
	assertEqual(t, resolve(t, th2, "", "--c"), "var(--c)")
	inline, _ := th2.Resolve(nil, []string{"--c"}, Inline)
	assertEqual(t, inline, "3")
}

func TestThemeDefaultAndClearing(t *testing.T) {
	th := NewTheme()
	_ = th.Add("--color-a", "author", 0)
	_ = th.Add("--color-a", "default", Default)
	assertEqual(t, th.Entry("--color-a").Value, "author")
	_ = th.Add("--color-b", "default", Default)
	_ = th.Add("--color-b", "author", 0)
	assertEqual(t, th.Entry("--color-b").Value, "author")
	_ = th.Add("--color-c", "default-1", Default)
	_ = th.Add("--color-c", "default-2", Default)
	assertEqual(t, th.Entry("--color-c").Value, "default-2")

	th = makeTheme()
	_ = th.Add("--color-red-500", "initial", 0)
	assertEqual(t, th.Has("--color-red-500"), false)
	_ = th.Add("--color-*", "initial", 0)
	assertEqual(t, th.Has("--color-blue-500"), false)
	assertEqual(t, th.Has("--spacing"), true)
	_ = th.Add("--text-*", "initial", 0)
	assertEqual(t, th.Has("--text-lg"), false)
	assertEqual(t, th.Has("--text-shadow-sm"), true)
	_ = th.Add("--font-*", "initial", 0)
	assertEqual(t, th.Has("--font-sans"), false)
	assertEqual(t, th.Has("--font-weight-bold"), true)
	if th.Add("--color-*", "red", 0) == nil {
		t.Fatal("expected error")
	}
	_ = th.Add("--*", "initial", 0)
	assertEqual(t, th.Size(), 0)
}

func TestThemeNamespaces(t *testing.T) {
	th := makeTheme()
	lg := "lg"
	value, extra, ok := th.ResolveWith(&lg, []string{"--text"}, []string{"--line-height", "--letter-spacing"})
	assertEqual(t, ok, true)
	assertEqual(t, value, "var(--text-lg)")
	assertEqual(t, extra, map[string]string{"--line-height": "var(--text-lg--line-height)"})
	assertEqual(t, th.Namespace("--text"), []NamespaceEntry{
		{Key: "lg", Value: "1.125rem"},
		{Key: "lg--line-height", Value: "calc(1.75 / 1.125)"},
		{Key: "shadow-sm", Value: "0 1px 1px black"},
	})
	assertEqual(t, th.Namespace("--spacing"), []NamespaceEntry{{Self: true, Value: "0.25rem"}})
	assertEqual(t, th.KeysInNamespaces([]string{"--text"}), []string{"lg"})
	assertEqual(t, th.KeysInNamespaces([]string{"--color", "--breakpoint"}), []string{"red-500", "blue-500", "md"})
	v, _ := th.Get("--missing", "--spacing")
	assertEqual(t, v, "0.25rem")
}

func TestThemePrefixAndUsed(t *testing.T) {
	th := makeTheme()
	th.Prefix = "tw"
	assertEqual(t, resolve(t, th, "red-500", "--color"), "var(--tw-color-red-500)")
	assertEqual(t, th.PrefixKey("--spacing"), "--tw-spacing")
	assertEqual(t, th.MarkUsedVariable("--tw-color-red-500"), true)
	assertEqual(t, th.Options("--color-red-500")&Used != 0, true)
	assertEqual(t, th.MarkUsedVariable("--tw-color-red-500"), false)
	assertEqual(t, th.MarkUsedVariable("--missing"), false)

	th = makeTheme()
	assertEqual(t, th.MarkUsedVariable(`--container-1\.5`), false)
	assertEqual(t, th.MarkUsedVariable("--container-1_5"), true)
	v, _ := th.ResolveThemeValue("--color-red-500", true)
	assertEqual(t, v, "red")
	v, _ = th.ResolveThemeValue("--color-red-500", false)
	assertEqual(t, v, "var(--color-red-500)")
	v, _ = th.ResolveThemeValue("--color-red-500/0.5", true)
	assertEqual(t, v, "color-mix(in oklab, red 50%, transparent)")
	v, _ = th.ResolveThemeValue("--color-red-500 / 100%", true)
	assertEqual(t, v, "red")
	_, ok := th.ResolveThemeValue("--missing", true)
	assertEqual(t, ok, false)
}
