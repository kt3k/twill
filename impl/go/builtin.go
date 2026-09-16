package twill

import (
	"embed"
	"strings"
)

// The built-in stylesheets (SPEC §6.3) as embedded resources. theme.css and
// preflight.css are ported from Tailwind CSS (MIT License, Copyright (c)
// Tailwind Labs, Inc.); see the header of each file.
//
//go:embed css/index.css css/theme.css css/preflight.css css/utilities.css
var builtinFS embed.FS

// BuiltinBase is the base of every built-in stylesheet. It is distinct from
// any user directory, so a relative import from user CSS never resolves to
// an embedded resource.
const BuiltinBase = "twill:"

// LoadedStylesheet is the result of loading a stylesheet (SPEC §6.2).
type LoadedStylesheet struct {
	Path    string
	Base    string
	Content string
}

// StylesheetLoader loads a stylesheet by id; relative ids resolve against base.
type StylesheetLoader func(id, base string) (*LoadedStylesheet, error)

// IsBuiltinPath reports whether path names an embedded resource.
func IsBuiltinPath(path string) bool { return strings.HasPrefix(path, BuiltinBase) }

// BuiltinCSS returns the content of a built-in stylesheet by file name.
func BuiltinCSS(name string) (string, bool) {
	data, err := builtinFS.ReadFile("css/" + name)
	if err != nil {
		return "", false
	}
	return string(data), true
}

// ResolveBuiltin resolves the id `twill`, the ids `twill/<name>`, and
// relative ids requested from inside a built-in stylesheet. It returns nil
// for any other request so that the host loader can fall through.
func ResolveBuiltin(id, base string) (*LoadedStylesheet, error) {
	var name string
	switch {
	case id == "twill":
		name = "index.css"
	case strings.HasPrefix(id, "twill/"):
		name = id[len("twill/"):]
	case base == BuiltinBase:
		name = strings.TrimPrefix(id, "./")
	default:
		return nil, nil
	}
	content, ok := BuiltinCSS(name)
	if !ok {
		return nil, Errorf("Unknown built-in stylesheet `%s`", id)
	}
	return &LoadedStylesheet{Path: BuiltinBase + name, Base: BuiltinBase, Content: content}, nil
}

// BuiltinLoader resolves only the built-in stylesheets.
func BuiltinLoader(id, base string) (*LoadedStylesheet, error) {
	loaded, err := ResolveBuiltin(id, base)
	if err != nil {
		return nil, err
	}
	if loaded == nil {
		return nil, Errorf("Cannot resolve `%s`: no stylesheet loader was provided", id)
	}
	return loaded, nil
}
