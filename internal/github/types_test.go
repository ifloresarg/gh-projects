package github

import "testing"

func TestProjectItemIssueContentAccess(t *testing.T) {
	t.Parallel()

	item := ProjectItem{
		Title:   "Fallback",
		Status:  "Todo",
		Content: &Issue{Number: 42, Title: "Actual issue", RepoOwner: "octocat", RepoName: "gh-projects"},
	}

	issue, ok := item.Content.(*Issue)
	if !ok {
		t.Fatal("ProjectItem.Content type assertion to *Issue failed")
	}
	if issue.Number != 42 || issue.Title != "Actual issue" {
		t.Fatalf("unexpected issue content = %#v", issue)
	}
	if item.RepoOwner != "" || item.RepoName != "" {
		t.Fatalf("struct field expectations changed unexpectedly: %#v", item)
	}
}

func TestProjectItemPullRequestContentAccess(t *testing.T) {
	t.Parallel()

	item := ProjectItem{
		ContentNumber: 55,
		Content: &PullRequest{
			Number:    55,
			Title:     "Refine board rendering",
			State:     "MERGED",
			RepoOwner: "octocat",
			RepoName:  "gh-projects",
		},
	}

	pr, ok := item.Content.(*PullRequest)
	if !ok {
		t.Fatal("ProjectItem.Content type assertion to *PullRequest failed")
	}
	if pr.Number != item.ContentNumber {
		t.Fatalf("pull request number = %d, want %d", pr.Number, item.ContentNumber)
	}
	if pr.RepoOwner != "octocat" || pr.RepoName != "gh-projects" {
		t.Fatalf("unexpected pull request repo fields = %#v", pr)
	}
}

func TestProjectItemAllowsNilContentForDraftItems(t *testing.T) {
	t.Parallel()

	item := ProjectItem{
		ID:       "item-3",
		Title:    "Draft spec",
		Type:     "DraftIssue",
		Status:   "No Status",
		StatusID: "",
		Content:  nil,
	}

	if item.Content != nil {
		t.Fatalf("draft item content = %#v, want nil", item.Content)
	}
	if item.Title != "Draft spec" || item.Type != "DraftIssue" {
		t.Fatalf("unexpected draft item fields = %#v", item)
	}
}
