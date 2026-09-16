package twill

import (
	"math"
	"regexp"
	"strconv"
	"strings"
)

// Named values and named modifiers must match this pattern (SPEC §4.2).
var namedValuePattern = regexp.MustCompile(`^[a-zA-Z0-9_.%-]+$`)

// Custom variant names must match this pattern and not end in `_` or `-`.
var variantNamePattern = regexp.MustCompile(`^@?[a-z0-9][a-zA-Z0-9_-]*$`)

// A theme prefix must match this pattern.
var prefixPattern = regexp.MustCompile(`^[a-z]+$`)

// IsNamedValue reports whether v matches the named value pattern.
func IsNamedValue(v string) bool { return namedValuePattern.MatchString(v) }

// IsValidVariantName reports whether name is a valid custom variant name.
func IsValidVariantName(name string) bool {
	if !variantNamePattern.MatchString(name) {
		return false
	}
	last := name[len(name)-1]
	return last != '_' && last != '-'
}

// IsValidPrefix reports whether p is a valid theme prefix.
func IsValidPrefix(p string) bool { return prefixPattern.MatchString(p) }

// Escape escapes a string for use in a class selector, matching CSS.escape.
func Escape(value string) string {
	var b strings.Builder
	length := len(value)
	var first byte
	if length > 0 {
		first = value[0]
	}
	for i := 0; i < length; i++ {
		c := value[i]
		switch {
		case c == 0:
			b.WriteString("�")
		case (c >= 0x01 && c <= 0x1f) || c == 0x7f ||
			(i == 0 && c >= '0' && c <= '9') ||
			(i == 1 && c >= '0' && c <= '9' && first == '-'):
			b.WriteString("\\")
			b.WriteString(strconv.FormatInt(int64(c), 16))
			b.WriteString(" ")
		case i == 0 && length == 1 && c == '-':
			b.WriteString("\\-")
		case c >= 0x80 || c == '-' || c == '_' ||
			(c >= '0' && c <= '9') || (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z'):
			b.WriteByte(c)
		default:
			b.WriteByte('\\')
			b.WriteByte(c)
		}
	}
	return b.String()
}

var unescapePattern = regexp.MustCompile(`\\([0-9a-fA-F]{1,6}\s?|[\s\S])`)

// Unescape reverses Escape.
func Unescape(value string) string {
	if !strings.Contains(value, "\\") {
		return value
	}
	return unescapePattern.ReplaceAllStringFunc(value, func(m string) string {
		esc := m[1:]
		c := esc[0]
		if (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F') {
			n, err := strconv.ParseInt(strings.TrimSpace(esc), 16, 32)
			if err == nil {
				return string(rune(n))
			}
		}
		return esc
	})
}

// Segment splits input on a single-byte separator, ignoring separators
// inside brackets, quotes, and after a backslash.
func Segment(input string, sep byte) []string {
	var parts []string
	stack := ""
	last := 0
	for i := 0; i < len(input); i++ {
		c := input[i]
		switch {
		case c == '\\':
			i++
		case c == '"' || c == '\'':
			for i++; i < len(input); i++ {
				d := input[i]
				if d == '\\' {
					i++
				} else if d == c {
					break
				}
			}
		case c == '(':
			stack += ")"
		case c == '[':
			stack += "]"
		case c == '{':
			stack += "}"
		case c == ')' || c == ']' || c == '}':
			if stack != "" && stack[len(stack)-1] == c {
				stack = stack[:len(stack)-1]
			}
		case c == sep && stack == "":
			parts = append(parts, input[last:i])
			last = i + 1
		}
	}
	parts = append(parts, input[last:])
	return parts
}

// IsValidArbitrary checks bracket balance of an arbitrary value.
func IsValidArbitrary(input string) bool {
	stack := ""
	for i := 0; i < len(input); i++ {
		c := input[i]
		switch {
		case c == '\\':
			i++
		case c == '"' || c == '\'':
			for i++; i < len(input); i++ {
				d := input[i]
				if d == '\\' {
					i++
				} else if d == c {
					break
				}
			}
		case c == '(':
			stack += ")"
		case c == '[':
			stack += "]"
		case c == ')' || c == ']' || c == '}':
			if stack == "" || stack[len(stack)-1] != c {
				return false
			}
			stack = stack[:len(stack)-1]
		case c == ';' && stack == "":
			return false
		}
	}
	return true
}

func isDigit(c byte) bool { return c >= '0' && c <= '9' }

// Compare is a natural string comparison: digit runs compare numerically.
func Compare(a, z string) int {
	i, j := 0, 0
	for i < len(a) && j < len(z) {
		ca, cz := a[i], z[j]
		if isDigit(ca) && isDigit(cz) {
			ei := i
			for ei < len(a) && isDigit(a[ei]) {
				ei++
			}
			ej := j
			for ej < len(z) && isDigit(z[ej]) {
				ej++
			}
			ra, rz := a[i:ei], z[j:ej]
			na := strings.TrimLeft(ra, "0")
			nz := strings.TrimLeft(rz, "0")
			if na == "" {
				na = "0"
			}
			if nz == "" {
				nz = "0"
			}
			if len(na) != len(nz) {
				return len(na) - len(nz)
			}
			if na != nz {
				return strings.Compare(na, nz)
			}
			if ra != rz {
				return strings.Compare(ra, rz)
			}
			i, j = ei, ej
			continue
		}
		if ca != cz {
			return int(ca) - int(cz)
		}
		i++
		j++
	}
	return (len(a) - i) - (len(z) - j)
}

// IsPositiveInteger reports whether v is an integer >= 0 in canonical form.
func IsPositiveInteger(v string) bool {
	n, err := strconv.Atoi(v)
	return err == nil && n >= 0 && strconv.Itoa(n) == v
}

// IsStrictPositiveInteger reports whether v is an integer > 0 in canonical form.
func IsStrictPositiveInteger(v string) bool {
	n, err := strconv.Atoi(v)
	return err == nil && n > 0 && strconv.Itoa(n) == v
}

var quarterPattern = regexp.MustCompile(`^-?(0|[1-9]\d*)(\.\d*[1-9])?$`)

// IsMultipleOfQuarter reports whether v is a multiple of 0.25 without
// redundant zeros.
func IsMultipleOfQuarter(v string) bool {
	if !quarterPattern.MatchString(v) {
		return false
	}
	n, err := strconv.ParseFloat(v, 64)
	if err != nil || math.IsInf(n, 0) {
		return false
	}
	return math.Mod(n, 0.25) == 0
}

var rangePattern = regexp.MustCompile(`^(-?\d+)\.\.(-?\d+)(?:\.\.(-?\d+))?$`)

// ExpandBraces performs brace expansion: `{a,b}` enumerates and `{1..5}`
// produces ranges; nesting is allowed.
func ExpandBraces(pattern string) ([]string, error) {
	depth := 0
	open, close := -1, -1
	for i := 0; i < len(pattern); i++ {
		c := pattern[i]
		if c == '\\' {
			i++
			continue
		}
		if c == '{' {
			if depth == 0 {
				open = i
			}
			depth++
		} else if c == '}' {
			if depth == 0 {
				return nil, Errorf("Unbalanced braces in `%s`", pattern)
			}
			depth--
			if depth == 0 {
				close = i
				break
			}
		}
	}
	if depth != 0 {
		return nil, Errorf("Unbalanced braces in `%s`", pattern)
	}
	if open == -1 {
		return []string{pattern}, nil
	}
	prefix := pattern[:open]
	inner := pattern[open+1 : close]
	suffix := pattern[close+1:]

	var items []string
	if m := rangePattern.FindStringSubmatch(inner); m != nil {
		start, _ := strconv.Atoi(m[1])
		end, _ := strconv.Atoi(m[2])
		step := 1
		if m[3] != "" {
			step, _ = strconv.Atoi(m[3])
			if step < 0 {
				step = -step
			}
		}
		if step == 0 {
			return nil, Errorf("Step cannot be zero in `%s`", pattern)
		}
		if end < start {
			for n := start; n >= end; n -= step {
				items = append(items, strconv.Itoa(n))
			}
		} else {
			for n := start; n <= end; n += step {
				items = append(items, strconv.Itoa(n))
			}
		}
	} else {
		items = Segment(inner, ',')
	}

	var result []string
	for _, item := range items {
		expanded, err := ExpandBraces(item + suffix)
		if err != nil {
			return nil, err
		}
		for _, e := range expanded {
			result = append(result, prefix+e)
		}
	}
	return result, nil
}

// Unquote strips one pair of surrounding quotes; ok is false when absent.
func Unquote(value string) (string, bool) {
	if len(value) < 2 {
		return "", false
	}
	first, last := value[0], value[len(value)-1]
	if (first == '"' || first == '\'') && last == first {
		return value[1 : len(value)-1], true
	}
	return "", false
}

var mathFunctions = map[string]bool{
	"calc": true, "min": true, "max": true, "clamp": true, "round": true, "mod": true,
	"rem": true, "sin": true, "cos": true, "tan": true, "asin": true, "acos": true,
	"atan": true, "atan2": true, "pow": true, "sqrt": true, "hypot": true, "log": true,
	"exp": true, "abs": true, "sign": true,
}

// IsMathFunction reports whether name is a CSS math function.
func IsMathFunction(name string) bool { return mathFunctions[name] }

// DecodeArbitraryValue decodes underscores and spaces math operators.
func DecodeArbitraryValue(input string) string {
	if !strings.Contains(input, "(") {
		return replaceUnderscores(input)
	}
	ast := ParseValue(input)
	decodeValueNodes(ast)
	return AddWhitespaceAroundMathOperators(ValueToCSS(ast))
}

func replaceUnderscores(input string) string {
	if !strings.Contains(input, "_") {
		return input
	}
	var b strings.Builder
	for i := 0; i < len(input); i++ {
		c := input[i]
		if c == '\\' && i+1 < len(input) && input[i+1] == '_' {
			b.WriteByte('_')
			i++
		} else if c == '_' {
			b.WriteByte(' ')
		} else {
			b.WriteByte(c)
		}
	}
	return b.String()
}

func decodeValueNodes(nodes []ValueNode) {
	for _, n := range nodes {
		switch n := n.(type) {
		case *ValueWord:
			n.Value = replaceUnderscores(n.Value)
		case *ValueFunction:
			name := n.Name
			if name == "url" || strings.HasSuffix(name, "_url") {
				continue
			}
			if name == "var" || name == "theme" || name == "--theme" {
				firstComma := len(n.Nodes)
				for i, child := range n.Nodes {
					if s, ok := child.(*ValueSeparator); ok && s.Value == "," {
						firstComma = i
						break
					}
				}
				for _, child := range n.Nodes[:firstComma] {
					if w, ok := child.(*ValueWord); ok {
						w.Value = strings.ReplaceAll(w.Value, "\\_", "_")
					}
				}
				decodeValueNodes(n.Nodes[firstComma:])
				continue
			}
			decodeValueNodes(n.Nodes)
		}
	}
}

var mathCallPattern = regexp.MustCompile(`(?:^|[^a-zA-Z0-9_-])(?:calc|min|max|clamp|round|mod|rem|sin|cos|tan|asin|acos|atan|atan2|pow|sqrt|hypot|log|exp|abs|sign)\(`)

func isIdentByte(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || isDigit(c) || c == '_' || c == '-'
}

func isLetter(c byte) bool { return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') }

// followsValue reports whether the operator at index follows a value.
func followsValue(input string, index int) bool {
	i := index - 1
	for i >= 0 && input[i] == ' ' {
		i--
	}
	if i < 0 {
		return false
	}
	c := input[i]
	if isDigit(c) || c == '%' || c == ')' {
		return true
	}
	if isLetter(c) {
		for i >= 0 && isLetter(input[i]) {
			i--
		}
		return i >= 0 && (isDigit(input[i]) || input[i] == '.')
	}
	return false
}

// AddWhitespaceAroundMathOperators inserts spaces around operators inside
// math functions.
func AddWhitespaceAroundMathOperators(input string) string {
	if !mathCallPattern.MatchString(input) {
		return input
	}
	var b strings.Builder
	var stack []bool
	inMath := func() bool { return len(stack) > 0 && stack[len(stack)-1] }

	for i := 0; i < len(input); i++ {
		c := input[i]
		if c == '\\' {
			end := i + 2
			if end > len(input) {
				end = len(input)
			}
			b.WriteString(input[i:end])
			i++
			continue
		}
		if c == '"' || c == '\'' {
			start := i
			for i++; i < len(input); i++ {
				if input[i] == '\\' {
					i++
				} else if input[i] == c {
					break
				}
			}
			end := i + 1
			if end > len(input) {
				end = len(input)
			}
			b.WriteString(input[start:end])
			continue
		}
		if c == '(' {
			start := i
			for start > 0 && isIdentByte(input[start-1]) {
				start--
			}
			name := input[start:i]
			if strings.HasPrefix(name, "-") && !strings.HasPrefix(name, "--") {
				name = name[1:]
			}
			if name == "" {
				stack = append(stack, inMath())
			} else {
				stack = append(stack, IsMathFunction(name))
			}
			b.WriteByte(c)
			continue
		}
		if c == ')' {
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
			b.WriteByte(c)
			continue
		}
		if !inMath() {
			b.WriteByte(c)
			continue
		}
		isOperator := false
		if c == '*' || c == '/' {
			isOperator = true
		} else if c == '+' || c == '-' {
			next := i + 1
			for next < len(input) && input[next] == ' ' {
				next++
			}
			isOperator = followsValue(input, i) && next < len(input) && input[next] != ')'
		}
		if isOperator {
			s := b.String()
			if !strings.HasSuffix(s, " ") {
				b.WriteByte(' ')
			}
			b.WriteByte(c)
			b.WriteByte(' ')
			for i+1 < len(input) && input[i+1] == ' ' {
				i++
			}
			continue
		}
		b.WriteByte(c)
	}
	return b.String()
}
