// Package router picks a child component from the app's current route.
package router

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/Hayao0819/reactea/v2"
)

type Params = map[string]string

type RouteInitializer func(Params) reactea.Component

type Routes = map[string]RouteInitializer

// Component renders whichever route matches, re-initialising when the route
// changes and destroying the page it replaces.
type Component struct {
	Routes Routes

	// NotFound renders when nothing matched and there is no "default" route.
	NotFound reactea.RenderFunc

	current reactea.Component
	route   string
}

func New() *Component { return &Component{} }

func NewWithRoutes(routes Routes) *Component { return &Component{Routes: routes} }

// Current is the routed component, or nil when nothing matched.
func (c *Component) Current() reactea.Component { return c.current }

func (c *Component) Init(ctx *reactea.Ctx) tea.Cmd {
	return c.initRoute(ctx)
}

func (c *Component) Destroy() {
	if c.current != nil {
		c.current.Destroy()
		c.current = nil
	}
}

func (c *Component) Update(ctx *reactea.Ctx, msg tea.Msg) tea.Cmd {
	var initCmd tea.Cmd

	if _, ok := msg.(reactea.RouteChangedMsg); ok {
		c.Destroy()

		initCmd = c.initRoute(ctx)
	}

	if c.current == nil {
		return initCmd
	}

	return tea.Batch(initCmd, c.current.Update(ctx, msg))
}

func (c *Component) Render(ctx *reactea.Ctx) string {
	// Init may never have run — a router built after the app started, or one
	// mounted by a parent that skipped Init — so route lazily rather than
	// rendering the not-found page by accident.
	if c.current == nil && c.route != ctx.Route() {
		c.initRoute(ctx)
	}

	if c.current != nil {
		return c.current.Render(ctx)
	}

	if c.NotFound != nil {
		return c.NotFound(ctx)
	}

	return fmt.Sprintf("Couldn't route for %q", ctx.Route())
}

func (c *Component) initRoute(ctx *reactea.Ctx) tea.Cmd {
	c.current, c.route = nil, ctx.Route()

	if initializer, params, ok := c.match(ctx.Route()); ok {
		c.current = initializer(params)

		return c.current.Init(ctx)
	}

	if initializer, ok := c.Routes["default"]; ok {
		c.current = initializer(nil)

		return c.current.Init(ctx)
	}

	return nil
}

func (c *Component) match(route string) (RouteInitializer, Params, bool) {
	// Go randomises map iteration, so when more than one placeholder matches we
	// must not take the first hit. Rank by specificity instead, with a string
	// tie-break, so the choice is the same on every run.
	var (
		bestPlaceholder string
		bestInit        RouteInitializer
		bestParams      Params
		found           bool
	)

	for placeholder, initializer := range c.Routes {
		if placeholder == "default" {
			continue
		}

		params, ok := reactea.MatchRoute(route, placeholder)
		if !ok {
			continue
		}

		if !found || moreSpecific(placeholder, bestPlaceholder) {
			bestPlaceholder, bestInit, bestParams, found = placeholder, initializer, params, true
		}
	}

	return bestInit, bestParams, found
}

// moreSpecific reports whether placeholder a beats b. Segments are scored left
// to right: a literal beats a param (":"), which beats an optional ("?:"),
// which beats a catch-all ("+?:"). More segments win when one is a prefix of
// the other, and equal scores fall back to string order.
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
		case strings.HasPrefix(level, "+?:"):
			score[i] = 0
		case strings.HasPrefix(level, "?:"):
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
