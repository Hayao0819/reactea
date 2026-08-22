package reactea

import tea "charm.land/bubbletea/v2"

type RerenderMsg struct{}

// Utility tea.Cmd for requesting rerender
func Rerender() tea.Msg {
	return RerenderMsg{}
}

// Renders all AnyRenderers in one function
//
// Note: If you are using ProplessRenderer/DumbRenderer just pass
// reactea.NoProps{} or struct{}{}
//
// Note: Using named return type for 100% coverage
func RenderAny[TProps any, TRenderer AnyRenderer[TProps]](renderer TRenderer, props TProps, width, height int) (result string) {
	switch renderer := any(renderer).(type) {
	case Renderer[TProps]:
		result = renderer(props, width, height)
	case ProplessRenderer:
		result = renderer(width, height)
	case DumbRenderer:
		result = renderer()
	}

	return
}

// Handles rendering of all AnyProplessRenderers in one function
//
// Note: Using named return type for 100% coverage
func RenderDumb[TRenderer AnyProplessRenderer](renderer TRenderer, width, height int) (result string) {
	switch renderer := any(renderer).(type) {
	case ProplessRenderer:
		result = renderer(width, height)
	case DumbRenderer:
		result = renderer()
	}

	return
}

// Wraps propful into propless renderer
func PropfulToLess[TProps any](renderer Renderer[TProps], props TProps) ProplessRenderer {
	return func(width, height int) string {
		return renderer(props, width, height)
	}
}

// Static component for displaying static text
type staticComponent struct {
	BasicComponent

	content string
}

func (c *staticComponent) Render(int, int) string { return c.content }

func StaticComponent(content string) Component {
	return &staticComponent{content: content}
}

// Transformer for AnyRenderer -> Component
type componentTransformer[TProps any, TRenderer AnyRenderer[TProps]] struct {
	BasicComponent

	props    TProps
	renderer TRenderer
}

func (c *componentTransformer[TProps, TRenderer]) Render(width, height int) string {
	return RenderAny(c.renderer, c.props, width, height)
}

// Componentifies AnyRenderer
// Returns uninitialized component with renderer taking care of .Render()
func Componentify[TProps any, TRenderer AnyRenderer[TProps]](renderer TRenderer, props TProps) Component {
	return &componentTransformer[TProps, TRenderer]{renderer: renderer, props: props}
}

// Transformer for AnyProplessRenderer -> Component
type dumbComponentTransformer[TRenderer AnyProplessRenderer] struct {
	BasicComponent

	renderer TRenderer
}

func (c *dumbComponentTransformer[T]) Render(width, height int) string {
	return RenderDumb(c.renderer, width, height)
}

// Componentifies AnyProplessRenderer
// Returns uninitialized component with renderer taking care of .Render()
func ComponentifyDumb[TRenderer AnyProplessRenderer](renderer TRenderer) Component {
	return &dumbComponentTransformer[TRenderer]{renderer: renderer}
}

// Reactifies a whole Bubbletea model into a reactea component. This is for
// types that satisfy tea.Model — a self-contained program you want to nest, not
// a bubbles widget. Widgets return their own concrete type from Update and a
// plain string from View, so they never satisfy tea.Model; use ReactifyWidget
// for those.
type Reactified[TModel tea.Model] struct {
	BasicComponent

	Model TModel

	view tea.View
}

// Reactify wraps a tea.Model as a reactea.Component.
func Reactify[TModel tea.Model](model TModel) *Reactified[TModel] {
	return &Reactified[TModel]{Model: model}
}

func (c *Reactified[TModel]) Init() tea.Cmd {
	return c.Model.Init()
}

func (c *Reactified[TModel]) Update(msg tea.Msg) tea.Cmd {
	updated, cmd := c.Model.Update(msg)

	// Models are usually value receivers: Update returns a NEW value, so we must
	// store it back or all of its state is lost. The assertion guards against a
	// model whose Update returns a different concrete type.
	if model, ok := updated.(TModel); ok {
		c.Model = model
	}

	return cmd
}

func (c *Reactified[TModel]) Render(width, height int) string {
	// In v2 a model's View() returns a tea.View; components render to a string,
	// so hand back the view's content and keep the rest for DecorateView.
	c.view = c.Model.View()

	return c.view.Content
}

// DecorateView forwards the wrapped model's cursor, which is how a v2 text
// input reports where it wants the terminal cursor. The position is relative to
// the widget's own render, so a parent that draws it at an offset has to call
// TranslateCursor. The view's other fields stay behind: a widget nested in a
// tree has no business flipping the alt-screen or the window title.
func (c *Reactified[TModel]) DecorateView(view *tea.View) {
	if c.view.Cursor != nil {
		view.Cursor = c.view.Cursor
	}
}

// Widget is the shape every bubbles widget has: Update returns the widget's own
// concrete type (which is why a widget never satisfies tea.Model) and View
// returns a plain string. The self-referential type parameter is what lets one
// adapter cover textinput, textarea, viewport, list, table and friends.
type Widget[T any] interface {
	Update(tea.Msg) (T, tea.Cmd)
	View() string
}

// Reactifies a bubbles widget into a reactea component.
type ReactifiedWidget[TWidget Widget[TWidget]] struct {
	BasicComponent

	Widget TWidget
}

// ReactifyWidget wraps a bubbles widget as a reactea.Component.
func ReactifyWidget[TWidget Widget[TWidget]](widget TWidget) *ReactifiedWidget[TWidget] {
	return &ReactifiedWidget[TWidget]{Widget: widget}
}

// Init runs the widget's own Init when it has one. Only some widgets (timer,
// stopwatch, filepicker, progress) do, so it is not part of Widget.
func (c *ReactifiedWidget[TWidget]) Init() tea.Cmd {
	if initializer, ok := any(c.Widget).(interface{ Init() tea.Cmd }); ok {
		return initializer.Init()
	}

	return nil
}

func (c *ReactifiedWidget[TWidget]) Update(msg tea.Msg) tea.Cmd {
	updated, cmd := c.Widget.Update(msg)

	// Widgets are value receivers: Update returns a NEW value, so it has to be
	// stored back or all state (text, cursor position, scroll offset) is lost.
	c.Widget = updated

	return cmd
}

func (c *ReactifiedWidget[TWidget]) Render(width, height int) string {
	return c.Widget.View()
}

// DecorateView reports the widget's real terminal cursor. Widgets that can do
// this expose Cursor() separately from View() and only return a cursor once the
// caller has turned the virtual cursor off (textinput.SetVirtualCursor(false))
// and focused the widget — reactea does not force either, since the virtual
// cursor drawn into the string is still the default and works fine.
//
// The position is relative to the widget's own render, so a parent that draws it
// at an offset has to call TranslateCursor.
func (c *ReactifiedWidget[TWidget]) DecorateView(view *tea.View) {
	reporter, ok := any(c.Widget).(interface{ Cursor() *tea.Cursor })
	if !ok {
		return
	}

	if cursor := reporter.Cursor(); cursor != nil {
		view.Cursor = cursor
	}
}

// Used for tests
type mockComponent[TState any] struct {
	initFunc    func(Component, *TState) tea.Cmd
	updateFunc  func(Component, *TState, tea.Msg) tea.Cmd
	renderFunc  func(Component, *TState, int, int) string
	destroyFunc func(Component, *TState)

	state TState
}

func (c *mockComponent[TState]) Init() tea.Cmd {
	if c.initFunc == nil {
		return nil
	}

	return c.initFunc(c, &c.state)
}

func (c *mockComponent[TState]) Update(msg tea.Msg) tea.Cmd {
	if c.updateFunc == nil {
		return nil
	}

	return c.updateFunc(c, &c.state, msg)
}

func (c *mockComponent[TState]) Render(width, height int) string {
	if c.renderFunc == nil {
		return ""
	}

	return c.renderFunc(c, &c.state, width, height)
}

func (c *mockComponent[TState]) Destroy() {
	if c.destroyFunc == nil {
		return
	}

	c.destroyFunc(c, &c.state)
}
