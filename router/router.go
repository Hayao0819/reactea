package router

import (
	"fmt"
	"strings"

	"github.com/Hayao0819/reactea"
	tea "github.com/charmbracelet/bubbletea"
)

type Params = map[string]string
type RouteInitializer func(Params) reactea.Component
type Routes = map[string]RouteInitializer

type Component struct {
	reactea.BasicComponent

	Routes Routes

	currentComponent reactea.Component
}

func New() *Component {
	return &Component{}
}

func NewWithRoutes(routes Routes) *Component {
	return &Component{Routes: routes}
}

func (c *Component) Init() tea.Cmd {
	return c.initRoute()
}

func (c *Component) Update(msg tea.Msg) tea.Cmd {
	var initCmd, updateCmd tea.Cmd

	switch msg.(type) {
	case reactea.RouteUpdatedMsg:
		if c.currentComponent != nil {
			c.currentComponent.Destroy()
		}

		initCmd = c.initRoute()
	}

	if c.currentComponent != nil {
		updateCmd = c.currentComponent.Update(msg)
	}

	return tea.Batch(initCmd, updateCmd)
}

// Destroy tears down the routed component. Without it the router would inherit
// BasicComponent's no-op and the current page would never be destroyed when the
// app itself is torn down.
func (c *Component) Destroy() {
	if c.currentComponent != nil {
		c.currentComponent.Destroy()
		c.currentComponent = nil
	}
}

func (c *Component) Render(width, height int) string {
	if c.currentComponent != nil {
		return c.currentComponent.Render(width, height)
	}

	return fmt.Sprintf("Couldn't route for \"%s\"", reactea.CurrentRoute())
}

func (c *Component) initRoute() tea.Cmd {
	// Reset first so a failed (re-)route leaves currentComponent nil instead
	// of a stale, already-Destroyed component. initRoute is only ever called
	// after any previous component has been Destroyed (see Update), so this is
	// safe.
	c.currentComponent = nil

	if initializer, params, ok := c.findMatchingRouteInitializer(); ok {
		c.currentComponent = initializer(params)
		return c.currentComponent.Init()
	}

	if initializer, ok := c.Routes["default"]; ok {
		c.currentComponent = initializer(nil)
		return c.currentComponent.Init()
	}

	return nil
}

func (c *Component) findMatchingRouteInitializer() (RouteInitializer, Params, bool) {
	currentRoute := reactea.CurrentRoute()

	// Go randomizes map iteration order, so when more than one placeholder
	// matches the current route we must not just return the first hit — that
	// would pick a route at random. Instead choose the most specific match
	// deterministically (literal segments beat params beat wildcards; ties are
	// broken by placeholder string). "default" is handled as a fallback in
	// initRoute and is never treated as a match here.
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

		params, ok := reactea.RouteMatchesPlaceholder(currentRoute, placeholder)
		if !ok {
			continue
		}

		if !found || moreSpecific(placeholder, bestPlaceholder) {
			bestPlaceholder = placeholder
			bestInit = initializer
			bestParams = params
			found = true
		}
	}

	return bestInit, bestParams, found
}

// moreSpecific reports whether route placeholder a is a strictly better match
// than b. A placeholder is scored segment by segment (left to right): a literal
// segment is more specific than a single-level param (":"), which beats an
// optional param ("?:"), which beats a catch-all ("+?:"). More segments win
// when one is a prefix of the other. Equal specificity is broken by string
// order so selection is always deterministic.
func moreSpecific(a, b string) bool {
	switch compareSpecificity(routeSpecificity(a), routeSpecificity(b)) {
	case 1:
		return true
	case -1:
		return false
	default:
		return a < b
	}
}

func routeSpecificity(placeholder string) []int {
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
