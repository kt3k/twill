package twill

import (
	"regexp"
	"strings"
)

func isSpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == '\f'
}

var importantPattern = regexp.MustCompile(`(?i)\s*!important\s*$`)

// Parse parses CSS text into a list of nodes (SPEC §5.1).
func Parse(input string) ([]Node, error) {
	input = strings.ReplaceAll(input, "\r\n", "\n")

	ast := []Node{}
	var stack []Node
	var parent Node

	var buffer strings.Builder
	bufferStart := 0
	brackets := ""
	customPropertyBraces := 0

	fail := func(message string, index int) error {
		p := PositionAt(input, index)
		return &Error{Message: message, Position: &p}
	}
	push := func(n Node) {
		if parent != nil {
			children := Children(parent)
			*children = append(*children, n)
		} else {
			ast = append(ast, n)
		}
	}
	finish := func(end int) error {
		text := strings.TrimSpace(buffer.String())
		buffer.Reset()
		brackets = ""
		customPropertyBraces = 0
		if text == "" {
			return nil
		}
		if text[0] == '@' {
			push(ParseAtRulePrelude(text))
			return nil
		}
		d := ParseDeclaration(text)
		if d == nil {
			return fail("Invalid declaration", end)
		}
		push(d)
		return nil
	}

	for i := 0; i < len(input); i++ {
		c := input[i]

		if c == '\\' {
			if buffer.Len() == 0 {
				bufferStart = i
			}
			end := i + 2
			if end > len(input) {
				end = len(input)
			}
			buffer.WriteString(input[i:end])
			i++
			continue
		}

		if c == '/' && i+1 < len(input) && input[i+1] == '*' {
			start := i
			end := strings.Index(input[i+2:], "*/")
			if end == -1 {
				return nil, fail("Unterminated comment", start)
			}
			end += i + 2
			text := input[start+2 : end]
			if strings.HasPrefix(text, "!") && strings.TrimSpace(buffer.String()) == "" {
				push(NewComment(text))
			}
			i = end + 1
			continue
		}

		if c == '"' || c == '\'' {
			start := i
			if buffer.Len() == 0 {
				bufferStart = i
			}
			j := i + 1
			for ; j < len(input); j++ {
				d := input[j]
				if d == '\\' {
					j++
				} else if d == c {
					break
				} else if d == '\n' {
					return nil, fail("Unterminated string", start)
				}
			}
			if j >= len(input) {
				return nil, fail("Unterminated string", start)
			}
			buffer.WriteString(input[start : j+1])
			i = j
			continue
		}

		if buffer.Len() == 0 && isSpace(c) {
			continue
		}
		if buffer.Len() == 0 {
			bufferStart = i
		}

		current := buffer.String()
		inCustomProperty := strings.HasPrefix(current, "--")

		if c == '(' || c == '[' {
			if c == '(' {
				brackets += ")"
			} else {
				brackets += "]"
			}
			buffer.WriteByte(c)
			continue
		}
		if c == ')' || c == ']' {
			if brackets == "" || brackets[len(brackets)-1] != c {
				return nil, fail("Unexpected `"+string(c)+"`", i)
			}
			brackets = brackets[:len(brackets)-1]
			buffer.WriteByte(c)
			continue
		}
		if brackets != "" {
			buffer.WriteByte(c)
			continue
		}

		if c == ';' {
			if customPropertyBraces > 0 {
				buffer.WriteByte(c)
				continue
			}
			if err := finish(i); err != nil {
				return nil, err
			}
			continue
		}

		if c == '{' {
			if inCustomProperty && strings.Contains(current, ":") {
				customPropertyBraces++
				buffer.WriteByte(c)
				continue
			}
			prelude := strings.TrimSpace(current)
			if prelude == "" {
				return nil, fail("Missing selector before `{`", i)
			}
			var node Node
			if prelude[0] == '@' {
				node = ParseAtRulePrelude(prelude)
			} else {
				node = StyleRule(prelude)
			}
			push(node)
			stack = append(stack, parent)
			parent = node
			buffer.Reset()
			continue
		}

		if c == '}' {
			if customPropertyBraces > 0 {
				customPropertyBraces--
				buffer.WriteByte(c)
				continue
			}
			if parent == nil {
				return nil, fail("Unexpected `}`", i)
			}
			if err := finish(i); err != nil {
				return nil, err
			}
			parent = stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			continue
		}

		buffer.WriteByte(c)
	}

	if parent != nil {
		return nil, fail("Missing closing `}`", len(input))
	}
	if brackets != "" {
		return nil, fail("Missing closing `"+string(brackets[len(brackets)-1])+"`", len(input))
	}
	if strings.TrimSpace(buffer.String()) != "" {
		if err := finish(bufferStart); err != nil {
			return nil, err
		}
	}
	return ast, nil
}

// ParseDeclaration parses `property: value [!important]`, or returns nil
// when the text has no `:`.
func ParseDeclaration(text string) *Declaration {
	colon := strings.Index(text, ":")
	if colon == -1 {
		return nil
	}
	property := strings.TrimSpace(text[:colon])
	if property == "" {
		return nil
	}
	value := strings.TrimSpace(text[colon+1:])
	important := false
	if loc := importantPattern.FindStringIndex(value); loc != nil {
		important = true
		value = strings.TrimRight(value[:loc[0]], " \t\n\r\f")
	}
	return &Declaration{Property: property, Value: value, Important: important}
}
