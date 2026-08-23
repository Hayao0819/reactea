package modal

import (
	"testing"
	"time"

	"github.com/Hayao0819/reactea"
	tea "github.com/charmbracelet/bubbletea"
)

// promptModal is a minimal modal that returns its name when Enter is pressed.
type promptModal struct {
	reactea.BasicComponent
	Modal[string]

	name string
}

func (m *promptModal) Render(int, int) string { return "prompt:" + m.name }

func (m *promptModal) Update(msg tea.Msg) tea.Cmd {
	if key, ok := msg.(tea.KeyMsg); ok && key.Type == tea.KeyEnter {
		return m.Ok(m.name)
	}

	return nil
}

// A single modal that returns must not deadlock the event loop — this is the
// case that the old waiter-based handshake could freeze permanently.
func TestControllerSingleModal(t *testing.T) {
	result := make(chan string, 1)

	ctrl := NewController(func(c *Controller) func() tea.Cmd {
		result <- Show(c, &promptModal{name: "hello"}).Return

		return func() tea.Cmd { return reactea.Destroy }
	})

	program := reactea.NewProgram(ctrl, reactea.WithoutInput(), tea.WithoutRenderer())

	go func() {
		time.Sleep(20 * time.Millisecond)
		program.Send(tea.KeyMsg{Type: tea.KeyEnter})
	}()

	if _, err := program.Run(); err != nil {
		t.Fatal(err)
	}

	select {
	case got := <-result:
		if got != "hello" {
			t.Errorf("expected \"hello\", got %q", got)
		}
	default:
		t.Error("modal never returned a result")
	}
}

// A flow of two modals shown in sequence must deliver both results in order and
// tear down cleanly.
func TestControllerSequentialModals(t *testing.T) {
	out := make(chan []string, 1)
	step := make(chan struct{})

	ctrl := NewController(func(c *Controller) func() tea.Cmd {
		var got []string

		got = append(got, Show(c, &promptModal{name: "first"}).Return)
		step <- struct{}{}

		got = append(got, Show(c, &promptModal{name: "second"}).Return)
		step <- struct{}{}

		out <- got

		return func() tea.Cmd { return reactea.Destroy }
	})

	program := reactea.NewProgram(ctrl, reactea.WithoutInput(), tea.WithoutRenderer())

	go func() {
		enter := tea.KeyMsg{Type: tea.KeyEnter}

		program.Send(enter) // completes the first modal
		<-step
		program.Send(enter) // completes the second modal
		<-step
	}()

	if _, err := program.Run(); err != nil {
		t.Fatal(err)
	}

	got := <-out
	if len(got) != 2 || got[0] != "first" || got[1] != "second" {
		t.Errorf("expected [first second], got %v", got)
	}
}
