package board

import (
	"strings"
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
