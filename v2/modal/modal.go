// Package modal stacks blocking overlays on a base component. A modal finishes
// with Return, which pops it and delivers its answer as an ordinary message, so
// nothing here blocks the event loop.
package modal

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/Hayao0819/reactea/v2"
	"github.com/Hayao0819/reactea/v2/internal/render"
)

// Result carries a modal's answer back to whoever pushed it.
type Result[T any] struct {
	Value T
	Err   error
}

// Ok reports whether the modal succeeded.
func (r Result[T]) Ok() bool { return r.Err == nil }

type dismissMsg struct{}

// Dismiss closes the modal on top without an answer.
func Dismiss() tea.Msg { return dismissMsg{} }

// Return closes the modal on top and delivers value.
func Return[T any](value T) tea.Cmd {
	return tea.Batch(
		Dismiss,
		func() tea.Msg { return Result[T]{Value: value} },
	)
}

// Fail closes the modal on top and delivers err.
func Fail[T any](err error) tea.Cmd {
	return tea.Batch(
		Dismiss,
		func() tea.Msg { return Result[T]{Err: err} },
	)
}

// Placement is where a modal sits inside the stack's box. A zero Width or Height
// spans that axis; an X or Y of Center puts it in the middle.
type Placement struct {
	X, Y          int
	Width, Height int
}

// Center asks for the middle of the axis.
const Center = -1

// FullScreen covers the whole box, which is what Push uses.
var FullScreen = Placement{}

func (p Placement) rect(width, height int) (x, y, w, h int) {
	w, h = p.Width, p.Height
	if w <= 0 || w > width {
		w = width
	}

	if h <= 0 || h > height {
		h = height
	}

	x, y = p.X, p.Y
	if x == Center {
		x = (width - w) / 2
	}

	if y == Center {
		y = (height - h) / 2
	}

	return max(0, x), max(0, y), w, h
}

type mounted struct {
	component reactea.Component
	scope     *reactea.Scope
	placement Placement
}

// Stack renders base until something is pushed on top of it.
type Stack struct {
	base   reactea.Component
	modals []mounted
}

// New builds a Stack over base.
func New(base reactea.Component) *Stack {
	return &Stack{base: base}
}

// The modal is initialised on the next Update, so Push is safe from anywhere.
func (s *Stack) Push(modal reactea.Component) tea.Cmd {
	return s.PushAt(modal, FullScreen)
}

// PushAt puts a modal somewhere other than over the whole box, which is how a
// confirmation sits in the middle with the page still visible around it.
func (s *Stack) PushAt(modal reactea.Component, placement Placement) tea.Cmd {
	return func() tea.Msg { return pushMsg{modal: modal, placement: placement} }
}

type pushMsg struct {
	modal     reactea.Component
	placement Placement
}

// Top is the modal currently on top, or nil.
func (s *Stack) Top() reactea.Component {
	if len(s.modals) == 0 {
		return nil
	}

	return s.modals[len(s.modals)-1].component
}

func (s *Stack) top() *mounted {
	if len(s.modals) == 0 {
		return nil
	}

	return &s.modals[len(s.modals)-1]
}

func (s *Stack) Init(ctx *reactea.Ctx) tea.Cmd {
	return s.base.Init(ctx)
}

func (s *Stack) Update(ctx *reactea.Ctx, msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case pushMsg:
		// Its own scope, so dismissing one runs exactly its cleanups.
		scope := ctx.Scope().Child()
		s.modals = append(s.modals, mounted{component: msg.modal, scope: scope, placement: msg.placement})

		// A modal takes the keys, so the root's global keys stand down while it
		// is up without the app having to remember.
		return tea.Batch(msg.modal.Init(ctx.WithScope(scope)), reactea.CaptureInput)

	case dismissMsg:
		if top := s.top(); top != nil {
			top.scope.Close()
			s.modals = s.modals[:len(s.modals)-1]

			return reactea.ReleaseInput
		}

		return nil
	}

	// A modal blocks input, not data: the base keeps receiving its own ticks and
	// async results while a modal is up, or its work would stall unfinishable.
	if top := s.top(); top != nil && reactea.IsInput(msg) {
		modalCtx := s.box(ctx, *top)

		if reactea.IsMouse(msg) {
			outerX, outerY := ctx.Origin()
			innerX, innerY := modalCtx.Origin()

			msg = reactea.TranslateMouse(msg, innerX-outerX, innerY-outerY)

			// A click beside a placed modal belongs to nobody: the base is covered
			// as far as input goes.
			if _, _, inside := reactea.Mouse(modalCtx, msg); !inside {
				return nil
			}
		}

		return top.component.Update(modalCtx, msg)
	}

	cmds := make([]tea.Cmd, 0, len(s.modals)+1)

	cmds = append(cmds, s.base.Update(ctx, msg))

	for _, mounted := range s.modals {
		cmds = append(cmds, mounted.component.Update(ctx.WithScope(mounted.scope), msg))
	}

	return tea.Batch(cmds...)
}

func (s *Stack) Render(ctx *reactea.Ctx) string {
	// The base does not hold the focus while a modal is up — that is already true
	// of its input, and it is what keeps its cursor from showing through.
	base := s.base.Render(ctx.WithFocus(len(s.modals) == 0))

	if len(s.modals) == 0 {
		return base
	}

	width, height := ctx.Size()

	layers := make([]*lipgloss.Layer, 0, len(s.modals))

	for i, modal := range s.modals {
		x, y, _, _ := modal.placement.rect(width, height)
		box := s.box(ctx, modal)
		boxWidth, boxHeight := box.Size()

		// Fitting is what makes a layer opaque: padded blanks cover the base, and
		// an overrunning modal cannot stretch the stack past its own box.
		content := render.Fit(modal.component.Render(box), boxWidth, boxHeight)

		layers = append(layers, lipgloss.NewLayer(content).X(x).Y(y).Z(i+1))
	}

	composed := lipgloss.NewCompositor(append([]*lipgloss.Layer{lipgloss.NewLayer(base)}, layers...)...).Render()

	// The compositor drops trailing blanks, so fit again to hold the same contract
	// Box and Frame do: what comes out is the box that went in.
	return render.Fit(composed, width, height)
}

// box is the Ctx a modal draws into: its placement within the stack's box, bound
// to its own scope.
func (s *Stack) box(ctx *reactea.Ctx, modal mounted) *reactea.Ctx {
	width, height := ctx.Size()
	x, y, w, h := modal.placement.rect(width, height)

	return ctx.Inset(x, y, w, h).WithScope(modal.scope)
}
