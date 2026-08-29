package layout

import (
	"strconv"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/Hayao0819/reactea/v2"
)

type counted struct {
	reactea.BasicComponent

	renders    int
	updates    int
	generation int
}

func (c *counted) Update(*reactea.Ctx, tea.Msg) tea.Cmd {
	c.updates++

	return nil
}

func (c *counted) Render(*reactea.Ctx) string {
	c.renders++

	return strconv.Itoa(c.generation)
}

func TestMemoSkipsRenderUntilSomethingChanges(t *testing.T) {
	child := &counted{}
	memo := Memo(child, func() any { return child.generation })

	// A focused pane is never cached, so the focus sits on the other one.
	box := Column(Grow(1, &probe{label: "o"}).Focusable(), Grow(1, memo))
	app := reactea.New(box, reactea.WithSize(10, 3))

	for range 5 {
		app.View()
	}

	if child.renders != 1 {
		t.Errorf("child rendered %d times for an unchanged key", child.renders)
	}

	child.generation++

	if got := app.View().Content; !strings.Contains(got, "1") {
		t.Errorf("content = %q, want the new generation", got)
	}

	if child.renders != 2 {
		t.Errorf("child rendered %d times, want 2", child.renders)
	}
}

func TestMemoAlwaysUpdates(t *testing.T) {
	child := &counted{}
	memo := Memo(child, func() any { return child.generation })

	app := reactea.New(Column(Grow(1, memo).Focusable()), reactea.WithSize(10, 3))

	app.View()
	app.Update(tea.KeyPressMsg{Code: 'x', Text: "x"})

	if child.updates != 1 {
		t.Errorf("Update was skipped: %d", child.updates)
	}
}

func TestMemoRedrawsOnResize(t *testing.T) {
	child := &counted{}
	memo := Memo(child, func() any { return child.generation })

	box := Row(Grow(1, &probe{label: "o"}).Focusable(), Grow(1, memo))

	app := reactea.New(box, reactea.WithSize(20, 2))

	app.View()

	if child.renders != 1 {
		t.Fatalf("renders = %d", child.renders)
	}

	app.Update(tea.WindowSizeMsg{Width: 40, Height: 2})
	app.View()

	if child.renders != 2 {
		t.Errorf("a resize did not invalidate the cache: %d", child.renders)
	}
}

func TestMemoInvalidate(t *testing.T) {
	child := &counted{}
	memo := Memo(child, func() any { return 0 })

	box := Column(Grow(1, &probe{label: "o"}).Focusable(), Grow(1, memo))
	app := reactea.New(box, reactea.WithSize(10, 3))

	app.View()
	app.View()

	if child.renders != 1 {
		t.Fatalf("renders = %d", child.renders)
	}

	memo.Invalidate()
	app.View()

	if child.renders != 2 {
		t.Errorf("Invalidate did not drop the cache: %d", child.renders)
	}
}

type cursorChild struct {
	reactea.BasicComponent

	renders int
}

func (c *cursorChild) Render(ctx *reactea.Ctx) string {
	c.renders++
	ctx.CursorAt(1, 0)

	return "x"
}

func TestMemoKeepsTheCursorOfAFocusedChild(t *testing.T) {
	child := &cursorChild{}

	app := reactea.New(
		Column(Grow(1, Memo(child, func() any { return 0 })).Focusable()),
		reactea.WithSize(10, 2),
	)

	for frame := range 3 {
		if app.View().Cursor == nil {
			t.Fatalf("frame %d: the focused child lost its cursor", frame)
		}
	}
}

func TestMemoStillCachesAnUnfocusedChild(t *testing.T) {
	child := &counted{}

	box := Column(
		Grow(1, Memo(child, func() any { return 0 })).Focusable(),
		Grow(1, &probe{label: "o"}).Focusable(),
	)

	app := reactea.New(box, reactea.WithSize(10, 2))

	box.FocusNext()

	for range 3 {
		app.View()
	}

	if child.renders != 1 {
		t.Errorf("an unfocused child rendered %d times", child.renders)
	}
}

func TestMemoTreatsAnUncomparableKeyAsAMiss(t *testing.T) {
	child := &counted{}

	app := reactea.New(
		Column(Grow(1, Memo(child, func() any { return []int{1} })).Focusable()),
		reactea.WithSize(10, 2),
	)

	app.View()
	app.View()

	if child.renders != 2 {
		t.Errorf("renders = %d, want a redraw per frame for a key it cannot compare", child.renders)
	}
}

func TestMemoPassesFocusThrough(t *testing.T) {
	inner := Row(Grow(1, &probe{label: "a"}).Focusable(), Grow(1, &probe{label: "b"}).Focusable())

	outer := Column(Grow(1, Memo(inner, func() any { return 0 })))

	if !outer.FocusNext() {
		t.Fatal("Tab could not descend into a memoised box")
	}

	if inner.Focused() != 1 {
		t.Errorf("inner focused = %d, want 1", inner.Focused())
	}
}
