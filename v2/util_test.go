package reactea

import (
	"fmt"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// counterModel is a value-receiver tea.Model (like bubbles widgets): its Update
// returns a NEW model value rather than mutating the receiver.
type counterModel struct {
	n int
}

func (m counterModel) Init() tea.Cmd { return nil }

func (m counterModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if _, ok := msg.(tea.KeyMsg); ok {
		m.n++
	}

	return m, nil
}

func (m counterModel) View() tea.View { return tea.NewView(fmt.Sprintf("count:%d", m.n)) }

// Reactified must satisfy the Component interface.
var _ Component = (*Reactified[counterModel])(nil)

// Because Renderer[TProps] is a type alias, a Renderer-typed value satisfies
// the AnyRenderer constraint directly — no explicit conversion. This wouldn't
// compile if Renderer were a distinct defined type.
func TestRendererAliasNeedsNoCast(t *testing.T) {
	var r Renderer[string] = func(s string, width, height int) string {
		return s
	}

	if got := RenderAny(r, "hi", 0, 0); got != "hi" {
		t.Errorf("RenderAny: expected \"hi\", got %q", got)
	}

	if got := Componentify(r, "hi").Render(0, 0); got != "hi" {
		t.Errorf("Componentify: expected \"hi\", got %q", got)
	}

	if got := PropfulToLess(r, "hi")(0, 0); got != "hi" {
		t.Errorf("PropfulToLess: expected \"hi\", got %q", got)
	}
}

func TestReactify(t *testing.T) {
	c := Reactify(counterModel{})

	key := tea.KeyPressMsg{Code: 'a', Text: "a"}
	c.Update(key)
	c.Update(key)

	// Without storing the model Update returns back into c.Model, a
	// value-receiver widget would reset to zero every Update.
	if c.Model.n != 2 {
		t.Errorf("expected wrapped model state to persist, got n=%d", c.Model.n)
	}

	if result := c.Render(0, 0); result != "count:2" {
		t.Errorf("expected Render to delegate to View, got %q", result)
	}
}

func TestRenderAny(t *testing.T) {
	t.Run("renderer", func(t *testing.T) {
		renderer := func(struct{}, int, int) string {
			return "working"
		}

		if result := RenderAny(renderer, struct{}{}, 1, 1); result != "working" {
			t.Errorf("invalid result, expected \"working\", got \"%s\"", result)
		}
	})

	t.Run("proplessRenderer", func(t *testing.T) {
		proplessRenderer := func(int, int) string {
			return "working"
		}

		if result := RenderAny(proplessRenderer, struct{}{}, 1, 1); result != "working" {
			t.Errorf("invalid result, expected \"working\", got \"%s\"", result)
		}
	})

	t.Run("dumbRenderer", func(t *testing.T) {
		dumbRenderer := func() string {
			return "working"
		}

		if result := RenderAny(dumbRenderer, struct{}{}, 1, 1); result != "working" {
			t.Errorf("invalid result, expected \"working\", got \"%s\"", result)
		}
	})
}

func TestPropfulToLess(t *testing.T) {
	renderer := func(struct{}, int, int) string {
		return "working"
	}

	proplessRenderer := PropfulToLess(renderer, struct{}{})

	if result := proplessRenderer(1, 1); result != "working" {
		t.Errorf("wrapped value doesn't render correctly, expected \"working\", got \"%s\"", result)
	}
}

func TestComponentify(t *testing.T) {
	t.Run("renderer", func(t *testing.T) {
		renderer := func(struct{}, int, int) string {
			return "working"
		}

		if result := Componentify(renderer, struct{}{}).Render(1, 1); result != "working" {
			t.Errorf("transformed value doesn't render correctly, expected \"working\", got \"%s\"", result)
		}
	})

	t.Run("proplessRenderer", func(t *testing.T) {
		proplessRenderer := func(int, int) string {
			return "working"
		}

		if result := Componentify(proplessRenderer, struct{}{}).Render(1, 1); result != "working" {
			t.Errorf("transformed value doesn't render correctly, expected \"working\", got \"%s\"", result)
		}
	})

	t.Run("dumbRenderer", func(t *testing.T) {
		dumbRenderer := func() string {
			return "working"
		}

		if result := Componentify(dumbRenderer, struct{}{}).Render(1, 1); result != "working" {
			t.Errorf("transformed value doesn't render correctly, expected \"working\", got \"%s\"", result)
		}
	})
}
