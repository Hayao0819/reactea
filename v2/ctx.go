package reactea

import (
	tea "charm.land/bubbletea/v2"
)

// Ctx is what a component is told about the frame it is taking part in: the box
// it may draw into, where the app currently is, and the scope its cleanups
// belong to.
//
// A parent hands a child its own box with Inset. Because a Ctx knows its origin
// on screen, a cursor set through it is translated for the component, however
// deep it sits, so no parent does offset arithmetic.
type Ctx struct {
	app   *App
	scope *Scope

	x, y          int
	width, height int
}

func (c *Ctx) Size() (int, int) { return c.width, c.height }

func (c *Ctx) Width() int { return c.width }

func (c *Ctx) Height() int { return c.height }

// Origin is where this box sits on screen. Components rarely need it; layout
// code does.
func (c *Ctx) Origin() (int, int) { return c.x, c.y }

// Inset carves a child box out of this one. dx and dy are relative to this box.
// Sizes are clamped to what is left, so a child can never be handed a negative
// box or one that starts outside its parent.
func (c *Ctx) Inset(dx, dy, width, height int) *Ctx {
	dx, dy = max(0, dx), max(0, dy)

	child := *c
	child.x, child.y = c.x+dx, c.y+dy
	child.width = clamp(width, 0, max(0, c.width-dx))
	child.height = clamp(height, 0, max(0, c.height-dy))

	return &child
}

// WithScope returns the same box bound to another scope. A parent that mounts a
// child it may later unmount gives it one, and closes it when the child goes.
func (c *Ctx) WithScope(scope *Scope) *Ctx {
	child := *c
	child.scope = scope

	return &child
}

// Scope is the scope cleanups registered through this Ctx belong to.
func (c *Ctx) Scope() *Scope { return c.scope }

// OnDestroy registers a cleanup with this Ctx's scope: closing a file, stopping
// a ticker, cancelling a context. It runs when the enclosing scope closes, which
// for most components means when the program ends.
func (c *Ctx) OnDestroy(cleanup func()) { c.scope.OnDestroy(cleanup) }

// Route is the app's current route.
func (c *Ctx) Route() string { return c.app.route }

// PreviousRoute is where the app was before the last route change.
func (c *Ctx) PreviousRoute() string { return c.app.previousRoute }

// SetRoute moves to an absolute route. The move happens when the returned
// command runs, so it is safe from anywhere — including a command goroutine.
func (c *Ctx) SetRoute(target string) tea.Cmd {
	return func() tea.Msg { return routeRequestMsg{target: target} }
}

// Navigate moves to target, which may be relative to the current route (".",
// "..", "sub" all work).
func (c *Ctx) Navigate(target string) tea.Cmd {
	if target == "" {
		return nil
	}

	return func() tea.Msg { return routeRequestMsg{target: target, relative: true} }
}

// SetCursor puts the terminal cursor at a position inside this box, in this
// frame. The cursor is the one thing that cannot be decided outside Render: it
// depends on the layout, which only exists while rendering — which is exactly
// why Bubbletea v2 carries it on the view too. Everything else the terminal can
// be asked for is a command; see EnterAltScreen and friends.
//
// Pass nil to leave the cursor hidden.
func (c *Ctx) SetCursor(cursor *tea.Cursor) {
	if cursor == nil {
		c.app.cursor = nil

		return
	}

	moved := *cursor
	moved.X += c.x
	moved.Y += c.y

	c.app.cursor = &moved
}

// CursorAt is SetCursor for the common case of a plain block cursor.
func (c *Ctx) CursorAt(x, y int) { c.SetCursor(tea.NewCursor(x, y)) }

func clamp(value, low, high int) int { return min(max(value, low), high) }
