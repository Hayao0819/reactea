package reactea

import (
	tea "charm.land/bubbletea/v2"
	"image/color"
)

// Ctx is what a component is told about the frame it is taking part in: the box
// it may draw into, where the app currently is, and the handle it uses to ask
// for terminal features Bubbletea v2 moved into tea.View.
//
// A parent hands a child its own box with Inset. Because a Ctx knows its origin
// on screen, anything positional a child reports — a cursor, most of all — is
// translated for it, so no parent does offset arithmetic.
type Ctx struct {
	app *App

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

	return &Ctx{
		app:    c.app,
		x:      c.x + dx,
		y:      c.y + dy,
		width:  clamp(width, 0, max(0, c.width-dx)),
		height: clamp(height, 0, max(0, c.height-dy)),
	}
}

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

// SetCursor puts the terminal cursor at a position inside this box. Pass nil to
// take the cursor back.
func (c *Ctx) SetCursor(cursor *tea.Cursor) {
	if cursor == nil {
		c.app.view.Cursor = nil

		return
	}

	moved := *cursor
	moved.X += c.x
	moved.Y += c.y

	c.app.view.Cursor = &moved
}

// CursorAt is SetCursor for the common case of a plain block cursor.
func (c *Ctx) CursorAt(x, y int) { c.SetCursor(tea.NewCursor(x, y)) }

func (c *Ctx) AltScreen(on bool) { c.app.view.AltScreen = on }

func (c *Ctx) Title(title string) { c.app.view.WindowTitle = title }

func (c *Ctx) MouseMode(mode tea.MouseMode) { c.app.view.MouseMode = mode }

func (c *Ctx) ReportFocus(on bool) { c.app.view.ReportFocus = on }

func (c *Ctx) BackgroundColor(colour color.Color) { c.app.view.BackgroundColor = colour }

func (c *Ctx) ForegroundColor(colour color.Color) { c.app.view.ForegroundColor = colour }

// View exposes the frame's view for a test to inspect after Render.
func (c *Ctx) View() tea.View { return c.app.view }

func clamp(value, low, high int) int { return min(max(value, low), high) }
