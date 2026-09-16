package twill

import (
	"strings"
	"testing"
)

// designSystemFor builds a design system from a stylesheet without running
// the full compile pipeline.
func designSystemFor(t *testing.T, css string) *DesignSystem {
	t.Helper()
	if css == "" {
		css = `@import "twill";`
	}
	th, _, state := importAndCollect(t, css, nil)
	ds := BuildDesignSystem(th)
	ds.Important = state.Important
	return ds
}

// compileRaw compiles one raw candidate into nested CSS through the
// registries, without theme function substitution.
func compileRaw(ds *DesignSystem, raw string) string {
	var rules []Node
	for _, c := range ds.ParseCandidate(raw) {
		var results [][]Node
		if c.Kind == CandidateArbitrary {
			if value, ok := AsColor(c.ArbitraryValue, c.Modifier, ds.Theme); ok {
				results = append(results, []Node{Decl(c.Property, value)})
			}
		} else {
			var defs []*UtilityDefinition
			for _, d := range ds.Utilities.Get(c.Root) {
				if d.Kind == UtilityStatic && c.Kind == CandidateStatic || d.Kind == UtilityFunctional && c.Kind == CandidateFunctional {
					defs = append(defs, d)
				}
			}
			var ordered []*UtilityDefinition
			for _, d := range defs {
				if !d.IsFallback() {
					ordered = append(ordered, d)
				}
			}
			for _, d := range defs {
				if d.IsFallback() {
					ordered = append(ordered, d)
				}
			}
			for _, d := range ordered {
				nodes, status := d.Compile(c)
				if status == NotHandled {
					continue
				}
				if status == Invalid {
					if d.Types != nil {
						break
					}
					continue
				}
				results = append(results, nodes)
			}
		}
		for _, nodes := range results {
			rule := StyleRule("."+Escape(raw), nodes...)
			ok := true
			for _, v := range c.Variants {
				if !ApplyVariant(rule, v, ds.Variants, 0) {
					ok = false
					break
				}
			}
			if ok {
				rules = append(rules, rule)
			}
		}
	}
	return Serialize(rules)
}

func block(selector, body string) string {
	lines := strings.Split(strings.TrimRight(body, "\n"), "\n")
	for i, l := range lines {
		lines[i] = "  " + l
	}
	return selector + " {\n" + strings.Join(lines, "\n") + "\n}\n"
}
