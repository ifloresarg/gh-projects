package tui

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ifloresarg/gh-projects/internal/cache"
	"github.com/ifloresarg/gh-projects/internal/config"
	"github.com/ifloresarg/gh-projects/internal/github"
	"github.com/ifloresarg/gh-projects/internal/tui/board"
	"github.com/ifloresarg/gh-projects/internal/tui/detail"
	"github.com/ifloresarg/gh-projects/internal/tui/picker"
	"github.com/ifloresarg/gh-projects/internal/tui/setup"
	"github.com/ifloresarg/gh-projects/internal/tui/viewpicker"
)

func TestNewAppStartsInSetupWhenDefaultOwnerEmpty(t *testing.T) {
	t.Parallel()

	a := NewApp(config.Config{}, &github.MockClient{})
	if a.state != ViewSetup {
		t.Fatalf("state = %v, want %v", a.state, ViewSetup)
	}
}

func TestNewAppStartsInPickerWhenDefaultOwnerPresent(t *testing.T) {
	t.Parallel()

	a := NewApp(config.Config{DefaultOwner: "octocat"}, &github.MockClient{})
	if a.state != ViewPicker {
		t.Fatalf("state = %v, want %v", a.state, ViewPicker)
	}
}

func TestConfigDirectModeAllDefaults(t *testing.T) {
	t.Parallel()

	app := NewApp(config.Config{DefaultOwner: "org", DefaultProject: 5, DefaultView: "Board"}, &github.MockClient{})

	if app.state != ViewLoading {
		t.Fatalf("state = %v, want %v", app.state, ViewLoading)
	}
	if !app.configDirectMode {
		t.Fatal("configDirectMode = false, want true")
	}
	if app.directMode {
		t.Fatal("directMode = true, want false")
	}
	if app.initialViewName != "Board" {
		t.Fatalf("initialViewName = %q, want %q", app.initialViewName, "Board")
	}
}

func TestConfigDirectModeMissingView(t *testing.T) {
	t.Parallel()

	app := NewApp(config.Config{DefaultOwner: "org", DefaultProject: 5}, &github.MockClient{})

	if app.configDirectMode {
		t.Fatal("configDirectMode = true, want false")
	}
	if app.state == ViewLoading {
		t.Fatalf("state = %v, want state other than %v", app.state, ViewLoading)
	}
}

func TestConfigDirectModeMissingProject(t *testing.T) {
	t.Parallel()

	app := NewApp(config.Config{DefaultOwner: "org", DefaultProject: 0, DefaultView: "Board"}, &github.MockClient{})

	if app.configDirectMode {
		t.Fatal("configDirectMode = true, want false")
	}
}

func TestConfigDirectModeCLIOverride(t *testing.T) {
	t.Parallel()

	app := NewApp(config.Config{
		DefaultOwner:   "flag-owner",
		DefaultProject: 9,
		DefaultView:    "Config Board",
		OwnerFromFlag:  true,
	}, &github.MockClient{})
	app = app.WithInitialState(ViewLoading).WithInitialView("CLI Board")

	if !app.directMode {
		t.Fatal("directMode = false, want true")
	}
	if app.configDirectMode {
		t.Fatal("configDirectMode = true, want false")
	}
	if app.initialViewName != "CLI Board" {
		t.Fatalf("initialViewName = %q, want %q", app.initialViewName, "CLI Board")
	}
	if app.state != ViewLoading {
		t.Fatalf("state = %v, want %v", app.state, ViewLoading)
	}
}

func TestAppSetupCompleteTransitionsToPicker(t *testing.T) {
	t.Parallel()

	a := NewApp(config.Config{}, &github.MockClient{})

	model, cmd := a.Update(setup.SetupCompleteMsg{Owner: "org"})
	updated := model.(App)

	if updated.state != ViewPicker {
		t.Fatalf("state = %v, want %v", updated.state, ViewPicker)
	}
	if updated.config.DefaultOwner != "org" {
		t.Fatalf("config.DefaultOwner = %q, want %q", updated.config.DefaultOwner, "org")
	}
	if cmd == nil {
		t.Fatal("cmd = nil, want picker init command")
	}
}

func TestAppQuitKeys(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		msg  tea.KeyMsg
	}{
		{"q quits immediately", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}},
		{"ctrl+c quits immediately", tea.KeyMsg{Type: tea.KeyCtrlC}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			app := NewApp(config.Config{DefaultOwner: "octocat"}, &github.MockClient{})
			_, cmd := app.Update(tt.msg)
			if !isQuitCmd(cmd) {
				t.Fatalf("isQuitCmd(cmd) = false, want true")
			}
		})
	}
}

func isQuitCmd(cmd tea.Cmd) bool {
	if cmd == nil {
		return false
	}

	_, ok := cmd().(tea.QuitMsg)
	return ok
}

func TestAppProjectSelectionTransitionsToViewPicker(t *testing.T) {
	t.Parallel()

	app := NewApp(config.Config{DefaultOwner: "octocat"}, &github.MockClient{})
	model, cmd := app.Update(picker.ProjectSelectedMsg{Project: github.Project{ID: "PVT_1", Title: "Roadmap", Number: 1, Owner: "octocat"}})
	updated := model.(App)

	if updated.state != ViewViewPicker {
		t.Fatalf("state = %v, want ViewViewPicker", updated.state)
	}
	if updated.selectedProject == nil || updated.selectedProject.ID != "PVT_1" {
		t.Fatalf("selectedProject = %#v, want project PVT_1", updated.selectedProject)
	}
	if cmd == nil {
		t.Fatal("expected viewpicker init command after project selection")
	}
}

func TestProjectSelectedMsgSavesOwnerAndProjectToConfig(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmp, "config"))
	home := filepath.Join(tmp, "home")
	t.Setenv("HOME", home)

	app := NewApp(config.Config{DefaultOwner: "octocat"}, &github.MockClient{})
	_, _ = app.Update(picker.ProjectSelectedMsg{Project: github.Project{ID: "test-id", Title: "Test", Number: 42, Owner: "testorg"}})

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("config.Load(): %v", err)
	}
	if cfg.DefaultOwner != "testorg" {
		t.Fatalf("cfg.DefaultOwner = %q, want %q", cfg.DefaultOwner, "testorg")
	}
	if cfg.DefaultProject != 42 {
		t.Fatalf("cfg.DefaultProject = %d, want %d", cfg.DefaultProject, 42)
	}
}

func TestViewSelectedMsgSavesViewToConfig(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmp, "config"))
	home := filepath.Join(tmp, "home")
	t.Setenv("HOME", home)

	app := NewApp(config.Config{DefaultOwner: "octocat"}, &github.MockClient{})
	app.selectedProject = &github.Project{ID: "test-id", Title: "Test", Number: 42, Owner: "testorg"}

	_, _ = app.Update(viewpicker.ViewSelectedMsg{View: github.ProjectView{ID: "v1", Name: "sprint", Layout: "BOARD_LAYOUT"}})

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("config.Load(): %v", err)
	}
	if cfg.DefaultView != "sprint" {
		t.Fatalf("cfg.DefaultView = %q, want %q", cfg.DefaultView, "sprint")
	}
}

func TestAutoSavePreservesExistingConfigFields(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmp, "config"))
	home := filepath.Join(tmp, "home")
	t.Setenv("HOME", home)

	if err := config.Save(config.Config{CacheTTL: 600}); err != nil {
		t.Fatalf("config.Save(): %v", err)
	}

	app := NewApp(config.Config{DefaultOwner: "octocat"}, &github.MockClient{})
	_, _ = app.Update(picker.ProjectSelectedMsg{Project: github.Project{ID: "test-id", Title: "Test", Number: 42, Owner: "testorg"}})

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("config.Load(): %v", err)
	}
	if cfg.CacheTTL != 600 {
		t.Fatalf("cfg.CacheTTL = %d, want %d", cfg.CacheTTL, 600)
	}
	if cfg.DefaultOwner != "testorg" {
		t.Fatalf("cfg.DefaultOwner = %q, want %q", cfg.DefaultOwner, "testorg")
	}
}

func TestAppHelpOverlayTakesPrecedenceInView(t *testing.T) {
	t.Parallel()

	app := NewApp(config.Config{DefaultOwner: "octocat"}, &github.MockClient{})
	app.help.Show()

	view := app.View()
	if !strings.Contains(view, "Press any key to close") {
		t.Fatalf("View() = %q, want help overlay content", view)
	}

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	updated := model.(App)
	if updated.help.IsVisible() {
		t.Fatal("help overlay remained visible after key press")
	}
}

func TestDetailTransitionReceivesDimensions(t *testing.T) {
	t.Parallel()

	app := NewApp(config.Config{DefaultOwner: "octocat"}, &github.MockClient{})

	model, _ := app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	app = model.(App)

	app.selectedProject = &github.Project{ID: "PVT_1", Title: "Roadmap", Number: 1, Owner: "octocat"}
	app.state = ViewBoard

	testItem := github.ProjectItem{
		ID:            "item-1",
		Title:         "Implement GraphQL client",
		Type:          "Issue",
		StatusID:      "status_todo",
		RepoOwner:     "octocat",
		RepoName:      "gh-projects",
		ContentNumber: 101,
		Content: &github.Issue{
			ID:        "I_kwDOA1-101",
			Number:    101,
			Title:     "Implement GraphQL client",
			Body:      "Build the core GitHub GraphQL client for Projects v2.",
			State:     "OPEN",
			Author:    github.User{Login: "octocat"},
			RepoOwner: "octocat",
			RepoName:  "gh-projects",
		},
	}
	testFields := []github.ProjectField{{
		ID:       "status-field",
		Name:     "Status",
		DataType: "SINGLE_SELECT",
		Options: []github.FieldOption{{
			ID:   "status_todo",
			Name: "Todo",
		}},
	}}

	app.board.LoadItemsForTest([]github.ProjectItem{testItem}, testFields)

	model, _ = app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	app = model.(App)

	if app.state != ViewDetail {
		t.Fatalf("state = %v, want ViewDetail", app.state)
	}

	view := app.View()
	if !strings.Contains(view, "Loading issue...") {
		t.Fatalf("View() = %q, want it to contain \"Loading issue...\" (detail view loading state)", view)
	}
}

func TestAppInitReturnsCommandForViewLoading(t *testing.T) {
	t.Parallel()

	mockClient := &github.MockClient{
		GetProjectFn: func(owner string, number int) (*github.Project, error) {
			return &github.Project{ID: "PVT_1", Title: "Roadmap", Number: 1, Owner: owner}, nil
		},
		GetProjectViewsFn: func(projectID string) ([]github.ProjectView, error) {
			return []github.ProjectView{
				{ID: "view-1", Name: "Kanban", Layout: "BOARD_LAYOUT", Number: 1},
			}, nil
		},
	}

	app := NewApp(config.Config{DefaultOwner: "octocat", DefaultProject: 1}, mockClient)
	app = app.WithInitialState(ViewLoading)

	cmd := app.Init()

	if cmd == nil {
		t.Fatal("Init() returned nil command for ViewLoading state, expected non-nil command")
	}
}

func TestAppProjectResolvedNoViewTransitionsToViewPicker(t *testing.T) {
	t.Parallel()

	app := NewApp(config.Config{DefaultOwner: "octocat"}, &github.MockClient{})
	app = app.WithInitialState(ViewLoading)
	app = app.WithInitialView("")

	project := &github.Project{ID: "PVT_1", Title: "Roadmap", Number: 1, Owner: "octocat"}
	views := []github.ProjectView{
		{ID: "view-1", Name: "Kanban", Layout: "BOARD_LAYOUT", Number: 1},
	}

	msg := projectResolvedMsg{
		project: project,
		views:   views,
		err:     nil,
	}

	model, cmd := app.Update(msg)
	updated := model.(App)

	if updated.state != ViewViewPicker {
		t.Fatalf("state = %v, want ViewViewPicker", updated.state)
	}

	if updated.selectedProject == nil || updated.selectedProject.ID != "PVT_1" {
		t.Fatalf("selectedProject = %#v, want project PVT_1", updated.selectedProject)
	}

	if cmd == nil {
		t.Fatal("expected viewpicker init command after project resolution with no initial view")
	}

	if updated.loadErr != "" {
		t.Fatalf("loadErr = %q, want empty string", updated.loadErr)
	}
}

func TestAppProjectResolvedWithMatchingViewTransitionsToBoard(t *testing.T) {
	t.Parallel()

	app := NewApp(config.Config{DefaultOwner: "octocat"}, &github.MockClient{})
	app = app.WithInitialState(ViewLoading)
	app = app.WithInitialView("Kanban")

	project := &github.Project{ID: "PVT_1", Title: "Roadmap", Number: 1, Owner: "octocat"}
	views := []github.ProjectView{
		{ID: "view-1", Name: "Kanban", Layout: "BOARD_LAYOUT", Number: 1},
		{ID: "view-2", Name: "Table", Layout: "TABLE_LAYOUT", Number: 2},
	}

	msg := projectResolvedMsg{
		project: project,
		views:   views,
		err:     nil,
	}

	model, cmd := app.Update(msg)
	updated := model.(App)

	if updated.state != ViewBoard {
		t.Fatalf("state = %v, want ViewBoard", updated.state)
	}

	if updated.selectedProject == nil || updated.selectedProject.ID != "PVT_1" {
		t.Fatalf("selectedProject = %#v, want project PVT_1", updated.selectedProject)
	}

	if cmd == nil {
		t.Fatal("expected board init command after project resolution with matching view")
	}

	if updated.loadErr != "" {
		t.Fatalf("loadErr = %q, want empty string", updated.loadErr)
	}
}

func TestAppProjectResolvedWithNonMatchingViewSetsError(t *testing.T) {
	t.Parallel()

	app := NewApp(config.Config{DefaultOwner: "octocat"}, &github.MockClient{})
	app = app.WithInitialState(ViewLoading)
	app = app.WithInitialView("NonExistent")

	project := &github.Project{ID: "PVT_1", Title: "Roadmap", Number: 1, Owner: "octocat"}
	views := []github.ProjectView{
		{ID: "view-1", Name: "Kanban", Layout: "BOARD_LAYOUT", Number: 1},
	}

	msg := projectResolvedMsg{
		project: project,
		views:   views,
		err:     nil,
	}

	model, _ := app.Update(msg)
	updated := model.(App)

	if updated.state != ViewLoading {
		t.Fatalf("state = %v, want ViewLoading", updated.state)
	}

	if updated.loadErr == "" {
		t.Fatal("loadErr should be set when view is not found, but got empty string")
	}

	if !strings.Contains(updated.loadErr, "NonExistent") {
		t.Fatalf("loadErr = %q, want it to contain \"NonExistent\"", updated.loadErr)
	}
}

func TestExtractReposFiltersByProjectOwner(t *testing.T) {
	t.Parallel()

	items := []github.ProjectItem{
		{RepoOwner: "hifihub", RepoName: "app"},
		{RepoOwner: "octocat", RepoName: "other"},
		{RepoOwner: "hifihub", RepoName: "app"},
		{RepoOwner: "hifihub", RepoName: "api"},
	}

	got := extractRepos(items, "hifihub")
	want := []detail.RepoRef{{Owner: "hifihub", Name: "app"}, {Owner: "hifihub", Name: "api"}}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("extractRepos() = %#v, want %#v", got, want)
	}
}

func TestExtractReposWithEmptyOwnerReturnsAllRepos(t *testing.T) {
	t.Parallel()

	items := []github.ProjectItem{
		{RepoOwner: "hifihub", RepoName: "app"},
		{RepoOwner: "octocat", RepoName: "other"},
		{RepoOwner: "hifihub", RepoName: "app"},
		{RepoOwner: " ", RepoName: "ignored"},
	}

	got := extractRepos(items, "")
	want := []detail.RepoRef{{Owner: "hifihub", Name: "app"}, {Owner: "octocat", Name: "other"}}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("extractRepos() = %#v, want %#v", got, want)
	}
}

func TestProjectKeyNavigationFromBoard(t *testing.T) {
	t.Parallel()

	app := NewApp(config.Config{}, &github.MockClient{})
	app.state = ViewBoard
	app.selectedProject = &github.Project{ID: "1", Title: "Test"}
	app.board = board.New(&github.MockClient{}, github.Project{ID: "1", Title: "Test"})

	updatedModel, _ := app.Update(board.SwitchProjectMsg{})
	updated := updatedModel.(App)

	if updated.state != ViewPicker {
		t.Fatalf("state = %v, want ViewPicker after SwitchProjectMsg in board", updated.state)
	}
}

func TestProjectKeyNavigationFromViewPicker(t *testing.T) {
	t.Parallel()

	app := NewApp(config.Config{}, &github.MockClient{})
	app.state = ViewViewPicker
	app.selectedProject = &github.Project{ID: "1", Title: "Test"}
	app.viewpicker = viewpicker.New(&github.MockClient{}, "1", "Test")

	updatedModel, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	updated := updatedModel.(App)

	if updated.state != ViewPicker {
		t.Fatalf("state = %v, want ViewPicker after 'p' key press from ViewViewPicker", updated.state)
	}
}

func TestOwnersKeyNavigationFromPicker(t *testing.T) {
	t.Parallel()

	app := NewApp(config.Config{}, &github.MockClient{})
	app.state = ViewPicker
	app.picker = picker.NewMultiOwner(&github.MockClient{})

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}})
	updated := model.(App)

	if updated.state != ViewSetup {
		t.Fatalf("state = %v, want ViewSetup after 'o' key in picker", updated.state)
	}
}

func TestProjectKeyWorksInConfigDirectMode(t *testing.T) {
	t.Parallel()

	app := NewApp(config.Config{DefaultOwner: "org", DefaultProject: 5, DefaultView: "Board"}, &github.MockClient{})
	app.state = ViewBoard
	app.selectedProject = &github.Project{ID: "1", Title: "Test"}
	app.board = board.New(&github.MockClient{}, github.Project{ID: "1", Title: "Test"})

	if !app.configDirectMode {
		t.Fatal("configDirectMode = false, want true")
	}

	updatedModel, _ := app.Update(board.SwitchProjectMsg{})
	updated := updatedModel.(App)

	if updated.state != ViewPicker {
		t.Fatalf("state = %v, want ViewPicker even in configDirectMode", updated.state)
	}
}

func TestConfigDirectModeFallback(t *testing.T) {
	t.Parallel()

	mockErr := fmt.Errorf("project not found")
	mockClient := &github.MockClient{
		GetProjectFn: func(owner string, number int) (*github.Project, error) {
			return nil, mockErr
		},
	}

	app := NewApp(config.Config{DefaultOwner: "org", DefaultProject: 5, DefaultView: "Board"}, mockClient)

	if !app.configDirectMode {
		t.Fatal("configDirectMode = false, want true")
	}
	if app.state != ViewLoading {
		t.Fatalf("initial state = %v, want ViewLoading", app.state)
	}

	msg := projectResolvedMsg{err: mockErr}
	model, cmd := app.Update(msg)
	updated := model.(App)

	if updated.state != ViewPicker {
		t.Fatalf("state = %v, want ViewPicker after fallback", updated.state)
	}
	if updated.configDirectMode {
		t.Fatal("configDirectMode = true after fallback, want false")
	}
	if updated.loadErr == "" {
		t.Fatal("loadErr should be set with fallback message")
	}
	if !strings.Contains(updated.loadErr, "Default project not found") {
		t.Fatalf("loadErr = %q, want it to contain 'Default project not found'", updated.loadErr)
	}
	if cmd == nil {
		t.Fatal("expected picker init command after fallback")
	}
}

func TestCLIDirectModeErrorUnchanged(t *testing.T) {
	t.Parallel()

	mockErr := fmt.Errorf("project not found")
	mockClient := &github.MockClient{
		GetProjectFn: func(owner string, number int) (*github.Project, error) {
			return nil, mockErr
		},
	}

	app := NewApp(config.Config{DefaultOwner: "org", DefaultProject: 5}, mockClient)
	app = app.WithInitialState(ViewLoading).WithInitialView("Kanban")

	if app.directMode != true {
		t.Fatal("directMode = false, want true")
	}
	if app.configDirectMode {
		t.Fatal("configDirectMode = true, want false")
	}

	msg := projectResolvedMsg{err: mockErr}
	model, _ := app.Update(msg)
	updated := model.(App)

	if updated.state != ViewLoading {
		t.Fatalf("state = %v, want ViewLoading (unchanged CLI error behavior)", updated.state)
	}
	if updated.loadErr == "" {
		t.Fatal("loadErr should be set for CLI direct mode error")
	}
	if !strings.Contains(updated.loadErr, "Error loading project") {
		t.Fatalf("loadErr = %q, want it to start with 'Error loading project'", updated.loadErr)
	}
}

func TestSWRFirstRun(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("XDG_CACHE_HOME", filepath.Join(tmp, "cache"))

	cacheDir, err := os.UserCacheDir()
	if err != nil {
		t.Fatalf("UserCacheDir() error = %v", err)
	}

	diskCache, err := cache.NewDiskCache(filepath.Join(cacheDir, "gh-projects"))
	if err != nil {
		t.Fatalf("NewDiskCache() error = %v", err)
	}

	projectCalls := 0
	viewCalls := 0
	inner := &github.MockClient{
		GetProjectFn: func(owner string, number int) (*github.Project, error) {
			projectCalls++
			return &github.Project{ID: "PVT_1", Title: "Roadmap", Number: number, Owner: owner}, nil
		},
		GetProjectViewsFn: func(projectID string) ([]github.ProjectView, error) {
			viewCalls++
			return []github.ProjectView{{ID: "view-1", Name: "Board", Layout: "BOARD_LAYOUT", Number: 1}}, nil
		},
	}

	client := github.NewCachedClient(inner, time.Minute, diskCache)
	app := NewApp(config.Config{DefaultOwner: "org", DefaultProject: 1, DefaultView: "Board"}, client)

	cmd := app.Init()
	if cmd == nil {
		t.Fatal("Init() command is nil, want project load command")
	}

	msg, ok := cmd().(projectResolvedMsg)
	if !ok {
		t.Fatalf("Init() message type = %T, want projectResolvedMsg", cmd())
	}
	if msg.fromCache {
		t.Fatal("projectResolvedMsg.fromCache = true, want false on first run")
	}

	model, nextCmd := app.Update(msg)
	updated := model.(App)

	if updated.state != ViewBoard {
		t.Fatalf("state = %v, want ViewBoard", updated.state)
	}
	if nextCmd == nil {
		t.Fatal("next command is nil, want board init command")
	}
	if projectCalls != 1 {
		t.Fatalf("GetProject calls = %d, want 1", projectCalls)
	}
	if viewCalls != 1 {
		t.Fatalf("GetProjectViews calls = %d, want 1", viewCalls)
	}

	var cachedProject []github.Project
	if err := diskCache.Load("project:org:1", &cachedProject); err != nil {
		t.Fatalf("Load(project:org:1) error = %v", err)
	}
	if len(cachedProject) != 1 || cachedProject[0].ID != "PVT_1" {
		t.Fatalf("cached project = %#v, want PVT_1", cachedProject)
	}
}

func TestSWRRefreshFailure(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("XDG_CACHE_HOME", filepath.Join(tmp, "cache"))

	cacheDir, err := os.UserCacheDir()
	if err != nil {
		t.Fatalf("UserCacheDir() error = %v", err)
	}

	diskCache, err := cache.NewDiskCache(filepath.Join(cacheDir, "gh-projects"))
	if err != nil {
		t.Fatalf("NewDiskCache() error = %v", err)
	}

	if err := diskCache.Save("project:org:1", []github.Project{{ID: "PVT_1", Title: "Roadmap", Number: 1, Owner: "org"}}); err != nil {
		t.Fatalf("Save(project cache) error = %v", err)
	}
	if err := diskCache.Save("views:PVT_1", []github.ProjectView{{ID: "view-1", Name: "Board", Layout: "BOARD_LAYOUT", Number: 1}}); err != nil {
		t.Fatalf("Save(views cache) error = %v", err)
	}

	client := github.NewCachedClient(&github.MockClient{
		GetProjectFn: func(owner string, number int) (*github.Project, error) {
			return nil, errors.New("refresh failed")
		},
	}, time.Minute, diskCache)

	app := NewApp(config.Config{DefaultOwner: "org", DefaultProject: 1, DefaultView: "Board"}, client)
	initCmd := app.Init()
	if initCmd == nil {
		t.Fatal("Init() command is nil, want cache-hit command")
	}

	resolved := initCmd().(projectResolvedMsg)
	if !resolved.fromCache {
		t.Fatal("projectResolvedMsg.fromCache = false, want true on warm cache")
	}

	model, _ := app.Update(resolved)
	updated := model.(App)
	if updated.state != ViewBoard {
		t.Fatalf("state = %v, want ViewBoard after cached startup", updated.state)
	}

	refresh := backgroundRefreshCmd(client, "org", 1)().(backgroundRefreshMsg)
	if refresh.err == nil {
		t.Fatal("backgroundRefreshMsg.err = nil, want error (InvalidateAll clears disk, so no fallback on API failure)")
	}
	if refresh.project != nil {
		t.Fatal("backgroundRefreshMsg.project != nil, want nil on API failure with no disk cache")
	}
}

func TestSWRRefreshUpdate(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("XDG_CACHE_HOME", filepath.Join(tmp, "cache"))

	cacheDir, err := os.UserCacheDir()
	if err != nil {
		t.Fatalf("UserCacheDir() error = %v", err)
	}

	diskCache, err := cache.NewDiskCache(filepath.Join(cacheDir, "gh-projects"))
	if err != nil {
		t.Fatalf("NewDiskCache() error = %v", err)
	}

	if err := diskCache.Save("project:org:1", []github.Project{{ID: "PVT_1", Title: "Roadmap", Number: 1, Owner: "org", ItemCount: 3}}); err != nil {
		t.Fatalf("Save(project cache) error = %v", err)
	}
	if err := diskCache.Save("views:PVT_1", []github.ProjectView{{ID: "view-1", Name: "Board", Layout: "BOARD_LAYOUT", Number: 1}}); err != nil {
		t.Fatalf("Save(views cache) error = %v", err)
	}

	freshItems := []github.ProjectItem{
		{ID: "i-1", Title: "one", Type: "Issue", StatusID: "todo", Content: &github.Issue{Title: "one"}},
		{ID: "i-2", Title: "two", Type: "Issue", StatusID: "todo", Content: &github.Issue{Title: "two"}},
		{ID: "i-3", Title: "three", Type: "Issue", StatusID: "todo", Content: &github.Issue{Title: "three"}},
		{ID: "i-4", Title: "four", Type: "Issue", StatusID: "todo", Content: &github.Issue{Title: "four"}},
		{ID: "i-5", Title: "five", Type: "Issue", StatusID: "todo", Content: &github.Issue{Title: "five"}},
	}

	fields := []github.ProjectField{{
		ID:       "status-field",
		Name:     "Status",
		DataType: "SINGLE_SELECT",
		Options:  []github.FieldOption{{ID: "todo", Name: "Todo"}},
	}}

	client := github.NewCachedClient(&github.MockClient{
		GetProjectFn: func(owner string, number int) (*github.Project, error) {
			return &github.Project{ID: "PVT_1", Title: "Roadmap", Number: 1, Owner: "org", ItemCount: 5}, nil
		},
		GetProjectViewsFn: func(projectID string) ([]github.ProjectView, error) {
			return []github.ProjectView{{ID: "view-1", Name: "Board", Layout: "BOARD_LAYOUT", Number: 1}}, nil
		},
		GetProjectItemsFn: func(projectID string) ([]github.ProjectItem, error) {
			return freshItems, nil
		},
		GetProjectFieldsFn: func(projectID string) ([]github.ProjectField, error) {
			return fields, nil
		},
	}, time.Minute, diskCache)

	app := NewApp(config.Config{DefaultOwner: "org", DefaultProject: 1, DefaultView: "Board"}, client)
	resolved := app.Init()().(projectResolvedMsg)
	if !resolved.fromCache {
		t.Fatal("projectResolvedMsg.fromCache = false, want true on warm cache")
	}

	model, _ := app.Update(resolved)
	updated := model.(App)
	if updated.state != ViewBoard {
		t.Fatalf("state = %v, want ViewBoard", updated.state)
	}

	staleItems := freshItems[:3]
	updated.board.LoadItemsForTest(staleItems, fields)

	refresh := backgroundRefreshCmd(client, "org", 1)().(backgroundRefreshMsg)
	if refresh.err != nil {
		t.Fatalf("backgroundRefreshMsg.err = %v, want nil", refresh.err)
	}
	if len(refresh.items) != 5 {
		t.Fatalf("refresh item count = %d, want 5", len(refresh.items))
	}

	model, refreshCmd := updated.Update(refresh)
	updated = model.(App)
	if updated.state != ViewBoard {
		t.Fatalf("state = %v, want ViewBoard after refresh update", updated.state)
	}
	if updated.loadErr != "Board updated" {
		t.Fatalf("loadErr = %q, want %q", updated.loadErr, "Board updated")
	}
	if refreshCmd == nil {
		t.Fatal("refresh command is nil, want board re-init command")
	}
}
