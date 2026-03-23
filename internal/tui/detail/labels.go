package detail

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ifloresarg/gh-projects/internal/github"
)

type labelsLoadedMsg struct {
	labels []github.Label
	err    error
}

type labelToggledMsg struct {
	label  github.Label
	labels []github.Label
	added  bool
	err    error
}

type labelsModel struct {
	client    github.GitHubClient
	issue     *github.Issue
	allLabels []github.Label
	cursor    int
	loading   bool
	opMsg     string
	width     int
	height    int
	closing   bool
}

func newLabelsModel(client github.GitHubClient, issue *github.Issue, width, height int) labelsModel {
	return labelsModel{
		client:  client,
		issue:   issue,
		loading: true,
		width:   width,
		height:  height,
	}
}

func (m labelsModel) Init() tea.Cmd {
	if m.issue == nil {
		return func() tea.Msg {
			return labelsLoadedMsg{err: fmt.Errorf("issue detail unavailable")}
		}
	}

	owner := m.issue.RepoOwner
	repo := m.issue.RepoName

	return func() tea.Msg {
		labels, err := m.client.ListRepositoryLabels(owner, repo)
		return labelsLoadedMsg{labels: labels, err: err}
	}
}

func (m labelsModel) Update(msg tea.Msg) (labelsModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case labelsLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.opMsg = fmt.Sprintf("Error loading labels: %v", msg.err)
			return m, nil
		}

		m.allLabels = append([]github.Label(nil), msg.labels...)
		if len(m.allLabels) == 0 {
			m.cursor = 0
			m.opMsg = "No repository labels found"
			return m, nil
		}
		if m.cursor >= len(m.allLabels) {
			m.cursor = len(m.allLabels) - 1
		}
		m.opMsg = ""
		return m, nil
	case labelToggledMsg:
		if msg.err != nil {
			m.opMsg = msg.err.Error()
			return m, nil
		}

		if m.issue != nil {
			m.issue.Labels = append([]github.Label(nil), msg.labels...)
		}
		if msg.added {
			m.opMsg = fmt.Sprintf("Added label %q", msg.label.Name)
		} else {
			m.opMsg = fmt.Sprintf("Removed label %q", msg.label.Name)
		}
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			m.closing = true
			return m, nil
		case "j", "down":
			if !m.loading && m.cursor < len(m.allLabels)-1 {
				m.cursor++
			}
			return m, nil
		case "k", "up":
			if !m.loading && m.cursor > 0 {
				m.cursor--
			}
			return m, nil
		case "enter", " ":
			if m.loading || m.issue == nil || len(m.allLabels) == 0 || m.cursor < 0 || m.cursor >= len(m.allLabels) {
				return m, nil
			}

			label := m.allLabels[m.cursor]
			owner := m.issue.RepoOwner
			repo := m.issue.RepoName
			number := m.issue.Number
			if hasIssueLabel(m.issue.Labels, label) {
				return m, func() tea.Msg {
					err := m.client.RemoveLabel(owner, repo, number, label.ID)
					if err != nil {
						return labelToggledMsg{label: label, err: err}
					}

					labels := removeIssueLabel(m.issue.Labels, label)
					return labelToggledMsg{label: label, labels: labels}
				}
			}

			return m, func() tea.Msg {
				err := m.client.AddLabel(owner, repo, number, label.Name)
				if err != nil {
					return labelToggledMsg{label: label, added: true, err: err}
				}

				labels := appendIssueLabel(m.issue.Labels, label)
				return labelToggledMsg{label: label, labels: labels, added: true}
			}
		}
	}

	return m, nil
}

func (m labelsModel) View() string {
	status := ""
	if m.opMsg != "" {
		status = lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render(m.opMsg)
	}

	return strings.Join([]string{
		"Labels — j/k navigate, Enter/Space toggle, Esc cancel",
		m.visibleLabels(max(m.height-2, 1)),
		status,
	}, "\n")
}

func (m labelsModel) visibleLabels(limit int) string {
	if m.loading {
		return "Loading labels..."
	}
	if len(m.allLabels) == 0 {
		return "(no labels)"
	}

	start := 0
	if len(m.allLabels) > limit && m.cursor >= limit {
		start = m.cursor - limit + 1
	}
	end := min(start+limit, len(m.allLabels))
	lineWidth := max(m.width-2, 10)
	baseStyle := lipgloss.NewStyle().Width(lineWidth)
	selectedStyle := lipgloss.NewStyle().Width(lineWidth).Background(lipgloss.Color("12")).Foreground(lipgloss.Color("0"))

	lines := make([]string, 0, end-start)
	for i := start; i < end; i++ {
		label := m.allLabels[i]
		check := "[ ]"
		if m.issue != nil && hasIssueLabel(m.issue.Labels, label) {
			check = "[✓]"
		}

		dotColor := lipgloss.Color("7")
		if strings.TrimSpace(label.Color) != "" {
			dotColor = lipgloss.Color("#" + label.Color)
		}
		dot := lipgloss.NewStyle().Foreground(dotColor).Render("●")
		row := fmt.Sprintf("%s %s %s", check, dot, label.Name)
		if i == m.cursor {
			row = selectedStyle.Render(row)
		} else {
			row = baseStyle.Render(row)
		}
		lines = append(lines, row)
	}

	return strings.Join(lines, "\n")
}

func hasIssueLabel(labels []github.Label, label github.Label) bool {
	for _, existing := range labels {
		if label.ID != "" && existing.ID == label.ID {
			return true
		}
		if strings.EqualFold(existing.Name, label.Name) {
			return true
		}
	}

	return false
}

func appendIssueLabel(labels []github.Label, label github.Label) []github.Label {
	if hasIssueLabel(labels, label) {
		return append([]github.Label(nil), labels...)
	}

	return append(append([]github.Label(nil), labels...), label)
}

func removeIssueLabel(labels []github.Label, label github.Label) []github.Label {
	filtered := make([]github.Label, 0, len(labels))
	for _, existing := range labels {
		if label.ID != "" && existing.ID == label.ID {
			continue
		}
		if strings.EqualFold(existing.Name, label.Name) {
			continue
		}
		filtered = append(filtered, existing)
	}

	return filtered
}
