package reactea

import (
	"context"
	"sync"
)

// Scope owns the cleanups of everything mounted under it. It replaces a Destroy
// method on Component: a parent that mounts and unmounts children gives each a
// Child scope, and everything else falls back to the app's root scope, so a
// forgotten call cannot strand a cleanup.
type Scope struct {
	// A cleanup is easy to register from a tea.Cmd's goroutine, so the bookkeeping
	// is guarded even though the event loop is the usual caller.
	mu       sync.Mutex
	cleanups []func()
	children []*Scope
	closed   bool

	once   sync.Once
	ctx    context.Context
	cancel context.CancelFunc
}

// Context is cancelled when this scope closes, so anything started under it
// stops when the component goes away — no cancel func to hold and no cleanup to
// remember.
func (s *Scope) Context() context.Context {
	s.once.Do(func() {
		s.ctx, s.cancel = context.WithCancel(context.Background())

		s.OnDestroy(s.cancel)
	})

	return s.ctx
}

// NewScope opens a scope with no parent.
func NewScope() *Scope { return &Scope{} }

// Child opens a nested scope. Closing the parent closes it too; closing the
// child early unmounts one thing without touching the rest.
func (s *Scope) Child() *Scope {
	child := NewScope()

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		child.Close()

		return child
	}

	// Drop closed ones so a long-lived parent does not collect a scope per route
	// change.
	live := s.children[:0]

	for _, existing := range s.children {
		if !existing.closed {
			live = append(live, existing)
		}
	}

	s.children = append(live, child)

	return child
}

// OnDestroy runs the cleanup at once if the scope has already closed.
func (s *Scope) OnDestroy(cleanup func()) {
	if cleanup == nil {
		return
	}

	s.mu.Lock()

	if s.closed {
		s.mu.Unlock()
		cleanup()

		return
	}

	s.cleanups = append(s.cleanups, cleanup)
	s.mu.Unlock()
}

// Close runs the cleanups newest first. Closing twice is a no-op.
func (s *Scope) Close() {
	s.mu.Lock()

	if s.closed {
		s.mu.Unlock()

		return
	}

	s.closed = true
	children, cleanups := s.children, s.cleanups
	s.children, s.cleanups = nil, nil

	s.mu.Unlock()

	for i := len(children) - 1; i >= 0; i-- {
		children[i].Close()
	}

	for i := len(cleanups) - 1; i >= 0; i-- {
		run(cleanups[i])
	}
}

// Closed reports whether Close has run.
func (s *Scope) Closed() bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.closed
}

// run isolates a cleanup: one that panics must not strand the ones registered
// before it.
func run(cleanup func()) {
	defer func() { _ = recover() }()

	cleanup()
}
