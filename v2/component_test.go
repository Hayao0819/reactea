package reactea

import (
	"bytes"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
)

func TestDefaultComponent(t *testing.T) {
	var out bytes.Buffer
	var rendered string

	component := &mockComponent[struct{}]{
		renderFunc: func(c Component, s *struct{}, width, height int) string {
			rendered = "test passed"
			return rendered
		},
	}

	program := NewProgram(component, WithoutInput(), tea.WithOutput(&out), tea.WithWindowSize(20, 5))

	go func() {
		time.Sleep(20 * time.Millisecond)

		program.Quit()
	}()

	if _, err := program.Run(); err != nil {
		t.Fatal(err)
	}

	// The v2 renderer emits ANSI to the buffer, so assert the component's
	// Render ran during the program instead of scraping terminal bytes.
	if rendered != "test passed" {
		t.Errorf("component was not rendered, got %q", rendered)
	}
}

func TestInvisibleComponent(t *testing.T) {
	component := &InvisibleComponent{}

	if result := component.Render(1, 1); result != "" {
		t.Errorf("expected empty string, got \"%s\"", result)
	}
}

type decoratingComponent struct {
	BasicComponent

	title  string
	cursor *tea.Cursor
}

func (c *decoratingComponent) Render(int, int) string { return "CONTENT" }

func (c *decoratingComponent) DecorateView(view *tea.View) {
	view.AltScreen = true
	view.WindowTitle = c.title
	view.Cursor = c.cursor
}

// A component that implements ViewDecorator must reach the tea.View the program
// renders, not just its content.
func TestDecorateViewReachesProgram(t *testing.T) {
	root := &decoratingComponent{title: "reactea", cursor: tea.NewCursor(3, 4)}

	m := model{root: root, width: 20, height: 10}

	view := m.View()

	if view.Content != "CONTENT" {
		t.Fatalf("content = %q", view.Content)
	}

	if !view.AltScreen {
		t.Error("AltScreen was not carried through")
	}

	if view.WindowTitle != "reactea" {
		t.Errorf("WindowTitle = %q", view.WindowTitle)
	}

	if view.Cursor == nil || view.Cursor.X != 3 || view.Cursor.Y != 4 {
		t.Errorf("Cursor = %+v", view.Cursor)
	}
}

// A component that doesn't implement ViewDecorator leaves the view alone.
func TestDecorateViewIsOptional(t *testing.T) {
	m := model{root: StaticComponent("PLAIN"), width: 20, height: 10}

	view := m.View()

	if view.Content != "PLAIN" {
		t.Fatalf("content = %q", view.Content)
	}

	if view.AltScreen || view.Cursor != nil || view.WindowTitle != "" {
		t.Errorf("view was decorated by a plain component: %+v", view)
	}
}

func TestTranslateCursor(t *testing.T) {
	view := tea.NewView("")
	view.Cursor = tea.NewCursor(1, 2)

	TranslateCursor(&view, 10, 20)

	if view.Cursor.X != 11 || view.Cursor.Y != 22 {
		t.Errorf("cursor = %+v", view.Cursor)
	}

	view.Cursor = nil

	TranslateCursor(&view, 10, 20)
}
