package layout

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/Hayao0819/reactea/v2"
)

// Frame draws a lipgloss style around a component. Lipgloss counts Width and
// Height as the outer size, so the child has to be rendered at the box minus the
// border, padding and margin — that subtraction, and the matching cursor shift,
// is what this saves every parent from writing.
type Frame struct {
	style     lipgloss.Style
	component reactea.Component

	offset offset
}

// Framed wraps component in style.
func Framed(style lipgloss.Style, component reactea.Component) *Frame {
	return &Frame{style: style, component: component}
}

func (f *Frame) Init() tea.Cmd { return f.component.Init() }

func (f *Frame) Destroy() { f.component.Destroy() }

func (f *Frame) Update(msg tea.Msg) tea.Cmd { return f.component.Update(msg) }

func (f *Frame) Render(width, height int) string {
	inner := f.component.Render(
		max(0, width-f.style.GetHorizontalFrameSize()),
		max(0, height-f.style.GetVerticalFrameSize()),
	)

	f.offset = offset{
		x: f.style.GetMarginLeft() + f.style.GetBorderLeftSize() + f.style.GetPaddingLeft(),
		y: f.style.GetMarginTop() + f.style.GetBorderTopSize() + f.style.GetPaddingTop(),
	}

	return f.style.Width(width).Height(height).Render(inner)
}

func (f *Frame) DecorateView(view *tea.View) {
	child := tea.NewView("")

	reactea.DecorateView(f.component, &child)
	reactea.TranslateCursor(&child, f.offset.x, f.offset.y)

	merge(view, child)
}
