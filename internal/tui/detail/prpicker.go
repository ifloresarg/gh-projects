package detail

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/ifloresarg/gh-projects/internal/github"
)

type RepoRef struct {
	Owner string
	Name  string
}

type prsLoadedMsg struct {
	prs []github.PullRequest
	err error
}

type prSelectedMsg struct {
	pr github.PullRequest
}

type prPickerSwitchToManualMsg struct{}

type prItem struct {
	pr github.PullRequest
}

func (i prItem) Title() string {
	return fmt.Sprintf("#%d %s", i.pr.Number, i.pr.Title)
}

func (i prItem) Description() string {
	if i.pr.State == "MERGED" {
		return fmt.Sprintf("%s/%s · %s · merged %s", i.pr.RepoOwner, i.pr.RepoName, i.pr.Author.Login, relativeTime(i.pr.MergedAt))
	}

	return fmt.Sprintf("%s/%s · %s · %s", i.pr.RepoOwner, i.pr.RepoName, i.pr.Author.Login, relativeTime(i.pr.CreatedAt))
}

func (i prItem) FilterValue() string {
	return fmt.Sprintf("%s/%s#%d %s %s", i.pr.RepoOwner, i.pr.RepoName, i.pr.Number, i.pr.Title, i.pr.Author.Login)
}

type prPickerModel struct {
	list           list.Model
	client         github.GitHubClient
	repos          []RepoRef
	prFetchLimit   int
	loading        bool
	err            error
	allPRs         []github.PullRequest
	mergedPRWindow time.Duration
}

func newPRPicker(client github.GitHubClient, repos []RepoRef, mergedPRWindow time.Duration, prFetchLimit int) prPickerModel {
	l := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Link PR to issue"
	l.SetFilteringEnabled(true)
	l.SetShowStatusBar(false)
	l.SetShowHelp(true)

	return prPickerModel{
		list:           l,
		client:         client,
		repos:          append([]RepoRef(nil), repos...),
		prFetchLimit:   prFetchLimit,
		loading:        true,
		mergedPRWindow: mergedPRWindow,
	}
}

func (m prPickerModel) Init() tea.Cmd {
	return fetchPRs(m.client, m.repos, m.mergedPRWindow, m.prFetchLimit)
}

func fetchPRs(client github.GitHubClient, repos []RepoRef, mergedPRWindow time.Duration, limit int) tea.Cmd {
	return func() tea.Msg {
		prs := make([]github.PullRequest, 0)
		for _, repo := range repos {
			repoPRs, err := client.ListRepositoryPullRequests(repo.Owner, repo.Name, limit)
			if err != nil {
				return prsLoadedMsg{prs: nil, err: err}
			}
			prs = append(prs, repoPRs...)
		}

		filtered := make([]github.PullRequest, 0, len(prs))
		for _, pr := range prs {
			if pr.State == "MERGED" && mergedPRWindow > 0 && !pr.MergedAt.IsZero() && time.Since(pr.MergedAt) > mergedPRWindow {
				continue
			}
			filtered = append(filtered, pr)
		}

		return prsLoadedMsg{prs: filtered, err: nil}
	}
}

func (m *prPickerModel) SetSize(width, height int) {
	listWidth := min(max(width-12, 20), 64)
	listHeight := max(height-10, 8)
	m.list.SetSize(listWidth, listHeight)
}

func (m prPickerModel) Update(msg tea.Msg) (prPickerModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.SetSize(msg.Width, msg.Height)
		return m, nil
	case prsLoadedMsg:
		m.loading = false
		m.err = msg.err
		if msg.err != nil {
			return m, nil
		}

		prs := append([]github.PullRequest(nil), msg.prs...)
		effectiveTime := func(pr github.PullRequest) time.Time {
			if pr.MergedAt.After(pr.CreatedAt) {
				return pr.MergedAt
			}
			return pr.CreatedAt
		}
		sort.Slice(prs, func(i, j int) bool {
			return effectiveTime(prs[i]).After(effectiveTime(prs[j]))
		})
		m.allPRs = prs

		items := make([]list.Item, 0, len(prs))
		for _, pr := range prs {
			items = append(items, prItem{pr: pr})
		}
		m.list.SetItems(items)
		return m, nil
	case tea.KeyMsg:
		if m.loading || m.err != nil {
			return m, nil
		}

		switch msg.Type {
		case tea.KeyEnter:
			selected, ok := m.list.SelectedItem().(prItem)
			if !ok {
				return m, nil
			}
			return m, func() tea.Msg {
				return prSelectedMsg(selected)
			}
		case tea.KeyEsc:
			return m, func() tea.Msg { return prPickerSwitchToManualMsg{} }
		case tea.KeyTab, tea.KeyCtrlT:
			return m, func() tea.Msg { return prPickerSwitchToManualMsg{} }
		default:
			var cmd tea.Cmd
			m.list, cmd = m.list.Update(msg)
			return m, cmd
		}
	}

	if m.loading || m.err != nil {
		return m, nil
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m prPickerModel) View() string {
	if m.loading {
		return "Loading..."
	}

	if m.err != nil {
		return fmt.Sprintf("Error loading PRs: %v", m.err)
	}

	if len(m.allPRs) == 0 {
		return strings.TrimSpace(m.list.View())
	}

	return m.list.View()
}

func relativeTime(t time.Time) string {
	if t.IsZero() {
		return "unknown"
	}

	d := time.Since(t)
	if d < time.Minute {
		return "just now"
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm ago", int(d/time.Minute))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh ago", int(d/time.Hour))
	}
	return fmt.Sprintf("%d days ago", int(d/(24*time.Hour)))
}
