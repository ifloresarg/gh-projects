package detail

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ifloresarg/gh-projects/internal/github"
)

type assignModel struct {
	client   github.GitHubClient
	issue    *github.Issue
	allUsers []github.User
	filtered []github.User
	input    textinput.Model
	cursor   int
	loading  bool
	opMsg    string
	width    int
	height   int
	closing  bool
}

type collaboratorsLoadedMsg struct {
	users []github.User
	err   error
}

type assignResultMsg struct {
	assign    bool
	login     string
	assignees []github.User
	err       error
}

func newAssignModel(client github.GitHubClient, issue *github.Issue, width, height int) assignModel {
	input := textinput.New()
	input.Prompt = "Filter: "
	input.Placeholder = "type to search..."
	input.Focus()
	input.Width = max(width-8, 10)

	return assignModel{
		client:  client,
		issue:   issue,
		input:   input,
		loading: true,
		width:   width,
		height:  height,
	}
}

func (m assignModel) Init() tea.Cmd {
	if m.issue == nil {
		return func() tea.Msg {
			return collaboratorsLoadedMsg{err: fmt.Errorf("issue detail unavailable")}
		}
	}

	owner := m.issue.RepoOwner
	repo := m.issue.RepoName

	return func() tea.Msg {
		users, err := m.client.ListAssignableUsers(owner, repo)
		return collaboratorsLoadedMsg{users: users, err: err}
	}
}

func (m assignModel) Update(msg tea.Msg) (assignModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.input.Width = max(m.width-8, 10)
		return m, nil
	case collaboratorsLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.opMsg = fmt.Sprintf("Error loading collaborators: %v", msg.err)
			return m, nil
		}

		m.allUsers = append([]github.User(nil), msg.users...)
		m.filtered = filterUsers(m.allUsers, m.input.Value())
		m.cursor = 0
		m.opMsg = ""
		return m, nil
	case assignResultMsg:
		if msg.err != nil {
			m.opMsg = msg.err.Error()
			return m, nil
		}

		if m.issue != nil {
			m.issue.Assignees = append([]github.User(nil), msg.assignees...)
		}

		if msg.assign {
			m.opMsg = fmt.Sprintf("Assigned @%s", msg.login)
		} else {
			m.opMsg = fmt.Sprintf("Unassigned @%s", msg.login)
		}

		m.filtered = filterUsers(m.allUsers, m.input.Value())
		if len(m.filtered) == 0 {
			m.cursor = 0
		} else if m.cursor >= len(m.filtered) {
			m.cursor = len(m.filtered) - 1
		}
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			m.closing = true
			return m, nil
		case "down":
			if !m.loading && m.cursor < len(m.filtered)-1 {
				m.cursor++
			}
			return m, nil
		case "up":
			if !m.loading && m.cursor > 0 {
				m.cursor--
			}
			return m, nil
		case "j":
			if strings.TrimSpace(m.input.Value()) == "" {
				if !m.loading && m.cursor < len(m.filtered)-1 {
					m.cursor++
				}
				return m, nil
			}
		case "k":
			if strings.TrimSpace(m.input.Value()) == "" {
				if !m.loading && m.cursor > 0 {
					m.cursor--
				}
				return m, nil
			}
		case "enter":
			if m.loading || m.issue == nil {
				return m, nil
			}

			if len(m.filtered) > 0 && m.cursor >= 0 && m.cursor < len(m.filtered) {
				user := m.filtered[m.cursor]
				owner := m.issue.RepoOwner
				repo := m.issue.RepoName
				number := m.issue.Number

				if hasAssignee(m.issue.Assignees, user.Login) {
					return m, func() tea.Msg {
						err := m.client.UnassignUser(owner, repo, number, user.Login)
						if err != nil {
							return assignResultMsg{assign: false, login: user.Login, err: err}
						}

						newAssignees := removeAssignee(m.issue.Assignees, user.Login)
						return assignResultMsg{assign: false, login: user.Login, assignees: newAssignees}
					}
				}

				return m, func() tea.Msg {
					err := m.client.AssignUser(owner, repo, number, user.Login)
					if err != nil {
						return assignResultMsg{assign: true, login: user.Login, err: err}
					}

					newAssignees := addAssignee(m.issue.Assignees, user)
					return assignResultMsg{assign: true, login: user.Login, assignees: newAssignees}
				}
			}

			login := strings.TrimSpace(m.input.Value())
			if len(m.filtered) == 0 && login != "" {
				owner := m.issue.RepoOwner
				repo := m.issue.RepoName
				number := m.issue.Number
				m.opMsg = fmt.Sprintf("Assigning @%s...", login)
				return m, func() tea.Msg {
					err := m.client.AssignUser(owner, repo, number, login)
					if err != nil {
						return assignResultMsg{assign: true, login: login, err: err}
					}

					newAssignees := addAssignee(m.issue.Assignees, github.User{Login: login})
					return assignResultMsg{assign: true, login: login, assignees: newAssignees}
				}
			}

			return m, nil
		case " ":
			if m.loading || m.issue == nil {
				return m, nil
			}
			if len(m.filtered) == 0 || m.cursor < 0 || m.cursor >= len(m.filtered) {
				return m, nil
			}

			user := m.filtered[m.cursor]
			owner := m.issue.RepoOwner
			repo := m.issue.RepoName
			number := m.issue.Number

			if hasAssignee(m.issue.Assignees, user.Login) {
				return m, func() tea.Msg {
					err := m.client.UnassignUser(owner, repo, number, user.Login)
					if err != nil {
						return assignResultMsg{assign: false, login: user.Login, err: err}
					}

					newAssignees := removeAssignee(m.issue.Assignees, user.Login)
					return assignResultMsg{assign: false, login: user.Login, assignees: newAssignees}
				}
			}

			return m, func() tea.Msg {
				err := m.client.AssignUser(owner, repo, number, user.Login)
				if err != nil {
					return assignResultMsg{assign: true, login: user.Login, err: err}
				}

				newAssignees := addAssignee(m.issue.Assignees, user)
				return assignResultMsg{assign: true, login: user.Login, assignees: newAssignees}
			}
		}

		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		m.filtered = filterUsers(m.allUsers, m.input.Value())
		m.cursor = 0
		return m, cmd
	}

	return m, nil
}

func (m assignModel) View() string {
	lineWidth := max(m.width-2, 10)
	status := ""
	if m.opMsg != "" {
		status = lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render(m.opMsg)
	}

	return strings.Join([]string{
		"Assignees — ↑/↓ navigate, Enter/Space toggle, type to filter, Esc cancel",
		m.visibleUsers(max(m.height-6, 1)),
		strings.Repeat("─", lineWidth),
		m.input.View(),
		status,
	}, "\n")
}

func (m assignModel) visibleUsers(limit int) string {
	if m.loading {
		return "Loading collaborators..."
	}
	if len(m.allUsers) == 0 {
		return "(no assignable users found)"
	}

	query := strings.TrimSpace(m.input.Value())
	if len(m.filtered) == 0 && query != "" {
		return fmt.Sprintf("No matches — press Enter to assign @%s directly", query)
	}

	start := 0
	if len(m.filtered) > limit && m.cursor >= limit {
		start = m.cursor - limit + 1
	}
	end := min(start+limit, len(m.filtered))
	lineWidth := max(m.width-2, 10)
	baseStyle := lipgloss.NewStyle().Width(lineWidth)
	selectedStyle := lipgloss.NewStyle().Width(lineWidth).Background(lipgloss.Color("12")).Foreground(lipgloss.Color("0"))

	lines := make([]string, 0, end-start)
	for i := start; i < end; i++ {
		u := m.filtered[i]
		check := "[ ]"
		if m.issue != nil && hasAssignee(m.issue.Assignees, u.Login) {
			check = "[✓]"
		}

		display := fmt.Sprintf("@%s", u.Login)
		if strings.TrimSpace(u.Name) != "" {
			display = fmt.Sprintf("@%s — %s", u.Login, u.Name)
		}

		row := fmt.Sprintf("%s %s", check, display)
		if i == m.cursor {
			row = selectedStyle.Render(row)
		} else {
			row = baseStyle.Render(row)
		}
		lines = append(lines, row)
	}

	return strings.Join(lines, "\n")
}

func filterUsers(users []github.User, query string) []github.User {
	if query == "" {
		return users
	}

	q := strings.ToLower(query)
	filtered := make([]github.User, 0, len(users))
	for _, u := range users {
		if strings.Contains(strings.ToLower(u.Login), q) || strings.Contains(strings.ToLower(u.Name), q) {
			filtered = append(filtered, u)
		}
	}

	return filtered
}

func hasAssignee(assignees []github.User, login string) bool {
	for _, assignee := range assignees {
		if strings.EqualFold(assignee.Login, login) {
			return true
		}
	}

	return false
}

func removeAssignee(assignees []github.User, login string) []github.User {
	filtered := make([]github.User, 0, len(assignees))
	for _, assignee := range assignees {
		if strings.EqualFold(assignee.Login, login) {
			continue
		}
		filtered = append(filtered, assignee)
	}

	return filtered
}

func addAssignee(assignees []github.User, user github.User) []github.User {
	if hasAssignee(assignees, user.Login) {
		return assignees
	}

	return append(append([]github.User(nil), assignees...), user)
}
