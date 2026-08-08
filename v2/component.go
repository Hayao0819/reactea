package reactea

import tea "charm.land/bubbletea/v2"

// The lifecycle is
//
//           \/ Usually won't be called on first render
// Init ---> Update -> Render ---> Destroy?
//       |                     |   /\ implementation detail and
//       |---------------------|   therefore doesn't return tea.Cmd
//
// Reactea takes pointer approach for components
// making state mutable in any lifecycle method
//
// Note: Lifecycle is fully controlled by parent component
// making graph above fully theoretical and possibly
// invalid for third-party components

type Component interface {
	// Common lifecycle methods

	// Init() Is meant to both initialize subcomponents and run
	// long IO operations through tea.Cmd
	Init() tea.Cmd

	// It's called when component is about to be destroyed
	Destroy()

	// Typical tea.Model Update(), we handle all IO events here
	Update(tea.Msg) tea.Cmd

	// Render() is called when component should render itself
	// Provided width and height are target dimensions
	Render(int, int) string
}

// ViewDecorator is an optional Component interface. Render() still produces the
// content; a component that also needs one of the terminal features Bubbletea
// v2 moved into tea.View (cursor, alt-screen, window title, mouse mode, colors)
// implements this and sets the fields it cares about.
//
// Only the root is asked directly, so a composite component has to pass the
// view down to whichever children it renders — see DecorateView.
type ViewDecorator interface {
	DecorateView(*tea.View)
}

// DecorateView lets component decorate view when it implements ViewDecorator.
// Composite components call it on each child they render.
func DecorateView(component Component, view *tea.View) {
	if decorator, ok := component.(ViewDecorator); ok {
		decorator.DecorateView(view)
	}
}

// TranslateCursor moves a cursor set by a child into the parent's coordinate
// space. Reactea has no layout engine, so a parent that draws a child at an
// offset is the only one that knows the offset and has to apply it.
func TranslateCursor(view *tea.View, dx, dy int) {
	if view.Cursor != nil {
		view.Cursor.X += dx
		view.Cursor.Y += dy
	}
}

// AnyRenderer is the set of stateless renderer function types.
type AnyRenderer[TProps any] interface {
	Renderer[TProps] | AnyProplessRenderer
}

type AnyProplessRenderer interface {
	ProplessRenderer | DumbRenderer
}

// Ultra shorthand for components = just renderer
// One could say it's a stateless component
// Also note that it doesn't handle any IO by itself
//
// It's a type alias, so a Renderer[TProps] value and a plain
// func(TProps, int, int) string are interchangeable without a conversion.
type Renderer[TProps any] = func(TProps, int, int) string

// SUPEEEEEER shorthand for components
type ProplessRenderer = func(int, int) string

// Doesn't have state, props, even scalling for
// target dimensions = DumbRenderer, or Stringer
type DumbRenderer = func() string

// Alias for no props
type NoProps = struct{}

// Basic component that implements all methods
// required by reactea.Component
// except Render(int, int)
type BasicComponent struct{}

func (c *BasicComponent) Init() tea.Cmd              { return nil }
func (c *BasicComponent) Destroy()                   {}
func (c *BasicComponent) Update(msg tea.Msg) tea.Cmd { return nil }

// Utility component for displaying empty string on Render()
type InvisibleComponent struct{}

func (c *InvisibleComponent) Render(int, int) string { return "" }

// Destroys app before quiting
func Destroy() tea.Msg {
	return destroyAppMsg{}
}

type destroyAppMsg struct{}
