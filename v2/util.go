package reactea

import tea "charm.land/bubbletea/v2"

// Reactified adapts a whole Bubbletea model. This is for types that satisfy
// tea.Model — a self-contained program you want to nest — not a bubbles widget.
// A widget returns its own concrete type from Update and a plain string from
// View, so it never satisfies tea.Model; use ReactifyWidget for those.
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

func (c *Reactified[TModel]) Update(_ *Ctx, msg tea.Msg) tea.Cmd {
	updated, cmd := c.Model.Update(msg)

	// Models are usually value receivers: Update returns a NEW value, so it has
	// to be stored back or all of its state is lost.
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

// Widget is the shape every bubbles widget has: Update returns the widget's own
// concrete type (which is why a widget never satisfies tea.Model) and View
// returns a plain string. The self-referential type parameter is what lets one
// adapter cover textinput, textarea, viewport, list, table and friends.
//
// An interface whose Update returns the interface itself also satisfies this;
// name it as the type argument to wrap one, as in
// ReactifyWidget[huh.Model](form).
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

// Init runs the widget's own Init when it has one. Only some widgets (timer,
// stopwatch, filepicker, progress) do, so it is not part of Widget.
func (c *ReactifiedWidget[TWidget]) Init(*Ctx) tea.Cmd {
	if initializer, ok := any(c.Widget).(interface{ Init() tea.Cmd }); ok {
		return initializer.Init()
	}

	return nil
}

func (c *ReactifiedWidget[TWidget]) Update(_ *Ctx, msg tea.Msg) tea.Cmd {
	updated, cmd := c.Widget.Update(msg)

	// Widgets are value receivers: Update returns a NEW value, so it has to be
	// stored back or all state (text, cursor position, scroll offset) is lost.
	c.Widget = updated

	return cmd
}

// Render draws the widget and reports its real terminal cursor. Widgets that
// can do this expose Cursor() separately from View() and only return one once
// the caller has turned the virtual cursor off
// (textinput.SetVirtualCursor(false)) and focused the widget — reactea forces
// neither, since the virtual cursor drawn into the string is still the default.
func (c *ReactifiedWidget[TWidget]) Render(ctx *Ctx) string {
	if reporter, ok := any(c.Widget).(interface{ Cursor() *tea.Cursor }); ok {
		if cursor := reporter.Cursor(); cursor != nil {
			ctx.SetCursor(cursor)
		}
	}

	return c.Widget.View()
}
