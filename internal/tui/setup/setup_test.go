package setup

import (
	"errors"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ifloresarg/gh-projects/internal/github"
)

func TestSetupWizardScenarios(t *testing.T) {
	t.Parallel()

	projectA := github.Project{ID: "p1", Title: "Roadmap", Number: 11, Owner: "testorg", ItemCount: 3}
	projectB := github.Project{ID: "p2", Title: "Backlog", Number: 12, Owner: "testorg", ItemCount: 7}
	viewA := github.ProjectView{ID: "v1", Name: "All Items", Layout: "BOARD_LAYOUT"}
	viewB := github.ProjectView{ID: "v2", Name: "Sprint", Layout: "BOARD_LAYOUT"}

	tests := []struct {
		name string
		run  func(*testing.T)
	}{
		{
			name: "step 1 empty owner blocks continue",
			run: func(t *testing.T) {
				t.Parallel()

				m := New(&github.MockClient{})
				updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})

				if updated.step != stepOwner {
					t.Fatalf("step = %v, want %v", updated.step, stepOwner)
				}
				if cmd != nil {
					if _, ok := cmd().(SetupCompleteMsg); ok {
						t.Fatal("unexpected SetupCompleteMsg when owner is empty")
					}
				}
			},
		},
		{
			name: "step 1 non-empty owner proceeds",
			run: func(t *testing.T) {
				t.Parallel()

				m := New(&github.MockClient{})
				m = typeOwner(t, m, "testorg")

				updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
				if updated.step != stepProjectLoading {
					t.Fatalf("step = %v, want %v", updated.step, stepProjectLoading)
				}
				if cmd == nil {
					t.Fatal("cmd = nil, want fetch projects command")
				}
			},
		},
		{
			name: "step 1 esc quits",
			run: func(t *testing.T) {
				t.Parallel()

				m := New(&github.MockClient{})
				_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
				requireCancelMsg(t, cmd)
			},
		},
		{
			name: "step 1 ctrl+c quits",
			run: func(t *testing.T) {
				t.Parallel()

				m := New(&github.MockClient{})
				_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
				requireCancelMsg(t, cmd)
			},
		},
		{
			name: "step 2 projects load and display",
			run: func(t *testing.T) {
				t.Parallel()

				client := &github.MockClient{ListProjectsFn: func(owner string) ([]github.Project, error) {
					return []github.Project{projectA, projectB}, nil
				}}
				m := New(client)
				m, _ = submitOwner(t, m, "testorg")

				if m.step != stepProjectSelect {
					t.Fatalf("step = %v, want %v", m.step, stepProjectSelect)
				}

				view := m.View()
				for _, title := range []string{"Roadmap", "Backlog"} {
					if !strings.Contains(view, title) {
						t.Fatalf("View() missing %q in %q", title, view)
					}
				}
			},
		},
		{
			name: "step 2 tab skips project selection",
			run: func(t *testing.T) {
				t.Parallel()

				client := &github.MockClient{ListProjectsFn: func(owner string) ([]github.Project, error) {
					return []github.Project{projectA}, nil
				}}
				m := New(client)
				m, _ = submitOwner(t, m, "testorg")
				if m.step != stepProjectSelect {
					t.Fatalf("step = %v, want %v", m.step, stepProjectSelect)
				}

				_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyTab})
				complete := requireCompleteMsg(t, cmd)
				if complete.Owner != "testorg" || complete.Project != 0 || complete.View != "" {
					t.Fatalf("SetupCompleteMsg = %#v, want owner=testorg project=0 view=''", complete)
				}
			},
		},
		{
			name: "step 2 enter selects project",
			run: func(t *testing.T) {
				t.Parallel()

				client := &github.MockClient{ListProjectsFn: func(owner string) ([]github.Project, error) {
					return []github.Project{projectA, projectB}, nil
				}}
				m := New(client)
				m, _ = submitOwner(t, m, "testorg")

				updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
				if updated.step != stepViewLoading {
					t.Fatalf("step = %v, want %v", updated.step, stepViewLoading)
				}
				if cmd == nil {
					t.Fatal("cmd = nil, want fetch views command")
				}
			},
		},
		{
			name: "step 2 api error shows error state",
			run: func(t *testing.T) {
				t.Parallel()

				client := &github.MockClient{ListProjectsFn: func(owner string) ([]github.Project, error) {
					return nil, errors.New("api error")
				}}
				m := New(client)
				m, _ = submitOwner(t, m, "testorg")

				if m.step != stepProjectError {
					t.Fatalf("step = %v, want %v", m.step, stepProjectError)
				}
				if !strings.Contains(m.View(), "Error loading projects: api error") {
					t.Fatalf("View() = %q, want api error text", m.View())
				}
			},
		},
		{
			name: "step 2 enter retries on error",
			run: func(t *testing.T) {
				t.Parallel()

				calls := 0
				client := &github.MockClient{ListProjectsFn: func(owner string) ([]github.Project, error) {
					calls++
					if calls == 1 {
						return nil, errors.New("api error")
					}
					return []github.Project{projectA}, nil
				}}
				m := New(client)
				m, _ = submitOwner(t, m, "testorg")
				if m.step != stepProjectError {
					t.Fatalf("step = %v, want %v", m.step, stepProjectError)
				}

				updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
				if updated.step != stepProjectLoading {
					t.Fatalf("step = %v, want %v", updated.step, stepProjectLoading)
				}
				if cmd == nil {
					t.Fatal("cmd = nil, want retry fetch command")
				}
			},
		},
		{
			name: "step 2 tab skips on error",
			run: func(t *testing.T) {
				t.Parallel()

				client := &github.MockClient{ListProjectsFn: func(owner string) ([]github.Project, error) {
					return nil, errors.New("api error")
				}}
				m := New(client)
				m, _ = submitOwner(t, m, "testorg")

				_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyTab})
				complete := requireCompleteMsg(t, cmd)
				if complete.Owner != "testorg" || complete.Project != 0 {
					t.Fatalf("SetupCompleteMsg = %#v, want owner=testorg project=0", complete)
				}
			},
		},
		{
			name: "step 2 zero projects shows empty state",
			run: func(t *testing.T) {
				t.Parallel()

				client := &github.MockClient{ListProjectsFn: func(owner string) ([]github.Project, error) {
					return []github.Project{}, nil
				}}
				m := New(client)
				m, _ = submitOwner(t, m, "testorg")

				if m.step != stepProjectEmpty {
					t.Fatalf("step = %v, want %v", m.step, stepProjectEmpty)
				}
				if !strings.Contains(m.View(), "No projects found") {
					t.Fatalf("View() = %q, want empty state text", m.View())
				}
			},
		},
		{
			name: "step 2 tab skips on empty",
			run: func(t *testing.T) {
				t.Parallel()

				client := &github.MockClient{ListProjectsFn: func(owner string) ([]github.Project, error) {
					return []github.Project{}, nil
				}}
				m := New(client)
				m, _ = submitOwner(t, m, "testorg")

				_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyTab})
				complete := requireCompleteMsg(t, cmd)
				if complete.Owner != "testorg" || complete.Project != 0 {
					t.Fatalf("SetupCompleteMsg = %#v, want owner=testorg project=0", complete)
				}
			},
		},
		{
			name: "step 3 view loads and displays",
			run: func(t *testing.T) {
				t.Parallel()

				client := &github.MockClient{
					ListProjectsFn: func(owner string) ([]github.Project, error) {
						return []github.Project{projectA}, nil
					},
					GetProjectViewsFn: func(projectID string) ([]github.ProjectView, error) {
						return []github.ProjectView{viewA, viewB}, nil
					},
				}

				m := New(client)
				m, _ = submitOwner(t, m, "testorg")
				m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
				m, _, _ = updateFromCmd(t, m, cmd)

				if m.step != stepViewSelect {
					t.Fatalf("step = %v, want %v", m.step, stepViewSelect)
				}
				view := m.View()
				for _, name := range []string{"All Items", "Sprint"} {
					if !strings.Contains(view, name) {
						t.Fatalf("View() missing %q in %q", name, view)
					}
				}
			},
		},
		{
			name: "step 3 single view auto-completes",
			run: func(t *testing.T) {
				t.Parallel()

				client := &github.MockClient{
					ListProjectsFn: func(owner string) ([]github.Project, error) {
						return []github.Project{projectA}, nil
					},
					GetProjectViewsFn: func(projectID string) ([]github.ProjectView, error) {
						return []github.ProjectView{viewA}, nil
					},
				}

				m := New(client)
				m, _ = submitOwner(t, m, "testorg")
				m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
				_, returned, _ := updateFromCmd(t, m, cmd)

				complete := requireCompleteMsg(t, returned)
				if complete.Owner != "testorg" || complete.Project != projectA.Number || complete.View != viewA.Name {
					t.Fatalf("SetupCompleteMsg = %#v, want owner/project/view from first project and view", complete)
				}
			},
		},
		{
			name: "step 3 zero views auto-completes",
			run: func(t *testing.T) {
				t.Parallel()

				client := &github.MockClient{
					ListProjectsFn: func(owner string) ([]github.Project, error) {
						return []github.Project{projectA}, nil
					},
					GetProjectViewsFn: func(projectID string) ([]github.ProjectView, error) {
						return []github.ProjectView{}, nil
					},
				}

				m := New(client)
				m, _ = submitOwner(t, m, "testorg")
				m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
				_, returned, _ := updateFromCmd(t, m, cmd)

				complete := requireCompleteMsg(t, returned)
				if complete.Owner != "testorg" || complete.Project != projectA.Number || complete.View != "" {
					t.Fatalf("SetupCompleteMsg = %#v, want owner=testorg project=%d view=''", complete, projectA.Number)
				}
			},
		},
		{
			name: "step 3 tab skips view",
			run: func(t *testing.T) {
				t.Parallel()

				client := &github.MockClient{
					ListProjectsFn: func(owner string) ([]github.Project, error) {
						return []github.Project{projectA}, nil
					},
					GetProjectViewsFn: func(projectID string) ([]github.ProjectView, error) {
						return []github.ProjectView{viewA, viewB}, nil
					},
				}

				m := New(client)
				m, _ = submitOwner(t, m, "testorg")
				m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
				m, _, _ = updateFromCmd(t, m, cmd)

				_, completeCmd := m.Update(tea.KeyMsg{Type: tea.KeyTab})
				complete := requireCompleteMsg(t, completeCmd)
				if complete.Project != projectA.Number || complete.View != "" {
					t.Fatalf("SetupCompleteMsg = %#v, want project=%d view=''", complete, projectA.Number)
				}
			},
		},
		{
			name: "step 3 enter selects view",
			run: func(t *testing.T) {
				t.Parallel()

				client := &github.MockClient{
					ListProjectsFn: func(owner string) ([]github.Project, error) {
						return []github.Project{projectA}, nil
					},
					GetProjectViewsFn: func(projectID string) ([]github.ProjectView, error) {
						return []github.ProjectView{viewA, viewB}, nil
					},
				}

				m := New(client)
				m, _ = submitOwner(t, m, "testorg")
				m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
				m, _, _ = updateFromCmd(t, m, cmd)

				_, completeCmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
				complete := requireCompleteMsg(t, completeCmd)
				if complete.Project != projectA.Number || complete.View != viewA.Name {
					t.Fatalf("SetupCompleteMsg = %#v, want project=%d view=%q", complete, projectA.Number, viewA.Name)
				}
			},
		},
		{
			name: "happy path end to end",
			run: func(t *testing.T) {
				t.Parallel()

				client := &github.MockClient{
					ListProjectsFn: func(owner string) ([]github.Project, error) {
						if owner != "testorg" {
							t.Fatalf("owner = %q, want testorg", owner)
						}
						return []github.Project{projectA, projectB}, nil
					},
					GetProjectViewsFn: func(projectID string) ([]github.ProjectView, error) {
						if projectID != projectA.ID {
							t.Fatalf("projectID = %q, want %q", projectID, projectA.ID)
						}
						return []github.ProjectView{viewA, viewB}, nil
					},
				}

				m := New(client)
				m, _ = submitOwner(t, m, "testorg")
				m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
				m, _, _ = updateFromCmd(t, m, cmd)
				_, completeCmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})

				complete := requireCompleteMsg(t, completeCmd)
				if complete.Owner != "testorg" || complete.Project != projectA.Number || complete.View != viewA.Name {
					t.Fatalf("SetupCompleteMsg = %#v, want owner=testorg project=%d view=%q", complete, projectA.Number, viewA.Name)
				}
			},
		},
		{
			name: "cancel on step 2 via esc goes back to owner",
			run: func(t *testing.T) {
				t.Parallel()

				client := &github.MockClient{ListProjectsFn: func(owner string) ([]github.Project, error) {
					return []github.Project{projectA}, nil
				}}
				m := New(client)
				m, _ = submitOwner(t, m, "testorg")
				if m.step != stepProjectSelect {
					t.Fatalf("step = %v, want %v", m.step, stepProjectSelect)
				}

				updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
				if updated.step != stepOwner {
					t.Fatalf("step = %v, want %v", updated.step, stepOwner)
				}
				if cmd != nil {
					t.Fatalf("cmd = %v, want nil", cmd)
				}
			},
		},
		{
			name: "step 3 view api error shows error state",
			run: func(t *testing.T) {
				t.Parallel()

				client := &github.MockClient{
					ListProjectsFn: func(owner string) ([]github.Project, error) {
						return []github.Project{projectA}, nil
					},
					GetProjectViewsFn: func(projectID string) ([]github.ProjectView, error) {
						return nil, errors.New("view api error")
					},
				}

				m := New(client)
				m, _ = submitOwner(t, m, "testorg")
				m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
				m, _, _ = updateFromCmd(t, m, cmd)

				if m.step != stepViewError {
					t.Fatalf("step = %v, want %v", m.step, stepViewError)
				}
				if !strings.Contains(m.View(), "Error loading views:") {
					t.Fatalf("View() = %q, want error text", m.View())
				}
			},
		},
		{
			name: "step 3 tab skips on view error",
			run: func(t *testing.T) {
				t.Parallel()

				client := &github.MockClient{
					ListProjectsFn: func(owner string) ([]github.Project, error) {
						return []github.Project{projectA}, nil
					},
					GetProjectViewsFn: func(projectID string) ([]github.ProjectView, error) {
						return nil, errors.New("view api error")
					},
				}

				m := New(client)
				m, _ = submitOwner(t, m, "testorg")
				m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
				m, _, _ = updateFromCmd(t, m, cmd)

				_, completeCmd := m.Update(tea.KeyMsg{Type: tea.KeyTab})
				complete := requireCompleteMsg(t, completeCmd)
				if complete.Owner != "testorg" || complete.Project != projectA.Number || complete.View != "" {
					t.Fatalf("SetupCompleteMsg = %#v, want owner=testorg project=%d view=''", complete, projectA.Number)
				}
			},
		},
		{
			name: "step 3 enter retries on view error",
			run: func(t *testing.T) {
				t.Parallel()

				calls := 0
				client := &github.MockClient{
					ListProjectsFn: func(owner string) ([]github.Project, error) {
						return []github.Project{projectA}, nil
					},
					GetProjectViewsFn: func(projectID string) ([]github.ProjectView, error) {
						calls++
						if calls == 1 {
							return nil, errors.New("view api error")
						}
						return []github.ProjectView{viewA}, nil
					},
				}

				m := New(client)
				m, _ = submitOwner(t, m, "testorg")
				m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
				m, _, _ = updateFromCmd(t, m, cmd)
				if m.step != stepViewError {
					t.Fatalf("step = %v, want %v", m.step, stepViewError)
				}

				updated, retryCmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
				if updated.step != stepViewLoading {
					t.Fatalf("step = %v, want %v", updated.step, stepViewLoading)
				}
				if retryCmd == nil {
					t.Fatal("cmd = nil, want fetch views retry command")
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, tc.run)
	}
}

func typeOwner(t *testing.T, m Model, owner string) Model {
	t.Helper()

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(owner)})
	return updated
}

func submitOwner(t *testing.T, m Model, owner string) (Model, tea.Cmd) {
	t.Helper()

	m = typeOwner(t, m, owner)
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("owner submission returned nil command")
	}

	updated, nextCmd, _ := updateFromCmd(t, updated, cmd)
	return updated, nextCmd
}

func updateFromCmd(t *testing.T, m Model, cmd tea.Cmd) (Model, tea.Cmd, tea.Msg) {
	t.Helper()

	if cmd == nil {
		t.Fatal("cmd = nil")
	}

	msg := cmd()
	updated, next := m.Update(msg)
	return updated, next, msg
}

func requireCancelMsg(t *testing.T, cmd tea.Cmd) {
	t.Helper()

	if cmd == nil {
		t.Fatal("cmd = nil, want SetupCancelMsg command")
	}
	if _, ok := cmd().(SetupCancelMsg); !ok {
		t.Fatalf("cmd() type = %T, want SetupCancelMsg", cmd())
	}
}

func requireCompleteMsg(t *testing.T, cmd tea.Cmd) SetupCompleteMsg {
	t.Helper()

	if cmd == nil {
		t.Fatal("cmd = nil, want SetupCompleteMsg command")
	}
	msg := cmd()
	complete, ok := msg.(SetupCompleteMsg)
	if !ok {
		t.Fatalf("cmd() type = %T, want SetupCompleteMsg", msg)
	}

	return complete
}
