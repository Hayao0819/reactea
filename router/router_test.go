package router

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/Hayao0819/reactea"
	tea "github.com/charmbracelet/bubbletea"
)

type testComponenent struct {
	reactea.BasicComponent

	router *Component

	testUpdater func(*testComponenent) tea.Cmd

	updateN int
}

func (c *testComponenent) Init() tea.Cmd {
	return c.router.Init()
}

func (c *testComponenent) Update(msg tea.Msg) tea.Cmd {
	defer func() {
		c.updateN++
	}()

	if c.testUpdater != nil {
		return tea.Batch(c.router.Update(c.router.Update(msg)), c.testUpdater(c))
	}

	return tea.Batch(c.router.Update(msg), reactea.Destroy)
}

func (c *testComponenent) Render(width, height int) string {
	return c.router.Render(width, height)
}

func TestDefault(t *testing.T) {
	var in, out bytes.Buffer

	in.WriteString("123")

	root := &testComponenent{
		router: NewWithRoutes(map[string]RouteInitializer{
			"default": func(Params) reactea.Component {
				renderer := func() string {
					return "Hello Default!"
				}

				return reactea.ComponentifyDumb(renderer)
			},
		}),
	}

	program := reactea.NewProgram(root, tea.WithInput(&in), tea.WithOutput(&out))

	if _, err := program.Run(); err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(out.String(), "Hello Default!") {
		t.Fatalf("no default route message")
	}
}

func TestNonDefault(t *testing.T) {
	var in, out bytes.Buffer

	in.WriteString("123")

	root := &testComponenent{
		router: NewWithRoutes(map[string]RouteInitializer{
			"default": func(Params) reactea.Component {
				renderer := func() string {
					return "Hello Default!"
				}

				return reactea.ComponentifyDumb(renderer)
			},
			"/test/test": func(Params) reactea.Component {
				renderer := func() string {
					return "Hello Tests!"
				}

				return reactea.ComponentifyDumb(renderer)
			},
		}),
	}

	program := reactea.NewProgram(root, reactea.WithRoute("/test/test"), tea.WithInput(&in), tea.WithOutput(&out))

	if _, err := program.Run(); err != nil {
		t.Fatal(err)
	}

	if strings.Contains(out.String(), "Hello Default!") {
		t.Fatalf("got default route message")
	}

	if !strings.Contains(out.String(), "Hello Tests!") {
		t.Fatalf("got invalid route message")
	}
}

func TestRouteChange(t *testing.T) {
	var in, out bytes.Buffer

	in.WriteString("123")

	root := &testComponenent{
		testUpdater: func(c *testComponenent) tea.Cmd {
			if c.updateN == 0 {
				reactea.SetRoute("/test/test")

				return nil
			} else {
				return reactea.Destroy
			}
		},
		router: NewWithRoutes(map[string]RouteInitializer{
			"default": func(Params) reactea.Component {
				renderer := func() string {
					return "Hello Default!"
				}

				return reactea.ComponentifyDumb(renderer)
			},
			"/test/test": func(Params) reactea.Component {
				renderer := func() string {
					return "Hello Tests!"
				}

				return reactea.ComponentifyDumb(renderer)
			},
		}),
	}

	program := reactea.NewProgram(root, tea.WithInput(&in), tea.WithOutput(&out))

	if _, err := program.Run(); err != nil {
		t.Fatal(err)
	}

	if strings.Contains(out.String(), "Hello Default!") {
		t.Fatalf("got default route message")
	}

	if !strings.Contains(out.String(), "Hello Tests!") {
		t.Fatalf("got invalid route message")
	}
}

func TestNotFound(t *testing.T) {
	var in, out bytes.Buffer

	in.WriteString("123")

	root := &testComponenent{
		router: New(),
	}

	program := reactea.NewProgram(root, tea.WithInput(&in), tea.WithOutput(&out))

	if _, err := program.Run(); err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(out.String(), "Couldn't route for") {
		t.Fatalf("got invalid route message")
	}
}

func TestRouteWithParam(t *testing.T) {
	var in, out bytes.Buffer

	in.WriteString("123")

	root := &testComponenent{
		testUpdater: func(c *testComponenent) tea.Cmd {
			if c.updateN == 0 {
				reactea.SetRoute("/test/wellDone")

				return nil
			} else {
				return reactea.Destroy
			}
		},
		router: NewWithRoutes(map[string]RouteInitializer{
			"default": func(Params) reactea.Component {
				renderer := func() string {
					return "Hello Default!"
				}

				return reactea.ComponentifyDumb(renderer)
			},
			"/test/:foo": func(params Params) reactea.Component {
				renderer := func() string {
					return fmt.Sprintf("Hello Tests! Param foo is %s", params["foo"])
				}

				return reactea.ComponentifyDumb(renderer)
			},
		}),
	}

	program := reactea.NewProgram(root, tea.WithInput(&in), tea.WithOutput(&out))

	if _, err := program.Run(); err != nil {
		t.Fatal(err)
	}

	if strings.Contains(out.String(), "Hello Default!") {
		t.Fatalf("got default route message")
	}

	if !strings.Contains(out.String(), "Hello Tests!") {
		t.Fatalf("got invalid route message")
	}

	if !strings.Contains(out.String(), "Hello Tests! Param foo is wellDone") {
		t.Fatalf("got valid route message, but most likely wrong param")
	}
}

// When more than one placeholder matches, the most specific (literal over
// param) must win, deterministically across runs — not be picked at random by
// Go's map iteration order.
func TestRouteSpecificity(t *testing.T) {
	for i := 0; i < 50; i++ {
		var in, out bytes.Buffer

		in.WriteString("123")

		root := &testComponenent{
			router: NewWithRoutes(map[string]RouteInitializer{
				"/user/:id": func(Params) reactea.Component {
					return reactea.ComponentifyDumb(func() string { return "PARAM" })
				},
				"/user/settings": func(Params) reactea.Component {
					return reactea.ComponentifyDumb(func() string { return "EXACT" })
				},
			}),
		}

		program := reactea.NewProgram(root, reactea.WithRoute("/user/settings"), tea.WithInput(&in), tea.WithOutput(&out))

		if _, err := program.Run(); err != nil {
			t.Fatal(err)
		}

		if !strings.Contains(out.String(), "EXACT") {
			t.Fatalf("iteration %d: expected the exact route to win, got %q", i, out.String())
		}

		if strings.Contains(out.String(), "PARAM") {
			t.Fatalf("iteration %d: param route was selected over the exact one, got %q", i, out.String())
		}
	}
}

// Specificity precedence: literal > param (":") > optional ("?:") > catch-all
// ("+?:"); more segments beat fewer on an equal prefix; equal specificity is
// broken deterministically by placeholder string.
func TestRouteSpecificityRanking(t *testing.T) {
	cases := []struct {
		a, b string
		want bool // moreSpecific(a, b)
	}{
		{"/user/settings", "/user/:id", true}, // literal over param
		{"/user/:id", "/user/settings", false},
		{"/a/:x", "/a/?:x", true},   // param over optional
		{"/a/?:x", "/a/+?:x", true}, // optional over catch-all
		{"/a/b/c", "/a/b", true},    // more segments win
		{"/a/b", "/a/b/c", false},
		{"/a", "/b", true}, // tie broken by string order
		{"/b", "/a", false},
	}

	for _, tc := range cases {
		if got := moreSpecific(tc.a, tc.b); got != tc.want {
			t.Errorf("moreSpecific(%q, %q) = %v, want %v", tc.a, tc.b, got, tc.want)
		}
	}
}

// A re-route to an unmatched path (with no "default") must Destroy the current
// component and fall back to the "Couldn't route" render, not keep rendering
// the destroyed component.
func TestFailedRerouteFallsBack(t *testing.T) {
	var in, out bytes.Buffer

	in.WriteString("123")

	root := &testComponenent{
		testUpdater: func(c *testComponenent) tea.Cmd {
			if c.updateN == 0 {
				reactea.SetRoute("/nowhere")

				return nil
			}

			return reactea.Destroy
		},
		router: NewWithRoutes(map[string]RouteInitializer{
			"/start": func(Params) reactea.Component {
				return reactea.ComponentifyDumb(func() string { return "STARTED" })
			},
		}),
	}

	program := reactea.NewProgram(root, reactea.WithRoute("/start"), tea.WithInput(&in), tea.WithOutput(&out))

	if _, err := program.Run(); err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(out.String(), "Couldn't route for \"/nowhere\"") {
		t.Fatalf("expected fallback render after a failed re-route, got %q", out.String())
	}
}
