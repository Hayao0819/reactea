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
	inited        bool
	destroyed     bool
	updates       int

	cursor *tea.Cursor
}

func (c *probe) Init() tea.Cmd {
	c.inited = true

	return nil
}

func (c *probe) Destroy() { c.destroyed = true }

func (c *probe) Update(tea.Msg) tea.Cmd {
	c.updates++

	return nil
}

func (c *probe) Render(width, height int) string {
	c.width, c.height = width, height

	lines := make([]string, height)
	for i := range lines {
		lines[i] = c.label
	}

	return strings.Join(lines, "\n")
}

func (c *probe) DecorateView(view *tea.View) {
	if c.cursor != nil {
		view.Cursor = c.cursor
	}
}

func TestColumnSplitsHeight(t *testing.T) {
	header, body, footer := &probe{label: "h"}, &probe{label: "b"}, &probe{label: "f"}

	box := Column(
		Fixed(1, header),
		Grow(1, body),
		Fixed(1, footer),
	)

	box.Render(20, 10)

	if header.height != 1 || footer.height != 1 {
		t.Errorf("fixed heights = %d, %d, want 1, 1", header.height, footer.height)
	}

	if body.height != 8 {
		t.Errorf("body height = %d, want 8", body.height)
	}

	if header.width != 20 || body.width != 20 {
		t.Errorf("cross axis = %d, %d, want 20", header.width, body.width)
	}
}

func TestRowSplitsWidthByWeight(t *testing.T) {
	sidebar, main := &probe{label: "s"}, &probe{label: "m"}

	Row(Grow(1, sidebar), Grow(3, main)).Render(100, 5)

	if sidebar.width != 25 || main.width != 75 {
		t.Errorf("widths = %d, %d, want 25, 75", sidebar.width, main.width)
	}

	if sidebar.height != 5 {
		t.Errorf("cross axis = %d, want 5", sidebar.height)
	}
}

// Every cell has to be handed out even when the split does not divide evenly.
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
	sizes := distribute(100, []Item{
		Bounded(1, 0, 20, &probe{}),
		Grow(1, &probe{}),
	})

	if sizes[0] != 20 {
		t.Errorf("max was not applied: %v", sizes)
	}

	if sizes[0]+sizes[1] != 100 {
		t.Errorf("sizes %v do not fill the box", sizes)
	}

	sizes = distribute(10, []Item{
		Bounded(1, 8, 0, &probe{}),
		Grow(9, &probe{}),
	})

	if sizes[0] != 8 {
		t.Errorf("min was not applied: %v", sizes)
	}

	if sizes[0]+sizes[1] != 10 {
		t.Errorf("sizes %v do not fill the box", sizes)
	}
}

// The whole point of the package: a child's cursor comes back in the parent's
// coordinate space without the parent doing the arithmetic.
func TestCursorIsTranslatedByOffset(t *testing.T) {
	header := &probe{label: "h"}
	body := &probe{label: "b", cursor: tea.NewCursor(3, 2)}

	box := Column(Fixed(2, header), Grow(1, body))

	box.Render(20, 10)

	view := tea.NewView("")

	box.DecorateView(&view)

	if view.Cursor == nil {
		t.Fatal("the child cursor did not reach the parent")
	}

	if view.Cursor.X != 3 || view.Cursor.Y != 4 {
		t.Errorf("cursor = (%d, %d), want (3, 4)", view.Cursor.X, view.Cursor.Y)
	}
}

func TestRowTranslatesCursorHorizontally(t *testing.T) {
	sidebar := &probe{label: "s"}
	main := &probe{label: "m", cursor: tea.NewCursor(1, 1)}

	box := Row(Fixed(10, sidebar), Grow(1, main))

	box.Render(40, 5)

	view := tea.NewView("")

	box.DecorateView(&view)

	if view.Cursor.X != 11 || view.Cursor.Y != 1 {
		t.Errorf("cursor = (%d, %d), want (11, 1)", view.Cursor.X, view.Cursor.Y)
	}
}

func TestDecorationsMerge(t *testing.T) {
	first := &probe{label: "a"}
	second := &probe{label: "b"}

	box := Column(Grow(1, first), Grow(1, second))
	box.Render(10, 4)

	view := tea.NewView("")
	view.WindowTitle = "kept"

	box.DecorateView(&view)

	if view.WindowTitle != "kept" {
		t.Errorf("an empty child title overwrote the parent's: %q", view.WindowTitle)
	}
}

func TestLifecycleReachesEveryItem(t *testing.T) {
	first, second := &probe{label: "a"}, &probe{label: "b"}

	box := Column(Grow(1, first), Grow(1, second))

	box.Init()
	box.Update(tea.KeyPressMsg{Code: 'x', Text: "x"})
	box.Destroy()

	for _, c := range []*probe{first, second} {
		if !c.inited || c.updates != 1 || !c.destroyed {
			t.Errorf("%s: inited=%v updates=%d destroyed=%v", c.label, c.inited, c.updates, c.destroyed)
		}
	}
}

func TestRenderJoinsChildren(t *testing.T) {
	box := Column(
		Fixed(1, reactea.StaticComponent("top")),
		Fixed(1, reactea.StaticComponent("bottom")),
	)

	rendered := box.Render(10, 2)

	if !strings.Contains(rendered, "top") || !strings.Contains(rendered, "bottom") {
		t.Errorf("render = %q", rendered)
	}

	if lines := strings.Count(rendered, "\n"); lines != 1 {
		t.Errorf("render has %d newlines, want 1:\n%q", lines, rendered)
	}
}
