package reactea_test

import (
	"bytes"
	"strings"
	"sync"
	"sync/atomic"
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

func (c *probe) Init(ctx *reactea.Ctx) tea.Cmd {
	c.inited = true

	ctx.OnDestroy(func() { c.destroyed = true })

	return nil
}

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

func TestTeaQuitRunsCleanups(t *testing.T) {
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
		t.Fatal("tea.Quit skipped the cleanups")
	}
}

func TestCleanupsRunNewestFirst(t *testing.T) {
	var order []string

	app := reactea.New(reactea.Func(func(*reactea.Ctx) string { return "" }))

	app.Scope().OnDestroy(func() { order = append(order, "first") })
	app.Scope().OnDestroy(func() { order = append(order, "second") })

	app.Scope().Close()

	if len(order) != 2 || order[0] != "second" || order[1] != "first" {
		t.Errorf("order = %v, want [second first]", order)
	}
}

func TestCleanupOnAClosedScopeRunsAtOnce(t *testing.T) {
	scope := reactea.NewScope()

	scope.Close()

	ran := false

	scope.OnDestroy(func() { ran = true })

	if !ran {
		t.Error("the late cleanup never ran")
	}
}

func TestCloseIsIdempotent(t *testing.T) {
	runs := 0

	scope := reactea.NewScope()

	scope.OnDestroy(func() { runs++ })

	scope.Close()
	scope.Close()

	if runs != 1 {
		t.Errorf("cleanup ran %d times", runs)
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

func TestTerminalCommandsPersistAcrossFrames(t *testing.T) {
	app := reactea.New(&probe{label: "x"}, reactea.WithSize(10, 3))

	app.Update(reactea.EnterAltScreen())
	app.Update(reactea.SetWindowTitle("reactea")())

	for frame := range 3 {
		view := app.View()

		if !view.AltScreen {
			t.Errorf("frame %d: AltScreen was lost", frame)
		}

		if view.WindowTitle != "reactea" {
			t.Errorf("frame %d: WindowTitle = %q", frame, view.WindowTitle)
		}
	}

	app.Update(reactea.ExitAltScreen())

	if app.View().AltScreen {
		t.Error("ExitAltScreen did not take")
	}
}

func TestCursorResetsEachFrame(t *testing.T) {
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

func routeRequest(t *testing.T, app *reactea.App, target string) tea.Msg {
	t.Helper()

	return app.Ctx().SetRoute(target)()
}

func navigate(t *testing.T, app *reactea.App, target string) tea.Msg {
	t.Helper()

	return app.Ctx().Navigate(target)()
}

func TestPanickingCleanupIsIsolated(t *testing.T) {
	ran := false

	scope := reactea.NewScope()

	scope.OnDestroy(func() { ran = true })
	scope.OnDestroy(func() { panic("boom") })

	scope.Close()

	if !ran {
		t.Error("an earlier cleanup was stranded by a panicking one")
	}
}

func TestProgressBarAndBracketedPaste(t *testing.T) {
	app := reactea.New(&probe{label: "x"}, reactea.WithSize(10, 1))

	bar := &tea.ProgressBar{State: tea.ProgressBarDefault, Value: 42}

	app.Update(reactea.SetProgressBar(bar)())
	app.Update(reactea.SetBracketedPaste(false)())

	view := app.View()

	if view.ProgressBar == nil || view.ProgressBar.Value != 42 {
		t.Errorf("ProgressBar = %+v", view.ProgressBar)
	}

	if !view.DisableBracketedPasteMode {
		t.Error("bracketed paste was not disabled")
	}

	app.Update(reactea.SetProgressBar(nil)())
	app.Update(reactea.SetBracketedPaste(true)())

	view = app.View()

	if view.ProgressBar != nil {
		t.Errorf("ProgressBar = %+v, want nil", view.ProgressBar)
	}

	if view.DisableBracketedPasteMode {
		t.Error("bracketed paste was not restored")
	}
}

func TestInputCaptureNests(t *testing.T) {
	app := reactea.New(&probe{label: "x"}, reactea.WithSize(10, 2))
	var first, second reactea.InputCapture

	if app.InputCaptured() {
		t.Fatal("input was captured before anything asked")
	}

	first.CaptureInput(app.Ctx())
	second.CaptureInput(app.Ctx())

	if !app.Ctx().InputCaptured() {
		t.Error("the Ctx did not see the capture")
	}

	first.ReleaseInput()

	if !app.InputCaptured() {
		t.Error("one release cleared two captures")
	}

	second.ReleaseInput()

	if app.InputCaptured() {
		t.Error("the capture was not released")
	}

	second.ReleaseInput()

	if app.InputCaptured() {
		t.Error("an unbalanced release went negative")
	}
}

func TestScopedCaptureIsReleasedWithItsScope(t *testing.T) {
	app := reactea.New(&probe{label: "x"}, reactea.WithSize(10, 2))

	page := app.Scope().Child()
	var capture reactea.InputCapture

	capture.CaptureInput(app.Ctx().WithScope(page))

	if !app.InputCaptured() {
		t.Fatal("the capture did not take")
	}

	page.Close()

	if app.InputCaptured() {
		t.Error("closing the scope did not release the capture")
	}
}

func TestReleasedScopeDoesNotReleaseAnotherCapture(t *testing.T) {
	app := reactea.New(&probe{label: "x"}, reactea.WithSize(10, 2))
	page := app.Scope().Child()
	var first, second reactea.InputCapture

	first.CaptureInput(app.Ctx().WithScope(page))
	first.ReleaseInput()
	second.CaptureInput(app.Ctx())

	page.Close()

	if !app.InputCaptured() {
		t.Error("closing an old scope released another owner's capture")
	}

	second.ReleaseInput()
}

func TestScopeSurvivesConcurrentRegistration(t *testing.T) {
	scope := reactea.NewScope()

	var registered atomic.Int64

	var wg sync.WaitGroup

	for range 8 {
		wg.Go(func() {
			for range 50 {
				scope.OnDestroy(func() { registered.Add(1) })
			}
		})
	}

	wg.Go(scope.Close)

	wg.Wait()

	scope.Close()

	if registered.Load() != 400 {
		t.Errorf("%d of 400 cleanups ran", registered.Load())
	}
}

// Opening a child while a sibling closes used to read the sibling's closed flag
// without its lock, which -race catches.
func TestScopeSurvivesChildrenOpeningAndClosingAtOnce(t *testing.T) {
	root := reactea.NewScope()

	var wg sync.WaitGroup

	for range 50 {
		wg.Go(func() { root.Child() })
		wg.Go(func() { root.Child().Close() })
	}

	wg.Wait()
	root.Close()
}

// A scope-bound claim is released wherever its scope closes, which need not be
// the loop reading the count. -race catches an unguarded counter.
func TestReleasingACaptureOffTheLoopDoesNotRaceTheCount(t *testing.T) {
	app := reactea.New(&probe{label: "x"}, reactea.WithSize(10, 2))

	page := app.Scope().Child()
	var capture reactea.InputCapture
	capture.CaptureInput(app.Ctx().WithScope(page))

	var wg sync.WaitGroup

	wg.Go(page.Close)
	wg.Go(func() { _ = app.InputCaptured() })

	wg.Wait()

	if app.InputCaptured() {
		t.Error("closing the scope left the keys claimed")
	}
}

func TestCaptureOnAClosedScopeIsDropped(t *testing.T) {
	app := reactea.New(&probe{label: "x"}, reactea.WithSize(10, 2))

	page := app.Scope().Child()
	page.Close()

	var capture reactea.InputCapture
	capture.CaptureInput(app.Ctx().WithScope(page))

	if app.InputCaptured() {
		t.Error("a closed scope kept the input captured")
	}
}

func TestCaptureIsIdempotent(t *testing.T) {
	app := reactea.New(&probe{label: "x"}, reactea.WithSize(10, 2))
	var capture reactea.InputCapture

	capture.CaptureInput(app.Ctx())
	capture.CaptureInput(app.Ctx())
	capture.ReleaseInput()

	if app.InputCaptured() {
		t.Error("capturing twice required two releases")
	}
}

func TestCopiedCaptureKeepsOneOwnershipClaim(t *testing.T) {
	app := reactea.New(&probe{label: "x"}, reactea.WithSize(10, 2))
	var capture reactea.InputCapture

	capture.CaptureInput(app.Ctx())
	alias := capture
	alias.ReleaseInput()

	if app.InputCaptured() {
		t.Error("a copied capture could not release its ownership claim")
	}

	capture.ReleaseInput()
	if app.InputCaptured() {
		t.Error("releasing both aliases left an unbalanced capture")
	}
}

type chaining struct {
	reactea.BasicComponent

	steps []string
}

type stepMsg string

func (c *chaining) Render(*reactea.Ctx) string { return strings.Join(c.steps, ",") }

func (c *chaining) Update(_ *reactea.Ctx, msg tea.Msg) tea.Cmd {
	step, ok := msg.(stepMsg)
	if !ok {
		return nil
	}

	c.steps = append(c.steps, string(step))

	switch step {
	case "one":
		return tea.Batch(
			func() tea.Msg { return stepMsg("two") },
			func() tea.Msg { return stepMsg("three") },
		)
	case "two":
		return func() tea.Msg { return stepMsg("four") }
	}

	return nil
}

func TestSendRunsWhatItProduces(t *testing.T) {
	root := &chaining{}

	app := reactea.New(root, reactea.WithSize(20, 1))

	app.Send(stepMsg("one"))

	if got := app.View().Content; got != "one,two,three,four" {
		t.Errorf("steps = %q, want the whole chain", got)
	}
}

type broadBatch struct {
	reactea.BasicComponent

	ran int
}

func (c *broadBatch) Init(*reactea.Ctx) tea.Cmd {
	cmds := make([]tea.Cmd, 150)
	for i := range cmds {
		cmds[i] = func() tea.Msg {
			c.ran++

			return nil
		}
	}

	return tea.Batch(cmds...)
}

func (c *broadBatch) Render(*reactea.Ctx) string { return "" }

func TestStartRunsLargeFiniteBatch(t *testing.T) {
	root := &broadBatch{}

	reactea.New(root).Start()

	if root.ran != 150 {
		t.Errorf("batch ran %d commands, want 150", root.ran)
	}
}

type resched struct {
	reactea.BasicComponent
}

func (c *resched) Render(*reactea.Ctx) string { return "" }

func (c *resched) Update(_ *reactea.Ctx, msg tea.Msg) tea.Cmd {
	if _, ok := msg.(stepMsg); !ok {
		return nil
	}

	return func() tea.Msg { return stepMsg("again") }
}

func TestSendRejectsACommandThatReschedulesItself(t *testing.T) {
	root := &resched{}

	app := reactea.New(root, reactea.WithSize(20, 1))

	defer func() {
		if got := recover(); got != "reactea: command chain did not settle" {
			t.Fatalf("Send panic = %v", got)
		}
	}()

	app.Send(stepMsg("again"))
}

func TestScopeContextIsCancelledOnClose(t *testing.T) {
	app := reactea.New(&probe{label: "x"}, reactea.WithSize(10, 2))

	page := app.Scope().Child()

	ctx := app.Ctx().WithScope(page).Context()

	if ctx.Err() != nil {
		t.Fatal("the context started cancelled")
	}

	if same := page.Context(); same != ctx {
		t.Error("Context returned a different context on the second call")
	}

	page.Close()

	if ctx.Err() == nil {
		t.Error("closing the scope did not cancel the context")
	}
}

func TestScopeContextOnAClosedScopeIsAlreadyCancelled(t *testing.T) {
	scope := reactea.NewScope()

	scope.Close()

	if scope.Context().Err() == nil {
		t.Error("a context taken from a closed scope was live")
	}
}

type startChain struct {
	reactea.BasicComponent

	steps []string
}

func (c *startChain) Render(*reactea.Ctx) string { return strings.Join(c.steps, ",") }

func (c *startChain) Init(*reactea.Ctx) tea.Cmd {
	return func() tea.Msg { return stepMsg("loaded") }
}

func (c *startChain) Update(_ *reactea.Ctx, msg tea.Msg) tea.Cmd {
	if step, ok := msg.(stepMsg); ok {
		c.steps = append(c.steps, string(step))
	}

	return nil
}

func TestStartRunsWhatInitProduces(t *testing.T) {
	app := reactea.New(&startChain{}, reactea.WithSize(20, 1))

	app.Start()

	if got := app.View().Content; got != "loaded" {
		t.Errorf("content = %q, want Init's result", got)
	}
}

func TestStartOnAComponentWithNoInitCommand(t *testing.T) {
	app := reactea.New(&probe{label: "x"}, reactea.WithSize(10, 1))

	app.Start()

	if !(&probe{}).inited && app.View().Content != "x" {
		t.Errorf("content = %q", app.View().Content)
	}
}
