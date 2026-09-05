package reactea

import tea "charm.land/bubbletea/v2"

// Reactified adapts a tea.Model. ReactifyWidget handles bubbles widgets whose
// Update method returns a concrete widget type.
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
	// Translate the terminal size broadcast to the nested model's own box.
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

	resize func(TWidget, int, int) TWidget
	width  int
	height int

	Widget TWidget
}

// ReactifyWidget wraps a bubbles widget as a Component.
func ReactifyWidget[TWidget Widget[TWidget]](widget TWidget) *ReactifiedWidget[TWidget] {
	return &ReactifiedWidget[TWidget]{Widget: widget}
}

// OnResize tells a widget the size of its box through its native setters.
//
//	reactea.ReactifyWidget(vp).OnResize(func(v viewport.Model, w, h int) viewport.Model {
//	    v.SetWidth(w)
//	    v.SetHeight(h)
//
//	    return v
//	})
func (c *ReactifiedWidget[TWidget]) OnResize(resize func(TWidget, int, int) TWidget) *ReactifiedWidget[TWidget] {
	c.resize = resize

	return c
}

// Init invokes the optional initializer implemented by widgets such as timer,
// stopwatch, filepicker and progress.
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

// Render forwards a widget's optional real cursor through Ctx.
func (c *ReactifiedWidget[TWidget]) Render(ctx *Ctx) string {
	if width, height := ctx.Size(); c.resize != nil && (width != c.width || height != c.height) {
		c.width, c.height = width, height
		c.Widget = c.resize(c.Widget, width, height)
	}

	if reporter, ok := any(c.Widget).(interface{ Cursor() *tea.Cursor }); ok {
		if cursor := reporter.Cursor(); cursor != nil {
			ctx.SetCursor(cursor)
		}
	}

	return c.Widget.View()
}
