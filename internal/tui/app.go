package tui

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
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
	config           config.Config
	client           github.GitHubClient
	state            ViewState
	loadErr          string
	setup            setup.Model
	picker           picker.Model
	viewpicker       viewpicker.Model
	board            board.Model
	detail           detail.Model
	help             help.Model
	selectedProject  *github.Project
	width            int
	height           int
	keys             KeyMap
	owner            string
	projectNumber    int
	initialViewName  string
	directMode       bool
	configDirectMode bool
}

type projectResolvedMsg struct {
	project   *github.Project
	views     []github.ProjectView
	err       error
	fromCache bool
}

type backgroundRefreshMsg struct {
	project *github.Project
	views   []github.ProjectView
	items   []github.ProjectItem
	fields  []github.ProjectField
	err     error
}

const appDiskCacheVersion = 1

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

	if !a.directMode && cfg.DefaultOwner != "" && cfg.DefaultProject != 0 && cfg.DefaultView != "" {
		a.configDirectMode = true
		a.state = ViewLoading
		a.initialViewName = cfg.DefaultView
		if a.owner == "" {
			a.owner = cfg.DefaultOwner
		}
		if a.projectNumber == 0 {
			a.projectNumber = cfg.DefaultProject
		}
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
		if a.owner != "" {
			owner = a.owner
		}
		if a.projectNumber != 0 {
			number = a.projectNumber
		}

		if a.configDirectMode && hasWarmProjectCache(owner, number) {
			project, err := client.GetProject(owner, number)
			if err == nil {
				views, err := client.GetProjectViews(project.ID)
				if err == nil {
					return func() tea.Msg {
						return projectResolvedMsg{project: project, views: views, fromCache: true}
					}
				}
			}
		}

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
		if msg.err != nil && a.configDirectMode {
			a.configDirectMode = false
			a.loadErr = "Default project not found or inaccessible, showing project picker"
			a.picker = picker.NewMultiOwner(a.client)
			a.state = ViewPicker
			if a.width > 0 || a.height > 0 {
				a.picker, _ = a.picker.Update(tea.WindowSizeMsg{Width: a.width, Height: a.height})
			}
			return a, a.picker.Init()
		}
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
				if msg.fromCache && a.configDirectMode {
					return a, tea.Batch(
						a.board.Init(),
						backgroundRefreshCmd(a.client, msg.project.Owner, msg.project.Number),
					)
				}
				return a, a.board.Init()
			}
		}

		a.loadErr = fmt.Sprintf("View %q not found among board views", a.initialViewName)
		a.state = ViewLoading
		return a, nil
	case backgroundRefreshMsg:
		if msg.err != nil {
			a.loadErr = "⚠ Using cached data, refresh failed"
			return a, nil
		}

		if a.state != ViewBoard || a.selectedProject == nil {
			return a, nil
		}

		if reflect.DeepEqual(a.selectedProject, msg.project) && reflect.DeepEqual(a.board.Items(), msg.items) {
			return a, nil
		}

		a.selectedProject = msg.project
		a.loadErr = "Board updated"
		return a, a.board.Init()
	case picker.ProjectSelectedMsg:
		// Enforce single-project disk cache: clear stale project data before switching.
		if cc, ok := a.client.(*github.CachedClient); ok {
			cc.InvalidateAll()
		}
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
	case board.SwitchProjectMsg:
		a.picker = picker.NewMultiOwner(a.client)
		a.state = ViewPicker
		if a.width > 0 || a.height > 0 {
			a.picker, _ = a.picker.Update(tea.WindowSizeMsg{Width: a.width, Height: a.height})
		}
		return a, a.picker.Init()
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return a, tea.Quit
		}

		switch {
		case msg.String() == "q":
			return a, tea.Quit
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
		if keyMsg, ok := msg.(tea.KeyMsg); ok && key.Matches(keyMsg, a.keys.Owners) {
			a.setup = setup.New(a.client)
			a.state = ViewSetup
			return a, a.setup.Init()
		}
		var cmd tea.Cmd
		a.picker, cmd = a.picker.Update(msg)
		return a, cmd
	}

	if a.state == ViewViewPicker {
		if keyMsg, ok := msg.(tea.KeyMsg); ok && key.Matches(keyMsg, a.keys.Projects) {
			a.picker = picker.NewMultiOwner(a.client)
			a.state = ViewPicker
			if a.width > 0 || a.height > 0 {
				a.picker, _ = a.picker.Update(tea.WindowSizeMsg{Width: a.width, Height: a.height})
			}
			return a, a.picker.Init()
		}
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

	if a.loadErr != "" && a.state != ViewLoading {
		color := lipgloss.Color("9")
		if !strings.HasPrefix(a.loadErr, "⚠") && !strings.HasPrefix(a.loadErr, "Error") {
			color = lipgloss.Color("10")
		}
		notice := lipgloss.NewStyle().Foreground(color).Render(a.loadErr)
		if baseView == "" {
			return notice
		}
		return baseView + "\n" + notice
	}

	return baseView
}

func backgroundRefreshCmd(client github.GitHubClient, owner string, number int) tea.Cmd {
	return func() tea.Msg {
		// Force memory cache clear so background refresh actually hits the API,
		// not the warm in-memory cache from the initial disk-cache load.
		if cc, ok := client.(*github.CachedClient); ok {
			cc.InvalidateAll()
		}

		project, err := client.GetProject(owner, number)
		if err != nil {
			return backgroundRefreshMsg{err: err}
		}

		views, err := client.GetProjectViews(project.ID)
		if err != nil {
			return backgroundRefreshMsg{err: err}
		}

		items, err := client.GetProjectItems(project.ID)
		if err != nil {
			return backgroundRefreshMsg{err: err}
		}

		fields, err := client.GetProjectFields(project.ID)
		if err != nil {
			return backgroundRefreshMsg{err: err}
		}

		return backgroundRefreshMsg{project: project, views: views, items: items, fields: fields}
	}
}

func hasWarmProjectCache(owner string, number int) bool {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return false
	}

	projectPath := filepath.Join(cacheDir, "gh-projects", "project:"+owner+":"+fmt.Sprint(number)+".json")
	payload, err := os.ReadFile(projectPath)
	if err != nil {
		return false
	}

	var envelope struct {
		Version int             `json:"version"`
		Data    json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(payload, &envelope); err != nil || envelope.Version != appDiskCacheVersion {
		return false
	}

	var projects []github.Project
	if err := json.Unmarshal(envelope.Data, &projects); err != nil || len(projects) == 0 || projects[0].ID == "" {
		return false
	}

	viewsPath := filepath.Join(cacheDir, "gh-projects", "views:"+projects[0].ID+".json")
	if _, err := os.Stat(viewsPath); err != nil {
		return false
	}

	return true
}
