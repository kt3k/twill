package twill

import "strings"

// Serialize prints nodes with two-space indentation per depth (SPEC §5.2).
func Serialize(nodes []Node) string {
	var b strings.Builder
	serialize(&b, nodes, 0)
	return b.String()
}

func serialize(b *strings.Builder, nodes []Node, depth int) {
	for _, n := range nodes {
		serializeNode(b, n, depth)
	}
}

func serializeNode(b *strings.Builder, n Node, depth int) {
	indent := strings.Repeat("  ", depth)
	switch n := n.(type) {
	case *Declaration:
		if n.NoValue {
			return
		}
		b.WriteString(indent)
		b.WriteString(n.Property)
		b.WriteString(": ")
		b.WriteString(n.Value)
		if n.Important {
			b.WriteString(" !important")
		}
		b.WriteString(";\n")
	case *Rule:
		b.WriteString(indent)
		b.WriteString(n.Selector)
		b.WriteString(" {\n")
		serialize(b, n.Nodes, depth+1)
		b.WriteString(indent)
		b.WriteString("}\n")
	case *AtRule:
		b.WriteString(indent)
		b.WriteString(n.Name)
		if n.Params != "" {
			b.WriteString(" ")
			b.WriteString(n.Params)
		}
		if len(n.Nodes) == 0 {
			b.WriteString(";\n")
			return
		}
		b.WriteString(" {\n")
		serialize(b, n.Nodes, depth+1)
		b.WriteString(indent)
		b.WriteString("}\n")
	case *Comment:
		b.WriteString(indent)
		b.WriteString("/*")
		b.WriteString(n.Value)
		b.WriteString("*/\n")
	case *Context:
		serialize(b, n.Nodes, depth)
	case *AtRoot:
		serialize(b, n.Nodes, depth)
	}
}

// SerializeCompact prints nodes without indentation or line breaks. Used for
// `--minify`; the output is semantically identical to Serialize.
func SerializeCompact(nodes []Node) string {
	var b strings.Builder
	compact(&b, nodes)
	return b.String()
}

func compact(b *strings.Builder, nodes []Node) {
	for _, n := range nodes {
		switch n := n.(type) {
		case *Declaration:
			if n.NoValue {
				continue
			}
			b.WriteString(n.Property)
			b.WriteString(":")
			b.WriteString(n.Value)
			if n.Important {
				b.WriteString("!important")
			}
			b.WriteString(";")
		case *Rule:
			b.WriteString(n.Selector)
			b.WriteString("{")
			compact(b, n.Nodes)
			b.WriteString("}")
		case *AtRule:
			b.WriteString(n.Name)
			if n.Params != "" {
				b.WriteString(" ")
				b.WriteString(n.Params)
			}
			if len(n.Nodes) == 0 {
				b.WriteString(";")
				continue
			}
			b.WriteString("{")
			compact(b, n.Nodes)
			b.WriteString("}")
		case *Comment:
			b.WriteString("/*")
			b.WriteString(n.Value)
			b.WriteString("*/")
		case *Context:
			compact(b, n.Nodes)
		case *AtRoot:
			compact(b, n.Nodes)
		}
	}
}
