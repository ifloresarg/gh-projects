package picker

import (
	"errors"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ifloresarg/gh-projects/internal/github"
)

type ProjectSelectedMsg struct {
	Project github.Project
}

type projectsLoadedMsg struct {
	projects []github.Project
	err      error
}

type item struct {
	project    github.Project
	multiOwner bool
}

func (i item) Title() string {
	if i.multiOwner {
		return fmt.Sprintf("%s · #%d %s", i.project.Owner, i.project.Number, i.project.Title)
	}

	return fmt.Sprintf("#%d %s", i.project.Number, i.project.Title)
}

func (i item) Description() string { return fmt.Sprintf("%d items", i.project.ItemCount) }

func (i item) FilterValue() string {
	if i.multiOwner {
		return i.project.Owner + " " + i.project.Title
	}

	return i.project.Title
}

type Model struct {
	client     github.GitHubClient
	owner      string
	multiOwner bool
	list       list.Model
	spinner    spinner.Model
	loading    bool
	err        error
	infoNote   string
	width      int
	height     int
}

func New(client github.GitHubClient, owner string) Model {
	l, s := newPickerListAndSpinner()
	l.Title = fmt.Sprintf("GitHub Projects · %s", owner)

	return Model{
		client:  client,
		owner:   owner,
		list:    l,
		spinner: s,
		loading: true,
	}
}

func NewMultiOwner(client github.GitHubClient) Model {
	l, s := newPickerListAndSpinner()
	l.Title = "GitHub Projects · All Organizations"

	return Model{
		client:     client,
		multiOwner: true,
		list:       l,
		spinner:    s,
		loading:    true,
	}
}

func newPickerListAndSpinner() (list.Model, spinner.Model) {
	l := list.New(nil, list.NewDefaultDelegate(), 0, 0)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.SetShowHelp(true)

	s := spinner.New()
	s.Spinner = spinner.Dot

	return l, s
}

func (m Model) Init() tea.Cmd {
	if m.multiOwner {
		return tea.Batch(m.spinner.Tick, fetchAllProjects(m.client))
	}

	return tea.Batch(m.spinner.Tick, fetchProjects(m.client, m.owner))
}

func fetchProjects(client github.GitHubClient, owner string) tea.Cmd {
	return func() tea.Msg {
		projects, err := client.ListProjects(owner)
		return projectsLoadedMsg{projects: projects, err: err}
	}
}

func fetchAllProjects(client github.GitHubClient) tea.Cmd {
	return func() tea.Msg {
		projects, err := client.ListAllAccessibleProjects()
		return projectsLoadedMsg{projects: projects, err: err}
	}
}

func isPartialScopeError(err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, github.ErrMissingScopeReadOrg) {
		return true
	}

	return strings.Contains(err.Error(), github.ErrMissingScopeReadOrg.Error())
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.list.SetSize(msg.Width, max(msg.Height-2, 0))
		return m, nil
	case projectsLoadedMsg:
		m.loading = false
		m.err = nil
		m.infoNote = ""
		if msg.err != nil && !m.multiOwner || msg.err != nil && !isPartialScopeError(msg.err) {
			m.err = msg.err
			return m, nil
		}
		if m.multiOwner && isPartialScopeError(msg.err) {
			m.infoNote = "Note: some organization projects may be unavailable."
		}

		items := make([]list.Item, 0, len(msg.projects))
		for _, project := range msg.projects {
			items = append(items, item{project: project, multiOwner: m.multiOwner})
		}
		m.list.SetItems(items)
		return m, nil
	case spinner.TickMsg:
		if !m.loading {
			return m, nil
		}
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case tea.KeyMsg:
		if m.loading || m.err != nil {
			return m, nil
		}
		if msg.String() == "enter" {
			selected, ok := m.list.SelectedItem().(item)
			if !ok {
				return m, nil
			}
			return m, func() tea.Msg {
				return ProjectSelectedMsg{Project: selected.project}
			}
		}
	}

	if m.loading || m.err != nil {
		return m, nil
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m Model) View() string {
	if m.loading {
		content := fmt.Sprintf("%s Loading projects...", m.spinner.View())
		if m.width <= 0 || m.height <= 0 {
			return content
		}
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
	}

	if m.err != nil {
		content := lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Render(fmt.Sprintf("Error loading projects: %v", m.err))
		if m.width <= 0 || m.height <= 0 {
			return content
		}
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
	}

	if m.infoNote != "" {
		return m.infoNote + "\n\n" + m.list.View()
	}

	return m.list.View()
}
