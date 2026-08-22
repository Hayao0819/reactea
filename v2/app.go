package reactea

import (
	tea "charm.land/bubbletea/v2"
)

// RouteChangedMsg is delivered to the whole tree after the route moves.
type RouteChangedMsg struct {
	From string
	To   string
}

// routeRequestMsg asks the app to move. It is a message rather than a direct
// mutation so a route change is safe to issue from anywhere, including a
// command goroutine.
type routeRequestMsg struct {
	target   string
	relative bool
}

// teardownMsg is how a quit is turned into a teardown. Bubbletea's event loop
// returns on QuitMsg before Update ever sees it, so without this filter hop the
// root scope would never close.
type teardownMsg struct{ next tea.Msg }

// App owns everything a running program needs. Route and terminal state live
// here rather than in package variables, so two apps in one process — or two
// tests in parallel — never see each other's.
//
// App implements tea.Model; commands are handed straight back to Bubbletea,
// which already runs them under its own panic guard.
type App struct {
	root    Component
	program *tea.Program
	scope   *Scope

	route         string
	previousRoute string

	// terminal holds what the program asked the terminal for; it persists across
	// frames. cursor is per frame and is cleared before every Render.
	terminal tea.View
	cursor   *tea.Cursor

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
// Tests use it to render without a terminal.
func WithSize(width, height int) Option {
	return func(a *App) {
		a.width, a.height = width, height
	}
}

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

// Program wraps the app in a Bubbletea program. Quitting through tea.Quit,
// Ctrl+C or a signal all close the root scope first.
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

// Scope is the root scope. It closes when the program ends.
func (a *App) Scope() *Scope { return a.scope }

// Ctx is the root context: the whole screen, bound to the root scope. Tests use
// it to drive a component without a terminal.
func (a *App) Ctx() *Ctx {
	return &Ctx{app: a, scope: a.scope, width: a.width, height: a.height}
}

func (a *App) setRoute(target string) (RouteChangedMsg, bool) {
	if target == a.route {
		return RouteChangedMsg{}, false
	}

	changed := RouteChangedMsg{From: a.route, To: target}

	a.previousRoute, a.route = a.route, target

	return changed, true
}

// filter runs before Bubbletea's event loop inspects a message, which is the
// only place a quit can be intercepted.
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
	return a.root.Init(a.Ctx())
}

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case teardownMsg:
		a.scope.Close()
		a.tornDown = true

		next := msg.next

		return a, func() tea.Msg { return next }

	case terminalMsg:
		msg.apply(&a.terminal)

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

	// The change is delivered like any other message, so a component observes it
	// where it observes everything else.
	return func() tea.Msg { return changed }
}

func (a *App) View() tea.View {
	a.cursor = nil

	content := a.root.Render(a.Ctx())

	view := a.terminal
	view.Cursor = a.cursor
	view.SetContent(content)

	return view
}
