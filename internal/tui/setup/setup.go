package setup

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/ifloresarg/gh-projects/internal/github"
)

type wizardStep int

const (
	stepOwner wizardStep = iota
	stepProjectLoading
	stepProjectSelect
	stepProjectError
	stepProjectEmpty
	stepViewLoading
	stepViewSelect
	stepViewError
)

type SetupCompleteMsg struct {
	Owner   string
	Project int
	View    string
}

type SetupCancelMsg struct{}

type projectsLoadedMsg struct {
	projects []github.Project
	err      error
}

type viewsLoadedMsg struct {
	views []github.ProjectView
	err   error
}

type Model struct {
	step            wizardStep
	ownerInput      textinput.Model
	width           int
	height          int
	client          github.GitHubClient
	projects        []github.Project
	views           []github.ProjectView
	selectedProject github.Project
	projectCursor   int
	viewCursor      int
	err             error
}

func New(client github.GitHubClient) Model {
	ownerInput := textinput.New()
	ownerInput.Prompt = "> "
	ownerInput.Placeholder = "owner"
	ownerInput.Focus()
	ownerInput.CharLimit = 100
	ownerInput.Width = 40

	return Model{
		step:       stepOwner,
		ownerInput: ownerInput,
		client:     client,
	}
}

func (m Model) Init() tea.Cmd {
	return textinput.Blink
}

func fetchProjects(client github.GitHubClient, owner string) tea.Cmd {
	return func() tea.Msg {
		projects, err := client.ListProjects(owner)
		return projectsLoadedMsg{projects: projects, err: err}
	}
}

func fetchViews(client github.GitHubClient, projectID string) tea.Cmd {
	return func() tea.Msg {
		views, err := client.GetProjectViews(projectID)
		return viewsLoadedMsg{views: views, err: err}
	}
}

func completeCmd(owner string, project int, view string) tea.Cmd {
	return func() tea.Msg {
		return SetupCompleteMsg{Owner: owner, Project: project, View: view}
	}
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case projectsLoadedMsg:
		if m.step != stepProjectLoading {
			return m, nil
		}
		m.err = msg.err
		if msg.err != nil {
			m.step = stepProjectError
			return m, nil
		}

		m.projects = msg.projects
		m.projectCursor = 0
		if len(msg.projects) == 0 {
			m.step = stepProjectEmpty
			return m, nil
		}

		m.step = stepProjectSelect
		return m, nil
	case viewsLoadedMsg:
		if m.step != stepViewLoading {
			return m, nil
		}
		m.err = msg.err
		if msg.err != nil {
			m.step = stepViewError
			return m, nil
		}

		m.views = msg.views
		m.viewCursor = 0
		owner := m.owner()
		if len(msg.views) == 0 {
			return m, completeCmd(owner, m.selectedProject.Number, "")
		}
		if len(msg.views) == 1 {
			return m, completeCmd(owner, m.selectedProject.Number, msg.views[0].Name)
		}

		m.step = stepViewSelect
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, func() tea.Msg { return SetupCancelMsg{} }
		case "esc":
			if m.step == stepOwner {
				return m, func() tea.Msg { return SetupCancelMsg{} }
			}
			return m.goBack(), nil
		case "enter":
			switch m.step {
			case stepOwner:
				owner := m.owner()
				if owner == "" {
					return m, nil
				}
				m.step = stepProjectLoading
				m.projects = nil
				m.views = nil
				m.selectedProject = github.Project{}
				m.projectCursor = 0
				m.viewCursor = 0
				m.err = nil
				return m, fetchProjects(m.client, owner)
			case stepProjectSelect:
				if len(m.projects) == 0 {
					return m, nil
				}
				m.selectedProject = m.projects[m.projectCursor]
				m.views = nil
				m.viewCursor = 0
				m.err = nil
				m.step = stepViewLoading
				return m, fetchViews(m.client, m.selectedProject.ID)
			case stepProjectError:
				m.step = stepProjectLoading
				m.err = nil
				return m, fetchProjects(m.client, m.owner())
			case stepViewError:
				m.step = stepViewLoading
				m.err = nil
				return m, fetchViews(m.client, m.selectedProject.ID)
			case stepViewSelect:
				if len(m.views) == 0 {
					return m, nil
				}
				return m, completeCmd(m.owner(), m.selectedProject.Number, m.views[m.viewCursor].Name)
			}
		case "tab":
			switch m.step {
			case stepProjectSelect, stepProjectError, stepProjectEmpty:
				return m, completeCmd(m.owner(), 0, "")
			case stepViewSelect, stepViewError:
				return m, completeCmd(m.owner(), m.selectedProject.Number, "")
			}
		case "s":
			if m.step == stepProjectSelect {
				return m, completeCmd(m.owner(), 0, "")
			}
		case "j", "down":
			switch m.step {
			case stepProjectSelect:
				if m.projectCursor < len(m.projects)-1 {
					m.projectCursor++
				}
				return m, nil
			case stepViewSelect:
				if m.viewCursor < len(m.views)-1 {
					m.viewCursor++
				}
				return m, nil
			}
		case "k", "up":
			switch m.step {
			case stepProjectSelect:
				if m.projectCursor > 0 {
					m.projectCursor--
				}
				return m, nil
			case stepViewSelect:
				if m.viewCursor > 0 {
					m.viewCursor--
				}
				return m, nil
			}
		}
	}

	if m.step == stepOwner {
		var cmd tea.Cmd
		m.ownerInput, cmd = m.ownerInput.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m Model) View() string {
	var content string

	switch m.step {
	case stepProjectLoading:
		content = strings.Join([]string{
			lipgloss.NewStyle().Bold(true).Render("Choose a default project"),
			"",
			fmt.Sprintf("Loading projects for %s...", m.owner()),
			"",
			lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render("Esc to go back"),
		}, "\n")
	case stepProjectSelect:
		content = strings.Join([]string{
			lipgloss.NewStyle().Bold(true).Render("Choose a default project"),
			"",
			fmt.Sprintf("Select a project for %s, or skip this step.", m.owner()),
			"",
			renderProjectList(m.projects, m.projectCursor),
			"",
			lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render("j/k ↑/↓ • Enter to select • Tab to skip"),
		}, "\n")
	case stepProjectError:
		content = strings.Join([]string{
			lipgloss.NewStyle().Bold(true).Render("Choose a default project"),
			"",
			lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Render(fmt.Sprintf("Error loading projects: %v", m.err)),
			"",
			lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render("Enter to retry • Tab to skip • Esc to go back"),
		}, "\n")
	case stepProjectEmpty:
		content = strings.Join([]string{
			lipgloss.NewStyle().Bold(true).Render("Choose a default project"),
			"",
			fmt.Sprintf("No projects found for %s.", m.owner()),
			"",
			lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render("Tab to skip • Esc to go back"),
		}, "\n")
	case stepViewLoading:
		content = strings.Join([]string{
			lipgloss.NewStyle().Bold(true).Render("Choose a default view"),
			"",
			fmt.Sprintf("Loading views for #%d %s...", m.selectedProject.Number, m.selectedProject.Title),
			"",
			lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render("Esc to go back"),
		}, "\n")
	case stepViewSelect:
		content = strings.Join([]string{
			lipgloss.NewStyle().Bold(true).Render("Choose a default view"),
			"",
			fmt.Sprintf("Select a view for #%d %s, or skip this step.", m.selectedProject.Number, m.selectedProject.Title),
			"",
			renderViewList(m.views, m.viewCursor),
			"",
			lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render("j/k ↑/↓ • Enter to select • Tab to skip"),
		}, "\n")
	case stepViewError:
		content = strings.Join([]string{
			lipgloss.NewStyle().Bold(true).Render("Choose a default view"),
			"",
			lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Render(fmt.Sprintf("Error loading views: %v", m.err)),
			"",
			lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render("Enter to retry • Tab to skip • Esc to go back"),
		}, "\n")
	default:
		content = strings.Join([]string{
			lipgloss.NewStyle().Bold(true).Render("Welcome to gh-projects"),
			"",
			"Enter the GitHub username or organization that owns the projects you want to manage.",
			"",
			m.ownerInput.View(),
			"",
			lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render("Tip: this is the name shown in your GitHub profile URL (github.com/<owner>)"),
			"",
			lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render("Press Enter to continue • Esc to quit"),
		}, "\n")
	}

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("8")).
		Background(lipgloss.Color("235")).
		Foreground(lipgloss.Color("7")).
		Padding(1, 2).
		Width(55).
		Render(content)

	if m.width > 0 && m.height > 0 {
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
	}

	return box
}

func (m Model) owner() string {
	return strings.TrimSpace(m.ownerInput.Value())
}

func (m Model) goBack() Model {
	m.err = nil

	switch m.step {
	case stepProjectLoading, stepProjectSelect, stepProjectError, stepProjectEmpty:
		m.step = stepOwner
		m.projects = nil
		m.projectCursor = 0
	case stepViewLoading, stepViewSelect, stepViewError:
		m.step = stepProjectSelect
		m.views = nil
		m.viewCursor = 0
	}

	return m
}

func renderProjectList(projects []github.Project, cursor int) string {
	lines := make([]string, 0, len(projects))
	for i, project := range projects {
		prefix := "  "
		if i == cursor {
			prefix = "> "
		}
		lines = append(lines, fmt.Sprintf("%s%d. #%d %s (%d items)", prefix, i+1, project.Number, project.Title, project.ItemCount))
	}
	return strings.Join(lines, "\n")
}

func renderViewList(views []github.ProjectView, cursor int) string {
	lines := make([]string, 0, len(views))
	for i, view := range views {
		prefix := "  "
		if i == cursor {
			prefix = "> "
		}
		line := fmt.Sprintf("%s%d. %s", prefix, i+1, view.Name)
		if view.Layout != "" {
			line = fmt.Sprintf("%s [%s]", line, view.Layout)
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}
