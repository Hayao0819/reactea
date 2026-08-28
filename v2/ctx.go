package reactea

import (
	tea "charm.land/bubbletea/v2"
)

// Ctx is the box a component may draw into, where the app is, and the scope its
// cleanups belong to. A Ctx knows its origin on screen, so a cursor set through
// it is translated for the component and no parent does offset arithmetic.
type Ctx struct {
	app   *App
	scope *Scope

	x, y          int
	width, height int
	focused       bool
}

// Size is the box this component may draw into.
func (c *Ctx) Size() (int, int) { return c.width, c.height }

func (c *Ctx) Width() int { return c.width }

func (c *Ctx) Height() int { return c.height }

// Origin is where this box sits on screen.
func (c *Ctx) Origin() (int, int) { return c.x, c.y }

// Inset carves a child box out of this one, clamped to what is left so a child
// can never start outside its parent.
func (c *Ctx) Inset(dx, dy, width, height int) *Ctx {
	dx, dy = max(0, dx), max(0, dy)

	child := *c
	child.x, child.y = c.x+dx, c.y+dy
	child.width = clamp(width, 0, max(0, c.width-dx))
	child.height = clamp(height, 0, max(0, c.height-dy))

	return &child
}

// Focused reports whether this component holds the keyboard focus. Containers
// decide it; a component reads it to style itself and to know whether keys are
// meant for it.
func (c *Ctx) Focused() bool { return c.focused }

// WithFocus marks the child branch as holding, or not holding, the focus.
// Containers call it as they route.
func (c *Ctx) WithFocus(focused bool) *Ctx {
	child := *c
	child.focused = focused

	return &child
}

// WithScope binds the same box to another scope, for a child that may later be
// unmounted on its own.
func (c *Ctx) WithScope(scope *Scope) *Ctx {
	child := *c
	child.scope = scope

	return &child
}

// Scope is the scope cleanups registered here belong to.
func (c *Ctx) Scope() *Scope { return c.scope }

// OnDestroy registers a cleanup with this Ctx's scope.
func (c *Ctx) OnDestroy(cleanup func()) { c.scope.OnDestroy(cleanup) }

// Route is the app's current route.
func (c *Ctx) Route() string { return c.app.route }

// PreviousRoute is where the app was before the last route change.
func (c *Ctx) PreviousRoute() string { return c.app.previousRoute }

// SetRoute moves to an absolute route when the returned command runs, so it is
// safe to call from any goroutine.
func (c *Ctx) SetRoute(target string) tea.Cmd {
	return func() tea.Msg { return routeRequestMsg{target: target} }
}

// Navigate accepts a route relative to the current one.
func (c *Ctx) Navigate(target string) tea.Cmd {
	if target == "" {
		return nil
	}

	return func() tea.Msg { return routeRequestMsg{target: target, relative: true} }
}

// SetCursor places the cursor inside this box for this frame. It stays a render
// concern because it depends on the layout; everything else the terminal can be
// asked for is a command. Pass nil to hide it.
//
// A component without the focus is ignored, so one cursor per frame falls out of
// the focus rules instead of being a race between siblings.
func (c *Ctx) SetCursor(cursor *tea.Cursor) {
	if !c.focused {
		return
	}

	if cursor == nil {
		c.app.cursor = nil

		return
	}

	moved := *cursor
	moved.X += c.x
	moved.Y += c.y

	c.app.cursor = &moved
}

// CursorAt is SetCursor with a plain block cursor.
func (c *Ctx) CursorAt(x, y int) { c.SetCursor(tea.NewCursor(x, y)) }

func clamp(value, low, high int) int { return min(max(value, low), high) }
