package reactea

import tea "charm.land/bubbletea/v2"

// Component is a piece of UI. The lifecycle permits Render before the first
// Update, so Init establishes critical state. Ctx.OnDestroy binds cleanup to
// the component's mounted scope.
type Component interface {
	Init(*Ctx) tea.Cmd
	Update(*Ctx, tea.Msg) tea.Cmd

	// Render should be a function of the component's state. Request terminal
	// features with commands and set the layout-dependent cursor through Ctx.
	Render(*Ctx) string
}

// BasicComponent provides default Init and Update implementations.
type BasicComponent struct{}

func (c *BasicComponent) Init(*Ctx) tea.Cmd            { return nil }
func (c *BasicComponent) Update(*Ctx, tea.Msg) tea.Cmd { return nil }

// InvisibleComponent renders an empty string.
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
