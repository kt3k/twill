package twill

import (
	"regexp"
	"strconv"
	"strings"
)

// ThemeOptions are the theme entry option bits (SPEC §4.1.2).
type ThemeOptions int

const (
	// Inline: consumers embed the raw value instead of var(...).
	Inline ThemeOptions = 1 << iota
	// Reference: the variable is not printed; consumers embed var(key, value).
	Reference
	// Default: a later non-default entry with the same key wins.
	Default
	// Static: the variable is printed even when unused.
	Static
	// Used: something referenced the variable.
	Used
)

// ThemeEntry is a stored theme value.
type ThemeEntry struct {
	Value   string
	Options ThemeOptions
}

// Sub-namespaces that namespace clearing and resolution skip (SPEC §7.2.1).
var ignoredNamespaces = map[string][]string{
	"--font":        {"--font-weight", "--font-size"},
	"--inset":       {"--inset-shadow", "--inset-ring"},
	"--text":        {"--text-color", "--text-decoration-color", "--text-decoration-thickness", "--text-indent", "--text-shadow", "--text-underline-offset"},
	"--grid-column": {"--grid-column-start", "--grid-column-end"},
	"--grid-row":    {"--grid-row-start", "--grid-row-end"},
}

func isIgnoredKey(namespace, key string) bool {
	for _, ns := range ignoredNamespaces[namespace] {
		if key == ns || strings.HasPrefix(key, ns+"-") {
			return true
		}
	}
	return false
}

// Theme stores design tokens (SPEC §7).
type Theme struct {
	keys      []string
	values    map[string]*ThemeEntry
	keyframes []*AtRule
	Prefix    string
}

// NewTheme creates an empty theme.
func NewTheme() *Theme {
	return &Theme{values: map[string]*ThemeEntry{}}
}

// Size is the number of stored entries.
func (t *Theme) Size() int { return len(t.values) }

func (t *Theme) delete(key string) {
	if _, ok := t.values[key]; !ok {
		return
	}
	delete(t.values, key)
	for i, k := range t.keys {
		if k == key {
			t.keys = append(t.keys[:i], t.keys[i+1:]...)
			break
		}
	}
}

// Add registers a value (SPEC §7.2).
func (t *Theme) Add(key, value string, options ThemeOptions) error {
	if strings.HasSuffix(key, "-*") {
		if value != "initial" {
			return Errorf("Invalid theme value `%s` for namespace `%s`", value, key)
		}
		if key == "--*" {
			t.keys = nil
			t.values = map[string]*ThemeEntry{}
		} else {
			t.clearNamespace(key[:len(key)-2])
		}
		return nil
	}
	if options&Default != 0 {
		if existing, ok := t.values[key]; ok && existing.Options&Default == 0 {
			return nil
		}
	}
	if value == "initial" {
		t.delete(key)
		return nil
	}
	if existing, ok := t.values[key]; ok {
		existing.Value = value
		existing.Options = options
		return nil
	}
	t.keys = append(t.keys, key)
	t.values[key] = &ThemeEntry{Value: value, Options: options}
	return nil
}

func (t *Theme) clearNamespace(namespace string) {
	for _, key := range append([]string{}, t.keys...) {
		if strings.HasPrefix(key, namespace) && !isIgnoredKey(namespace, key) {
			t.delete(key)
		}
	}
}

// AddKeyframes stores a @keyframes block.
func (t *Theme) AddKeyframes(node *AtRule) {
	for _, k := range t.keyframes {
		if k == node {
			return
		}
	}
	t.keyframes = append(t.keyframes, node)
}

// Keyframes returns the stored @keyframes blocks.
func (t *Theme) Keyframes() []*AtRule { return append([]*AtRule{}, t.keyframes...) }

// Keys returns every key in insertion order.
func (t *Theme) Keys() []string { return append([]string{}, t.keys...) }

// Entry returns the entry for key, or nil.
func (t *Theme) Entry(key string) *ThemeEntry { return t.values[key] }

// Has reports whether key exists.
func (t *Theme) Has(key string) bool { _, ok := t.values[key]; return ok }

// Options returns the options of key, or 0.
func (t *Theme) Options(key string) ThemeOptions {
	if e, ok := t.values[key]; ok {
		return e.Options
	}
	return 0
}

// ResolveKey finds the key for a candidate value in the namespaces. A nil
// candidateValue resolves the namespace itself.
func (t *Theme) ResolveKey(candidateValue *string, namespaces []string) (string, bool) {
	for _, ns := range namespaces {
		var key string
		if candidateValue == nil {
			key = ns
		} else {
			key = ns + "-" + *candidateValue
		}
		if !t.Has(key) {
			if candidateValue != nil && strings.Contains(*candidateValue, ".") {
				key = ns + "-" + strings.ReplaceAll(*candidateValue, ".", "_")
				if !t.Has(key) {
					continue
				}
			} else {
				continue
			}
		}
		if isIgnoredKey(ns, key) {
			continue
		}
		return key, true
	}
	return "", false
}

func (t *Theme) reference(key string, options ThemeOptions) string {
	entry := t.values[key]
	if (options|entry.Options)&Inline != 0 {
		return entry.Value
	}
	name := Escape(t.PrefixKey(key))
	if entry.Options&Reference != 0 {
		return "var(" + name + ", " + entry.Value + ")"
	}
	return "var(" + name + ")"
}

// Resolve returns a var(...) reference or inline value for a candidate value.
func (t *Theme) Resolve(candidateValue *string, namespaces []string, options ThemeOptions) (string, bool) {
	key, ok := t.ResolveKey(candidateValue, namespaces)
	if !ok {
		return "", false
	}
	return t.reference(key, options), true
}

// ResolveValue returns the raw value for a candidate value.
func (t *Theme) ResolveValue(candidateValue *string, namespaces []string) (string, bool) {
	key, ok := t.ResolveKey(candidateValue, namespaces)
	if !ok {
		return "", false
	}
	return t.values[key].Value, true
}

// ResolveWith resolves a key and its nested sub-keys.
func (t *Theme) ResolveWith(candidateValue *string, namespaces []string, nestedKeys []string) (string, map[string]string, bool) {
	key, ok := t.ResolveKey(candidateValue, namespaces)
	if !ok {
		return "", nil, false
	}
	extra := map[string]string{}
	for _, nested := range nestedKeys {
		nestedKey := key + nested
		if t.Has(nestedKey) {
			extra[nested] = t.reference(nestedKey, 0)
		}
	}
	return t.reference(key, 0), extra, true
}

// Get returns the raw value of the first key that exists.
func (t *Theme) Get(keys ...string) (string, bool) {
	for _, key := range keys {
		if e, ok := t.values[key]; ok {
			return e.Value, true
		}
	}
	return "", false
}

// NamespaceEntry is an entry of Namespace; Key is "" for the namespace itself.
type NamespaceEntry struct {
	Key   string
	Self  bool
	Value string
}

// Namespace returns the entries under ns: the namespace itself (Self), keys
// with `<ns>-` removed, and keys starting with `<ns>--` with only `<ns>`
// removed.
func (t *Theme) Namespace(ns string) []NamespaceEntry {
	var result []NamespaceEntry
	prefix := ns + "-"
	for _, key := range t.keys {
		entry := t.values[key]
		switch {
		case key == ns:
			result = append(result, NamespaceEntry{Self: true, Value: entry.Value})
		case strings.HasPrefix(key, ns+"--"):
			result = append(result, NamespaceEntry{Key: key[len(ns):], Value: entry.Value})
		case strings.HasPrefix(key, prefix):
			result = append(result, NamespaceEntry{Key: key[len(prefix):], Value: entry.Value})
		}
	}
	return result
}

// KeysInNamespaces returns every key under each namespace with the prefix
// removed, excluding keys with a second `--` and ignored keys.
func (t *Theme) KeysInNamespaces(namespaces []string) []string {
	var keys []string
	for _, ns := range namespaces {
		prefix := ns + "-"
		for _, key := range t.keys {
			if !strings.HasPrefix(key, prefix) {
				continue
			}
			if strings.Contains(key[2:], "--") {
				continue
			}
			if isIgnoredKey(ns, key) {
				continue
			}
			keys = append(keys, key[len(prefix):])
		}
	}
	return keys
}

// PrefixKey applies the theme prefix to key.
func (t *Theme) PrefixKey(key string) string {
	if t.Prefix == "" {
		return key
	}
	return "--" + t.Prefix + "-" + key[2:]
}

// UnprefixKey removes the theme prefix from key.
func (t *Theme) UnprefixKey(key string) string {
	if t.Prefix == "" {
		return key
	}
	prefix := "--" + t.Prefix + "-"
	if strings.HasPrefix(key, prefix) {
		return "--" + key[len(prefix):]
	}
	return key
}

// MarkUsedVariable sets Used on the unprefixed, unescaped key. Returns true
// when the flag was not set before.
func (t *Theme) MarkUsedVariable(key string) bool {
	entry, ok := t.values[t.UnprefixKey(Unescape(key))]
	if !ok {
		return false
	}
	if entry.Options&Used != 0 {
		return false
	}
	entry.Options |= Used
	return true
}

// ResolveThemeValue resolves a path such as `--color-red-500/50`.
func (t *Theme) ResolveThemeValue(path string, forceInline bool) (string, bool) {
	trimmed := strings.TrimSpace(path)
	key := trimmed
	modifier := ""
	hasModifier := false
	if slash := strings.LastIndex(trimmed, "/"); slash != -1 {
		key = strings.TrimSpace(trimmed[:slash])
		modifier = strings.TrimSpace(trimmed[slash+1:])
		hasModifier = true
	}
	var options ThemeOptions
	if forceInline {
		options = Inline
	}
	value, ok := t.Resolve(nil, []string{key}, options)
	if !ok {
		return "", false
	}
	if hasModifier {
		return WithAlpha(value, modifier), true
	}
	return value, true
}

var numberPattern = regexp.MustCompile(`^[+-]?(\d+\.?\d*|\.\d+)([eE][+-]?\d+)?$`)

// WithAlpha applies an alpha value to a color (SPEC §10.3).
func WithAlpha(color, alpha string) string {
	if alpha == "" {
		return color
	}
	if numberPattern.MatchString(alpha) {
		n, err := strconv.ParseFloat(alpha, 64)
		if err == nil {
			alpha = strconv.FormatFloat(n*100, 'f', -1, 64) + "%"
		}
	}
	if alpha == "100%" {
		return color
	}
	return "color-mix(in oklab, " + color + " " + alpha + ", transparent)"
}

// Features are the feature flags reported by compile (SPEC §4.1.11).
type Features int

const (
	// FeatureAtApply is set when @apply was expanded.
	FeatureAtApply Features = 1 << 0
	// FeatureAtImport is set when an import was resolved.
	FeatureAtImport Features = 1 << 1
	// FeatureThemeFunction is set when a theme function was substituted.
	FeatureThemeFunction Features = 1 << 3
	// FeatureUtilities is set when @twill utilities was found.
	FeatureUtilities Features = 1 << 4
	// FeatureVariants is set when @variant was expanded.
	FeatureVariants Features = 1 << 5
	// FeatureAtTheme is set when @theme was found.
	FeatureAtTheme Features = 1 << 6
)
