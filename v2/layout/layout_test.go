package layout

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
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
}

func (c *probe) Init(ctx *reactea.Ctx) tea.Cmd {
	c.inited = true

	ctx.OnDestroy(func() { c.destroyed = true })

	return nil
}

func (c *probe) Update(*reactea.Ctx, tea.Msg) tea.Cmd {
	c.updates++

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

func render(box *Box, width, height int) *reactea.App {
	app := reactea.New(box, reactea.WithSize(width, height))

	app.View()

	return app
}

func TestColumnSplitsHeight(t *testing.T) {
	header, body, footer := &probe{label: "h"}, &probe{label: "b"}, &probe{label: "f"}

	render(Column(Fixed(1, header), Grow(1, body), Fixed(1, footer)), 20, 10)

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

	render(Row(Grow(1, sidebar), Grow(3, main)), 100, 5)

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

	app := render(Column(Fixed(2, &probe{label: "h"}), Grow(1, body)), 20, 10)

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

	app := render(Row(Fixed(10, &probe{label: "s"}), Grow(1, main)), 40, 5)

	cursor := app.View().Cursor

	if cursor.X != 11 || cursor.Y != 1 {
		t.Errorf("cursor = (%d, %d), want (11, 1)", cursor.X, cursor.Y)
	}
}

func TestUpdateSeesTheSameSplit(t *testing.T) {
	body := &probe{label: "b"}

	box := Column(Fixed(2, &probe{label: "h"}), Grow(1, body))

	app := reactea.New(box, reactea.WithSize(20, 10))

	app.Update(tea.KeyPressMsg{Code: 'x', Text: "x"})

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
	app.Update(tea.KeyPressMsg{Code: 'x', Text: "x"})
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
