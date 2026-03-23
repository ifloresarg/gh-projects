package viewpicker

import (
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ifloresarg/gh-projects/internal/github"
)

type ViewSelectedMsg struct {
	View github.ProjectView
}

type viewsLoadedMsg struct {
	views []github.ProjectView
	err   error
}

type item struct {
	view github.ProjectView
}

func (i item) Title() string { return i.view.Name }

func (i item) Description() string {
	if i.view.Filter != "" {
		return i.view.Filter
	}

	return "All items"
}

func (i item) FilterValue() string { return i.view.Name }

type Model struct {
	client    github.GitHubClient
	projectID string
	list      list.Model
	spinner   spinner.Model
	loading   bool
	err       error
	width     int
	height    int
}

func New(client github.GitHubClient, projectID string, projectTitle string) Model {
	l := list.New(nil, list.NewDefaultDelegate(), 0, 0)
	l.Title = fmt.Sprintf("Views · %s", projectTitle)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.SetShowHelp(true)

	s := spinner.New()
	s.Spinner = spinner.Dot

	return Model{
		client:    client,
		projectID: projectID,
		list:      l,
		spinner:   s,
		loading:   true,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, fetchViews(m.client, m.projectID))
}

func fetchViews(client github.GitHubClient, projectID string) tea.Cmd {
	return func() tea.Msg {
		views, err := client.GetProjectViews(projectID)
		if err != nil {
			return viewsLoadedMsg{views: nil, err: err}
		}

		boardViews := make([]github.ProjectView, 0, len(views))
		for _, view := range views {
			if view.Layout == "BOARD_LAYOUT" {
				boardViews = append(boardViews, view)
			}
		}

		return viewsLoadedMsg{views: boardViews, err: nil}
	}
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.list.SetSize(msg.Width, max(msg.Height-2, 0))
		return m, nil
	case viewsLoadedMsg:
		m.loading = false
		m.err = msg.err
		if msg.err != nil {
			return m, nil
		}

		items := make([]list.Item, 0, len(msg.views))
		for _, view := range msg.views {
			items = append(items, item{view: view})
		}
		m.list.SetItems(items)

		if len(msg.views) == 1 {
			return m, func() tea.Msg {
				return ViewSelectedMsg{View: msg.views[0]}
			}
		}

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
				return ViewSelectedMsg{View: selected.view}
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
		content := fmt.Sprintf("%s Loading views...", m.spinner.View())
		if m.width <= 0 || m.height <= 0 {
			return content
		}
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
	}

	if m.err != nil {
		content := lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Render(fmt.Sprintf("Error loading views: %v", m.err))
		if m.width <= 0 || m.height <= 0 {
			return content
		}
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
	}

	return m.list.View()
}
