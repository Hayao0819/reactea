package testkit_test

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/Hayao0819/reactea/v2"
	"github.com/Hayao0819/reactea/v2/testkit"
)

// echoing draws the last key it saw and where it was last clicked.
type echoing struct {
	reactea.BasicComponent

	key   string
	click string
}

func (c *echoing) Render(*reactea.Ctx) string {
	return lipgloss.NewStyle().Bold(true).Render("key=" + c.key + " click=" + c.click)
}

func (c *echoing) Update(_ *reactea.Ctx, msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		c.key = msg.String()
	case tea.MouseClickMsg:
		c.click = string(rune('0'+msg.X)) + string(rune('0'+msg.Y))
	}

	return nil
}

func TestPlainDropsTheStyling(t *testing.T) {
	app := reactea.New(&echoing{}, reactea.WithSize(40, 1))
	app.Start()

	if got := testkit.Plain(app); strings.ContainsRune(got, 0x1b) {
		t.Errorf("Plain left an escape in %q", got)
	}
}

func TestPlainDropsTerminalLinks(t *testing.T) {
	app := reactea.New(reactea.Func(func(*reactea.Ctx) string {
		return "\x1b]8;;https://example.com\x1b\\link\x1b]8;;\x1b\\"
	}), reactea.WithSize(40, 1))

	if got := testkit.Plain(app); got != "link" {
		t.Errorf("Plain returned %q, want link", got)
	}
}

func TestSendKeysSpellsKeysTheWayTheKeymapDoes(t *testing.T) {
	for _, key := range []string{"a", "A", "enter", "esc", "tab", "up", "ctrl+c", "shift+tab", "alt+x"} {
		seen := &echoing{}

		app := reactea.New(seen, reactea.WithSize(40, 1))
		app.Start()

		testkit.SendKeys(app, key)

		if seen.key != key {
			t.Errorf("SendKeys(%q) arrived as %q", key, seen.key)
		}
	}
}

func TestKeyRejectsInvalidNames(t *testing.T) {
	for _, key := range []string{"", "ctrl+", "unknown"} {
		t.Run(key, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Errorf("Key(%q) did not panic", key)
				}
			}()

			testkit.Key(key)
		})
	}
}

func TestClickLandsWhereItWasAimed(t *testing.T) {
	seen := &echoing{}

	app := reactea.New(seen, reactea.WithSize(40, 5))
	app.Start()

	testkit.Click(app, 3, 2)

	if seen.click != "32" {
		t.Errorf("the click arrived at %q, want 3,2", seen.click)
	}
}

func TestRenderAtResizesFirst(t *testing.T) {
	app := reactea.New(reactea.Func(func(ctx *reactea.Ctx) string {
		width, height := ctx.Size()

		return strings.Repeat("x", width) + "\n" + strings.Repeat("y", height)
	}), reactea.WithSize(1, 1))
	app.Start()

	if got := testkit.RenderAt(app, 6, 3); !strings.Contains(got, "xxxxxx") {
		t.Errorf("RenderAt(6, 3) drew %q", got)
	}
}
