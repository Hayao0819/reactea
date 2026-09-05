// Package render provides terminal display-cell fitting shared by containers.
package render

import (
	"strings"

	"charm.land/lipgloss/v2"
)

// Fit pads or trims content exactly to its component box.
func Fit(content string, width, height int) string {
	// Lipgloss treats zero maxima as unset, so zero-sized boxes produce their
	// empty output here.
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

// Left fits one line to exactly width terminal cells. It trims by display width
// and pads any cell left after clipping a double-width character.
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
