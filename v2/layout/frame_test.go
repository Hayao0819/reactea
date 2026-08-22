package layout

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

func TestFramedShrinksTheChild(t *testing.T) {
	child := &probe{label: "x"}

	style := lipgloss.NewStyle().Border(lipgloss.NormalBorder()).Padding(1, 2)

	Framed(style, child).Render(30, 10)

	// Border is 1 on each side, padding 1 vertical and 2 horizontal.
	if child.width != 30-2-4 {
		t.Errorf("child width = %d, want %d", child.width, 30-6)
	}

	if child.height != 10-2-2 {
		t.Errorf("child height = %d, want %d", child.height, 10-4)
	}
}

// Lipgloss counts Width/Height as the outer size, so a framed child still fits
// the box the parent handed out.
func TestFramedFillsItsBox(t *testing.T) {
	framed := Framed(lipgloss.NewStyle().Border(lipgloss.NormalBorder()), &probe{label: "x"})

	width, height := lipgloss.Size(framed.Render(24, 6))

	if width != 24 || height != 6 {
		t.Errorf("rendered %dx%d, want 24x6", width, height)
	}
}

func TestFramedTranslatesCursor(t *testing.T) {
	child := &probe{label: "x", cursor: tea.NewCursor(2, 1)}

	style := lipgloss.NewStyle().Border(lipgloss.NormalBorder()).Padding(1, 3)

	framed := Framed(style, child)
	framed.Render(30, 10)

	view := tea.NewView("")

	framed.DecorateView(&view)

	if view.Cursor == nil {
		t.Fatal("the child cursor did not reach the frame")
	}

	// left border 1 + left padding 3, top border 1 + top padding 1.
	if view.Cursor.X != 2+4 || view.Cursor.Y != 1+2 {
		t.Errorf("cursor = (%d, %d), want (6, 3)", view.Cursor.X, view.Cursor.Y)
	}
}

// A frame inside a box has to stack both offsets.
func TestFramedInsideColumn(t *testing.T) {
	child := &probe{label: "x", cursor: tea.NewCursor(0, 0)}

	framed := Framed(lipgloss.NewStyle().Border(lipgloss.NormalBorder()), child)

	box := Column(Fixed(2, &probe{label: "h"}), Grow(1, framed))
	box.Render(20, 10)

	view := tea.NewView("")

	box.DecorateView(&view)

	if view.Cursor.X != 1 || view.Cursor.Y != 3 {
		t.Errorf("cursor = (%d, %d), want (1, 3)", view.Cursor.X, view.Cursor.Y)
	}
}

func TestFramedSmallerThanItsFrame(t *testing.T) {
	child := &probe{label: "x"}

	style := lipgloss.NewStyle().Border(lipgloss.NormalBorder()).Padding(2)

	Framed(style, child).Render(2, 2)

	if child.width != 0 || child.height != 0 {
		t.Errorf("child size = %dx%d, want 0x0", child.width, child.height)
	}
}

func TestFramedForwardsLifecycle(t *testing.T) {
	child := &probe{label: "x"}

	framed := Framed(lipgloss.NewStyle(), child)

	framed.Init()
	framed.Update(tea.KeyPressMsg{Code: 'x', Text: "x"})
	framed.Destroy()

	if !child.inited || child.updates != 1 || !child.destroyed {
		t.Errorf("inited=%v updates=%d destroyed=%v", child.inited, child.updates, child.destroyed)
	}
}
