package twill

import (
	"regexp"
	"strings"
)

var keepEmptyAtRules = map[string]bool{"@layer": true, "@charset": true, "@custom-media": true, "@namespace": true, "@import": true, "@apply": true}

// At-rules whose child rules are not nested style rules.
var opaqueAtRules = map[string]bool{"@keyframes": true, "@property": true, "@font-face": true, "@counter-style": true, "@page": true}

var varPattern = regexp.MustCompile(`var\(\s*(--[^\s,)]+)`)

type trackedDeclaration struct {
	node *Declaration
	key  string
}

// OptimizeAst optimizes the whole document before serialization (SPEC §12).
// The input is not mutated; a new tree is returned.
func OptimizeAst(ast []Node, ds *DesignSystem) []Node {
	var themeDeclarations []trackedDeclaration
	var themeKeyframes []*AtRule
	usedVariables := map[string]bool{}
	dependencies := map[string]map[string]bool{}
	usedKeyframes := map[string]bool{}
	seenProperties := map[string]bool{}
	var roots []Node

	recordVariables := func(value string, ownerKey string, isTheme bool) {
		if !strings.Contains(value, "var(") {
			return
		}
		for _, m := range varPattern.FindAllStringSubmatch(value, -1) {
			if !isTheme {
				usedVariables[m[1]] = true
				continue
			}
			if dependencies[ownerKey] == nil {
				dependencies[ownerKey] = map[string]bool{}
			}
			dependencies[ownerKey][m[1]] = true
		}
	}

	var transform func(nodes []Node, out *[]Node, ctx ContextMap, depth int)
	transform = func(nodes []Node, out *[]Node, ctx ContextMap, depth int) {
		for _, n := range nodes {
			switch n := n.(type) {
			case *Declaration:
				if n.NoValue || n.Property == "--tw-sort" {
					continue
				}
				copy := *n
				if ctx.Bool("theme") && strings.HasPrefix(n.Property, "--") {
					if n.Value == "initial" {
						continue
					}
					key := ds.Theme.UnprefixKey(Unescape(n.Property))
					recordVariables(n.Value, key, true)
					themeDeclarations = append(themeDeclarations, trackedDeclaration{&copy, key})
				} else {
					recordVariables(n.Value, "", false)
				}
				if n.Property == "animation" {
					for _, token := range strings.FieldsFunc(n.Value, func(r rune) bool { return r == ' ' || r == ',' || r == '\t' || r == '\n' }) {
						usedKeyframes[token] = true
					}
				}
				*out = append(*out, &copy)
			case *Rule:
				var children []Node
				transform(n.Nodes, &children, ctx, depth+1)
				if len(children) == 0 {
					continue
				}
				*out = append(*out, StyleRule(n.Selector, children...))
			case *AtRule:
				if n.Name == "@property" && depth == 0 {
					if seenProperties[n.Params] {
						continue
					}
					seenProperties[n.Params] = true
				}
				var children []Node
				transform(n.Nodes, &children, ctx, depth+1)
				if len(children) == 0 && !keepEmptyAtRules[n.Name] {
					continue
				}
				copy := NewAtRule(n.Name, n.Params, children...)
				if n.Name == "@keyframes" && ctx.Bool("theme") {
					themeKeyframes = append(themeKeyframes, copy)
				}
				*out = append(*out, copy)
			case *AtRoot:
				transform(n.Nodes, &roots, ctx, 0)
			case *Context:
				if n.Context.Bool("reference") {
					continue
				}
				transform(n.Nodes, out, ctx.Merged(n.Context), depth)
			case *Comment:
				copy := *n
				*out = append(*out, &copy)
			}
		}
	}

	var result []Node
	transform(ast, &result, ContextMap{}, 0)

	for name := range usedVariables {
		ds.Theme.MarkUsedVariable(name)
	}
	keep := map[string]bool{}
	for _, d := range themeDeclarations {
		if ds.Theme.Options(d.key)&(Static|Used) != 0 {
			keep[d.key] = true
		}
	}
	for changed := true; changed; {
		changed = false
		for key := range keep {
			for dep := range dependencies[key] {
				depKey := ds.Theme.UnprefixKey(Unescape(dep))
				if !keep[depKey] && ds.Theme.Has(depKey) {
					keep[depKey] = true
					changed = true
				}
			}
		}
	}
	removed := map[Node]bool{}
	for _, d := range themeDeclarations {
		if !keep[d.key] {
			removed[d.node] = true
		}
	}
	for key := range keep {
		if !strings.HasPrefix(key, "--animate") {
			continue
		}
		if value, ok := ds.Theme.Get(key); ok {
			for _, token := range strings.FieldsFunc(value, func(r rune) bool { return r == ' ' || r == ',' }) {
				usedKeyframes[token] = true
			}
		}
	}
	for _, k := range themeKeyframes {
		if !usedKeyframes[strings.TrimSpace(k.Params)] {
			removed[k] = true
		}
	}

	if len(removed) > 0 {
		result = removeNodes(result, removed)
		roots = removeNodes(roots, removed)
	}
	return Flatten(append(result, roots...))
}

func removeNodes(nodes []Node, removed map[Node]bool) []Node {
	var out []Node
	for _, n := range nodes {
		if removed[n] {
			continue
		}
		switch n := n.(type) {
		case *Rule:
			children := removeNodes(n.Nodes, removed)
			if len(children) == 0 {
				continue
			}
			out = append(out, StyleRule(n.Selector, children...))
		case *AtRule:
			children := removeNodes(n.Nodes, removed)
			if len(children) == 0 && len(n.Nodes) > 0 && (!keepEmptyAtRules[n.Name] || n.Name == "@layer") {
				continue
			}
			out = append(out, NewAtRule(n.Name, n.Params, children...))
		default:
			out = append(out, n)
		}
	}
	return out
}

// CombineSelectors combines a parent selector with a nested selector (SPEC §12.3).
func CombineSelectors(parent, child string) string {
	wrapped := parent
	if len(Segment(parent, ',')) > 1 {
		wrapped = ":is(" + parent + ")"
	}
	if strings.Contains(child, "&") {
		return strings.ReplaceAll(child, "&", wrapped)
	}
	if len(child) > 0 && (child[0] == '>' || child[0] == '+' || child[0] == '~') {
		return wrapped + " " + child
	}
	return wrapped + child
}

// Flatten flattens nested rules and at-rules into flat CSS (SPEC §12.3).
func Flatten(nodes []Node) []Node {
	var out []Node
	for _, n := range nodes {
		switch n := n.(type) {
		case *Rule:
			out = append(out, flattenRule(n, "", false)...)
		case *AtRule:
			if opaqueAtRules[n.Name] {
				out = append(out, n)
			} else {
				out = append(out, NewAtRule(n.Name, n.Params, Flatten(n.Nodes)...))
			}
		case *Context:
			out = append(out, Flatten(n.Nodes)...)
		case *AtRoot:
			out = append(out, Flatten(n.Nodes)...)
		default:
			out = append(out, n)
		}
	}
	return out
}

func flattenRule(rule *Rule, parentSelector string, hasParent bool) []Node {
	var selector string
	switch {
	case !hasParent:
		selector = rule.Selector
	case rule.Selector == "&":
		selector = parentSelector
	default:
		selector = CombineSelectors(parentSelector, rule.Selector)
	}
	return flattenChildren(rule.Nodes, selector)
}

func flattenChildren(nodes []Node, selector string) []Node {
	var declarations, hoisted []Node
	for _, child := range nodes {
		switch child := child.(type) {
		case *Declaration, *Comment:
			declarations = append(declarations, child)
		case *Rule:
			hoisted = append(hoisted, flattenRule(child, selector, true)...)
		case *AtRule:
			if len(child.Nodes) == 0 {
				declarations = append(declarations, child)
			} else {
				hoisted = append(hoisted, flattenAtRule(child, selector)...)
			}
		case *Context:
			hoisted = append(hoisted, flattenChildren(child.Nodes, selector)...)
		case *AtRoot:
			hoisted = append(hoisted, flattenChildren(child.Nodes, selector)...)
		}
	}
	var out []Node
	if len(declarations) > 0 {
		out = append(out, StyleRule(selector, declarations...))
	}
	return append(out, hoisted...)
}

func flattenAtRule(node *AtRule, selector string) []Node {
	if opaqueAtRules[node.Name] {
		return []Node{node}
	}
	children := flattenChildren(node.Nodes, selector)
	if len(children) == 0 {
		return nil
	}
	return []Node{NewAtRule(node.Name, node.Params, children...)}
}
