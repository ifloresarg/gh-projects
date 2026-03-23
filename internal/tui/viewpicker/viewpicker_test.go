package viewpicker

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ifloresarg/gh-projects/internal/github"
)

func TestViewPicker_LoadsViews(t *testing.T) {
	t.Parallel()

	client := &github.MockClient{
		GetProjectViewsFn: func(projectID string) ([]github.ProjectView, error) {
			return []github.ProjectView{
				{ID: "v1", Name: "All Items", Number: 1, Layout: "BOARD_LAYOUT", Filter: ""},
				{ID: "v2", Name: "Development", Number: 2, Layout: "BOARD_LAYOUT", Filter: `-status:Backlog,"To Design"`},
				{ID: "v3", Name: "Table", Number: 3, Layout: "TABLE_LAYOUT", Filter: ""},
			}, nil
		},
	}

	m := New(client, "project-1", "Platform Roadmap")
	m, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	msg := fetchViews(client, "project-1")()
	loaded, ok := msg.(viewsLoadedMsg)
	if !ok {
		t.Fatalf("fetchViews() message type = %T, want viewsLoadedMsg", msg)
	}
	if len(loaded.views) != 2 {
		t.Fatalf("filtered views = %d, want 2", len(loaded.views))
	}

	m, _ = m.Update(loaded)
	if len(m.list.Items()) != 2 {
		t.Fatalf("list item count = %d, want 2", len(m.list.Items()))
	}

	view := m.View()
	for _, fragment := range []string{"Views · Platform Roadmap", "All Items", "All items", "Development", `-status:Backlog,"To Design"`} {
		if !strings.Contains(view, fragment) {
			t.Fatalf("View() missing %q in %q", fragment, view)
		}
	}
	if strings.Contains(view, "Table") {
		t.Fatalf("View() unexpectedly contains filtered table view: %q", view)
	}
}

func TestViewPicker_AutoSelectSingleView(t *testing.T) {
	t.Parallel()

	m := New(&github.MockClient{}, "project-1", "Platform Roadmap")
	singleView := github.ProjectView{ID: "v1", Name: "All Items", Number: 1, Layout: "BOARD_LAYOUT", Filter: ""}

	updated, cmd := m.Update(viewsLoadedMsg{views: []github.ProjectView{singleView}})
	if cmd == nil {
		t.Fatal("Update() cmd = nil, want ViewSelectedMsg command")
	}

	msg := cmd()
	selected, ok := msg.(ViewSelectedMsg)
	if !ok {
		t.Fatalf("Update() cmd message type = %T, want ViewSelectedMsg", msg)
	}
	if selected.View != singleView {
		t.Fatalf("selected view = %#v, want %#v", selected.View, singleView)
	}
	if updated.loading {
		t.Fatal("view picker remained in loading state after views loaded")
	}
}

func TestViewPicker_Selection(t *testing.T) {
	t.Parallel()

	m := New(&github.MockClient{}, "project-1", "Platform Roadmap")
	m, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m, _ = m.Update(viewsLoadedMsg{views: []github.ProjectView{
		{ID: "v1", Name: "All Items", Number: 1, Layout: "BOARD_LAYOUT", Filter: ""},
		{ID: "v2", Name: "Development", Number: 2, Layout: "BOARD_LAYOUT", Filter: `-status:Backlog`},
	}})

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("Update() cmd = nil, want ViewSelectedMsg command")
	}

	msg := cmd()
	selected, ok := msg.(ViewSelectedMsg)
	if !ok {
		t.Fatalf("Update() cmd message type = %T, want ViewSelectedMsg", msg)
	}
	if selected.View.Name != "Development" {
		t.Fatalf("selected view = %#v", selected.View)
	}
	if updated.loading {
		t.Fatal("view picker remained in loading state after selection")
	}
}
