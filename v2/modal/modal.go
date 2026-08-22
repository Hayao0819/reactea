// Package modal stacks blocking overlays on top of a base component.
//
// A modal is an ordinary Component. It is pushed onto a Stack, it receives every
// message while it is on top, and it finishes by returning a command built with
// Return — which pops it and delivers its answer to the rest of the tree as an
// ordinary message. Nothing blocks: no extra goroutine, no channel handshake,
// no chance of parking the event loop.
package modal

import (
	tea "charm.land/bubbletea/v2"
	"github.com/Hayao0819/reactea/v2"
)

// Result carries a modal's answer back to whoever pushed it.
type Result[T any] struct {
	Value T
	Err   error
}

// Ok reports a value.
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

// Stack renders base until something is pushed on top of it.
type Stack struct {
	base   reactea.Component
	modals []reactea.Component
}

func New(base reactea.Component) *Stack {
	return &Stack{base: base}
}

// Push puts a modal on top. It is initialised on the next Update, so Push is
// safe to call from anywhere.
func (s *Stack) Push(modal reactea.Component) tea.Cmd {
	return func() tea.Msg { return pushMsg{modal: modal} }
}

type pushMsg struct{ modal reactea.Component }

// Top is the modal currently on top, or nil.
func (s *Stack) Top() reactea.Component {
	if len(s.modals) == 0 {
		return nil
	}

	return s.modals[len(s.modals)-1]
}

func (s *Stack) Init(ctx *reactea.Ctx) tea.Cmd {
	return s.base.Init(ctx)
}

func (s *Stack) Destroy() {
	for _, modal := range s.modals {
		modal.Destroy()
	}

	s.modals = nil

	s.base.Destroy()
}

func (s *Stack) Update(ctx *reactea.Ctx, msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case pushMsg:
		s.modals = append(s.modals, msg.modal)

		return msg.modal.Init(ctx)

	case dismissMsg:
		if top := s.Top(); top != nil {
			top.Destroy()
			s.modals = s.modals[:len(s.modals)-1]
		}

		return nil
	}

	// A modal is a blocking overlay: while one is up it takes the input, and the
	// base sees nothing.
	if top := s.Top(); top != nil {
		return top.Update(ctx, msg)
	}

	return s.base.Update(ctx, msg)
}

func (s *Stack) Render(ctx *reactea.Ctx) string {
	if top := s.Top(); top != nil {
		return top.Render(ctx)
	}

	return s.base.Render(ctx)
}
