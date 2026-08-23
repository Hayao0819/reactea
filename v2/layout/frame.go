package layout

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/Hayao0819/reactea/v2"
)

// Frame draws a lipgloss style around a component. Lipgloss counts Width and
// Height as the outer size, so the child is rendered at the box minus border,
// padding and margin, with its cursor shifted to match.
type Frame struct {
	style     lipgloss.Style
	component reactea.Component
}

// Framed wraps component in style.
func Framed(style lipgloss.Style, component reactea.Component) *Frame {
	return &Frame{style: style, component: component}
}

func (f *Frame) Init(ctx *reactea.Ctx) tea.Cmd { return f.component.Init(f.inner(ctx)) }

func (f *Frame) Update(ctx *reactea.Ctx, msg tea.Msg) tea.Cmd {
	return f.component.Update(f.inner(ctx), msg)
}

func (f *Frame) Render(ctx *reactea.Ctx) string {
	width, height := ctx.Size()

	return f.style.Width(width).Height(height).Render(f.component.Render(f.inner(ctx)))
}

func (f *Frame) inner(ctx *reactea.Ctx) *reactea.Ctx {
	width, height := ctx.Size()

	return ctx.Inset(
		f.style.GetMarginLeft()+f.style.GetBorderLeftSize()+f.style.GetPaddingLeft(),
		f.style.GetMarginTop()+f.style.GetBorderTopSize()+f.style.GetPaddingTop(),
		width-f.style.GetHorizontalFrameSize(),
		height-f.style.GetVerticalFrameSize(),
	)
}
