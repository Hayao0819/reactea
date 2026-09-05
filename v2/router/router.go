// Package router picks a child component from the app's current route.
package router

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/Hayao0819/reactea/v2"
)

// RouteInitializer builds the component for one matched route.
type RouteInitializer func(Params) reactea.Component

// Routes maps patterns to component constructors. "default" is the fallback.
type Routes = map[string]RouteInitializer

// ReloadMode says what SetRoutes does about the page already on screen.
type ReloadMode int

const (
	// PreserveCurrent leaves a mounted page alone while its route still resolves
	// to the same entry, which is what a table that only ever grows wants.
	PreserveCurrent ReloadMode = iota

	// RemountCurrent tears the page down and builds it again from the new table.
	RemountCurrent
)

// Component renders whichever route matches, remounting when the match changes.
type Component struct {
	// NotFound renders when nothing matched and there is no "default" route.
	NotFound reactea.RenderFunc

	routes Routes

	current     reactea.Component
	scope       *reactea.Scope
	placeholder string
	params      Params
}

// New builds an empty router whose routes can be supplied with SetRoutes.
func New() *Component { return &Component{} }

// NewWithRoutes builds a router from routes.
func NewWithRoutes(routes Routes) *Component { return &Component{routes: routes} }

// SetRoutes replaces the table. It takes no Ctx because a table is often built
// before anything is mounted; RemountCurrent takes effect on the next Update.
func (c *Component) SetRoutes(routes Routes, mode ReloadMode) {
	c.routes = routes

	if mode == RemountCurrent {
		c.Unmount()
	}
}

// Reload rebuilds the page in this update, so the frame Bubble Tea draws next
// has one. Unmounting alone would leave the router empty until a message
// arrived.
func (c *Component) Reload(ctx *reactea.Ctx) tea.Cmd {
	c.Unmount()

	return c.sync(ctx)
}

// Current is the routed component, or nil when nothing matched.
func (c *Component) Current() reactea.Component { return c.current }

// The focus methods reach the routed page, so Tab can move among the panes
// inside it rather than stopping at the router.
func (c *Component) FocusNext() bool { return reactea.FocusOf(c.current).FocusNext() }

func (c *Component) FocusPrev() bool { return reactea.FocusOf(c.current).FocusPrev() }

func (c *Component) FocusFirst() { reactea.FocusOf(c.current).FocusFirst() }

func (c *Component) FocusLast() { reactea.FocusOf(c.current).FocusLast() }

func (c *Component) HasFocusable() bool { return reactea.FocusOf(c.current).HasFocusable() }

func (c *Component) Init(ctx *reactea.Ctx) tea.Cmd {
	return c.sync(ctx)
}

// Unmount closes the page's scope. The router does this on every route change;
// an enclosing component does not have to.
func (c *Component) Unmount() {
	if c.scope != nil {
		c.scope.Close()
		c.scope = nil
	}

	c.current, c.placeholder, c.params = nil, "", Params{}
}

func (c *Component) Update(ctx *reactea.Ctx, msg tea.Msg) tea.Cmd {
	var initCmd tea.Cmd

	_, routed := msg.(reactea.RouteChangedMsg)
	if routed || c.current == nil {
		initCmd = c.sync(ctx)
	}

	if c.current == nil {
		return initCmd
	}

	return tea.Batch(initCmd, c.current.Update(ctx.WithScope(c.scope), msg))
}

func (c *Component) Render(ctx *reactea.Ctx) string {
	// Mounting here would throw away the page's Init command, so a router that
	// has never been through Init or Update renders not-found until it has.
	if c.current != nil {
		return c.current.Render(ctx.WithScope(c.scope))
	}

	if c.NotFound != nil {
		return c.NotFound(ctx)
	}

	return fmt.Sprintf("Couldn't route for %q", ctx.Route())
}

// A route change resolving to the same page leaves it mounted, so a nested
// router does not tear down its parent's page.
func (c *Component) sync(ctx *reactea.Ctx) tea.Cmd {
	placeholder, initializer, params, ok := c.resolve(ctx.Route())

	if c.current != nil && placeholder == c.placeholder && sameParams(params, c.params, placeholder) {
		return nil
	}

	c.Unmount()

	if !ok {
		return nil
	}

	c.placeholder, c.params = placeholder, params

	// Its own scope, so the next route change tears down this page and nothing
	// else.
	c.scope = ctx.Scope().Child()
	c.current = initializer(params)

	return c.current.Init(ctx.WithScope(c.scope))
}

func (c *Component) resolve(route string) (string, RouteInitializer, Params, bool) {
	if placeholder, initializer, params, ok := c.match(route); ok {
		return placeholder, initializer, params, true
	}

	if initializer, ok := c.routes["default"]; ok {
		return "default", initializer, Params{path: route}, true
	}

	return "", nil, Params{}, false
}

// sameParams ignores a trailing catch-all because it belongs to the nested page.
func sameParams(a, b Params, placeholder string) bool {
	if placeholder == "default" {
		return a.path == b.path
	}

	owned := func(params Params) map[string]string {
		mine := make(map[string]string, len(params.values))

		for key, value := range params.values {
			if key != catchAllName(placeholder) {
				mine[key] = value
			}
		}

		return mine
	}

	first, second := owned(a), owned(b)

	if len(first) != len(second) {
		return false
	}

	for key, value := range first {
		if second[key] != value {
			return false
		}
	}

	return true
}

func catchAllName(placeholder string) string {
	levels := strings.Split(placeholder, "/")

	last := levels[len(levels)-1]
	if !strings.HasPrefix(last, "*") {
		return ""
	}

	return last[1:]
}

func (c *Component) match(route string) (string, RouteInitializer, Params, bool) {
	// Go randomises map iteration, so ties are broken by specificity then string
	// order to keep the choice stable across runs.
	var (
		bestPlaceholder string
		bestInit        RouteInitializer
		bestParams      Params
		found           bool
	)

	for placeholder, initializer := range c.routes {
		if placeholder == "default" {
			continue
		}

		params, ok := Match(route, placeholder)
		if !ok {
			continue
		}

		if !found || moreSpecific(placeholder, bestPlaceholder) {
			bestPlaceholder, bestInit, bestParams, found = placeholder, initializer, params, true
		}
	}

	return bestPlaceholder, bestInit, bestParams, found
}

func moreSpecific(a, b string) bool {
	switch compareSpecificity(specificity(a), specificity(b)) {
	case 1:
		return true
	case -1:
		return false
	default:
		return a < b
	}
}

func specificity(placeholder string) []int {
	levels := strings.Split(strings.TrimPrefix(placeholder, "/"), "/")
	score := make([]int, len(levels))

	for i, level := range levels {
		switch {
		case strings.HasPrefix(level, "*"):
			score[i] = 0
		case strings.HasPrefix(level, ":") && strings.HasSuffix(level, "?"):
			score[i] = 1
		case strings.HasPrefix(level, ":"):
			score[i] = 2
		default:
			score[i] = 3
		}
	}

	return score
}

func compareSpecificity(a, b []int) int {
	for i := 0; i < len(a) && i < len(b); i++ {
		if a[i] != b[i] {
			if a[i] > b[i] {
				return 1
			}

			return -1
		}
	}

	switch {
	case len(a) > len(b):
		return 1
	case len(a) < len(b):
		return -1
	default:
		return 0
	}
}

// Page adapts a constructor that takes no route params.
func Page[TComponent reactea.Component](construct func() TComponent) RouteInitializer {
	return func(Params) reactea.Component { return construct() }
}

// Param adapts a constructor that takes one route param by name.
func Param[TComponent reactea.Component](name string, construct func(string) TComponent) RouteInitializer {
	return func(params Params) reactea.Component { return construct(params.Get(name)) }
}
