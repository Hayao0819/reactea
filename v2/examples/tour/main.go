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
	stack *modal.Stack
}

func newRoot() *root {
	pages := router.NewWithRoutes(router.Routes{
		"/inbox":    func(router.Params) reactea.Component { return newInbox() },
		"/mail/:id": func(p router.Params) reactea.Component { return newMail(p["id"]) },
		"default":   func(router.Params) reactea.Component { return reactea.Text("nothing here") },
	})

	pages.NotFound = func(ctx *reactea.Ctx) string { return "no page at " + ctx.Route() }

	body := layout.Column(
		layout.Fixed(1, reactea.Func(func(ctx *reactea.Ctx) string {
			return headerStyle.Width(ctx.Width()).Render(" reactea tour  " + ctx.Route())
		})),
		layout.Grow(1, layout.Row(
			layout.Bounded(1, 12, 20, layout.Framed(paneStyle, newSidebar())),
			layout.Grow(3, layout.Framed(paneStyle, pages)),
		)),
		layout.Fixed(1, reactea.Text(" tab compose · q quit")),
	)

	return &root{stack: modal.New(body)}
}

func (r *root) Init(ctx *reactea.Ctx) tea.Cmd { return r.stack.Init(ctx) }

func (r *root) Render(ctx *reactea.Ctx) string { return r.stack.Render(ctx) }

func (r *root) Update(ctx *reactea.Ctx, msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return tea.Quit
		case "tab":
			return r.stack.Push(newCompose())
		}

	case modal.Result[string]:
		if msg.Ok() {
			return ctx.SetRoute("/mail/" + msg.Value)
		}
	}

	return r.stack.Update(ctx, msg)
}

type sidebar struct {
	reactea.BasicComponent
}

func newSidebar() *sidebar { return &sidebar{} }

func (c *sidebar) Render(ctx *reactea.Ctx) string {
	marker := "  "
	if ctx.Route() == "/inbox" {
		marker = "▸ "
	}

	return marker + "Inbox\n  Sent"
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
	if key, ok := msg.(tea.KeyPressMsg); ok && key.String() == "esc" {
		return ctx.Navigate("..")
	}

	return nil
}

type compose struct {
	reactea.BasicComponent

	input *reactea.ReactifiedWidget[textinput.Model]
}

func newCompose() *compose {
	input := textinput.New()
	input.SetVirtualCursor(false)
	input.Focus()

	return &compose{input: reactea.ReactifyWidget(input)}
}

func (c *compose) Init(ctx *reactea.Ctx) tea.Cmd { return c.input.Init(ctx) }

func (c *compose) Update(ctx *reactea.Ctx, msg tea.Msg) tea.Cmd {
	if key, ok := msg.(tea.KeyPressMsg); ok {
		switch key.String() {
		case "enter":
			return modal.Return(c.input.Widget.Value())
		case "esc":
			return modal.Dismiss
		}
	}

	return c.input.Update(ctx, msg)
}

func (c *compose) Render(ctx *reactea.Ctx) string {
	return paneStyle.Width(ctx.Width()).Render("open mail id:\n" + c.input.Render(ctx.Inset(1, 1, ctx.Width()-2, 1)))
}
