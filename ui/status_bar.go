package ui

import (
	"github.com/charmbracelet/lipgloss"
)

// RenderStatus renders a status message with an icon.
// kind: 0=success, 1=error, 2=info
func RenderStatus(text string, kind int, w int) string {
	var fg lipgloss.Color
	var prefix string
	switch kind {
	case 1:
		fg = ColorStatusErrFg
		prefix = "✗ "
	case 2:
		fg = lipgloss.Color("33")
		prefix = "ℹ "
	default:
		fg = ColorStatusOkFg
		prefix = "✓ "
	}

	msg := prefix + text
	plain := padRight(truncateStr(msg, w), w)
	return lipgloss.NewStyle().Foreground(fg).Render(plain)
}
