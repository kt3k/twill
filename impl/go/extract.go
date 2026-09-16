package twill

// Candidate extraction from arbitrary text (SPEC §13.4).

func extractIsWhitespace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == '\f'
}

func extractIsQuote(c byte) bool { return c == '"' || c == '\'' || c == '`' }

func extractIsStartBoundary(c byte) bool {
	return extractIsWhitespace(c) || extractIsQuote(c) || c == '.' || c == '}' || c == '>'
}

func extractIsEndBoundary(c byte) bool {
	return extractIsWhitespace(c) || extractIsQuote(c) || c == ']' || c == '{' || c == '=' || c == '\\' || c == '<'
}

func extractIsLetter(c byte) bool { return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' }

func extractIsNameChar(c byte) bool {
	return extractIsLetter(c) || isDigit(c) || c == '_' || c == '-'
}

// at returns the byte at i, or 0 past the end.
func at(text string, i int) byte {
	if i < 0 || i >= len(text) {
		return 0
	}
	return text[i]
}

// consumeBalanced consumes a balanced bracket group starting at i (the
// opener). It returns the index after the closer, or -1 when unbalanced or
// interrupted by a line break.
func consumeBalanced(text string, i int, open, close byte) int {
	depth := 0
	for j := i; j < len(text); j++ {
		c := text[j]
		if c == '\n' {
			return -1
		}
		if c == '\\' {
			j++
			continue
		}
		if c == open {
			depth++
		} else if c == close {
			depth--
			if depth == 0 {
				return j + 1
			}
		}
	}
	return -1
}

// consumeBracketSuffix consumes `[...]` or `(...)` at i.
func consumeBracketSuffix(text string, i int) int {
	switch at(text, i) {
	case '[':
		return consumeBalanced(text, i, '[', ']')
	case '(':
		return consumeBalanced(text, i, '(', ')')
	}
	return -1
}

// consumeModifier consumes `/modifier` at i (the slash) and returns the end.
func consumeModifier(text string, i int) int {
	c := at(text, i+1)
	if c == '[' || c == '(' {
		end := consumeBracketSuffix(text, i+1)
		if end == -1 {
			return i
		}
		return end
	}
	j := i + 1
	for j < len(text) {
		d := text[j]
		if extractIsNameChar(d) || d == '.' || d == '%' {
			j++
		} else {
			break
		}
	}
	if j == i+1 {
		return i
	}
	last := text[j-1]
	if last == '-' || last == '_' {
		return i
	}
	return j
}

// consumeVariant consumes a variant at i and returns the index after it
// (before the `:`), or -1.
func consumeVariant(text string, i int) int {
	c := at(text, i)
	if c == '[' {
		return consumeBalanced(text, i, '[', ']')
	}
	if !extractIsLetter(c) && c != '@' {
		return -1
	}
	j := i + 1
	for j < len(text) {
		d := text[j]
		if extractIsNameChar(d) {
			j++
			continue
		}
		if (d == '[' || d == '(') && text[j-1] == '-' {
			end := consumeBracketSuffix(text, j)
			if end == -1 {
				return -1
			}
			j = end
			continue
		}
		break
	}
	if j == i+1 && c == '@' {
		return -1
	}
	last := text[j-1]
	if last == '-' || last == '_' {
		return -1
	}
	if at(text, j) == '/' {
		j = consumeModifier(text, j)
	}
	return j
}

// consumeUtility consumes a utility at i and returns the index after it, or -1.
func consumeUtility(text string, i int) int {
	j := i
	if at(text, j) == '!' {
		j++
	}
	c := at(text, j)

	if c == '[' {
		end := consumeBalanced(text, j, '[', ']')
		if end == -1 {
			return -1
		}
		found := false
		for k := j; k < end; k++ {
			if text[k] == ':' {
				found = true
				break
			}
		}
		if !found {
			return -1
		}
		j = end
	} else {
		if c == '-' {
			next := at(text, j+1)
			if !extractIsLetter(next) && !isDigit(next) {
				return -1
			}
			j++
		} else if !extractIsLetter(c) && c != '@' {
			return -1
		}
		j++
		for j < len(text) {
			d := text[j]
			if extractIsNameChar(d) || d == '%' {
				j++
				continue
			}
			if d == '.' {
				if isDigit(text[j-1]) && isDigit(at(text, j+1)) {
					j++
					continue
				}
				break
			}
			if (d == '[' || d == '(') && text[j-1] == '-' {
				end := consumeBracketSuffix(text, j)
				if end == -1 {
					return -1
				}
				j = end
				continue
			}
			break
		}
		last := text[j-1]
		if last == '-' || last == '_' {
			return -1
		}
	}

	if at(text, j) == '/' {
		j = consumeModifier(text, j)
	}
	if at(text, j) == '!' {
		j++
	}
	return j
}

// consumeCandidate consumes a full candidate `(variant ":")* utility` at i.
func consumeCandidate(text string, i int) int {
	j := i
	for {
		end := consumeVariant(text, j)
		if end != -1 && at(text, end) == ':' {
			j = end + 1
			continue
		}
		break
	}
	return consumeUtility(text, j)
}

// consumeVariable consumes a `--name` custom property reference at i.
func consumeVariable(text string, i int) int {
	j := i + 2
	for j < len(text) && extractIsNameChar(text[j]) {
		j++
	}
	if j == i+2 {
		return -1
	}
	return j
}

// candidateSet is an insertion-ordered set of candidates.
type candidateSet struct {
	seen map[string]bool
	list []string
}

func newCandidateSet() *candidateSet { return &candidateSet{seen: map[string]bool{}} }

func (s *candidateSet) add(c string) bool {
	if s.seen[c] {
		return false
	}
	s.seen[c] = true
	s.list = append(s.list, c)
	return true
}

func (s *candidateSet) has(c string) bool { return s.seen[c] }

func (s *candidateSet) len() int { return len(s.list) }

func (s *candidateSet) slice() []string { return append([]string{}, s.list...) }

// ExtractCandidates extracts every candidate-like span from text (SPEC
// §13.4) in order of first appearance. Recall is favored over precision:
// unknown candidates are harmless.
func ExtractCandidates(text string) []string {
	set := newCandidateSet()
	extractInto(text, set)
	return set.list
}

func extractInto(text string, into *candidateSet) {
	length := len(text)
	i := 0
	for i < length {
		c := text[i]
		if i > 0 && !extractIsStartBoundary(text[i-1]) {
			i++
			continue
		}

		end := -1
		if c == '-' && at(text, i+1) == '-' {
			end = consumeVariable(text, i)
		} else if extractIsLetter(c) || c == '@' || c == '-' || c == '!' || c == '[' {
			end = consumeCandidate(text, i)
		}

		if end != -1 && end > i && (end == length || extractIsEndBoundary(text[end])) {
			into.add(text[i:end])
			i = end
			continue
		}

		// Skip to the next boundary so that substrings are never emitted.
		i++
		for i < length && !extractIsStartBoundary(text[i-1]) {
			i++
		}
	}
}
