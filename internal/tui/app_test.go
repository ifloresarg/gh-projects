package tui

import (
	"reflect"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ifloresarg/gh-projects/internal/config"
	"github.com/ifloresarg/gh-projects/internal/github"
	"github.com/ifloresarg/gh-projects/internal/tui/detail"
	"github.com/ifloresarg/gh-projects/internal/tui/picker"
	"github.com/ifloresarg/gh-projects/internal/tui/setup"
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

func TestAppQuitConfirmationFlow(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		app             App
		msg             tea.KeyMsg
		wantQuitConfirm bool
		wantQuit        bool
	}{
		{
			name:            "q sets confirmation without quitting",
			app:             NewApp(config.Config{DefaultOwner: "octocat"}, &github.MockClient{}),
			msg:             tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}},
			wantQuitConfirm: true,
		},
		{
			name:            "y quits when confirming",
			app:             App{quitConfirm: true},
			msg:             tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}},
			wantQuitConfirm: true,
			wantQuit:        true,
		},
		{
			name:            "n cancels confirmation",
			app:             App{quitConfirm: true},
			msg:             tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}},
			wantQuitConfirm: false,
		},
		{
			name:            "ctrl+c quits immediately",
			app:             NewApp(config.Config{DefaultOwner: "octocat"}, &github.MockClient{}),
			msg:             tea.KeyMsg{Type: tea.KeyCtrlC},
			wantQuitConfirm: false,
			wantQuit:        true,
		},
		{
			name:            "ctrl+c quits immediately while confirming",
			app:             App{quitConfirm: true},
			msg:             tea.KeyMsg{Type: tea.KeyCtrlC},
			wantQuitConfirm: true,
			wantQuit:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			model, cmd := tt.app.Update(tt.msg)
			updated := model.(App)

			if updated.quitConfirm != tt.wantQuitConfirm {
				t.Fatalf("quitConfirm = %v, want %v", updated.quitConfirm, tt.wantQuitConfirm)
			}

			if gotQuit := isQuitCmd(cmd); gotQuit != tt.wantQuit {
				t.Fatalf("isQuitCmd(cmd) = %v, want %v", gotQuit, tt.wantQuit)
			}
		})
	}
}

func TestAppViewShowsQuitConfirmationOverlay(t *testing.T) {
	t.Parallel()

	app := NewApp(config.Config{DefaultOwner: "octocat"}, &github.MockClient{})
	app.quitConfirm = true

	view := app.View()
	if !strings.Contains(view, "Quit? (y/n)") {
		t.Fatalf("View() = %q, want quit confirmation overlay", view)
	}
}

func TestAppQuitConfirmationSwallowsOtherKeys(t *testing.T) {
	t.Parallel()

	app := NewApp(config.Config{DefaultOwner: "octocat"}, &github.MockClient{})
	app.quitConfirm = true
	app.help.Hide()

	model, cmd := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	updated := model.(App)

	if !updated.quitConfirm {
		t.Fatal("quitConfirm = false, want true")
	}

	if updated.help.IsVisible() {
		t.Fatal("help overlay became visible while quit confirmation was active")
	}

	if cmd != nil {
		t.Fatalf("cmd = %v, want nil", cmd)
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
