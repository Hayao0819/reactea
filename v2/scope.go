package reactea

// Scope owns the cleanups of everything mounted under it. Closing a scope runs
// them, newest first, and closes the scopes nested inside it.
//
// This is what replaced a Destroy method on Component. A component that holds no
// resource writes nothing; one that does registers a cleanup with
// Ctx.OnDestroy and stops caring who tears it down. A parent that mounts and
// unmounts children — the router, the modal stack — gives each child a Child
// scope and closes it when the child goes away. Everything else is covered by
// the app's root scope, which closes when the program ends, so nothing can be
// stranded by a parent that forgot to forward a call.
type Scope struct {
	cleanups []func()
	children []*Scope
	closed   bool
}

func NewScope() *Scope { return &Scope{} }

// Child opens a nested scope. Closing the parent closes it too, so a child
// scope is never stranded, and closing the child early is the normal way to
// unmount one thing without touching the rest.
func (s *Scope) Child() *Scope {
	child := NewScope()

	if s.closed {
		child.Close()

		return child
	}

	// Drop the ones that already closed so a long-lived parent does not collect
	// a scope per route change.
	live := s.children[:0]

	for _, existing := range s.children {
		if !existing.closed {
			live = append(live, existing)
		}
	}

	s.children = append(live, child)

	return child
}

// OnDestroy registers a cleanup. Registering on a scope that is already closed
// runs the cleanup at once, so a late arrival cannot be stranded.
func (s *Scope) OnDestroy(cleanup func()) {
	if cleanup == nil {
		return
	}

	if s.closed {
		cleanup()

		return
	}

	s.cleanups = append(s.cleanups, cleanup)
}

// Close closes the nested scopes and runs the registered cleanups, newest
// first. Closing twice is a no-op.
func (s *Scope) Close() {
	if s.closed {
		return
	}

	s.closed = true

	for i := len(s.children) - 1; i >= 0; i-- {
		s.children[i].Close()
	}

	s.children = nil

	for i := len(s.cleanups) - 1; i >= 0; i-- {
		s.cleanups[i]()
	}

	s.cleanups = nil
}

// Closed reports whether Close has run.
func (s *Scope) Closed() bool { return s.closed }
