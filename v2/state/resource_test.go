package state_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/Hayao0819/reactea/v2"
	"github.com/Hayao0819/reactea/v2/state"
	"github.com/Hayao0819/reactea/v2/testkit"
)

func fetched[T any](value T, err error) func(context.Context) (T, error) {
	return func(context.Context) (T, error) { return value, err }
}

type greeting struct {
	reactea.BasicComponent

	name state.Resource[string]
}

func (g *greeting) Init(ctx *reactea.Ctx) tea.Cmd {
	return g.name.Load(ctx, fetched("world", nil))
}

func (g *greeting) Update(_ *reactea.Ctx, msg tea.Msg) tea.Cmd {
	g.name.Handle(msg)

	return nil
}

func (g *greeting) Render(*reactea.Ctx) string {
	if g.name.Loading() {
		return "loading"
	}

	return "hello " + g.name.Value()
}

func TestALoadedValueReachesTheComponent(t *testing.T) {
	app := reactea.New(&greeting{}, reactea.WithSize(20, 1))
	app.Start()

	if got := testkit.Plain(app); !strings.Contains(got, "hello world") {
		t.Errorf("after loading = %q", got)
	}
}

func TestOnlyTheLatestLoadLands(t *testing.T) {
	app := reactea.New(reactea.Text(""))

	var (
		resource state.Resource[string]
		first    context.Context
	)

	older := resource.Load(app.Ctx(), func(ctx context.Context) (string, error) {
		first = ctx

		return "older", nil
	})
	newer := resource.Load(app.Ctx(), fetched("newer", nil))

	if !resource.Handle(newer()) || !resource.Handle(older()) {
		t.Fatal("Handle refused its own results")
	}

	if got := resource.Value(); got != "newer" {
		t.Errorf("Value = %q, want the newer load", got)
	}

	if first.Err() == nil {
		t.Error("the superseded load was not cancelled")
	}

	if resource.Loading() {
		t.Error("still loading after the latest load landed")
	}
}

func TestAFailedLoadKeepsTheLastValue(t *testing.T) {
	app := reactea.New(reactea.Text(""))
	failure := errors.New("offline")

	var resource state.Resource[int]

	resource.Handle(resource.Load(app.Ctx(), fetched(7, nil))())
	resource.Handle(resource.Load(app.Ctx(), fetched(0, failure))())

	if resource.Value() != 7 || !resource.Loaded() {
		t.Errorf("Value = %d, Loaded = %v after a failure", resource.Value(), resource.Loaded())
	}

	if !errors.Is(resource.Err(), failure) {
		t.Errorf("Err = %v", resource.Err())
	}

	resource.Handle(resource.Load(app.Ctx(), fetched(8, nil))())

	if resource.Err() != nil {
		t.Errorf("Err = %v after a success", resource.Err())
	}
}

func TestClosingTheScopeCancelsTheLoad(t *testing.T) {
	app := reactea.New(reactea.Text(""))
	scope := app.Scope().Child()

	var resource state.Resource[string]

	load := resource.Load(app.Ctx().WithScope(scope), func(ctx context.Context) (string, error) {
		scope.Close()

		return "", ctx.Err()
	})

	resource.Handle(load())

	if !errors.Is(resource.Err(), context.Canceled) {
		t.Errorf("Err = %v, want context.Canceled", resource.Err())
	}
}

func TestResourcesOfOneTypeKeepTheirOwnResults(t *testing.T) {
	app := reactea.New(reactea.Text(""))

	var first, second state.Resource[string]

	msg := first.Load(app.Ctx(), fetched("first", nil))()
	second.Load(app.Ctx(), fetched("second", nil))

	if second.Handle(msg) {
		t.Error("a resource took another resource's result")
	}

	if !first.Handle(msg) || first.Value() != "first" {
		t.Errorf("first = %q", first.Value())
	}
}
