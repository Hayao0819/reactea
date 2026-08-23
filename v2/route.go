package reactea

import "strings"

// Resolve turns target into an absolute route, honouring "." and "..".
func Resolve(base, target string) string {
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

// MatchRoute reports whether route (e.g. /teams/123/12) matches placeholder
// (e.g. /teams/:teamId/:playerId) and returns the params it captured.
//
// Params follow ^:.*$, where ^ is the start of a path level and $ its end.
//
//   - The whole matched route is available under the key "$".
//   - Placeholders can be optional: /foo/?:/?: matches /foo, /foo/bar and
//     /foo/bar/baz.
//   - A trailing placeholder can be an optional catch-all: /foo/+?: matches
//     /foo and everything below it.
//   - Wildcards are allowed: /foo/:/bar.
//   - A repeated param name keeps the value of its last occurrence.
func MatchRoute(route string, placeholder string) (map[string]string, bool) {
	var (
		routeLevels       = strings.Split(route, "/")
		placeholderLevels = strings.Split(placeholder, "/")
	)

	if len(route) > 0 && route[0] == '/' {
		routeLevels = routeLevels[1:]
	} else {
		// Checking against a non-root route is forbidden.
		return nil, false
	}

	if len(placeholder) > 0 && placeholder[0] == '/' {
		placeholderLevels = placeholderLevels[1:]
	} else {
		return nil, false
	}

	if len(routeLevels) > len(placeholderLevels) && !strings.HasPrefix(placeholderLevels[len(placeholderLevels)-1], "+?:") {
		return nil, false
	}

	params := make(map[string]string, len(placeholderLevels)+1)

	params["$"] = route

	for i, placeholderLevel := range placeholderLevels {
		if i > len(routeLevels)-1 {
			switch {
			case strings.HasPrefix(placeholderLevel, "+?:"):
				if placeholderLevel != "+?:" {
					params[placeholderLevel[3:]] = ""
				}

				return params, true

			case strings.HasPrefix(placeholderLevel, "?:"):
				if placeholderLevel != "?:" {
					params[placeholderLevel[2:]] = ""
				}

				continue

			default:
				return nil, false
			}
		}

		routeLevel := routeLevels[i]

		switch {
		case strings.HasPrefix(placeholderLevel, "+?:"):
			if placeholderLevel != "+?:" {
				params[placeholderLevel[3:]] = strings.Join(routeLevels[i:], "/")
			}

			return params, true

		case strings.HasPrefix(placeholderLevel, "?:"):
			if placeholderLevel != "?:" {
				params[placeholderLevel[2:]] = routeLevel
			}

		case strings.HasPrefix(placeholderLevel, ":"):
			if placeholderLevel != ":" {
				params[placeholderLevel[1:]] = routeLevel
			}

		default:
			if routeLevel != placeholderLevel {
				return nil, false
			}
		}
	}

	return params, true
}
