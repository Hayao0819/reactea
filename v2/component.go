package reactea

import tea "charm.land/bubbletea/v2"

// Component is a piece of UI. Three methods, one argument in common.
//
//	Init ---> Update -> Render
//	      |            /\ Update is not guaranteed to run before the first
//	      |---------->    Render, so put anything critical in Init
//
// Components are pointers and mutate in place; Update returns work to do, not a
// new component.
//
// There is no Destroy. A component that owns a resource registers a cleanup with
// Ctx.OnDestroy instead, which means no parent can leak a child by forgetting to
// forward a teardown call. See Scope.
type Component interface {
	// Init initialises subcomponents and kicks off long IO through tea.Cmd.
	Init(*Ctx) tea.Cmd

	// Update handles a message and may return work to do.
	Update(*Ctx, tea.Msg) tea.Cmd

	// Render draws into the box the Ctx describes. It should be a function of
	// the component's state: ask for terminal features with commands
	// (EnterAltScreen, SetWindowTitle) rather than while drawing. The cursor is
	// the exception — it depends on the layout, so it is set through the Ctx.
	Render(*Ctx) string
}

// BasicComponent implements every lifecycle method except Render.
type BasicComponent struct{}

func (c *BasicComponent) Init(*Ctx) tea.Cmd            { return nil }
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
