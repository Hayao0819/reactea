// Package layout splits a box among child components the way a single-axis
// flexbox does, so a parent no longer computes child sizes or cursor offsets by
// hand.
package layout

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/Hayao0819/reactea/v2"
)

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
}

// Fixed gives the child exactly size cells on the main axis.
func Fixed(size int, component reactea.Component) Item {
	return Item{Component: component, Size: size}
}

// Grow gives the child a share of the leftover space, proportional to weight
// against the other growing items.
func Grow(weight int, component reactea.Component) Item {
	return Item{Component: component, Grow: weight}
}

// Bounded is Grow with a floor and a ceiling. A zero max means unbounded.
func Bounded(weight, min, max int, component reactea.Component) Item {
	return Item{Component: component, Grow: weight, Min: min, Max: max}
}

// Box lays its items out along one axis and gives each the full cross axis.
type Box struct {
	direction Direction
	items     []Item

	// offsets are where the last Render placed each item, which is what makes
	// automatic cursor translation possible.
	offsets []offset
}

type offset struct{ x, y int }

func New(direction Direction, items ...Item) *Box {
	return &Box{direction: direction, items: items}
}

// Column stacks items top to bottom.
func Column(items ...Item) *Box { return New(Vertical, items...) }

// Row places items left to right.
func Row(items ...Item) *Box { return New(Horizontal, items...) }

func (b *Box) Init() tea.Cmd {
	cmds := make([]tea.Cmd, 0, len(b.items))
	for _, item := range b.items {
		cmds = append(cmds, item.Component.Init())
	}

	return tea.Batch(cmds...)
}

func (b *Box) Update(msg tea.Msg) tea.Cmd {
	cmds := make([]tea.Cmd, 0, len(b.items))
	for _, item := range b.items {
		cmds = append(cmds, item.Component.Update(msg))
	}

	return tea.Batch(cmds...)
}

func (b *Box) Destroy() {
	for _, item := range b.items {
		item.Component.Destroy()
	}
}

func (b *Box) Render(width, height int) string {
	main, cross := width, height
	if b.direction == Vertical {
		main, cross = height, width
	}

	sizes := distribute(main, b.items)

	rendered := make([]string, 0, len(b.items))
	b.offsets = make([]offset, len(b.items))

	position := 0

	for i, item := range b.items {
		if b.direction == Vertical {
			b.offsets[i] = offset{x: 0, y: position}
			rendered = append(rendered, item.Component.Render(cross, sizes[i]))
		} else {
			b.offsets[i] = offset{x: position, y: 0}
			rendered = append(rendered, item.Component.Render(sizes[i], cross))
		}

		position += sizes[i]
	}

	if b.direction == Vertical {
		return lipgloss.JoinVertical(lipgloss.Left, rendered...)
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, rendered...)
}

// DecorateView collects each child's decoration, translating any cursor by the
// offset the last Render placed that child at. Later items win on the scalar
// fields; a cursor is taken from the last item that asked for one.
func (b *Box) DecorateView(view *tea.View) {
	for i, item := range b.items {
		child := tea.NewView("")

		reactea.DecorateView(item.Component, &child)

		if i < len(b.offsets) {
			reactea.TranslateCursor(&child, b.offsets[i].x, b.offsets[i].y)
		}

		merge(view, child)
	}
}

func merge(into *tea.View, from tea.View) {
	if from.Cursor != nil {
		into.Cursor = from.Cursor
	}

	if from.AltScreen {
		into.AltScreen = true
	}

	if from.ReportFocus {
		into.ReportFocus = true
	}

	if from.DisableBracketedPasteMode {
		into.DisableBracketedPasteMode = true
	}

	if from.WindowTitle != "" {
		into.WindowTitle = from.WindowTitle
	}

	if from.MouseMode != 0 {
		into.MouseMode = from.MouseMode
	}

	if from.BackgroundColor != nil {
		into.BackgroundColor = from.BackgroundColor
	}

	if from.ForegroundColor != nil {
		into.ForegroundColor = from.ForegroundColor
	}

	if from.ProgressBar != nil {
		into.ProgressBar = from.ProgressBar
	}
}

// distribute hands out the main axis: fixed items first, then the remainder to
// growing items by weight. Clamping happens in a second pass, and what clamping
// frees up is shared among the items that are still free to move.
func distribute(total int, items []Item) []int {
	sizes := make([]int, len(items))

	if total < 0 {
		total = 0
	}

	remaining := total
	weight := 0

	for i, item := range items {
		if item.Size > 0 {
			sizes[i] = min(item.Size, remaining)
			remaining -= sizes[i]

			continue
		}

		// A flexible item with no weight still deserves a share.
		if item.Grow <= 0 {
			items[i].Grow = 1
		}

		weight += items[i].Grow
	}

	if weight == 0 {
		return sizes
	}

	share(sizes, items, remaining, weight)
	clamp(sizes, items, total)

	return sizes
}

func share(sizes []int, items []Item, remaining, weight int) {
	// Largest-remainder apportionment: hand out the floor of each share, then
	// give the leftover cells to the items with the biggest fractional part, so
	// the sizes always add up to remaining.
	leftover := remaining
	fractions := make([]int, len(items))

	for i, item := range items {
		if item.Size > 0 {
			continue
		}

		exact := remaining * item.Grow
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

func clamp(sizes []int, items []Item, total int) {
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

	// Clamping can push the total off; settle the difference on whatever is
	// still free to move, one cell at a time so nothing goes negative.
	for used > total && len(free) > 0 {
		moved := false

		for _, i := range free {
			if sizes[i] > 0 {
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

	for used < total && len(free) > 0 {
		sizes[free[0]] += total - used

		return
	}
}
