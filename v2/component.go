package reactea

import tea "charm.land/bubbletea/v2"

// The lifecycle is
//
//	\/ Usually won't be called on first render
//
// Init ---> Update -> Render ---> Destroy?
//
//	|                     |   /\ implementation detail and
//	|---------------------|   therefore doesn't return tea.Cmd
//
// Reactea takes pointer approach for components
// making state mutable in any lifecycle method
//
// Note: Lifecycle is fully controlled by parent component
// making graph above fully theoretical and possibly
// invalid for third-party components
type Component interface {
	// Init initializes subcomponents and kicks off long IO through tea.Cmd.
	Init(*Ctx) tea.Cmd

	// Update handles a message and may return work to do.
	Update(*Ctx, tea.Msg) tea.Cmd

	// Render draws into the box the Ctx describes, and asks for whatever
	// terminal features it needs through that same Ctx.
	Render(*Ctx) string

	// Destroy is called when the component is about to end its lifecycle. A
	// parent is responsible for destroying its children.
	Destroy()
}

// BasicComponent implements every lifecycle method except Render.
type BasicComponent struct{}

func (c *BasicComponent) Init(*Ctx) tea.Cmd            { return nil }
func (c *BasicComponent) Destroy()                     {}
func (c *BasicComponent) Update(*Ctx, tea.Msg) tea.Cmd { return nil }

// InvisibleComponent renders nothing.
type InvisibleComponent struct{}

func (c *InvisibleComponent) Render(*Ctx) string { return "" }

// RenderFunc is a stateless component: props are whatever the closure captures.
type RenderFunc = func(*Ctx) string

type funcComponent struct {
	BasicComponent

	render RenderFunc
}

func (c *funcComponent) Render(ctx *Ctx) string { return c.render(ctx) }

// Func turns a render function into a component.
func Func(render RenderFunc) Component { return &funcComponent{render: render} }

// Text is a component that always renders the same string.
func Text(content string) Component {
	return Func(func(*Ctx) string { return content })
}
