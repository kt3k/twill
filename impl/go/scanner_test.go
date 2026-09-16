package twill

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func scanFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeFile(t, filepath.Join(root, ".gitignore"), "ignored/\n*.tmp\n")
	writeFile(t, filepath.Join(root, "index.html"), `<div class="flex p-4"></div>`)
	writeFile(t, filepath.Join(root, "src", "app.js"), `el.className = "hover:underline";`)
	writeFile(t, filepath.Join(root, "src", "sub", "page.html"), `<p class="md:grid">`)
	writeFile(t, filepath.Join(root, "src", "notes.tmp"), `tmp-only`)
	writeFile(t, filepath.Join(root, "styles.scss"), `.scss-only {}`)
	writeFile(t, filepath.Join(root, "logo.png"), `png-only`)
	writeFile(t, filepath.Join(root, "package-lock.json"), `lock-only`)
	writeFile(t, filepath.Join(root, ".env"), `env-only`)
	writeFile(t, filepath.Join(root, "node_modules", "pkg", "x.js"), `nm-only`)
	writeFile(t, filepath.Join(root, "vendor", "v.html"), `vendor-only`)
	writeFile(t, filepath.Join(root, "ignored", "i.html"), `ignored-only`)
	return root
}

func contains(list []string, item string) bool {
	for _, l := range list {
		if l == item {
			return true
		}
	}
	return false
}

func TestScannerAutoDetectionIgnoreRules(t *testing.T) {
	root := scanFixture(t)
	scanner := NewScanner([]SourceEntry{{Base: root, Pattern: "**/*"}})
	candidates := scanner.Scan()
	for _, expected := range []string{"flex", "p-4", "hover:underline", "md:grid", "vendor-only"} {
		if !contains(candidates, expected) {
			t.Errorf("expected %q in %v", expected, candidates)
		}
	}
	for _, unexpected := range []string{"tmp-only", "scss-only", "png-only", "lock-only", "env-only", "nm-only", "ignored-only"} {
		if contains(candidates, unexpected) {
			t.Errorf("unexpected %q in %v", unexpected, candidates)
		}
	}
	assertEqual(t, len(scanner.ScannedFiles), 4)
}

func TestScannerExplicitGlobsAndNegatedSources(t *testing.T) {
	root := scanFixture(t)
	scanner := NewScanner([]SourceEntry{
		{Base: root, Pattern: "**/*"},
		{Base: root, Pattern: "node_modules/**/*.js"},
		{Base: root, Pattern: "./ignored/*.html"},
		{Base: root, Pattern: "vendor", Negated: true},
	})
	candidates := scanner.Scan()
	assertEqual(t, contains(candidates, "nm-only"), true)
	assertEqual(t, contains(candidates, "ignored-only"), true)
	assertEqual(t, contains(candidates, "vendor-only"), false)
	assertEqual(t, contains(candidates, "flex"), true)
}

func TestScannerBraceGlobs(t *testing.T) {
	root := scanFixture(t)
	scanner := NewScanner([]SourceEntry{{Base: root, Pattern: "src/**/*.{js,html}"}})
	candidates := scanner.Scan()
	assertEqual(t, contains(candidates, "hover:underline"), true)
	assertEqual(t, contains(candidates, "md:grid"), true)
	assertEqual(t, contains(candidates, "tmp-only"), false)
	assertEqual(t, contains(candidates, "flex"), false)
}

func TestScannerDirectoryAndFileSources(t *testing.T) {
	root := scanFixture(t)
	scanner := NewScanner([]SourceEntry{
		{Base: root, Pattern: "src"},
		{Base: root, Pattern: "./index.html"},
		{Base: root, Pattern: "src/sub/*.html", Negated: true},
	})
	candidates := scanner.Scan()
	assertEqual(t, contains(candidates, "hover:underline"), true)
	assertEqual(t, contains(candidates, "flex"), true)
	assertEqual(t, contains(candidates, "md:grid"), false)
	assertEqual(t, contains(candidates, "vendor-only"), false)
}

func TestScannerIncrementalScansAndScanFiles(t *testing.T) {
	root := scanFixture(t)
	scanner := NewScanner([]SourceEntry{{Base: root, Pattern: "**/*"}})
	scanner.Scan()
	again := scanner.Scan()
	assertEqual(t, scanner.ScannedFiles, []string{})
	assertEqual(t, contains(again, "flex"), true)

	changed := filepath.Join(root, "src", "app.js")
	writeFile(t, changed, `el.className = "hover:underline new-one";`)
	future := time.Now().Add(5 * time.Second)
	if err := os.Chtimes(changed, future, future); err != nil {
		t.Fatal(err)
	}
	third := scanner.Scan()
	assertEqual(t, scanner.ScannedFiles, []string{changed})
	assertEqual(t, contains(third, "new-one"), true)

	fresh := filepath.Join(root, "fresh.html")
	writeFile(t, fresh, `<i class="fresh-one flex">`)
	assertEqual(t, scanner.ScanFiles([]string{fresh, filepath.Join(root, "missing.html")}), []string{"fresh-one"})
	assertEqual(t, scanner.ScanFiles([]string{fresh}), []string{})
	assertEqual(t, contains(scanner.Candidates(), "fresh-one"), true)
}

func TestGitignoreRules(t *testing.T) {
	g := ParseGitignore("# comment\n\nbuild/\n*.log\n!keep.log\n/root.txt\ndocs/*.md\nfoo?\n**/deep\n")
	assertEqual(t, g.Size(), 7)
	assertEqual(t, g.Ignores("build", true), true)
	assertEqual(t, g.Ignores("build", false), false)
	assertEqual(t, g.Ignores("a/b/build", true), true)
	assertEqual(t, g.Ignores("x.log", false), true)
	assertEqual(t, g.Ignores("sub/x.log", false), true)
	assertEqual(t, g.Ignores("keep.log", false), false)
	assertEqual(t, g.Ignores("root.txt", false), true)
	assertEqual(t, g.Ignores("sub/root.txt", false), false)
	assertEqual(t, g.Ignores("docs/a.md", false), true)
	assertEqual(t, g.Ignores("docs/sub/a.md", false), false)
	assertEqual(t, g.Ignores("food", false), true)
	assertEqual(t, g.Ignores("fooddd", false), false)
	assertEqual(t, g.Ignores("a/b/deep", false), true)
	assertEqual(t, g.Ignores("deep", false), true)
}
