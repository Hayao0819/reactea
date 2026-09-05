package router_test

import (
	"testing"

	"github.com/Hayao0819/reactea/v2/router"
)

func TestMatch(t *testing.T) {
	tests := []struct {
		path, pattern string
		values        map[string]string
		ok            bool
	}{
		{"/", "/", nil, true},
		{"/a", "/a", nil, true},
		{"/a", "/b", nil, false},
		{"/teams/9", "/teams/:id", map[string]string{"id": "9"}, true},
		{"/teams/9/1", "/teams/:id", nil, false},
		{"/teams", "/teams/:id?", map[string]string{"id": ""}, true},
		{"/teams/9", "/teams/:id?", map[string]string{"id": "9"}, true},
		{"/teams/edit", "/teams/:id?/edit", map[string]string{"id": ""}, true},
		{"/teams/9/edit", "/teams/:id?/edit", map[string]string{"id": "9"}, true},
		{"/a/b/c", "/a/*rest", map[string]string{"rest": "b/c"}, true},
		{"/a", "/a/*rest", map[string]string{"rest": ""}, true},
		{"/a/b", "/a/:id", map[string]string{"id": "b"}, true},
		{"a", "/a", nil, false},
		{"/a", "a", nil, false},
		{"/a/b", "/a/*rest/more", nil, false},
	}

	for _, test := range tests {
		params, ok := router.Match(test.path, test.pattern)
		if ok != test.ok {
			t.Errorf("Match(%q, %q) ok = %v, want %v", test.path, test.pattern, ok, test.ok)

			continue
		}

		if !ok {
			continue
		}

		if params.Path() != test.path {
			t.Errorf("Match(%q, %q).Path() = %q", test.path, test.pattern, params.Path())
		}

		for name, want := range test.values {
			if got := params.Get(name); got != want {
				t.Errorf("Match(%q, %q).Get(%q) = %q, want %q", test.path, test.pattern, name, got, want)
			}
		}
	}
}
