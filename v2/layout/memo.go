package layout

import (
	"reflect"

	tea "charm.land/bubbletea/v2"
	"github.com/Hayao0819/reactea/v2"
)

// Memoized reuses a child's last render while nothing it draws from has changed.
type Memoized struct {
	child reactea.Component
	key   func() any

	cached        string
	lastKey       any
	width, height int
	valid         bool
}

// Memo caches the child's render and reuses it while key returns the same value
// and the box and the focus stay as they were. Init and Update always run: only
// drawing is skipped, so the child's state is never stale — its picture is
// merely reused.
//
// key is what the child draws from, reduced to something comparable: a
// generation counter, a timestamp, the length of a slice. A key the cache cannot
// compare is a key it cannot trust, so the child is drawn again.
//
// A focused child is never cached. Only a focused component may set the cursor,
// and the cursor is set while drawing, so skipping its Render would take the
// cursor away with it.
func Memo(child reactea.Component, key func() any) *Memoized {
	return &Memoized{child: child, key: key}
}

// Invalidate drops the cache, for a change the key cannot see.
func (m *Memoized) Invalidate() { m.valid = false }

func (m *Memoized) Init(ctx *reactea.Ctx) tea.Cmd { return m.child.Init(ctx) }

// The focus methods pass straight through, so a memoised box still takes part in
// the Tab order.
func (m *Memoized) FocusNext() bool { return reactea.FocusOf(m.child).FocusNext() }

func (m *Memoized) FocusPrev() bool { return reactea.FocusOf(m.child).FocusPrev() }

func (m *Memoized) FocusFirst() { reactea.FocusOf(m.child).FocusFirst() }

func (m *Memoized) FocusLast() { reactea.FocusOf(m.child).FocusLast() }

func (m *Memoized) HasFocusable() bool { return reactea.FocusOf(m.child).HasFocusable() }

func (m *Memoized) Update(ctx *reactea.Ctx, msg tea.Msg) tea.Cmd {
	return m.child.Update(ctx, msg)
}

func (m *Memoized) Render(ctx *reactea.Ctx) string {
	width, height := ctx.Size()
	key := m.key()

	if m.valid && !ctx.Focused() && sameKey(key, m.lastKey) && width == m.width && height == m.height {
		return m.cached
	}

	rendered := m.child.Render(ctx)

	// Only an unfocused render is worth keeping: a focused one carries the focused
	// styling, which would be wrong the moment the focus moves on.
	if ctx.Focused() {
		m.valid = false

		return rendered
	}

	m.cached, m.lastKey, m.width, m.height = rendered, key, width, height
	m.valid = true

	return rendered
}

func sameKey(a, b any) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}

	kind := reflect.TypeOf(a)

	return kind == reflect.TypeOf(b) && kind.Comparable() && a == b
}
