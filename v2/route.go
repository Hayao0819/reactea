package reactea

import "strings"

func resolveRoute(base, target string) string {
	if target == "" {
		return base
	}

	var levels []string

	if target[0] != '/' {
		levels = split(base)
	}

	for _, level := range strings.Split(target, "/") {
		switch level {
		case "", ".":
		case "..":
			if len(levels) > 0 {
				levels = levels[:len(levels)-1]
			}
		default:
			levels = append(levels, level)
		}
	}

	return "/" + strings.Join(levels, "/")
}

func split(route string) []string {
	levels := make([]string, 0, strings.Count(route, "/"))

	for _, level := range strings.Split(route, "/") {
		if level != "" {
			levels = append(levels, level)
		}
	}

	return levels
}
