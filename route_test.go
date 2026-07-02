package reactea

import (
	"reflect"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestRoutePanic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic, but it didn't")
		} else {
			if r != "tried updating global route not in update" {
				t.Errorf("expected panic, got it, but with invalid message, got \"%s\", expected \"tried updating global route not in update\"", r)
			}
		}
	}()

	root := &mockComponent[struct{}]{}

	NewProgram(root, tea.WithoutRenderer())

	SetRoute("/shouldFail")
}

// Expecting:
// / -> /foo -> /foo/bar -> /baz -> / -> /foo -> /bar -> /test -> / -> / -> /foo -> /bar
func TestNavigate(t *testing.T) {
	type testState struct {
		routeHistory []string
		step         int
	}

	root := &mockComponent[testState]{
		initFunc: func(Component, *testState) tea.Cmd {
			return Rerender
		},
		updateFunc: func(c Component, s *testState, msg tea.Msg) tea.Cmd {
			switch s.step {
			case 0:
				// Don't navigate
			case 1:
				Navigate("foo")
			case 2:
				Navigate("foo/bar")
			case 3:
				Navigate("/baz")
			case 4:
				Navigate("..")
			case 5:
				Navigate("./foo")
			case 6:
				Navigate(".//bar")
			case 7:
				Navigate("../../test")
			case 8:
				Navigate("/")
			case 9:
				Navigate("")
			case 10:
				Navigate(".")
			case 11:
				Navigate("foo")
			case 12:
				Navigate("foo/bar")
			case 13:
				Navigate("baz")
			default:
				return Destroy
			}

			s.routeHistory = append(s.routeHistory, CurrentRoute())

			s.step += 1

			return Rerender
		},
	}

	program := NewProgram(root, tea.WithoutRenderer(), WithoutInput())

	if _, err := program.Run(); err != nil {
		t.Fatal(err)
	}

	expectedRouteHistory := []string{
		"/",
		"/foo",
		"/foo/bar",
		"/baz",
		"/",
		"/foo",
		"/bar",
		"/test",
		"/",
		"/",
		"/",
		"/foo",
		"/foo/bar",
		"/foo/baz",
	}

	if strings.Join(root.state.routeHistory, " - ") != strings.Join(expectedRouteHistory, " - ") {
		t.Errorf("wrong route history, expected \"%s\", got \"%s\". Note that routes are delimited by \" - \"", strings.Join(expectedRouteHistory, " - "), strings.Join(root.state.routeHistory, " - "))
	}
}

// A change followed by a revert within a single Update must not report a net
// route change, otherwise the router would needlessly Destroy and re-init the
// current component.
func TestSetRouteRevertWithinUpdate(t *testing.T) {
	isUpdate = true // SetRoute panics outside an update; emulate being inside one

	defer func() {
		isUpdate = false
		currentRoute = "/"
		lastRoute = "/"
		wasRouteChanged = false
	}()

	currentRoute = "/"
	lastRoute = "/"
	wasRouteChanged = false

	SetRoute("/a")
	if !WasRouteChanged() {
		t.Fatalf("expected a route change after SetRoute(\"/a\")")
	}
	if CurrentRoute() != "/a" {
		t.Fatalf("expected current route \"/a\", got %q", CurrentRoute())
	}
	if LastRoute() != "/" {
		t.Fatalf("expected last route \"/\", got %q", LastRoute())
	}

	// Revert to the original route: the net change is nothing.
	SetRoute("/")
	if WasRouteChanged() {
		t.Errorf("change-then-revert within one update must not report a change")
	}
	if CurrentRoute() != "/" {
		t.Errorf("expected current route \"/\", got %q", CurrentRoute())
	}

	// A genuine subsequent change still registers, still relative to the
	// pre-update route.
	SetRoute("/b")
	if !WasRouteChanged() {
		t.Errorf("expected a route change after SetRoute(\"/b\")")
	}
	if LastRoute() != "/" {
		t.Errorf("expected last route to stay \"/\", got %q", LastRoute())
	}
}

// Navigate must not panic slicing currentRoute even if currentRoute is somehow
// malformed (not root-prefixed).
func TestNavigateMalformedCurrentRoute(t *testing.T) {
	isUpdate = true

	defer func() {
		isUpdate = false
		currentRoute = "/"
		lastRoute = "/"
		wasRouteChanged = false
	}()

	currentRoute = "malformed"
	lastRoute = "malformed"
	wasRouteChanged = false

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Navigate panicked on malformed currentRoute: %v", r)
		}
	}()

	Navigate("foo")

	if CurrentRoute() != "/foo" {
		t.Errorf("expected current route \"/foo\", got %q", CurrentRoute())
	}
}

func TestRoutePlaceholderMatching(t *testing.T) {
	testCases := []struct {
		route, placeholder string
		expected           map[string]string
	}{
		// Matching against non-root routes is forbidden
		{"", "", nil},
		{"invalidRoute", "", nil},
		{"", "invalidPlaceholder", nil},
		{"/invalidRoute", "invalidPlaceholder", nil},

		{"/teams/foo", "/teams", nil},
		{"/teams", "/teams/foo", nil},
		{"/", "/teams", nil},
		{"/teams", "/", nil},
		{"/teams", "/teams", map[string]string{"$": "/teams"}},

		{"/teams", "/teams/?:", map[string]string{"$": "/teams"}},
		{"/teams/123", "/teams/?:", map[string]string{"$": "/teams/123"}},
		{"/teams/123/456", "/teams/?:", nil},
		{"/teams", "/teams/?:teamId", map[string]string{"$": "/teams", "teamId": ""}},
		{"/teams/123", "/teams/?:teamId", map[string]string{"$": "/teams/123", "teamId": "123"}},
		{"/teams/123/456", "/teams/?:teamId", nil},

		{"/teams/123/456", "/teams/123/456/+?:foo", map[string]string{"$": "/teams/123/456", "foo": ""}},
		{"/teams/123/456", "/teams/+?:foo", map[string]string{"$": "/teams/123/456", "foo": "123/456"}},
		{"/teams/123/456", "/teams/+?:", map[string]string{"$": "/teams/123/456"}},
		{"/teams/123", "/teams/+?:", map[string]string{"$": "/teams/123"}},
		{"/teams", "/teams/+?:", map[string]string{"$": "/teams"}},

		{"/teams/123", "/teams/:teamId", map[string]string{"$": "/teams/123", "teamId": "123"}},
		{"/teams/foo/234", "/teams/:/:teamId", map[string]string{"$": "/teams/foo/234", "teamId": "234"}},
		{"/teams/123/234", "/teams/:teamId/:teamId", map[string]string{"$": "/teams/123/234", "teamId": "234"}},
		{"/teams/123/234", "/teams/:teamId/:playerId", map[string]string{"$": "/teams/123/234", "teamId": "123", "playerId": "234"}},

		{"/detail/abcgsd-dsfhh2342-sdfhs-234", "/detail/:id", map[string]string{"$": "/detail/abcgsd-dsfhh2342-sdfhs-234", "id": "abcgsd-dsfhh2342-sdfhs-234"}},
	}

	for _, testCase := range testCases {
		got, ok := RouteMatchesPlaceholder(testCase.route, testCase.placeholder)

		if testCase.expected == nil {
			if !ok {
				continue
			}

			t.Errorf("Bad result. Route: \"%s\", Placeholder: \"%s\". Expected not ok, got ok", testCase.route, testCase.placeholder)
			continue
		}

		if !reflect.DeepEqual(got, testCase.expected) {
			t.Errorf("Bad result. Route: \"%s\", Placeholder: \"%s\". Expected %v, got %v", testCase.route, testCase.placeholder, testCase.expected, got)
		}
	}
}
