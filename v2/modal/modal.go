// Package modal stacks blocking overlays on a base component. A modal finishes
// with Return, which pops it and delivers its answer as an ordinary message, so
// nothing here blocks the event loop.
package modal

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/Hayao0819/reactea/v2"
	"github.com/Hayao0819/reactea/v2/render"
)

// Result carries a modal's answer back to whoever pushed it.
type Result[T any] struct {
	Value T
	Err   error
}

// Ok reports whether the modal succeeded.
func (r Result[T]) Ok() bool { return r.Err == nil }

type completeMsg struct {
	target *Stack
	id     uint64
	result tea.Cmd
}

type closer interface {
	CloseOverlay(tea.Cmd) tea.Cmd
}

// Push opens component on the nearest Stack above ctx.
func Push(ctx *reactea.Ctx, component reactea.Component) tea.Cmd {
	host, ok := ctx.Overlay()
	if !ok {
		return nil
	}

	return host.Push(component)
}

// PushAt opens component at placement on the nearest Stack above ctx.
func PushAt(ctx *reactea.Ctx, component reactea.Component, placement Placement) tea.Cmd {
	host, ok := ctx.Overlay()
	if !ok {
		return nil
	}

	return host.PushAt(component, placement)
}

// Dismiss closes the modal associated with ctx without an answer.
func Dismiss(ctx *reactea.Ctx) tea.Cmd { return close(ctx, nil) }

// Return closes the modal associated with ctx and delivers value.
func Return[T any](ctx *reactea.Ctx, value T) tea.Cmd {
	return close(ctx, func() tea.Msg { return Result[T]{Value: value} })
}

// Fail closes the modal associated with ctx and delivers err.
func Fail[T any](ctx *reactea.Ctx, err error) tea.Cmd {
	return close(ctx, func() tea.Msg { return Result[T]{Err: err} })
}

func close(ctx *reactea.Ctx, result tea.Cmd) tea.Cmd {
	host, ok := ctx.Overlay()
	if !ok {
		return nil
	}

	bound, ok := host.(closer)
	if !ok {
		return nil
	}

	return bound.CloseOverlay(result)
}

// Placement is where a modal sits inside the stack's box. It lives in the root
// package so that Ctx can name it without importing this one.
type Placement = reactea.Placement

// Center asks for the middle of the axis.
const Center = reactea.Center

// Centered places a modal of the given size in the middle of both axes.
func Centered(width, height int) Placement {
	return Placement{X: Center, Y: Center, Width: width, Height: height}
}

func rect(p Placement, width, height int) (x, y, w, h int) {
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
	id        uint64
	capture   reactea.InputCapture
}

// Stack renders base until something is pushed on top of it.
type Stack struct {
	base   reactea.Component
	modals []mounted
	nextID uint64
}

// New builds a Stack over base.
func New(base reactea.Component) *Stack {
	return &Stack{base: base}
}

// Push opens modal on this stack during the next Update.
func (s *Stack) Push(modal reactea.Component) tea.Cmd {
	return s.PushAt(modal, Placement{})
}

// PushAt puts a modal somewhere other than over the whole box, which is how a
// confirmation sits in the middle with the page still visible around it.
func (s *Stack) PushAt(modal reactea.Component, placement Placement) tea.Cmd {
	return func() tea.Msg { return pushMsg{target: s, modal: modal, placement: placement} }
}

// A push names the stack it was asked of. Without that the outermost stack in
// the tree would take every one, since it sees the message first.
type pushMsg struct {
	target    *Stack
	modal     reactea.Component
	placement Placement
}

type host struct {
	stack *Stack
	id    uint64
}

func (h host) Push(component reactea.Component) tea.Cmd {
	return h.stack.Push(component)
}

func (h host) PushAt(component reactea.Component, placement Placement) tea.Cmd {
	return h.stack.PushAt(component, placement)
}

func (h host) CloseOverlay(result tea.Cmd) tea.Cmd {
	if h.id == 0 {
		return nil
	}

	return func() tea.Msg { return completeMsg{target: h.stack, id: h.id, result: result} }
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

// The focus methods reach whatever is taking the input: the modal on top, or the
// base when the stack is empty.
func (s *Stack) FocusNext() bool { return reactea.FocusOf(s.focused()).FocusNext() }

func (s *Stack) FocusPrev() bool { return reactea.FocusOf(s.focused()).FocusPrev() }

func (s *Stack) FocusFirst() { reactea.FocusOf(s.focused()).FocusFirst() }

func (s *Stack) FocusLast() { reactea.FocusOf(s.focused()).FocusLast() }

func (s *Stack) HasFocusable() bool { return reactea.FocusOf(s.focused()).HasFocusable() }

func (s *Stack) focused() reactea.Component {
	if top := s.Top(); top != nil {
		return top
	}

	return s.base
}

func (s *Stack) Init(ctx *reactea.Ctx) tea.Cmd {
	ctx.OnDestroy(s.clear)

	return s.base.Init(s.baseCtx(ctx))
}

func (s *Stack) clear() {
	for i := len(s.modals) - 1; i >= 0; i-- {
		s.modals[i].capture.ReleaseInput()
		s.modals[i].scope.Close()
	}

	s.modals = nil
}

func (s *Stack) Update(ctx *reactea.Ctx, msg tea.Msg) tea.Cmd {
	ctx = s.baseCtx(ctx)

	switch msg := msg.(type) {
	case pushMsg:
		if msg.target == s {
			scope := ctx.Scope().Child()
			s.nextID++
			s.modals = append(s.modals, mounted{
				component: msg.modal,
				scope:     scope,
				placement: msg.placement,
				id:        s.nextID,
			})

			top := s.top()
			top.capture.CaptureInput(ctx.WithScope(scope))

			return msg.modal.Init(s.modalCtx(ctx, top))
		}

	case completeMsg:
		if top := s.top(); msg.target == s && top != nil && top.id == msg.id {
			top.capture.ReleaseInput()
			top.scope.Close()
			s.modals = s.modals[:len(s.modals)-1]

			return msg.result
		}
	}

	// A modal blocks input, not data: the base keeps receiving its own ticks and
	// async results while a modal is up, or its work would stall unfinishable.
	if top := s.top(); top != nil && reactea.IsInput(msg) {
		modalCtx := s.modalCtx(ctx, top)

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

	for i := range s.modals {
		mounted := &s.modals[i]
		cmds = append(cmds, mounted.component.Update(s.modalCtx(ctx, mounted), msg))
	}

	return tea.Batch(cmds...)
}

func (s *Stack) Render(ctx *reactea.Ctx) string {
	// The base does not hold the focus while a modal is up — that is already true
	// of its input, and it is what keeps its cursor from showing through.
	ctx = s.baseCtx(ctx)

	base := s.base.Render(ctx.WithFocus(len(s.modals) == 0))

	if len(s.modals) == 0 {
		return base
	}

	width, height := ctx.Size()

	layers := make([]*lipgloss.Layer, 0, len(s.modals))

	for i := range s.modals {
		modal := &s.modals[i]
		x, y, _, _ := rect(modal.placement, width, height)
		box := s.modalCtx(ctx, modal)
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
	x, y, w, h := rect(modal.placement, width, height)

	return ctx.Inset(x, y, w, h).WithScope(modal.scope)
}

func (s *Stack) baseCtx(ctx *reactea.Ctx) *reactea.Ctx {
	return ctx.WithOverlay(host{stack: s})
}

func (s *Stack) modalCtx(ctx *reactea.Ctx, modal *mounted) *reactea.Ctx {
	return s.box(ctx, *modal).WithOverlay(host{stack: s, id: modal.id})
}
