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

	m := New(&github.MockClient{}, "ifloresarg")
	m, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m, _ = m.Update(projectsLoadedMsg{projects: []github.Project{{ID: "p1", Number: 1, Title: "Platform Roadmap", ItemCount: 12}}})

	view := m.View()
	for _, fragment := range []string{"GitHub Projects · ifloresarg", "#1 Platform Roadmap", "12 items"} {
		if !strings.Contains(view, fragment) {
			t.Fatalf("View() missing %q in %q", fragment, view)
		}
	}
}

func TestPickerEnterReturnsSelectedProjectMsg(t *testing.T) {
	t.Parallel()

	m := New(&github.MockClient{}, "ifloresarg")
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

	m := New(&github.MockClient{}, "ifloresarg")
	m, _ = m.Update(tea.WindowSizeMsg{Width: 60, Height: 20})
	m, _ = m.Update(projectsLoadedMsg{err: errors.New("network error")})

	view := m.View()
	if !strings.Contains(view, "Error loading projects: network error") {
		t.Fatalf("View() = %q, want error message", view)
	}
}
