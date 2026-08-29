// Package layout splits a box among child components along one axis.
package layout

import (
	"reflect"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/Hayao0819/reactea/v2"
	"github.com/Hayao0819/reactea/v2/internal/render"
)

// Direction is the axis a Box lays its items out along.
type Direction int

const (
	Vertical Direction = iota
	Horizontal
)

// Item is one child and how much of the main axis it wants. A Size of 0 means
// the item is flexible and takes a share of what is left, weighted by Grow.
type Item struct {
	Component reactea.Component

	Size int
	Grow int
	Min  int
	Max  int

	focusable bool
}

// Focusable marks the item as something Tab can land on.
func (i Item) Focusable() Item {
	i.focusable = true

	return i
}

// IsFocusable reports what Focusable set, for code that rebuilds an item list.
func (i Item) IsFocusable() bool { return i.focusable }

// sameComponent compares without the == that would panic on a component whose
// dynamic type is uncomparable. Losing the focus beats losing the program.
func sameComponent(a, b reactea.Component) bool {
	if a == nil || b == nil {
		return false
	}

	kind := reflect.TypeOf(a)

	return kind == reflect.TypeOf(b) && kind.Comparable() && a == b
}

// Fixed gives the child exactly size cells on the main axis.
func Fixed(size int, component reactea.Component) Item {
	return Item{Component: component, Size: size}
}

// Grow gives the child a share of what is left, proportional to weight.
func Grow(weight int, component reactea.Component) Item {
	return Item{Component: component, Grow: weight}
}

// Bounded is Grow with a floor and a ceiling. A zero maximum means unbounded.
func Bounded(weight, minimum, maximum int, component reactea.Component) Item {
	return Item{Component: component, Grow: weight, Min: minimum, Max: maximum}
}

// Box lays its items out along one axis and gives each the full cross axis.
// Focuser is a container that can move the focus among its children. Box
// implements it, so nested boxes hand Tab down before advancing themselves.
type Focuser = reactea.Focuser

type Box struct {
	direction Direction
	items     []Item

	focused int
	starved []int
}

// New builds a Box laying out along direction.
func New(direction Direction, items ...Item) *Box {
	box := &Box{direction: direction, items: items, focused: -1}

	box.FocusFirst()

	return box
}

// Items is what the box currently lays out.
func (b *Box) Items() []Item { return b.items }

// SetItems replaces the children, which is how a pane is hidden, maximised or
// reordered without rebuilding the tree and losing everyone's state. The focus
// stays on the same component when it is still there.
func (b *Box) SetItems(items ...Item) {
	var focused reactea.Component

	if b.focused >= 0 && b.focused < len(b.items) {
		focused = b.items[b.focused].Component
	}

	b.items = items
	b.focused = -1

	for i := range items {
		if sameComponent(items[i].Component, focused) && b.takesFocus(i) {
			b.focused = i

			return
		}
	}

	b.FocusFirst()
}

// Starved reports the items the last split had no room for: those handed nothing
// at all, and those left below the Size or Min they asked for. A caller that
// would rather hide a pane than draw it crushed reads this and decides — the box
// itself never drops one. The slice is a copy; the box refills its own on every
// phase, including a mouse move.
func (b *Box) Starved() []int {
	if len(b.starved) == 0 {
		return nil
	}

	return append([]int(nil), b.starved...)
}

// Focused is the index of the item holding the focus, or -1.
func (b *Box) Focused() int { return b.focused }

// Focus moves the focus to item index, if it can take it.
func (b *Box) Focus(index int) bool {
	if index < 0 || index >= len(b.items) || !b.takesFocus(index) {
		return false
	}

	b.focused = index

	if child, ok := b.items[index].Component.(Focuser); ok {
		child.FocusFirst()
	}

	return true
}

// FocusNext moves to the next focusable leaf, descending into a nested box
// first. It reports false when there is nothing further, which is the caller's
// cue to wrap with FocusFirst.
func (b *Box) FocusNext() bool { return b.step(1) }

// FocusPrev is FocusNext backwards; wrap it with FocusLast.
func (b *Box) FocusPrev() bool { return b.step(-1) }

func (b *Box) step(by int) bool {
	if b.focused >= 0 && b.focused < len(b.items) {
		if child, ok := b.items[b.focused].Component.(Focuser); ok {
			if (by > 0 && child.FocusNext()) || (by < 0 && child.FocusPrev()) {
				return true
			}
		}
	}

	for i := b.focused + by; i >= 0 && i < len(b.items); i += by {
		if !b.takesFocus(i) {
			continue
		}

		b.focused = i

		if child, ok := b.items[i].Component.(Focuser); ok {
			if by > 0 {
				child.FocusFirst()
			} else {
				child.FocusLast()
			}
		}

		return true
	}

	return false
}

// FocusFirst puts the focus on the first item that can take it.
func (b *Box) FocusFirst() {
	b.focused = -1
	b.step(1)
}

// FocusLast puts the focus on the last item that can take it.
func (b *Box) FocusLast() {
	b.focused = len(b.items)
	b.step(-1)
}

// takesFocus reports whether item i can hold the focus itself or contains
// something that can.
func (b *Box) takesFocus(i int) bool {
	if b.items[i].focusable {
		return true
	}

	nested, ok := b.items[i].Component.(Focuser)

	return ok && nested.HasFocusable()
}

// HasFocusable implements Focuser.
func (b *Box) HasFocusable() bool {
	for i := range b.items {
		if b.takesFocus(i) {
			return true
		}
	}

	return false
}

// Column stacks items top to bottom.
func Column(items ...Item) *Box { return New(Vertical, items...) }

// Row places items left to right.
func Row(items ...Item) *Box { return New(Horizontal, items...) }

func (b *Box) Init(ctx *reactea.Ctx) tea.Cmd {
	cmds := make([]tea.Cmd, 0, len(b.items))

	b.each(ctx, func(_ int, item Item, childCtx *reactea.Ctx) {
		cmds = append(cmds, item.Component.Init(childCtx))
	})

	return tea.Batch(cmds...)
}

// Update routes by addressee: the keyboard reaches whatever holds the focus,
// the mouse reaches whatever sits under the pointer, and everything else
// reaches every item.
func (b *Box) Update(ctx *reactea.Ctx, msg tea.Msg) tea.Cmd {
	if reactea.IsMouse(msg) {
		return b.routeMouse(ctx, msg)
	}

	if reactea.IsKeyboard(msg) {
		return b.routeKeyboard(ctx, msg)
	}

	cmds := make([]tea.Cmd, 0, len(b.items))

	b.each(ctx, func(_ int, item Item, childCtx *reactea.Ctx) {
		cmds = append(cmds, item.Component.Update(childCtx, msg))
	})

	return tea.Batch(cmds...)
}

func (b *Box) routeKeyboard(ctx *reactea.Ctx, msg tea.Msg) tea.Cmd {
	if !ctx.Focused() || b.focused < 0 {
		return nil
	}

	var cmd tea.Cmd

	b.each(ctx, func(i int, item Item, childCtx *reactea.Ctx) {
		if i == b.focused {
			cmd = item.Component.Update(childCtx, msg)
		}
	})

	return cmd
}

func (b *Box) routeMouse(ctx *reactea.Ctx, msg tea.Msg) tea.Cmd {
	x, y, ok := reactea.MouseAt(msg)
	if !ok {
		return nil
	}

	var (
		hit    = -1
		hitCtx *reactea.Ctx
	)

	b.each(ctx, func(i int, _ Item, childCtx *reactea.Ctx) {
		if hit >= 0 {
			return
		}

		width, height := childCtx.Size()
		if local := b.local(ctx, childCtx, x, y); local.x >= 0 && local.y >= 0 && local.x < width && local.y < height {
			hit, hitCtx = i, childCtx
		}
	})

	if hit < 0 {
		return nil
	}

	// A press moves the focus to whatever was pressed, the way every pointer UI
	// behaves. A wheel or a motion leaves it alone.
	if _, press := msg.(tea.MouseClickMsg); press && b.takesFocus(hit) {
		b.Focus(hit)
		hitCtx = hitCtx.WithFocus(ctx.Focused())
	}

	offset := b.local(ctx, hitCtx, 0, 0)

	return b.items[hit].Component.Update(hitCtx, reactea.TranslateMouse(msg, -offset.x, -offset.y))
}

type point struct{ x, y int }

// local turns a coordinate in this box's space into the child's.
func (b *Box) local(ctx, child *reactea.Ctx, x, y int) point {
	parentX, parentY := ctx.Origin()
	childX, childY := child.Origin()

	return point{x: x - (childX - parentX), y: y - (childY - parentY)}
}

func (b *Box) Render(ctx *reactea.Ctx) string {
	rendered := make([]string, 0, len(b.items))

	b.each(ctx, func(_ int, item Item, childCtx *reactea.Ctx) {
		width, height := childCtx.Size()

		// An item with no cells on the main axis is left out entirely; joining its
		// empty string would still cost a row or a column.
		if (b.direction == Vertical && height <= 0) || (b.direction == Horizontal && width <= 0) {
			return
		}

		rendered = append(rendered, render.Fit(item.Component.Render(childCtx), width, height))
	})

	if b.direction == Vertical {
		return lipgloss.JoinVertical(lipgloss.Left, rendered...)
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, rendered...)
}

// The split is recomputed per phase rather than cached, so Update and Render can
// be called in any order. Every child's box, and the record of who went short,
// is settled before the first one is visited, so an item that reads Starved
// during its own Render sees the whole answer.
func (b *Box) each(ctx *reactea.Ctx, visit func(int, Item, *reactea.Ctx)) {
	width, height := ctx.Size()

	main, cross := width, height
	if b.direction == Vertical {
		main, cross = height, width
	}

	sizes := distribute(main, b.items)
	boxes := make([]*reactea.Ctx, len(b.items))

	b.starved = b.starved[:0]

	position := 0

	for i, item := range b.items {
		focused := ctx.Focused() && i == b.focused

		var got int

		if b.direction == Vertical {
			boxes[i] = ctx.Inset(0, position, cross, sizes[i]).WithFocus(focused)
			_, got = boxes[i].Size()
		} else {
			boxes[i] = ctx.Inset(position, 0, sizes[i], cross).WithFocus(focused)
			got, _ = boxes[i].Size()
		}

		// Measure what the child actually got: distribute can hand out more than
		// the box holds, and Inset clamps the overflow away.
		if got == 0 || (item.Size > 0 && got < item.Size) || (item.Min > 0 && got < item.Min) {
			b.starved = append(b.starved, i)
		}

		position += sizes[i]
	}

	for i, item := range b.items {
		visit(i, item, boxes[i])
	}
}

func distribute(total int, items []Item) []int {
	sizes := make([]int, len(items))

	if total < 0 {
		total = 0
	}

	remaining := total
	weight := 0

	grow := make([]int, len(items))

	for i, item := range items {
		if item.Size > 0 {
			sizes[i] = min(item.Size, remaining)
			remaining -= sizes[i]

			continue
		}

		// A flexible item with no weight still deserves a share.
		grow[i] = max(item.Grow, 1)
		weight += grow[i]
	}

	if weight == 0 {
		return sizes
	}

	share(sizes, items, grow, remaining, weight)
	clampSizes(sizes, items, total)

	return sizes
}

func share(sizes []int, items []Item, grow []int, remaining, weight int) {
	// Largest-remainder apportionment, so the sizes always add up to remaining.
	leftover := remaining
	fractions := make([]int, len(items))

	for i, item := range items {
		if item.Size > 0 {
			continue
		}

		exact := remaining * grow[i]
		sizes[i] = exact / weight
		fractions[i] = exact % weight
		leftover -= sizes[i]
	}

	for range leftover {
		best, bestFraction := -1, -1

		for i, item := range items {
			if item.Size > 0 {
				continue
			}

			if fractions[i] > bestFraction {
				best, bestFraction = i, fractions[i]
			}
		}

		if best < 0 {
			break
		}

		sizes[best]++
		fractions[best] = -1
	}
}

func clampSizes(sizes []int, items []Item, total int) {
	free := make([]int, 0, len(items))
	used := 0

	for i, item := range items {
		switch {
		case item.Size > 0:
		case item.Min > 0 && sizes[i] < item.Min:
			sizes[i] = item.Min
		case item.Max > 0 && sizes[i] > item.Max:
			sizes[i] = item.Max
		default:
			free = append(free, i)
		}

		used += sizes[i]
	}

	// Settle the difference clamping caused, one cell at a time so nothing goes
	// negative and nothing is pushed below the Min it asked for. When even that
	// leaves the items over the total, the box is simply too small: Ctx.Inset
	// clamps the overflow away, and the items nearest the end lose out.
	for used > total && len(free) > 0 {
		moved := false

		for _, i := range free {
			if sizes[i] > items[i].Min && sizes[i] > 0 {
				sizes[i]--
				used--
				moved = true

				if used == total {
					return
				}
			}
		}

		if !moved {
			return
		}
	}

	if used < total && len(free) > 0 {
		sizes[free[0]] += total - used
	}
}
