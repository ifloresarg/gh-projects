package picker

import (
	"errors"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ifloresarg/gh-projects/internal/github"
)

func TestPickerViewShowsProjectsAfterLoad(t *testing.T) {
	t.Parallel()

	m := New(&github.MockClient{}, "octocat")
	m, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m, _ = m.Update(projectsLoadedMsg{projects: []github.Project{{ID: "p1", Number: 1, Title: "Platform Roadmap", ItemCount: 12}}})

	view := m.View()
	for _, fragment := range []string{"GitHub Projects · octocat", "#1 Platform Roadmap", "12 items"} {
		if !strings.Contains(view, fragment) {
			t.Fatalf("View() missing %q in %q", fragment, view)
		}
	}
}

func TestPickerEnterReturnsSelectedProjectMsg(t *testing.T) {
	t.Parallel()

	m := New(&github.MockClient{}, "octocat")
	m, _ = m.Update(projectsLoadedMsg{projects: []github.Project{{ID: "p1", Number: 3, Title: "Engineering Backlog", ItemCount: 27}}})

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	msg := cmd()
	selected, ok := msg.(ProjectSelectedMsg)
	if !ok {
		t.Fatalf("Update() cmd message type = %T, want ProjectSelectedMsg", msg)
	}
	if selected.Project.Title != "Engineering Backlog" {
		t.Fatalf("selected project = %#v", selected.Project)
	}
	if updated.loading {
		t.Fatal("picker remained in loading state after projects loaded")
	}
}

func TestPickerViewShowsErrorState(t *testing.T) {
	t.Parallel()

	m := New(&github.MockClient{}, "octocat")
	m, _ = m.Update(tea.WindowSizeMsg{Width: 60, Height: 20})
	m, _ = m.Update(projectsLoadedMsg{err: errors.New("network error")})

	view := m.View()
	if !strings.Contains(view, "Error loading projects: network error") {
		t.Fatalf("View() = %q, want error message", view)
	}
}

func TestNewMultiOwnerInitializesCorrectly(t *testing.T) {
	t.Parallel()

	m := NewMultiOwner(&github.MockClient{})

	if !m.multiOwner {
		t.Fatal("multiOwner = false, want true")
	}
	if m.list.Title != "GitHub Projects · All Organizations" {
		t.Fatalf("list.Title = %q, want 'GitHub Projects · All Organizations'", m.list.Title)
	}
	if m.loading != true {
		t.Fatal("loading = false, want true")
	}
}

func TestMultiOwnerItemTitleIncludesOwner(t *testing.T) {
	t.Parallel()

	itm := item{
		project: github.Project{
			ID:     "p1",
			Number: 3,
			Title:  "My Project",
			Owner:  "acme",
		},
		multiOwner: true,
	}

	title := itm.Title()
	if title != "acme · #3 My Project" {
		t.Fatalf("Title() = %q, want 'acme · #3 My Project'", title)
	}
}

func TestMultiOwnerItemFilterValueIncludesOwner(t *testing.T) {
	t.Parallel()

	itm := item{
		project: github.Project{
			ID:     "p1",
			Number: 3,
			Title:  "My Project",
			Owner:  "acme",
		},
		multiOwner: true,
	}

	filterVal := itm.FilterValue()
	if filterVal != "acme My Project" {
		t.Fatalf("FilterValue() = %q, want 'acme My Project'", filterVal)
	}
}

func TestSingleOwnerItemTitleNoPrefix(t *testing.T) {
	t.Parallel()

	itm := item{
		project: github.Project{
			ID:     "p1",
			Number: 3,
			Title:  "My Project",
			Owner:  "acme",
		},
		multiOwner: false,
	}

	title := itm.Title()
	if title != "#3 My Project" {
		t.Fatalf("Title() = %q, want '#3 My Project'", title)
	}
}

func TestMultiOwnerViewShowsProjectsWithOwnerPrefix(t *testing.T) {
	t.Parallel()

	m := NewMultiOwner(&github.MockClient{})
	m, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m, _ = m.Update(projectsLoadedMsg{projects: []github.Project{
		{ID: "p1", Number: 1, Title: "ProjectA", Owner: "owner1", ItemCount: 5},
		{ID: "p2", Number: 2, Title: "ProjectB", Owner: "owner2", ItemCount: 8},
	}})

	view := m.View()
	for _, fragment := range []string{"owner1 · #1 ProjectA", "owner2 · #2 ProjectB", "5 items", "8 items"} {
		if !strings.Contains(view, fragment) {
			t.Fatalf("View() missing %q in %q", fragment, view)
		}
	}
}

func TestMultiOwnerPartialScopeErrorShowsInfoNote(t *testing.T) {
	t.Parallel()

	m := NewMultiOwner(&github.MockClient{})
	m, _ = m.Update(projectsLoadedMsg{
		projects: []github.Project{
			{ID: "p1", Number: 1, Title: "PersonalProject", Owner: "octocat", ItemCount: 3},
		},
		err: github.ErrMissingScopeReadOrg,
	})

	if m.infoNote == "" {
		t.Fatal("infoNote is empty, want non-empty info message")
	}
	if m.err != nil {
		t.Fatalf("err = %v, want nil (partial scope error should not be fatal)", m.err)
	}
	if m.loading {
		t.Fatal("loading = true, want false (should finish loading even with partial scope)")
	}

	view := m.View()
	if !strings.Contains(view, "Note: some organization projects may be unavailable.") {
		t.Fatalf("View() missing info note in %q", view)
	}
}
