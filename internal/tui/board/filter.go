package board

import (
	"strings"

	github "github.com/ifloresarg/gh-projects/internal/github"
)

// ParseStatusFilter parses a GitHub view filter string and returns the list of excluded status names.
// It extracts the -status: clause and parses CSV-like values (respecting double-quotes).
// Example: "-status:Backlog,\"To Design\",\"W4 Design Approval\" -is:closed"
// Returns: ["Backlog", "To Design", "W4 Design Approval"]
// Returns nil if filter is empty or contains no -status: clause.
func ParseStatusFilter(filter string) []string {
	if filter == "" {
		return nil
	}

	_, afterStatus, found := strings.Cut(filter, "-status:")
	if !found {
		return nil
	}

	var statusClause string
	statusClause, _, _ = strings.Cut(afterStatus, " -")
	if statusClause == afterStatus {
		// No more tokens, take everything after -status:
		statusClause = afterStatus
	}

	// Parse CSV-like values, respecting double-quotes
	statuses := parseCSVValues(statusClause)
	if len(statuses) == 0 {
		return nil
	}
	return statuses
}

// ParseIsClosedFilter returns true if the filter string contains the -is:closed clause.
func ParseIsClosedFilter(filter string) bool {
	return strings.Contains(filter, "-is:closed")
}

// ParseRepoFilter parses a GitHub view filter string and returns the list of repository names.
// It extracts all -repo: clauses from the filter string.
// Example: "-status:Backlog -repo:org/repo -label:marketing"
// Returns: ["org/repo"]
// Returns nil if filter is empty or contains no -repo: clauses.
func ParseRepoFilter(filter string) []string {
	if filter == "" {
		return nil
	}

	var repos []string
	for _, token := range strings.Fields(filter) {
		if val, ok := strings.CutPrefix(token, "-repo:"); ok {
			repos = append(repos, val)
		}
	}

	if len(repos) == 0 {
		return nil
	}
	return repos
}

// ParseLabelFilter parses a GitHub view filter string and returns the list of label names.
// It extracts all -label: clauses from the filter string.
// Labels may be quoted: -label:"bug fix"
// Example: "-label:marketing -label:\"bug fix\" -label:design"
// Returns: ["marketing", "bug fix", "design"]
// Returns nil if filter is empty or contains no -label: clauses.
func ParseLabelFilter(filter string) []string {
	if filter == "" {
		return nil
	}

	var labels []string
	i := 0

	for i < len(filter) {
		// Look for -label:
		idx := strings.Index(filter[i:], "-label:")
		if idx == -1 {
			break
		}

		// Move past -label:
		i += idx + len("-label:")

		// Extract label value (quoted or unquoted)
		if i < len(filter) && filter[i] == '"' {
			// Quoted label: skip opening quote and find closing quote
			i++
			end := i
			for end < len(filter) && filter[end] != '"' {
				end++
			}
			if end < len(filter) {
				// Found closing quote
				labels = append(labels, filter[i:end])
				i = end + 1 // Move past closing quote
			}
		} else {
			// Unquoted label: read until space or end of string
			start := i
			for i < len(filter) && filter[i] != ' ' {
				i++
			}
			if i > start {
				labels = append(labels, filter[start:i])
			}
		}
	}

	if len(labels) == 0 {
		return nil
	}
	return labels
}

// parseCSVValues parses a CSV-like string with double-quote support.
// Example: `Backlog,"To Design","W4 Design Approval"`
// Returns: ["Backlog", "To Design", "W4 Design Approval"]
func parseCSVValues(csvStr string) []string {
	var result []string
	var current strings.Builder
	inQuotes := false
	i := 0

	for i < len(csvStr) {
		ch := csvStr[i]

		if ch == '"' {
			inQuotes = !inQuotes
			i++
			continue
		}

		if ch == ',' && !inQuotes {
			// End of value
			value := strings.TrimSpace(current.String())
			if value != "" {
				result = append(result, value)
			}
			current.Reset()
			i++
			continue
		}

		current.WriteByte(ch)
		i++
	}

	// Add final value
	value := strings.TrimSpace(current.String())
	if value != "" {
		result = append(result, value)
	}

	return result
}

// FilterColumns returns a new slice excluding columns whose name matches any excluded status.
// Match is case-sensitive.
// If excludedStatuses is nil or empty, returns all columns unchanged.
func FilterColumns(columns []column, excludedStatuses []string) []column {
	if len(excludedStatuses) == 0 {
		return columns
	}

	// Build a set for O(1) lookup
	excludeSet := make(map[string]bool)
	for _, status := range excludedStatuses {
		excludeSet[status] = true
	}

	var result []column
	for _, col := range columns {
		// Exclude if name matches an excluded status
		if !excludeSet[col.name] {
			result = append(result, col)
		}
	}

	return result
}

// FilterItemsByRepoAndLabel returns a new slice excluding items that match excluded repositories or labels.
// Repo matching is case-sensitive against "owner/name" format.
// Label matching is case-insensitive.
// DraftIssue items (Content == nil) are never excluded.
// PullRequest items are always excluded if any excludedLabels exist (conservative approach, can't verify labels).
// If both excludedRepos and excludedLabels are empty, returns items unchanged.
func FilterItemsByRepoAndLabel(items []github.ProjectItem, excludedRepos, excludedLabels []string) []github.ProjectItem {
	if len(excludedRepos) == 0 && len(excludedLabels) == 0 {
		return items
	}

	// Build repo set for O(1) lookup
	repoSet := make(map[string]bool)
	for _, repo := range excludedRepos {
		repoSet[repo] = true
	}

	// Build label set for O(1) lookup (lowercase for case-insensitive matching)
	labelSet := make(map[string]bool)
	for _, label := range excludedLabels {
		labelSet[strings.ToLower(label)] = true
	}

	var result []github.ProjectItem
	for _, item := range items {
		excluded := false

		// Check repo exclusion
		if len(repoSet) > 0 {
			repoKey := item.RepoOwner + "/" + item.RepoName
			if repoSet[repoKey] {
				excluded = true
			}
		}

		// Check label exclusion (only if not already excluded)
		if !excluded && len(labelSet) > 0 {
			// DraftIssue has Content == nil, never excluded by label
			if item.Content == nil {
				excluded = false
			} else if issue, ok := item.Content.(*github.Issue); ok {
				// Check if any of the issue's labels match the exclusion set
				for _, label := range issue.Labels {
					if labelSet[strings.ToLower(label.Name)] {
						excluded = true
						break
					}
				}
			} else if _, ok := item.Content.(*github.PullRequest); ok {
				// PullRequest: exclude conservatively since we can't verify labels
				excluded = true
			}
		}

		if !excluded {
			result = append(result, item)
		}
	}

	return result
}
