// Package testkit is the glue a test writes around an App: the terminal
// specifics that every application repeats and none of them differ on.
package testkit

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/Hayao0819/reactea/v2"
	"github.com/charmbracelet/x/ansi"
)

// Plain is the frame with its styling taken off, which is what an assertion
// about the screen wants to read.
func Plain(app *reactea.App) string {
	return ansi.Strip(app.View().Content)
}

// Lines is Plain split into rows, for a test that counts or indexes them.
func Lines(app *reactea.App) []string { return strings.Split(Plain(app), "\n") }

// SendKeys presses each key, spelled the way tea.KeyPressMsg.String does.
func SendKeys(app *reactea.App, keys ...string) {
	for _, key := range keys {
		app.Send(Key(key))
	}
}

// Key builds one press, spelled the way tea.KeyPressMsg.String does: modifier
// prefixes, then a name or a single character. A shifted letter is its capital,
// not "shift+" and the letter, which is how the terminal reports it. It panics
// when key is neither a known name nor one character.
func Key(key string) tea.KeyPressMsg {
	press := tea.KeyPressMsg{}

	for {
		prefix, mod, found := modifier(key)
		if !found {
			break
		}

		press.Mod |= mod
		key = key[len(prefix):]
	}

	if code, ok := named[key]; ok {
		press.Code = code

		return press
	}

	runes := []rune(key)
	if len(runes) != 1 {
		panic("testkit.Key: key must be a name or one character")
	}

	press.Code = runes[0]

	if press.Mod == 0 {
		press.Text = key
	}

	return press
}

func modifier(key string) (string, tea.KeyMod, bool) {
	for prefix, mod := range map[string]tea.KeyMod{
		"ctrl+":  tea.ModCtrl,
		"alt+":   tea.ModAlt,
		"shift+": tea.ModShift,
	} {
		if strings.HasPrefix(key, prefix) {
			return prefix, mod, true
		}
	}

	return "", 0, false
}

var named = map[string]rune{
	"enter":     tea.KeyEnter,
	"esc":       tea.KeyEscape,
	"tab":       tea.KeyTab,
	"space":     tea.KeySpace,
	"backspace": tea.KeyBackspace,
	"delete":    tea.KeyDelete,
	"up":        tea.KeyUp,
	"down":      tea.KeyDown,
	"left":      tea.KeyLeft,
	"right":     tea.KeyRight,
	"home":      tea.KeyHome,
	"end":       tea.KeyEnd,
	"pgup":      tea.KeyPgUp,
	"pgdown":    tea.KeyPgDown,
}

// Click presses the left button at a point in the app's own coordinates.
func Click(app *reactea.App, x, y int) {
	app.Send(tea.MouseClickMsg{X: x, Y: y, Button: tea.MouseLeft})
}

// Wheel scrolls at a point. A negative amount is up.
func Wheel(app *reactea.App, x, y, amount int) {
	button := tea.MouseWheelDown
	if amount < 0 {
		button, amount = tea.MouseWheelUp, -amount
	}

	for range amount {
		app.Send(tea.MouseWheelMsg{X: x, Y: y, Button: button})
	}
}

// RenderAt draws the app at a size without going through the terminal, for a
// test about layout rather than about input.
func RenderAt(app *reactea.App, width, height int) string {
	app.Send(tea.WindowSizeMsg{Width: width, Height: height})

	return Plain(app)
}
