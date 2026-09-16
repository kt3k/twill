package cli

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	twill "github.com/kt3k/twill/impl/go"
)

type syncBuffer struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (s *syncBuffer) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.Write(p)
}

func (s *syncBuffer) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.String()
}

type result struct {
	code   int
	stdout string
	stderr string
}

func runCLI(t *testing.T, args []string, cwd string, stdin io.Reader) result {
	t.Helper()
	if cwd != "" {
		args = append([]string{"--cwd", cwd}, args...)
	}
	var stdout, stderr syncBuffer
	code := Main(context.Background(), args, stdin, &stdout, &stderr)
	return result{code, stdout.String(), stderr.String()}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func project(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "input.css"), "@import \"twill/theme.css\";\n@source not \"dist\";\n@twill utilities;\n")
	writeFile(t, filepath.Join(root, "index.html"), `<div class="flex p-4 hover:underline">`)
	writeFile(t, filepath.Join(root, "dist", "old.html"), `<div class="hidden">`)
	return root
}

func assertEqual(t *testing.T, got, want any) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got:\n%#v\nwant:\n%#v", got, want)
	}
}

func includes(t *testing.T, output, want string) {
	t.Helper()
	if !strings.Contains(output, want) {
		t.Fatalf("expected output to contain %q:\n%s", want, output)
	}
}

func TestParseArgs(t *testing.T) {
	defaults, err := ParseArgs(nil)
	if err != nil {
		t.Fatal(err)
	}
	assertEqual(t, defaults, &Options{Output: "-", Cwd: "."})

	full, err := ParseArgs([]string{"build", "-i", "a.css", "-o", "b.css", "--watch", "always", "--poll", "500", "-m", "--optimize", "--cwd", "x", "--silent"})
	if err != nil {
		t.Fatal(err)
	}
	assertEqual(t, full, &Options{Input: "a.css", Output: "b.css", Watch: WatchAlways, Poll: 500 * time.Millisecond, Minify: true, Optimize: true, Cwd: "x", Silent: true})

	watch, _ := ParseArgs([]string{"-w", "-i", "a.css"})
	assertEqual(t, watch.Watch, WatchOn)
	assertEqual(t, watch.Input, "a.css")
	always, _ := ParseArgs([]string{"--watch=always"})
	assertEqual(t, always.Watch, WatchAlways)
	poll, _ := ParseArgs([]string{"--poll"})
	assertEqual(t, poll.Poll, DefaultPollInterval)
	poll100, _ := ParseArgs([]string{"--poll=100"})
	assertEqual(t, poll100.Poll, 100*time.Millisecond)
	inline, _ := ParseArgs([]string{"--input=in.css", "--output=out.css"})
	assertEqual(t, inline.Input, "in.css")
	assertEqual(t, inline.Output, "out.css")
	help, _ := ParseArgs([]string{"-h"})
	assertEqual(t, help.Help, true)

	if _, err := ParseArgs([]string{"--poll=0"}); err == nil || !strings.Contains(err.Error(), "positive") {
		t.Fatalf("expected a positive interval error, got %v", err)
	}
	if _, err := ParseArgs([]string{"--nope"}); err == nil || !strings.Contains(err.Error(), "Unknown option") {
		t.Fatalf("expected an unknown option error, got %v", err)
	}
	if _, err := ParseArgs([]string{"-i"}); err == nil {
		t.Fatal("expected a missing value error")
	}
}

func TestAssembleSources(t *testing.T) {
	compiler := &twill.Compiler{Sources: []twill.SourceEntry{{Base: "/p", Pattern: "./src/**/*"}}}
	assertEqual(t, AssembleSources(compiler, "/p", "/p/in.css", "/bin/twill"), []twill.SourceEntry{
		{Base: "/p", Pattern: "**/*"},
		{Base: "/p", Pattern: "./src/**/*"},
		{Base: "/bin", Pattern: "twill", Negated: true},
		{Base: "/p", Pattern: "in.css"},
	})
	none := &twill.Compiler{Root: &twill.SourceRoot{None: true}}
	assertEqual(t, len(AssembleSources(none, "/p", "", "")), 0)
	rooted := &twill.Compiler{Root: &twill.SourceRoot{Base: "/p", Pattern: "../app"}}
	assertEqual(t, AssembleSources(rooted, "/p", "", ""), []twill.SourceEntry{{Base: "/p", Pattern: "../app"}})
}

func TestMinifyCSS(t *testing.T) {
	out, err := MinifyCSS(".a {\n  color: red !important;\n}\n@media (x) {\n  .b {\n    y: z;\n  }\n}\n")
	if err != nil {
		t.Fatal(err)
	}
	assertEqual(t, out, ".a{color:red!important;}@media (x){.b{y:z;}}")
}

func TestCLISingleBuild(t *testing.T) {
	root := project(t)
	r := runCLI(t, []string{"-i", "input.css"}, root, nil)
	assertEqual(t, r.code, 0)
	includes(t, r.stdout, ".flex {\n  display: flex;\n}")
	includes(t, r.stdout, ".p-4 {\n  padding: calc(var(--spacing) * 4);\n}")
	includes(t, r.stdout, ".hover\\:underline:hover")
	if strings.Contains(r.stdout, ".hidden") {
		t.Fatal("expected the excluded source to be skipped")
	}
	includes(t, r.stderr, "twill")
	includes(t, r.stderr, "Done in")

	toFile := runCLI(t, []string{"-i", "input.css", "-o", "out/build.css", "--silent"}, root, nil)
	assertEqual(t, toFile.code, 0)
	assertEqual(t, toFile.stdout, "")
	assertEqual(t, toFile.stderr, "")
	written, err := os.ReadFile(filepath.Join(root, "out", "build.css"))
	if err != nil {
		t.Fatal(err)
	}
	includes(t, string(written), ".flex {")

	minified := runCLI(t, []string{"-i", "input.css", "--minify", "--silent"}, root, nil)
	assertEqual(t, minified.code, 0)
	includes(t, minified.stdout, ".flex{display:flex;}")
}

func TestCLIStdinAndDefaultInput(t *testing.T) {
	root := project(t)
	r := runCLI(t, []string{"-i", "-", "--silent"}, root, strings.NewReader("@import \"twill/theme.css\";\n@twill utilities;"))
	assertEqual(t, r.code, 0)
	includes(t, r.stdout, ".flex {")

	defaulted := runCLI(t, []string{"--silent"}, root, strings.NewReader(""))
	assertEqual(t, defaulted.code, 0)
	includes(t, defaulted.stdout, "@layer theme, base, components, utilities;")
	includes(t, defaulted.stdout, ".flex {")
}

func TestCLIErrors(t *testing.T) {
	root := project(t)
	missing := runCLI(t, []string{"-i", "nope.css"}, root, nil)
	assertEqual(t, missing.code, 1)
	includes(t, missing.stderr, "does not exist")

	same := runCLI(t, []string{"-i", "input.css", "-o", "input.css"}, root, nil)
	assertEqual(t, same.code, 1)
	includes(t, same.stderr, "identical")

	writeFile(t, filepath.Join(root, "bad.css"), ".a { color: red;")
	bad := runCLI(t, []string{"-i", "bad.css", "--silent"}, root, nil)
	assertEqual(t, bad.code, 1)
	includes(t, bad.stderr, "Missing closing")

	unknown := runCLI(t, []string{"--nope"}, "", nil)
	assertEqual(t, unknown.code, 1)
	includes(t, unknown.stderr, "Unknown option")
	includes(t, unknown.stderr, "Usage: twill")
}

func TestCLIHelp(t *testing.T) {
	r := runCLI(t, []string{"--help"}, "", nil)
	assertEqual(t, r.code, 0)
	includes(t, r.stderr, "Usage: twill")
}

func waitFor(t *testing.T, check func() bool) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if check() {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("timed out waiting for the watcher")
}

func fileIncludes(path, text string) func() bool {
	return func() bool {
		content, err := os.ReadFile(path)
		return err == nil && strings.Contains(string(content), text)
	}
}

func TestCLIPollingRebuilds(t *testing.T) {
	root := project(t)
	output := filepath.Join(root, "out.css")
	ctx, cancel := context.WithCancel(context.Background())
	var stderr syncBuffer
	done := make(chan int)
	go func() {
		done <- Main(ctx, []string{"--cwd", root, "-i", "input.css", "-o", "out.css", "--poll", "50", "--silent"}, nil, io.Discard, &stderr)
	}()
	defer func() {
		cancel()
		<-done
	}()
	waitFor(t, fileIncludes(output, ".flex {"))
	// A source change is picked up incrementally.
	writeFile(t, filepath.Join(root, "page.html"), `<i class="hidden">`)
	waitFor(t, fileIncludes(output, ".hidden {"))
	// A stylesheet change triggers a full rebuild.
	writeFile(t, filepath.Join(root, "input.css"), "@import \"twill/theme.css\";\n@theme { --color-brand: blue; }\n@twill utilities;\n")
	writeFile(t, filepath.Join(root, "page.html"), `<i class="hidden bg-brand">`)
	waitFor(t, fileIncludes(output, ".bg-brand {"))
	if stderr.String() != "" {
		t.Fatalf("unexpected stderr: %s", stderr.String())
	}
}

func TestCLIWatchRebuildsAndExitsOnStdinEOF(t *testing.T) {
	root := project(t)
	output := filepath.Join(root, "out.css")
	stdinReader, stdinWriter := io.Pipe()
	var stderr syncBuffer
	done := make(chan int)
	go func() {
		done <- Main(context.Background(), []string{"--cwd", root, "-i", "input.css", "-o", "out.css", "--watch", "--silent"}, stdinReader, io.Discard, &stderr)
	}()
	waitFor(t, fileIncludes(output, ".flex {"))
	writeFile(t, filepath.Join(root, "page.html"), `<i class="hidden">`)
	waitFor(t, fileIncludes(output, ".hidden {"))
	writeFile(t, filepath.Join(root, "input.css"), "@import \"twill/theme.css\";\n@theme static { --color-brand: blue; }\n@twill utilities;\n")
	waitFor(t, fileIncludes(output, "--color-brand"))
	// A broken stylesheet reports an error and keeps watching.
	writeFile(t, filepath.Join(root, "input.css"), ".a { color: red;")
	waitFor(t, func() bool { return strings.Contains(stderr.String(), "Missing closing") })
	writeFile(t, filepath.Join(root, "input.css"), "@import \"twill/theme.css\";\n@theme static { --color-other: red; }\n@twill utilities;\n")
	waitFor(t, fileIncludes(output, "--color-other"))
	// A deleted source keeps the watcher alive.
	if err := os.Remove(filepath.Join(root, "page.html")); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(root, "other.html"), `<i class="underline">`)
	waitFor(t, fileIncludes(output, ".underline {"))

	stdinWriter.Close()
	select {
	case code := <-done:
		assertEqual(t, code, 0)
	case <-time.After(5 * time.Second):
		t.Fatal("watch mode did not exit on stdin EOF")
	}
}

func TestCLIWatchAlwaysIgnoresStdinEOF(t *testing.T) {
	root := project(t)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan int)
	go func() {
		done <- Main(ctx, []string{"--cwd", root, "-i", "input.css", "-o", "out.css", "--watch=always", "--silent"}, strings.NewReader(""), io.Discard, io.Discard)
	}()
	select {
	case <-done:
		t.Fatal("watch=always exited on stdin EOF")
	case <-time.After(500 * time.Millisecond):
	}
	cancel()
	assertEqual(t, <-done, 0)
}
