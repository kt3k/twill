// Package twill implements the Twill utility-first CSS compiler.
package twill

import "strings"

// Node is an element of the AST (SPEC §4.1.1).
type Node interface {
	isNode()
}

// Rule is a style rule.
type Rule struct {
	Selector string
	Nodes    []Node
}

// AtRule is an at-rule. Name includes the leading `@`; Nodes is empty for
// statement at-rules.
type AtRule struct {
	Name   string
	Params string
	Nodes  []Node
}

// Declaration is a property declaration. A declaration with NoValue is
// never printed.
type Declaration struct {
	Property  string
	Value     string
	Important bool
	NoValue   bool
}

// Comment is a kept `/*!` comment; Value is the text between the markers.
type Comment struct {
	Value string
}

// ContextMap carries metadata such as base, reference, theme, source, and
// sourceBase. Boolean flags are stored as "true".
type ContextMap map[string]string

// Bool reports whether the flag key is set.
func (c ContextMap) Bool(key string) bool { return c[key] == "true" }

// Merged returns a copy of c with other's entries applied.
func (c ContextMap) Merged(other ContextMap) ContextMap {
	m := make(ContextMap, len(c)+len(other))
	for k, v := range c {
		m[k] = v
	}
	for k, v := range other {
		m[k] = v
	}
	return m
}

// Context is never printed; it carries metadata to its subtree.
type Context struct {
	Context ContextMap
	Nodes   []Node
}

// AtRoot is never printed in place; its children are hoisted to the end of
// the document during optimization.
type AtRoot struct {
	Nodes []Node
}

func (*Rule) isNode()        {}
func (*AtRule) isNode()      {}
func (*Declaration) isNode() {}
func (*Comment) isNode()     {}
func (*Context) isNode()     {}
func (*AtRoot) isNode()      {}

// StyleRule creates a style rule.
func StyleRule(selector string, nodes ...Node) *Rule {
	if nodes == nil {
		nodes = []Node{}
	}
	return &Rule{Selector: selector, Nodes: nodes}
}

// NewAtRule creates an at-rule.
func NewAtRule(name, params string, nodes ...Node) *AtRule {
	if nodes == nil {
		nodes = []Node{}
	}
	return &AtRule{Name: name, Params: params, Nodes: nodes}
}

// NewRule creates an at-rule when selector starts with `@` and a style rule
// otherwise.
func NewRule(selector string, nodes ...Node) Node {
	if strings.HasPrefix(selector, "@") {
		return ParseAtRulePrelude(selector, nodes...)
	}
	return StyleRule(selector, nodes...)
}

// ParseAtRulePrelude splits `@media (x)` into name and params.
func ParseAtRulePrelude(prelude string, nodes ...Node) *AtRule {
	text := strings.TrimSpace(prelude)
	for i := 1; i < len(text); i++ {
		c := text[i]
		if c == ' ' || c == '\t' || c == '\n' || c == '\f' || c == '\r' || c == '(' {
			return NewAtRule(text[:i], strings.TrimSpace(text[i:]), nodes...)
		}
	}
	return NewAtRule(text, "", nodes...)
}

// Decl creates a declaration.
func Decl(property, value string) *Declaration {
	return &Declaration{Property: property, Value: value}
}

// ImportantDecl creates an important declaration.
func ImportantDecl(property, value string) *Declaration {
	return &Declaration{Property: property, Value: value, Important: true}
}

// NewComment creates a comment node.
func NewComment(value string) *Comment { return &Comment{Value: value} }

// NewContext creates a context node.
func NewContext(ctx ContextMap, nodes ...Node) *Context {
	if nodes == nil {
		nodes = []Node{}
	}
	if ctx == nil {
		ctx = ContextMap{}
	}
	return &Context{Context: ctx, Nodes: nodes}
}

// NewAtRoot creates an at-root node.
func NewAtRoot(nodes ...Node) *AtRoot {
	if nodes == nil {
		nodes = []Node{}
	}
	return &AtRoot{Nodes: nodes}
}

// Children returns a pointer to the child list of a parent node, or nil.
func Children(n Node) *[]Node {
	switch n := n.(type) {
	case *Rule:
		return &n.Nodes
	case *AtRule:
		return &n.Nodes
	case *Context:
		return &n.Nodes
	case *AtRoot:
		return &n.Nodes
	}
	return nil
}

// CloneNodes deep-clones a node list.
func CloneNodes(nodes []Node) []Node {
	out := make([]Node, len(nodes))
	for i, n := range nodes {
		out[i] = CloneNode(n)
	}
	return out
}

// CloneNode deep-clones a node.
func CloneNode(n Node) Node {
	switch n := n.(type) {
	case *Rule:
		return &Rule{Selector: n.Selector, Nodes: CloneNodes(n.Nodes)}
	case *AtRule:
		return &AtRule{Name: n.Name, Params: n.Params, Nodes: CloneNodes(n.Nodes)}
	case *Declaration:
		d := *n
		return &d
	case *Comment:
		c := *n
		return &c
	case *Context:
		return &Context{Context: n.Context.Merged(nil), Nodes: CloneNodes(n.Nodes)}
	case *AtRoot:
		return &AtRoot{Nodes: CloneNodes(n.Nodes)}
	}
	return n
}

// WalkAction controls a walk.
type WalkAction int

const (
	// Continue descends into children.
	Continue WalkAction = iota
	// Skip does not descend into this node's children.
	Skip
	// Stop ends the whole walk.
	Stop
)

// WalkUtils is passed to a walk visitor.
type WalkUtils struct {
	// Parent is the parent node, or nil at the top level.
	Parent Node
	// Context is the merged context of every enclosing context node.
	Context ContextMap
	// Path holds the ancestors, outermost first.
	Path []Node

	replaced    bool
	replacement []Node
}

// ReplaceWith replaces the current node with the given nodes, which are
// then walked in turn.
func (u *WalkUtils) ReplaceWith(nodes ...Node) {
	u.replaced = true
	u.replacement = nodes
	if u.replacement == nil {
		u.replacement = []Node{}
	}
}

// Visitor visits a node during a walk.
type Visitor func(n Node, u *WalkUtils) WalkAction

// Walk walks the tree depth first, parents before children. Replacement
// nodes are visited in turn.
func Walk(nodes *[]Node, visit Visitor) WalkAction {
	return walk(nodes, visit, nil, ContextMap{}, nil)
}

// WalkFrom walks with an explicit parent, context, and path.
func WalkFrom(nodes *[]Node, visit Visitor, parent Node, ctx ContextMap, path []Node) WalkAction {
	return walk(nodes, visit, parent, ctx, path)
}

func walk(nodes *[]Node, visit Visitor, parent Node, ctx ContextMap, path []Node) WalkAction {
	for i := 0; i < len(*nodes); i++ {
		node := (*nodes)[i]
		nodeCtx := ctx
		if c, ok := node.(*Context); ok {
			nodeCtx = ctx.Merged(c.Context)
		}
		u := &WalkUtils{Parent: parent, Context: nodeCtx, Path: path}
		action := visit(node, u)
		if u.replaced {
			rest := append([]Node{}, (*nodes)[i+1:]...)
			*nodes = append(append((*nodes)[:i], u.replacement...), rest...)
		}
		if action == Stop {
			return Stop
		}
		if u.replaced {
			i--
			continue
		}
		if action == Skip {
			continue
		}
		if children := Children(node); children != nil {
			if walk(children, visit, node, nodeCtx, append(path, node)) == Stop {
				return Stop
			}
		}
	}
	return Continue
}

// WalkDepth walks the tree depth first, children before parents.
func WalkDepth(nodes *[]Node, visit func(n Node, u *WalkUtils)) {
	walkDepth(nodes, visit, nil, ContextMap{}, nil)
}

func walkDepth(nodes *[]Node, visit func(n Node, u *WalkUtils), parent Node, ctx ContextMap, path []Node) {
	for i := 0; i < len(*nodes); i++ {
		node := (*nodes)[i]
		nodeCtx := ctx
		if c, ok := node.(*Context); ok {
			nodeCtx = ctx.Merged(c.Context)
		}
		if children := Children(node); children != nil {
			walkDepth(children, visit, node, nodeCtx, append(path, node))
		}
		u := &WalkUtils{Parent: parent, Context: nodeCtx, Path: path}
		visit(node, u)
		if u.replaced {
			rest := append([]Node{}, (*nodes)[i+1:]...)
			*nodes = append(append((*nodes)[:i], u.replacement...), rest...)
			i += len(u.replacement) - 1
		}
	}
}
