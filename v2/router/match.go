package router

import "strings"

// Params contains the values captured while matching a route.
type Params struct {
	path   string
	values map[string]string
}

// Get returns a captured parameter, or an empty string when it was absent.
func (p Params) Get(name string) string { return p.values[name] }

// Path returns the complete route that matched.
func (p Params) Path() string { return p.path }

// Match reports whether path matches pattern and returns its captured params.
// Named, optional, and trailing catch-all segments use :id, :id?, and *rest.
func Match(path, pattern string) (Params, bool) {
	if !strings.HasPrefix(path, "/") || !strings.HasPrefix(pattern, "/") {
		return Params{}, false
	}

	pathLevels := levels(path)
	patternLevels := levels(pattern)
	values, ok := matchLevels(pathLevels, patternLevels, 0, 0, nil)
	if !ok {
		return Params{}, false
	}

	return Params{path: path, values: values}, true
}

func matchLevels(path, pattern []string, atPath, atPattern int, values map[string]string) (map[string]string, bool) {
	if atPattern == len(pattern) {
		return values, atPath == len(path)
	}

	segment := pattern[atPattern]
	if strings.HasPrefix(segment, "*") {
		if atPattern != len(pattern)-1 {
			return nil, false
		}

		rest := ""
		if atPath < len(path) {
			rest = strings.Join(path[atPath:], "/")
		}

		return withParam(values, segment[1:], rest), true
	}

	if strings.HasPrefix(segment, ":") {
		optional := strings.HasSuffix(segment, "?")
		name := strings.TrimSuffix(segment[1:], "?")

		if atPath < len(path) {
			withValue := withParam(values, name, path[atPath])
			if matched, ok := matchLevels(path, pattern, atPath+1, atPattern+1, withValue); ok {
				return matched, true
			}
		}

		if optional {
			return matchLevels(path, pattern, atPath, atPattern+1, withParam(values, name, ""))
		}

		return nil, false
	}

	if atPath >= len(path) || segment != path[atPath] {
		return nil, false
	}

	return matchLevels(path, pattern, atPath+1, atPattern+1, values)
}

func withParam(values map[string]string, name, value string) map[string]string {
	copy := make(map[string]string, len(values)+1)
	for key, existing := range values {
		copy[key] = existing
	}

	if name != "" {
		copy[name] = value
	}

	return copy
}

func levels(path string) []string {
	trimmed := strings.TrimPrefix(path, "/")
	if trimmed == "" {
		return nil
	}

	return strings.Split(trimmed, "/")
}
