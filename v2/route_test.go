package reactea_test

import (
	"reflect"
	"testing"

	"github.com/Hayao0819/reactea/v2"
)

func TestResolve(t *testing.T) {
	cases := []struct {
		base, target, want string
	}{
		{"/a/b", "/c", "/c"},
		{"/a/b", "c", "/a/b/c"},
		{"/a/b", "..", "/a"},
		{"/a/b", "../..", "/"},
		{"/a/b", "../../..", "/"},
		{"/a/b", ".", "/a/b"},
		{"/a/b", "", "/a/b"},
		{"/", "x", "/x"},
		{"nonsense", "x", "/nonsense/x"},
	}

	for _, tc := range cases {
		if got := reactea.Resolve(tc.base, tc.target); got != tc.want {
			t.Errorf("Resolve(%q, %q) = %q, want %q", tc.base, tc.target, got, tc.want)
		}
	}
}

func TestMatchRoute(t *testing.T) {
	cases := []struct {
		route, placeholder string
		want               map[string]string
		ok                 bool
	}{
		{"/", "/", map[string]string{"$": "/"}, true},
		{"/a", "/a", map[string]string{"$": "/a"}, true},
		{"/a", "/b", nil, false},
		{"/teams/9", "/teams/:id", map[string]string{"$": "/teams/9", "id": "9"}, true},
		{"/teams/9/1", "/teams/:id", nil, false},
		{"/teams", "/teams/?:id", map[string]string{"$": "/teams", "id": ""}, true},
		{"/teams/9", "/teams/?:id", map[string]string{"$": "/teams/9", "id": "9"}, true},
		{"/a/b/c", "/a/+?:rest", map[string]string{"$": "/a/b/c", "rest": "b/c"}, true},
		{"/a", "/a/+?:rest", map[string]string{"$": "/a", "rest": ""}, true},
		{"/a/b", "/a/:", map[string]string{"$": "/a/b"}, true},
		{"a", "/a", nil, false},
		{"/a", "a", nil, false},
	}

	for _, tc := range cases {
		got, ok := reactea.MatchRoute(tc.route, tc.placeholder)

		if ok != tc.ok {
			t.Errorf("MatchRoute(%q, %q) ok = %v, want %v", tc.route, tc.placeholder, ok, tc.ok)

			continue
		}

		if ok && !reflect.DeepEqual(got, tc.want) {
			t.Errorf("MatchRoute(%q, %q) = %v, want %v", tc.route, tc.placeholder, got, tc.want)
		}
	}
}
