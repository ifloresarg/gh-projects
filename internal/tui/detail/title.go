package detail

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ifloresarg/gh-projects/internal/github"
)

type titleResultMsg struct {
	title string
	err   error
}

type titleModel struct {
	client    github.GitHubClient
	issue     *github.Issue
	textinput textinput.Model
	opMsg     string
	width     int
	height    int
	closing   bool
	saving    bool
}

func newTitleModel(client github.GitHubClient, issue *github.Issue, width, height int) titleModel {
	ti := textinput.New()
	ti.Placeholder = "Issue title"
	ti.Width = max(width-8, 20)
	ti.CharLimit = 256
	if issue != nil {
		ti.SetValue(issue.Title)
		ti.CursorEnd()
	}

	return titleModel{
		client:    client,
		issue:     issue,
		textinput: ti,
		width:     width,
		height:    height,
	}
}

func (m titleModel) Init() tea.Cmd {
	return m.textinput.Focus()
}

func (m titleModel) Update(msg tea.Msg) (titleModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.textinput.Width = max(m.width-8, 20)
		return m, nil
	case titleResultMsg:
		m.saving = false
		if msg.err != nil {
			m.opMsg = fmt.Sprintf("Error: %v", msg.err)
			return m, nil
		}

		if m.issue != nil {
			m.issue.Title = msg.title
		}
		m.closing = true
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			m.closing = true
			return m, nil
		case "enter":
			if m.saving || m.issue == nil {
				return m, nil
			}

			newTitle := strings.TrimSpace(m.textinput.Value())
			if newTitle == "" {
				m.opMsg = "Title cannot be empty"
				return m, nil
			}
			if newTitle == m.issue.Title {
				m.closing = true
				return m, nil
			}

			m.saving = true
			m.opMsg = ""
			issueID := m.issue.ID
			return m, func() tea.Msg {
				err := m.client.UpdateIssueTitle(issueID, newTitle)
				if err != nil {
					return titleResultMsg{err: err}
				}
				return titleResultMsg{title: newTitle}
			}
		}
	}

	var cmd tea.Cmd
	m.textinput, cmd = m.textinput.Update(msg)
	if m.opMsg != "" && strings.TrimSpace(m.textinput.Value()) != "" {
		m.opMsg = ""
	}
	return m, cmd
}

func (m titleModel) View() string {
	status := lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render("Enter save · Esc cancel")
	if m.saving {
		status = lipgloss.NewStyle().Foreground(lipgloss.Color("12")).Render("Saving...")
	}
	if m.opMsg != "" {
		status = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Render(m.opMsg)
	}

	body := strings.Join([]string{
		"Edit Title",
		strings.Repeat("─", max(min(m.width-2, 16), 16)),
		m.textinput.View(),
		status,
	}, "\n")

	if m.width <= 0 || m.height <= 0 {
		return body
	}

	panel := lipgloss.NewStyle().
		Padding(1, 2).
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("8")).
		Width(min(max(m.width-6, 32), 72)).
		Render(body)

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, panel)
}
