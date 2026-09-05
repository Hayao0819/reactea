package reactea

import (
	"testing"
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
		if got := resolveRoute(tc.base, tc.target); got != tc.want {
			t.Errorf("Resolve(%q, %q) = %q, want %q", tc.base, tc.target, got, tc.want)
		}
	}
}
