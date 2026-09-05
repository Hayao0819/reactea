package reactea

import (
	"sync"

	tea "charm.land/bubbletea/v2"
)

// RouteChangedMsg is delivered to the whole tree after the route moves.
type RouteChangedMsg struct {
	From string
	To   string
}

// routeRequestMsg carries a route mutation safely from any goroutine.
type routeRequestMsg struct {
	target   string
	relative bool
}

// Bubble Tea returns on QuitMsg before Update sees it, so this filter hop is what
// closes the root scope.
type teardownMsg struct{ next tea.Msg }

// App owns a running program's state. Each instance has isolated route,
// terminal and input-capture state.
type App struct {
	root  Component
	scope *Scope

	route         string
	previousRoute string

	// terminal persists across frames; cursor is per frame.
	terminal tea.View
	cursor   *tea.Cursor

	tornDown bool

	// Scope cleanup can release a claim concurrently with the event loop.
	capturesMu sync.Mutex
	captures   int

	reverseBatches bool

	width, height int
}

// Option configures an App.
type Option func(*App)

// WithRoute sets the app's initial route; the default is "/".
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

// WithTerminal seeds terminal state before the first frame. Bubble Tea renders
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

// Program wraps the app in a Bubble Tea program. QuitMsg and InterruptMsg close
// the root scope first.
func (a *App) Program(options ...tea.ProgramOption) *tea.Program {
	options = append(options, tea.WithFilter(a.filter))

	return tea.NewProgram(a, options...)
}

// sendRounds bounds command processing for one Send.
const sendRounds = 100

// Send delivers msg and recursively runs commands until the queue settles, so a
// test sees the state a user would after the same event.
//
// Commands run inline: one that sleeps makes Send wait for it, so keep test
// intervals short. A command that keeps rescheduling itself panics after a
// hundred rounds.
func (a *App) Send(msgs ...tea.Msg) {
	pending := make([]tea.Cmd, 0, len(msgs))

	for _, msg := range msgs {
		_, cmd := a.Update(msg)
		pending = append(pending, cmd)
	}

	a.drain(pending...)
}

// ReverseBatches makes Send run the commands inside a tea.Batch back to front,
// allowing tests to exercise both orders permitted by Bubble Tea.
func (a *App) ReverseBatches(on bool) { a.reverseBatches = on }

func (a *App) drain(pending ...tea.Cmd) {
	for round := 0; len(pending) > 0; round++ {
		if round == sendRounds {
			panic("reactea: command chain did not settle")
		}

		current := pending
		pending = nil

		for _, cmd := range current {
			if cmd == nil {
				continue
			}

			produced := cmd()

			if batch, ok := produced.(tea.BatchMsg); ok {
				pending = append(pending, a.order(batch)...)

				continue
			}

			if produced == nil {
				continue
			}

			_, next := a.Update(produced)
			pending = append(pending, next)
		}
	}
}

func (a *App) order(batch tea.BatchMsg) []tea.Cmd {
	if !a.reverseBatches {
		return batch
	}

	reversed := make([]tea.Cmd, 0, len(batch))
	for i := len(batch) - 1; i >= 0; i-- {
		reversed = append(reversed, batch[i])
	}

	return reversed
}

// Start runs Init and its command chain as a program would before its first
// frame. Tests call it to reproduce that lifecycle.
func (a *App) Start() {
	a.drain(a.Init())
}

// Run builds a program and runs it.
func (a *App) Run(options ...tea.ProgramOption) error {
	_, err := a.Program(options...).Run()

	return err
}

// Route is the app's current route.
func (a *App) Route() string { return a.route }

// InputCaptured reports whether something below has claimed the keys.
func (a *App) InputCaptured() bool {
	a.capturesMu.Lock()
	defer a.capturesMu.Unlock()

	return a.captures > 0
}

func (a *App) addCapture(delta int) {
	a.capturesMu.Lock()
	defer a.capturesMu.Unlock()

	a.captures = max(0, a.captures+delta)
}

// Scope is the app's root scope. QuitMsg and InterruptMsg close it.
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

// filter intercepts quit messages before the event loop handles them.
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

// closeOnPanic closes component scopes before propagating a panic to Bubble
// Tea's runner.
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
		target = resolveRoute(a.route, target)
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
