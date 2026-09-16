// Package cli implements the Twill command-line interface (SPEC §14).
package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	twill "github.com/kt3k/twill/impl/go"
)

// Usage is the help text.
const Usage = `Usage: twill [build] [options]

Options:
  -i, --input <path>     Entry stylesheet ("-" reads stdin; default: @import 'twill';)
  -o, --output <path>    Output file ("-" writes stdout; default: -)
  -w, --watch [always]   Rebuild on changes ("always" keeps watching after stdin closes)
      --poll [ms]        Poll for changes instead of using filesystem events (default: 250)
  -m, --minify           Optimize and minify the output
      --optimize         Optimize without minifying
      --cwd <dir>        Base directory (default: .)
      --silent           Suppress everything except errors
  -h, --help             Show this help
`

const defaultInput = "@import 'twill';\n"

// DefaultPollInterval is the --poll interval when none is given.
const DefaultPollInterval = 250 * time.Millisecond

// WatchInterval is how often watch mode checks the watched trees for
// changes. The standard library has no filesystem event API, so watch mode
// detects change batches by comparing modification times.
const WatchInterval = 100 * time.Millisecond

// WatchMode is the --watch setting.
type WatchMode int

const (
	// WatchOff disables watching.
	WatchOff WatchMode = iota
	// WatchOn watches until stdin reaches end of file.
	WatchOn
	// WatchAlways keeps watching after stdin closes.
	WatchAlways
)

// Options are the parsed command-line options.
type Options struct {
	// Input is the entry stylesheet; "" means the default input.
	Input  string
	Output string
	Watch  WatchMode
	// Poll is the polling interval; zero disables polling.
	Poll     time.Duration
	Minify   bool
	Optimize bool
	Cwd      string
	Silent   bool
	Help     bool
}

var digitsPattern = regexp.MustCompile(`^\d+$`)

// ParseArgs parses command-line arguments.
func ParseArgs(argv []string) (*Options, error) {
	o := &Options{Output: "-", Cwd: "."}
	value := func(i *int, name string) (string, error) {
		if *i+1 >= len(argv) {
			return "", fmt.Errorf("Missing value for %s", name)
		}
		*i++
		return argv[*i], nil
	}
	pollSet := false
	for i := 0; i < len(argv); i++ {
		arg := argv[i]
		name, inline, hasInline := arg, "", false
		if strings.HasPrefix(arg, "-") {
			if eq := strings.Index(arg, "="); eq != -1 {
				name, inline, hasInline = arg[:eq], arg[eq+1:], true
			}
		}
		take := func() (string, error) {
			if hasInline {
				return inline, nil
			}
			return value(&i, name)
		}
		var err error
		switch name {
		case "build":
			if strings.HasPrefix(arg, "-") {
				return nil, fmt.Errorf("Unknown option: %s", arg)
			}
		case "-i", "--input":
			o.Input, err = take()
		case "-o", "--output":
			o.Output, err = take()
		case "--cwd":
			o.Cwd, err = take()
		case "-w", "--watch":
			o.Watch = WatchOn
			if hasInline {
				if inline != "always" {
					return nil, fmt.Errorf("Invalid --watch value: %s", inline)
				}
				o.Watch = WatchAlways
			} else if i+1 < len(argv) && argv[i+1] == "always" {
				o.Watch = WatchAlways
				i++
			}
		case "--poll":
			pollSet = true
			ms := ""
			if hasInline {
				if !digitsPattern.MatchString(inline) {
					return nil, fmt.Errorf("Invalid --poll interval: %s", inline)
				}
				ms = inline
			} else if i+1 < len(argv) && digitsPattern.MatchString(argv[i+1]) {
				ms = argv[i+1]
				i++
			}
			if ms == "" {
				o.Poll = DefaultPollInterval
			} else {
				n, _ := strconv.Atoi(ms)
				o.Poll = time.Duration(n) * time.Millisecond
			}
		case "-m", "--minify":
			o.Minify = true
		case "--optimize":
			o.Optimize = true
		case "--silent":
			o.Silent = true
		case "-h", "--help":
			o.Help = true
		default:
			if strings.HasPrefix(arg, "-") {
				return nil, fmt.Errorf("Unknown option: %s", arg)
			}
		}
		if err != nil {
			return nil, err
		}
	}
	if pollSet && o.Poll <= 0 {
		return nil, errors.New("The --poll interval must be a positive number of milliseconds.")
	}
	return o, nil
}

// NewLoader creates a loader resolving relative ids against the filesystem
// and recording every loaded path as a full-rebuild path.
func NewLoader(fullRebuildPaths map[string]bool) twill.StylesheetLoader {
	return func(id, base string) (*twill.LoadedStylesheet, error) {
		builtin, err := twill.ResolveBuiltin(id, base)
		if err != nil || builtin != nil {
			return builtin, err
		}
		if twill.IsBuiltinPath(base) {
			return nil, twill.Errorf("Cannot import `%s` from a built-in stylesheet.", id)
		}
		path := id
		if !filepath.IsAbs(path) {
			path = filepath.Join(base, id)
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return nil, twill.Errorf("Cannot read `%s`: %v", path, err)
		}
		fullRebuildPaths[path] = true
		return &twill.LoadedStylesheet{Path: path, Base: filepath.Dir(path), Content: string(content)}, nil
	}
}

// AssembleSources assembles the source set (SPEC §13.1). inputPath and
// executable may be empty.
func AssembleSources(compiler *twill.Compiler, cwd, inputPath, executable string) []twill.SourceEntry {
	var sources []twill.SourceEntry
	switch {
	case compiler.Root != nil && compiler.Root.None:
		// Auto-detection disabled.
	case compiler.Root == nil:
		sources = append(sources, twill.SourceEntry{Base: cwd, Pattern: "**/*"})
	default:
		sources = append(sources, twill.SourceEntry{Base: compiler.Root.Base, Pattern: compiler.Root.Pattern})
	}
	sources = append(sources, compiler.Sources...)
	if executable != "" {
		sources = append(sources, twill.SourceEntry{Base: filepath.Dir(executable), Pattern: filepath.Base(executable), Negated: true})
	}
	if inputPath != "" {
		sources = append(sources, twill.SourceEntry{Base: filepath.Dir(inputPath), Pattern: filepath.Base(inputPath)})
	}
	return sources
}

// MinifyCSS minifies CSS by re-serializing it compactly.
func MinifyCSS(css string) (string, error) {
	nodes, err := twill.Parse(css)
	if err != nil {
		return "", err
	}
	return twill.SerializeCompact(nodes), nil
}

func formatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	return fmt.Sprintf("%.2fs", d.Seconds())
}

type state struct {
	compiler         *twill.Compiler
	scanner          *twill.Scanner
	fullRebuildPaths map[string]bool
}

type runner struct {
	options    *Options
	stdin      io.Reader
	stdout     io.Writer
	stderr     io.Writer
	cwd        string
	inputPath  string
	outputPath string
	executable string
	state      *state

	previousCSS     string
	previousWritten string
	hasPrevious     bool
	lastStdout      string
	hasStdout       bool
}

// Main runs the CLI and returns the exit code. The context cancels watch
// and polling modes.
func Main(ctx context.Context, argv []string, stdin io.Reader, stdout, stderr io.Writer) int {
	options, err := ParseArgs(argv)
	if err != nil {
		fmt.Fprintf(stderr, "%s\n\n%s", err.Error(), Usage)
		return 1
	}
	if options.Help {
		fmt.Fprint(stderr, Usage)
		return 0
	}
	if len(argv) == 0 && isTerminal(stdin) {
		fmt.Fprint(stderr, Usage)
		return 0
	}

	cwd, err := filepath.Abs(options.Cwd)
	if err != nil {
		fmt.Fprintf(stderr, "%v\n", err)
		return 1
	}
	r := &runner{options: options, stdin: stdin, stdout: stdout, stderr: stderr, cwd: cwd}
	if options.Input != "" && options.Input != "-" {
		r.inputPath = resolveIn(cwd, options.Input)
	}
	if options.Output != "-" {
		r.outputPath = resolveIn(cwd, options.Output)
	}
	if r.inputPath != "" {
		info, err := os.Stat(r.inputPath)
		if err != nil || !info.Mode().IsRegular() {
			fmt.Fprintf(stderr, "Specified input file `%s` does not exist.\n", options.Input)
			return 1
		}
	}
	if r.inputPath != "" && r.outputPath != "" && r.inputPath == r.outputPath {
		fmt.Fprint(stderr, "Specified input file and output file are identical.\n")
		return 1
	}
	if !options.Silent {
		fmt.Fprint(stderr, "twill\n\n")
	}
	if exe, err := os.Executable(); err == nil {
		r.executable = exe
	}

	if err := r.timed(func() error {
		st, err := r.createState()
		if err != nil {
			return err
		}
		r.state = st
		return r.write(st.compiler.Build(st.scanner.Scan()))
	}); err != nil {
		fmt.Fprintf(stderr, "%s\n", err.Error())
		return 1
	}

	if options.Watch == WatchOff && options.Poll == 0 {
		return 0
	}
	if options.Poll > 0 {
		r.poll(ctx)
		return 0
	}
	r.watch(ctx)
	return 0
}

func resolveIn(cwd, path string) string {
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}
	return filepath.Join(cwd, path)
}

func isTerminal(r io.Reader) bool {
	f, ok := r.(*os.File)
	if !ok {
		return false
	}
	return isTerminalFile(f)
}

func (r *runner) readInput() (string, error) {
	if r.inputPath != "" {
		content, err := os.ReadFile(r.inputPath)
		return string(content), err
	}
	if r.options.Input == "-" {
		content, err := io.ReadAll(r.stdin)
		return string(content), err
	}
	return defaultInput, nil
}

func (r *runner) createState() (*state, error) {
	css, err := r.readInput()
	if err != nil {
		return nil, err
	}
	fullRebuildPaths := map[string]bool{}
	base := r.cwd
	if r.inputPath != "" {
		fullRebuildPaths[r.inputPath] = true
		base = filepath.Dir(r.inputPath)
	}
	compiler, err := twill.Compile(css, twill.CompileOptions{Base: base, LoadStylesheet: NewLoader(fullRebuildPaths)})
	if err != nil {
		return nil, err
	}
	sources := AssembleSources(compiler, r.cwd, r.inputPath, r.executable)
	return &state{compiler: compiler, scanner: twill.NewScanner(sources), fullRebuildPaths: fullRebuildPaths}, nil
}

// write writes the CSS to the output (SPEC §14.5).
func (r *runner) write(css string) error {
	output := css
	if r.options.Minify || r.options.Optimize {
		if r.hasPrevious && css == r.previousCSS {
			output = r.previousWritten
		} else {
			minified, err := MinifyCSS(css)
			if err != nil {
				return err
			}
			output = minified
		}
	}
	r.previousCSS, r.previousWritten, r.hasPrevious = css, output, true
	if r.outputPath != "" {
		if err := os.MkdirAll(filepath.Dir(r.outputPath), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(r.outputPath, []byte(output), 0o644); err != nil {
			return err
		}
	} else if !r.hasStdout || output != r.lastStdout {
		if _, err := io.WriteString(r.stdout, output); err != nil {
			return err
		}
	}
	r.lastStdout, r.hasStdout = output, true
	return nil
}

func (r *runner) timed(fn func() error) error {
	start := time.Now()
	if err := fn(); err != nil {
		return err
	}
	if !r.options.Silent {
		fmt.Fprintf(r.stderr, "Done in %s\n", formatDuration(time.Since(start)))
	}
	return nil
}

func (r *runner) report(err error) {
	fmt.Fprintf(r.stderr, "%s\n", err.Error())
}

// fullRebuild re-reads the input and recreates the compiler and scanner.
func (r *runner) fullRebuild() error {
	previous := r.state.fullRebuildPaths
	next, err := r.createState()
	if err != nil {
		// Keep the previous dependency list so a later change to a deleted
		// dependency still triggers a rebuild.
		r.state.fullRebuildPaths = previous
		return err
	}
	candidates := next.scanner.Scan()
	r.state = next
	return r.write(next.compiler.Build(candidates))
}

// poll implements polling watch mode (SPEC §14.4).
func (r *runner) poll(ctx context.Context) {
	ticker := time.NewTicker(r.options.Poll)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
		if err := r.pollTick(); err != nil {
			r.report(err)
		}
	}
}

func (r *runner) pollTick() error {
	candidates := r.state.scanner.Scan()
	var files []string
	for _, f := range r.state.scanner.ScannedFiles {
		if f != r.outputPath {
			files = append(files, f)
		}
	}
	if len(files) == 0 {
		return nil
	}
	for _, f := range files {
		if r.state.fullRebuildPaths[f] {
			return r.timed(r.fullRebuild)
		}
	}
	if len(candidates) == 0 {
		return nil
	}
	return r.timed(func() error { return r.write(r.state.compiler.Build(candidates)) })
}

// watch implements event-driven watch mode (SPEC §14.3). Change batches
// are detected by comparing modification times across the watched trees.
func (r *runner) watch(ctx context.Context) {
	stdinDone := make(chan struct{})
	go func() {
		if r.stdin != nil {
			_, _ = io.Copy(io.Discard, r.stdin)
		}
		close(stdinDone)
	}()
	if r.options.Watch != WatchAlways {
		select {
		case <-stdinDone:
			return
		default:
		}
	}

	previous := r.snapshot()
	ticker := time.NewTicker(WatchInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-stdinDone:
			if r.options.Watch != WatchAlways {
				return
			}
			stdinDone = nil
		case <-ticker.C:
		}
		current := r.snapshot()
		changed := diffSnapshots(previous, current)
		previous = current
		if len(changed) == 0 {
			continue
		}
		if err := r.handle(changed); err != nil {
			r.report(err)
		}
		// The next snapshot is compared against `current`, so files written
		// while the batch was handled (and files added to the watched set by
		// a full rebuild) are picked up on the next tick.
	}
}

func (r *runner) handle(paths []string) error {
	var relevant []string
	for _, p := range paths {
		if p != r.outputPath {
			relevant = append(relevant, p)
		}
	}
	if len(relevant) == 0 {
		return nil
	}
	for _, p := range relevant {
		if r.state.fullRebuildPaths[p] {
			return r.timed(r.fullRebuild)
		}
	}
	fresh := r.state.scanner.ScanFiles(relevant)
	if len(fresh) == 0 {
		return nil
	}
	return r.timed(func() error { return r.write(r.state.compiler.Build(fresh)) })
}

// snapshot records the modification time of every watched file.
func (r *runner) snapshot() map[string]int64 {
	snap := map[string]int64{}
	record := func(path string) {
		info, err := os.Stat(path)
		if err != nil {
			snap[path] = -1
			return
		}
		snap[path] = info.ModTime().UnixNano()
	}
	for _, path := range r.state.scanner.Files() {
		record(path)
	}
	for path := range r.state.fullRebuildPaths {
		if !twill.IsBuiltinPath(path) {
			record(path)
		}
	}
	return snap
}

func diffSnapshots(previous, current map[string]int64) []string {
	var changed []string
	for path, mtime := range current {
		if before, ok := previous[path]; !ok || before != mtime {
			changed = append(changed, path)
		}
	}
	for path := range previous {
		if _, ok := current[path]; !ok {
			changed = append(changed, path)
		}
	}
	sort.Strings(changed)
	return changed
}
