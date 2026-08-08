package router

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/Hayao0819/reactea/v2"
)

type testComponenent struct {
	reactea.BasicComponent

	router *Component

	testUpdater func(*testComponenent) tea.Cmd

	updateN int

	// rendered holds the most recent Render output. v2's renderer emits ANSI
	// diff sequences to a buffer, so we observe the routed component's output
	// here instead of scraping the terminal bytes.
	rendered string
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
	c.rendered = c.router.Render(width, height)
	return c.rendered
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

	if !strings.Contains(root.rendered, "Hello Default!") {
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

	if strings.Contains(root.rendered, "Hello Default!") {
		t.Fatalf("got default route message")
	}

	if !strings.Contains(root.rendered, "Hello Tests!") {
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

	if strings.Contains(root.rendered, "Hello Default!") {
		t.Fatalf("got default route message")
	}

	if !strings.Contains(root.rendered, "Hello Tests!") {
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

	if !strings.Contains(root.rendered, "Couldn't route for") {
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

	if strings.Contains(root.rendered, "Hello Default!") {
		t.Fatalf("got default route message")
	}

	if !strings.Contains(root.rendered, "Hello Tests!") {
		t.Fatalf("got invalid route message")
	}

	if !strings.Contains(root.rendered, "Hello Tests! Param foo is wellDone") {
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

		if !strings.Contains(root.rendered, "EXACT") {
			t.Fatalf("iteration %d: expected the exact route to win, got %q", i, root.rendered)
		}

		if strings.Contains(root.rendered, "PARAM") {
			t.Fatalf("iteration %d: param route was selected over the exact one, got %q", i, root.rendered)
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

	if !strings.Contains(root.rendered, "Couldn't route for \"/nowhere\"") {
		t.Fatalf("expected fallback render after a failed re-route, got %q", root.rendered)
	}
}

type destroyTrackingComponent struct {
	reactea.BasicComponent

	destroyed *bool
}

func (c *destroyTrackingComponent) Render(int, int) string { return "PAGE" }
func (c *destroyTrackingComponent) Destroy()               { *c.destroyed = true }

// A root component the way an app would write one: it owns a router and hands
// every lifecycle method down to it.
type destroyForwardingComponent struct {
	router *Component
}

func (c *destroyForwardingComponent) Init() tea.Cmd { return c.router.Init() }
func (c *destroyForwardingComponent) Destroy()      { c.router.Destroy() }

func (c *destroyForwardingComponent) Update(msg tea.Msg) tea.Cmd {
	return tea.Batch(c.router.Update(msg), reactea.Destroy)
}

func (c *destroyForwardingComponent) Render(width, height int) string {
	return c.router.Render(width, height)
}

// Tearing the app down has to reach the routed component too, not stop at the
// router.
func TestDestroyReachesRoutedComponent(t *testing.T) {
	var in, out bytes.Buffer

	in.WriteString("123")

	destroyed := false

	router := NewWithRoutes(map[string]RouteInitializer{
		"default": func(Params) reactea.Component {
			return &destroyTrackingComponent{destroyed: &destroyed}
		},
	})

	root := &destroyForwardingComponent{router: router}

	program := reactea.NewProgram(root, tea.WithInput(&in), tea.WithOutput(&out))

	if _, err := program.Run(); err != nil {
		t.Fatal(err)
	}

	if !destroyed {
		t.Fatal("routed component was not destroyed on teardown")
	}
}
