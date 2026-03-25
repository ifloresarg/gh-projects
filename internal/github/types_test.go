package github

import (
	"encoding/json"
	"testing"
	"time"
)

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

func TestProjectItemJSONRoundTripIssue(t *testing.T) {
	t.Parallel()

	original := ProjectItem{
		ID:            "item-1",
		Title:         "Test Issue",
		Type:          "Issue",
		Status:        "Todo",
		StatusID:      "status-1",
		TypeValue:     "bug",
		TypeID:        "type-1",
		RepoOwner:     "octocat",
		RepoName:      "repo",
		ContentNumber: 42,
		Content: &Issue{
			ID:        "issue-id",
			Number:    42,
			Title:     "Fix bug",
			Body:      "Description",
			State:     "OPEN",
			IssueType: "bug",
			Author:    User{Login: "author", Name: "Author Name"},
			Assignees: []User{{Login: "assignee", Name: "Assignee Name"}},
			Labels:    []Label{{ID: "label-1", Name: "bug", Color: "FF0000"}},
			LinkedPRs: []LinkedPullRequest{{Number: 10, State: "OPEN"}},
			CreatedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			UpdatedAt: time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
			RepoOwner: "octocat",
			RepoName:  "repo",
		},
	}

	// Marshal to JSON
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	// Unmarshal from JSON
	var restored ProjectItem
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	// Verify fields preserved
	if restored.ID != original.ID || restored.Type != original.Type {
		t.Fatalf("basic fields not preserved: got %#v, want %#v", restored, original)
	}

	// Verify Content is *Issue (not map[string]interface{})
	issue, ok := restored.Content.(*Issue)
	if !ok {
		t.Fatalf("Content is %T, want *Issue", restored.Content)
	}

	// Verify Issue fields preserved
	if issue.Number != 42 || issue.Title != "Fix bug" || issue.State != "OPEN" {
		t.Fatalf("issue fields not preserved: got %#v", issue)
	}
	if issue.Author.Login != "author" || len(issue.Assignees) != 1 {
		t.Fatalf("issue nested fields not preserved: got %#v", issue)
	}
	if len(issue.Labels) != 1 || issue.Labels[0].Name != "bug" {
		t.Fatalf("issue labels not preserved: got %#v", issue.Labels)
	}
	if len(issue.LinkedPRs) != 1 || issue.LinkedPRs[0].Number != 10 {
		t.Fatalf("issue linked PRs not preserved: got %#v", issue.LinkedPRs)
	}
}

func TestProjectItemJSONRoundTripPullRequest(t *testing.T) {
	t.Parallel()

	original := ProjectItem{
		ID:            "item-2",
		Title:         "Test PR",
		Type:          "PullRequest",
		Status:        "In Progress",
		StatusID:      "status-2",
		RepoOwner:     "octocat",
		RepoName:      "repo",
		ContentNumber: 55,
		Content: &PullRequest{
			ID:        "pr-id",
			Number:    55,
			Title:     "Add feature",
			Body:      "Feature description",
			State:     "OPEN",
			Author:    User{Login: "pr-author", Name: "PR Author"},
			URL:       "https://github.com/octocat/repo/pull/55",
			CreatedAt: time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC),
			MergedAt:  time.Time{},
			RepoOwner: "octocat",
			RepoName:  "repo",
		},
	}

	// Marshal to JSON
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	// Unmarshal from JSON
	var restored ProjectItem
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	// Verify fields preserved
	if restored.ID != original.ID || restored.Type != original.Type {
		t.Fatalf("basic fields not preserved: got %#v, want %#v", restored, original)
	}

	// Verify Content is *PullRequest (not map[string]interface{})
	pr, ok := restored.Content.(*PullRequest)
	if !ok {
		t.Fatalf("Content is %T, want *PullRequest", restored.Content)
	}

	// Verify PullRequest fields preserved
	if pr.Number != 55 || pr.Title != "Add feature" || pr.State != "OPEN" {
		t.Fatalf("pull request fields not preserved: got %#v", pr)
	}
	if pr.Author.Login != "pr-author" || pr.URL != "https://github.com/octocat/repo/pull/55" {
		t.Fatalf("pull request nested fields not preserved: got %#v", pr)
	}
}

func TestProjectItemJSONRoundTripDraftIssue(t *testing.T) {
	t.Parallel()

	original := ProjectItem{
		ID:      "item-3",
		Title:   "Draft spec",
		Type:    "DraftIssue",
		Status:  "No Status",
		Content: nil,
	}

	// Marshal to JSON
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	// Unmarshal from JSON
	var restored ProjectItem
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	// Verify fields preserved
	if restored.ID != original.ID || restored.Type != original.Type {
		t.Fatalf("basic fields not preserved: got %#v, want %#v", restored, original)
	}

	// Verify Content stays nil
	if restored.Content != nil {
		t.Fatalf("Content should be nil for DraftIssue, got %#v", restored.Content)
	}
}

func TestProjectItemJSONRoundTripREDACTED(t *testing.T) {
	t.Parallel()

	original := ProjectItem{
		ID:     "item-4",
		Title:  "Redacted item",
		Type:   "REDACTED",
		Status: "Done",
		Content: map[string]interface{}{
			"restricted": true,
		},
	}

	// Marshal to JSON
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	// Unmarshal from JSON
	var restored ProjectItem
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	// Verify fields preserved
	if restored.ID != original.ID || restored.Type != original.Type {
		t.Fatalf("basic fields not preserved: got %#v, want %#v", restored, original)
	}

	// Verify Content stays nil (REDACTED type maps to nil regardless of API content)
	if restored.Content != nil {
		t.Fatalf("Content should be nil for REDACTED type, got %#v", restored.Content)
	}
}

func TestProjectItemJSONRoundTripWithNullContent(t *testing.T) {
	t.Parallel()

	// Direct JSON with null content
	jsonStr := `{
		"id": "item-5",
		"title": "Null content item",
		"type": "Issue",
		"status": "Todo",
		"statusId": "",
		"typeValue": "",
		"typeId": "",
		"content": null,
		"repoOwner": "",
		"repoName": "",
		"contentNumber": 0
	}`

	var item ProjectItem
	if err := json.Unmarshal([]byte(jsonStr), &item); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	// Content should be nil even with Issue type if it's explicitly null in JSON
	if item.Content != nil {
		t.Fatalf("Content should be nil when JSON has null, got %#v", item.Content)
	}
}

func TestProjectItemJSONUnmarshalPreservesPointerType(t *testing.T) {
	t.Parallel()

	// Simulate raw JSON from API with Issue content
	jsonStr := `{
		"id": "item-1",
		"title": "Test",
		"type": "Issue",
		"status": "Todo",
		"statusId": "s1",
		"typeValue": "bug",
		"typeId": "t1",
		"content": {
			"id": "issue-123",
			"number": 42,
			"title": "Issue title",
			"body": "Issue body",
			"state": "OPEN",
			"issueType": "bug",
			"author": {"login": "user1", "name": "User One"},
			"assignees": [],
			"labels": [],
			"linkedPRs": [],
			"createdAt": "2024-01-01T00:00:00Z",
			"updatedAt": "2024-01-02T00:00:00Z",
			"repoOwner": "octocat",
			"repoName": "repo"
		},
		"repoOwner": "octocat",
		"repoName": "repo",
		"contentNumber": 42
	}`

	var item ProjectItem
	if err := json.Unmarshal([]byte(jsonStr), &item); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	// Critical test: Content must be *Issue, not map[string]interface{}
	issue, ok := item.Content.(*Issue)
	if !ok {
		t.Fatalf("Content is %T (want *Issue). Standard unmarshaling broke.", item.Content)
	}

	if issue.Number != 42 || issue.Title != "Issue title" {
		t.Fatalf("Issue fields incorrect: got %#v", issue)
	}
}
