package layout

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/Hayao0819/reactea/v2"
	"github.com/Hayao0819/reactea/v2/internal/render"
)

// Frame draws a lipgloss style around a component. Lipgloss counts Width and
// Height as the outer size, so the child is rendered at the box minus border,
// padding and margin, with its cursor shifted to match.
type Frame struct {
	style     lipgloss.Style
	focused   *lipgloss.Style
	component reactea.Component
}

// SetStyle changes the frame drawn around the component.
func (f *Frame) SetStyle(style lipgloss.Style) { f.style = style }

// WhenFocused draws the frame in another style while the component inside holds
// the focus, which is how a multi-pane UI shows where the keys are going.
func (f *Frame) WhenFocused(style lipgloss.Style) *Frame {
	f.focused = &style

	return f
}

// Framed wraps component in style.
func Framed(style lipgloss.Style, component reactea.Component) *Frame {
	return &Frame{style: style, component: component}
}

func (f *Frame) Init(ctx *reactea.Ctx) tea.Cmd { return f.component.Init(f.inner(ctx)) }

func (f *Frame) Update(ctx *reactea.Ctx, msg tea.Msg) tea.Cmd {
	inner := f.inner(ctx)

	if reactea.IsMouse(msg) {
		outerX, outerY := ctx.Origin()
		innerX, innerY := inner.Origin()

		msg = reactea.TranslateMouse(msg, innerX-outerX, innerY-outerY)

		// A click on the border is not a click on the child. Box hit-tests before
		// it routes, so dropping here keeps the two containers consistent.
		if _, _, inside := reactea.Mouse(inner, msg); !inside {
			return nil
		}
	}

	return f.component.Update(inner, msg)
}

// FocusNext and the rest pass straight through, so a framed box still takes part
// in the Tab order.
func (f *Frame) FocusNext() bool { return f.focuser().FocusNext() }

func (f *Frame) FocusPrev() bool { return f.focuser().FocusPrev() }

func (f *Frame) FocusFirst() { f.focuser().FocusFirst() }

func (f *Frame) FocusLast() { f.focuser().FocusLast() }

func (f *Frame) HasFocusable() bool { return f.focuser().HasFocusable() }

func (f *Frame) focuser() Focuser {
	if child, ok := f.component.(Focuser); ok {
		return child
	}

	return noFocus{}
}

type noFocus struct{}

func (noFocus) FocusNext() bool    { return false }
func (noFocus) FocusPrev() bool    { return false }
func (noFocus) FocusFirst()        {}
func (noFocus) FocusLast()         {}
func (noFocus) HasFocusable() bool { return false }

func (f *Frame) Render(ctx *reactea.Ctx) string {
	width, height := ctx.Size()

	inner := f.inner(ctx)
	innerWidth, innerHeight := inner.Size()

	// Trim the child to the inner box first. Trimming the framed result instead
	// would cut the border off whichever side overran.
	content := render.Fit(f.component.Render(inner), innerWidth, innerHeight)

	return f.current(ctx).
		Width(width).Height(height).
		MaxWidth(width).MaxHeight(height).
		Render(content)
}

func (f *Frame) current(ctx *reactea.Ctx) lipgloss.Style {
	if f.focused != nil && ctx.Focused() {
		return *f.focused
	}

	return f.style
}

func (f *Frame) inner(ctx *reactea.Ctx) *reactea.Ctx {
	width, height := ctx.Size()
	style := f.current(ctx)

	return ctx.Inset(
		style.GetMarginLeft()+style.GetBorderLeftSize()+style.GetPaddingLeft(),
		style.GetMarginTop()+style.GetBorderTopSize()+style.GetPaddingTop(),
		width-style.GetHorizontalFrameSize(),
		height-style.GetVerticalFrameSize(),
	)
}
