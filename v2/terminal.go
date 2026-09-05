package reactea

import (
	"image/color"

	tea "charm.land/bubbletea/v2"
)

// Bubbletea v2 moved these onto the view. Reactea keeps them as commands so
// Render stays a function of the component's state.
type terminalMsg struct{ apply func(*tea.View) }

func terminalCmd(apply func(*tea.View)) tea.Cmd {
	return func() tea.Msg { return terminalMsg{apply: apply} }
}

// EnterAltScreen puts the program in the alternate screen buffer.
func EnterAltScreen() tea.Cmd {
	return terminalCmd(func(view *tea.View) { view.AltScreen = true })
}

// ExitAltScreen returns to the normal screen buffer.
func ExitAltScreen() tea.Cmd {
	return terminalCmd(func(view *tea.View) { view.AltScreen = false })
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

// SetProgressBar shows a progress bar in the terminal's progress area. Pass nil
// to take it away.
func SetProgressBar(bar *tea.ProgressBar) tea.Cmd {
	return terminalCmd(func(view *tea.View) { view.ProgressBar = bar })
}

// SetBracketedPaste turns bracketed paste on or off. It is on by default.
func SetBracketedPaste(on bool) tea.Cmd {
	return terminalCmd(func(view *tea.View) { view.DisableBracketedPasteMode = !on })
}

// SetKeyboardEnhancements asks the terminal for richer key reporting.
func SetKeyboardEnhancements(enhancements tea.KeyboardEnhancements) tea.Cmd {
	return terminalCmd(func(view *tea.View) { view.KeyboardEnhancements = enhancements })
}
