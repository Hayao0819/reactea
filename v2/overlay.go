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

// Overlay is a host that can put a component above existing content.
// modal.Stack implements it, and Ctx provides the nearest host to a component.
type Overlay interface {
	Push(component Component) tea.Cmd
	PushAt(component Component, placement Placement) tea.Cmd
}

// WithOverlay installs the host for this branch. Nested branches resolve to the
// nearest mounted host.
func (c *Ctx) WithOverlay(host Overlay) *Ctx {
	child := *c
	child.overlay = host

	return &child
}

// Overlay is the nearest host above this component, if there is one.
func (c *Ctx) Overlay() (Overlay, bool) { return c.overlay, c.overlay != nil }
