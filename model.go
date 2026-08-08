package reactea

import (
	"fmt"
	"runtime/debug"

	tea "github.com/charmbracelet/bubbletea"
)

// Useful for constraining some actions to update-stage only
var isUpdate bool

// Internal message used to run teardown on the event-loop goroutine. The
// destroyAppMsg path can't call root.Destroy() directly from execute's
// goroutine, because that would race the event loop still calling
// root.Update()/root.Render(). Instead execute forwards this marker back
// through the program so Destroy() runs serialized inside Update().
type destroyNowMsg struct{}

type model struct {
	program *tea.Program
	root    Component

	width, height int
}

func (m model) Init() tea.Cmd {
	return m.root.Init()
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Teardown runs here, on the event-loop goroutine, so it can't interleave
	// with root.Update()/root.Render(). tea.Quit then stops the loop, so no
	// further Update lands on the destroyed component tree.
	if _, ok := msg.(destroyNowMsg); ok {
		m.root.Destroy()
		return m, tea.Quit
	}

	switch msg := msg.(type) {
	// We want component to know at what size should it render
	// and unify size handling across all Reactea components
	// We pass WindowSizeMsg to root component just for
	// sake of utility.
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	}

	beginUpdate()

	m.execute(m.root.Update(msg))

	previousRoute, routeChanged := endUpdate()

	// Guarantee rerender if route was changed
	if routeChanged {
		return m, updatedRoute(previousRoute)
	}

	return m, nil
}

func (m model) View() string {
	return m.root.Render(m.width, m.height)
}

func (m model) execute(cmd tea.Cmd) {
	if cmd == nil {
		return
	}

	go func() {
		// Bubbletea runs Update commands under a recover that restores the
		// terminal on panic (see (*tea.Program).recoverFromPanic). Reactea runs
		// them in its own goroutine, which bypasses that guard, so mirror it
		// here — otherwise a panicking command (an HTTP call, a JSON decode, a
		// stream read) crashes the process and leaves the terminal in raw or
		// alt-screen mode. Kill() restores the terminal and makes Run() return
		// ErrProgramKilled.
		defer func() {
			if r := recover(); r != nil {
				m.program.Kill()
				fmt.Printf("Caught panic:\n\n%s\n\nRestoring terminal...\n\n", r)
				debug.PrintStack()
			}
		}()

		msg := cmd()
		switch msg := msg.(type) {
		case destroyAppMsg:
			// Don't Destroy() here — that would race the event loop. Hand
			// teardown back to Update() via the program's message channel.
			m.program.Send(destroyNowMsg{})
		case tea.BatchMsg:
			for _, cmd := range msg {
				m.execute(cmd)
			}
		default:
			m.program.Send(msg)
		}
	}()
}
