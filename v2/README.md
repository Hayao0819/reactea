# reactea v2

A companion to [Bubble Tea v2](https://github.com/charmbracelet/bubbletea) that
adds a component hierarchy, routing, layout and modals.

```sh
go get github.com/Hayao0819/reactea/v2
```

## Scope

reactea supplies a component tree and lifecycle on top of Bubble Tea. Scopes
bind cleanup to mounted components, and the app closes the root scope for Bubble
Tea quit and interruption messages.

Components receive `tea.Msg` values directly and return strings from `Render`.
Containers route input by focus and pointer, then fit child output to the boxes
they assign. Applications pass values such as stores and themes through
constructors. `layout` divides one axis, applies item bounds and reports
starvation.

## Quickstart

```go
type App struct {
	router *router.Component
}

func (a *App) Init(ctx *reactea.Ctx) tea.Cmd {
	return tea.Batch(a.router.Init(ctx), reactea.EnterAltScreen())
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

## Examples

| | |
|---|---|
| `examples/minimal` | one component with state, one key, a line to run it |
| `examples/component` | the shapes a component takes, and the two ways a parent mounts one |
| `examples/tour` | routing, layout, modals, focus and widget adapters in one screen |

## The component

```go
type Component interface {
	Init(*Ctx) tea.Cmd
	Update(*Ctx, tea.Msg) tea.Cmd
	Render(*Ctx) string
}
```

Three methods, one argument in common. `BasicComponent` supplies default `Init`
and `Update` implementations, so most components implement their relevant
methods.

A component that owns children forwards `Init` and `Update` to them and
`Render`s them into whatever boxes it decides on. `layout` does this for you in
the common cases.

`Render` should be a function of the component's state. Components request
terminal features through commands:

```go
func (c *App) Init(ctx *reactea.Ctx) tea.Cmd {
	return tea.Batch(reactea.EnterAltScreen(), reactea.SetWindowTitle("my-app"))
}
```

`EnterAltScreen`, `ExitAltScreen`, `SetWindowTitle`, `SetMouseMode`,
`SetReportFocus`, `SetBackgroundColor`, `SetForegroundColor` and
`SetKeyboardEnhancements` all work this way: the App holds what was asked for and
puts it on every frame. Cursor coordinates depend on the current layout, so a
component sets the cursor through its `Ctx` during rendering.

## Cleanup

Scopes bind cleanup to component lifetime. A component registers a resource
where it acquires one:

```go
func (c *Page) Init(ctx *reactea.Ctx) tea.Cmd {
	ticker := time.NewTicker(time.Second)
	ctx.OnDestroy(ticker.Stop)

	return c.poll(ticker.C)
}
```

Background work uses the scope's context:

```go
func (c *Page) Init(ctx *reactea.Ctx) tea.Cmd {
	return c.follow(ctx.Context())
}
```

Closing the scope cancels its context.

Cleanups belong to a `Scope`. The app closes its root scope for `tea.Quit` and
Bubble Tea interruption messages such as Ctrl+C or a termination signal. A
parent that mounts and unmounts children gives each one `ctx.Scope().Child()`
and closes it on unmount. `router` and `modal` manage these child scopes, so
routing away from a page runs the cleanups registered by that page.

## Ctx

`Ctx` is what a component is told about the frame it is taking part in.

| | |
|---|---|
| `Size()`, `Width()`, `Height()` | the box this component may draw into |
| `Inset(dx, dy, w, h)` | the box for a child, in the parent's coordinates |
| `Route()`, `PreviousRoute()` | where the app is |
| `SetRoute(r)`, `Navigate(r)` | commands that move it |
| `SetCursor`, `CursorAt` | where the terminal cursor goes this frame |
| `OnDestroy`, `Context` | cleanup and cancellation when the component is unmounted |

A cursor set through a `Ctx` is translated into screen coordinates at any depth.
Cursor state is per frame and appears on each frame where a component requests
it.

Routing uses messages. `SetRoute` and `Navigate` return commands, so they are
safe to issue from a command goroutine, and the route change reaches the tree as
a `RouteChangedMsg`.

Application components normally use the methods above. `Inset`, `WithFocus`,
`WithScope`, `WithOverlay`, `Scope`, `Origin`, `IsInput`, `TranslateMouse` and
`FocusOf` support containers that mount children or route input. `layout`,
`router`, `modal`, `Frame` and `Wrapper` provide those container behaviours.

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

`Box` pads or trims each child to its assigned size, keeping sibling positions
stable. Items assigned zero cells produce empty output. `Frame` trims the child
to its inner box before drawing a complete border.

`Box.Starved()` reports unsatisfied items from the last split, each with its key
and reason: zero assigned cells, a size below its fixed size or bounds, or a
cross-axis size below `MinCross`. It is settled before any child renders, so a
header inside the same box can report it. Applications can hide starved panes by
updating the item list with `SetItems`.

`MinCross` sets a bound on the cross axis and reports starvation. Every child
receives the full cross-axis size.

`Spacer(n)` is blank space of exactly n cells, for the gap between two panes.

`Fixed` takes exactly that many cells, including zero. `Grow` takes a share of
what is left weighted against the other growing items. Add a floor and ceiling
with `Grow(1, child).Bounds(minimum, maximum)`; a zero maximum is unbounded.
Every cell that a growing item can accept is handed out: the remainder from an
uneven split goes to the items with the largest fractional share, so three
`Grow(1, …)` items in a 10-cell box get 4, 3 and 3.

`Framed` draws a lipgloss style around a component. Lipgloss counts `Width` and
`Height` as the outer size, so the child is rendered at the box minus the border,
padding and margin, and its cursor shifted to match.

`Box.SetItems` replaces the children while the program runs — hiding a pane,
maximising one or reordering them — while preserving focus and mounted
component state. `Frame.SetStyle` updates a border while preserving its child.

Name an item with `Key` for stable addressing across inserted spacers and
reordered items:

```go
layout.Grow(1, pages).Key("pane").Focusable()

box.FocusKey("pane")
box.FocusedKey()
box.Replace("pane", replacement)
```

`SetItems` matches on the key first, so focus follows a rebuilt item list and
also supports components with uncomparable dynamic types. `Items()` returns a
copy; apply changes by passing the edited copy to `SetItems`.

A `Box` computes its split independently in each phase, allowing `Update` and
`Render` in either order.

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
		r.body.CycleNext()

		return nil
	}

	return r.body.Update(ctx, msg)
}
```

`Box` delivers keyboard input and cursor ownership through its focused item.
Mark each box child that accepts keyboard input with `Focusable()`.

Capturing and focus are separate mechanisms: `InputCapture` silences the root's
global keys, while focus selects the input recipient. Give a capturing component
focus while it accepts keyboard input.

A container that wraps another passes the focus methods through — `Frame`,
`Memo`, `reactea.Wrapper`, `router.Component` and `modal.Stack` all do, via
`reactea.FocusOf` — so Tab reaches panes inside routed pages. Implement
`reactea.Focuser` to include a custom container in the same order.

`CycleNext` and `CyclePrev` descend into nested boxes and wrap at either end.
`FocusNext` and `FocusPrev` expose the boundary for a container that needs to
hand focus back to its parent. A component reads `ctx.Focused()` to style itself;
the focused component owns the cursor.

Global keys are read above the tree, so they have to stand down while something
below is typing. A component that takes the keys says so, and the root asks:

```go
type Filter struct {
	reactea.InputCapture
}

// entering filter mode
return f.CaptureInput(ctx)

// leaving it
return f.ReleaseInput()

func (r *root) Update(ctx *reactea.Ctx, msg tea.Msg) tea.Cmd {
	if ctx.InputCaptured() {
		return r.body.Update(ctx, msg)
	}
	...
}
```

Each `InputCapture` owns one idempotent claim. Releasing it preserves other
claims, and closing its scope releases it automatically. `modal.Stack` manages
its claim itself.

Mouse events arrive with box-local coordinates, so a click is `msg.Y` rows into
your own pane. A press moves focus to its target; wheel and motion events
preserve current focus.

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
	"/user/:id":      func(p router.Params) reactea.Component { return profile.New(p.Get("id")) },
	"/files/*rest":   func(p router.Params) reactea.Component { return files.New(p.Get("rest")) },
	"default":        func(router.Params) reactea.Component { return home.New() },
})
```

Placeholders capture params (`:id`), may be optional (`:id?`) or a trailing
catch-all (`*rest`). `Params.Path()` returns the complete matched path. When
more than one pattern matches, the most specific wins —
literal over param over optional over catch-all, with a string tie-break, so the
choice is deterministic. Set `NotFound` to customize the page shown for an
unmatched route; the default is a plain message.

`Reload(ctx)` rebuilds the page inside the update it is called from, so the frame
Bubble Tea draws next has one — which is what a configuration reload wants.
`SetRoutes(routes, mode)` replaces the table: `PreserveCurrent` leaves a mounted
page alone while its route still resolves the same way, and `RemountCurrent`
builds it again.

## Modals

Place one stack around the application root:

```go
app := reactea.New(modal.New(root))
```

```go
func (p *Page) Update(ctx *reactea.Ctx, msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		return modal.Push(ctx, &NameInput{})
	case modal.Result[string]:
		p.name = msg.Value
	}

	return nil
}
```

The package functions use `Ctx` to address the nearest stack:

```go
return modal.PushAt(ctx, &Confirm{}, modal.Centered(40, 8))
```

Each stack passes itself through `Ctx` while routing. In a nested stack, the
inner value replaces the enclosing value for its component branch.

A modal is an ordinary component pushed onto a `modal.Stack`. While it is on top
it takes the input, though the base keeps receiving its own ticks and async
results so its work can finish; it finishes with `modal.Return(ctx, value)` or
`modal.Fail(ctx, err)`, which closes that modal and delivers a `modal.Result[T]`
to the tree. `modal.Dismiss(ctx)` closes it silently. Each modal gets its own
scope, so dismissing one runs its registered cleanups.

`Push` covers the whole box. `PushAt` puts the modal somewhere else and composites
it over the base, so a confirmation can sit in the middle with the page still
visible around it:

```go
return modal.PushAt(ctx, &Confirm{}, modal.Centered(40, 7))
```

A zero `Width` or `Height` spans that axis. The stack fits a modal to its
placement and composites it opaquely over the base. While a modal is active, the
stack owns input across its full area; pointer events outside the placement are
absorbed, while the active modal owns focus and the cursor.

## Fitting text

`render` exports the terminal-cell operations used by the containers for reuse
in application tables:

```go
render.Fit(content, width, height)   // both axes, pad or trim
render.Left(text, width)             // one line, exactly that wide
render.Right(text, width)
render.Clip(text, width)             // trim to width
```

`Left` and `Right` trim by display width and then pad any remaining cell. This
also handles clipping a double-width character at an odd boundary.

## Wrapping Bubble Tea models and bubbles widgets

Bubble Tea models and bubbles widgets expose different method signatures. A
widget's `Update` returns its concrete type
(`func (m Model) Update(tea.Msg) (Model, tea.Cmd)`), while a `tea.Model` returns
`tea.Model`; Bubble Tea v2 also uses a different `View` return type. Reactea
provides an adapter for each shape.

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

Widgets draw a virtual cursor into their string by default. Disabling the
virtual cursor lets the adapter report the terminal cursor through `Ctx`.

An interface whose `Update` returns the interface itself also fits `Widget[T]` —
name it as the type argument, as in `ReactifyWidget[huh.Model](form)`.

## Performance

A frame costs what drawing costs. On a 20-pane bordered dashboard at 200x50:

```
BenchmarkRenderDashboard-12            3238854 ns/op    460548 B/op    11594 allocs/op
BenchmarkRenderDashboardMemoized-12    1669440 ns/op    167311 B/op      707 allocs/op
BenchmarkUpdateDashboard-12              36956 ns/op      3800 B/op       51 allocs/op
```

Rendering is ~3 ms and handling a message is ~37 µs, so at the one or two frames
a second a monitor redraws, drawing costs about a percent of a core. Most of it
is lipgloss styling the borders, which is why `layout.Memo` halves the frame and
takes the allocations from 11594 to 707:

```go
layout.Memo(pane, func() any { return snapshot.Generation })
```

`Init` and `Update` always run. The cache reuses render output until the key
changes or the box is resized. A **focused** pane renders every frame so it can
set the cursor.

Call `Invalidate()` after `SetItems` or `SetStyle`, because these external
mutations are independent of the memoization key.

Every Bubble Tea message walks the tree. For frequent data collection, write
readings to an application store and send the UI a consolidated redraw tick.

## Testing

An `App` supports in-process component tests:

```go
app := reactea.New(page, reactea.WithSize(70, 20), reactea.WithRoute("/inbox"))

app.Start()
app.Send(tea.KeyPressMsg{Code: 'r', Text: "r"})

if !strings.Contains(app.View().Content, "Reloading") {
	t.Error(...)
}
```

`Start` runs `Init` and its commands; `Send` handles a message and recursively
runs commands until the queue settles. For timers and recurring commands, call
`Init` or `Update` directly and deliver the event under test explicitly.

`App.View()` returns the whole `tea.View`, so the cursor is assertable too. The
alt-screen and the title arrive by command, so feed the batch `Init` returns back
through `Update` before asserting on them. Each app owns isolated state, so tests
can run in parallel.

`testkit` provides common terminal-application test operations:

```go
testkit.SendKeys(app, "/", "e", "r", "enter")
testkit.Click(app, 12, 3)

if !strings.Contains(testkit.Plain(app), "3 results") {
	t.Error(...)
}
```

`Plain` strips the styling, `SendKeys` spells keys the way `KeyPressMsg.String`
does, and `RenderAt` resizes before drawing.

`App.ReverseBatches(true)` runs the commands inside every `tea.Batch` back to
front. Bubble Tea permits either command order, so run ordering-sensitive tests
in both modes.
