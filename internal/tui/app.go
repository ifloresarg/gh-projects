package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ifloresarg/gh-projects/internal/config"
	"github.com/ifloresarg/gh-projects/internal/github"
	"github.com/ifloresarg/gh-projects/internal/tui/board"
	"github.com/ifloresarg/gh-projects/internal/tui/detail"
	"github.com/ifloresarg/gh-projects/internal/tui/help"
	"github.com/ifloresarg/gh-projects/internal/tui/picker"
	"github.com/ifloresarg/gh-projects/internal/tui/setup"
	"github.com/ifloresarg/gh-projects/internal/tui/viewpicker"
)

type ViewState int

const (
	ViewLoading ViewState = iota
	ViewSetup
	ViewPicker
	ViewViewPicker
	ViewBoard
	ViewDetail
)

type App struct {
	config          config.Config
	client          github.GitHubClient
	state           ViewState
	quitConfirm     bool
	loadErr         string
	setup           setup.Model
	picker          picker.Model
	viewpicker      viewpicker.Model
	board           board.Model
	detail          detail.Model
	help            help.Model
	selectedProject *github.Project
	width           int
	height          int
	keys            KeyMap
	initialViewName string
	directMode      bool
}

type projectResolvedMsg struct {
	project *github.Project
	views   []github.ProjectView
	err     error
}

func NewApp(cfg config.Config, client github.GitHubClient) App {
	a := App{
		config: cfg,
		client: client,
		help:   help.New(0, 0),
		keys:   DefaultKeyMap,
	}

	if !cfg.OwnerFromFlag && cfg.DefaultOwner == "" && cfg.DefaultProject == 0 {
		a.state = ViewSetup
		a.setup = setup.New(client)
		return a
	}

	if cfg.OwnerFromFlag {
		a.state = ViewPicker
		a.picker = picker.New(client, cfg.DefaultOwner)
		return a
	}

	a.state = ViewPicker
	a.picker = picker.NewMultiOwner(client)
	return a
}

func (a App) WithInitialState(state ViewState) App {
	a.state = state
	a.directMode = true
	return a
}

func (a App) WithInitialView(name string) App {
	a.initialViewName = name
	return a
}

func (a App) Init() tea.Cmd {
	switch a.state {
	case ViewSetup:
		return a.setup.Init()
	case ViewPicker:
		return a.picker.Init()
	case ViewLoading:
		cfg := a.config
		client := a.client
		owner := cfg.DefaultOwner
		number := cfg.DefaultProject
		return func() tea.Msg {
			project, err := client.GetProject(owner, number)
			if err != nil {
				return projectResolvedMsg{err: err}
			}

			views, err := client.GetProjectViews(project.ID)
			if err != nil {
				return projectResolvedMsg{err: err}
			}

			return projectResolvedMsg{project: project, views: views}
		}
	}

	return nil
}

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if a.help.IsVisible() {
		var cmd tea.Cmd
		a.help, cmd = a.help.Update(msg)
		return a, cmd
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.help, _ = a.help.Update(msg)
		if a.state == ViewSetup {
			var cmd tea.Cmd
			a.setup, cmd = a.setup.Update(msg)
			return a, cmd
		}
		if a.state == ViewPicker {
			var cmd tea.Cmd
			a.picker, cmd = a.picker.Update(msg)
			return a, cmd
		}
		if a.state == ViewViewPicker {
			var cmd tea.Cmd
			a.viewpicker, cmd = a.viewpicker.Update(msg)
			return a, cmd
		}
		if a.state == ViewBoard {
			var cmd tea.Cmd
			a.board, cmd = a.board.Update(msg)
			return a, cmd
		}
		if a.state == ViewDetail {
			var cmd tea.Cmd
			a.detail, cmd = a.detail.Update(msg)
			return a, cmd
		}
		return a, nil
	case setup.SetupCompleteMsg:
		a.config.DefaultOwner = msg.Owner
		if msg.Project != 0 {
			a.config.DefaultProject = msg.Project
		}
		if msg.View != "" {
			a.config.DefaultView = msg.View
		}
		a.initialViewName = msg.View
		if err := config.Save(a.config); err != nil {
			a.loadErr = fmt.Sprintf("config save failed: %v", err)
		}

		if msg.Project != 0 {
			a.loadErr = ""
			a.selectedProject = nil
			a.state = ViewLoading
			return a, a.Init()
		}

		a.loadErr = ""
		if msg.Owner == "" {
			a.picker = picker.NewMultiOwner(a.client)
		} else {
			a.picker = picker.New(a.client, msg.Owner)
		}
		a.state = ViewPicker
		if a.width > 0 || a.height > 0 {
			a.picker, _ = a.picker.Update(tea.WindowSizeMsg{Width: a.width, Height: a.height})
		}
		return a, a.picker.Init()
	case setup.SetupCancelMsg:
		return a, tea.Quit
	case projectResolvedMsg:
		if msg.err != nil {
			a.loadErr = fmt.Sprintf("Error loading project: %v", msg.err)
			a.state = ViewLoading
			return a, nil
		}

		a.loadErr = ""
		a.selectedProject = msg.project

		if a.initialViewName == "" {
			a.viewpicker = viewpicker.New(a.client, msg.project.ID, msg.project.Title)
			a.state = ViewViewPicker
			if a.width > 0 || a.height > 0 {
				a.viewpicker, _ = a.viewpicker.Update(tea.WindowSizeMsg{Width: a.width, Height: a.height})
			}
			return a, a.viewpicker.Init()
		}

		boardViews := make([]github.ProjectView, 0, len(msg.views))
		for _, v := range msg.views {
			if v.Layout == "BOARD_LAYOUT" {
				boardViews = append(boardViews, v)
			}
		}

		for _, v := range boardViews {
			if strings.EqualFold(v.Name, a.initialViewName) {
				a.board = board.New(a.client, *msg.project)
				a.board.SetActiveView(&v)
				a.state = ViewBoard
				if a.width > 0 || a.height > 0 {
					a.board, _ = a.board.Update(tea.WindowSizeMsg{Width: a.width, Height: a.height})
				}
				return a, a.board.Init()
			}
		}

		a.loadErr = fmt.Sprintf("View %q not found among board views", a.initialViewName)
		a.state = ViewLoading
		return a, nil
	case picker.ProjectSelectedMsg:
		a.loadErr = ""
		a.selectedProject = &msg.Project
		if cfg, err := config.Load(); err == nil {
			cfg.DefaultOwner = msg.Project.Owner
			cfg.DefaultProject = msg.Project.Number
			_ = config.Save(cfg)
		}
		a.viewpicker = viewpicker.New(a.client, msg.Project.ID, msg.Project.Title)
		a.state = ViewViewPicker
		if a.width > 0 || a.height > 0 {
			a.viewpicker, _ = a.viewpicker.Update(tea.WindowSizeMsg{Width: a.width, Height: a.height})
		}
		return a, a.viewpicker.Init()
	case viewpicker.ViewSelectedMsg:
		if a.selectedProject == nil {
			return a, nil
		}
		a.loadErr = ""
		if cfg, err := config.Load(); err == nil {
			cfg.DefaultView = msg.View.Name
			_ = config.Save(cfg)
		}
		a.board = board.New(a.client, *a.selectedProject)
		a.board.SetActiveView(&msg.View)
		a.state = ViewBoard
		if a.width > 0 || a.height > 0 {
			a.board, _ = a.board.Update(tea.WindowSizeMsg{Width: a.width, Height: a.height})
		}
		return a, a.board.Init()
	case board.SwitchViewMsg:
		if a.selectedProject != nil {
			a.viewpicker = viewpicker.New(a.client, a.selectedProject.ID, a.selectedProject.Title)
			a.state = ViewViewPicker
			if a.width > 0 || a.height > 0 {
				a.viewpicker, _ = a.viewpicker.Update(tea.WindowSizeMsg{Width: a.width, Height: a.height})
			}
			return a, a.viewpicker.Init()
		}
		return a, nil
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return a, tea.Quit
		}

		if a.quitConfirm {
			switch msg.String() {
			case "y":
				return a, tea.Quit
			case "n", "esc":
				a.quitConfirm = false
				return a, nil
			}
			return a, nil
		}

		switch {
		case msg.String() == "q":
			a.quitConfirm = true
			return a, nil
		case key.Matches(msg, a.keys.Help):
			a.help.Show()
			return a, nil
		case key.Matches(msg, a.keys.Back):
			if a.state == ViewBoard && a.board.IsShowingSettings() {
				break
			}
			switch a.state {
			case ViewDetail:
				a.board.UpdateItem(a.detail.UpdatedItem())
				a.state = ViewBoard
				a.board.ClearSelection()
				return a, nil
			case ViewViewPicker:
				if !a.directMode {
					a.state = ViewPicker
					return a, nil
				}
			case ViewBoard:
				if !a.directMode {
					a.state = ViewPicker
					return a, nil
				}
			}
		}
	}

	if a.state == ViewSetup {
		var cmd tea.Cmd
		a.setup, cmd = a.setup.Update(msg)
		return a, cmd
	}

	if a.state == ViewPicker {
		var cmd tea.Cmd
		a.picker, cmd = a.picker.Update(msg)
		return a, cmd
	}

	if a.state == ViewViewPicker {
		var cmd tea.Cmd
		a.viewpicker, cmd = a.viewpicker.Update(msg)
		return a, cmd
	}

	if a.state == ViewBoard {
		var cmd tea.Cmd
		a.board, cmd = a.board.Update(msg)
		if item, ok := a.board.SelectedItem(); ok && a.board.IsSelected() {
			projectID := ""
			owner := ""
			if a.selectedProject != nil {
				projectID = a.selectedProject.ID
				owner = a.selectedProject.Owner
			}
			repos := extractRepos(a.board.Items(), owner)
			a.detail = detail.New(a.client, item, projectID, repos, time.Duration(a.config.MergedPRWindow)*time.Hour, a.config.PRFetchLimit)
			if a.width > 0 || a.height > 0 {
				a.detail, _ = a.detail.Update(tea.WindowSizeMsg{Width: a.width, Height: a.height})
			}
			a.state = ViewDetail
			return a, tea.Batch(cmd, a.detail.Init())
		}
		return a, cmd
	}

	if a.state == ViewDetail {
		var cmd tea.Cmd
		a.detail, cmd = a.detail.Update(msg)
		return a, cmd
	}

	return a, nil
}

func extractRepos(items []github.ProjectItem, projectOwner string) []detail.RepoRef {
	seen := make(map[string]struct{}, len(items))
	repos := make([]detail.RepoRef, 0, len(items))
	for _, item := range items {
		owner := strings.TrimSpace(item.RepoOwner)
		repo := strings.TrimSpace(item.RepoName)
		if owner == "" || repo == "" {
			continue
		}
		if projectOwner != "" && owner != projectOwner {
			continue
		}

		key := owner + "/" + repo
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		repos = append(repos, detail.RepoRef{Owner: owner, Name: repo})
	}

	return repos
}

func (a App) View() string {
	var baseView string

	switch a.state {
	case ViewSetup:
		baseView = a.setup.View()
	case ViewPicker:
		baseView = a.picker.View()
	case ViewViewPicker:
		baseView = a.viewpicker.View()
	case ViewBoard:
		baseView = a.board.View()
	case ViewDetail:
		baseView = a.detail.View()
	default:
		content := "Loading... (press q to quit)"
		if a.loadErr != "" {
			content = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Render(a.loadErr)
		}

		if a.width <= 0 || a.height <= 0 {
			baseView = content
		} else {
			baseView = lipgloss.Place(
				a.width,
				a.height,
				lipgloss.Center,
				lipgloss.Center,
				content,
			)
		}
	}

	if a.help.IsVisible() {
		return a.help.View()
	}

	if a.quitConfirm {
		confirmBox := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("8")).
			Foreground(lipgloss.Color("3")).
			Background(lipgloss.Color("235")).
			Padding(0, 1).
			Render("Quit? (y/n)")

		if a.width <= 0 || a.height <= 0 {
			if baseView == "" {
				return confirmBox
			}
			return baseView + "\n" + confirmBox
		}

		return overlayCenter(baseView, confirmBox, a.width, a.height)
	}

	return baseView
}

func overlayCenter(base, overlay string, width, height int) string {
	baseLines := strings.Split(lipgloss.Place(width, height, lipgloss.Left, lipgloss.Top, base), "\n")
	overlayLines := strings.Split(overlay, "\n")

	startRow := max(0, (len(baseLines)-len(overlayLines))/2)
	for i, overlayLine := range overlayLines {
		row := startRow + i
		if row >= len(baseLines) {
			break
		}

		baseLines[row] = lipgloss.PlaceHorizontal(width, lipgloss.Center, overlayLine)
	}

	return strings.Join(baseLines, "\n")
}
