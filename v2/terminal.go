package reactea

import (
	"image/color"

	tea "charm.land/bubbletea/v2"
)

// Bubbletea v2 moved the alt-screen, the window title, mouse mode and the
// terminal colours out of commands and onto the view the root model returns.
// They are still state, though: a component decides them when it mounts, not
// sixty times a second. So reactea takes them as commands, holds them on the
// App and puts them on every frame — which is what keeps Render free to be a
// function of the component's state and nothing else.
type terminalMsg struct{ apply func(*tea.View) }

func terminalCmd(apply func(*tea.View)) tea.Cmd {
	return func() tea.Msg { return terminalMsg{apply: apply} }
}

// EnterAltScreen puts the program in the alternate screen buffer.
func EnterAltScreen() tea.Msg {
	return terminalMsg{apply: func(view *tea.View) { view.AltScreen = true }}
}

// ExitAltScreen returns to the normal screen buffer.
func ExitAltScreen() tea.Msg {
	return terminalMsg{apply: func(view *tea.View) { view.AltScreen = false }}
}

// SetWindowTitle sets the terminal window title.
func SetWindowTitle(title string) tea.Cmd {
	return terminalCmd(func(view *tea.View) { view.WindowTitle = title })
}

// SetMouseMode asks the terminal for mouse reporting.
func SetMouseMode(mode tea.MouseMode) tea.Cmd {
	return terminalCmd(func(view *tea.View) { view.MouseMode = mode })
}

// SetReportFocus turns focus reporting on or off.
func SetReportFocus(on bool) tea.Cmd {
	return terminalCmd(func(view *tea.View) { view.ReportFocus = on })
}

// SetBackgroundColor sets the terminal background. Pass nil to reset it.
func SetBackgroundColor(colour color.Color) tea.Cmd {
	return terminalCmd(func(view *tea.View) { view.BackgroundColor = colour })
}

// SetForegroundColor sets the terminal foreground. Pass nil to reset it.
func SetForegroundColor(colour color.Color) tea.Cmd {
	return terminalCmd(func(view *tea.View) { view.ForegroundColor = colour })
}

// SetKeyboardEnhancements asks the terminal for richer key reporting.
func SetKeyboardEnhancements(enhancements tea.KeyboardEnhancements) tea.Cmd {
	return terminalCmd(func(view *tea.View) { view.KeyboardEnhancements = enhancements })
}
