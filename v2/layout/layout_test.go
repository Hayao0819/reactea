package layout

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/Hayao0819/reactea/v2"
)

type probe struct {
	reactea.BasicComponent

	label string

	width, height int
	originX       int
	originY       int
	inited        bool
	destroyed     bool
	updates       int

	cursorX, cursorY int
	wantsCursor      bool

	messages []tea.Msg
}

func (c *probe) Init(ctx *reactea.Ctx) tea.Cmd {
	c.inited = true

	ctx.OnDestroy(func() { c.destroyed = true })

	return nil
}

func (c *probe) Update(_ *reactea.Ctx, msg tea.Msg) tea.Cmd {
	c.updates++
	c.messages = append(c.messages, msg)

	return nil
}

func (c *probe) Render(ctx *reactea.Ctx) string {
	c.width, c.height = ctx.Size()
	c.originX, c.originY = ctx.Origin()

	if c.wantsCursor {
		ctx.CursorAt(c.cursorX, c.cursorY)
	}

	lines := make([]string, max(c.height, 1))
	for i := range lines {
		lines[i] = c.label
	}

	return strings.Join(lines, "\n")
}

type tickMsg struct{}

func renderBox(box *Box, width, height int) *reactea.App {
	app := reactea.New(box, reactea.WithSize(width, height))

	app.View()

	return app
}

func TestColumnSplitsHeight(t *testing.T) {
	header, body, footer := &probe{label: "h"}, &probe{label: "b"}, &probe{label: "f"}

	renderBox(Column(Fixed(1, header), Grow(1, body), Fixed(1, footer)), 20, 10)

	if header.height != 1 || footer.height != 1 {
		t.Errorf("fixed heights = %d, %d, want 1, 1", header.height, footer.height)
	}

	if body.height != 8 {
		t.Errorf("body height = %d, want 8", body.height)
	}

	if header.width != 20 || body.width != 20 {
		t.Errorf("cross axis = %d, %d, want 20", header.width, body.width)
	}

	if body.originY != 1 {
		t.Errorf("body origin y = %d, want 1", body.originY)
	}
}

func TestRowSplitsWidthByWeight(t *testing.T) {
	sidebar, main := &probe{label: "s"}, &probe{label: "m"}

	renderBox(Row(Grow(1, sidebar), Grow(3, main)), 100, 5)

	if sidebar.width != 25 || main.width != 75 {
		t.Errorf("widths = %d, %d, want 25, 75", sidebar.width, main.width)
	}

	if main.originX != 25 {
		t.Errorf("main origin x = %d, want 25", main.originX)
	}
}

func TestDistributionLosesNoCells(t *testing.T) {
	cases := []struct {
		total   int
		weights []int
	}{
		{10, []int{1, 1, 1}},
		{7, []int{2, 3}},
		{100, []int{1, 1, 1, 1, 1, 1, 7}},
		{1, []int{1, 1}},
		{0, []int{1, 1}},
	}

	for _, tc := range cases {
		items := make([]Item, 0, len(tc.weights))
		for _, w := range tc.weights {
			items = append(items, Grow(w, &probe{label: "x"}))
		}

		sizes := distribute(tc.total, items)

		sum := 0
		for _, size := range sizes {
			if size < 0 {
				t.Errorf("total %d weights %v: negative size in %v", tc.total, tc.weights, sizes)
			}

			sum += size
		}

		if sum != tc.total {
			t.Errorf("total %d weights %v: sizes %v sum to %d", tc.total, tc.weights, sizes, sum)
		}
	}
}

func TestFixedLargerThanBoxIsTruncated(t *testing.T) {
	sizes := distribute(3, []Item{Fixed(10, &probe{}), Grow(1, &probe{})})

	if sizes[0] != 3 || sizes[1] != 0 {
		t.Errorf("sizes = %v, want [3 0]", sizes)
	}
}

func TestBoundedRespectsMinAndMax(t *testing.T) {
	sizes := distribute(100, []Item{Bounded(1, 0, 20, &probe{}), Grow(1, &probe{})})

	if sizes[0] != 20 || sizes[0]+sizes[1] != 100 {
		t.Errorf("max: sizes = %v", sizes)
	}

	sizes = distribute(10, []Item{Bounded(1, 8, 0, &probe{}), Grow(9, &probe{})})

	if sizes[0] != 8 || sizes[0]+sizes[1] != 10 {
		t.Errorf("min: sizes = %v", sizes)
	}
}

func TestCursorIsTranslatedByOffset(t *testing.T) {
	body := &probe{label: "b", wantsCursor: true, cursorX: 3, cursorY: 2}

	app := renderBox(Column(Fixed(2, &probe{label: "h"}), Grow(1, body).Focusable()), 20, 10)

	cursor := app.View().Cursor

	if cursor == nil {
		t.Fatal("the child cursor did not reach the app")
	}

	if cursor.X != 3 || cursor.Y != 4 {
		t.Errorf("cursor = (%d, %d), want (3, 4)", cursor.X, cursor.Y)
	}
}

func TestRowTranslatesCursorHorizontally(t *testing.T) {
	main := &probe{label: "m", wantsCursor: true, cursorX: 1, cursorY: 1}

	app := renderBox(Row(Fixed(10, &probe{label: "s"}), Grow(1, main).Focusable()), 40, 5)

	cursor := app.View().Cursor

	if cursor.X != 11 || cursor.Y != 1 {
		t.Errorf("cursor = (%d, %d), want (11, 1)", cursor.X, cursor.Y)
	}
}

func TestUpdateSeesTheSameSplit(t *testing.T) {
	body := &probe{label: "b"}

	box := Column(Fixed(2, &probe{label: "h"}), Grow(1, body))

	app := reactea.New(box, reactea.WithSize(20, 10))

	app.Update(tickMsg{})

	if body.updates != 1 {
		t.Fatalf("body saw %d updates", body.updates)
	}

	app.View()

	if body.height != 8 || body.originY != 2 {
		t.Errorf("body box = %dx%d at y=%d", body.width, body.height, body.originY)
	}
}

func TestLifecycleReachesEveryItem(t *testing.T) {
	first, second := &probe{label: "a"}, &probe{label: "b"}

	box := Column(Grow(1, first), Grow(1, second))

	app := reactea.New(box, reactea.WithSize(10, 4))

	app.Init()
	app.Update(tickMsg{})
	app.Scope().Close()

	for _, c := range []*probe{first, second} {
		if !c.inited || c.updates != 1 || !c.destroyed {
			t.Errorf("%s: inited=%v updates=%d destroyed=%v", c.label, c.inited, c.updates, c.destroyed)
		}
	}
}

func TestRenderJoinsChildren(t *testing.T) {
	box := Column(Fixed(1, reactea.Text("top")), Fixed(1, reactea.Text("bottom")))

	rendered := reactea.New(box, reactea.WithSize(10, 2)).View().Content

	if !strings.Contains(rendered, "top") || !strings.Contains(rendered, "bottom") {
		t.Errorf("render = %q", rendered)
	}

	if lines := strings.Count(rendered, "\n"); lines != 1 {
		t.Errorf("render has %d newlines, want 1:\n%q", lines, rendered)
	}
}

func keys(box *Box, app *reactea.App, presses ...string) {
	for _, key := range presses {
		app.Update(tea.KeyPressMsg{Code: rune(key[0]), Text: key})
	}
}

func TestKeyboardOnlyReachesTheFocusedItem(t *testing.T) {
	left, right := &probe{label: "l"}, &probe{label: "r"}

	box := Row(Grow(1, left).Focusable(), Grow(1, right).Focusable())

	app := reactea.New(box, reactea.WithSize(20, 4))

	keys(box, app, "x")

	if left.updates != 1 || right.updates != 0 {
		t.Errorf("updates = %d, %d, want 1, 0", left.updates, right.updates)
	}

	box.FocusNext()
	keys(box, app, "x")

	if left.updates != 1 || right.updates != 1 {
		t.Errorf("after FocusNext updates = %d, %d, want 1, 1", left.updates, right.updates)
	}
}

func TestFocusSkipsItemsThatCannotTakeIt(t *testing.T) {
	header, body := &probe{label: "h"}, &probe{label: "b"}

	box := Column(Fixed(1, header), Grow(1, body).Focusable())

	if box.Focused() != 1 {
		t.Errorf("focused = %d, want 1", box.Focused())
	}

	if box.FocusNext() {
		t.Error("FocusNext ran past the end and reported success")
	}
}

func TestFocusDescendsIntoNestedBoxes(t *testing.T) {
	a, b, c := &probe{label: "a"}, &probe{label: "b"}, &probe{label: "c"}

	inner := Column(Grow(1, b).Focusable(), Grow(1, c).Focusable())
	outer := Row(Grow(1, a).Focusable(), Grow(1, inner))

	app := reactea.New(outer, reactea.WithSize(20, 4))

	seen := []int{}

	for {
		app.Update(tea.KeyPressMsg{Code: 'x', Text: "x"})
		seen = append(seen, a.updates+b.updates+c.updates)

		if !outer.FocusNext() {
			break
		}
	}

	if a.updates != 1 || b.updates != 1 || c.updates != 1 {
		t.Errorf("updates = %d, %d, %d, want 1, 1, 1 (seen %v)", a.updates, b.updates, c.updates, seen)
	}
}

func TestMouseGoesToTheBoxUnderThePointer(t *testing.T) {
	left, right := &probe{label: "l"}, &probe{label: "r"}

	box := Row(Grow(1, left).Focusable(), Grow(1, right).Focusable())

	app := reactea.New(box, reactea.WithSize(20, 4))

	app.Update(tea.MouseClickMsg{X: 15, Y: 2, Button: tea.MouseLeft})

	if right.updates != 1 || left.updates != 0 {
		t.Fatalf("updates = %d, %d, want 0, 1", left.updates, right.updates)
	}

	click, ok := right.messages[0].(tea.MouseClickMsg)
	if !ok {
		t.Fatalf("got %T", right.messages[0])
	}

	if click.X != 5 || click.Y != 2 {
		t.Errorf("coordinates = (%d, %d), want (5, 2) in the box's own space", click.X, click.Y)
	}

	if box.Focused() != 1 {
		t.Errorf("a press did not move the focus: %d", box.Focused())
	}
}

func TestWheelDoesNotMoveTheFocus(t *testing.T) {
	left, right := &probe{label: "l"}, &probe{label: "r"}

	box := Row(Grow(1, left).Focusable(), Grow(1, right).Focusable())

	app := reactea.New(box, reactea.WithSize(20, 4))

	app.Update(tea.MouseWheelMsg{X: 15, Y: 2})

	if right.updates != 1 {
		t.Errorf("the wheel did not reach the box under the pointer")
	}

	if box.Focused() != 0 {
		t.Errorf("the wheel moved the focus to %d", box.Focused())
	}
}

func TestOnlyTheFocusedItemMaySetTheCursor(t *testing.T) {
	left := &probe{label: "l", wantsCursor: true, cursorX: 1, cursorY: 1}
	right := &probe{label: "r", wantsCursor: true, cursorX: 2, cursorY: 2}

	box := Row(Grow(1, left).Focusable(), Grow(1, right).Focusable())

	app := reactea.New(box, reactea.WithSize(20, 4))

	cursor := app.View().Cursor

	if cursor == nil || cursor.X != 1 || cursor.Y != 1 {
		t.Errorf("cursor = %+v, want the focused item's", cursor)
	}
}

func TestDecorativeContainersAreNotFocusable(t *testing.T) {
	meter, list := &probe{label: "m"}, &probe{label: "l"}

	box := Column(
		Grow(1, Framed(lipgloss.NewStyle(), meter)),
		Grow(1, Framed(lipgloss.NewStyle(), list).WhenFocused(lipgloss.NewStyle())).Focusable(),
	)

	if box.Focused() != 1 {
		t.Fatalf("focused = %d, want the only focusable item", box.Focused())
	}

	app := reactea.New(box, reactea.WithSize(20, 6))

	app.Update(tea.KeyPressMsg{Code: 'x', Text: "x"})

	if meter.updates != 0 || list.updates != 1 {
		t.Errorf("updates = %d, %d, want 0, 1", meter.updates, list.updates)
	}
}

func TestEmptyNestedBoxIsNotFocusable(t *testing.T) {
	meters := Row(Grow(1, &probe{label: "a"}), Grow(1, &probe{label: "b"}))
	list := &probe{label: "l"}

	box := Column(Grow(1, meters), Grow(1, list).Focusable())

	if meters.HasFocusable() {
		t.Error("a box with no focusable items claimed to have one")
	}

	if box.Focused() != 1 {
		t.Errorf("focused = %d, want 1", box.Focused())
	}
}

func TestSettlingNeverTakesAnItemBelowItsMin(t *testing.T) {
	sizes := distribute(10, []Item{
		Bounded(1, 8, 0, &probe{label: "a"}),
		Bounded(1, 3, 0, &probe{label: "b"}),
	})

	if sizes[1] < 3 {
		t.Errorf("sizes = %v, the second item fell below its Min of 3", sizes)
	}
}

func TestSetItemsKeepsTheFocusOnTheSameComponent(t *testing.T) {
	first, second, third := &probe{label: "a"}, &probe{label: "b"}, &probe{label: "c"}

	box := Row(Grow(1, first).Focusable(), Grow(1, second).Focusable())

	box.FocusNext()

	if box.Focused() != 1 {
		t.Fatalf("focused = %d", box.Focused())
	}

	// Hiding the first pane must not move the focus off the second.
	box.SetItems(Grow(1, third).Focusable(), Grow(1, second).Focusable())

	if box.Focused() != 1 {
		t.Errorf("focused = %d, want the item that kept its component", box.Focused())
	}

	// Dropping the focused component falls back to the first that can take it.
	box.SetItems(Grow(1, third).Focusable())

	if box.Focused() != 0 {
		t.Errorf("focused = %d, want 0 after the focused item went away", box.Focused())
	}

	if len(box.Items()) != 1 {
		t.Errorf("Items() = %d entries", len(box.Items()))
	}
}

// valueComponent is uncomparable, which == would panic on.
type valueComponent struct{ tags []string }

func (c valueComponent) Init(*reactea.Ctx) tea.Cmd            { return nil }
func (c valueComponent) Update(*reactea.Ctx, tea.Msg) tea.Cmd { return nil }
func (c valueComponent) Render(*reactea.Ctx) string           { return "" }

func TestSetItemsSurvivesUncomparableComponents(t *testing.T) {
	box := Row(Grow(1, valueComponent{tags: []string{"a"}}).Focusable())

	box.SetItems(Grow(1, valueComponent{tags: []string{"b"}}).Focusable())

	if box.Focused() != 0 {
		t.Errorf("focused = %d", box.Focused())
	}
}

func TestStarvedNamesTheItemsThatLostOut(t *testing.T) {
	box := Row(
		Bounded(1, 12, 0, &probe{label: "a"}),
		Bounded(1, 12, 0, &probe{label: "b"}),
		Bounded(1, 12, 0, &probe{label: "c"}),
	)

	renderBox(box, 20, 1)

	// 12, 8 and 0: the second is below its Min, the third got nothing.
	if got := box.Starved(); len(got) != 2 || got[0] != 1 || got[1] != 2 {
		t.Errorf("Starved() = %v, want [1 2]", got)
	}

	renderBox(box, 40, 1)

	if got := box.Starved(); len(got) != 0 {
		t.Errorf("Starved() = %v in a box with room for everyone", got)
	}
}

func TestStarvedReportsAShortenedFixed(t *testing.T) {
	box := Row(Fixed(15, &probe{label: "a"}), Fixed(15, &probe{label: "b"}))

	renderBox(box, 20, 1)

	if got := box.Starved(); len(got) != 1 || got[0] != 1 {
		t.Errorf("Starved() = %v, want [1]", got)
	}
}

func TestStarvedIsACopy(t *testing.T) {
	box := Row(
		Bounded(1, 40, 0, &probe{label: "a"}),
		Bounded(1, 40, 0, &probe{label: "b"}),
	)

	renderBox(box, 20, 1)

	kept := box.Starved()

	if len(kept) != 2 {
		t.Fatalf("Starved() = %v, want both items", kept)
	}

	// Wide enough for both, so the box refills its own record with nothing.
	renderBox(box, 200, 1)

	if len(box.Starved()) != 0 {
		t.Fatalf("Starved() = %v in a box with room", box.Starved())
	}

	if len(kept) != 2 || kept[0] != 0 || kept[1] != 1 {
		t.Errorf("a kept result changed under the caller: %v", kept)
	}
}

// starvedReader reports what it could see from inside its own Box.
type starvedReader struct {
	reactea.BasicComponent

	box  *Box
	seen []int
}

func (c *starvedReader) Render(*reactea.Ctx) string {
	c.seen = c.box.Starved()

	return ""
}

func TestStarvedIsSettledBeforeAnyChildRenders(t *testing.T) {
	reader := &starvedReader{}

	box := Column(
		Fixed(1, reader),
		Bounded(1, 40, 0, &probe{label: "a"}),
		Bounded(1, 40, 0, &probe{label: "b"}),
	)

	reader.box = box

	renderBox(box, 20, 4)

	outside := box.Starved()

	if len(reader.seen) != len(outside) {
		t.Errorf("the first item saw %v, the caller outside saw %v", reader.seen, outside)
	}
}

type wrapping struct {
	reactea.Wrapper
}

func TestAWrapperStaysInTheTabOrder(t *testing.T) {
	inner := Row(Grow(1, &probe{label: "a"}).Focusable(), Grow(1, &probe{label: "b"}).Focusable())

	outer := Column(Grow(1, &wrapping{Wrapper: reactea.Wrap(inner)}))

	if !outer.HasFocusable() {
		t.Fatal("a Wrapper hid the focusables inside it")
	}

	if !outer.FocusNext() {
		t.Fatal("Tab could not descend through the Wrapper")
	}

	if inner.Focused() != 1 {
		t.Errorf("inner focused = %d, want 1", inner.Focused())
	}
}

func TestAWrapperWithNothingFocusableIsNotFocusable(t *testing.T) {
	outer := Column(
		Grow(1, &wrapping{Wrapper: reactea.Wrap(reactea.Text("plain"))}),
		Grow(1, &probe{label: "b"}).Focusable(),
	)

	if outer.Focused() != 1 {
		t.Errorf("focused = %d, want the only focusable item", outer.Focused())
	}
}

func TestSpacerTakesRoomAndDrawsNothing(t *testing.T) {
	left, right := &probe{label: "l"}, &probe{label: "r"}

	renderBox(Row(Fixed(4, left), Spacer(2), Fixed(4, right)), 10, 1)

	if left.width != 4 || right.width != 4 {
		t.Errorf("widths = %d, %d, want 4, 4", left.width, right.width)
	}

	content := reactea.New(
		Row(Fixed(2, reactea.Text("ab")), Spacer(2), Fixed(2, reactea.Text("cd"))),
		reactea.WithSize(6, 1),
	).View().Content

	if content != "ab  cd" {
		t.Errorf("content = %q, want %q", content, "ab  cd")
	}
}
