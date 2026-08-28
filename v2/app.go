package reactea

import (
	tea "charm.land/bubbletea/v2"
)

// RouteChangedMsg is delivered to the whole tree after the route moves.
type RouteChangedMsg struct {
	From string
	To   string
}

// A message rather than a direct mutation, so a route change is safe from any
// goroutine.
type routeRequestMsg struct {
	target   string
	relative bool
}

// Bubbletea returns on QuitMsg before Update sees it, so this filter hop is what
// closes the root scope.
type teardownMsg struct{ next tea.Msg }

// App owns a running program's state. It lives here rather than in package
// variables, so two apps in one process never share a route. Commands go
// straight back to Bubbletea, which already runs them under a panic guard.
type App struct {
	root    Component
	program *tea.Program
	scope   *Scope

	route         string
	previousRoute string

	// terminal persists across frames; cursor is per frame.
	terminal tea.View
	cursor   *tea.Cursor

	captures int
	tornDown bool

	width, height int
}

// Option configures an App.
type Option func(*App)

// WithRoute starts the app on route instead of "/".
func WithRoute(route string) Option {
	return func(a *App) {
		a.route, a.previousRoute = route, route
	}
}

// WithSize sets the size the app assumes until the terminal reports its own.
func WithSize(width, height int) Option {
	return func(a *App) {
		a.width, a.height = width, height
	}
}

// WithTerminal seeds terminal state before the first frame. Bubbletea renders
// once before running Init's commands, so asking with a command alone flashes a
// frame onto the primary screen.
func WithTerminal(apply func(*tea.View)) Option {
	return func(a *App) { apply(&a.terminal) }
}

// WithAltScreen starts in the alternate screen buffer.
func WithAltScreen() Option {
	return WithTerminal(func(view *tea.View) { view.AltScreen = true })
}

// WithWindowTitle sets the terminal window title.
func WithWindowTitle(title string) Option {
	return WithTerminal(func(view *tea.View) { view.WindowTitle = title })
}

// New builds an App around root.
func New(root Component, options ...Option) *App {
	app := &App{
		root:          root,
		scope:         NewScope(),
		route:         "/",
		previousRoute: "/",
	}

	for _, option := range options {
		option(app)
	}

	return app
}

// Program wraps the app in a Bubbletea program. Any quit closes the root scope
// first.
func (a *App) Program(options ...tea.ProgramOption) *tea.Program {
	options = append(options, tea.WithFilter(a.filter))

	a.program = tea.NewProgram(a, options...)

	return a.program
}

// Run builds a program and runs it.
func (a *App) Run(options ...tea.ProgramOption) error {
	_, err := a.Program(options...).Run()

	return err
}

// Route is the app's current route.
func (a *App) Route() string { return a.route }

// InputCaptured reports whether something below has claimed the keys.
func (a *App) InputCaptured() bool { return a.captures > 0 }

// Scope is the app's root scope. It closes when the program ends.
func (a *App) Scope() *Scope { return a.scope }

// Ctx is the root context: the whole screen, bound to the root scope.
func (a *App) Ctx() *Ctx {
	return &Ctx{app: a, scope: a.scope, width: a.width, height: a.height, focused: true}
}

func (a *App) setRoute(target string) (RouteChangedMsg, bool) {
	if target == a.route {
		return RouteChangedMsg{}, false
	}

	changed := RouteChangedMsg{From: a.route, To: target}

	a.previousRoute, a.route = a.route, target

	return changed, true
}

// filter runs before the event loop inspects a message, the only place a quit
// can be intercepted.
func (a *App) filter(_ tea.Model, msg tea.Msg) tea.Msg {
	if a.tornDown {
		return msg
	}

	switch msg.(type) {
	case tea.QuitMsg, tea.InterruptMsg:
		return teardownMsg{next: msg}
	}

	return msg
}

func (a *App) Init() tea.Cmd {
	defer a.closeOnPanic()

	return a.root.Init(a.Ctx())
}

// Bubbletea's only recover is around Run, which never reaches the model, so
// without this a panicking component strands every cleanup in the tree.
func (a *App) closeOnPanic() {
	if r := recover(); r != nil {
		a.scope.Close()
		a.tornDown = true

		panic(r)
	}
}

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	defer a.closeOnPanic()

	switch msg := msg.(type) {
	case teardownMsg:
		a.scope.Close()
		a.tornDown = true

		next := msg.next

		return a, func() tea.Msg { return next }

	case terminalMsg:
		msg.apply(&a.terminal)

		return a, nil

	case captureMsg:
		a.captures = max(0, a.captures+msg.delta)

		if msg.held != nil {
			msg.held.held = true
		}

		return a, nil

	case tea.WindowSizeMsg:
		a.width, a.height = msg.Width, msg.Height

	case routeRequestMsg:
		return a, a.applyRoute(msg)
	}

	return a, a.root.Update(a.Ctx(), msg)
}

func (a *App) applyRoute(request routeRequestMsg) tea.Cmd {
	target := request.target
	if request.relative {
		target = Resolve(a.route, target)
	}

	if target == "" || target[0] != '/' {
		return nil
	}

	changed, moved := a.setRoute(target)
	if !moved {
		return nil
	}

	return func() tea.Msg { return changed }
}

func (a *App) View() tea.View {
	defer a.closeOnPanic()

	a.cursor = nil

	content := a.root.Render(a.Ctx())

	view := a.terminal
	view.Cursor = a.cursor
	view.SetContent(content)

	return view
}
