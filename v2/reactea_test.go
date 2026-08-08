package reactea

import (
	"bytes"
	"errors"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
)

func TestComponent(t *testing.T) {
	var out bytes.Buffer

	type testState struct {
		echoKey               string
		lastWidth, lastHeight int
	}

	root := &mockComponent[testState]{
		updateFunc: func(c Component, s *testState, msg tea.Msg) tea.Cmd {
			switch msg := msg.(type) {
			case tea.KeyMsg:
				if msg.String() == "x" {
					return Destroy
				}

				s.echoKey = msg.String()
			}

			SetRoute("/test/test/test")

			return nil
		},
		renderFunc: func(c Component, s *testState, width, height int) string {
			s.lastWidth, s.lastHeight = width, height

			return s.echoKey
		},
	}

	// WithWindowSize makes the startup WindowSizeMsg deterministic (v2 reports
	// 0x0 for a non-TTY otherwise). We observe echo and size through component
	// state rather than scraping the v2 renderer's ANSI output.
	program := NewProgram(root, WithoutInput(), tea.WithOutput(&out), tea.WithWindowSize(1, 1))

	go func() {
		program.Send(tea.KeyPressMsg{Code: '~', Text: "~"})

		time.Sleep(50 * time.Millisecond)

		program.Send(tea.KeyPressMsg{Code: 'x', Text: "x"})
	}()

	if _, err := program.Run(); err != nil {
		t.Fatal(err)
	}

	if root.state.echoKey != "~" {
		t.Errorf("expected echoed key \"~\", got %q", root.state.echoKey)
	}

	if WasRouteChanged() {
		t.Errorf("current route was changed")
	}

	if CurrentRoute() != "/test/test/test" {
		t.Errorf("current route is wrong, expected \"/test/test/test\", got \"%s\"", CurrentRoute())
	}

	if root.state.lastWidth != 1 {
		t.Errorf("expected lastWidth 1, but got %d", root.state.lastWidth)
	}

	if root.state.lastHeight != 1 {
		t.Errorf("expected lastHeight 1, but got %d", root.state.lastHeight)
	}
}

// A panic in a command returned from Update must not crash the process with
// the terminal left in raw/alt-screen mode. Reactea should restore the terminal
// (via Kill) and let Run return, mirroring Bubbletea's own panic handling.
func TestPanicInCommandRestoresTerminal(t *testing.T) {
	root := &mockComponent[struct{}]{
		updateFunc: func(c Component, s *struct{}, msg tea.Msg) tea.Cmd {
			if _, ok := msg.(tea.KeyMsg); ok {
				return func() tea.Msg { panic("boom from a command") }
			}

			return nil
		},
		renderFunc: func(c Component, s *struct{}, width, height int) string {
			return "test"
		},
	}

	program := NewProgram(root, WithoutInput(), tea.WithoutRenderer())

	go func() {
		time.Sleep(20 * time.Millisecond)
		program.Send(tea.KeyPressMsg{Code: 'a', Text: "a"})
	}()

	_, err := program.Run()

	// The process survived (no crash) and Run returned; Kill surfaces as
	// ErrProgramKilled.
	if !errors.Is(err, tea.ErrProgramKilled) {
		t.Errorf("expected ErrProgramKilled after a panicking command, got %v", err)
	}
}

func TestNew(t *testing.T) {
	t.Run("NewProgram", func(t *testing.T) {
		root := &mockComponent[struct{}]{
			renderFunc: func(c Component, s *struct{}, width, height int) string {
				return "test passed"
			},
		}

		program := NewProgram(root, WithoutInput(), tea.WithoutRenderer())

		go program.Quit()

		if _, err := program.Run(); err != nil {
			t.Fatal(err)
		}
	})
}
