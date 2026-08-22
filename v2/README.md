# reactea v2

A companion to [Bubble Tea v2](https://github.com/charmbracelet/bubbletea) that
adds a component hierarchy, routing, layout and modals.

```sh
go get github.com/Hayao0819/reactea/v2
```

## Quickstart

```go
type App struct {
	router *router.Component
}

func (a *App) Init(ctx *reactea.Ctx) tea.Cmd { return a.router.Init(ctx) }
func (a *App) Destroy()                      { a.router.Destroy() }

func (a *App) Update(ctx *reactea.Ctx, msg tea.Msg) tea.Cmd {
	if key, ok := msg.(tea.KeyPressMsg); ok && key.String() == "q" {
		return tea.Quit
	}

	return a.router.Update(ctx, msg)
}

func (a *App) Render(ctx *reactea.Ctx) string {
	ctx.AltScreen(true)

	return a.router.Render(ctx)
}

func main() {
	app := reactea.New(&App{router: router.NewWithRoutes(routes)})

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
```

## The component

```go
type Component interface {
	Init(*Ctx) tea.Cmd
	Update(*Ctx, tea.Msg) tea.Cmd
	Render(*Ctx) string
	Destroy()
}
```

Four methods, one argument in common. `BasicComponent` supplies no-op versions
of everything but `Render`, so most components only write what they mean.

Lifecycle is the parent's job: a component that owns children forwards `Init`,
`Update` and `Destroy` to them, and `Render`s them into whatever boxes it decides
on. `layout` does this for you in the common cases.

## Ctx

`Ctx` is what a component is told about the frame it is taking part in.

| | |
|---|---|
| `Size()`, `Width()`, `Height()` | the box this component may draw into |
| `Inset(dx, dy, w, h)` | the box for a child, in the parent's coordinates |
| `Route()`, `PreviousRoute()` | where the app is |
| `SetRoute(r)`, `Navigate(r)` | commands that move it |
| `SetCursor`, `CursorAt`, `AltScreen`, `Title`, `MouseMode`, `ReportFocus`, `BackgroundColor`, `ForegroundColor` | the terminal features Bubble Tea v2 moved into `tea.View` |

Two things follow from `Ctx` carrying an origin. A cursor set through it is
translated into screen coordinates automatically, however deep the component
sits, so no parent does offset arithmetic. And decorations are per frame: a
component that stops asking for the alt-screen gets a view without it, with no
state to unwind.

Routing is a message, not a mutation. `SetRoute` and `Navigate` return commands,
so they are safe to issue from a command goroutine, and the move arrives at the
tree as a `RouteChangedMsg` where every other message arrives.

## Quitting

`tea.Quit`, Ctrl+C and a `SIGTERM` all run the tree's `Destroy` before the
program ends — `App` installs a `tea.WithFilter` to catch the quit before
Bubble Tea's event loop returns on it.

## Layout

```go
layout.Column(
	layout.Fixed(1, header),
	layout.Grow(1, layout.Row(
		layout.Fixed(16, layout.Framed(paneStyle, sidebar)),
		layout.Grow(1, layout.Framed(paneStyle, pages)),
	)),
	layout.Fixed(1, footer),
)
```

`Fixed` takes exactly that many cells, `Grow` takes a share of what is left
weighted against the other growing items, and `Bounded` is `Grow` with a floor
and a ceiling. Every cell is handed out: the remainder from an uneven split goes
to the items with the largest fractional share, so three `Grow(1, …)` items in a
10-cell box get 4, 3 and 3.

`Framed` draws a lipgloss style around a component. Lipgloss counts `Width` and
`Height` as the outer size, so the child is rendered at the box minus the border,
padding and margin, and its cursor shifted to match.

A `Box` recomputes its split in each phase rather than caching it from the last
`Render`, so `Update` and `Render` can be called in any order.

## Routing

```go
router.NewWithRoutes(router.Routes{
	"/user/settings": func(router.Params) reactea.Component { return settings.New() },
	"/user/:id":      func(p router.Params) reactea.Component { return profile.New(p["id"]) },
	"default":        func(router.Params) reactea.Component { return home.New() },
})
```

Placeholders capture params (`:id`), may be optional (`?:id`) or a trailing
catch-all (`+?:rest`). When more than one matches, the most specific wins —
literal over param over optional over catch-all, with a string tie-break, so the
choice never depends on Go's map iteration order. Set `NotFound` for a page of
your own; otherwise an unmatched route renders a plain message.

## Modals

```go
func (p *Page) Update(ctx *reactea.Ctx, msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		return p.stack.Push(&NameInput{})
	case modal.Result[string]:
		p.name = msg.Value
	}

	return nil
}
```

A modal is an ordinary component pushed onto a `modal.Stack`. While it is on top
it takes the input and the base sees nothing; it finishes with `modal.Return` or
`modal.Fail`, which pops it and delivers a `modal.Result[T]` to the tree. Nothing
blocks — no extra goroutine, no channel handshake.

## Wrapping Bubble Tea models and bubbles widgets

The two need different adapters, because a bubbles widget is not a `tea.Model`
and never has been. A widget's `Update` returns its own concrete type
(`func (m Model) Update(tea.Msg) (Model, tea.Cmd)`) so that you can write
`m.input, cmd = m.input.Update(msg)` without a type assertion, and Go has no
covariant returns. In v2 the `View() string` signature is a second mismatch.

| | Wraps | Constraint |
|---|---|---|
| `Reactify` | a self-contained Bubble Tea model | `tea.Model` |
| `ReactifyWidget` | a bubbles widget | `Widget[T]` |

```go
input := textinput.New()
input.SetVirtualCursor(false)
input.Focus()

component := reactea.ReactifyWidget(input)
```

`ReactifyWidget` stores the widget value back after every `Update`, calls the
widget's `Init()` when it has one, and reports its cursor through the `Ctx`.
Widgets draw a virtual cursor into their string by default and report no real
cursor in that mode; reactea leaves that choice to you.

An interface whose `Update` returns the interface itself also fits `Widget[T]` —
name it as the type argument, as in `ReactifyWidget[huh.Model](form)`.

## Testing

An `App` runs without a terminal, which is all a component test needs.

```go
app := reactea.New(page, reactea.WithSize(70, 20), reactea.WithRoute("/inbox"))

app.Init()
app.Update(tea.KeyPressMsg{Code: 'r', Text: "r"})

if !strings.Contains(app.View().Content, "Reloading") {
	t.Error(...)
}
```

`App.View()` returns the whole `tea.View`, so cursor, alt-screen and title are
assertable too. Two apps in one process share nothing, so tests can run in
parallel.
