package reactea

import tea "charm.land/bubbletea/v2"

// Input messages are the ones with an addressee: a keyboard message goes to
// whatever holds focus, a mouse message to whatever sits under the pointer.
// Everything else — ticks, async results, window size, route changes — is data
// and reaches the whole tree.
//
// Containers route on this distinction. It is what keeps a modal from starving
// the page underneath it of its own results.

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

// IsInput reports whether msg is addressed to one component rather than to all
// of them.
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

// MouseAt returns where msg landed, if it is a mouse event at all.
func MouseAt(msg tea.Msg) (x, y int, ok bool) {
	mouse, isMouse := msg.(tea.MouseMsg)
	if !isMouse {
		return 0, 0, false
	}

	event := mouse.Mouse()

	return event.X, event.Y, true
}
