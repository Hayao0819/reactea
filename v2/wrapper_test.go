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
