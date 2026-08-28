package modal_test

import (
	"errors"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/Hayao0819/reactea/v2"
	"github.com/Hayao0819/reactea/v2/modal"
)

type prompt struct {
	reactea.BasicComponent

	name      string
	destroyed bool
}

func (c *prompt) Render(*reactea.Ctx) string { return "prompt:" + c.name }

func (c *prompt) Init(ctx *reactea.Ctx) tea.Cmd {
	ctx.OnDestroy(func() { c.destroyed = true })

	return nil
}

func (c *prompt) Update(_ *reactea.Ctx, msg tea.Msg) tea.Cmd {
	if key, ok := msg.(tea.KeyPressMsg); ok && key.String() == "enter" {
		return modal.Return(c.name)
	}

	return nil
}

type base struct {
	reactea.BasicComponent

	seen int
}

func (c *base) Render(*reactea.Ctx) string { return "base" }

func (c *base) Update(*reactea.Ctx, tea.Msg) tea.Cmd {
	c.seen++

	return nil
}

func drive(t *testing.T, app *reactea.App, msg tea.Msg) {
	t.Helper()

	for depth := 0; msg != nil && depth < 8; depth++ {
		_, cmd := app.Update(msg)
		if cmd == nil {
			return
		}

		next := cmd()

		if batch, ok := next.(tea.BatchMsg); ok {
			for _, sub := range batch {
				if produced := sub(); produced != nil {
					drive(t, app, produced)
				}
			}

			return
		}

		msg = next
	}
}

func TestPushRendersTheModalOverTheBase(t *testing.T) {
	page := &base{}
	stack := modal.New(page)

	app := reactea.New(stack, reactea.WithSize(20, 5))

	app.Init()

	if got := app.View().Content; got != "base" {
		t.Fatalf("content = %q", got)
	}

	drive(t, app, stack.Push(&prompt{name: "who"})())

	if got := app.View().Content; got != "prompt:who" {
		t.Errorf("content = %q", got)
	}
}

func TestModalTakesTheInput(t *testing.T) {
	page := &base{}
	stack := modal.New(page)

	app := reactea.New(stack, reactea.WithSize(20, 5))

	app.Init()
	app.Update(tea.KeyPressMsg{Code: 'x', Text: "x"})

	if page.seen != 1 {
		t.Fatalf("base saw %d messages before the modal", page.seen)
	}

	drive(t, app, stack.Push(&prompt{name: "who"})())

	app.Update(tea.KeyPressMsg{Code: 'x', Text: "x"})

	if page.seen != 1 {
		t.Errorf("base saw input while a modal was up (%d)", page.seen)
	}
}

func TestReturnPopsAndDeliversTheResult(t *testing.T) {
	stack := modal.New(&base{})

	app := reactea.New(stack, reactea.WithSize(20, 5))

	app.Init()

	shown := &prompt{name: "answer"}

	drive(t, app, stack.Push(shown)())

	_, cmd := app.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("enter produced no command")
	}

	batch, ok := cmd().(tea.BatchMsg)
	if !ok {
		t.Fatalf("expected a batch, got %T", cmd())
	}

	var result modal.Result[string]

	for _, sub := range batch {
		switch msg := sub().(type) {
		case modal.Result[string]:
			result = msg
		default:
			app.Update(msg)
		}
	}

	if !result.Ok() || result.Value != "answer" {
		t.Errorf("result = %+v", result)
	}

	if stack.Top() != nil {
		t.Error("the modal was not popped")
	}

	if !shown.destroyed {
		t.Error("the popped modal's cleanup never ran")
	}

	if got := app.View().Content; got != "base" {
		t.Errorf("content after the modal = %q", got)
	}
}

func TestFailDeliversAnError(t *testing.T) {
	stack := modal.New(&base{})

	app := reactea.New(stack, reactea.WithSize(20, 5))

	app.Init()

	drive(t, app, stack.Push(&prompt{name: "x"})())

	batch, ok := modal.Fail[string](errors.New("nope"))().(tea.BatchMsg)
	if !ok {
		t.Fatal("Fail did not produce a batch")
	}

	var result modal.Result[string]

	for _, sub := range batch {
		switch msg := sub().(type) {
		case modal.Result[string]:
			result = msg
		default:
			app.Update(msg)
		}
	}

	if result.Ok() {
		t.Error("a failed result reported Ok")
	}

	if stack.Top() != nil {
		t.Error("Fail did not pop the modal")
	}
}

func TestRootTeardownReachesEveryModal(t *testing.T) {
	stack := modal.New(&base{})

	app := reactea.New(stack, reactea.WithSize(20, 5))

	app.Init()

	first, second := &prompt{name: "a"}, &prompt{name: "b"}

	drive(t, app, stack.Push(first)())
	drive(t, app, stack.Push(second)())

	app.Scope().Close()

	if !first.destroyed || !second.destroyed {
		t.Errorf("stacked modals were not cleaned up: %v %v", first.destroyed, second.destroyed)
	}
}

func TestDismissOnlyTearsDownTheTop(t *testing.T) {
	stack := modal.New(&base{})

	app := reactea.New(stack, reactea.WithSize(20, 5))

	app.Init()

	first, second := &prompt{name: "a"}, &prompt{name: "b"}

	drive(t, app, stack.Push(first)())
	drive(t, app, stack.Push(second)())

	app.Update(modal.Dismiss())

	if !second.destroyed {
		t.Error("the dismissed modal's cleanup never ran")
	}

	if first.destroyed {
		t.Error("dismissing the top modal tore down the one below it")
	}

	if stack.Top() != first {
		t.Error("the modal below did not come back to the top")
	}
}

type tickMsg struct{}

func TestDataReachesTheBaseWhileAModalIsUp(t *testing.T) {
	page := &base{}
	stack := modal.New(page)

	app := reactea.New(stack, reactea.WithSize(20, 5))

	app.Init()

	drive(t, app, stack.Push(&prompt{name: "x"})())

	before := page.seen

	app.Update(tickMsg{})

	if page.seen != before+1 {
		t.Error("the base was starved of a data message while a modal was up")
	}

	app.Update(tea.KeyPressMsg{Code: 'x', Text: "x"})

	if page.seen != before+1 {
		t.Error("input reached the base while a modal was up")
	}
}

func TestAModalCapturesInput(t *testing.T) {
	stack := modal.New(&base{})

	app := reactea.New(stack, reactea.WithSize(20, 5))

	app.Init()

	drive(t, app, stack.Push(&prompt{name: "x"})())

	if !app.InputCaptured() {
		t.Error("a modal did not capture the input")
	}

	drive(t, app, modal.Dismiss())

	if app.InputCaptured() {
		t.Error("dismissing the modal did not release the input")
	}
}
