package board

import (
	"testing"

	github "github.com/ifloresarg/gh-projects/internal/github"
)

// ParseStatusFilter tests

func TestParseStatusFilter_Empty(t *testing.T) {
	result := ParseStatusFilter("")
	if len(result) > 0 {
		t.Errorf("ParseStatusFilter(\"\") = %v, want nil or empty", result)
	}
}

func TestParseStatusFilter_NoStatusClause(t *testing.T) {
	result := ParseStatusFilter("-is:closed")
	if len(result) > 0 {
		t.Errorf("ParseStatusFilter(\"-is:closed\") = %v, want nil or empty", result)
	}
}

func TestParseStatusFilter_SingleUnquoted(t *testing.T) {
	result := ParseStatusFilter("-status:Backlog")
	expected := []string{"Backlog"}
	if !sliceEqual(result, expected) {
		t.Errorf("ParseStatusFilter(\"-status:Backlog\") = %v, want %v", result, expected)
	}
}

func TestParseStatusFilter_MultipleUnquoted(t *testing.T) {
	result := ParseStatusFilter("-status:Backlog,Marketing")
	expected := []string{"Backlog", "Marketing"}
	if !sliceEqual(result, expected) {
		t.Errorf("ParseStatusFilter(\"-status:Backlog,Marketing\") = %v, want %v", result, expected)
	}
}

func TestParseStatusFilter_QuotedValues(t *testing.T) {
	result := ParseStatusFilter(`-status:Backlog,"To Design","W4 Design Approval"`)
	expected := []string{"Backlog", "To Design", "W4 Design Approval"}
	if !sliceEqual(result, expected) {
		t.Errorf("ParseStatusFilter with quoted values = %v, want %v", result, expected)
	}
}

func TestParseStatusFilter_MixedWithOtherTokens(t *testing.T) {
	result := ParseStatusFilter(`-status:Backlog,"To Design" -is:closed`)
	expected := []string{"Backlog", "To Design"}
	if !sliceEqual(result, expected) {
		t.Errorf("ParseStatusFilter with mixed tokens = %v, want %v", result, expected)
	}
}

func TestParseStatusFilter_RealDevView(t *testing.T) {
	filter := `-status:Backlog,"To Design","Designing Process","W4 Design Approval","Approved Designs",Marketing -is:closed`
	result := ParseStatusFilter(filter)
	expected := []string{"Backlog", "To Design", "Designing Process", "W4 Design Approval", "Approved Designs", "Marketing"}
	if !sliceEqual(result, expected) {
		t.Errorf("ParseStatusFilter real dev view = %v, want %v", result, expected)
	}
}

func TestParseStatusFilter_RealQAView(t *testing.T) {
	filter := `-status:Backlog,"To Design","Designing Process","W4 Design Approval","Approved Designs","Ready for Dev","In Progress",Marketing -is:closed`
	result := ParseStatusFilter(filter)
	expected := []string{"Backlog", "To Design", "Designing Process", "W4 Design Approval", "Approved Designs", "Ready for Dev", "In Progress", "Marketing"}
	if !sliceEqual(result, expected) {
		t.Errorf("ParseStatusFilter real QA view = %v, want %v", result, expected)
	}
}

func TestParseStatusFilter_RealDesignersView(t *testing.T) {
	filter := `-status:Backlog,"Ready for Dev","In Progress","In Review (QA)","Ready for Deploy"`
	result := ParseStatusFilter(filter)
	expected := []string{"Backlog", "Ready for Dev", "In Progress", "In Review (QA)", "Ready for Deploy"}
	if !sliceEqual(result, expected) {
		t.Errorf("ParseStatusFilter real designers view = %v, want %v", result, expected)
	}
}

func TestParseIsClosedFilter(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		filter string
		want   bool
	}{
		{name: "empty", filter: "", want: false},
		{name: "closed only", filter: "-is:closed", want: true},
		{name: "status and closed", filter: "-status:Backlog -is:closed", want: true},
		{name: "status only", filter: "-status:Backlog", want: false},
		{name: "open only", filter: "-is:open", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ParseIsClosedFilter(tt.filter); got != tt.want {
				t.Fatalf("ParseIsClosedFilter(%q) = %v, want %v", tt.filter, got, tt.want)
			}
		})
	}
}

// FilterColumns tests

func TestFilterColumns_RemovesExcluded(t *testing.T) {
	cols := []column{
		{name: "Backlog", itemID: "1", items: nil},
		{name: "To Do", itemID: "2", items: nil},
		{name: "In Progress", itemID: "3", items: nil},
		{name: "Done", itemID: "4", items: nil},
	}
	excluded := []string{"Backlog", "Done"}

	result := FilterColumns(cols, excluded)

	if len(result) != 2 {
		t.Errorf("FilterColumns with 4 cols, excluding 2 = %d cols, want 2", len(result))
	}

	expectedNames := map[string]bool{"To Do": true, "In Progress": true}
	for _, col := range result {
		if !expectedNames[col.name] {
			t.Errorf("FilterColumns result includes unexpected column: %s", col.name)
		}
	}
}

func TestFilterColumns_EmptyExclusions(t *testing.T) {
	cols := []column{
		{name: "Backlog", itemID: "1", items: nil},
		{name: "To Do", itemID: "2", items: nil},
		{name: "Done", itemID: "3", items: nil},
	}

	result := FilterColumns(cols, nil)

	if len(result) != len(cols) {
		t.Errorf("FilterColumns with nil exclusions = %d cols, want %d", len(result), len(cols))
	}

	for i, col := range result {
		if col.name != cols[i].name {
			t.Errorf("FilterColumns changed column order or content")
		}
	}
}

// ParseRepoFilter tests

func TestParseRepoFilter(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		filter string
		want   []string
	}{
		{name: "empty", filter: "", want: nil},
		{name: "no repo clause", filter: "-is:closed", want: nil},
		{name: "single repo", filter: "-repo:hifihub/hh-scraping", want: []string{"hifihub/hh-scraping"}},
		{name: "multiple repos", filter: "-repo:org/repo-a -repo:org/repo-b", want: []string{"org/repo-a", "org/repo-b"}},
		{name: "mixed with other filters", filter: "-status:Backlog -repo:org/repo -label:marketing", want: []string{"org/repo"}},
		{name: "repo with closed filter", filter: "-repo:org/repo -is:closed", want: []string{"org/repo"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseRepoFilter(tt.filter)
			if !sliceEqual(got, tt.want) {
				t.Fatalf("ParseRepoFilter(%q) = %v, want %v", tt.filter, got, tt.want)
			}
		})
	}
}

// ParseLabelFilter tests

func TestParseLabelFilter(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		filter string
		want   []string
	}{
		{name: "empty", filter: "", want: nil},
		{name: "no label clause", filter: "-is:closed", want: nil},
		{name: "single label", filter: "-label:marketing", want: []string{"marketing"}},
		{name: "multiple labels", filter: "-label:marketing -label:design", want: []string{"marketing", "design"}},
		{name: "quoted label", filter: "-label:\"bug fix\"", want: []string{"bug fix"}},
		{name: "mixed with other filters", filter: "-status:Backlog -label:marketing -repo:a/b", want: []string{"marketing"}},
		{name: "multiple labels with quoted", filter: "-label:marketing -label:\"bug fix\" -label:design", want: []string{"marketing", "bug fix", "design"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseLabelFilter(tt.filter)
			if !sliceEqual(got, tt.want) {
				t.Fatalf("ParseLabelFilter(%q) = %v, want %v", tt.filter, got, tt.want)
			}
		})
	}
}

// FilterItemsByRepoAndLabel tests

func TestFilterItemsByRepoAndLabel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		items          []github.ProjectItem
		excludedRepos  []string
		excludedLabels []string
		expectedLen    int
		expectedTitles []string
	}{
		{
			name: "no filters",
			items: []github.ProjectItem{
				{ID: "1", Title: "Issue 1", Type: "Issue", RepoOwner: "org", RepoName: "repo", Content: &github.Issue{Labels: []github.Label{}}},
				{ID: "2", Title: "PR 1", Type: "PullRequest", RepoOwner: "org", RepoName: "repo", Content: &github.PullRequest{}},
				{ID: "3", Title: "Draft 1", Type: "DraftIssue", RepoOwner: "", RepoName: "", Content: nil},
			},
			excludedRepos:  nil,
			excludedLabels: nil,
			expectedLen:    3,
			expectedTitles: []string{"Issue 1", "PR 1", "Draft 1"},
		},
		{
			name: "repo exclusion",
			items: []github.ProjectItem{
				{ID: "1", Title: "Excluded issue", Type: "Issue", RepoOwner: "org", RepoName: "excluded", Content: &github.Issue{Labels: []github.Label{}}},
				{ID: "2", Title: "Kept issue", Type: "Issue", RepoOwner: "org", RepoName: "kept", Content: &github.Issue{Labels: []github.Label{}}},
			},
			excludedRepos:  []string{"org/excluded"},
			excludedLabels: nil,
			expectedLen:    1,
			expectedTitles: []string{"Kept issue"},
		},
		{
			name: "label exclusion",
			items: []github.ProjectItem{
				{ID: "1", Title: "Marketing issue", Type: "Issue", RepoOwner: "org", RepoName: "repo", Content: &github.Issue{Labels: []github.Label{{Name: "marketing"}}}},
				{ID: "2", Title: "Bug issue", Type: "Issue", RepoOwner: "org", RepoName: "repo", Content: &github.Issue{Labels: []github.Label{{Name: "bug"}}}},
			},
			excludedRepos:  nil,
			excludedLabels: []string{"marketing"},
			expectedLen:    1,
			expectedTitles: []string{"Bug issue"},
		},
		{
			name: "label case insensitive",
			items: []github.ProjectItem{
				{ID: "1", Title: "Capital marketing", Type: "Issue", RepoOwner: "org", RepoName: "repo", Content: &github.Issue{Labels: []github.Label{{Name: "Marketing"}}}},
				{ID: "2", Title: "Lowercase marketing", Type: "Issue", RepoOwner: "org", RepoName: "repo", Content: &github.Issue{Labels: []github.Label{{Name: "marketing"}}}},
			},
			excludedRepos:  nil,
			excludedLabels: []string{"marketing"},
			expectedLen:    0,
			expectedTitles: []string{},
		},
		{
			name: "PR filtered when label active",
			items: []github.ProjectItem{
				{ID: "1", Title: "PR without labels", Type: "PullRequest", RepoOwner: "org", RepoName: "repo", Content: &github.PullRequest{}},
			},
			excludedRepos:  nil,
			excludedLabels: []string{"marketing"},
			expectedLen:    0,
			expectedTitles: []string{},
		},
		{
			name: "DraftIssue passes through",
			items: []github.ProjectItem{
				{ID: "1", Title: "Draft item", Type: "DraftIssue", RepoOwner: "", RepoName: "", Content: nil},
			},
			excludedRepos:  nil,
			excludedLabels: []string{"marketing"},
			expectedLen:    1,
			expectedTitles: []string{"Draft item"},
		},
		{
			name: "combined repo and label",
			items: []github.ProjectItem{
				{ID: "1", Title: "Excluded repo", Type: "Issue", RepoOwner: "org", RepoName: "excluded", Content: &github.Issue{Labels: []github.Label{}}},
				{ID: "2", Title: "Marketing label", Type: "Issue", RepoOwner: "org", RepoName: "repo", Content: &github.Issue{Labels: []github.Label{{Name: "marketing"}}}},
				{ID: "3", Title: "Clean issue", Type: "Issue", RepoOwner: "org", RepoName: "repo", Content: &github.Issue{Labels: []github.Label{{Name: "bug"}}}},
			},
			excludedRepos:  []string{"org/excluded"},
			excludedLabels: []string{"marketing"},
			expectedLen:    1,
			expectedTitles: []string{"Clean issue"},
		},
		{
			name:           "empty items list",
			items:          []github.ProjectItem{},
			excludedRepos:  []string{"org/excluded"},
			excludedLabels: []string{"marketing"},
			expectedLen:    0,
			expectedTitles: []string{},
		},
		{
			name: "multiple label exclusions",
			items: []github.ProjectItem{
				{ID: "1", Title: "Marketing issue", Type: "Issue", RepoOwner: "org", RepoName: "repo", Content: &github.Issue{Labels: []github.Label{{Name: "marketing"}}}},
				{ID: "2", Title: "Design issue", Type: "Issue", RepoOwner: "org", RepoName: "repo", Content: &github.Issue{Labels: []github.Label{{Name: "design"}}}},
				{ID: "3", Title: "Bug issue", Type: "Issue", RepoOwner: "org", RepoName: "repo", Content: &github.Issue{Labels: []github.Label{{Name: "bug"}}}},
			},
			excludedRepos:  nil,
			excludedLabels: []string{"marketing", "design"},
			expectedLen:    1,
			expectedTitles: []string{"Bug issue"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FilterItemsByRepoAndLabel(tt.items, tt.excludedRepos, tt.excludedLabels)

			if len(got) != tt.expectedLen {
				t.Fatalf("FilterItemsByRepoAndLabel returned %d items, want %d", len(got), tt.expectedLen)
			}

			gotTitles := make([]string, len(got))
			for i, item := range got {
				gotTitles[i] = item.Title
			}

			if !sliceEqual(gotTitles, tt.expectedTitles) {
				t.Errorf("FilterItemsByRepoAndLabel titles = %v, want %v", gotTitles, tt.expectedTitles)
			}
		})
	}
}

// sliceEqual compares two string slices for equality.
func sliceEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
