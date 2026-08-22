package reactea_test

import (
	"strings"
	"testing"

	"charm.land/bubbles/v2/stopwatch"
	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"github.com/Hayao0819/reactea/v2"
)

func TestReactifyWidget(t *testing.T) {
	input := textinput.New()
	input.SetVirtualCursor(false)
	input.Focus()

	component := reactea.ReactifyWidget(input)

	var (
		_ reactea.Component = component
		_ reactea.Component = reactea.ReactifyWidget(viewport.New())
		_ reactea.Component = reactea.ReactifyWidget(textarea.New())
	)

	app := reactea.New(component, reactea.WithSize(20, 1))

	app.Update(tea.KeyPressMsg{Code: 'a', Text: "a"})

	if got := component.Widget.Value(); got != "a" {
		t.Fatalf("widget state was not stored back: %q", got)
	}

	view := app.View()

	if !strings.Contains(view.Content, "a") {
		t.Errorf("content = %q", view.Content)
	}

	if view.Cursor == nil {
		t.Error("a focused textinput with the virtual cursor off reported no cursor")
	}
}

// A widget drawing its own virtual cursor reports none, and the view is clean.
func TestReactifyWidgetVirtualCursor(t *testing.T) {
	input := textinput.New()
	input.Focus()

	app := reactea.New(reactea.ReactifyWidget(input), reactea.WithSize(20, 1))

	if cursor := app.View().Cursor; cursor != nil {
		t.Errorf("cursor = %+v, want nil while the virtual cursor is on", cursor)
	}
}

// The widget's cursor is reported in its own coordinates, so a parent that
// insets it gets the translation for free.
func TestReactifyWidgetCursorIsTranslated(t *testing.T) {
	input := textinput.New()
	input.SetVirtualCursor(false)
	input.Focus()

	widget := reactea.ReactifyWidget(input)

	root := reactea.Func(func(ctx *reactea.Ctx) string {
		return widget.Render(ctx.Inset(4, 2, 10, 1))
	})

	app := reactea.New(root, reactea.WithSize(20, 5))

	cursor := app.View().Cursor

	if cursor == nil {
		t.Fatal("no cursor")
	}

	if cursor.Y != 2 || cursor.X < 4 {
		t.Errorf("cursor = (%d, %d), want it offset by (4, 2)", cursor.X, cursor.Y)
	}
}

func TestReactifyWidgetInitIsOptional(t *testing.T) {
	ctx := reactea.New(nil).Ctx()

	if cmd := reactea.ReactifyWidget(textinput.New()).Init(ctx); cmd != nil {
		t.Error("textinput has no Init, expected a nil cmd")
	}

	if cmd := reactea.ReactifyWidget(stopwatch.New()).Init(ctx); cmd == nil {
		t.Error("stopwatch has an Init, expected its cmd")
	}
}

// A whole tea.Model nests through Reactify, cursor included.
type nestedModel struct{ text string }

func (m nestedModel) Init() tea.Cmd { return nil }

func (m nestedModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyPressMsg); ok {
		m.text += key.Text
	}

	return m, nil
}

func (m nestedModel) View() tea.View {
	view := tea.NewView(m.text)
	view.Cursor = tea.NewCursor(len(m.text), 0)

	return view
}

func TestReactify(t *testing.T) {
	component := reactea.Reactify[tea.Model](nestedModel{})

	app := reactea.New(component, reactea.WithSize(10, 1))

	app.Update(tea.KeyPressMsg{Code: 'h', Text: "h"})
	app.Update(tea.KeyPressMsg{Code: 'i', Text: "i"})

	view := app.View()

	if view.Content != "hi" {
		t.Errorf("content = %q", view.Content)
	}

	if view.Cursor == nil || view.Cursor.X != 2 {
		t.Errorf("cursor = %+v", view.Cursor)
	}
}
