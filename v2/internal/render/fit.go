// Package render holds the drawing rules the containers share.
package render

import "charm.land/lipgloss/v2"

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
