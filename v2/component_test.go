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
