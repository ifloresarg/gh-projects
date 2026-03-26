package detail

import (
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ifloresarg/gh-projects/internal/github"
)

func testIssueItem() github.ProjectItem {
	issue := &github.Issue{
		ID:        "I_1",
		Number:    101,
		Title:     "Implement GraphQL client",
		Body:      "Adds markdown body support.",
		State:     "OPEN",
		Author:    github.User{Login: "octocat"},
		Assignees: []github.User{{Login: "octocat"}},
		Labels:    []github.Label{{ID: "bug", Name: "bug", Color: "d73a4a"}},
		CreatedAt: time.Date(2026, time.March, 20, 0, 0, 0, 0, time.UTC),
		RepoOwner: "octocat",
		RepoName:  "gh-projects",
	}

	return github.ProjectItem{
		ID:            "item-1",
		Title:         issue.Title,
		Type:          "Issue",
		Content:       issue,
		RepoOwner:     issue.RepoOwner,
		RepoName:      issue.RepoName,
		ContentNumber: issue.Number,
	}
}

func TestDetailViewRendersIssueContent(t *testing.T) {
	t.Parallel()

	m := New(&github.MockClient{}, testIssueItem(), "PVT_1", nil, 0, 0)
	m, _ = m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	m, _ = m.Update(issueLoadedMsg{issue: testIssueItem().Content.(*github.Issue)})
	m, _ = m.Update(linkedPRsLoadedMsg{prs: []github.PullRequest{{Number: 56, Title: "Fix keyboard navigation", State: "OPEN", Author: github.User{Login: "octocat"}}}})

	view := m.View()
	for _, fragment := range []string{"Issue #101 [OPEN]", "Implement GraphQL client", "Assignees: octocat", "Labels: ● bug", "Linked Pull Requests", "#56"} {
		if !strings.Contains(view, fragment) {
			t.Fatalf("View() missing %q in %q", fragment, view)
		}
	}
}

func TestDetailViewForPullRequestShowsReadonlyContent(t *testing.T) {
	t.Parallel()

	item := github.ProjectItem{
		ID:    "pr-item",
		Title: "Refine board rendering",
		Type:  "PullRequest",
		Content: &github.PullRequest{
			Number: 55,
			Title:  "Refine board rendering",
			State:  "MERGED",
			Author: github.User{Login: "hubot"},
			URL:    "https://github.com/octocat/gh-projects/pull/55",
		},
	}

	m := New(&github.MockClient{}, item, "PVT_1", nil, 0, 0)
	view := m.View()
	for _, fragment := range []string{"PR #55 [MERGED]", "by hubot", "use GitHub web"} {
		if !strings.Contains(view, fragment) {
			t.Fatalf("View() missing %q in %q", fragment, view)
		}
	}
}

func TestDetailInitReturnsUnavailableIssueError(t *testing.T) {
	t.Parallel()

	m := New(&github.MockClient{}, github.ProjectItem{ID: "item-1", Type: "Issue", Title: "Broken item"}, "PVT_1", nil, 0, 0)
	cmd := m.Init()
	msg := cmd()
	loaded, ok := msg.(issueLoadedMsg)
	if !ok {
		t.Fatalf("Init() message type = %T, want issueLoadedMsg", msg)
	}
	if loaded.err == nil || !strings.Contains(loaded.err.Error(), "issue content unavailable") {
		t.Fatalf("issueLoadedMsg.err = %v, want unavailable issue error", loaded.err)
	}
}

func TestParsePRRefTableDriven(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		input      string
		wantOwner  string
		wantRepo   string
		wantNumber int
		wantErr    bool
	}{
		{name: "owner repo ref", input: "octocat/gh-projects#12", wantOwner: "octocat", wantRepo: "gh-projects", wantNumber: 12},
		{name: "github url", input: "https://github.com/octocat/gh-projects/pull/34", wantOwner: "octocat", wantRepo: "gh-projects", wantNumber: 34},
		{name: "invalid ref", input: "octocat/gh-projects", wantErr: true},
		{name: "invalid number", input: "octocat/gh-projects#abc", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			owner, repo, number, err := parsePRRef(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("parsePRRef(%q) error = nil, want non-nil", tt.input)
				}
				return
			}

			if err != nil {
				t.Fatalf("parsePRRef(%q) error = %v", tt.input, err)
			}
			if owner != tt.wantOwner || repo != tt.wantRepo || number != tt.wantNumber {
				t.Fatalf("parsePRRef(%q) = (%q, %q, %d), want (%q, %q, %d)", tt.input, owner, repo, number, tt.wantOwner, tt.wantRepo, tt.wantNumber)
			}
		})
	}
}

func TestAddPRToProjectCmdReturnsLookupError(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("network error")
	cmd := addPRToProjectCmd(&github.MockClient{
		GetPullRequestNodeIDFn: func(owner, repo string, number int) (string, error) {
			return "", wantErr
		},
	}, "PVT_1", "octocat", "gh-projects", 77)

	msg := cmd()
	result, ok := msg.(addPRResultMsg)
	if !ok {
		t.Fatalf("addPRToProjectCmd() message type = %T, want addPRResultMsg", msg)
	}
	if result.err == nil || !strings.Contains(result.err.Error(), "PR not found") {
		t.Fatalf("addPRResultMsg.err = %v, want PR not found error", result.err)
	}
}

func TestDetailPKeyOpensPRPicker(t *testing.T) {
	t.Parallel()

	m := New(&github.MockClient{}, testIssueItem(), "PVT_1", nil, 0, 0)
	m.loading = false
	m.issue = testIssueItem().Content.(*github.Issue)
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})

	if m.showPRPicker != true {
		t.Fatalf("showPRPicker = %v, want true", m.showPRPicker)
	}
	if m.showAddPR != false {
		t.Fatalf("showAddPR = %v, want false", m.showAddPR)
	}
}

func TestDetailPRSelectedMsgTriggersAddPR(t *testing.T) {
	t.Parallel()

	m := New(&github.MockClient{
		GetPullRequestNodeIDFn: func(_, _ string, _ int) (string, error) {
			return "PR_node_1", nil
		},
		AddItemToProjectFn: func(_, _ string) error {
			return nil
		},
	}, testIssueItem(), "PVT_1", nil, 0, 0)
	m.showPRPicker = true
	m, cmd := m.Update(prSelectedMsg{pr: github.PullRequest{
		Number:    42,
		Title:     "Fix bug",
		RepoOwner: "octocat",
		RepoName:  "gh-projects",
	}})

	if cmd == nil {
		t.Fatalf("returned cmd = nil, want non-nil")
	}
	if m.showPRPicker != false {
		t.Fatalf("showPRPicker = %v, want false", m.showPRPicker)
	}
}

func TestDetailPRPickerSwitchToManualToggles(t *testing.T) {
	t.Parallel()

	m := New(&github.MockClient{}, testIssueItem(), "PVT_1", nil, 0, 0)
	m.showPRPicker = true
	m, _ = m.Update(prPickerSwitchToManualMsg{})

	if m.showPRPicker != false {
		t.Fatalf("showPRPicker = %v, want false", m.showPRPicker)
	}
	if m.showAddPR != true {
		t.Fatalf("showAddPR = %v, want true", m.showAddPR)
	}
}

func TestDetailEscClosesPRPicker(t *testing.T) {
	t.Parallel()

	m := New(&github.MockClient{}, testIssueItem(), "PVT_1", nil, 0, 0)
	m.showPRPicker = true
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})

	if m.showPRPicker != false {
		t.Fatalf("showPRPicker = %v, want false", m.showPRPicker)
	}
}

func TestEditBodyKeyGuardsAndLaunch(t *testing.T) {
	originalVisual, hadVisual := os.LookupEnv("VISUAL")
	originalEditor, hadEditor := os.LookupEnv("EDITOR")
	_ = os.Setenv("VISUAL", "vi")
	_ = os.Unsetenv("EDITOR")
	t.Cleanup(func() {
		if hadVisual {
			_ = os.Setenv("VISUAL", originalVisual)
		} else {
			_ = os.Unsetenv("VISUAL")
		}
		if hadEditor {
			_ = os.Setenv("EDITOR", originalEditor)
		} else {
			_ = os.Unsetenv("EDITOR")
		}
	})

	tests := []struct {
		name       string
		model      func() Model
		wantHint   string
		wantEdit   bool
		wantCmdNil bool
	}{
		{
			name: "issue starts editor flow",
			model: func() Model {
				m := New(&github.MockClient{}, testIssueItem(), "PVT_1", nil, 0, 0)
				m.loading = false
				m.issue = testIssueItem().Content.(*github.Issue)
				return m
			},
			wantEdit:   true,
			wantCmdNil: false,
		},
		{
			name: "pull request shows hint",
			model: func() Model {
				m := New(&github.MockClient{}, github.ProjectItem{ID: "pr", Type: "PullRequest"}, "PVT_1", nil, 0, 0)
				return m
			},
			wantHint:   "PR detail: use GitHub web",
			wantEdit:   false,
			wantCmdNil: true,
		},
		{
			name: "draft issue shows hint",
			model: func() Model {
				m := New(&github.MockClient{}, github.ProjectItem{ID: "d", Type: "DraftIssue"}, "PVT_1", nil, 0, 0)
				return m
			},
			wantHint:   "Draft items are read-only",
			wantEdit:   false,
			wantCmdNil: true,
		},
		{
			name: "loading issue shows hint",
			model: func() Model {
				m := New(&github.MockClient{}, testIssueItem(), "PVT_1", nil, 0, 0)
				m.loading = true
				m.issue = testIssueItem().Content.(*github.Issue)
				return m
			},
			wantHint:   "Issue still loading",
			wantEdit:   false,
			wantCmdNil: true,
		},
		{
			name: "nil issue shows hint",
			model: func() Model {
				m := New(&github.MockClient{}, testIssueItem(), "PVT_1", nil, 0, 0)
				m.loading = false
				m.issue = nil
				return m
			},
			wantHint:   "Issue detail unavailable",
			wantEdit:   false,
			wantCmdNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := tt.model()
			updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})

			if updated.opHint != tt.wantHint {
				t.Fatalf("opHint = %q, want %q", updated.opHint, tt.wantHint)
			}
			if updated.editingBody != tt.wantEdit {
				t.Fatalf("editingBody = %v, want %v", updated.editingBody, tt.wantEdit)
			}
			if (cmd == nil) != tt.wantCmdNil {
				t.Fatalf("cmd == nil = %v, want %v", cmd == nil, tt.wantCmdNil)
			}
		})
	}
}

func TestEditBodyKeyIgnoredWhenSubviewActive(t *testing.T) {
	t.Parallel()

	m := New(&github.MockClient{}, testIssueItem(), "PVT_1", nil, 0, 0)
	m.loading = false
	m.issue = testIssueItem().Content.(*github.Issue)
	m.assign = newAssignModel(&github.MockClient{}, m.issue, 80, 24)
	m.showAssign = true

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	if updated.opHint != "" {
		t.Fatalf("opHint = %q, want empty", updated.opHint)
	}
	if updated.editingBody {
		t.Fatal("editingBody = true, want false")
	}
}

func TestEditBodyProcessResult(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		execErr         error
		fileContent     string
		removeFile      bool
		updateErr       error
		wantChanged     bool
		wantCleanup     bool
		wantErrContains string
		wantUpdateCalls int
	}{
		{name: "exec error", execErr: errors.New("editor exited with code 1"), wantCleanup: true, wantErrContains: "editor exited", wantUpdateCalls: 0},
		{name: "read error", removeFile: true, wantCleanup: true, wantErrContains: "no such file", wantUpdateCalls: 0},
		{name: "no changes", fileContent: "Adds markdown body support.", wantChanged: false, wantCleanup: true, wantUpdateCalls: 0},
		{name: "mutation error", fileContent: "new body", updateErr: errors.New("mutation failed"), wantChanged: false, wantCleanup: false, wantErrContains: "mutation failed", wantUpdateCalls: 1},
		{name: "success", fileContent: "updated body", wantChanged: true, wantCleanup: true, wantUpdateCalls: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			updateCalls := 0
			m := New(&github.MockClient{UpdateIssueBodyFn: func(issueID string, body string) error {
				updateCalls++
				if issueID != "I_1" {
					t.Fatalf("issueID = %q, want I_1", issueID)
				}
				return tt.updateErr
			}}, testIssueItem(), "PVT_1", nil, 0, 0)
			m.loading = false
			m.issue = testIssueItem().Content.(*github.Issue)

			tmpFile, err := os.CreateTemp("", "detail-edit-*.md")
			if err != nil {
				t.Fatalf("CreateTemp() error = %v", err)
			}
			tmpPath := tmpFile.Name()
			if tt.fileContent != "" {
				if _, err := tmpFile.WriteString(tt.fileContent); err != nil {
					t.Fatalf("WriteString() error = %v", err)
				}
			}
			if err := tmpFile.Close(); err != nil {
				t.Fatalf("Close() error = %v", err)
			}
			if tt.removeFile {
				if err := os.Remove(tmpPath); err != nil {
					t.Fatalf("Remove() error = %v", err)
				}
			}

			cleanupCalled := false
			msg := m.editBodyExecResult(tt.execErr, tmpPath, m.issue.Body, func() {
				cleanupCalled = true
				_ = os.Remove(tmpPath)
			})

			result, ok := msg.(editBodyResultMsg)
			if !ok {
				t.Fatalf("message type = %T, want editBodyResultMsg", msg)
			}
			if cleanupCalled != tt.wantCleanup {
				t.Fatalf("cleanupCalled = %v, want %v", cleanupCalled, tt.wantCleanup)
			}
			if result.changed != tt.wantChanged {
				t.Fatalf("changed = %v, want %v", result.changed, tt.wantChanged)
			}
			if tt.wantErrContains == "" {
				if result.err != nil {
					t.Fatalf("err = %v, want nil", result.err)
				}
			} else {
				if result.err == nil || !strings.Contains(result.err.Error(), tt.wantErrContains) {
					t.Fatalf("err = %v, want substring %q", result.err, tt.wantErrContains)
				}
				if tt.name == "mutation error" && !strings.Contains(result.err.Error(), tmpPath) {
					t.Fatalf("err = %v, want to include tmp path %q", result.err, tmpPath)
				}
			}
			if updateCalls != tt.wantUpdateCalls {
				t.Fatalf("UpdateIssueBody calls = %d, want %d", updateCalls, tt.wantUpdateCalls)
			}
		})
	}
}

func TestEditBodyResultMsgHandling(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		msg             editBodyResultMsg
		wantBody        string
		wantMsgContains string
		wantMsgExact    string
	}{
		{
			name:            "success updates issue body and viewport",
			msg:             editBodyResultMsg{body: "New markdown body", changed: true},
			wantBody:        "New markdown body",
			wantMsgContains: "updated",
		},
		{
			name:         "no-op edit message",
			msg:          editBodyResultMsg{body: "Adds markdown body support.", changed: false},
			wantBody:     "Adds markdown body support.",
			wantMsgExact: "No changes made",
		},
		{
			name:            "error message",
			msg:             editBodyResultMsg{err: errors.New("editor failed")},
			wantBody:        "Adds markdown body support.",
			wantMsgContains: "Error: editor failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			m := New(&github.MockClient{}, testIssueItem(), "PVT_1", nil, 0, 0)
			m.loading = false
			m.issue = testIssueItem().Content.(*github.Issue)
			m.viewport.Width = 80
			m.viewport.Height = 20
			m.viewport.SetContent(m.renderIssueContent())
			m.editingBody = true
			beforeRendered := m.renderIssueContent()

			updated, _ := m.Update(tt.msg)
			if updated.editingBody {
				t.Fatal("editingBody = true, want false")
			}
			if updated.issue == nil || updated.issue.Body != tt.wantBody {
				t.Fatalf("issue.Body = %q, want %q", updated.issue.Body, tt.wantBody)
			}
			contentIssue, _ := updated.item.Content.(*github.Issue)
			if contentIssue == nil || contentIssue.Body != tt.wantBody {
				t.Fatalf("item.Content body = %q, want %q", contentIssue.Body, tt.wantBody)
			}

			if tt.wantMsgExact != "" {
				if updated.opMsg != tt.wantMsgExact {
					t.Fatalf("opMsg = %q, want %q", updated.opMsg, tt.wantMsgExact)
				}
			} else if !strings.Contains(updated.opMsg, tt.wantMsgContains) {
				t.Fatalf("opMsg = %q, want substring %q", updated.opMsg, tt.wantMsgContains)
			}

			if tt.msg.changed && tt.msg.err == nil {
				afterRendered := updated.renderIssueContent()
				if afterRendered == beforeRendered {
					t.Fatalf("rendered issue content did not change after successful edit")
				}
			}
		})
	}
}
