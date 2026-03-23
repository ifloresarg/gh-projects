package board

import (
	"testing"
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

// Helper function for test assertions
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
