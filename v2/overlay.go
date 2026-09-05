package reactea

import tea "charm.land/bubbletea/v2"

// Placement is where something sits inside a box. A zero Width or Height spans
// that axis; an X or Y of Center puts it in the middle.
type Placement struct {
	X, Y          int
	Width, Height int
}

// Center asks for the middle of the axis.
const Center = -1

// Overlay is a host that can put a component above whatever is already drawn.
// modal.Stack is one; a component finds the nearest through its Ctx rather than
// being handed a callback from wherever the host happens to be built.
type Overlay interface {
	Push(component Component) tea.Cmd
	PushAt(component Component, placement Placement) tea.Cmd
}

// WithOverlay names the host for this branch. A host installs itself as it
// routes, so the nearest one wins and an unmounted one leaves nothing behind.
func (c *Ctx) WithOverlay(host Overlay) *Ctx {
	child := *c
	child.overlay = host

	return &child
}

// Overlay is the nearest host above this component, if there is one.
func (c *Ctx) Overlay() (Overlay, bool) { return c.overlay, c.overlay != nil }
