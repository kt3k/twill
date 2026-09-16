package twill

import "strings"

// ValueNode is a node of a parsed declaration value.
type ValueNode interface {
	isValueNode()
}

// ValueWord is a word.
type ValueWord struct{ Value string }

// ValueSeparator is `,`, `/`, or whitespace.
type ValueSeparator struct{ Value string }

// ValueFunction is a function call; Name is empty for a bare `( ... )`.
type ValueFunction struct {
	Name  string
	Nodes []ValueNode
}

func (*ValueWord) isValueNode()      {}
func (*ValueSeparator) isValueNode() {}
func (*ValueFunction) isValueNode()  {}

// Word creates a word node.
func Word(v string) *ValueWord { return &ValueWord{Value: v} }

// Sep creates a separator node.
func Sep(v string) *ValueSeparator { return &ValueSeparator{Value: v} }

// Fn creates a function node.
func Fn(name string, nodes ...ValueNode) *ValueFunction {
	if nodes == nil {
		nodes = []ValueNode{}
	}
	return &ValueFunction{Name: name, Nodes: nodes}
}

// ParseValue parses a value into words, separators, and function calls.
func ParseValue(input string) []ValueNode {
	ast := []ValueNode{}
	var stack []*ValueFunction
	var parent *ValueFunction
	var buffer strings.Builder

	push := func(n ValueNode) {
		if parent != nil {
			parent.Nodes = append(parent.Nodes, n)
		} else {
			ast = append(ast, n)
		}
	}
	flush := func() {
		if buffer.Len() > 0 {
			push(Word(buffer.String()))
			buffer.Reset()
		}
	}

	for i := 0; i < len(input); i++ {
		c := input[i]
		switch {
		case c == '\\':
			end := i + 2
			if end > len(input) {
				end = len(input)
			}
			buffer.WriteString(input[i:end])
			i++
		case c == '"' || c == '\'':
			start := i
			for i++; i < len(input); i++ {
				d := input[i]
				if d == '\\' {
					i++
				} else if d == c {
					break
				}
			}
			end := i + 1
			if end > len(input) {
				end = len(input)
			}
			buffer.WriteString(input[start:end])
		case c == '(':
			node := Fn(buffer.String())
			buffer.Reset()
			push(node)
			stack = append(stack, parent)
			parent = node
		case c == ')':
			flush()
			if parent != nil {
				parent = stack[len(stack)-1]
				stack = stack[:len(stack)-1]
			} else {
				buffer.WriteByte(')')
			}
		case c == ',' || c == '/':
			flush()
			push(Sep(string(c)))
		case isSpace(c):
			flush()
			j := i
			for j < len(input) && isSpace(input[j]) {
				j++
			}
			push(Sep(input[i:j]))
			i = j - 1
		default:
			buffer.WriteByte(c)
		}
	}
	flush()
	return ast
}

// ValueToCSS prints value nodes back to CSS text.
func ValueToCSS(nodes []ValueNode) string {
	var b strings.Builder
	valueToCSS(&b, nodes)
	return b.String()
}

func valueToCSS(b *strings.Builder, nodes []ValueNode) {
	for _, n := range nodes {
		switch n := n.(type) {
		case *ValueWord:
			b.WriteString(n.Value)
		case *ValueSeparator:
			b.WriteString(n.Value)
		case *ValueFunction:
			b.WriteString(n.Name)
			b.WriteByte('(')
			valueToCSS(b, n.Nodes)
			b.WriteByte(')')
		}
	}
}

// ValueVisitor visits a value node. It may return replacement nodes (which
// are not revisited) and whether to skip the node's children.
type ValueVisitor func(n ValueNode, parent *ValueFunction) (replacement []ValueNode, replaced bool, skip bool)

// WalkValue walks value nodes depth first with replacement support.
func WalkValue(nodes *[]ValueNode, visit ValueVisitor) {
	walkValue(nodes, visit, nil)
}

func walkValue(nodes *[]ValueNode, visit ValueVisitor, parent *ValueFunction) {
	for i := 0; i < len(*nodes); i++ {
		n := (*nodes)[i]
		replacement, replaced, skip := visit(n, parent)
		if replaced {
			rest := append([]ValueNode{}, (*nodes)[i+1:]...)
			*nodes = append(append((*nodes)[:i], replacement...), rest...)
			i += len(replacement) - 1
			continue
		}
		if skip {
			continue
		}
		if f, ok := n.(*ValueFunction); ok {
			walkValue(&f.Nodes, visit, f)
		}
	}
}
