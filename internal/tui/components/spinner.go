package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// FullScreenSpinner renders a centered loading message for full-screen loading states.
// msg is the text to display (e.g., "Loading projects...", "Fetching board...")
// width and height define the terminal area to center within.
// If width or height <= 0, returns msg directly without centering.
func FullScreenSpinner(msg string, width, height int) string {
	if width <= 0 || height <= 0 {
		return msg
	}
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, msg)
}

// StatusBar renders the bottom status bar with project context and keybinding hints.
// projectName and owner: current project context (empty string if not yet selected).
// opMsg: current operation message (e.g., "Moving card...", "") — shown as inline status.
// width: terminal width.
//
// Layout (left to right):
// - Left: [owner/projectName] (only if projectName non-empty)
// - Center/fill: spaces
// - Right: ? Help | q Quit (keybinding hints)
// - If opMsg non-empty: ● opMsg inserted between left and hints
func StatusBar(projectName, owner, opMsg string, width int) string {
	baseStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("236")).
		Foreground(lipgloss.Color("7")).
		Width(width)

	hintStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("236")).
		Foreground(lipgloss.Color("8"))

	hints := hintStyle.Render("? Help | q Quit")

	var left string
	if projectName != "" {
		left = fmt.Sprintf("[%s/%s]", owner, projectName)
	}

	var middle string
	if opMsg != "" {
		middle = "● " + opMsg
	}

	leftWidth := lipgloss.Width(left)
	middleWidth := lipgloss.Width(middle)
	hintsWidth := lipgloss.Width(hints)

	totalUsed := leftWidth + middleWidth + hintsWidth
	if middleWidth > 0 {
		totalUsed += 2
	}

	var content string
	if totalUsed >= width {
		if middle != "" {
			content = left + "  " + middle + "  " + hints
		} else {
			content = left + "  " + hints
		}
	} else {
		padding := strings.Repeat(" ", width-totalUsed)
		if middle != "" {
			content = left + "  " + middle + padding + hints
		} else {
			content = left + padding + hints
		}
	}

	if lipgloss.Width(content) > width {
		runes := []rune(content)
		if len(runes) > width {
			content = string(runes[:width])
		}
	} else if lipgloss.Width(content) < width {
		diff := width - lipgloss.Width(content)
		content += strings.Repeat(" ", diff)
	}

	return baseStyle.Render(content)
}
