// Runs every conformance case through the Go implementation and writes
// the output to out/go/<case>.css.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	twill "github.com/kt3k/twill/impl/go"
)

func loader(id, base string) (*twill.LoadedStylesheet, error) {
	builtin, err := twill.ResolveBuiltin(id, base)
	if err != nil || builtin != nil {
		return builtin, err
	}
	path := id
	if !filepath.IsAbs(path) {
		path = filepath.Join(base, id)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return &twill.LoadedStylesheet{Path: path, Base: filepath.Dir(path), Content: string(content)}, nil
}

func main() {
	here := os.Args[1]
	casesDir := filepath.Join(here, "cases")
	outDir := filepath.Join(here, "out", "go")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		panic(err)
	}
	entries, err := os.ReadDir(casesDir)
	if err != nil {
		panic(err)
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	for _, name := range names {
		dir := filepath.Join(casesDir, name)
		css, _ := os.ReadFile(filepath.Join(dir, "input.css"))
		raw, _ := os.ReadFile(filepath.Join(dir, "candidates.txt"))
		var candidates []string
		for _, l := range strings.Split(string(raw), "\n") {
			if l != "" {
				candidates = append(candidates, l)
			}
		}
		var output string
		compiler, err := twill.Compile(string(css), twill.CompileOptions{Base: dir, LoadStylesheet: loader})
		if err != nil {
			output = "ERROR: " + err.Error() + "\n"
		} else {
			output = compiler.Build(candidates)
			reversed := make([]string, len(candidates))
			for i, c := range candidates {
				reversed[len(candidates)-1-i] = c
			}
			again := compiler.Build(reversed)
			if again != output {
				output += "\n/* UNSTABLE: second build differs */\n" + again
			}
		}
		if err := os.WriteFile(filepath.Join(outDir, name+".css"), []byte(output), 0o644); err != nil {
			panic(err)
		}
	}
	fmt.Printf("go: %d cases\n", len(names))
}
