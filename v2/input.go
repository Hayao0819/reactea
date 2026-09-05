package reactea

import (
	"sync"

	tea "charm.land/bubbletea/v2"
)

// Input messages are the ones with an addressee: a keyboard message goes to
// whatever holds focus, a mouse message to whatever sits under the pointer.
// Everything else — ticks, async results, window size, route changes — is data
// and reaches the whole tree.
//
// Containers route on this distinction. It is what keeps a modal from starving
// the page underneath it of its own results.

// InputCapture owns one claim on global input. Its zero value is ready to use.
type InputCapture struct {
	state *inputCaptureState
}

type inputCaptureState struct {
	mu         sync.Mutex
	app        *App
	scope      *Scope
	generation uint64
	held       bool
}

// CaptureInput claims global input until ReleaseInput is called or ctx's scope
// closes. Repeated calls for the same scope are idempotent.
func (c *InputCapture) CaptureInput(ctx *Ctx) tea.Cmd {
	if c.state == nil {
		c.state = &inputCaptureState{}
	}
	state := c.state

	state.mu.Lock()
	if state.held && state.app == ctx.app && state.scope == ctx.scope {
		state.mu.Unlock()

		return nil
	}

	if state.held {
		state.app.addCapture(-1)
	}

	state.generation++
	generation := state.generation
	state.app, state.scope, state.held = ctx.app, ctx.scope, true
	state.app.addCapture(1)
	state.mu.Unlock()

	ctx.scope.OnDestroy(func() { state.release(generation) })

	return nil
}

// ReleaseInput releases this capture and is idempotent.
func (c *InputCapture) ReleaseInput() tea.Cmd {
	if c.state == nil {
		return nil
	}

	c.state.release(0)

	return nil
}

func (c *inputCaptureState) release(generation uint64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.held || generation != 0 && c.generation != generation {
		return
	}

	c.held = false
	c.app.addCapture(-1)
}

// IsKeyboard reports whether msg is a key or paste event.
func IsKeyboard(msg tea.Msg) bool {
	switch msg.(type) {
	case tea.KeyPressMsg, tea.KeyReleaseMsg,
		tea.PasteMsg, tea.PasteStartMsg, tea.PasteEndMsg:
		return true
	}

	return false
}

// IsMouse reports whether msg is a mouse event.
func IsMouse(msg tea.Msg) bool {
	_, ok := msg.(tea.MouseMsg)

	return ok
}

// IsInput reports whether msg has a single component as its addressee.
func IsInput(msg tea.Msg) bool { return IsKeyboard(msg) || IsMouse(msg) }

// TranslateMouse moves a mouse event into a child's coordinate space. A
// container that insets a child translates as it routes, so a component always
// reads box-local coordinates.
func TranslateMouse(msg tea.Msg, dx, dy int) tea.Msg {
	switch mouse := msg.(type) {
	case tea.MouseClickMsg:
		mouse.X, mouse.Y = mouse.X-dx, mouse.Y-dy

		return mouse

	case tea.MouseReleaseMsg:
		mouse.X, mouse.Y = mouse.X-dx, mouse.Y-dy

		return mouse

	case tea.MouseWheelMsg:
		mouse.X, mouse.Y = mouse.X-dx, mouse.Y-dy

		return mouse

	case tea.MouseMotionMsg:
		mouse.X, mouse.Y = mouse.X-dx, mouse.Y-dy

		return mouse
	}

	return msg
}

// MouseAt returns the coordinates and type-match result for a mouse event.
func MouseAt(msg tea.Msg) (x, y int, ok bool) {
	mouse, isMouse := msg.(tea.MouseMsg)
	if !isMouse {
		return 0, 0, false
	}

	event := mouse.Mouse()

	return event.X, event.Y, true
}
