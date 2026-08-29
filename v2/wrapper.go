package reactea

import tea "charm.land/bubbletea/v2"

// Focuser is a component that can move the focus among what it holds. layout.Box
// implements it, and every container that wraps one passes it through, so Tab
// reaches a focusable however many wrappers sit above it.
type Focuser interface {
	FocusNext() bool
	FocusPrev() bool
	FocusFirst()
	FocusLast()

	// HasFocusable reports whether there is anything inside to focus. A container
	// that holds none must not be handed the focus, or keys would vanish into it.
	HasFocusable() bool
}

// FocusOf is component's Focuser, or one that holds nothing. A container calls it
// to forward the focus methods to whatever it wraps.
func FocusOf(component Component) Focuser {
	if focuser, ok := component.(Focuser); ok {
		return focuser
	}

	return noFocus{}
}

type noFocus struct{}

func (noFocus) FocusNext() bool    { return false }
func (noFocus) FocusPrev() bool    { return false }
func (noFocus) FocusFirst()        {}
func (noFocus) FocusLast()         {}
func (noFocus) HasFocusable() bool { return false }

// Wrapper forwards the whole lifecycle to a single child. Embed it and write
// only the methods that differ.
type Wrapper struct {
	Child Component
}

// Wrap builds a Wrapper around child.
func Wrap(child Component) Wrapper { return Wrapper{Child: child} }

func (w Wrapper) Init(ctx *Ctx) tea.Cmd { return w.Child.Init(ctx) }

func (w Wrapper) Update(ctx *Ctx, msg tea.Msg) tea.Cmd { return w.Child.Update(ctx, msg) }

func (w Wrapper) Render(ctx *Ctx) string { return w.Child.Render(ctx) }

func (w Wrapper) FocusNext() bool { return FocusOf(w.Child).FocusNext() }

func (w Wrapper) FocusPrev() bool { return FocusOf(w.Child).FocusPrev() }

func (w Wrapper) FocusFirst() { FocusOf(w.Child).FocusFirst() }

func (w Wrapper) FocusLast() { FocusOf(w.Child).FocusLast() }

func (w Wrapper) HasFocusable() bool { return FocusOf(w.Child).HasFocusable() }

// Key reports whether msg is a press of one of keys, spelled the way
// tea.KeyPressMsg.String does.
func Key(msg tea.Msg, keys ...string) bool {
	press, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return false
	}

	pressed := press.String()

	for _, key := range keys {
		if key == pressed {
			return true
		}
	}

	return false
}

// Mouse reports whether msg is a mouse event inside this component's box.
// Containers translate as they route, so the coordinates are already box-local.
func Mouse(ctx *Ctx, msg tea.Msg) (x, y int, ok bool) {
	x, y, ok = MouseAt(msg)
	if !ok {
		return 0, 0, false
	}

	width, height := ctx.Size()

	return x, y, x >= 0 && y >= 0 && x < width && y < height
}
