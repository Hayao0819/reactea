package router_test

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/Hayao0819/reactea/v2"
	"github.com/Hayao0819/reactea/v2/layout"
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

func TestDefaultRouteReceivesAndTracksThePath(t *testing.T) {
	routes := router.Routes{
		"default": func(params router.Params) reactea.Component {
			return &page{label: params.Path()}
		},
	}

	component := router.NewWithRoutes(routes)
	app := reactea.New(component, reactea.WithRoute("/first"), reactea.WithSize(20, 5))
	app.Init()

	if got := app.View().Content; !strings.Contains(got, "/first") {
		t.Fatalf("content = %q, want first path", got)
	}

	_, cmd := app.Update(app.Ctx().SetRoute("/second")())
	app.Update(cmd())

	if got := app.View().Content; !strings.Contains(got, "/second") {
		t.Errorf("content = %q, want updated path", got)
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
			return &page{label: fmt.Sprintf("team %s", params.Get("id"))}
		},
	}

	content, _ := render(t, routes, "/team/42")

	if !strings.Contains(content, "team 42") {
		t.Errorf("content = %q", content)
	}
}

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

// A nested router would otherwise tear down its parent's page on every
// sub-route move.
func TestUnchangedMatchKeepsThePageMounted(t *testing.T) {
	routes := router.Routes{
		"/team/:id/*rest": func(params router.Params) reactea.Component {
			return &page{label: "team " + params.Get("id")}
		},
	}

	component := router.NewWithRoutes(routes)

	app := reactea.New(component, reactea.WithRoute("/team/42/chat"), reactea.WithSize(20, 5))

	app.Init()

	first := component.Current()

	_, cmd := app.Update(app.Ctx().SetRoute("/team/42/files")())
	app.Update(cmd())

	if component.Current() != first {
		t.Error("the page was remounted although its match did not change")
	}

	if first.(*page).destroyed {
		t.Error("the page's cleanup ran although it stayed mounted")
	}

	_, cmd = app.Update(app.Ctx().SetRoute("/team/7/chat")())
	app.Update(cmd())

	if component.Current() == first {
		t.Error("a changed param did not remount the page")
	}
}

func TestPageInitCommandSurvivesMounting(t *testing.T) {
	var initialised bool

	routes := router.Routes{
		"default": func(router.Params) reactea.Component {
			return &commandPage{onInit: func() { initialised = true }}
		},
	}

	component := router.NewWithRoutes(routes)

	app := reactea.New(component, reactea.WithSize(20, 5))

	cmd := app.Init()
	if cmd == nil {
		t.Fatal("the page's Init command was dropped")
	}

	cmd()

	if !initialised {
		t.Error("the page's Init command never ran")
	}
}

type commandPage struct {
	reactea.BasicComponent

	onInit func()
}

func (c *commandPage) Render(*reactea.Ctx) string { return "" }

func (c *commandPage) Init(*reactea.Ctx) tea.Cmd {
	return func() tea.Msg {
		c.onInit()

		return nil
	}
}

func TestPageAndParamAdapters(t *testing.T) {
	newHome := func() *page { return &page{label: "HOME"} }
	newMail := func(id string) *page { return &page{label: "mail " + id} }

	routes := router.Routes{
		"/home":     router.Page(newHome),
		"/mail/:id": router.Param("id", newMail),
	}

	if content, _ := render(t, routes, "/home"); !strings.Contains(content, "HOME") {
		t.Errorf("content = %q", content)
	}

	if content, _ := render(t, routes, "/mail/42"); !strings.Contains(content, "mail 42") {
		t.Errorf("content = %q", content)
	}
}

type paneBox struct {
	reactea.BasicComponent

	box *layout.Box
}

func (c *paneBox) Init(ctx *reactea.Ctx) tea.Cmd              { return c.box.Init(ctx) }
func (c *paneBox) Update(ctx *reactea.Ctx, m tea.Msg) tea.Cmd { return c.box.Update(ctx, m) }
func (c *paneBox) Render(ctx *reactea.Ctx) string             { return c.box.Render(ctx) }
func (c *paneBox) FocusNext() bool                            { return c.box.FocusNext() }
func (c *paneBox) FocusPrev() bool                            { return c.box.FocusPrev() }
func (c *paneBox) FocusFirst()                                { c.box.FocusFirst() }
func (c *paneBox) FocusLast()                                 { c.box.FocusLast() }
func (c *paneBox) HasFocusable() bool                         { return c.box.HasFocusable() }

// A routed page holding two panes must be reachable pane by pane, not as one
// stop on the way round.
func TestTabDescendsIntoARoutedPage(t *testing.T) {
	top, bottom := &page{label: "top"}, &page{label: "bottom"}

	inner := layout.Column(
		layout.Grow(1, top).Focusable(),
		layout.Grow(1, bottom).Focusable(),
	)

	pages := router.NewWithRoutes(router.Routes{
		"default": router.Page(func() reactea.Component { return &paneBox{box: inner} }),
	})

	rail := &page{label: "rail"}
	outer := layout.Row(layout.Fixed(4, rail).Focusable(), layout.Grow(1, pages))

	app := reactea.New(outer, reactea.WithSize(20, 6))

	app.Init()

	if !pages.HasFocusable() {
		t.Fatal("the router hid the panes inside its page")
	}

	reached := []int{}

	for range 4 {
		if !outer.FocusNext() {
			break
		}

		reached = append(reached, inner.Focused())
	}

	if len(reached) != 2 || reached[0] != 0 || reached[1] != 1 {
		t.Errorf("Tab reached panes %v, want both of them", reached)
	}
}

// counted names every page it builds, so a test can tell a rebuild from a page
// that was left where it was.
func counted(label string, built *int) router.Routes {
	return router.Routes{
		"default": func(router.Params) reactea.Component {
			*built++

			return &page{label: label}
		},
	}
}

func TestReloadBuildsThePageBeforeTheNextFrame(t *testing.T) {
	var built int

	component := router.NewWithRoutes(counted("one", &built))

	app := reactea.New(component, reactea.WithSize(20, 5))
	app.Start()

	if built != 1 {
		t.Fatalf("the first mount built %d pages", built)
	}

	component.SetRoutes(counted("two", &built), router.PreserveCurrent)

	if cmd := component.Reload(app.Ctx()); cmd != nil {
		app.Send(cmd())
	}

	if built != 2 {
		t.Errorf("Reload built %d pages in total, want 2", built)
	}

	if got := app.View().Content; !strings.Contains(got, "two") {
		t.Errorf("the frame after Reload reads %q; the router was left empty", got)
	}
}

func TestPreserveCurrentLeavesAMountedPageAlone(t *testing.T) {
	var built int

	component := router.NewWithRoutes(counted("one", &built))

	app := reactea.New(component, reactea.WithSize(20, 5))
	app.Start()

	component.SetRoutes(counted("two", &built), router.PreserveCurrent)
	app.Send(tea.KeyPressMsg{Code: 'x'})

	if built != 1 {
		t.Errorf("the page was rebuilt %d times; PreserveCurrent should have left it", built)
	}

	if got := app.View().Content; !strings.Contains(got, "one") {
		t.Errorf("content = %q, want the page that was already up", got)
	}
}

func TestRemountCurrentRebuildsFromTheNewTable(t *testing.T) {
	var built int

	component := router.NewWithRoutes(counted("one", &built))

	app := reactea.New(component, reactea.WithSize(20, 5))
	app.Start()

	component.SetRoutes(counted("two", &built), router.RemountCurrent)
	app.Send(tea.KeyPressMsg{Code: 'x'})

	if built != 2 {
		t.Errorf("the page was built %d times in total, want 2", built)
	}

	if got := app.View().Content; !strings.Contains(got, "two") {
		t.Errorf("content = %q, want the page from the new table", got)
	}
}

// Dropping the route that is up must not leave the old page on screen.
func TestTakingAwayTheMountedRouteFallsBack(t *testing.T) {
	component := router.NewWithRoutes(pages(map[string]string{
		"/one":    "ONE",
		"default": "HOME",
	}))

	app := reactea.New(component, reactea.WithSize(20, 5), reactea.WithRoute("/one"))
	app.Start()

	if got := app.View().Content; !strings.Contains(got, "ONE") {
		t.Fatalf("content = %q, want ONE", got)
	}

	component.SetRoutes(pages(map[string]string{"default": "HOME"}), router.RemountCurrent)
	app.Send(tea.KeyPressMsg{Code: 'x'})

	if got := app.View().Content; !strings.Contains(got, "HOME") {
		t.Errorf("content = %q, want the fallback", got)
	}
}
