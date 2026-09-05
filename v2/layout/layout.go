// Package layout splits a box among child components along one axis.
package layout

import (
	"reflect"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/Hayao0819/reactea/v2"
	"github.com/Hayao0819/reactea/v2/render"
)

// Direction is the axis a Box lays its items out along.
type Direction int

const (
	Vertical Direction = iota
	Horizontal
)

type sizing uint8

const (
	fixed sizing = iota
	flexible
)

// Item is one child and how much of the main axis it wants.
type Item struct {
	component reactea.Component
	sizing    sizing
	size      int
	weight    int
	minimum   int
	maximum   int
	crossMin  int
	key       string
	focusable bool
}

// Focusable marks the item as something Tab can land on.
func (i Item) Focusable() Item {
	i.focusable = true

	return i
}

// IsFocusable reports what Focusable set, for code that rebuilds an item list.
func (i Item) IsFocusable() bool { return i.focusable }

// Key gives the item a stable name across reordered items and inserted spacers.
func (i Item) Key(key string) Item {
	i.key = key

	return i
}

// ItemKey reports what Key set.
func (i Item) ItemKey() string { return i.key }

// Bounds gives a growing item a minimum and maximum size. A zero maximum is
// unbounded.
func (i Item) Bounds(minimum, maximum int) Item {
	if i.sizing != flexible {
		panic("layout: bounds require a growing item")
	}

	if minimum < 0 || maximum < 0 || maximum > 0 && minimum > maximum {
		panic("layout: invalid bounds")
	}

	i.minimum, i.maximum = minimum, maximum

	return i
}

// MinCross reports the item as starved below size on the other axis.
func (i Item) MinCross(size int) Item {
	if size < 0 {
		panic("layout: negative cross-axis minimum")
	}

	i.crossMin = size

	return i
}

// sameItem decides whether focus follows an item across SetItems. Keys support
// components with uncomparable dynamic types.
func sameItem(a, b Item) bool {
	if a.key != "" || b.key != "" {
		return a.key == b.key
	}

	return sameComponent(a.component, b.component)
}

// sameComponent guards == with the dynamic type's comparability.
func sameComponent(a, b reactea.Component) bool {
	if a == nil || b == nil {
		return false
	}

	kind := reflect.TypeOf(a)

	return kind == reflect.TypeOf(b) && kind.Comparable() && a == b
}

// Fixed gives the child exactly size cells on the main axis.
func Fixed(size int, component reactea.Component) Item {
	if size < 0 {
		panic("layout: negative fixed size")
	}

	return Item{component: component, sizing: fixed, size: size}
}

// Grow gives the child a share of what is left, proportional to weight.
func Grow(weight int, component reactea.Component) Item {
	if weight <= 0 {
		panic("layout: grow weight must be positive")
	}

	return Item{component: component, sizing: flexible, weight: weight}
}

// Spacer is blank space of exactly size cells on the main axis.
func Spacer(size int) Item { return Fixed(size, reactea.Text("")) }

// Focuser is a container that can move the focus among its children. Box
// implements it, so nested boxes hand Tab down before advancing themselves.
type Focuser = reactea.Focuser

// Box lays its items out along one axis and gives each the full cross axis.
type Box struct {
	direction Direction
	items     []Item

	focused int
	starved []Starvation
}

// New builds a Box laying out along direction.
func New(direction Direction, items ...Item) *Box {
	box := &Box{direction: direction, items: items, focused: -1}

	box.FocusFirst()

	return box
}

// Items returns a copy for editing and passing back to SetItems.
func (b *Box) Items() []Item { return append([]Item(nil), b.items...) }

// SetItems replaces the children while preserving mounted component state. It
// also preserves focus when the focused item remains present.
func (b *Box) SetItems(items ...Item) {
	var was Item

	if b.focused >= 0 && b.focused < len(b.items) {
		was = b.items[b.focused]
	}

	b.items = items
	b.focused = -1

	for i := range items {
		if sameItem(items[i], was) && b.takesFocus(i) {
			b.focused = i

			return
		}
	}

	b.FocusFirst()
}

// FocusKey moves the focus to the item named key, if it can take it.
func (b *Box) FocusKey(key string) bool { return b.Focus(b.indexOf(key)) }

// FocusedKey returns the focused item's key, or an empty string for an unkeyed
// item or empty focus.
func (b *Box) FocusedKey() string {
	if b.focused < 0 || b.focused >= len(b.items) {
		return ""
	}

	return b.items[b.focused].key
}

// Replace swaps an item's component while preserving its size, bounds and focus.
func (b *Box) Replace(key string, component reactea.Component) bool {
	index := b.indexOf(key)
	if index < 0 {
		return false
	}

	b.items[index].component = component

	return true
}

func (b *Box) indexOf(key string) int {
	if key == "" {
		return -1
	}

	for i := range b.items {
		if b.items[i].key == key {
			return i
		}
	}

	return -1
}

// StarveReason says which way an item came up short.
type StarveReason int

const (
	// NoSpace means the item received zero cells.
	NoSpace StarveReason = iota

	// MainAxis is below the fixed size or minimum set with Item.Bounds.
	MainAxis

	// CrossAxis is below the minimum set with Item.MinCross.
	CrossAxis
)

// Starvation describes an unsatisfied item size from the last split.
type Starvation struct {
	Key    string
	Index  int
	Reason StarveReason
}

// Starved reports unsatisfied items from the last split. Applications can use
// the result to hide compressed panes. The returned slice is a copy, and every
// phase refreshes the result, including a mouse move.
func (b *Box) Starved() []Starvation {
	if len(b.starved) == 0 {
		return nil
	}

	return append([]Starvation(nil), b.starved...)
}

// Focused is the index of the item holding the focus, or -1.
func (b *Box) Focused() int { return b.focused }

// Focus moves focus to item index when the item accepts it. Repeated focus keeps
// the pane's existing descendant focus.
func (b *Box) Focus(index int) bool {
	if index < 0 || index >= len(b.items) || !b.takesFocus(index) {
		return false
	}

	if index == b.focused {
		return true
	}

	b.focused = index

	if child, ok := b.items[index].component.(Focuser); ok {
		child.FocusFirst()
	}

	return true
}

// FocusNext moves to the next focusable leaf, descending into a nested box
// first. False marks the boundary where a caller can wrap with FocusFirst.
func (b *Box) FocusNext() bool { return b.step(1) }

// FocusPrev is FocusNext backwards; wrap it with FocusLast.
func (b *Box) FocusPrev() bool { return b.step(-1) }

// CycleNext moves forward and wraps to the first focusable item.
func (b *Box) CycleNext() {
	if !b.FocusNext() {
		b.FocusFirst()
	}
}

// CyclePrev moves backward and wraps to the last focusable item.
func (b *Box) CyclePrev() {
	if !b.FocusPrev() {
		b.FocusLast()
	}
}

func (b *Box) step(by int) bool {
	if b.focused >= 0 && b.focused < len(b.items) {
		if child, ok := b.items[b.focused].component.(Focuser); ok {
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

		if child, ok := b.items[i].component.(Focuser); ok {
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

	nested, ok := b.items[i].component.(Focuser)

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
		cmds = append(cmds, item.component.Init(childCtx))
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
		cmds = append(cmds, item.component.Update(childCtx, msg))
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
			cmd = item.component.Update(childCtx, msg)
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

	// A press moves focus to its target. Wheel and motion events preserve it.
	if _, press := msg.(tea.MouseClickMsg); press && b.takesFocus(hit) {
		b.Focus(hit)
		hitCtx = hitCtx.WithFocus(ctx.Focused())
	}

	offset := b.local(ctx, hitCtx, 0, 0)

	return b.items[hit].component.Update(hitCtx, reactea.TranslateMouse(msg, -offset.x, -offset.y))
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

		// Skip a zero-sized main axis during the join.
		if (b.direction == Vertical && height <= 0) || (b.direction == Horizontal && width <= 0) {
			return
		}

		rendered = append(rendered, render.Fit(item.component.Render(childCtx), width, height))
	})

	if b.direction == Vertical {
		return lipgloss.JoinVertical(lipgloss.Left, rendered...)
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, rendered...)
}

// Each phase computes its own split, allowing Update and Render in either order.
// Every child's box and all starvation records are settled before the first
// child is visited.
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
		if reason, short := starving(item, got, cross); short {
			b.starved = append(b.starved, Starvation{Key: item.key, Index: i, Reason: reason})
		}

		position += sizes[i]
	}

	for i, item := range b.items {
		visit(i, item, boxes[i])
	}
}

func starving(item Item, main, cross int) (StarveReason, bool) {
	switch {
	case item.sizing == flexible && main == 0, item.sizing == fixed && item.size > 0 && main == 0:
		return NoSpace, true
	case item.sizing == fixed && main < item.size, item.minimum > 0 && main < item.minimum:
		return MainAxis, true
	case item.crossMin > 0 && cross < item.crossMin:
		return CrossAxis, true
	}

	return 0, false
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
		if item.sizing == fixed {
			sizes[i] = min(item.size, remaining)
			remaining -= sizes[i]

			continue
		}

		grow[i] = item.weight
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
		if item.sizing == fixed {
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
			if item.sizing == fixed {
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
	used := 0

	for i, item := range items {
		if item.sizing == flexible {
			sizes[i] = bound(sizes[i], item)
		}

		used += sizes[i]
	}

	// Move one cell at a time to keep adjusted sizes inside their bounds. Ctx.Inset
	// clamps overflow from a fully constrained box.
	at := 0

	for used != total {
		step := 1
		if used > total {
			step = -1
		}

		next, ok := nudge(sizes, items, step, at)
		if !ok {
			return
		}

		at, used = next+1, used+step
	}
}

// bound is size brought inside the floor and ceiling item asked for.
func bound(size int, item Item) int {
	if item.minimum > 0 && size < item.minimum {
		size = item.minimum
	}

	if item.maximum > 0 && size > item.maximum {
		size = item.maximum
	}

	return size
}

// nudge moves one cell into or out of the first flexible item from start that
// can take it, wrapping once. Continuing from the previous position distributes
// the difference across items.
func nudge(sizes []int, items []Item, step, start int) (int, bool) {
	for offset := range items {
		i := (start + offset) % len(items)

		item := items[i]
		if item.sizing == fixed {
			continue
		}

		if size := sizes[i] + step; size >= 0 && size == bound(size, item) {
			sizes[i] = size

			return i, true
		}
	}

	return 0, false
}
