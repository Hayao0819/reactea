package reactea_test

import (
	"bytes"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/Hayao0819/reactea/v2"
)

type probe struct {
	reactea.BasicComponent

	label string

	width, height int
	inited        bool
	destroyed     bool
	messages      []tea.Msg

	onUpdate func(*reactea.Ctx, tea.Msg) tea.Cmd
	onRender func(*reactea.Ctx)
}

func (c *probe) Init(*reactea.Ctx) tea.Cmd {
	c.inited = true

	return nil
}

func (c *probe) Destroy() { c.destroyed = true }

func (c *probe) Update(ctx *reactea.Ctx, msg tea.Msg) tea.Cmd {
	c.messages = append(c.messages, msg)

	if c.onUpdate != nil {
		return c.onUpdate(ctx, msg)
	}

	return nil
}

func (c *probe) Render(ctx *reactea.Ctx) string {
	c.width, c.height = ctx.Size()

	if c.onRender != nil {
		c.onRender(ctx)
	}

	return c.label
}

// A component can be driven with nothing but an App — no terminal, no program.
func TestComponentDrivenWithoutAProgram(t *testing.T) {
	root := &probe{label: "hello"}

	app := reactea.New(root, reactea.WithSize(40, 10))

	app.Init()
	app.Update(tea.KeyPressMsg{Code: 'x', Text: "x"})

	view := app.View()

	if !root.inited {
		t.Error("Init did not reach the root")
	}

	if len(root.messages) != 1 {
		t.Errorf("root saw %d messages", len(root.messages))
	}

	if view.Content != "hello" {
		t.Errorf("content = %q", view.Content)
	}

	if root.width != 40 || root.height != 10 {
		t.Errorf("root box = %dx%d, want 40x10", root.width, root.height)
	}
}

// The whole point of the teardown filter: quitting the ordinary Bubbletea way
// still destroys the tree.
func TestTeaQuitRunsDestroy(t *testing.T) {
	root := &probe{label: ""}

	root.onUpdate = func(_ *reactea.Ctx, msg tea.Msg) tea.Cmd {
		if _, ok := msg.(tea.KeyPressMsg); ok {
			return tea.Quit
		}

		return nil
	}

	var in, out bytes.Buffer

	in.WriteString("q")

	if err := reactea.New(root).Run(tea.WithInput(&in), tea.WithOutput(&out)); err != nil {
		t.Fatal(err)
	}

	if !root.destroyed {
		t.Fatal("tea.Quit skipped Destroy")
	}
}

func TestCtxInsetTranslatesTheCursor(t *testing.T) {
	root := &probe{label: "x"}

	root.onRender = func(ctx *reactea.Ctx) {
		ctx.Inset(3, 2, 10, 5).CursorAt(1, 1)
	}

	app := reactea.New(root, reactea.WithSize(40, 10))

	view := app.View()

	if view.Cursor == nil {
		t.Fatal("no cursor was reported")
	}

	if view.Cursor.X != 4 || view.Cursor.Y != 3 {
		t.Errorf("cursor = (%d, %d), want (4, 3)", view.Cursor.X, view.Cursor.Y)
	}
}

func TestCtxInsetClampsToTheParent(t *testing.T) {
	ctx := reactea.New(&probe{}, reactea.WithSize(20, 8)).Ctx()

	child := ctx.Inset(5, 2, 100, 100)

	if w, h := child.Size(); w != 15 || h != 6 {
		t.Errorf("child box = %dx%d, want 15x6", w, h)
	}

	if w, h := ctx.Inset(-5, -5, 4, 4).Size(); w != 4 || h != 4 {
		t.Errorf("negative offsets produced %dx%d", w, h)
	}
}

func TestDecorationsReachTheView(t *testing.T) {
	root := &probe{label: "x"}

	root.onRender = func(ctx *reactea.Ctx) {
		ctx.AltScreen(true)
		ctx.Title("reactea")
	}

	view := reactea.New(root, reactea.WithSize(10, 3)).View()

	if !view.AltScreen {
		t.Error("AltScreen was not carried through")
	}

	if view.WindowTitle != "reactea" {
		t.Errorf("WindowTitle = %q", view.WindowTitle)
	}
}

// Decorations are per frame, so a component that stops asking gets its way.
func TestDecorationsResetEachFrame(t *testing.T) {
	root := &probe{label: "x"}

	decorate := true

	root.onRender = func(ctx *reactea.Ctx) {
		if decorate {
			ctx.CursorAt(1, 1)
		}
	}

	app := reactea.New(root, reactea.WithSize(10, 3))

	if app.View().Cursor == nil {
		t.Fatal("no cursor on the first frame")
	}

	decorate = false

	if cursor := app.View().Cursor; cursor != nil {
		t.Errorf("stale cursor survived the frame: %+v", cursor)
	}
}

func TestRouteChangeIsDeliveredAsAMessage(t *testing.T) {
	root := &probe{label: "x"}

	app := reactea.New(root, reactea.WithSize(10, 3))

	if app.Route() != "/" {
		t.Fatalf("initial route = %q", app.Route())
	}

	_, cmd := app.Update(routeRequest(t, app, "/inbox"))
	if cmd == nil {
		t.Fatal("the route request produced no command")
	}

	if app.Route() != "/inbox" {
		t.Errorf("route = %q, want /inbox", app.Route())
	}

	changed, ok := cmd().(reactea.RouteChangedMsg)
	if !ok {
		t.Fatalf("cmd produced %T, want RouteChangedMsg", cmd())
	}

	if changed.From != "/" || changed.To != "/inbox" {
		t.Errorf("changed = %+v", changed)
	}
}

func TestRouteChangeToTheSameRouteIsANoop(t *testing.T) {
	app := reactea.New(&probe{}, reactea.WithRoute("/inbox"))

	if _, cmd := app.Update(routeRequest(t, app, "/inbox")); cmd != nil {
		t.Error("re-routing to the current route produced a command")
	}
}

func TestNavigateIsRelative(t *testing.T) {
	app := reactea.New(&probe{}, reactea.WithRoute("/a/b"))

	app.Update(navigate(t, app, ".."))

	if app.Route() != "/a" {
		t.Errorf("route = %q, want /a", app.Route())
	}

	app.Update(navigate(t, app, "c"))

	if app.Route() != "/a/c" {
		t.Errorf("route = %q, want /a/c", app.Route())
	}
}

// Two apps in one process must not share a route.
func TestAppsAreIndependent(t *testing.T) {
	first := reactea.New(&probe{}, reactea.WithRoute("/first"))
	second := reactea.New(&probe{}, reactea.WithRoute("/second"))

	first.Update(routeRequest(t, first, "/moved"))

	if second.Route() != "/second" {
		t.Errorf("the second app's route moved to %q", second.Route())
	}
}

func TestFuncAndText(t *testing.T) {
	app := reactea.New(reactea.Func(func(ctx *reactea.Ctx) string {
		width, _ := ctx.Size()

		return strings.Repeat("-", width)
	}), reactea.WithSize(5, 1))

	if got := app.View().Content; got != "-----" {
		t.Errorf("Func rendered %q", got)
	}

	if got := reactea.New(reactea.Text("hi"), reactea.WithSize(5, 1)).View().Content; got != "hi" {
		t.Errorf("Text rendered %q", got)
	}
}

// routeRequest builds the message Ctx.SetRoute produces.
func routeRequest(t *testing.T, app *reactea.App, target string) tea.Msg {
	t.Helper()

	return app.Ctx().SetRoute(target)()
}

func navigate(t *testing.T, app *reactea.App, target string) tea.Msg {
	t.Helper()

	return app.Ctx().Navigate(target)()
}
