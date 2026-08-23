package reactea_test

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/Hayao0819/reactea/v2"
)

type wrapping struct {
	reactea.Wrapper

	keys int
}

func (w *wrapping) Update(ctx *reactea.Ctx, msg tea.Msg) tea.Cmd {
	if reactea.Key(msg, "q", "ctrl+c") {
		w.keys++

		return nil
	}

	return w.Wrapper.Update(ctx, msg)
}

func TestWrapperForwardsTheLifecycle(t *testing.T) {
	child := &probe{label: "child"}

	root := &wrapping{Wrapper: reactea.Wrap(child)}

	app := reactea.New(root, reactea.WithSize(10, 2))

	app.Init()
	app.Update(tea.KeyPressMsg{Code: 'q', Text: "q"})
	app.Update(tea.KeyPressMsg{Code: 'x', Text: "x"})

	if !child.inited {
		t.Error("Init did not reach the child")
	}

	if root.keys != 1 {
		t.Errorf("the wrapper handled %d keys, want 1", root.keys)
	}

	if len(child.messages) != 1 {
		t.Errorf("the child saw %d messages, want 1", len(child.messages))
	}

	if got := app.View().Content; got != "child" {
		t.Errorf("content = %q", got)
	}
}

func TestKey(t *testing.T) {
	press := tea.KeyPressMsg{Code: 'q', Text: "q"}

	if !reactea.Key(press, "a", "q") {
		t.Error("a bound key was not matched")
	}

	if reactea.Key(press, "a") {
		t.Error("an unbound key matched")
	}

	if reactea.Key(tea.WindowSizeMsg{}, "q") {
		t.Error("a non-key message matched")
	}
}

func TestMouseIsLocalToTheBox(t *testing.T) {
	ctx := reactea.New(&probe{}, reactea.WithSize(40, 10)).Ctx().Inset(10, 4, 8, 3)

	click := func(x, y int) tea.Msg {
		return tea.MouseClickMsg{X: x, Y: y, Button: tea.MouseLeft}
	}

	if x, y, ok := reactea.Mouse(ctx, click(12, 5)); !ok || x != 2 || y != 1 {
		t.Errorf("inside the box = (%d, %d, %v), want (2, 1, true)", x, y, ok)
	}

	if _, _, ok := reactea.Mouse(ctx, click(9, 5)); ok {
		t.Error("a click left of the box was claimed")
	}

	if _, _, ok := reactea.Mouse(ctx, click(18, 5)); ok {
		t.Error("a click right of the box was claimed")
	}

	if _, _, ok := reactea.Mouse(ctx, click(12, 7)); ok {
		t.Error("a click below the box was claimed")
	}

	if _, _, ok := reactea.Mouse(ctx, tea.KeyPressMsg{Code: 'q'}); ok {
		t.Error("a key press was claimed as a mouse event")
	}
}
