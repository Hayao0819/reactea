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

func (a *App) Init(ctx *reactea.Ctx) tea.Cmd {
	return tea.Batch(a.router.Init(ctx), reactea.EnterAltScreen)
}

func (a *App) Update(ctx *reactea.Ctx, msg tea.Msg) tea.Cmd {
	if key, ok := msg.(tea.KeyPressMsg); ok && key.String() == "q" {
		return tea.Quit
	}

	return a.router.Update(ctx, msg)
}

func (a *App) Render(ctx *reactea.Ctx) string {
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
}
```

Three methods, one argument in common. `BasicComponent` supplies no-op versions
of `Init` and `Update`, so most components only write what they mean.

A component that owns children forwards `Init` and `Update` to them and
`Render`s them into whatever boxes it decides on. `layout` does this for you in
the common cases.

`Render` should be a function of the component's state. Terminal features are
asked for with commands, not while drawing:

```go
func (c *App) Init(ctx *reactea.Ctx) tea.Cmd {
	return tea.Batch(reactea.EnterAltScreen, reactea.SetWindowTitle("my-app"))
}
```

`EnterAltScreen`, `ExitAltScreen`, `SetWindowTitle`, `SetMouseMode`,
`SetReportFocus`, `SetBackgroundColor`, `SetForegroundColor` and
`SetKeyboardEnhancements` all work this way: the App holds what was asked for and
puts it on every frame. The cursor is the exception — it depends on the layout,
which only exists while rendering — so it is set through the `Ctx`.

## Cleanup

There is no `Destroy`. A component that owns a resource says so where it acquires
it:

```go
func (c *Page) Init(ctx *reactea.Ctx) tea.Cmd {
	ticker := time.NewTicker(time.Second)
	ctx.OnDestroy(ticker.Stop)

	return c.poll(ticker.C)
}
```

Cleanups belong to a `Scope`. The app has a root scope that closes when the
program ends — through `tea.Quit`, Ctrl+C or a signal alike — so nothing is
stranded by a parent that forgot to forward a call. A parent that mounts and
unmounts children gives each one `ctx.Scope().Child()` and closes it when the
child goes; `router` and `modal` already do, so routing away from a page runs
that page's cleanups and nothing else.

## Ctx

`Ctx` is what a component is told about the frame it is taking part in.

| | |
|---|---|
| `Size()`, `Width()`, `Height()` | the box this component may draw into |
| `Inset(dx, dy, w, h)` | the box for a child, in the parent's coordinates |
| `Route()`, `PreviousRoute()` | where the app is |
| `SetRoute(r)`, `Navigate(r)` | commands that move it |
| `SetCursor`, `CursorAt` | where the terminal cursor goes this frame |
| `OnDestroy`, `Scope`, `WithScope` | cleanup, and which scope it belongs to |

A cursor set through a `Ctx` is translated into screen coordinates
automatically, however deep the component sits, so no parent does offset
arithmetic. It is also per frame: a component that stops asking for the cursor
gets a view without one, with no state to unwind.

Routing is a message, not a mutation. `SetRoute` and `Navigate` return commands,
so they are safe to issue from a command goroutine, and the move arrives at the
tree as a `RouteChangedMsg` where every other message arrives.

## Quitting

`tea.Quit`, Ctrl+C and a `SIGTERM` all close the root scope before the program
ends — `App` installs a `tea.WithFilter` to catch the quit before Bubble Tea's
event loop returns on it.

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

`Box` pads or trims each child to the box it was given, so a child that renders
short or long shifts nothing else; an item handed no cells is left out of the
frame entirely. `Frame` trims the child to its inner box before drawing the
border, so the border always has four sides.

`Fixed` takes exactly that many cells — `Fixed(0, c)` is flexible, not hidden —
`Grow` takes a share of what is left weighted against the other growing items,
and `Bounded` is `Grow` with a floor and a ceiling that the split honours only
while there is room for it. Every cell is handed out: the remainder from an uneven split goes
to the items with the largest fractional share, so three `Grow(1, …)` items in a
10-cell box get 4, 3 and 3.

`Framed` draws a lipgloss style around a component. Lipgloss counts `Width` and
`Height` as the outer size, so the child is rendered at the box minus the border,
padding and margin, and its cursor shifted to match.

`Box.SetItems` replaces the children while the program runs — hiding a pane,
maximising one, reordering them — and keeps the focus on the same component when
it is still there. `Frame.SetStyle` does the same for a border. Neither rebuilds
the tree, so nothing loses its state.

A `Box` recomputes its split in each phase rather than caching it from the last
`Render`, so `Update` and `Render` can be called in any order.

## Focus and input routing

Messages fall into two kinds. **Input** — keys, paste, mouse — has an addressee:
the keyboard reaches whatever holds the focus, the mouse reaches whatever sits
under the pointer. **Everything else** — ticks, async results, window size, route
changes — reaches the whole tree. `IsKeyboard`, `IsMouse` and `IsInput` are
exported so a custom container can route the same way.

Mark the items that can take the focus, and let the app decide which key moves
it:

```go
body := layout.Row(
    layout.Fixed(20, layout.Framed(paneStyle, cpu).WhenFocused(activeStyle)).Focusable(),
    layout.Grow(1, layout.Framed(paneStyle, procs).WhenFocused(activeStyle)).Focusable(),
)

func (r *root) Update(ctx *reactea.Ctx, msg tea.Msg) tea.Cmd {
    if reactea.Key(msg, "tab") {
        if !r.body.FocusNext() {
            r.body.FocusFirst()
        }

        return nil
    }

    return r.body.Update(ctx, msg)
}
```

A `Box` with nothing focusable in it drops keyboard messages, so a component that
expects keys must be marked `Focusable()` — otherwise it also never gets the
cursor, since only a focused component may set one.

Capturing and focus are separate mechanisms: `CaptureInput` silences the root's
global keys, it does not redirect them. Whatever captures must hold the focus
too, or the keys go elsewhere. `ctx.CaptureInput()` ties the claim to the Ctx's
scope, so a page routed away from mid-typing releases it automatically — even if
the claim was still in flight when the scope closed. The package-level
`reactea.CaptureInput` is the unscoped form and must be paired.

`FocusNext` descends into a nested box before advancing, and reports false at the
end so the caller decides how to wrap. A component reads `ctx.Focused()` to style
itself, and **only a focused component may set the cursor** — one cursor per frame
falls out of the focus rules instead of being a race between siblings.

Global keys are read above the tree, so they have to stand down while something
below is typing. A component that takes the keys says so, and the root asks:

```go
// entering filter mode
return reactea.CaptureInput

// leaving it
return reactea.ReleaseInput

func (r *root) Update(ctx *reactea.Ctx, msg tea.Msg) tea.Cmd {
    if ctx.InputCaptured() {
        return r.body.Update(ctx, msg)
    }
    ...
}
```

`modal.Stack` captures and releases on its own, so a modal needs nothing from the
app. Calls nest, so a capture must be paired with a release.

Mouse events arrive with box-local coordinates, so a click is `msg.Y` rows into
your own pane. A press also moves the focus to what was pressed; a wheel or a
motion leaves it alone.

```go
func (c *procs) Update(ctx *reactea.Ctx, msg tea.Msg) tea.Cmd {
    switch msg := msg.(type) {
    case tea.MouseClickMsg:
        c.selected = c.offset + msg.Y
    case tea.MouseWheelMsg:
        c.scroll(msg.Button)
    }

    return nil
}
```

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
it takes the input, though the base keeps receiving its own ticks and async
results so its work can finish; it finishes with `modal.Return` or
`modal.Fail`, which pops it and delivers a `modal.Result[T]` to the tree. Nothing
blocks — no extra goroutine, no channel handshake. Each modal gets its own
scope, so dismissing one runs exactly its cleanups.

`Push` covers the whole box. `PushAt` puts the modal somewhere else and composites
it over the base, so a confirmation can sit in the middle with the page still
visible around it:

```go
return p.stack.PushAt(&Confirm{}, modal.Placement{
    X: modal.Center, Y: modal.Center, Width: 40, Height: 7,
})
```

A zero `Width` or `Height` spans that axis. A modal is fitted to its placement
the way a `Box` fits its children, so it is opaque over the base and cannot
overrun the stack. Clicks beside a placed modal reach nobody: the base is covered
as far as input goes, however much of it is visible, and it does not hold the
focus while a modal is up — which is what keeps its cursor from showing through.

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
Widgets spell their size setters differently, so tell it how:

```go
reactea.ReactifyWidget(vp).OnResize(func(v viewport.Model, w, h int) viewport.Model {
    v.SetWidth(w)
    v.SetHeight(h)

    return v
})
```

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

`App.View()` returns the whole `tea.View`, so the cursor is assertable too. The
alt-screen and the title arrive by command, so feed the batch `Init` returns back
through `Update` before asserting on them. Two apps in one process share nothing,
so tests can run in parallel.
