package twill

// CompileOptions configure Compile.
type CompileOptions struct {
	// Base is the directory for resolving @source and relative imports.
	Base string
	// LoadStylesheet loads a stylesheet by id. When nil, only the built-in
	// stylesheets can be imported.
	LoadStylesheet StylesheetLoader
}

// Compiler is the handle returned by Compile (SPEC §4.1.11).
type Compiler struct {
	Sources      []SourceEntry
	Root         *SourceRoot
	Features     Features
	DesignSystem *DesignSystem

	css             string
	ast             []Node
	utilitiesNode   *Context
	validCandidates map[string]bool
	candidateOrder  []string
	pendingInline   bool
	cached          *string
	previousCount   int
}

// Compile compiles a stylesheet (SPEC §15.1).
func Compile(css string, options CompileOptions) (*Compiler, error) {
	load := options.LoadStylesheet
	if load == nil {
		load = BuiltinLoader
	}
	parsed, err := Parse(css)
	if err != nil {
		return nil, err
	}
	ast := []Node{NewContext(ContextMap{"base": options.Base}, parsed...)}

	features, err := SubstituteAtImports(&ast, options.Base, load)
	if err != nil {
		return nil, err
	}

	theme := NewTheme()
	state, err := CollectDirectives(&ast, theme)
	if err != nil {
		return nil, err
	}
	features |= state.Features

	ds := BuildDesignSystem(theme)
	ds.Important = state.Important
	for _, c := range state.IgnoredCandidates {
		ds.InvalidCandidates[c] = true
	}

	customNames := map[string]bool{}
	byName := map[string]*CustomVariant{}
	var order []string
	for _, v := range state.CustomVariants {
		customNames[v.Name] = true
		byName[v.Name] = v
		order = append(order, v.Name)
		ds.Variants.Static(v.Name, func(Node, *Variant) bool { return true }, CompoundsStyleRules)
	}
	dependencies := map[string]map[string]bool{}
	for _, v := range state.CustomVariants {
		deps := map[string]bool{}
		for dep := range v.Dependencies {
			if dep == v.Name {
				return nil, Errorf("Custom variant `%s` depends on itself, creating a circular dependency.", v.Name)
			}
			if customNames[dep] {
				deps[dep] = true
			}
		}
		dependencies[v.Name] = deps
	}
	sorted, err := TopologicalSort(order, dependencies, func(cycle string) error {
		return Errorf("Custom variant `%s` is part of a circular dependency between custom variants.", cycle)
	})
	if err != nil {
		return nil, err
	}
	for _, name := range sorted {
		byName[name].Register(ds)
	}
	for _, u := range state.CustomUtilities {
		u.Register(ds)
	}

	if state.FirstThemeRule != nil {
		var declarations []Node
		for _, key := range theme.Keys() {
			entry := theme.Entry(key)
			if entry.Options&Reference != 0 {
				continue
			}
			declarations = append(declarations, Decl(Escape(theme.PrefixKey(key)), entry.Value))
		}
		state.FirstThemeRule.Nodes = []Node{NewContext(ContextMap{"theme": "true"}, declarations...)}
	}
	for _, k := range theme.Keyframes() {
		ast = append(ast, NewContext(ContextMap{"theme": "true"}, NewAtRoot(k)))
	}

	f, err := SubstituteAtVariant(&ast, ds)
	if err != nil {
		return nil, err
	}
	features |= f
	f, err = SubstituteFunctions(&ast, ds)
	if err != nil {
		return nil, err
	}
	features |= f
	f, err = SubstituteAtApply(&ast, ds)
	if err != nil {
		return nil, err
	}
	features |= f

	var utilitiesNode *Context
	if state.UtilitiesNode != nil {
		utilitiesNode = NewContext(ContextMap{})
		target := Node(state.UtilitiesNode)
		Walk(&ast, func(n Node, u *WalkUtils) WalkAction {
			if n == target {
				u.ReplaceWith(utilitiesNode)
				return Stop
			}
			return Continue
		})
	}
	Walk(&ast, func(n Node, u *WalkUtils) WalkAction {
		if at, ok := n.(*AtRule); ok && at.Name == "@utility" {
			u.ReplaceWith()
			return Skip
		}
		return Continue
	})

	c := &Compiler{
		Sources: state.Sources, Root: state.Root, Features: features, DesignSystem: ds,
		css: css, ast: ast, utilitiesNode: utilitiesNode,
		validCandidates: map[string]bool{}, pendingInline: len(state.InlineCandidates) > 0, previousCount: -1,
	}
	for _, inline := range state.InlineCandidates {
		if !c.validCandidates[inline] {
			c.validCandidates[inline] = true
			c.candidateOrder = append(c.candidateOrder, inline)
		}
	}
	return c, nil
}

// Build builds the CSS for the accumulated candidates (SPEC §15.2).
func (c *Compiler) Build(candidates []string) string {
	if c.Features == 0 {
		return c.css
	}
	ds := c.DesignSystem
	if c.utilitiesNode == nil {
		if c.cached == nil {
			out := Serialize(OptimizeAst(c.ast, ds))
			c.cached = &out
		}
		return *c.cached
	}

	changed := c.pendingInline
	c.pendingInline = false
	markedVariable := false
	for _, candidate := range candidates {
		if ds.InvalidCandidates[candidate] {
			continue
		}
		if len(candidate) >= 2 && candidate[:2] == "--" {
			if ds.Theme.MarkUsedVariable(candidate) {
				changed = true
				markedVariable = true
			}
			continue
		}
		if !c.validCandidates[candidate] {
			c.validCandidates[candidate] = true
			c.candidateOrder = append(c.candidateOrder, candidate)
			changed = true
		}
	}
	if !changed && c.cached != nil {
		return *c.cached
	}

	raws := make([]string, 0, len(c.candidateOrder))
	for _, raw := range c.candidateOrder {
		if c.validCandidates[raw] {
			raws = append(raws, raw)
		}
	}
	nodes := CompileCandidates(raws, ds, CompileCandidatesOptions{
		OnInvalidCandidate: func(candidate string) {
			ds.InvalidCandidates[candidate] = true
			delete(c.validCandidates, candidate)
		},
	})
	if len(nodes) == c.previousCount && !markedVariable && c.cached != nil {
		return *c.cached
	}
	c.previousCount = len(nodes)
	c.utilitiesNode.Nodes = make([]Node, len(nodes))
	for i, n := range nodes {
		c.utilitiesNode.Nodes[i] = n
	}
	out := Serialize(OptimizeAst(c.ast, ds))
	c.cached = &out
	return out
}
