package detail

import (
	"errors"
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
