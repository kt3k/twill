package twill

import (
	"regexp"
	"strings"
)

// A small .gitignore matcher for source auto-detection (SPEC §13.2).
//
// Supports blank lines, `#` comments, `!` negation, trailing `/` for
// directories, anchoring with `/`, and the `*`, `**`, and `?` wildcards.

type gitignoreRule struct {
	pattern       *regexp.Regexp
	negated       bool
	directoryOnly bool
	// Anchored patterns match the full path relative to the ignore file.
	anchored bool
}

// Gitignore is a parsed .gitignore file.
type Gitignore struct {
	rules []gitignoreRule
}

func gitignoreGlobToRegexp(glob string) *regexp.Regexp {
	var b strings.Builder
	b.WriteString("^")
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

// ParseGitignore parses the content of a .gitignore file.
func ParseGitignore(content string) *Gitignore {
	g := &Gitignore{}
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSuffix(line, "\r")
		if strings.TrimSpace(line) == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = trimUnescapedTrailingSpace(line)
		negated := false
		if strings.HasPrefix(line, "!") {
			negated = true
			line = line[1:]
		} else if strings.HasPrefix(line, `\!`) || strings.HasPrefix(line, `\#`) {
			line = line[1:]
		}
		directoryOnly := false
		if strings.HasSuffix(line, "/") {
			directoryOnly = true
			line = line[:len(line)-1]
		}
		anchored := false
		if strings.HasPrefix(line, "/") {
			anchored = true
			line = line[1:]
		} else if strings.Contains(line, "/") {
			anchored = true
		}
		if line == "" {
			continue
		}
		g.rules = append(g.rules, gitignoreRule{
			pattern:       gitignoreGlobToRegexp(line),
			negated:       negated,
			directoryOnly: directoryOnly,
			anchored:      anchored,
		})
	}
	return g
}

func trimUnescapedTrailingSpace(line string) string {
	end := len(line)
	for end > 0 && (line[end-1] == ' ' || line[end-1] == '\t') {
		end--
	}
	if end > 0 && end < len(line) && line[end-1] == '\\' {
		end++
	}
	return line[:end]
}

// Size is the number of rules.
func (g *Gitignore) Size() int { return len(g.rules) }

// Ignores reports whether relativePath (relative to the directory of the
// ignore file, using `/` separators) is ignored.
func (g *Gitignore) Ignores(relativePath string, isDirectory bool) bool {
	ignored := false
	basename := relativePath[strings.LastIndex(relativePath, "/")+1:]
	for _, rule := range g.rules {
		if rule.directoryOnly && !isDirectory {
			continue
		}
		var matched bool
		if rule.anchored {
			matched = rule.pattern.MatchString(relativePath)
		} else {
			matched = rule.pattern.MatchString(basename)
		}
		if matched {
			ignored = !rule.negated
		}
	}
	return ignored
}
