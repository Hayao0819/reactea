package router_test

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/Hayao0819/reactea/v2"
	"github.com/Hayao0819/reactea/v2/router"
)

type page struct {
	reactea.BasicComponent

	label     string
	destroyed bool
}

func (c *page) Render(*reactea.Ctx) string { return c.label }

func (c *page) Init(ctx *reactea.Ctx) tea.Cmd {
	ctx.OnDestroy(func() { c.destroyed = true })

	return nil
}

func pages(labels map[string]string) router.Routes {
	routes := router.Routes{}

	for placeholder, label := range labels {
		routes[placeholder] = func(router.Params) reactea.Component {
			return &page{label: label}
		}
	}

	return routes
}

func render(t *testing.T, routes router.Routes, route string) (string, *reactea.App) {
	t.Helper()

	component := router.NewWithRoutes(routes)

	app := reactea.New(component, reactea.WithRoute(route), reactea.WithSize(20, 5))

	app.Init()

	return app.View().Content, app
}

func TestRoutesToTheMatchingPage(t *testing.T) {
	content, _ := render(t, pages(map[string]string{
		"default":    "DEFAULT",
		"/test/test": "TESTS",
	}), "/test/test")

	if !strings.Contains(content, "TESTS") {
		t.Errorf("content = %q", content)
	}
}

func TestFallsBackToDefault(t *testing.T) {
	content, _ := render(t, pages(map[string]string{"default": "DEFAULT"}), "/nowhere")

	if !strings.Contains(content, "DEFAULT") {
		t.Errorf("content = %q", content)
	}
}

func TestNotFound(t *testing.T) {
	content, _ := render(t, pages(map[string]string{"/known": "KNOWN"}), "/unknown")

	if !strings.Contains(content, `Couldn't route for "/unknown"`) {
		t.Errorf("content = %q", content)
	}
}

func TestNotFoundIsOverridable(t *testing.T) {
	component := router.NewWithRoutes(pages(map[string]string{"/known": "KNOWN"}))
	component.NotFound = func(ctx *reactea.Ctx) string {
		return "no page at " + ctx.Route()
	}

	app := reactea.New(component, reactea.WithRoute("/unknown"), reactea.WithSize(20, 5))

	app.Init()

	if got := app.View().Content; !strings.Contains(got, "no page at /unknown") {
		t.Errorf("content = %q", got)
	}
}

func TestParamsReachThePage(t *testing.T) {
	routes := router.Routes{
		"/team/:id": func(params router.Params) reactea.Component {
			return &page{label: fmt.Sprintf("team %s", params["id"])}
		},
	}

	content, _ := render(t, routes, "/team/42")

	if !strings.Contains(content, "team 42") {
		t.Errorf("content = %q", content)
	}
}

// When several placeholders match, the most specific one has to win on every
// run — not be picked at random by Go's map iteration order.
func TestSpecificityIsDeterministic(t *testing.T) {
	routes := pages(map[string]string{
		"/user/:id":      "PARAM",
		"/user/settings": "EXACT",
	})

	for i := range 50 {
		content, _ := render(t, routes, "/user/settings")

		if !strings.Contains(content, "EXACT") {
			t.Fatalf("iteration %d: content = %q", i, content)
		}
	}
}

func TestRouteChangeSwapsThePage(t *testing.T) {
	routes := pages(map[string]string{"/a": "A", "/b": "B"})

	component := router.NewWithRoutes(routes)

	app := reactea.New(component, reactea.WithRoute("/a"), reactea.WithSize(20, 5))

	app.Init()

	first := component.Current()

	if got := app.View().Content; !strings.Contains(got, "A") {
		t.Fatalf("content = %q", got)
	}

	_, cmd := app.Update(app.Ctx().SetRoute("/b")())
	app.Update(cmd())

	if got := app.View().Content; !strings.Contains(got, "B") {
		t.Errorf("content = %q", got)
	}

	if !first.(*page).destroyed {
		t.Error("the replaced page's cleanup never ran")
	}
}

// Closing the app's root scope has to reach a page mounted under the router,
// without the router being asked to forward anything.
func TestRootTeardownReachesTheRoutedPage(t *testing.T) {
	component := router.NewWithRoutes(pages(map[string]string{"default": "PAGE"}))

	app := reactea.New(component, reactea.WithSize(20, 5))

	app.Init()

	current := component.Current()

	app.Scope().Close()

	if !current.(*page).destroyed {
		t.Error("the page's cleanup never ran")
	}
}

// A page's cleanups run when it is routed away from, and not before.
func TestUnmountRunsOnlyThatPagesCleanups(t *testing.T) {
	component := router.NewWithRoutes(pages(map[string]string{"/a": "A", "/b": "B"}))

	app := reactea.New(component, reactea.WithRoute("/a"), reactea.WithSize(20, 5))

	app.Init()

	first := component.Current().(*page)

	if first.destroyed {
		t.Fatal("the page was torn down while still mounted")
	}

	_, cmd := app.Update(app.Ctx().SetRoute("/b")())
	app.Update(cmd())

	if !first.destroyed {
		t.Error("the replaced page's cleanup never ran")
	}

	if second := component.Current().(*page); second.destroyed {
		t.Error("the new page was torn down on arrival")
	}
}

func TestMessagesReachTheRoutedPage(t *testing.T) {
	var seen int

	routes := router.Routes{
		"default": func(router.Params) reactea.Component {
			return &countingPage{seen: &seen}
		},
	}

	component := router.NewWithRoutes(routes)

	app := reactea.New(component, reactea.WithSize(20, 5))

	app.Init()
	app.Update(tea.KeyPressMsg{Code: 'x', Text: "x"})

	if seen != 1 {
		t.Errorf("the page saw %d messages, want 1", seen)
	}
}

type countingPage struct {
	reactea.BasicComponent

	seen *int
}

func (c *countingPage) Render(*reactea.Ctx) string { return "" }

func (c *countingPage) Update(*reactea.Ctx, tea.Msg) tea.Cmd {
	*c.seen++

	return nil
}
