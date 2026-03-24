package detail

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/ifloresarg/gh-projects/internal/github"
)

func loadPRPicker(t *testing.T, prs []github.PullRequest, window time.Duration) prPickerModel {
	t.Helper()

	m := newPRPicker(&github.MockClient{
		ListRepositoryPullRequestsFn: func(owner, repo string, limit int) ([]github.PullRequest, error) {
			return prs, nil
		},
	}, []RepoRef{{Owner: "octocat", Name: "gh-projects"}}, window, 200)

	cmd := m.Init()
	if cmd == nil {
		t.Fatal("Init() returned nil command")
	}

	msg := cmd()
	updated, _ := m.Update(msg)
	return updated
}

func prListItems(m prPickerModel) []list.Item {
	return m.list.Items()
}

func TestPRPickerTitle(t *testing.T) {
	t.Parallel()

	m := newPRPicker(&github.MockClient{}, nil, 12*time.Hour, 200)
	if m.list.Title != "Link PR to issue" {
		t.Fatalf("list.Title = %q, want %q", m.list.Title, "Link PR to issue")
	}
}

func TestPRPickerMergedWithinWindowIncluded(t *testing.T) {
	t.Parallel()

	now := time.Now()
	m := loadPRPicker(t, []github.PullRequest{{
		Number:    1,
		Title:     "Recently merged",
		State:     "MERGED",
		Author:    github.User{Login: "octocat"},
		CreatedAt: now.Add(-24 * time.Hour),
		MergedAt:  now.Add(-1 * time.Hour),
		RepoOwner: "octocat",
		RepoName:  "gh-projects",
	}}, 12*time.Hour)

	if len(prListItems(m)) != 1 {
		t.Fatalf("list item count = %d, want 1", len(prListItems(m)))
	}
}

func TestPRPickerMergedOutsideWindowExcluded(t *testing.T) {
	t.Parallel()

	now := time.Now()
	m := loadPRPicker(t, []github.PullRequest{{
		Number:    1,
		Title:     "Old merged",
		State:     "MERGED",
		Author:    github.User{Login: "octocat"},
		CreatedAt: now.Add(-48 * time.Hour),
		MergedAt:  now.Add(-25 * time.Hour),
		RepoOwner: "octocat",
		RepoName:  "gh-projects",
	}}, 12*time.Hour)

	if len(prListItems(m)) != 0 {
		t.Fatalf("list item count = %d, want 0", len(prListItems(m)))
	}
}

func TestPRPickerOpenAlwaysIncluded(t *testing.T) {
	t.Parallel()

	now := time.Now()
	m := loadPRPicker(t, []github.PullRequest{{
		Number:    1,
		Title:     "Ancient open PR",
		State:     "OPEN",
		Author:    github.User{Login: "octocat"},
		CreatedAt: now.Add(-30 * 24 * time.Hour),
		RepoOwner: "octocat",
		RepoName:  "gh-projects",
	}}, 12*time.Hour)

	if len(prListItems(m)) != 1 {
		t.Fatalf("list item count = %d, want 1", len(prListItems(m)))
	}
}

func TestPRPickerMergedDescriptionContainsMerged(t *testing.T) {
	t.Parallel()

	description := (prItem{pr: github.PullRequest{
		State:     "MERGED",
		Author:    github.User{Login: "octocat"},
		MergedAt:  time.Now().Add(-1 * time.Hour),
		RepoOwner: "octocat",
		RepoName:  "gh-projects",
	}}).Description()

	if !strings.Contains(description, "merged") {
		t.Fatalf("Description() = %q, want substring %q", description, "merged")
	}
}

func TestPRPickerSortByEffectiveTime(t *testing.T) {
	t.Parallel()

	now := time.Now()
	m := loadPRPicker(t, []github.PullRequest{
		{
			Number:    1,
			Title:     "Older open PR",
			State:     "OPEN",
			Author:    github.User{Login: "octocat"},
			CreatedAt: now.Add(-2 * time.Hour),
			RepoOwner: "octocat",
			RepoName:  "gh-projects",
		},
		{
			Number:    2,
			Title:     "Recently merged PR",
			State:     "MERGED",
			Author:    github.User{Login: "octocat"},
			CreatedAt: now.Add(-30 * 24 * time.Hour),
			MergedAt:  now.Add(-1 * time.Hour),
			RepoOwner: "octocat",
			RepoName:  "gh-projects",
		},
	}, 12*time.Hour)

	items := prListItems(m)
	if len(items) != 2 {
		t.Fatalf("list item count = %d, want 2", len(items))
	}

	first, ok := items[0].(prItem)
	if !ok {
		t.Fatalf("first item type = %T, want prItem", items[0])
	}
	if first.pr.Number != 2 {
		t.Fatalf("first PR number = %d, want 2", first.pr.Number)
	}
}
