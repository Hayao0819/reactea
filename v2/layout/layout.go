// Package layout splits a box among child components along one axis.
package layout

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/Hayao0819/reactea/v2"
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
type Box struct {
	direction Direction
	items     []Item
}

// New builds a Box laying out along direction.
func New(direction Direction, items ...Item) *Box {
	return &Box{direction: direction, items: items}
}

// Column stacks items top to bottom.
func Column(items ...Item) *Box { return New(Vertical, items...) }

// Row places items left to right.
func Row(items ...Item) *Box { return New(Horizontal, items...) }

func (b *Box) Init(ctx *reactea.Ctx) tea.Cmd {
	cmds := make([]tea.Cmd, 0, len(b.items))

	b.each(ctx, func(item Item, childCtx *reactea.Ctx) {
		cmds = append(cmds, item.Component.Init(childCtx))
	})

	return tea.Batch(cmds...)
}

func (b *Box) Update(ctx *reactea.Ctx, msg tea.Msg) tea.Cmd {
	cmds := make([]tea.Cmd, 0, len(b.items))

	b.each(ctx, func(item Item, childCtx *reactea.Ctx) {
		cmds = append(cmds, item.Component.Update(childCtx, msg))
	})

	return tea.Batch(cmds...)
}

func (b *Box) Render(ctx *reactea.Ctx) string {
	rendered := make([]string, 0, len(b.items))

	b.each(ctx, func(item Item, childCtx *reactea.Ctx) {
		rendered = append(rendered, item.Component.Render(childCtx))
	})

	if b.direction == Vertical {
		return lipgloss.JoinVertical(lipgloss.Left, rendered...)
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, rendered...)
}

// The split is recomputed per phase rather than cached, so Update and Render can
// be called in any order.
func (b *Box) each(ctx *reactea.Ctx, visit func(Item, *reactea.Ctx)) {
	width, height := ctx.Size()

	main, cross := width, height
	if b.direction == Vertical {
		main, cross = height, width
	}

	sizes := distribute(main, b.items)

	position := 0

	for i, item := range b.items {
		if b.direction == Vertical {
			visit(item, ctx.Inset(0, position, cross, sizes[i]))
		} else {
			visit(item, ctx.Inset(position, 0, sizes[i], cross))
		}

		position += sizes[i]
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
	// negative.
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

	if used < total && len(free) > 0 {
		sizes[free[0]] += total - used
	}
}
