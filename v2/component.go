package reactea

import tea "charm.land/bubbletea/v2"

// Component is a piece of UI. Update is not guaranteed to run before the first
// Render, so put anything critical in Init. There is no Destroy; register
// cleanups with Ctx.OnDestroy so a parent cannot leak a child by forgetting to
// forward one.
type Component interface {
	Init(*Ctx) tea.Cmd
	Update(*Ctx, tea.Msg) tea.Cmd

	// Render should be a function of the component's state: ask for terminal
	// features with commands, not while drawing. The cursor is the exception,
	// since it depends on the layout.
	Render(*Ctx) string
}

// BasicComponent implements Init and Update as no-ops.
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

// Func turns a render function into a Component.
func Func(render RenderFunc) Component { return &funcComponent{render: render} }

// Text is a Component that always renders content.
func Text(content string) Component {
	return Func(func(*Ctx) string { return content })
}
