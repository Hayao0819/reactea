package render_test

import (
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/Hayao0819/reactea/v2/render"
)

func TestFittingCountsDisplayCellsNotBytes(t *testing.T) {
	cases := []struct {
		name string
		text string
	}{
		{"ascii", "abc"},
		{"wide characters", "日本語"},
		{"styled text", lipgloss.NewStyle().Bold(true).Render("abc")},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			for _, width := range []int{1, 3, 8} {
				if got := lipgloss.Width(render.Left(test.text, width)); got != width {
					t.Errorf("Left(%q, %d) is %d cells wide", test.text, width, got)
				}

				if got := lipgloss.Width(render.Right(test.text, width)); got != width {
					t.Errorf("Right(%q, %d) is %d cells wide", test.text, width, got)
				}

				if got := lipgloss.Width(render.Clip(test.text, width)); got > width {
					t.Errorf("Clip(%q, %d) is %d cells wide", test.text, width, got)
				}
			}
		})
	}
}

func TestNoRoomRendersNothing(t *testing.T) {
	for _, width := range []int{0, -1} {
		if got := render.Left("abc", width); got != "" {
			t.Errorf("Left at width %d = %q", width, got)
		}

		if got := render.Clip("abc", width); got != "" {
			t.Errorf("Clip at width %d = %q", width, got)
		}
	}

	if got := render.Fit("abc", 0, 4); got != "" {
		t.Errorf("Fit with no width = %q", got)
	}
}

func TestFitHoldsBothAxes(t *testing.T) {
	fitted := render.Fit("a\nb", 5, 4)

	if width, height := lipgloss.Size(fitted); width != 5 || height != 4 {
		t.Errorf("Fit gave %dx%d, want 5x4", width, height)
	}
}
