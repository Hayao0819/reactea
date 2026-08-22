# <p align="center">Reactea</p>

<div align="center">

[![Latest](https://img.shields.io/github/v/tag/Hayao0819/reactea?label=latest)](https://img.shields.io/github/v/tag/Hayao0819/reactea?label=latest)
[![build](https://github.com/Hayao0819/reactea/actions/workflows/build.yml/badge.svg)](https://github.com/Hayao0819/reactea/actions/workflows/build.yml)
![Codecov](https://img.shields.io/codecov/c/github/Londek/reactea)
[![Go Reference](https://pkg.go.dev/badge/github.com/Hayao0819/reactea.svg)](https://pkg.go.dev/github.com/Hayao0819/reactea)
[![Go Report Card](https://goreportcard.com/badge/github.com/Hayao0819/reactea)](https://goreportcard.com/report/github.com/Hayao0819/reactea)

<p align="center">reactea for Bubbletea v2 (module <code>github.com/Hayao0819/reactea/v2</code>).</p>

Rather simple **Bubbletea companion** for **handling hierarchy**, support for **lifting state up.**\
It Reactifies Bubbletea philosophy and makes it especially easy to work with in bigger projects.

For me, personally - **It's a must** in project with multiple pages and component communication

Check our quickstart [right here](#quickstart) or other examples [here!](/examples)

`go get -u github.com/Hayao0819/reactea/v2`
</div>

## General info

The goal is to create components that are

- dimensions-aware (especially unify all setSize conventions)
- scallable
- more robust
- easier to code
- all of that without code duplication

Extreme performance is not main goal of this package, however it should not be
that far off actual Bubbletea which is already blazing fast.
Most info is currently in source code so I suggest checking it out

Always return `reactea.Destroy` instead of `tea.Quit` in order to follow our convention (that way Destroy() will be called on your components)

## [Quickstart](/examples/quickstart)

Reactea unlike Bubbletea implements two-way communication, very React-like communication.\
If you have experience with React you are gonna love Reactea straight away!

In this tutorial we are going to make application that consists of 2 pages.

- The `/input` (aka `index`, in reactea `default`) page for inputting your name
- The `/displayname` page for displaying your name

### [Lifecycle](#component-lifecycle)

More detailed docs about component lifecycle can be found [here](#component-lifecycle), we are only gonna go through basics.

Reactea component lifecycle consists of 4 methods (while Bubbletea only 3)
|Method|Purpose|
|-|-|
| `Init() tea.Cmd` | It's called first. All critical stuff should happen here. It also supports IO through tea.Cmd |
| `Update(tea.Msg) tea.Cmd` | It reacts to Bubbletea IO and updates state accordingly |
| `Render(int, int) string` | It renders the UI. The two arguments are width and height, they should be calculated by parent |
| `Destroy()` | It's called whenever Component is about to end it's lifecycle. Please note that it's parent's responsibility to call `Destroy()` |

Let's get to work!

### The `/input` page

`/pages/input/input.go`

```go
type Component struct {
    reactea.BasicComponent                // It implements all reactea's core functionalities

    // Props
    SetText func(string)

    textinput textinput.Model             // Input for inputting name
}

func New() *Component {
    return &Component{textinput: textinput.New()}
}

func (c *Component) Init() tea.Cmd {
    return c.textinput.Focus()
}

func (c *Component) Update(msg tea.Msg) tea.Cmd {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        if msg.Type == tea.KeyEnter {
            // Lifted state power! Woohooo
            c.SetText(c.textinput.Value())

            // Navigate to displayname, please
            reactea.SetRoute("/displayname")
            return nil
        }
    }

    var cmd tea.Cmd
    c.textinput, cmd = c.textinput.Update(msg)
    return cmd
}

// Here we are not using width and height, but you can!
func (c *Component) Render(int, int) string {
    return fmt.Sprintf("Enter your name: %s\nAnd press [ Enter ]", c.textinput.View())
}
```

#### The `/displayname` page

`/pages/displayname/displayname.go`

```go
import (
 "fmt"
)

// Our prop(s) is a string itself!
type Props = string

// Stateless components?!?!
func Render(text Props, width, height int) string {
    return fmt.Sprintf("OMG! Hello %s!", text)
}
```

### Main component

`/app/app.go`

```go
type Component struct {
    reactea.BasicComponent                // It implements all reactea's core functionalities

    mainRouter *router.Component

    text string // The name
}

func New() *Component {
    c := &Component{}

    // Does it remind you of something? react-router!
    c.mainRouter = router.NewWithRoutes(router.Routes{
        "default": func(router.Params) reactea.Component {
			component := input.New()

			component.SetText = c.setText

			return component
		},
		"/displayname": func(router.Params) reactea.Component {
            // RouteInitializer requires Component so we have to convert
            // Stateless component (renderer) to Component
			return reactea.Componentify(displayname.Render, c.text)
		},
    })

    return c
}

func (c *Component) Init() tea.Cmd {
    return c.mainRouter.Init()
}

func (c *Component) Update(msg tea.Msg) tea.Cmd {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        // ctrl+c support 
        if msg.String() == "ctrl+c" {
            return reactea.Destroy
        }
    }

    return c.mainRouter.Update(msg)
}

func (c *Component) Render(width, height int) string {
    return c.mainRouter.Render(width, height)
}

func (c *Component) setText(text string) {
    c.text = text
}
```

#### Main

`main.go`

```go
// reactea.NewProgram initializes program with
// "translation layer", so Reactea components work
program := reactea.NewProgram(app.New())

if _, err := program.Run(); err != nil {
    panic(err)
}
```

## Component lifecycle

![Component lifecycle image](.github/lifecycle-diagram.png)

Reactea component lifecycle consists of 4 methods (while Bubbletea only 3)
|Method|Purpose|
|-|-|
| `Init() tea.Cmd` | It's called first. All critical stuff should happen here. It also supports IO through tea.Cmd |
| `Update(tea.Msg) tea.Cmd` | It reacts to Bubbletea IO and updates state accordingly |
| `Render(int, int) string` | It renders the UI. The two arguments are width and height, they should be calculated by parent |
| `Destroy()` | It's called whenever Component is about to end it's lifecycle. Please note that it's parent's responsibility to call `Destroy()` |

Reactea takes pointer approach for components making state modifiable in any lifecycle method\

### Notes

`Update()` **IS NOT** guaranteed to be called on first-run, `Init()` for most part is, and critical logic should be there

## Decorating the view

Bubbletea v2 moved the alt-screen, the cursor, the window title, mouse mode and
the terminal colors out of commands and options into the `tea.View` the root
model returns. Components still render to a string, so a component that needs
one of those implements `ViewDecorator`

```go
func (c *Component) DecorateView(view *tea.View) {
    view.AltScreen = true
    view.Cursor = tea.NewCursor(c.cursorX, c.cursorY)
}
```

Only the root component is asked directly. A composite passes the view on to the
children it renders with `reactea.DecorateView(child, view)`; `router.Component`
and `modal.Controller` already do. Cursor positions are relative to the child's
own render, so a parent that draws a child at an offset applies
`reactea.TranslateCursor(view, dx, dy)` after collecting it.

`Reactify` forwards the wrapped model's cursor, and nothing else from its view.

## Wrapping Bubbletea models and bubbles widgets

The two need different adapters, because a bubbles widget is not a `tea.Model`
and never has been — not in v1 either. A widget's `Update` returns its own
concrete type (`func (m Model) Update(tea.Msg) (Model, tea.Cmd)`) so that you can
write `m.input, cmd = m.input.Update(msg)` without a type assertion, and Go has
no covariant returns. In v2 the `View() string` signature is a second mismatch.

|                     | Wraps                          | Constraint            |
|---------------------|--------------------------------|-----------------------|
| `Reactify`          | a self-contained Bubbletea model | `tea.Model`         |
| `ReactifyWidget`    | a bubbles widget                 | `Widget[T]`         |

```go
input := textinput.New()
input.SetVirtualCursor(false)
input.Focus()

component := reactea.ReactifyWidget(input) // reactea.Component
```

`ReactifyWidget` stores the widget value back after every `Update`, calls the
widget's `Init()` when it has one, and reports its cursor through
`DecorateView`. Widgets draw a virtual cursor into their string by default and
report no real cursor in that mode; reactea leaves that choice to you.

## Stateless components

Stateless components are represented by following function types

|                | Renderer[TProps any]     | ProplessRenderer   | DumbRenderer  |
|----------------|:------------------------:|:------------------:|:-------------:|
| **Properties** | ✅                       | ❌                | ❌            |
| **Dimensions** | ✅                       | ✅                | ❌            |
| **Arguments** | `TProps, int, int`        | `int, int`         | ❌            |

There are many utility functions for transforming stateless into stateful components or for rendering any component without knowing its type (`reactea.RenderAny`)

## Routes API

Routes API allows developers for easy development of multi-page apps.
They are kind of substitute for window.Location inside Bubbletea

### reactea.CurrentRoute() Route

Returns current route

### reactea.LastRoute() Route

Returns last route

### reactea.WasRouteChanged() bool

returns `LastRoute() != CurrentRoute()`

## Reactea Routes now support params

Params have been introduced in order to allow routes like: `/teams/123/player/4`

Params have to follow regex `^:.*$`\
`^` being beginning of current path level (`/^level/`)\
`$`being end of current path level (`/level$/`)

Note that params support wildcards with single `:`, like `/teams/:/player`. `/teams/123/player`, `/teams/456/player` etc will be matched no matter what and param will be ignored in param map.

## Router Component

`router.Component` is a basic router. Give it routes with `router.NewWithRoutes(router.Routes{...})` or the `Routes` field on `New()`.

It matches route placeholders, including the params and wildcards described above. When more than one placeholder matches, the most specific wins: literal segments beat params, which beat catch-alls. Relative navigation is a separate concern, handled by `reactea.Navigate`.

### router.Routes

`router.Routes` is `map[string]RouteInitializer` keyed by route placeholder.

`RouteInitializer` is a `func(router.Params) reactea.Component` that builds the component for a matched route.
