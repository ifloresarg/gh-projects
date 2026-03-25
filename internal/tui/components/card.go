package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/ifloresarg/gh-projects/internal/github"
)

// Card renders a single project item as a styled card.
// focused: whether this card is the active cursor card
// width: the column width minus borders (use columnWidth from caller)
func Card(item github.ProjectItem, focused bool, width int, showLabels bool) string {
	var titlePrefix string
	switch item.Type {
	case "PullRequest":
		titlePrefix = fmt.Sprintf("PR #%d - ", item.ContentNumber)
	case "DraftIssue":
		titlePrefix = "(draft) - "
	default:
		titlePrefix = fmt.Sprintf("#%d - ", item.ContentNumber)
	}

	wrapped := lipgloss.NewStyle().Width(width - 2).Render(titlePrefix + item.Title)
	titleLineSlice := strings.Split(wrapped, "\n")
	for i := range titleLineSlice {
		titleLineSlice[i] = strings.TrimRight(titleLineSlice[i], " ")
	}
	if len(titleLineSlice) > 3 {
		titleLineSlice = titleLineSlice[:3]
		titleLineSlice[2] = truncate(strings.TrimRight(titleLineSlice[2], " ")+"…", width-2)
	}
	titleStyle := lipgloss.NewStyle()
	if focused {
		titleStyle = titleStyle.Foreground(lipgloss.Color("15"))
	}
	titleLines := make([]string, 0, len(titleLineSlice))
	for _, line := range titleLineSlice {
		titleLines = append(titleLines, titleStyle.Render(line))
	}

	var assignees []string
	var labels []string
	var prBadges []string
	var commentsCount int

	if c, ok := item.Content.(*github.Issue); ok {
		commentsCount = c.CommentsCount
		for i, a := range c.Assignees {
			if i >= 3 {
				break
			}
			if len(a.Login) > 0 {
				assignees = append(assignees, "@"+a.Login)
			}
		}
		if len(c.Assignees) > 3 {
			assignees = append(assignees, fmt.Sprintf("+%d", len(c.Assignees)-3))
		}

		if showLabels {
			for i, l := range c.Labels {
				if i >= 4 {
					break
				}
				color := "#" + l.Color
				labelText := lipgloss.NewStyle().Foreground(lipgloss.Color(color)).Render(l.Name)
				labels = append(labels, labelText)
			}
			if len(c.Labels) > 4 {
				labels = append(labels, fmt.Sprintf("+%d", len(c.Labels)-4))
			}
		}

		for i, pr := range c.LinkedPRs {
			if i >= 2 {
				break
			}
			prColor := "#888888"
			switch pr.State {
			case "OPEN":
				prColor = "#51CF66"
			case "MERGED":
				prColor = "#B197FC"
			case "CLOSED":
				prColor = "#FF6B6B"
			}
			prBadgeText := fmt.Sprintf("PR#%d", pr.Number)
			prBadge := lipgloss.NewStyle().Foreground(lipgloss.Color(prColor)).Render(prBadgeText)
			prBadges = append(prBadges, prBadge)
		}
		if len(c.LinkedPRs) > 2 {
			prBadges = append(prBadges, fmt.Sprintf("+%d", len(c.LinkedPRs)-2))
		}
	} else if c, ok := item.Content.(*github.PullRequest); ok {
		commentsCount = c.CommentsCount
	}

	var typeBadge string
	if item.TypeValue != "" {
		var typeColor lipgloss.Color
		switch item.TypeValue {
		case "Bug":
			typeColor = lipgloss.Color("#FF6B6B")
		case "Feature":
			typeColor = lipgloss.Color("#51CF66")
		case "Task":
			typeColor = lipgloss.Color("#FFD43B")
		default:
			typeColor = lipgloss.Color("#868E96")
		}
		typeBadge = lipgloss.NewStyle().Foreground(typeColor).Render("[" + item.TypeValue + "]")
	}

	var commentBadge string
	if commentsCount > 0 {
		commentBadge = lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render(fmt.Sprintf("💬 %d", commentsCount))
	}

	var secondLine string
	if typeBadge != "" || commentBadge != "" || len(prBadges) > 0 || len(assignees) > 0 || len(labels) > 0 {
		var parts []string
		if typeBadge != "" {
			parts = append(parts, typeBadge)
		}
		if commentBadge != "" {
			parts = append(parts, commentBadge)
		}
		if len(prBadges) > 0 {
			parts = append(parts, strings.Join(prBadges, " "))
		}
		if len(assignees) > 0 {
			parts = append(parts, strings.Join(assignees, " "))
		}
		if len(labels) > 0 {
			labelLine := strings.Join(labels, " ")
			innerWidth := width - 2
			if lipgloss.Width(labelLine) > innerWidth {
				truncated := ""
				for i := len(labels); i > 0; i-- {
					candidate := strings.Join(labels[:i], " ") + " ..."
					if lipgloss.Width(candidate) <= innerWidth {
						truncated = candidate
						break
					}
				}
				if truncated == "" {
					truncated = "..."
				}
				labelLine = truncated
			}
			parts = append(parts, labelLine)
		}
		secondLine = strings.Join(parts, "  ")
	}

	var lines []string
	lines = append(lines, titleLines...)
	if secondLine != "" {
		lines = append(lines, secondLine)
	}

	content := strings.Join(lines, "\n")

	borderStyle := lipgloss.NewStyle().
		Width(width).
		Padding(0, 1)

	if focused {
		borderStyle = borderStyle.
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("12"))
	} else {
		borderStyle = borderStyle.
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("8"))
	}

	return borderStyle.Render(content)
}

func truncate(s string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}

	if lipgloss.Width(s) <= maxWidth {
		return s
	}

	for len(s) > 0 {
		s = s[:len(s)-1]
		if lipgloss.Width(s+"…") <= maxWidth {
			return s + "…"
		}
	}

	return "…"
}
