package layout

import (
	"reflect"

	tea "charm.land/bubbletea/v2"
	"github.com/Hayao0819/reactea/v2"
)

// Memoized reuses a child's render for a stable key, size and focus state.
type Memoized struct {
	child reactea.Component
	key   func() any

	cached        string
	lastKey       any
	width, height int
	valid         bool
}

// Memo caches the child's render while key, box and focus remain stable. Init
// and Update continue to run while the cached picture is reused.
//
// key is what the child draws from, reduced to something comparable: a
// generation counter, a timestamp or the length of a slice. An uncomparable key
// triggers a fresh render.
//
// A focused child renders every frame so it can report its cursor.
func Memo(child reactea.Component, key func() any) *Memoized {
	return &Memoized{child: child, key: key}
}

// Invalidate drops the cache after an external visual change.
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

	// Cache unfocused output. Focused output is tied to the current focus and may
	// report a cursor.
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
