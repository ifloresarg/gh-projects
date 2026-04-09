package detail

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/ifloresarg/gh-projects/internal/editor"
	"github.com/ifloresarg/gh-projects/internal/github"
)

type issueLoadedMsg struct {
	issue *github.Issue
	err   error
}

type linkedPRsLoadedMsg struct {
	prs []github.PullRequest
	err error
}

type issueCloseResultMsg struct {
	err error
}

type issueReopenResultMsg struct {
	err error
}

type addPRResultMsg struct {
	prNumber int
	err      error
}

type linkPRResultMsg struct {
	prNumber int
	err      error
}

type editBodyResultMsg struct {
	body    string
	changed bool
	err     error
}

type Model struct {
	client        github.GitHubClient
	item          github.ProjectItem
	projectID     string
	issue         *github.Issue
	linkedPRs     []github.PullRequest
	assign        assignModel
	labels        labelsModel
	issueType     issueTypeModel
	title         titleModel
	comments      commentsModel
	addPRInput    textinput.Model
	prPicker      prPickerModel
	viewport      viewport.Model
	loading       bool
	err           error
	opHint        string
	showAssign    bool
	showLabels    bool
	showIssueType bool
	showTitle     bool
	showComments  bool
	showAddPR     bool
	showPRPicker  bool
	editingBody   bool
	width         int
	height        int
	confirming    bool
	confirmMsg    string
	confirmAction string
	opMsg         string
}

func New(client github.GitHubClient, item github.ProjectItem, projectID string, repos []RepoRef, mergedPRWindow time.Duration, prFetchLimit int) Model {
	vp := viewport.New(0, 0)
	issue, _ := item.Content.(*github.Issue)
	prInput := textinput.New()
	prInput.Placeholder = "owner/repo#number or PR URL"
	prInput.Width = 50

	return Model{
		client:     client,
		item:       item,
		projectID:  projectID,
		issue:      issue,
		comments:   newCommentsModel(client, issue, 0, 0),
		addPRInput: prInput,
		prPicker:   newPRPicker(client, repos, mergedPRWindow, prFetchLimit),
		viewport:   vp,
		loading:    item.Type == "Issue",
	}
}

func (m Model) UpdatedItem() github.ProjectItem {
	return m.item
}

func (m Model) IsInputFocused() bool {
	return m.showComments || m.showAddPR || m.showTitle || m.showAssign || m.showLabels || m.showIssueType || m.showPRPicker
}

func (m Model) Init() tea.Cmd {
	if m.item.Type != "Issue" {
		return nil
	}

	issue, ok := m.item.Content.(*github.Issue)
	if !ok || issue == nil {
		return func() tea.Msg {
			return issueLoadedMsg{err: fmt.Errorf("issue content unavailable")}
		}
	}

	owner := issue.RepoOwner
	repo := issue.RepoName
	number := issue.Number

	return tea.Batch(
		func() tea.Msg {
			loaded, err := m.client.GetIssue(owner, repo, number)
			return issueLoadedMsg{issue: loaded, err: err}
		},
		func() tea.Msg {
			prs, err := m.client.GetLinkedPullRequests(owner, repo, number)
			return linkedPRsLoadedMsg{prs: prs, err: err}
		},
	)
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	refreshLabels := func() {
		if m.issue == nil {
			return
		}

		if m.labels.issue != nil {
			m.issue.Labels = append([]github.Label(nil), m.labels.issue.Labels...)
		}
		if issue, ok := m.item.Content.(*github.Issue); ok && issue != nil {
			issue.Labels = append([]github.Label(nil), m.issue.Labels...)
		}
		m.viewport.SetContent(m.renderIssueContent())
	}

	if m.showLabels {
		var cmd tea.Cmd
		m.labels, cmd = m.labels.Update(msg)
		refreshLabels()
		if m.labels.closing {
			m.showLabels = false
			m.labels.closing = false
		}
		return m, cmd
	}

	if m.showIssueType {
		var cmd tea.Cmd
		m.issueType, cmd = m.issueType.Update(msg)
		if m.issueType.issue != nil && m.issue != nil {
			m.issue.IssueType = m.issueType.issue.IssueType
			if issue, ok := m.item.Content.(*github.Issue); ok && issue != nil {
				issue.IssueType = m.issueType.issue.IssueType
			}
			m.item.TypeValue = m.issueType.issue.IssueType
			m.viewport.SetContent(m.renderIssueContent())
		}
		if m.issueType.closing {
			m.showIssueType = false
			m.issueType.closing = false
		}
		return m, cmd
	}

	if m.showTitle {
		var cmd tea.Cmd
		m.title, cmd = m.title.Update(msg)
		if m.title.issue != nil && m.issue != nil {
			m.issue.Title = m.title.issue.Title
			if issue, ok := m.item.Content.(*github.Issue); ok && issue != nil {
				issue.Title = m.title.issue.Title
			}
			m.item.Title = m.title.issue.Title
			m.viewport.SetContent(m.renderIssueContent())
		}
		if m.title.closing {
			m.showTitle = false
			m.title.closing = false
		}
		return m, cmd
	}

	var commentsCmd tea.Cmd
	if m.showComments {
		m.comments, commentsCmd = m.comments.Update(msg)
		if !m.comments.active {
			m.showComments = false
			m.comments.active = false
			return m, commentsCmd
		}

		if _, ok := msg.(tea.KeyMsg); ok {
			return m, commentsCmd
		}
	}

	clearHint := func() {
		if m.opHint != "" {
			m.opHint = ""
		}
	}
	refreshIssue := func() {
		if m.issue == nil || m.assign.issue == nil {
			return
		}

		m.issue.Assignees = append([]github.User(nil), m.assign.issue.Assignees...)
		if issue, ok := m.item.Content.(*github.Issue); ok && issue != nil {
			issue.Assignees = append([]github.User(nil), m.assign.issue.Assignees...)
		}
		m.viewport.SetContent(m.renderIssueContent())
	}

	if m.showAssign {
		var cmd tea.Cmd
		m.assign, cmd = m.assign.Update(msg)
		refreshIssue()
		if m.assign.closing {
			m.showAssign = false
			m.assign.closing = false
		}
		return m, cmd
	}

	if m.showPRPicker {
		switch msg := msg.(type) {
		case tea.WindowSizeMsg:
			m.syncSize(msg)
			var cmd tea.Cmd
			m.prPicker, cmd = m.prPicker.Update(msg)
			return m, tea.Batch(commentsCmd, cmd)
		case prsLoadedMsg:
			var cmd tea.Cmd
			m.prPicker, cmd = m.prPicker.Update(msg)
			return m, cmd
		case prSelectedMsg:
			m.showPRPicker = false
			m.opMsg = ""
			if m.issue == nil {
				return m, func() tea.Msg {
					return linkPRResultMsg{err: fmt.Errorf("issue unavailable")}
				}
			}
			return m, linkPRToIssueCmd(m.client, m.issue.RepoOwner, m.issue.RepoName, msg.pr.Number, m.issue.Number)
		case prPickerSwitchToManualMsg:
			m.showPRPicker = false
			m.showAddPR = true
			return m, m.addPRInput.Focus()
		case tea.KeyMsg:
			switch msg.Type {
			case tea.KeyEsc:
				m.showPRPicker = false
				return m, nil
			case tea.KeyTab, tea.KeyCtrlT:
				m.showPRPicker = false
				m.showAddPR = true
				return m, m.addPRInput.Focus()
			}
			var cmd tea.Cmd
			m.prPicker, cmd = m.prPicker.Update(msg)
			return m, cmd
		default:
			var cmd tea.Cmd
			m.prPicker, cmd = m.prPicker.Update(msg)
			return m, cmd
		}
	}

	if m.showAddPR {
		switch msg := msg.(type) {
		case tea.WindowSizeMsg:
			m.syncSize(msg)
			return m, commentsCmd
		case addPRResultMsg:
			if msg.err != nil {
				m.opMsg = msg.err.Error()
				return m, commentsCmd
			}
			m.opMsg = fmt.Sprintf("PR #%d added to project", msg.prNumber)
			m.showAddPR = false
			m.addPRInput.Blur()
			m.addPRInput.SetValue("")
			return m, commentsCmd
		case tea.KeyMsg:
			switch msg.String() {
			case "esc":
				m.showAddPR = false
				m.addPRInput.Blur()
				m.addPRInput.SetValue("")
				return m, commentsCmd
			case "tab", "ctrl+t":
				m.showAddPR = false
				m.showPRPicker = true
				m.addPRInput.Blur()
				m.prPicker.SetSize(m.width, m.height)
				return m, tea.Batch(commentsCmd, m.prPicker.Init())
			case "enter":
				owner, repo, number, err := parsePRRef(m.addPRInput.Value())
				if err != nil {
					m.opMsg = "Invalid PR reference. Use owner/repo#number or PR URL"
					return m, commentsCmd
				}
				return m, addPRToProjectCmd(m.client, m.projectID, owner, repo, number)
			default:
				var cmd tea.Cmd
				m.addPRInput, cmd = m.addPRInput.Update(msg)
				return m, tea.Batch(commentsCmd, cmd)
			}
		}
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.syncSize(msg)
		return m, commentsCmd
	case issueLoadedMsg:
		m.loading = false
		m.err = msg.err
		m.issue = msg.issue
		if msg.err == nil && msg.issue != nil {
			m.item.Content = msg.issue
			m.labels.issue = msg.issue
			m.comments.issue = msg.issue
			m.viewport.SetContent(m.renderIssueContent())
		}
		return m, commentsCmd
	case linkedPRsLoadedMsg:
		if msg.err == nil {
			m.linkedPRs = append([]github.PullRequest(nil), msg.prs...)
		} else {
			m.linkedPRs = nil
		}
		if m.item.Type == "Issue" && m.issue != nil && !m.loading && m.err == nil {
			m.viewport.SetContent(m.renderIssueContent())
		}
		return m, commentsCmd
	case addPRResultMsg:
		if msg.err != nil {
			m.opMsg = msg.err.Error()
			return m, commentsCmd
		}
		m.opMsg = fmt.Sprintf("PR #%d added to project", msg.prNumber)
		m.showPRPicker = false
		m.showAddPR = false
		m.addPRInput.Blur()
		m.addPRInput.SetValue("")
		return m, commentsCmd
	case linkPRResultMsg:
		if msg.err != nil {
			m.opMsg = msg.err.Error()
			return m, commentsCmd
		}
		m.opMsg = fmt.Sprintf("PR #%d linked to issue", msg.prNumber)
		m.showPRPicker = false
		m.showAddPR = false
		m.addPRInput.Blur()
		m.addPRInput.SetValue("")
		return m, commentsCmd
	case issueCloseResultMsg:
		m.confirming = false
		m.confirmMsg = ""
		m.confirmAction = ""
		if msg.err != nil {
			m.opMsg = "Error: " + msg.err.Error()
		} else if m.issue != nil {
			m.issue.State = "CLOSED"
			m.opMsg = fmt.Sprintf("Issue #%d closed", m.issue.Number)
			if issue, ok := m.item.Content.(*github.Issue); ok && issue != nil {
				issue.State = "CLOSED"
			}
			m.viewport.SetContent(m.renderIssueContent())
		}
		return m, nil
	case issueReopenResultMsg:
		m.confirming = false
		m.confirmMsg = ""
		m.confirmAction = ""
		if msg.err != nil {
			m.opMsg = "Error: " + msg.err.Error()
		} else if m.issue != nil {
			m.issue.State = "OPEN"
			m.opMsg = fmt.Sprintf("Issue #%d reopened", m.issue.Number)
			if issue, ok := m.item.Content.(*github.Issue); ok && issue != nil {
				issue.State = "OPEN"
			}
			m.viewport.SetContent(m.renderIssueContent())
		}
		return m, nil
	case editBodyResultMsg:
		m.editingBody = false
		if msg.err != nil {
			m.opMsg = "Error: " + msg.err.Error()
			return m, nil
		}
		if !msg.changed {
			m.opMsg = "No changes made"
			return m, nil
		}

		if m.issue == nil {
			m.opMsg = "Error: issue unavailable"
			return m, nil
		}

		m.issue.Body = msg.body
		if issue, ok := m.item.Content.(*github.Issue); ok && issue != nil {
			issue.Body = msg.body
		}
		m.viewport.SetContent(m.renderIssueContent())
		m.opMsg = fmt.Sprintf("Issue #%d body updated", m.issue.Number)
		return m, nil
	case tea.KeyMsg:
		if m.confirming {
			switch msg.String() {
			case "y":
				m.confirming = false
				m.confirmMsg = ""
				switch m.confirmAction {
				case "close":
					return m, m.closeIssueCmd()
				case "reopen":
					return m, m.reopenIssueCmd()
				}
				return m, nil
			case "n", "esc":
				m.confirming = false
				m.confirmMsg = ""
				m.confirmAction = ""
				return m, nil
			}
			return m, nil
		}

		clearHint()
		switch msg.String() {
		case "j", "down":
			m.viewport.ScrollDown(1)
			return m, nil
		case "k", "up":
			m.viewport.ScrollUp(1)
			return m, nil
		case "g":
			m.viewport.GotoTop()
			return m, nil
		case "G":
			m.viewport.GotoBottom()
			return m, nil
		case "x":
			if m.item.Type != "Issue" {
				if m.item.Type == "PullRequest" {
					m.opHint = "PR detail: use GitHub web"
				} else {
					m.opHint = "Draft items are read-only"
				}
				return m, nil
			}
			if m.loading {
				m.opHint = "Issue still loading"
				return m, nil
			}
			if m.issue == nil {
				m.opHint = "Issue detail unavailable"
				return m, nil
			}
			if m.issue.State == "CLOSED" {
				m.opHint = "Issue is already closed"
				return m, nil
			}
			m.confirming = true
			m.confirmMsg = fmt.Sprintf("Close issue #%d? (y/n)", m.issue.Number)
			m.confirmAction = "close"
			return m, nil
		case "X":
			if m.item.Type != "Issue" {
				if m.item.Type == "PullRequest" {
					m.opHint = "PR detail: use GitHub web"
				} else {
					m.opHint = "Draft items are read-only"
				}
				return m, nil
			}
			if m.loading {
				m.opHint = "Issue still loading"
				return m, nil
			}
			if m.issue == nil {
				m.opHint = "Issue detail unavailable"
				return m, nil
			}
			if m.issue.State == "OPEN" {
				m.opHint = "Issue is already open"
				return m, nil
			}
			m.confirming = true
			m.confirmMsg = fmt.Sprintf("Reopen issue #%d? (y/n)", m.issue.Number)
			m.confirmAction = "reopen"
			return m, nil
		case "p":
			if m.item.Type != "Issue" {
				m.opHint = "Cannot add PR from PR/Draft view"
				return m, nil
			}
			if m.loading {
				m.opHint = "Issue still loading"
				return m, nil
			}
			if m.issue == nil {
				m.opHint = "Issue detail unavailable"
				return m, nil
			}

			m.showPRPicker = true
			m.showAddPR = false
			m.addPRInput.SetValue("")
			m.addPRInput.Blur()
			m.prPicker.SetSize(m.width, m.height)
			m.opMsg = ""
			return m, m.prPicker.Init()
		case "t":
			if m.item.Type != "Issue" {
				if m.item.Type == "PullRequest" {
					m.opHint = "PR detail: use GitHub web"
				} else {
					m.opHint = "Draft items are read-only"
				}
				return m, nil
			}
			if m.loading {
				m.opHint = "Issue still loading"
				return m, nil
			}
			if m.issue == nil {
				m.opHint = "Issue detail unavailable"
				return m, nil
			}

			m.issueType = newIssueTypeModel(m.client, m.issue, m.width, m.height)
			m.showIssueType = true
			return m, m.issueType.Init()
		case "T":
			if m.item.Type != "Issue" {
				if m.item.Type == "PullRequest" {
					m.opHint = "PR detail: use GitHub web"
				} else {
					m.opHint = "Draft items are read-only"
				}
				return m, nil
			}
			if m.loading {
				m.opHint = "Issue still loading"
				return m, nil
			}
			if m.issue == nil {
				m.opHint = "Issue detail unavailable"
				return m, nil
			}

			m.title = newTitleModel(m.client, m.issue, m.width, m.height)
			m.showTitle = true
			return m, m.title.Init()
		case "u":
			var issueURL string
			switch m.item.Type {
			case "Issue":
				if m.issue != nil {
					issueURL = fmt.Sprintf("https://github.com/%s/%s/issues/%d", m.issue.RepoOwner, m.issue.RepoName, m.issue.Number)
				} else if m.item.RepoOwner != "" && m.item.RepoName != "" && m.item.ContentNumber > 0 {
					issueURL = fmt.Sprintf("https://github.com/%s/%s/issues/%d", m.item.RepoOwner, m.item.RepoName, m.item.ContentNumber)
				}
			case "PullRequest":
				if pr, ok := m.item.Content.(*github.PullRequest); ok && pr != nil {
					issueURL = pr.URL
				}
			default:
				m.opHint = "Draft items have no URL"
				return m, nil
			}

			if issueURL == "" {
				m.opHint = "URL unavailable"
				return m, nil
			}

			if err := clipboard.WriteAll(issueURL); err != nil {
				m.opMsg = "Error: " + err.Error()
				return m, nil
			}
			m.opMsg = "Copied: " + issueURL
			return m, nil
		case "e":
			if m.showAssign || m.showLabels || m.showComments || m.showIssueType || m.showTitle || m.showAddPR || m.showPRPicker {
				return m, nil
			}
			if m.item.Type != "Issue" {
				if m.item.Type == "PullRequest" {
					m.opHint = "PR detail: use GitHub web"
				} else {
					m.opHint = "Draft items are read-only"
				}
				return m, nil
			}
			if m.loading {
				m.opHint = "Issue still loading"
				return m, nil
			}
			if m.issue == nil {
				m.opHint = "Issue detail unavailable"
				return m, nil
			}

			editorBin, err := editor.ResolveEditor()
			if err != nil {
				m.opMsg = "Error: " + err.Error()
				return m, nil
			}

			tmpPath, cleanup, err := editor.PrepareEdit(m.issue.Body)
			if err != nil {
				m.opMsg = "Error: " + err.Error()
				return m, nil
			}

			m.editingBody = true
			m.opMsg = ""
			return m, tea.ExecProcess(
				exec.Command(editorBin, tmpPath),
				func(err error) tea.Msg {
					return m.editBodyExecResult(err, tmpPath, m.issue.Body, cleanup)
				},
			)
		case "c", "a", "L":
			if m.item.Type != "Issue" {
				if m.item.Type == "PullRequest" {
					m.opHint = "PR detail: use GitHub web"
				} else {
					m.opHint = "Draft items are read-only"
				}
				return m, nil
			}
			if msg.String() == "c" {
				if m.showAssign {
					return m, nil
				}
				if m.loading {
					m.opHint = "Issue still loading"
					return m, nil
				}
				if m.issue == nil {
					m.opHint = "Issue detail unavailable"
					return m, nil
				}

				m.comments.issue = m.issue
				m.comments.width = m.width
				m.comments.height = m.height
				m.showComments = true
				return m, m.comments.Init()
			}
			if msg.String() == "a" {
				if m.loading {
					m.opHint = "Issue still loading"
					return m, nil
				}
				if m.issue == nil {
					m.opHint = "Issue detail unavailable"
					return m, nil
				}

				m.assign = newAssignModel(m.client, m.issue, m.width, m.height)
				m.showAssign = true
				return m, m.assign.Init()
			}
			if msg.String() == "L" {
				if m.loading {
					m.opHint = "Issue still loading"
					return m, nil
				}
				if m.issue == nil {
					m.opHint = "Issue detail unavailable"
					return m, nil
				}

				m.labels = newLabelsModel(m.client, m.issue, m.width, m.height)
				m.showLabels = true
				return m, m.labels.Init()
			}
		}
	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	if commentsCmd != nil {
		return m, tea.Batch(commentsCmd, cmd)
	}
	return m, cmd
}

func (m Model) editBodyExecResult(execErr error, tmpPath, originalBody string, cleanup func()) tea.Msg {
	if execErr != nil {
		cleanup()
		return editBodyResultMsg{err: fmt.Errorf("editor exited with error: %w", execErr)}
	}

	newBody, changed, err := editor.ReadResult(tmpPath, originalBody)
	if err != nil {
		cleanup()
		return editBodyResultMsg{err: err}
	}

	if !changed {
		cleanup()
		return editBodyResultMsg{body: originalBody, changed: false}
	}

	if m.issue == nil {
		cleanup()
		return editBodyResultMsg{err: fmt.Errorf("issue unavailable")}
	}

	err = m.client.UpdateIssueBody(m.issue.ID, newBody)
	if err != nil {
		return editBodyResultMsg{err: fmt.Errorf("%w (your edits are saved at %s)", err, tmpPath)}
	}

	cleanup()
	return editBodyResultMsg{body: newBody, changed: true}
}

func (m Model) closeIssueCmd() tea.Cmd {
	return func() tea.Msg {
		if m.issue == nil {
			return issueCloseResultMsg{err: fmt.Errorf("issue unavailable")}
		}
		err := m.client.CloseIssue(m.issue.RepoOwner, m.issue.RepoName, m.issue.Number)
		return issueCloseResultMsg{err: err}
	}
}

func (m Model) reopenIssueCmd() tea.Cmd {
	return func() tea.Msg {
		if m.issue == nil {
			return issueReopenResultMsg{err: fmt.Errorf("issue unavailable")}
		}
		err := m.client.ReopenIssue(m.issue.RepoOwner, m.issue.RepoName, m.issue.Number)
		return issueReopenResultMsg{err: err}
	}
}

func (m Model) View() string {
	if m.showLabels {
		return m.labels.View()
	}

	if m.showIssueType {
		return m.issueType.View()
	}

	if m.showTitle {
		return m.title.View()
	}

	if m.showComments {
		return m.comments.View()
	}

	if m.showPRPicker {
		return m.renderPRPickerOverlay()
	}

	if m.showAddPR {
		return m.renderAddPROverlay()
	}

	if m.loading {
		msg := "Loading issue..."
		if m.width <= 0 || m.height <= 0 {
			return msg
		}
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, msg)
	}

	if m.err != nil {
		msg := fmt.Sprintf("Error: %v (press Esc to go back)", m.err)
		if m.width <= 0 || m.height <= 0 {
			return msg
		}
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, msg)
	}

	var body string
	switch m.item.Type {
	case "Issue":
		if m.issue == nil {
			body = "Error: issue content unavailable (press Esc to go back)"
		} else if m.showAssign {
			body = m.assign.View()
		} else {
			body = m.viewport.View()
		}
	case "PullRequest":
		body = m.renderPRContent()
	default:
		body = m.renderDraftContent()
	}

	if m.confirming {
		confirmStr := lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Render(m.confirmMsg)
		if body == "" {
			return confirmStr
		}
		return body + "\n" + confirmStr
	}

	if m.opMsg != "" {
		var msgStyle lipgloss.Style
		if isErrorOpMsg(m.opMsg) {
			msgStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
		} else {
			msgStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
		}
		msg := msgStyle.Render(m.opMsg)
		if body == "" {
			return msg
		}
		return body + "\n" + msg
	}

	if m.opHint != "" {
		hint := lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render(m.opHint)
		if body == "" {
			return hint
		}
		return body + "\n" + hint
	}

	footerHints := m.renderFooterHints()
	if footerHints != "" {
		if body == "" {
			return footerHints
		}
		return body + "\n" + footerHints
	}

	return body
}

func (m Model) renderIssueContent() string {
	if m.issue == nil {
		return ""
	}

	body := strings.TrimSpace(m.issue.Body)
	if body == "" {
		body = "_No description provided._"
	}

	renderedBody, err := glamour.Render(body, "dark")
	if err != nil {
		renderedBody = body
	}

	assignees := "-"
	if len(m.issue.Assignees) > 0 {
		vals := make([]string, 0, len(m.issue.Assignees))
		for _, a := range m.issue.Assignees {
			vals = append(vals, a.Login)
		}
		assignees = strings.Join(vals, ", ")
	}

	labels := "-"
	if len(m.issue.Labels) > 0 {
		vals := make([]string, 0, len(m.issue.Labels))
		for _, l := range m.issue.Labels {
			vals = append(vals, "● "+l.Name)
		}
		labels = strings.Join(vals, ", ")
	}

	created := ""
	if !m.issue.CreatedAt.IsZero() {
		created = m.issue.CreatedAt.Format(time.DateOnly)
	}

	meta := "by " + m.issue.Author.Login
	if created != "" {
		meta += " · " + created
	}

	lineWidth := max(m.viewport.Width-2, 10)
	divider := strings.Repeat("─", lineWidth)

	return strings.Join([]string{
		fmt.Sprintf("Issue #%d [%s]", m.issue.Number, strings.ToUpper(m.issue.State)),
		m.issue.Title,
		meta,
		"",
		"Assignees: " + assignees,
		"Labels: " + labels,
		divider,
		strings.TrimRight(renderedBody, "\n"),
		"",
		m.renderLinkedPRsSection(lineWidth),
	}, "\n")
}

func (m Model) renderLinkedPRsSection(lineWidth int) string {
	divider := strings.Repeat("─", lineWidth)
	lines := []string{
		"Linked Pull Requests",
		divider,
	}

	if len(m.linkedPRs) == 0 {
		lines = append(lines, lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render("(None)"))
		return strings.Join(lines, "\n")
	}

	for _, pr := range m.linkedPRs {
		state := strings.ToLower(pr.State)
		stateColor := lipgloss.Color("8")
		switch state {
		case "open":
			stateColor = lipgloss.Color("10")
		case "merged":
			stateColor = lipgloss.Color("13")
		case "closed":
			stateColor = lipgloss.Color("9")
		}

		stateText := lipgloss.NewStyle().Foreground(stateColor).Render("[" + state + "]")
		lines = append(lines, fmt.Sprintf("#%-4d %s %s by @%s", pr.Number, pr.Title, stateText, pr.Author.Login))
	}

	return strings.Join(lines, "\n")
}

func (m Model) renderPRContent() string {
	pr, _ := m.item.Content.(*github.PullRequest)
	if pr == nil {
		return "PR detail unavailable"
	}

	return strings.Join([]string{
		fmt.Sprintf("PR #%d [%s]", pr.Number, strings.ToUpper(pr.State)),
		pr.Title,
		"by " + pr.Author.Login,
		pr.URL,
		"",
		"[PR detail: use GitHub web for comments, assignees, labels]",
	}, "\n")
}

func (m Model) renderDraftContent() string {
	return strings.Join([]string{
		"(Draft)",
		m.item.Title,
		"",
		"[Draft items are read-only]",
	}, "\n")
}

func (m Model) renderAddPROverlay() string {
	divider := strings.Repeat("─", max(min(m.width-2, 16), 16))
	parts := []string{
		"Add PR to project",
		divider,
		"owner/repo#number or PR URL:",
		m.addPRInput.View(),
		"Enter to confirm, Esc to cancel",
	}

	if m.opMsg != "" {
		msgStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
		if isErrorOpMsg(m.opMsg) {
			msgStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
		}
		parts = append(parts, msgStyle.Render(m.opMsg))
	}

	body := strings.Join(parts, "\n")
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

func (m Model) renderPRPickerOverlay() string {
	body := strings.Join([]string{
		m.prPicker.View(),
		lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render("Enter select • Tab/Ctrl+T manual • Esc cancel"),
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

func (m Model) renderFooterHints() string {
	if m.width <= 0 {
		return ""
	}

	hintStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("236")).
		Foreground(lipgloss.Color("8"))

	hints := "? Help  u Copy URL  c Comments  a Assign  L Labels  p Add PR  x Close  Esc Back"
	if m.item.Type == "Issue" {
		hints = "? Help  u Copy URL  c Comments  a Assign  L Labels  t Type  T Title  p Add PR  x Close  Esc Back"
	}

	renderedHints := hintStyle.Render(hints)
	hintsWidth := lipgloss.Width(renderedHints)

	if hintsWidth < m.width {
		diff := m.width - hintsWidth
		renderedHints = hintStyle.Render(hints + strings.Repeat(" ", diff))
	} else if hintsWidth > m.width {
		runes := []rune(hints)
		if len(runes) > m.width {
			hints = string(runes[:m.width])
		}
		renderedHints = hintStyle.Render(hints)
	}

	return renderedHints
}

func addPRToProjectCmd(client github.GitHubClient, projectID, owner, repo string, prNumber int) tea.Cmd {
	return func() tea.Msg {
		nodeID, err := client.GetPullRequestNodeID(owner, repo, prNumber)
		if err != nil {
			return addPRResultMsg{err: fmt.Errorf("PR not found: %w", err)}
		}

		err = client.AddItemToProject(projectID, nodeID)
		return addPRResultMsg{prNumber: prNumber, err: err}
	}
}

func linkPRToIssueCmd(client github.GitHubClient, owner, repo string, prNumber, issueNumber int) tea.Cmd {
	return func() tea.Msg {
		err := client.LinkPRToIssue(owner, repo, prNumber, issueNumber)
		if err != nil {
			return linkPRResultMsg{err: fmt.Errorf("failed to link PR to issue: %w", err)}
		}

		return linkPRResultMsg{prNumber: prNumber}
	}
}

func (m *Model) syncSize(msg tea.WindowSizeMsg) {
	m.width = msg.Width
	m.height = msg.Height
	m.viewport.Width = max(m.width-2, 0)
	m.viewport.Height = max(m.height-3, 0)
	m.labels.width = msg.Width
	m.labels.height = msg.Height
	m.issueType.width = msg.Width
	m.issueType.height = msg.Height
	m.title.width = msg.Width
	m.title.height = msg.Height
	m.title.textinput.Width = max(msg.Width-8, 20)
	m.comments.width = msg.Width
	m.comments.height = msg.Height
	m.comments.syncSize()
	m.addPRInput.Width = min(max(m.width-8, 10), 50)
	m.prPicker.SetSize(m.width, m.height)
	if m.item.Type == "Issue" && !m.loading && m.err == nil {
		m.viewport.SetContent(m.renderIssueContent())
	}
}

func parsePRRef(input string) (owner, repo string, number int, err error) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return "", "", 0, fmt.Errorf("empty PR reference")
	}

	if strings.Contains(trimmed, "github.com/") {
		trimmed = strings.TrimPrefix(trimmed, "https://")
		trimmed = strings.TrimPrefix(trimmed, "http://")
		trimmed = strings.TrimPrefix(trimmed, "github.com/")
		trimmed = strings.TrimPrefix(trimmed, "www.github.com/")
		parts := strings.Split(strings.Trim(trimmed, "/"), "/")
		if len(parts) != 4 || parts[0] == "" || parts[1] == "" || parts[2] != "pull" {
			return "", "", 0, fmt.Errorf("invalid PR URL")
		}

		number, err = strconv.Atoi(parts[3])
		if err != nil || number <= 0 {
			return "", "", 0, fmt.Errorf("invalid PR number")
		}

		return parts[0], parts[1], number, nil
	}

	refParts := strings.Split(trimmed, "#")
	if len(refParts) != 2 {
		return "", "", 0, fmt.Errorf("invalid PR reference")
	}

	repoParts := strings.Split(refParts[0], "/")
	if len(repoParts) != 2 || repoParts[0] == "" || repoParts[1] == "" {
		return "", "", 0, fmt.Errorf("invalid repository reference")
	}

	number, err = strconv.Atoi(refParts[1])
	if err != nil || number <= 0 {
		return "", "", 0, fmt.Errorf("invalid PR number")
	}

	return repoParts[0], repoParts[1], number, nil
}

func isErrorOpMsg(msg string) bool {
	return strings.HasPrefix(msg, "Error:") ||
		strings.HasPrefix(msg, "Invalid PR reference") ||
		strings.HasPrefix(msg, "PR not found:")
}
