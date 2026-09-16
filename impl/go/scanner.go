package twill

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Source scanning: file discovery, candidate extraction, and incremental
// scans (SPEC §13).

var ignoredDirectories = map[string]bool{
	".git": true, ".hg": true, ".jj": true, ".next": true, ".parcel-cache": true,
	".pnpm-store": true, ".svelte-kit": true, ".svn": true, ".turbo": true,
	".venv": true, ".vercel": true, ".yarn": true, "__pycache__": true,
	"node_modules": true, "venv": true,
}

var ignoredExtensions = map[string]bool{
	"less": true, "lock": true, "sass": true, "scss": true, "styl": true, "log": true,
	// Images
	"png": true, "jpg": true, "jpeg": true, "gif": true, "webp": true, "avif": true,
	"ico": true, "bmp": true, "tif": true, "tiff": true, "heic": true, "psd": true,
	// Audio
	"mp3": true, "wav": true, "ogg": true, "flac": true, "aac": true, "m4a": true,
	// Video
	"mp4": true, "webm": true, "mov": true, "avi": true, "mkv": true, "m4v": true,
	// Archives
	"zip": true, "gz": true, "tar": true, "tgz": true, "bz2": true, "xz": true,
	"7z": true, "rar": true,
	// Fonts
	"woff": true, "woff2": true, "ttf": true, "otf": true, "eot": true,
	// Executables and binaries
	"exe": true, "dll": true, "so": true, "dylib": true, "bin": true, "wasm": true,
	"pdf": true, "class": true, "jar": true, "pyc": true, "o": true, "a": true,
}

var ignoredFiles = map[string]bool{
	"package-lock.json": true, "pnpm-lock.yaml": true, "bun.lockb": true,
	".gitignore": true, ".env": true,
}

func isIgnoredFileName(name string) bool {
	if ignoredFiles[name] || strings.HasPrefix(name, ".env.") {
		return true
	}
	if dot := strings.LastIndex(name, "."); dot > 0 {
		if ignoredExtensions[strings.ToLower(name[dot+1:])] {
			return true
		}
	}
	return false
}

// IsGlob reports whether pattern contains glob metacharacters.
func IsGlob(pattern string) bool { return strings.ContainsAny(pattern, "*?[{") }

// GlobToRegexp converts an absolute posix glob (`**`, `*`, `?`, `{a,b}`)
// into a regular expression matching whole paths.
func GlobToRegexp(glob string) *regexp.Regexp {
	var b strings.Builder
	b.WriteString("^")
	braces := 0
	for i := 0; i < len(glob); i++ {
		c := glob[i]
		switch {
		case c == '*':
			if i+1 < len(glob) && glob[i+1] == '*' {
				i++
				if i+1 < len(glob) && glob[i+1] == '/' {
					i++
					b.WriteString("(?:.*/)?")
				} else {
					b.WriteString(".*")
				}
			} else {
				b.WriteString("[^/]*")
			}
		case c == '?':
			b.WriteString("[^/]")
		case c == '{':
			braces++
			b.WriteString("(?:")
		case c == '}' && braces > 0:
			braces--
			b.WriteString(")")
		case c == ',' && braces > 0:
			b.WriteString("|")
		case c == '\\' && i+1 < len(glob):
			i++
			b.WriteString(regexp.QuoteMeta(string(glob[i])))
		default:
			b.WriteString(regexp.QuoteMeta(string(c)))
		}
	}
	b.WriteString("$")
	return regexp.MustCompile(b.String())
}

// exclusion is a compiled exclusion from a negated source.
type exclusion struct {
	// Every path at or under this directory is excluded.
	directory string
	// Or: files matching this pattern (absolute, posix).
	pattern *regexp.Regexp
}

type scopedIgnore struct {
	directory string
	matcher   *Gitignore
}

func isDirectoryPath(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func isFilePath(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
}

func absPath(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		return filepath.Clean(path)
	}
	return abs
}

// Scanner enumerates source files from globs and auto-detection rules,
// extracts candidates, and tracks modification times for incremental scans.
type Scanner struct {
	sources    []SourceEntry
	candidates *candidateSet
	mtimes     map[string]int64
	exclusions []exclusion
	compiled   bool

	// ScannedFiles are the files read by the most recent Scan.
	ScannedFiles []string
}

// NewScanner creates a scanner for the given sources.
func NewScanner(sources []SourceEntry) *Scanner {
	return &Scanner{sources: sources, candidates: newCandidateSet(), mtimes: map[string]int64{}}
}

// Sources returns the sources this scanner was created with.
func (s *Scanner) Sources() []SourceEntry { return s.sources }

// Bases returns the distinct absolute base directories of the positive sources.
func (s *Scanner) Bases() []string {
	seen := map[string]bool{}
	var bases []string
	for _, source := range s.sources {
		if source.Negated {
			continue
		}
		base := absPath(source.Base)
		if !seen[base] {
			seen[base] = true
			bases = append(bases, base)
		}
	}
	return bases
}

func (s *Scanner) exclusionsFor() []exclusion {
	if s.compiled {
		return s.exclusions
	}
	for _, source := range s.sources {
		if !source.Negated {
			continue
		}
		target := absPath(filepath.Join(source.Base, source.Pattern))
		if !IsGlob(source.Pattern) {
			if isDirectoryPath(target) {
				s.exclusions = append(s.exclusions, exclusion{directory: target})
				continue
			}
			s.exclusions = append(s.exclusions, exclusion{pattern: regexp.MustCompile("^" + regexp.QuoteMeta(filepath.ToSlash(target)) + "$")})
			continue
		}
		s.exclusions = append(s.exclusions, exclusion{pattern: GlobToRegexp(filepath.ToSlash(target))})
	}
	s.compiled = true
	return s.exclusions
}

func (s *Scanner) isExcluded(path string, exclusions []exclusion) bool {
	posix := filepath.ToSlash(path)
	for _, e := range exclusions {
		if e.directory != "" {
			if path == e.directory || strings.HasPrefix(path, e.directory+string(filepath.Separator)) {
				return true
			}
		} else if e.pattern != nil && e.pattern.MatchString(posix) {
			return true
		}
	}
	return false
}

type fileSet struct {
	seen map[string]bool
	list []string
}

func (f *fileSet) add(path string) {
	if f.seen == nil {
		f.seen = map[string]bool{}
	}
	if !f.seen[path] {
		f.seen[path] = true
		f.list = append(f.list, path)
	}
}

// Files enumerates every file covered by the sources.
func (s *Scanner) Files() []string {
	exclusions := s.exclusionsFor()
	found := &fileSet{}
	for _, source := range s.sources {
		if source.Negated {
			continue
		}
		base := absPath(source.Base)
		if source.Pattern == "**/*" {
			s.walk(base, nil, exclusions, found)
			continue
		}
		target := absPath(filepath.Join(base, source.Pattern))
		if !IsGlob(source.Pattern) {
			if isDirectoryPath(target) {
				s.walk(target, nil, exclusions, found)
			} else if isFilePath(target) && !s.isExcluded(target, exclusions) {
				found.add(target)
			}
			continue
		}
		s.glob(target, exclusions, found)
	}
	return found.list
}

func (s *Scanner) walk(directory string, ignores []scopedIgnore, exclusions []exclusion, found *fileSet) {
	scoped := ignores
	if content, err := os.ReadFile(filepath.Join(directory, ".gitignore")); err == nil {
		matcher := ParseGitignore(string(content))
		if matcher.Size() > 0 {
			scoped = append(append([]scopedIgnore{}, ignores...), scopedIgnore{directory, matcher})
		}
	}

	entries, err := os.ReadDir(directory)
	if err != nil {
		return
	}
	for _, entry := range entries {
		path := filepath.Join(directory, entry.Name())
		isDir := entry.IsDir()
		if entry.Type()&os.ModeSymlink != 0 {
			isDir = isDirectoryPath(path)
		}
		if isDir {
			if ignoredDirectories[entry.Name()] || s.isGitignored(path, true, scoped) || s.isExcluded(path, exclusions) {
				continue
			}
			s.walk(path, scoped, exclusions, found)
			continue
		}
		if isIgnoredFileName(entry.Name()) || s.isGitignored(path, false, scoped) || s.isExcluded(path, exclusions) {
			continue
		}
		found.add(path)
	}
}

func (s *Scanner) isGitignored(path string, isDir bool, ignores []scopedIgnore) bool {
	for _, ignore := range ignores {
		rel, err := filepath.Rel(ignore.directory, path)
		if err != nil {
			continue
		}
		if ignore.matcher.Ignores(filepath.ToSlash(rel), isDir) {
			return true
		}
	}
	return false
}

func (s *Scanner) glob(target string, exclusions []exclusion, found *fileSet) {
	absolute := filepath.ToSlash(target)
	pattern := GlobToRegexp(absolute)
	// Walk from the first non-glob segment of the absolute pattern.
	var fixed []string
	for _, segment := range strings.Split(absolute, "/") {
		if IsGlob(segment) {
			break
		}
		fixed = append(fixed, segment)
	}
	start := strings.Join(fixed, "/")
	if start == "" {
		start = "/"
	}
	start = filepath.FromSlash(start)
	if isFilePath(start) {
		if !s.isExcluded(start, exclusions) {
			found.add(start)
		}
		return
	}
	s.walkAll(start, pattern, exclusions, found)
}

func (s *Scanner) walkAll(directory string, pattern *regexp.Regexp, exclusions []exclusion, found *fileSet) {
	entries, err := os.ReadDir(directory)
	if err != nil {
		return
	}
	for _, entry := range entries {
		path := filepath.Join(directory, entry.Name())
		isDir := entry.IsDir()
		if entry.Type()&os.ModeSymlink != 0 {
			isDir = isDirectoryPath(path)
		}
		if isDir {
			if entry.Name() == ".git" || s.isExcluded(path, exclusions) {
				continue
			}
			s.walkAll(path, pattern, exclusions, found)
			continue
		}
		if !pattern.MatchString(filepath.ToSlash(path)) || s.isExcluded(path, exclusions) {
			continue
		}
		found.add(path)
	}
}

func (s *Scanner) read(path string) bool {
	content, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	extractInto(string(content), s.candidates)
	return true
}

// Scan walks every source, reads files whose modification time changed
// since the last scan (all files on the first scan), and returns the full
// deduplicated candidate set seen so far.
func (s *Scanner) Scan() []string {
	read := []string{}
	for _, path := range s.Files() {
		info, err := os.Stat(path)
		if err != nil {
			continue
		}
		mtime := info.ModTime().UnixNano()
		if previous, ok := s.mtimes[path]; ok && previous == mtime {
			continue
		}
		s.mtimes[path] = mtime
		if s.read(path) {
			read = append(read, path)
		}
	}
	s.ScannedFiles = read
	return s.candidates.slice()
}

// ScanFiles reads only the given files and returns the candidates not seen
// before.
func (s *Scanner) ScanFiles(changed []string) []string {
	before := s.candidates.len()
	for _, path := range changed {
		info, err := os.Stat(path)
		if err != nil || !info.Mode().IsRegular() {
			continue
		}
		s.mtimes[path] = info.ModTime().UnixNano()
		s.read(path)
	}
	return append([]string{}, s.candidates.list[before:]...)
}

// Candidates returns every candidate seen so far.
func (s *Scanner) Candidates() []string { return s.candidates.slice() }
