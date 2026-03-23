package detail

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ifloresarg/gh-projects/internal/github"
)

type testError string

func (e testError) Error() string {
	return string(e)
}

func buildLoadedAssignModel(client *github.MockClient, issue *github.Issue) assignModel {
	m := newAssignModel(client, issue, 80, 24)
	m, _ = m.Update(collaboratorsLoadedMsg{users: []github.User{
		{Login: "octocat", Name: "The Octocat"},
		{Login: "monalisa", Name: "Mona Lisa"},
		{Login: "hubot", Name: ""},
	}})
	return m
}

func testAssignIssue(assignees ...github.User) *github.Issue {
	return &github.Issue{
		RepoOwner: "org",
		RepoName:  "repo",
		Number:    1,
		Assignees: append([]github.User(nil), assignees...),
	}
}

func runAssignCmd(t *testing.T, m assignModel, cmd tea.Cmd) assignModel {
	t.Helper()

	if cmd == nil {
		t.Fatal("cmd = nil, want non-nil")
	}

	msg := cmd()
	updated, _ := m.Update(msg)
	return updated
}

func TestAssignLoadingBlocksInteraction(t *testing.T) {
	t.Parallel()

	m := newAssignModel(&github.MockClient{}, testAssignIssue(), 80, 24)
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})

	if m2.cursor != 0 {
		t.Errorf("cursor = %d, want 0 while loading", m2.cursor)
	}
}

func TestAssignCollaboratorsLoaded(t *testing.T) {
	t.Parallel()

	m := newAssignModel(&github.MockClient{}, testAssignIssue(), 80, 24)
	m2, _ := m.Update(collaboratorsLoadedMsg{users: []github.User{{Login: "octocat"}}})

	if len(m2.allUsers) != 1 {
		t.Fatalf("len(allUsers) = %d, want 1", len(m2.allUsers))
	}
	if m2.loading {
		t.Fatal("loading = true, want false")
	}
	if len(m2.filtered) != 1 {
		t.Fatalf("len(filtered) = %d, want 1", len(m2.filtered))
	}
}

func TestAssignLoadError(t *testing.T) {
	t.Parallel()

	m := newAssignModel(&github.MockClient{}, testAssignIssue(), 80, 24)
	m2, _ := m.Update(collaboratorsLoadedMsg{err: testError("api error")})

	if !strings.Contains(m2.opMsg, "api error") {
		t.Fatalf("opMsg = %q, want substring %q", m2.opMsg, "api error")
	}
	if m2.loading {
		t.Fatal("loading = true, want false after error")
	}
}

func TestAssignFilterByLogin(t *testing.T) {
	t.Parallel()

	m := buildLoadedAssignModel(&github.MockClient{}, testAssignIssue())
	m.input.SetValue("oct")
	m.filtered = filterUsers(m.allUsers, m.input.Value())

	if len(m.filtered) != 1 || m.filtered[0].Login != "octocat" {
		t.Fatalf("filtered = %#v, want only octocat", m.filtered)
	}
}

func TestAssignFilterByName(t *testing.T) {
	t.Parallel()

	m := buildLoadedAssignModel(&github.MockClient{}, testAssignIssue())
	m.input.SetValue("lisa")
	m.filtered = filterUsers(m.allUsers, m.input.Value())

	if len(m.filtered) != 1 || m.filtered[0].Login != "monalisa" {
		t.Fatalf("filtered = %#v, want only monalisa", m.filtered)
	}
}

func TestAssignFilterCaseInsensitive(t *testing.T) {
	t.Parallel()

	m := buildLoadedAssignModel(&github.MockClient{}, testAssignIssue())
	m.input.SetValue("OCT")
	m.filtered = filterUsers(m.allUsers, m.input.Value())

	if len(m.filtered) != 1 || m.filtered[0].Login != "octocat" {
		t.Fatalf("filtered = %#v, want only octocat", m.filtered)
	}
}

func TestAssignCursorResetsOnFilterChange(t *testing.T) {
	t.Parallel()

	m := buildLoadedAssignModel(&github.MockClient{}, testAssignIssue())
	m.cursor = 2

	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}})

	if m2.cursor != 0 {
		t.Fatalf("cursor = %d, want 0", m2.cursor)
	}
}

func TestAssignArrowDownNavigatesWithText(t *testing.T) {
	t.Parallel()

	m := buildLoadedAssignModel(&github.MockClient{}, testAssignIssue())
	m.input.SetValue("o")
	m.filtered = filterUsers(m.allUsers, "o")

	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})

	if m2.cursor != 1 {
		t.Fatalf("cursor = %d, want 1", m2.cursor)
	}
}

func TestAssignJKeyNoNavWithText(t *testing.T) {
	t.Parallel()

	m := buildLoadedAssignModel(&github.MockClient{}, testAssignIssue())
	m.cursor = 1
	m.input.SetValue("o")
	m.filtered = filterUsers(m.allUsers, "o")

	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})

	if m2.cursor != 0 {
		t.Fatalf("cursor = %d, want 0 after input update", m2.cursor)
	}
	if m2.input.Value() != "oj" {
		t.Fatalf("input value = %q, want %q", m2.input.Value(), "oj")
	}
}

func TestAssignArrowUpNavigatesWithText(t *testing.T) {
	t.Parallel()

	m := buildLoadedAssignModel(&github.MockClient{}, testAssignIssue())
	m.cursor = 2
	m.input.SetValue("o")
	m.filtered = filterUsers(m.allUsers, "o")

	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyUp})

	if m2.cursor != 1 {
		t.Fatalf("cursor = %d, want 1", m2.cursor)
	}
}

func TestAssignKKeyNoNavWithText(t *testing.T) {
	t.Parallel()

	m := buildLoadedAssignModel(&github.MockClient{}, testAssignIssue())
	m.cursor = 2
	m.input.SetValue("o")
	m.filtered = filterUsers(m.allUsers, "o")

	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})

	if m2.cursor != 0 {
		t.Fatalf("cursor = %d, want 0 after input update", m2.cursor)
	}
	if m2.input.Value() != "ok" {
		t.Fatalf("input value = %q, want %q", m2.input.Value(), "ok")
	}
}

func TestAssignEnterAssignsUser(t *testing.T) {
	t.Parallel()

	assignCalled := false
	assignedLogin := ""
	client := &github.MockClient{
		AssignUserFn: func(owner, repo string, number int, login string) error {
			assignCalled = true
			assignedLogin = login
			if owner != "org" || repo != "repo" || number != 1 {
				t.Fatalf("AssignUser args = (%q, %q, %d), want (%q, %q, %d)", owner, repo, number, "org", "repo", 1)
			}
			return nil
		},
	}
	issue := testAssignIssue()
	m := buildLoadedAssignModel(client, issue)

	m2, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m2 = runAssignCmd(t, m2, cmd)

	if !assignCalled || assignedLogin != "octocat" {
		t.Fatalf("AssignUser called=%v login=%q, want true and octocat", assignCalled, assignedLogin)
	}
	if !hasAssignee(issue.Assignees, "octocat") {
		t.Fatal("issue assignees missing octocat after assign")
	}
	if !strings.Contains(m2.opMsg, "Assigned @octocat") {
		t.Fatalf("opMsg = %q, want assigned message", m2.opMsg)
	}
}

func TestAssignEnterUnassignsUser(t *testing.T) {
	t.Parallel()

	unassignCalled := false
	client := &github.MockClient{
		UnassignUserFn: func(owner, repo string, number int, login string) error {
			unassignCalled = true
			if owner != "org" || repo != "repo" || number != 1 || login != "octocat" {
				t.Fatalf("UnassignUser args = (%q, %q, %d, %q)", owner, repo, number, login)
			}
			return nil
		},
	}
	issue := testAssignIssue(github.User{Login: "octocat", Name: "The Octocat"})
	m := buildLoadedAssignModel(client, issue)

	m2, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m2 = runAssignCmd(t, m2, cmd)

	if !unassignCalled {
		t.Fatal("UnassignUser was not called")
	}
	if hasAssignee(issue.Assignees, "octocat") {
		t.Fatal("issue assignees still contain octocat after unassign")
	}
	if !strings.Contains(m2.opMsg, "Unassigned @octocat") {
		t.Fatalf("opMsg = %q, want unassigned message", m2.opMsg)
	}
}

func TestAssignSpaceToggles(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		issue         *github.Issue
		wantAssign    bool
		wantUnassign  bool
		wantAssignee  bool
		wantMsgPrefix string
	}{
		{
			name:          "assigns unassigned user",
			issue:         testAssignIssue(),
			wantAssign:    true,
			wantAssignee:  true,
			wantMsgPrefix: "Assigned @octocat",
		},
		{
			name:          "unassigns assigned user",
			issue:         testAssignIssue(github.User{Login: "octocat", Name: "The Octocat"}),
			wantUnassign:  true,
			wantAssignee:  false,
			wantMsgPrefix: "Unassigned @octocat",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assignCalled := false
			unassignCalled := false
			client := &github.MockClient{
				AssignUserFn: func(owner, repo string, number int, login string) error {
					assignCalled = true
					return nil
				},
				UnassignUserFn: func(owner, repo string, number int, login string) error {
					unassignCalled = true
					return nil
				},
			}

			m := buildLoadedAssignModel(client, tt.issue)
			m2, cmd := m.Update(tea.KeyMsg{Type: tea.KeySpace})
			m2 = runAssignCmd(t, m2, cmd)

			if assignCalled != tt.wantAssign {
				t.Fatalf("assignCalled = %v, want %v", assignCalled, tt.wantAssign)
			}
			if unassignCalled != tt.wantUnassign {
				t.Fatalf("unassignCalled = %v, want %v", unassignCalled, tt.wantUnassign)
			}
			if hasAssignee(tt.issue.Assignees, "octocat") != tt.wantAssignee {
				t.Fatalf("hasAssignee(octocat) = %v, want %v", hasAssignee(tt.issue.Assignees, "octocat"), tt.wantAssignee)
			}
			if !strings.Contains(m2.opMsg, tt.wantMsgPrefix) {
				t.Fatalf("opMsg = %q, want substring %q", m2.opMsg, tt.wantMsgPrefix)
			}
		})
	}
}

func TestAssignFallbackOnEnter(t *testing.T) {
	t.Parallel()

	assignCalled := false
	assignedLogin := ""
	client := &github.MockClient{
		AssignUserFn: func(owner, repo string, number int, login string) error {
			assignCalled = true
			assignedLogin = login
			return nil
		},
	}
	issue := testAssignIssue()
	m := buildLoadedAssignModel(client, issue)
	m.input.SetValue("external-user")
	m.filtered = filterUsers(m.allUsers, "external-user")

	m2, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m2 = runAssignCmd(t, m2, cmd)

	if !assignCalled || assignedLogin != "external-user" {
		t.Fatalf("AssignUser called=%v login=%q, want true and external-user", assignCalled, assignedLogin)
	}
	if !hasAssignee(issue.Assignees, "external-user") {
		t.Fatal("issue assignees missing external-user after fallback assign")
	}
	if !strings.Contains(m2.opMsg, "Assigned @external-user") {
		t.Fatalf("opMsg = %q, want assigned external-user message", m2.opMsg)
	}
}

func TestAssignNoFallbackOnSpace(t *testing.T) {
	t.Parallel()

	assignCalled := false
	client := &github.MockClient{
		AssignUserFn: func(owner, repo string, number int, login string) error {
			assignCalled = true
			return nil
		},
	}
	m := buildLoadedAssignModel(client, testAssignIssue())
	m.input.SetValue("external-user")
	m.filtered = filterUsers(m.allUsers, "external-user")

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeySpace})

	if cmd != nil {
		t.Fatal("cmd != nil, want nil for space fallback")
	}
	if assignCalled {
		t.Fatal("AssignUser was called, want no fallback on space")
	}
}

func TestAssignEscSetClosing(t *testing.T) {
	t.Parallel()

	m := buildLoadedAssignModel(&github.MockClient{}, testAssignIssue())
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})

	if !m2.closing {
		t.Fatal("closing = false, want true")
	}
}

func TestAssignEmptyNameRendersLoginOnly(t *testing.T) {
	t.Parallel()

	m := buildLoadedAssignModel(&github.MockClient{}, testAssignIssue())
	view := m.View()

	if strings.Contains(view, "hubot — ") {
		t.Fatalf("View() = %q, want no empty-name separator for hubot", view)
	}
	if !strings.Contains(view, "@hubot") {
		t.Fatalf("View() = %q, want @hubot", view)
	}
}
