// Command component shows the shapes a component takes, and the two ways a
// parent mounts one: by hand, and by handing the work to layout.
package main

import (
	"fmt"
	"log"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/Hayao0819/reactea/v2"
	"github.com/Hayao0819/reactea/v2/layout"
)

// A component with no state is a function.
func label(text string) reactea.Component {
	return reactea.Func(func(ctx *reactea.Ctx) string {
		return fmt.Sprintf("%s (%d wide)", text, ctx.Width())
	})
}

// One with state embeds BasicComponent and writes only what differs. Render is
// the only method it must have.
type clock struct {
	reactea.BasicComponent

	at time.Time
}

func (c *clock) Update(_ *reactea.Ctx, msg tea.Msg) tea.Cmd {
	if now, ok := msg.(time.Time); ok {
		c.at = now
	}

	return nil
}

func (c *clock) Render(*reactea.Ctx) string { return c.at.Format("15:04:05") }

// One that owns something that must be stopped says so where it starts it.
type ticker struct {
	reactea.BasicComponent

	ticks int
}

func (t *ticker) Init(ctx *reactea.Ctx) tea.Cmd {
	beat := time.NewTicker(time.Second)

	ctx.OnDestroy(beat.Stop)

	return func() tea.Msg { return <-beat.C }
}

func (t *ticker) Update(_ *reactea.Ctx, msg tea.Msg) tea.Cmd {
	if _, ok := msg.(time.Time); ok {
		t.ticks++
	}

	return nil
}

func (t *ticker) Render(*reactea.Ctx) string { return fmt.Sprintf("%d ticks", t.ticks) }

// A parent mounting children by hand has two jobs: forward the lifecycle, and
// carve each child's box. Both come from one place, or the three phases would
// disagree and hit-testing would land on the wrong component.
type split struct {
	reactea.BasicComponent

	top, bottom reactea.Component
}

func (s *split) Init(ctx *reactea.Ctx) tea.Cmd {
	top, bottom := s.boxes(ctx)

	return tea.Batch(s.top.Init(top), s.bottom.Init(bottom))
}

func (s *split) Update(ctx *reactea.Ctx, msg tea.Msg) tea.Cmd {
	top, bottom := s.boxes(ctx)

	return tea.Batch(s.top.Update(top, msg), s.bottom.Update(bottom, msg))
}

func (s *split) Render(ctx *reactea.Ctx) string {
	top, bottom := s.boxes(ctx)

	return s.top.Render(top) + "\n" + s.bottom.Render(bottom)
}

func (s *split) boxes(ctx *reactea.Ctx) (top, bottom *reactea.Ctx) {
	width, height := ctx.Size()

	return ctx.Inset(0, 0, width, 1), ctx.Inset(0, 1, width, height-1)
}

// layout does the same forwarding and carving, and holds each child to its box.
func laidOut() reactea.Component {
	return layout.Column(
		layout.Fixed(1, label("laid out")),
		layout.Grow(1, &clock{}),
		layout.Spacer(1),
		layout.Fixed(1, &ticker{}),
	)
}

// A parent with a single child embeds Wrapper and overrides what differs.
type quitting struct {
	reactea.Wrapper
}

func (q *quitting) Update(ctx *reactea.Ctx, msg tea.Msg) tea.Cmd {
	if reactea.Key(msg, "q", "ctrl+c") {
		return tea.Quit
	}

	return q.Wrapper.Update(ctx, msg)
}

func main() {
	byHand := &split{top: label("by hand"), bottom: &clock{}}

	root := &quitting{Wrapper: reactea.Wrap(layout.Column(
		layout.Fixed(2, byHand),
		layout.Grow(1, laidOut()),
	))}

	if err := reactea.New(root, reactea.WithAltScreen()).Run(); err != nil {
		log.Fatal(err)
	}
}
