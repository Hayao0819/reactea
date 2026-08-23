// Command tour exercises every part of the reactea v2 API in one screen.
package main

import (
	"fmt"
	"log"
	"time"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/Hayao0819/reactea/v2"
	"github.com/Hayao0819/reactea/v2/layout"
	"github.com/Hayao0819/reactea/v2/modal"
	"github.com/Hayao0819/reactea/v2/router"
)

var (
	headerStyle = lipgloss.NewStyle().Bold(true).Reverse(true)
	paneStyle   = lipgloss.NewStyle().Border(lipgloss.RoundedBorder())
)

func main() {
	app := reactea.New(
		newRoot(),
		reactea.WithRoute("/inbox"),
		reactea.WithAltScreen(),
		reactea.WithWindowTitle("reactea tour"),
	)

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}

type root struct {
	reactea.Wrapper

	stack *modal.Stack
}

func newRoot() *root {
	pages := router.NewWithRoutes(router.Routes{
		"/inbox":    router.Page(newInbox),
		"/mail/:id": router.Param("id", newMail),
		"default":   router.Page(func() reactea.Component { return reactea.Text("nothing here") }),
	})

	pages.NotFound = func(ctx *reactea.Ctx) string { return "no page at " + ctx.Route() }

	stack := modal.New(layout.Column(
		layout.Fixed(1, reactea.Func(func(ctx *reactea.Ctx) string {
			return headerStyle.Width(ctx.Width()).Render(" reactea tour  " + ctx.Route())
		})),
		layout.Grow(1, layout.Row(
			layout.Bounded(1, 12, 20, layout.Framed(paneStyle, newSidebar())),
			layout.Grow(3, layout.Framed(paneStyle, pages)),
		)),
		layout.Fixed(1, reactea.Text(" tab compose · q quit")),
	))

	return &root{Wrapper: reactea.Wrap(stack), stack: stack}
}

func (r *root) Update(ctx *reactea.Ctx, msg tea.Msg) tea.Cmd {
	switch {
	case reactea.Key(msg, "q", "ctrl+c"):
		return tea.Quit
	case reactea.Key(msg, "tab"):
		return r.stack.Push(newCompose())
	}

	if answer, ok := msg.(modal.Result[string]); ok && answer.Ok() {
		return ctx.SetRoute("/mail/" + answer.Value)
	}

	return r.Wrapper.Update(ctx, msg)
}

func newSidebar() reactea.Component {
	return reactea.Func(func(ctx *reactea.Ctx) string {
		if ctx.Route() == "/inbox" {
			return "▸ Inbox\n  Sent"
		}

		return "  Inbox\n  Sent"
	})
}

type inbox struct {
	reactea.BasicComponent

	ticks int
}

func newInbox() *inbox { return &inbox{} }

func (c *inbox) Init(ctx *reactea.Ctx) tea.Cmd {
	ticker := time.NewTicker(time.Second)

	ctx.OnDestroy(ticker.Stop)

	return func() tea.Msg { return <-ticker.C }
}

func (c *inbox) Update(_ *reactea.Ctx, msg tea.Msg) tea.Cmd {
	if _, ok := msg.(time.Time); ok {
		c.ticks++
	}

	return nil
}

func (c *inbox) Render(ctx *reactea.Ctx) string {
	width, height := ctx.Size()

	return fmt.Sprintf("inbox %dx%d, %d ticks", width, height, c.ticks)
}

type mail struct {
	reactea.BasicComponent

	id string
}

func newMail(id string) *mail { return &mail{id: id} }

func (c *mail) Render(*reactea.Ctx) string { return "mail " + c.id }

func (c *mail) Update(ctx *reactea.Ctx, msg tea.Msg) tea.Cmd {
	if reactea.Key(msg, "esc") {
		return ctx.Navigate("..")
	}

	return nil
}

type compose struct {
	reactea.Wrapper

	input *reactea.ReactifiedWidget[textinput.Model]
}

func newCompose() *compose {
	input := textinput.New()
	input.SetVirtualCursor(false)
	input.Focus()

	widget := reactea.ReactifyWidget(input)

	return &compose{Wrapper: reactea.Wrap(widget), input: widget}
}

func (c *compose) Update(ctx *reactea.Ctx, msg tea.Msg) tea.Cmd {
	switch {
	case reactea.Key(msg, "enter"):
		return modal.Return(c.input.Widget.Value())
	case reactea.Key(msg, "esc"):
		return modal.Dismiss
	}

	return c.Wrapper.Update(ctx, msg)
}

func (c *compose) Render(ctx *reactea.Ctx) string {
	return paneStyle.Width(ctx.Width()).Render(
		"open mail id:\n" + c.Wrapper.Render(ctx.Inset(1, 1, ctx.Width()-2, 1)))
}
