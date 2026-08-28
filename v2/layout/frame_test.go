package layout

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/Hayao0819/reactea/v2"
)

func TestFramedShrinksTheChild(t *testing.T) {
	child := &probe{label: "x"}

	style := lipgloss.NewStyle().Border(lipgloss.NormalBorder()).Padding(1, 2)

	reactea.New(Framed(style, child), reactea.WithSize(30, 10)).View()

	// Border is 1 on each side, padding 1 vertical and 2 horizontal.
	if child.width != 30-6 || child.height != 10-4 {
		t.Errorf("child box = %dx%d, want %dx%d", child.width, child.height, 30-6, 10-4)
	}
}

func TestFramedFillsItsBox(t *testing.T) {
	framed := Framed(lipgloss.NewStyle().Border(lipgloss.NormalBorder()), &probe{label: "x"})

	content := reactea.New(framed, reactea.WithSize(24, 6)).View().Content

	if width, height := lipgloss.Size(content); width != 24 || height != 6 {
		t.Errorf("rendered %dx%d, want 24x6", width, height)
	}
}

func TestFramedTranslatesCursor(t *testing.T) {
	child := &probe{label: "x", wantsCursor: true, cursorX: 2, cursorY: 1}

	style := lipgloss.NewStyle().Border(lipgloss.NormalBorder()).Padding(1, 3)

	app := reactea.New(Framed(style, child), reactea.WithSize(30, 10))

	cursor := app.View().Cursor

	if cursor == nil {
		t.Fatal("the child cursor did not reach the app")
	}

	// left border 1 + left padding 3, top border 1 + top padding 1.
	if cursor.X != 2+4 || cursor.Y != 1+2 {
		t.Errorf("cursor = (%d, %d), want (6, 3)", cursor.X, cursor.Y)
	}
}

func TestFramedInsideColumn(t *testing.T) {
	child := &probe{label: "x", wantsCursor: true}

	framed := Framed(lipgloss.NewStyle().Border(lipgloss.NormalBorder()), child)

	app := reactea.New(
		Column(Fixed(2, &probe{label: "h"}), Grow(1, framed).Focusable()),
		reactea.WithSize(20, 10),
	)

	cursor := app.View().Cursor

	if cursor.X != 1 || cursor.Y != 3 {
		t.Errorf("cursor = (%d, %d), want (1, 3)", cursor.X, cursor.Y)
	}
}

func TestFramedSmallerThanItsFrame(t *testing.T) {
	child := &probe{label: "x"}

	style := lipgloss.NewStyle().Border(lipgloss.NormalBorder()).Padding(2)

	reactea.New(Framed(style, child), reactea.WithSize(2, 2)).View()

	if child.width != 0 || child.height != 0 {
		t.Errorf("child size = %dx%d, want 0x0", child.width, child.height)
	}
}

func TestFramedForwardsLifecycle(t *testing.T) {
	child := &probe{label: "x"}

	framed := Framed(lipgloss.NewStyle(), child)

	app := reactea.New(framed, reactea.WithSize(10, 4))

	app.Init()
	app.Update(tea.KeyPressMsg{Code: 'x', Text: "x"})
	app.Scope().Close()

	if !child.inited || child.updates != 1 || !child.destroyed {
		t.Errorf("inited=%v updates=%d destroyed=%v", child.inited, child.updates, child.destroyed)
	}
}

func TestWhenFocusedSwapsTheStyle(t *testing.T) {
	plain := lipgloss.NewStyle().Border(lipgloss.NormalBorder())
	marked := lipgloss.NewStyle().Border(lipgloss.RoundedBorder())

	framed := Framed(plain, &probe{label: "x"}).WhenFocused(marked)

	focused := reactea.New(framed, reactea.WithSize(10, 3)).View().Content

	if !strings.Contains(focused, "╭") {
		t.Errorf("the focused style was not used:\n%s", focused)
	}

	box := Column(Fixed(1, &probe{label: "h"}).Focusable(), Grow(1, framed))

	blurred := reactea.New(box, reactea.WithSize(10, 4)).View().Content

	if strings.Contains(blurred, "╭") {
		t.Errorf("the focused style was used while blurred:\n%s", blurred)
	}
}

func TestFrameDropsClicksOnItsBorder(t *testing.T) {
	child := &probe{label: "x"}

	style := lipgloss.NewStyle().Border(lipgloss.NormalBorder())

	app := reactea.New(
		Column(Grow(1, Framed(style, child)).Focusable()),
		reactea.WithSize(20, 6),
	)

	app.Init()

	for _, y := range []int{0, 5} {
		app.Update(tea.MouseClickMsg{X: 5, Y: y, Button: tea.MouseLeft})
	}

	if child.updates != 0 {
		t.Errorf("the child saw %d border clicks", child.updates)
	}

	app.Update(tea.MouseClickMsg{X: 5, Y: 2, Button: tea.MouseLeft})

	if child.updates != 1 {
		t.Errorf("a click inside the frame did not reach the child (%d)", child.updates)
	}

	click, ok := child.messages[0].(tea.MouseClickMsg)
	if !ok {
		t.Fatalf("got %T", child.messages[0])
	}

	if click.X != 4 || click.Y != 1 {
		t.Errorf("coordinates = (%d, %d), want the child's own space", click.X, click.Y)
	}
}
