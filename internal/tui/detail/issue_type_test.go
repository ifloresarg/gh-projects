package detail

import (
	"errors"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ifloresarg/gh-projects/internal/github"
)

func issueTypeTestIssue(issueType string) *github.Issue {
	return &github.Issue{
		ID:        "I_123",
		RepoOwner: "ifloresarg",
		RepoName:  "gh-projects",
		IssueType: issueType,
	}
}

func issueTypeLoadedModel(client github.GitHubClient, issue *github.Issue) issueTypeModel {
	m := newIssueTypeModel(client, issue, 80, 24)
	m, _ = m.Update(issueTypesLoadedMsg{types: []github.IssueType{
		{ID: "IT_1", Name: "Bug"},
		{ID: "IT_2", Name: "Feature"},
		{ID: "IT_3", Name: "Enhancement"},
	}})
	return m
}

func runIssueTypeCommand(t *testing.T, m issueTypeModel, cmd tea.Cmd) issueTypeModel {
	t.Helper()

	if cmd == nil {
		t.Fatal("cmd = nil, want non-nil")
	}

	msg := cmd()
	updated, _ := m.Update(msg)
	return updated
}

func TestIssueTypeInit(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		issue         *github.Issue
		wantErrSubstr string
		wantListCall  bool
	}{
		{
			name:          "nil issue returns error",
			issue:         nil,
			wantErrSubstr: "issue detail unavailable",
			wantListCall:  false,
		},
		{
			name:         "valid issue calls list issue types",
			issue:        issueTypeTestIssue("Bug"),
			wantListCall: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			listed := false
			client := &github.MockClient{
				ListIssueTypesFn: func(owner, repo string) ([]github.IssueType, error) {
					listed = true
					if owner != "ifloresarg" || repo != "gh-projects" {
						t.Fatalf("ListIssueTypes args = (%q, %q), want (%q, %q)", owner, repo, "ifloresarg", "gh-projects")
					}
					return []github.IssueType{{ID: "IT_1", Name: "Bug"}}, nil
				},
			}

			m := newIssueTypeModel(client, tt.issue, 80, 24)
			cmd := m.Init()
			if cmd == nil {
				t.Fatal("Init() cmd = nil, want non-nil")
			}

			msg := cmd()
			loaded, ok := msg.(issueTypesLoadedMsg)
			if !ok {
				t.Fatalf("Init() message type = %T, want issueTypesLoadedMsg", msg)
			}

			if tt.wantErrSubstr != "" {
				if loaded.err == nil || !strings.Contains(loaded.err.Error(), tt.wantErrSubstr) {
					t.Fatalf("loaded.err = %v, want substring %q", loaded.err, tt.wantErrSubstr)
				}
			} else if loaded.err != nil {
				t.Fatalf("loaded.err = %v, want nil", loaded.err)
			}

			if listed != tt.wantListCall {
				t.Fatalf("ListIssueTypes called = %v, want %v", listed, tt.wantListCall)
			}
		})
	}
}

func TestIssueTypesLoadedMsgHandling(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		msg             issueTypesLoadedMsg
		wantLoading     bool
		wantCount       int
		wantOpMsgSubstr string
	}{
		{
			name:            "load error sets message",
			msg:             issueTypesLoadedMsg{err: errors.New("boom")},
			wantLoading:     false,
			wantCount:       0,
			wantOpMsgSubstr: "Error loading issue types: boom",
		},
		{
			name: "types populate list",
			msg: issueTypesLoadedMsg{types: []github.IssueType{
				{ID: "IT_1", Name: "Bug"},
				{ID: "IT_2", Name: "Feature"},
			}},
			wantLoading: false,
			wantCount:   2,
		},
		{
			name:            "empty list sets no types message",
			msg:             issueTypesLoadedMsg{types: []github.IssueType{}},
			wantLoading:     false,
			wantCount:       0,
			wantOpMsgSubstr: "No issue types found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			m := newIssueTypeModel(&github.MockClient{}, issueTypeTestIssue(""), 80, 24)
			updated, _ := m.Update(tt.msg)

			if updated.loading != tt.wantLoading {
				t.Fatalf("loading = %v, want %v", updated.loading, tt.wantLoading)
			}
			if len(updated.allTypes) != tt.wantCount {
				t.Fatalf("len(allTypes) = %d, want %d", len(updated.allTypes), tt.wantCount)
			}
			if tt.wantOpMsgSubstr != "" && !strings.Contains(updated.opMsg, tt.wantOpMsgSubstr) {
				t.Fatalf("opMsg = %q, want substring %q", updated.opMsg, tt.wantOpMsgSubstr)
			}
		})
	}
}

type issueTypeMutationCapture struct {
	called  bool
	issueID string
	typeID  *string
	err     error
}

func TestIssueTypeUpdateFlows(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		setup func(t *testing.T) (issueTypeModel, *issueTypeMutationCapture)
		msg   tea.Msg
		check func(t *testing.T, before issueTypeModel, after issueTypeModel, cmd tea.Cmd, cap *issueTypeMutationCapture)
	}{
		{
			name: "esc sets closing",
			setup: func(t *testing.T) (issueTypeModel, *issueTypeMutationCapture) {
				t.Helper()
				m := issueTypeLoadedModel(&github.MockClient{}, issueTypeTestIssue("Bug"))
				return m, nil
			},
			msg: tea.KeyMsg{Type: tea.KeyEsc},
			check: func(t *testing.T, _ issueTypeModel, after issueTypeModel, cmd tea.Cmd, _ *issueTypeMutationCapture) {
				t.Helper()
				if !after.closing {
					t.Fatal("closing = false, want true")
				}
				if cmd != nil {
					t.Fatal("cmd != nil, want nil")
				}
			},
		},
		{
			name: "j while loading does not move",
			setup: func(t *testing.T) (issueTypeModel, *issueTypeMutationCapture) {
				t.Helper()
				m := issueTypeLoadedModel(&github.MockClient{}, issueTypeTestIssue("Bug"))
				m.loading = true
				m.cursor = 0
				return m, nil
			},
			msg: tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}},
			check: func(t *testing.T, _ issueTypeModel, after issueTypeModel, cmd tea.Cmd, _ *issueTypeMutationCapture) {
				t.Helper()
				if after.cursor != 0 {
					t.Fatalf("cursor = %d, want 0", after.cursor)
				}
				if cmd != nil {
					t.Fatal("cmd != nil, want nil")
				}
			},
		},
		{
			name: "j at end does not move",
			setup: func(t *testing.T) (issueTypeModel, *issueTypeMutationCapture) {
				t.Helper()
				m := issueTypeLoadedModel(&github.MockClient{}, issueTypeTestIssue("Bug"))
				m.cursor = len(m.allTypes) - 1
				return m, nil
			},
			msg: tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}},
			check: func(t *testing.T, before issueTypeModel, after issueTypeModel, cmd tea.Cmd, _ *issueTypeMutationCapture) {
				t.Helper()
				if after.cursor != before.cursor {
					t.Fatalf("cursor = %d, want %d", after.cursor, before.cursor)
				}
				if cmd != nil {
					t.Fatal("cmd != nil, want nil")
				}
			},
		},
		{
			name: "j moves down",
			setup: func(t *testing.T) (issueTypeModel, *issueTypeMutationCapture) {
				t.Helper()
				m := issueTypeLoadedModel(&github.MockClient{}, issueTypeTestIssue("Bug"))
				m.cursor = 0
				return m, nil
			},
			msg: tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}},
			check: func(t *testing.T, _ issueTypeModel, after issueTypeModel, cmd tea.Cmd, _ *issueTypeMutationCapture) {
				t.Helper()
				if after.cursor != 1 {
					t.Fatalf("cursor = %d, want 1", after.cursor)
				}
				if cmd != nil {
					t.Fatal("cmd != nil, want nil")
				}
			},
		},
		{
			name: "enter while loading is no-op",
			setup: func(t *testing.T) (issueTypeModel, *issueTypeMutationCapture) {
				t.Helper()
				m := issueTypeLoadedModel(&github.MockClient{}, issueTypeTestIssue("Bug"))
				m.loading = true
				return m, nil
			},
			msg: tea.KeyMsg{Type: tea.KeyEnter},
			check: func(t *testing.T, before issueTypeModel, after issueTypeModel, cmd tea.Cmd, _ *issueTypeMutationCapture) {
				t.Helper()
				if cmd != nil {
					t.Fatal("cmd != nil, want nil")
				}
				if after.issue == nil || after.issue.IssueType != before.issue.IssueType {
					t.Fatalf("issue type = %q, want %q", after.issue.IssueType, before.issue.IssueType)
				}
			},
		},
		{
			name: "enter selects new type and calls mutation with type id",
			setup: func(t *testing.T) (issueTypeModel, *issueTypeMutationCapture) {
				t.Helper()
				cap := &issueTypeMutationCapture{}
				client := &github.MockClient{
					UpdateIssueTypeFn: func(issueID string, typeID *string) error {
						cap.called = true
						cap.issueID = issueID
						if typeID != nil {
							id := *typeID
							cap.typeID = &id
						}
						return cap.err
					},
				}
				m := issueTypeLoadedModel(client, issueTypeTestIssue("Bug"))
				m.cursor = 1
				return m, cap
			},
			msg: tea.KeyMsg{Type: tea.KeyEnter},
			check: func(t *testing.T, _ issueTypeModel, after issueTypeModel, cmd tea.Cmd, cap *issueTypeMutationCapture) {
				t.Helper()
				updated := runIssueTypeCommand(t, after, cmd)
				if !cap.called {
					t.Fatal("UpdateIssueType was not called")
				}
				if cap.issueID != "I_123" {
					t.Fatalf("issueID = %q, want I_123", cap.issueID)
				}
				if cap.typeID == nil || *cap.typeID != "IT_2" {
					t.Fatalf("typeID = %v, want pointer to IT_2", cap.typeID)
				}
				if updated.issue == nil || updated.issue.IssueType != "Feature" {
					t.Fatalf("issue type = %q, want Feature", updated.issue.IssueType)
				}
			},
		},
		{
			name: "enter on current type clears with nil type id",
			setup: func(t *testing.T) (issueTypeModel, *issueTypeMutationCapture) {
				t.Helper()
				cap := &issueTypeMutationCapture{}
				client := &github.MockClient{
					UpdateIssueTypeFn: func(issueID string, typeID *string) error {
						cap.called = true
						cap.issueID = issueID
						cap.typeID = typeID
						return cap.err
					},
				}
				m := issueTypeLoadedModel(client, issueTypeTestIssue("Bug"))
				m.cursor = 0
				return m, cap
			},
			msg: tea.KeyMsg{Type: tea.KeyEnter},
			check: func(t *testing.T, _ issueTypeModel, after issueTypeModel, cmd tea.Cmd, cap *issueTypeMutationCapture) {
				t.Helper()
				updated := runIssueTypeCommand(t, after, cmd)
				if !cap.called {
					t.Fatal("UpdateIssueType was not called")
				}
				if cap.typeID != nil {
					t.Fatalf("typeID = %v, want nil", cap.typeID)
				}
				if updated.issue == nil || updated.issue.IssueType != "" {
					t.Fatalf("issue type = %q, want empty", updated.issue.IssueType)
				}
			},
		},
		{
			name: "result success updates issue type",
			setup: func(t *testing.T) (issueTypeModel, *issueTypeMutationCapture) {
				t.Helper()
				m := issueTypeLoadedModel(&github.MockClient{}, issueTypeTestIssue("Bug"))
				return m, nil
			},
			msg: issueTypeResultMsg{typeName: "Enhancement"},
			check: func(t *testing.T, _ issueTypeModel, after issueTypeModel, cmd tea.Cmd, _ *issueTypeMutationCapture) {
				t.Helper()
				if cmd != nil {
					t.Fatal("cmd != nil, want nil")
				}
				if after.issue == nil || after.issue.IssueType != "Enhancement" {
					t.Fatalf("issue type = %q, want Enhancement", after.issue.IssueType)
				}
			},
		},
		{
			name: "result error sets op msg",
			setup: func(t *testing.T) (issueTypeModel, *issueTypeMutationCapture) {
				t.Helper()
				m := issueTypeLoadedModel(&github.MockClient{}, issueTypeTestIssue("Bug"))
				return m, nil
			},
			msg: issueTypeResultMsg{err: errors.New("update failed")},
			check: func(t *testing.T, _ issueTypeModel, after issueTypeModel, cmd tea.Cmd, _ *issueTypeMutationCapture) {
				t.Helper()
				if cmd != nil {
					t.Fatal("cmd != nil, want nil")
				}
				if !strings.Contains(after.opMsg, "Error: update failed") {
					t.Fatalf("opMsg = %q, want update error", after.opMsg)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			before, capture := tt.setup(t)
			after, cmd := before.Update(tt.msg)
			tt.check(t, before, after, cmd, capture)
		})
	}
}

func TestIssueTypeView(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		model    issueTypeModel
		contains []string
	}{
		{
			name: "loading state shows loading text",
			model: issueTypeModel{
				loading: true,
				width:   80,
				height:  24,
			},
			contains: []string{
				"Issue Type — j/k navigate, Enter select/clear, Esc cancel",
				"Loading issue types...",
			},
		},
		{
			name: "current type shows bullet marker",
			model: issueTypeModel{
				issue: issueTypeTestIssue("Feature"),
				allTypes: []github.IssueType{
					{ID: "IT_1", Name: "Bug"},
					{ID: "IT_2", Name: "Feature"},
				},
				cursor:  0,
				loading: false,
				width:   80,
				height:  24,
			},
			contains: []string{"[•] Feature"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			view := tt.model.View()
			for _, fragment := range tt.contains {
				if !strings.Contains(view, fragment) {
					t.Fatalf("View() missing %q in %q", fragment, view)
				}
			}
		})
	}
}
