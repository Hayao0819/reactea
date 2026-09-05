// Package render holds the drawing rules the containers share, in terminal
// display cells rather than bytes or runes.
package render

import (
	"strings"

	"charm.land/lipgloss/v2"
)

// Fit pads or trims content to the box its component was given. Containers apply
// it before composing, so a child that draws short or long shifts nothing else.
func Fit(content string, width, height int) string {
	// Lipgloss reads MaxWidth(0) and MaxHeight(0) as unset, so a component handed
	// no cells has to be emptied here rather than trimmed there.
	if width <= 0 || height <= 0 {
		return ""
	}

	if actual, rows := lipgloss.Size(content); actual == width && rows == height {
		return content
	}

	return lipgloss.NewStyle().
		Width(width).Height(height).
		MaxWidth(width).MaxHeight(height).
		Render(content)
}

// Left fits one line to exactly width cells, counting what the terminal shows
// rather than bytes or escape sequences. Padding comes after the trim as well as
// instead of it: a double-width character cannot half-fit, so trimming to an odd
// width leaves a cell over.
func Left(text string, width int) string {
	if width <= 0 {
		return ""
	}

	trimmed := Clip(text, width)

	if gap := width - lipgloss.Width(trimmed); gap > 0 {
		return trimmed + strings.Repeat(" ", gap)
	}

	return trimmed
}

// Right is Left against the other edge.
func Right(text string, width int) string {
	if width <= 0 {
		return ""
	}

	trimmed := Clip(text, width)

	if gap := width - lipgloss.Width(trimmed); gap > 0 {
		return strings.Repeat(" ", gap) + trimmed
	}

	return trimmed
}

// Clip trims one line and leaves a short one short.
func Clip(text string, width int) string {
	if width <= 0 {
		return ""
	}

	return lipgloss.NewStyle().MaxWidth(width).Render(text)
}
