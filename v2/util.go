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

// Reactifies Bubbletea's models into reactea's components, so existing
// Bubbletea/bubbles widgets drop straight into the component tree.
type Reactified[TModel tea.Model] struct {
	BasicComponent

	Model TModel
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

	// bubbles widgets are value-receiver models: Update returns a NEW model
	// value, so we must store it back or all widget state (cursor, text, scroll
	// offset) is lost. The assertion guards against a model whose Update returns
	// a different concrete type.
	if model, ok := updated.(TModel); ok {
		c.Model = model
	}

	return cmd
}

func (c *Reactified[TModel]) Render(width, height int) string {
	// In v2 a model's View() returns a tea.View; components render to a string,
	// so hand back the view's content.
	return c.Model.View().Content
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
