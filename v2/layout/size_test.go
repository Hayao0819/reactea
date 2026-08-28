package layout

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/Hayao0819/reactea/v2"
)

// misfit renders a box deliberately off from the one it was handed.
type misfit struct {
	reactea.BasicComponent

	fill   rune
	dw, dh int
}

func (c *misfit) Render(ctx *reactea.Ctx) string {
	width, height := ctx.Size()
	width, height = width+c.dw, height+c.dh

	if width <= 0 || height <= 0 {
		return ""
	}

	rows := make([]string, height)
	for i := range rows {
		rows[i] = strings.Repeat(string(c.fill), width)
	}

	return strings.Join(rows, "\n")
}

func rendered(component reactea.Component, width, height int) (int, int) {
	content := reactea.New(component, reactea.WithSize(width, height)).View().Content

	return lipgloss.Size(content)
}

func TestBoxHoldsItsBoxWhenAChildMisfits(t *testing.T) {
	cases := []struct {
		name          string
		box           *Box
		width, height int
	}{
		{"row, short child", Row(
			Grow(1, &misfit{fill: 'a', dw: -3}),
			Grow(1, &misfit{fill: 'b'}),
		), 20, 1},
		{"row, long child", Row(
			Grow(1, &misfit{fill: 'a', dw: 4}),
			Grow(1, &misfit{fill: 'b'}),
		), 20, 1},
		{"column, short child", Column(
			Grow(1, &misfit{fill: 'a', dh: -2}),
			Grow(1, &misfit{fill: 'b'}),
		), 6, 8},
		{"column, tall child", Column(
			Grow(1, &misfit{fill: 'a', dh: 3}),
			Grow(1, &misfit{fill: 'b'}),
		), 6, 8},
	}

	for _, tc := range cases {
		width, height := rendered(tc.box, tc.width, tc.height)

		if width != tc.width || height != tc.height {
			t.Errorf("%s: rendered %dx%d, want %dx%d", tc.name, width, height, tc.width, tc.height)
		}
	}
}

func TestFrameHoldsItsBoxWhenTheChildMisfits(t *testing.T) {
	style := lipgloss.NewStyle().Border(lipgloss.RoundedBorder())

	cases := []struct {
		name          string
		child         *misfit
		width, height int
	}{
		{"wide child", &misfit{fill: 'a', dw: 5}, 20, 3},
		{"tall child", &misfit{fill: 'a', dh: 3}, 12, 4},
		{"small child", &misfit{fill: 'a', dw: -4, dh: -1}, 20, 5},
	}

	for _, tc := range cases {
		width, height := rendered(Framed(style, tc.child), tc.width, tc.height)

		if width != tc.width || height != tc.height {
			t.Errorf("%s: rendered %dx%d, want %dx%d", tc.name, width, height, tc.width, tc.height)
		}
	}
}
