package detail

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/ifloresarg/gh-projects/internal/github"
)

const commentsInputHeight = 5

type commentsLoadedMsg struct {
	comments []github.Comment
	err      error
}

type commentSubmittedMsg struct {
	body string
	err  error
}

type commentsModel struct {
	client     github.GitHubClient
	issue      *github.Issue
	comments   []github.Comment
	viewport   viewport.Model
	textarea   textarea.Model
	loading    bool
	err        error
	submitErr  string
	width      int
	height     int
	active     bool
	submitting bool
}

func newCommentsModel(client github.GitHubClient, issue *github.Issue, width, height int) commentsModel {
	vp := viewport.New(0, 0)
	ta := textarea.New()
	ta.Placeholder = "Write a comment…"
	ta.ShowLineNumbers = false
	ta.SetHeight(commentsInputHeight)
	ta.SetWidth(max(width-2, 20))

	m := commentsModel{
		client:   client,
		issue:    issue,
		viewport: vp,
		textarea: ta,
		width:    width,
		height:   height,
	}
	m.syncSize()
	return m
}

func (m *commentsModel) Init() tea.Cmd {
	m.active = true
	m.loading = true
	m.err = nil
	m.submitErr = ""
	m.submitting = false
	m.comments = nil
	m.textarea.Reset()
	m.syncSize()

	focusCmd := m.textarea.Focus()
	if m.issue == nil {
		m.loading = false
		m.err = fmt.Errorf("issue content unavailable")
		m.refreshViewport()
		return focusCmd
	}

	owner := m.issue.RepoOwner
	repo := m.issue.RepoName
	number := m.issue.Number

	return tea.Batch(
		focusCmd,
		func() tea.Msg {
			comments, err := m.client.GetIssueComments(owner, repo, number)
			return commentsLoadedMsg{comments: comments, err: err}
		},
	)
}

func (m commentsModel) Update(msg tea.Msg) (commentsModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.syncSize()
		m.refreshViewport()
		return m, nil
	case commentsLoadedMsg:
		m.loading = false
		m.err = msg.err
		if msg.err == nil {
			m.comments = msg.comments
		}
		m.refreshViewport()
		return m, nil
	case commentSubmittedMsg:
		m.submitting = false
		if msg.err != nil {
			m.submitErr = msg.err.Error()
			return m, nil
		}

		m.submitErr = ""
		m.comments = append(m.comments, github.Comment{
			Author:    github.User{Login: "you"},
			CreatedAt: time.Now(),
			Body:      msg.body,
		})
		m.textarea.Reset()
		m.refreshViewport()
		m.viewport.GotoBottom()
		return m, m.textarea.Focus()
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			m.active = false
			return m, nil
		case "ctrl+s", "ctrl+enter":
			body := strings.TrimSpace(m.textarea.Value())
			if body == "" {
				m.submitErr = "Comment cannot be empty"
				return m, nil
			}
			if m.issue == nil || m.submitting {
				return m, nil
			}

			m.submitErr = ""
			m.submitting = true
			owner := m.issue.RepoOwner
			repo := m.issue.RepoName
			number := m.issue.Number
			return m, func() tea.Msg {
				err := m.client.AddComment(owner, repo, number, body)
				return commentSubmittedMsg{body: body, err: err}
			}
		}
	}

	var cmd tea.Cmd
	m.textarea, cmd = m.textarea.Update(msg)
	if m.submitErr != "" && strings.TrimSpace(m.textarea.Value()) != "" {
		m.submitErr = ""
	}
	return m, cmd
}

func (m commentsModel) View() string {
	title := "Comments"
	if m.issue != nil {
		title = fmt.Sprintf("Comments · Issue #%d", m.issue.Number)
	}

	divider := strings.Repeat("─", max(m.width-2, 10))
	meta := lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render("Esc closes · Ctrl+S/Ctrl+Enter submits")

	status := lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render("Enter inserts newline")
	if m.submitting {
		status = lipgloss.NewStyle().Foreground(lipgloss.Color("12")).Render("Submitting comment...")
	}
	if m.submitErr != "" {
		status = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Render(m.submitErr)
	}

	return strings.Join([]string{
		title,
		meta,
		"",
		m.viewport.View(),
		divider,
		m.textarea.View(),
		status,
	}, "\n")
}

func (m *commentsModel) syncSize() {
	m.textarea.SetWidth(max(m.width-2, 20))
	m.textarea.SetHeight(commentsInputHeight)
	m.viewport.Width = max(m.width-2, 0)
	m.viewport.Height = max(m.height-(commentsInputHeight+6), 3)
}

func (m *commentsModel) refreshViewport() {
	m.viewport.SetContent(m.renderCommentsContent())
	if len(m.comments) > 0 {
		m.viewport.GotoBottom()
	}
}

func (m commentsModel) renderCommentsContent() string {
	if m.loading {
		return "Loading comments..."
	}
	if m.err != nil {
		return fmt.Sprintf("Error loading comments: %v", m.err)
	}
	if len(m.comments) == 0 {
		return "_No comments yet._"
	}

	divider := strings.Repeat("─", max(m.viewport.Width-2, 10))
	parts := make([]string, 0, len(m.comments))
	for _, comment := range m.comments {
		body := strings.TrimSpace(comment.Body)
		if body == "" {
			body = "_Empty comment._"
		}

		renderedBody, err := glamour.Render(body, "dark")
		if err != nil {
			renderedBody = body
		}

		timestamp := comment.CreatedAt.Local().Format("2006-01-02 15:04")
		parts = append(parts, strings.Join([]string{
			fmt.Sprintf("@%s · %s", comment.Author.Login, timestamp),
			divider,
			strings.TrimRight(renderedBody, "\n"),
		}, "\n"))
	}

	return strings.Join(parts, "\n\n")
}
