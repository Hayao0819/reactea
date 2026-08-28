package reactea

import tea "charm.land/bubbletea/v2"

// Reactified adapts a tea.Model. A bubbles widget returns its own concrete type
// from Update, so it never satisfies tea.Model; use ReactifyWidget for those.
type Reactified[TModel tea.Model] struct {
	BasicComponent

	Model TModel

	view tea.View
}

// Reactify wraps a tea.Model as a Component.
func Reactify[TModel tea.Model](model TModel) *Reactified[TModel] {
	return &Reactified[TModel]{Model: model}
}

func (c *Reactified[TModel]) Init(*Ctx) tea.Cmd {
	return c.Model.Init()
}

func (c *Reactified[TModel]) Update(ctx *Ctx, msg tea.Msg) tea.Cmd {
	// The app broadcasts the terminal's size; a nested model must be told its own
	// box instead, or it draws as if it owned the screen.
	if _, ok := msg.(tea.WindowSizeMsg); ok {
		width, height := ctx.Size()
		msg = tea.WindowSizeMsg{Width: width, Height: height}
	}

	updated, cmd := c.Model.Update(msg)

	// Value receivers: the returned value has to be stored back or state is lost.
	model, ok := updated.(TModel)
	if !ok {
		panic("reactea: wrapped model's Update returned a different concrete type")
	}

	c.Model = model

	return cmd
}

func (c *Reactified[TModel]) Render(ctx *Ctx) string {
	c.view = c.Model.View()

	if c.view.Cursor != nil {
		ctx.SetCursor(c.view.Cursor)
	}

	return c.view.Content
}

// Widget is the shape every bubbles widget has. The self-referential type
// parameter is what lets one adapter cover textinput, textarea, viewport, list
// and friends. An interface whose Update returns itself also fits: name it as
// the type argument, as in ReactifyWidget[huh.Model](form).
type Widget[T any] interface {
	Update(tea.Msg) (T, tea.Cmd)
	View() string
}

// ReactifiedWidget adapts a bubbles widget.
type ReactifiedWidget[TWidget Widget[TWidget]] struct {
	BasicComponent

	Widget TWidget
}

// ReactifyWidget wraps a bubbles widget as a Component.
func ReactifyWidget[TWidget Widget[TWidget]](widget TWidget) *ReactifiedWidget[TWidget] {
	return &ReactifiedWidget[TWidget]{Widget: widget}
}

// Only some widgets (timer, stopwatch, filepicker, progress) have an Init, which
// is why it is not part of Widget.
func (c *ReactifiedWidget[TWidget]) Init(*Ctx) tea.Cmd {
	if initializer, ok := any(c.Widget).(interface{ Init() tea.Cmd }); ok {
		return initializer.Init()
	}

	return nil
}

func (c *ReactifiedWidget[TWidget]) Update(_ *Ctx, msg tea.Msg) tea.Cmd {
	updated, cmd := c.Widget.Update(msg)

	// Value receivers: the returned value has to be stored back or state is lost.
	c.Widget = updated

	return cmd
}

// A widget exposes Cursor() separately from View(), and only once its virtual
// cursor is off and it is focused. Reactea forces neither.
func (c *ReactifiedWidget[TWidget]) Render(ctx *Ctx) string {
	if reporter, ok := any(c.Widget).(interface{ Cursor() *tea.Cursor }); ok {
		if cursor := reporter.Cursor(); cursor != nil {
			ctx.SetCursor(cursor)
		}
	}

	return c.Widget.View()
}
