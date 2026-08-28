package reactea

import tea "charm.land/bubbletea/v2"

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
