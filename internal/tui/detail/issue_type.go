package detail

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ifloresarg/gh-projects/internal/github"
)

type issueTypesLoadedMsg struct {
	types []github.IssueType
	err   error
}

type issueTypeResultMsg struct {
	typeID   *string
	typeName string
	err      error
}

type issueTypeModel struct {
	client   github.GitHubClient
	issue    *github.Issue
	allTypes []github.IssueType
	cursor   int
	loading  bool
	opMsg    string
	width    int
	height   int
	closing  bool
}

func newIssueTypeModel(client github.GitHubClient, issue *github.Issue, width, height int) issueTypeModel {
	return issueTypeModel{
		client:  client,
		issue:   issue,
		loading: true,
		width:   width,
		height:  height,
	}
}

func (m issueTypeModel) Init() tea.Cmd {
	if m.issue == nil {
		return func() tea.Msg {
			return issueTypesLoadedMsg{err: fmt.Errorf("issue detail unavailable")}
		}
	}

	owner := m.issue.RepoOwner
	repo := m.issue.RepoName

	return func() tea.Msg {
		types, err := m.client.ListIssueTypes(owner, repo)
		return issueTypesLoadedMsg{types: types, err: err}
	}
}

func (m issueTypeModel) Update(msg tea.Msg) (issueTypeModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case issueTypesLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.opMsg = fmt.Sprintf("Error loading issue types: %v", msg.err)
			return m, nil
		}

		m.allTypes = append([]github.IssueType(nil), msg.types...)
		if len(m.allTypes) == 0 {
			m.cursor = 0
			m.opMsg = "No issue types found"
			return m, nil
		}
		if m.cursor >= len(m.allTypes) {
			m.cursor = len(m.allTypes) - 1
		}
		m.opMsg = ""
		return m, nil
	case issueTypeResultMsg:
		if msg.err != nil {
			m.opMsg = fmt.Sprintf("Error: %v", msg.err)
			return m, nil
		}

		if m.issue != nil {
			m.issue.IssueType = msg.typeName
		}
		m.opMsg = ""
		m.closing = true
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			m.closing = true
			return m, nil
		case "j", "down":
			if !m.loading && m.cursor < len(m.allTypes)-1 {
				m.cursor++
			}
			return m, nil
		case "k", "up":
			if !m.loading && m.cursor > 0 {
				m.cursor--
			}
			return m, nil
		case "enter", " ":
			if m.loading || m.issue == nil || len(m.allTypes) == 0 || m.cursor < 0 || m.cursor >= len(m.allTypes) {
				return m, nil
			}

			selected := m.allTypes[m.cursor]
			if selected.Name == m.issue.IssueType {
				return m, func() tea.Msg {
					err := m.client.UpdateIssueType(m.issue.ID, nil)
					if err != nil {
						return issueTypeResultMsg{err: err}
					}
					return issueTypeResultMsg{typeID: nil, typeName: ""}
				}
			}

			return m, func() tea.Msg {
				err := m.client.UpdateIssueType(m.issue.ID, &selected.ID)
				if err != nil {
					return issueTypeResultMsg{err: err}
				}
				return issueTypeResultMsg{typeID: &selected.ID, typeName: selected.Name}
			}
		}
	}

	return m, nil
}

func (m issueTypeModel) View() string {
	status := ""
	if m.opMsg != "" {
		status = lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render(m.opMsg)
	}

	return strings.Join([]string{
		"Issue Type — j/k navigate, Enter select/clear, Esc cancel",
		m.visibleIssueTypes(max(m.height-2, 1)),
		status,
	}, "\n")
}

func (m issueTypeModel) visibleIssueTypes(limit int) string {
	if m.loading {
		return "Loading issue types..."
	}
	if len(m.allTypes) == 0 {
		return "(no issue types)"
	}

	start := 0
	if len(m.allTypes) > limit && m.cursor >= limit {
		start = m.cursor - limit + 1
	}
	end := min(start+limit, len(m.allTypes))
	lineWidth := max(m.width-2, 10)
	baseStyle := lipgloss.NewStyle().Width(lineWidth)
	selectedStyle := lipgloss.NewStyle().Width(lineWidth).Background(lipgloss.Color("12")).Foreground(lipgloss.Color("0"))

	lines := make([]string, 0, end-start)
	for i := start; i < end; i++ {
		issueType := m.allTypes[i]
		check := "[ ]"
		if m.issue != nil && m.issue.IssueType == issueType.Name {
			check = "[•]"
		}

		row := fmt.Sprintf("%s %s", check, issueType.Name)
		if i == m.cursor {
			row = selectedStyle.Render(row)
		} else {
			row = baseStyle.Render(row)
		}
		lines = append(lines, row)
	}

	return strings.Join(lines, "\n")
}
