package board

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ifloresarg/gh-projects/internal/config"
	"github.com/ifloresarg/gh-projects/internal/github"
	"github.com/ifloresarg/gh-projects/internal/tui/components"
)

func testBoardFields() []github.ProjectField {
	return []github.ProjectField{{
		ID:       "status-field",
		Name:     "Status",
		DataType: "SINGLE_SELECT",
		Options: []github.FieldOption{
			{ID: "todo", Name: "Todo"},
			{ID: "doing", Name: "In Progress"},
			{ID: "done", Name: "Done"},
		},
	}}
}

func testBoardItems() []github.ProjectItem {
	return []github.ProjectItem{
		{ID: "1", Title: "Implement GraphQL client", Type: "Issue", StatusID: "todo", Content: &github.Issue{Title: "Implement GraphQL client", Number: 101}},
		{ID: "2", Title: "Add cache invalidation", Type: "Issue", StatusID: "doing", Content: &github.Issue{Title: "Add cache invalidation", Number: 102}},
		{ID: "3", Title: "Refine board rendering", Type: "PullRequest", StatusID: "done", Content: &github.PullRequest{Title: "Refine board rendering", Number: 55}},
		{ID: "4", Title: "Untriaged draft", Type: "DraftIssue", StatusID: ""},
	}
}

func testScrollFields(statusCount int) []github.ProjectField {
	options := make([]github.FieldOption, 0, statusCount)
	for i := range statusCount {
		options = append(options, github.FieldOption{ID: fmt.Sprintf("status-%d", i), Name: fmt.Sprintf("Col-%d", i)})
	}

	return []github.ProjectField{{
		ID:       "status-field",
		Name:     "Status",
		DataType: "SINGLE_SELECT",
		Options:  options,
	}}
}

func testScrollItems(statusCount int) []github.ProjectItem {
	items := make([]github.ProjectItem, 0, statusCount)
	for i := range statusCount {
		items = append(items, github.ProjectItem{
			ID:       fmt.Sprintf("item-%d", i),
			Title:    fmt.Sprintf("Item %d", i),
			Type:     "Issue",
			StatusID: fmt.Sprintf("status-%d", i),
			Content:  &github.Issue{Title: fmt.Sprintf("Item %d", i), Number: i + 1},
		})
	}

	return items
}

func testVerticalScrollFields() []github.ProjectField {
	return []github.ProjectField{{
		ID:       "status-field",
		Name:     "Status",
		DataType: "SINGLE_SELECT",
		Options: []github.FieldOption{
			{ID: "todo", Name: "Todo"},
		},
	}}
}

func testVerticalScrollItems(n int) []github.ProjectItem {
	items := make([]github.ProjectItem, 0, n)
	for i := range n {
		items = append(items, github.ProjectItem{
			ID:       fmt.Sprintf("card-%d", i),
			Title:    fmt.Sprintf("Card-%d", i),
			Type:     "Issue",
			StatusID: "todo",
			Content:  &github.Issue{Title: fmt.Sprintf("Card-%d", i), Number: i + 1},
		})
	}

	return items
}

func TestBuildColumnsGroupsItemsByStatus(t *testing.T) {
	t.Parallel()

	columns := buildColumns(testBoardItems(), testBoardFields())
	if len(columns) != 3 {
		t.Fatalf("buildColumns() len = %d, want 3", len(columns))
	}
	if columns[0].name != "Todo" || len(columns[0].items) != 1 {
		t.Fatalf("unexpected todo column = %#v", columns[0])
	}
}

func TestBoardViewShowsLoadedColumns(t *testing.T) {
	t.Parallel()

	m := New(&github.MockClient{}, github.Project{ID: "PVT_1", Title: "Roadmap"})
	m, _ = m.Update(tea.WindowSizeMsg{Width: 140, Height: 24})
	m, _ = m.Update(boardLoadedMsg{items: testBoardItems(), fields: testBoardFields()})

	view := m.View()
	for _, fragment := range []string{"Todo (1)", "In Progress (1)", "Done (1)", "Implement GraphQL"} {
		if !strings.Contains(view, fragment) {
			t.Fatalf("View() missing %q in %q", fragment, view)
		}
	}
}

func TestBoardApplyFilterUpdatesMatchCounts(t *testing.T) {
	t.Parallel()

	m := New(&github.MockClient{}, github.Project{ID: "PVT_1", Title: "Roadmap"})
	m, _ = m.Update(tea.WindowSizeMsg{Width: 140, Height: 24})
	m, _ = m.Update(boardLoadedMsg{items: testBoardItems(), fields: testBoardFields()})
	m.searchQuery = "cache"
	m.applyFilter()

	if got := m.matchCount(); got != 1 {
		t.Fatalf("matchCount() = %d, want 1", got)
	}
	view := m.View()
	if !strings.Contains(view, "Filter: 1 of 3 items") {
		t.Fatalf("View() = %q, want filter summary", view)
	}
	if strings.Contains(view, "Implement GraphQL client") {
		t.Fatalf("View() unexpectedly contains filtered-out item: %q", view)
	}
}

func TestBoardViewShowsLoadError(t *testing.T) {
	t.Parallel()

	m := New(&github.MockClient{}, github.Project{ID: "PVT_1", Title: "Roadmap"})
	m, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m, _ = m.Update(boardLoadedMsg{err: errors.New("network error")})

	view := m.View()
	if !strings.Contains(view, "Error: network error") {
		t.Fatalf("View() = %q, want error message", view)
	}
}

func TestScrollOffset_Navigation(t *testing.T) {
	t.Parallel()

	m := New(&github.MockClient{}, github.Project{ID: "PVT_1", Title: "Roadmap"})
	m, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m, _ = m.Update(boardLoadedMsg{items: testScrollItems(9), fields: testScrollFields(9)})

	for range 3 {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	}

	if m.activeCol != 3 {
		t.Fatalf("activeCol = %d, want 3", m.activeCol)
	}
	if m.scrollOffset <= 0 {
		t.Fatalf("scrollOffset = %d, want > 0", m.scrollOffset)
	}

	view := m.View()
	if !strings.Contains(view, "Col-3 (1)") {
		t.Fatalf("View() missing active column header: %q", view)
	}
	if strings.Contains(view, "Col-0 (1)") {
		t.Fatalf("View() unexpectedly contains hidden column header: %q", view)
	}
}

func TestScrollOffset_NoWrap(t *testing.T) {
	t.Parallel()

	m := New(&github.MockClient{}, github.Project{ID: "PVT_1", Title: "Roadmap"})
	m, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m, _ = m.Update(boardLoadedMsg{items: testScrollItems(4), fields: testScrollFields(4)})
	m.activeCol = len(m.columns) - 1
	m.clampScrollOffset()

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})

	if m.activeCol != len(m.columns)-1 {
		t.Fatalf("activeCol = %d, want %d", m.activeCol, len(m.columns)-1)
	}
}

func TestScrollOffset_Clamp(t *testing.T) {
	t.Parallel()

	m := New(&github.MockClient{}, github.Project{ID: "PVT_1", Title: "Roadmap"})
	m, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m, _ = m.Update(boardLoadedMsg{items: testScrollItems(9), fields: testScrollFields(9)})
	m.activeCol = len(m.columns) - 1
	m.scrollOffset = 999
	m.clampScrollOffset()

	maxVisible := max(m.width/(columnWidth+2), 1)
	maxOffset := len(m.columns) - maxVisible
	maxOffset = max(maxOffset, 0)

	if m.scrollOffset < 0 || m.scrollOffset > maxOffset {
		t.Fatalf("scrollOffset = %d, want in [0,%d]", m.scrollOffset, maxOffset)
	}
}

func TestScrollOffset_WindowResize(t *testing.T) {
	t.Parallel()

	m := New(&github.MockClient{}, github.Project{ID: "PVT_1", Title: "Roadmap"})
	m, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m, _ = m.Update(boardLoadedMsg{items: testScrollItems(9), fields: testScrollFields(9)})
	m.activeCol = 6
	m.scrollOffset = 5
	m.clampScrollOffset()

	m, _ = m.Update(tea.WindowSizeMsg{Width: 9999, Height: 24})

	if m.scrollOffset != 0 {
		t.Fatalf("scrollOffset = %d, want 0", m.scrollOffset)
	}
}

func TestColumnFiltering_WithView(t *testing.T) {
	t.Parallel()

	fields := []github.ProjectField{{
		ID:       "status-field",
		Name:     "Status",
		DataType: "SINGLE_SELECT",
		Options: []github.FieldOption{
			{ID: "backlog", Name: "Backlog"},
			{ID: "todo", Name: "Todo"},
			{ID: "doing", Name: "In Progress"},
			{ID: "review", Name: "Review"},
			{ID: "done", Name: "Done"},
		},
	}}
	items := []github.ProjectItem{
		{ID: "1", Title: "A", Type: "Issue", StatusID: "backlog", Content: &github.Issue{Title: "A", Number: 1}},
		{ID: "2", Title: "B", Type: "Issue", StatusID: "todo", Content: &github.Issue{Title: "B", Number: 2}},
		{ID: "3", Title: "C", Type: "Issue", StatusID: "doing", Content: &github.Issue{Title: "C", Number: 3}},
		{ID: "4", Title: "D", Type: "Issue", StatusID: "review", Content: &github.Issue{Title: "D", Number: 4}},
		{ID: "5", Title: "E", Type: "Issue", StatusID: "done", Content: &github.Issue{Title: "E", Number: 5}},
		{ID: "6", Title: "F", Type: "Issue", StatusID: "", Content: &github.Issue{Title: "F", Number: 6}},
	}

	m := New(&github.MockClient{}, github.Project{ID: "PVT_1", Title: "Roadmap"})
	m, _ = m.Update(tea.WindowSizeMsg{Width: 140, Height: 24})
	m, _ = m.Update(boardLoadedMsg{items: items, fields: fields})

	m.SetActiveView(&github.ProjectView{Filter: `-status:Backlog,"In Progress"`})

	if len(m.columns) != 3 {
		t.Fatalf("len(columns) = %d, want 3 (3 status columns)", len(m.columns))
	}

	gotNames := make(map[string]bool)
	for _, col := range m.columns {
		gotNames[col.name] = true
	}

	if gotNames["Backlog"] || gotNames["In Progress"] {
		t.Fatalf("excluded columns still present: %#v", gotNames)
	}
	for _, want := range []string{"Todo", "Review", "Done"} {
		if !gotNames[want] {
			t.Fatalf("missing expected column %q in %#v", want, gotNames)
		}
	}
}

func TestColumnFiltering_NoView(t *testing.T) {
	t.Parallel()

	m := New(&github.MockClient{}, github.Project{ID: "PVT_1", Title: "Roadmap"})
	m, _ = m.Update(tea.WindowSizeMsg{Width: 140, Height: 24})
	m, _ = m.Update(boardLoadedMsg{items: testBoardItems(), fields: testBoardFields()})

	m.SetActiveView(&github.ProjectView{Filter: `-status:Todo`})
	if len(m.columns) != len(m.allColumns)-1 {
		t.Fatalf("len(columns) after filtered view = %d, want %d", len(m.columns), len(m.allColumns)-1)
	}

	m.SetActiveView(nil)

	if len(m.columns) != len(m.allColumns) {
		t.Fatalf("len(columns) = %d, want %d", len(m.columns), len(m.allColumns))
	}
}

func TestColumnFiltering_WithClosedFilter(t *testing.T) {
	t.Parallel()

	fields := []github.ProjectField{{
		ID:       "status-field",
		Name:     "Status",
		DataType: "SINGLE_SELECT",
		Options: []github.FieldOption{
			{ID: "todo", Name: "Todo"},
			{ID: "done", Name: "Done"},
		},
	}}
	items := []github.ProjectItem{
		{ID: "1", Title: "Open issue", Type: "Issue", StatusID: "todo", Content: &github.Issue{Title: "Open issue", Number: 1, State: "OPEN"}},
		{ID: "2", Title: "Closed issue", Type: "Issue", StatusID: "todo", Content: &github.Issue{Title: "Closed issue", Number: 2, State: "CLOSED"}},
		{ID: "3", Title: "Closed PR", Type: "PullRequest", StatusID: "done", Content: &github.PullRequest{Title: "Closed PR", Number: 3, State: "CLOSED"}},
		{ID: "4", Title: "Merged PR", Type: "PullRequest", StatusID: "done", Content: &github.PullRequest{Title: "Merged PR", Number: 4, State: "MERGED"}},
		{ID: "5", Title: "Draft issue", Type: "DraftIssue", StatusID: "todo"},
	}

	m := New(&github.MockClient{}, github.Project{ID: "PVT_1", Title: "Roadmap"})
	m, _ = m.Update(tea.WindowSizeMsg{Width: 140, Height: 24})
	m, _ = m.Update(boardLoadedMsg{items: items, fields: fields})

	m.showClosedItems = true
	m.SetActiveView(&github.ProjectView{Filter: "-is:closed"})

	if len(m.viewColumns) != 2 {
		t.Fatalf("len(viewColumns) = %d, want 2", len(m.viewColumns))
	}

	gotByColumn := make(map[string][]string)
	for _, col := range m.viewColumns {
		for _, item := range col.items {
			gotByColumn[col.name] = append(gotByColumn[col.name], item.Title)
		}
	}

	if !sliceEqual(gotByColumn["Todo"], []string{"Open issue", "Closed issue", "Draft issue"}) {
		t.Fatalf("Todo items = %v, want %v", gotByColumn["Todo"], []string{"Open issue", "Closed issue", "Draft issue"})
	}
	if !sliceEqual(gotByColumn["Done"], []string{"Closed PR", "Merged PR"}) {
		t.Fatalf("Done items = %v, want %v", gotByColumn["Done"], []string{"Closed PR", "Merged PR"})
	}

	gotDisplayed := make(map[string][]string)
	for _, col := range m.columns {
		for _, item := range col.items {
			gotDisplayed[col.name] = append(gotDisplayed[col.name], item.Title)
		}
	}

	if !sliceEqual(gotDisplayed["Todo"], []string{"Open issue", "Closed issue", "Draft issue"}) {
		t.Fatalf("displayed Todo items = %v, want %v", gotDisplayed["Todo"], []string{"Open issue", "Closed issue", "Draft issue"})
	}
	if !sliceEqual(gotDisplayed["Done"], []string{"Closed PR", "Merged PR"}) {
		t.Fatalf("displayed Done items = %v, want %v", gotDisplayed["Done"], []string{"Closed PR", "Merged PR"})
	}
}

func TestColumnFiltering_ResetOnViewChange(t *testing.T) {
	t.Parallel()

	m := New(&github.MockClient{}, github.Project{ID: "PVT_1", Title: "Roadmap"})
	m, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m, _ = m.Update(boardLoadedMsg{items: testScrollItems(9), fields: testScrollFields(9)})
	m.activeCol = 3
	m.activeCard = 1
	m.scrollOffset = 2

	m.SetActiveView(&github.ProjectView{Filter: `-status:Col-0`})

	if m.activeCol != 0 {
		t.Fatalf("activeCol = %d, want 0", m.activeCol)
	}
	if m.scrollOffset != 0 {
		t.Fatalf("scrollOffset = %d, want 0", m.scrollOffset)
	}
	if m.activeCard != 0 {
		t.Fatalf("activeCard = %d, want 0", m.activeCard)
	}
}

func TestCardScrollOffset_ScrollDown(t *testing.T) {
	t.Parallel()

	m := New(&github.MockClient{}, github.Project{ID: "PVT_1", Title: "Roadmap"})
	m, _ = m.Update(tea.WindowSizeMsg{Width: 140, Height: 24})
	m, _ = m.Update(boardLoadedMsg{items: testVerticalScrollItems(20), fields: testVerticalScrollFields()})

	for range 15 {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	}

	if m.activeCard != 15 {
		t.Fatalf("activeCard = %d, want 15", m.activeCard)
	}
	if m.cardScrollOffset <= 0 {
		t.Fatalf("cardScrollOffset = %d, want > 0", m.cardScrollOffset)
	}

	view := m.View()
	if !strings.Contains(view, fmt.Sprintf("Card-%d", m.cardScrollOffset)) {
		t.Fatalf("View() missing scrolled window start Card-%d: %q", m.cardScrollOffset, view)
	}
	if !strings.Contains(view, "▲") {
		t.Fatalf("View() missing top scroll indicator: %q", view)
	}

	// Navigate all the way to bottom
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
	if m.activeCard != 19 {
		t.Fatalf("activeCard = %d after G, want 19", m.activeCard)
	}
}

func TestCardScrollOffset_ScrollUp(t *testing.T) {
	t.Parallel()

	m := New(&github.MockClient{}, github.Project{ID: "PVT_1", Title: "Roadmap"})
	m, _ = m.Update(tea.WindowSizeMsg{Width: 140, Height: 24})
	m, _ = m.Update(boardLoadedMsg{items: testVerticalScrollItems(20), fields: testVerticalScrollFields()})

	for range 15 {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	}
	for range 15 {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	}

	if m.activeCard != 0 {
		t.Fatalf("activeCard = %d, want 0", m.activeCard)
	}
	if m.cardScrollOffset != 0 {
		t.Fatalf("cardScrollOffset = %d, want 0", m.cardScrollOffset)
	}

	view := m.View()
	if !strings.Contains(view, "Card-0") {
		t.Fatalf("View() missing Card-0: %q", view)
	}
	if strings.Contains(view, "▲") {
		t.Fatalf("View() unexpectedly shows top scroll indicator: %q", view)
	}
}

func TestCardScrollOffset_GoToTop(t *testing.T) {
	t.Parallel()

	m := New(&github.MockClient{}, github.Project{ID: "PVT_1", Title: "Roadmap"})
	m, _ = m.Update(tea.WindowSizeMsg{Width: 140, Height: 24})
	m, _ = m.Update(boardLoadedMsg{items: testVerticalScrollItems(20), fields: testVerticalScrollFields()})

	for range 15 {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	}
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})

	if m.activeCard != 0 {
		t.Fatalf("activeCard = %d, want 0", m.activeCard)
	}
	if m.cardScrollOffset != 0 {
		t.Fatalf("cardScrollOffset = %d, want 0", m.cardScrollOffset)
	}

	view := m.View()
	if !strings.Contains(view, "Card-0") {
		t.Fatalf("View() missing Card-0: %q", view)
	}
	if strings.Contains(view, "▲") {
		t.Fatalf("View() unexpectedly shows top scroll indicator: %q", view)
	}
}

func TestCardScrollOffset_GoToBottom(t *testing.T) {
	t.Parallel()

	m := New(&github.MockClient{}, github.Project{ID: "PVT_1", Title: "Roadmap"})
	m, _ = m.Update(tea.WindowSizeMsg{Width: 140, Height: 24})
	m, _ = m.Update(boardLoadedMsg{items: testVerticalScrollItems(20), fields: testVerticalScrollFields()})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})

	if m.activeCard != 19 {
		t.Fatalf("activeCard = %d, want 19", m.activeCard)
	}
	if m.cardScrollOffset <= 0 {
		t.Fatalf("cardScrollOffset = %d, want > 0", m.cardScrollOffset)
	}

	view := m.View()
	if !strings.Contains(view, fmt.Sprintf("Card-%d", m.cardScrollOffset)) {
		t.Fatalf("View() missing scrolled window start Card-%d: %q", m.cardScrollOffset, view)
	}
	if !strings.Contains(view, "▲") {
		t.Fatalf("View() missing top scroll indicator at bottom: %q", view)
	}
}

func TestCardScrollOffset_ColumnSwitch(t *testing.T) {
	t.Parallel()

	fields := []github.ProjectField{{
		ID:       "status-field",
		Name:     "Status",
		DataType: "SINGLE_SELECT",
		Options: []github.FieldOption{
			{ID: "col-a", Name: "Col-A"},
			{ID: "col-b", Name: "Col-B"},
		},
	}}
	items := make([]github.ProjectItem, 0, 16)
	for i := range 15 {
		items = append(items, github.ProjectItem{
			ID:       fmt.Sprintf("col-a-%d", i),
			Title:    fmt.Sprintf("Col-A Card-%d", i),
			Type:     "Issue",
			StatusID: "col-a",
			Content:  &github.Issue{Title: fmt.Sprintf("Col-A Card-%d", i), Number: i + 1},
		})
	}
	items = append(items, github.ProjectItem{
		ID:       "col-b-0",
		Title:    "Col-B Card-0",
		Type:     "Issue",
		StatusID: "col-b",
		Content:  &github.Issue{Title: "Col-B Card-0", Number: 16},
	})

	m := New(&github.MockClient{}, github.Project{ID: "PVT_1", Title: "Roadmap"})
	m, _ = m.Update(tea.WindowSizeMsg{Width: 140, Height: 24})
	m, _ = m.Update(boardLoadedMsg{items: items, fields: fields})

	for range 10 {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	}
	if m.cardScrollOffset <= 0 {
		t.Fatalf("cardScrollOffset before switch = %d, want > 0", m.cardScrollOffset)
	}

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})

	if m.activeCol != 1 {
		t.Fatalf("activeCol = %d, want 1", m.activeCol)
	}
	if m.cardScrollOffset != 0 {
		t.Fatalf("cardScrollOffset = %d, want 0 after column switch", m.cardScrollOffset)
	}

	// Switch back to col 0
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	if m.activeCol != 0 {
		t.Fatalf("activeCol = %d after h, want 0", m.activeCol)
	}
	if m.cardScrollOffset > m.activeCard {
		t.Fatalf("cardScrollOffset = %d, want <= activeCard %d after switch back to col 0", m.cardScrollOffset, m.activeCard)
	}
}

func TestCardScrollOffset_WindowResize(t *testing.T) {
	t.Parallel()

	m := New(&github.MockClient{}, github.Project{ID: "PVT_1", Title: "Roadmap"})
	m, _ = m.Update(tea.WindowSizeMsg{Width: 140, Height: 24})
	m, _ = m.Update(boardLoadedMsg{items: testVerticalScrollItems(20), fields: testVerticalScrollFields()})

	for range 15 {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	}
	offset := m.cardScrollOffset

	m, _ = m.Update(tea.WindowSizeMsg{Width: 9999, Height: 60})
	if m.cardScrollOffset > offset {
		t.Fatalf("cardScrollOffset = %d, want <= %d after growing window", m.cardScrollOffset, offset)
	}
	_ = m.View()

	m, _ = m.Update(tea.WindowSizeMsg{Width: 140, Height: 10})
	view := m.View()
	if !strings.Contains(view, "Todo (20)") {
		t.Fatalf("View() missing column header after resize: %q", view)
	}
	if !strings.Contains(view, "▲") && !strings.Contains(view, "▼") {
		t.Fatalf("View() missing scroll indicators after resize: %q", view)
	}
}

func TestCardScrollOffset_EmptyColumn(t *testing.T) {
	t.Parallel()

	m := New(&github.MockClient{}, github.Project{ID: "PVT_1", Title: "Roadmap"})
	m, _ = m.Update(tea.WindowSizeMsg{Width: 140, Height: 24})
	m, _ = m.Update(boardLoadedMsg{items: []github.ProjectItem{}, fields: testBoardFields()})

	for _, key := range []rune{'j', 'k', 'g', 'G'} {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{key}})
	}

	if m.cardScrollOffset != 0 {
		t.Fatalf("cardScrollOffset = %d, want 0", m.cardScrollOffset)
	}
}

func TestCardScrollOffset_SingleItem(t *testing.T) {
	t.Parallel()

	m := New(&github.MockClient{}, github.Project{ID: "PVT_1", Title: "Roadmap"})
	m, _ = m.Update(tea.WindowSizeMsg{Width: 140, Height: 24})
	m, _ = m.Update(boardLoadedMsg{items: testVerticalScrollItems(1), fields: testVerticalScrollFields()})

	for _, key := range []rune{'j', 'k', 'g', 'G'} {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{key}})
	}

	if m.cardScrollOffset != 0 {
		t.Fatalf("cardScrollOffset = %d, want 0", m.cardScrollOffset)
	}
	if m.activeCard != 0 {
		t.Fatalf("activeCard = %d, want 0", m.activeCard)
	}

	view := m.View()
	if strings.Contains(view, "▲") {
		t.Fatalf("View() unexpectedly shows top scroll indicator: %q", view)
	}
	if strings.Contains(view, "▼") {
		t.Fatalf("View() unexpectedly shows bottom scroll indicator: %q", view)
	}
}

func TestCardScrollOffset_AfterMoveCard(t *testing.T) {
	t.Parallel()

	m := New(&github.MockClient{}, github.Project{ID: "PVT_1", Title: "Roadmap"})
	m, _ = m.Update(tea.WindowSizeMsg{Width: 140, Height: 24})
	m, _ = m.Update(boardLoadedMsg{items: testVerticalScrollItems(5), fields: testVerticalScrollFields()})

	for range 2 {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	}
	if m.activeCard != 2 {
		t.Fatalf("activeCard = %d, want 2 before move result", m.activeCard)
	}

	m, _ = m.Update(moveResultMsg{item: testVerticalScrollItems(5)[2], fromCol: 0, toCol: 1})

	if m.cardScrollOffset < 0 {
		t.Fatalf("cardScrollOffset = %d, want >= 0", m.cardScrollOffset)
	}
	if m.cardScrollOffset > m.activeCard {
		t.Fatalf("cardScrollOffset = %d, want <= activeCard %d", m.cardScrollOffset, m.activeCard)
	}
}

func TestCardScrollOffset_AfterFilter(t *testing.T) {
	t.Parallel()

	m := New(&github.MockClient{}, github.Project{ID: "PVT_1", Title: "Roadmap"})
	m, _ = m.Update(tea.WindowSizeMsg{Width: 140, Height: 24})
	m, _ = m.Update(boardLoadedMsg{items: testVerticalScrollItems(15), fields: testVerticalScrollFields()})

	for range 10 {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	}
	if m.cardScrollOffset <= 0 {
		t.Fatalf("cardScrollOffset before filter = %d, want > 0", m.cardScrollOffset)
	}

	m.searchQuery = "Card-1"
	m.applyFilter()
	m.clampActiveCard()
	m.clampCardScrollOffset()

	if m.cardScrollOffset > m.activeCard {
		t.Fatalf("cardScrollOffset = %d, want <= activeCard %d", m.cardScrollOffset, m.activeCard)
	}

	view := m.View()
	activeTitle := m.columns[m.activeCol].items[m.activeCard].Title
	if !strings.Contains(view, activeTitle) {
		t.Fatalf("View() missing active card title %q: %q", activeTitle, view)
	}
}

func TestSettingsKeyOpensSettingsOverlay(t *testing.T) {
	t.Parallel()

	m := New(&github.MockClient{}, github.Project{ID: "PVT_1", Title: "Roadmap"})
	m, _ = m.Update(tea.WindowSizeMsg{Width: 140, Height: 24})
	m, _ = m.Update(boardLoadedMsg{items: testBoardItems(), fields: testBoardFields()})

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})

	if !m.showSettings {
		t.Fatalf("showSettings = %t, want true", m.showSettings)
	}
}

func TestSettingsUpdateMsgUpdatesOwner(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmp, "config"))
	home := filepath.Join(tmp, "home")
	if err := os.MkdirAll(home, 0755); err != nil {
		t.Fatalf("MkdirAll(home): %v", err)
	}
	t.Setenv("HOME", home)

	m := New(&github.MockClient{}, github.Project{ID: "PVT_1", Title: "Roadmap"})
	m, _ = m.Update(tea.WindowSizeMsg{Width: 140, Height: 24})
	m, _ = m.Update(boardLoadedMsg{items: testBoardItems(), fields: testBoardFields()})

	m, _ = m.Update(components.SettingsUpdateMsg{Field: "DefaultOwner", Value: "neworg"})

	if m.config.DefaultOwner != "neworg" {
		t.Fatalf("config.DefaultOwner = %q, want %q", m.config.DefaultOwner, "neworg")
	}
}

func TestSettingsUpdateMsgUpdatesView(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmp, "config"))
	home := filepath.Join(tmp, "home")
	if err := os.MkdirAll(home, 0755); err != nil {
		t.Fatalf("MkdirAll(home): %v", err)
	}
	t.Setenv("HOME", home)

	m := New(&github.MockClient{}, github.Project{ID: "PVT_1", Title: "Roadmap"})
	m, _ = m.Update(tea.WindowSizeMsg{Width: 140, Height: 24})
	m, _ = m.Update(boardLoadedMsg{items: testBoardItems(), fields: testBoardFields()})

	m, _ = m.Update(components.SettingsUpdateMsg{Field: "DefaultView", Value: "sprint"})

	if m.config.DefaultView != "sprint" {
		t.Fatalf("config.DefaultView = %q, want %q", m.config.DefaultView, "sprint")
	}
}

func TestSettingsToggleShowLabelsUpdatesCheckboxInView(t *testing.T) {
	t.Parallel()

	m := New(&github.MockClient{}, github.Project{ID: "PVT_1", Title: "Roadmap"})
	m.showLabels = true
	m.config.ShowLabels = true
	m.LoadItemsForTest(testBoardItems(), testBoardFields())
	m, _ = m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	var cmd tea.Cmd
	m, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		m, _ = m.Update(cmd())
	}

	if got := m.settingsModel.View(); !strings.Contains(got, "[ ]") {
		t.Fatalf("settingsModel.View() = %q, want unchecked checkbox", got)
	}

	if m.showLabels {
		t.Fatalf("showLabels = %t, want false", m.showLabels)
	}
}

func TestSettingsToggleShowClosedUpdatesCheckboxInView(t *testing.T) {
	t.Parallel()

	m := New(&github.MockClient{}, github.Project{ID: "PVT_1", Title: "Roadmap"})
	m.showClosedItems = false
	m.config.ShowClosedItems = false
	m.LoadItemsForTest(testBoardItems(), testBoardFields())
	m, _ = m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	var cmd tea.Cmd
	m, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		m, _ = m.Update(cmd())
	}

	if got := m.settingsModel.View(); !strings.Contains(got, "[✓]") {
		t.Fatalf("settingsModel.View() = %q, want checked checkbox", got)
	}

	if !m.showClosedItems {
		t.Fatalf("showClosedItems = %t, want true", m.showClosedItems)
	}
}

func TestSettingsToggleRoundTripPreservesState(t *testing.T) {
	t.Parallel()

	m := New(&github.MockClient{}, github.Project{ID: "PVT_1", Title: "Roadmap"})
	m.showLabels = true
	m.config.ShowLabels = true
	m.LoadItemsForTest(testBoardItems(), testBoardFields())
	m, _ = m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	var cmd tea.Cmd
	m, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		m, _ = m.Update(cmd())
	}
	m, _ = m.Update(components.SettingsToggleMsg{Field: "ShowLabels", Value: false})

	if m.showLabels {
		t.Fatalf("showLabels = %t, want false", m.showLabels)
	}

	if got := m.settingsModel.View(); !strings.Contains(got, "[ ]") {
		t.Fatalf("settingsModel.View() = %q, want unchecked checkbox", got)
	}
}

func TestClosedItemsFilteredWhenShowClosedItemsFalse(t *testing.T) {
	t.Parallel()

	fields := []github.ProjectField{{
		ID:       "status-field",
		Name:     "Status",
		DataType: "SINGLE_SELECT",
		Options: []github.FieldOption{
			{ID: "todo", Name: "Todo"},
		},
	}}
	items := []github.ProjectItem{
		{ID: "open", Title: "Open issue", Type: "Issue", StatusID: "todo", Content: &github.Issue{Title: "Open issue", Number: 1, State: "OPEN"}},
		{ID: "closed", Title: "Closed issue", Type: "Issue", StatusID: "todo", Content: &github.Issue{Title: "Closed issue", Number: 2, State: "CLOSED"}},
	}

	m := New(&github.MockClient{}, github.Project{ID: "PVT_1", Title: "Roadmap"})
	m, _ = m.Update(tea.WindowSizeMsg{Width: 140, Height: 24})
	m, _ = m.Update(boardLoadedMsg{items: items, fields: fields})
	m.showClosedItems = false
	m.applyFilter()

	if len(m.columns) != 1 {
		t.Fatalf("len(columns) = %d, want 1", len(m.columns))
	}
	if len(m.columns[0].items) != 1 {
		t.Fatalf("len(columns[0].items) = %d, want 1", len(m.columns[0].items))
	}
	for _, item := range m.columns[0].items {
		if item.ID == "closed" {
			t.Fatalf("applyFilter() unexpectedly kept closed item: %#v", item)
		}
	}
}

func TestSearchInputStyle(t *testing.T) {
	t.Parallel()

	m := New(&github.MockClient{}, github.Project{ID: "PVT_1", Title: "Roadmap"})

	textStyleRender := m.searchInput.TextStyle.Render("test")
	promptStyleRender := m.searchInput.PromptStyle.Render("test")

	if textStyleRender == "" {
		t.Fatalf("TextStyle render result is empty")
	}

	if promptStyleRender == "" {
		t.Fatalf("PromptStyle render result is empty")
	}
}

func TestEmptyColumnsPreservedAfterFilter(t *testing.T) {
	t.Parallel()

	// Create items only in "Todo" column
	items := []github.ProjectItem{
		{ID: "1", Title: "Task in todo", Type: "Issue", StatusID: "todo", Content: &github.Issue{Title: "Task in todo", Number: 1}},
		{ID: "2", Title: "Another todo item", Type: "Issue", StatusID: "todo", Content: &github.Issue{Title: "Another todo item", Number: 2}},
	}

	m := New(&github.MockClient{}, github.Project{ID: "PVT_1", Title: "Roadmap"})
	m, _ = m.Update(tea.WindowSizeMsg{Width: 140, Height: 24})
	m, _ = m.Update(boardLoadedMsg{items: items, fields: testBoardFields()})

	// Verify all 3 columns exist before filter
	if len(m.columns) != 3 {
		t.Fatalf("len(columns) before filter = %d, want 3", len(m.columns))
	}

	// Apply search filter that matches nothing
	m.searchQuery = "xyznonexistent"
	m.applyFilter()

	// Assert all 3 columns are still present despite having no matching items
	if len(m.columns) != 3 {
		t.Fatalf("len(m.columns) = %d, want 3 (all columns preserved)", len(m.columns))
	}

	// Verify column names are correct
	columnNames := make(map[string]bool)
	for _, col := range m.columns {
		columnNames[col.name] = true
	}
	for _, want := range []string{"Todo", "In Progress", "Done"} {
		if !columnNames[want] {
			t.Fatalf("missing expected column %q in %#v", want, columnNames)
		}
	}

	// Verify all columns are empty (no items matched the filter)
	for i, col := range m.columns {
		if len(col.items) != 0 {
			t.Fatalf("columns[%d] (%q) has %d items, want 0 (filter matched nothing)", i, col.name, len(col.items))
		}
	}
}

func TestArrowKeyNavigation_Right(t *testing.T) {
	t.Parallel()

	m := New(&github.MockClient{}, github.Project{ID: "PVT_1", Title: "Roadmap"})
	m, _ = m.Update(tea.WindowSizeMsg{Width: 140, Height: 24})
	m, _ = m.Update(boardLoadedMsg{items: testBoardItems(), fields: testBoardFields()})

	initialCol := m.activeCol

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRight})

	if m.activeCol != initialCol+1 {
		t.Fatalf("activeCol after right arrow = %d, want %d", m.activeCol, initialCol+1)
	}
}

func TestArrowKeyNavigation_Left(t *testing.T) {
	t.Parallel()

	m := New(&github.MockClient{}, github.Project{ID: "PVT_1", Title: "Roadmap"})
	m, _ = m.Update(tea.WindowSizeMsg{Width: 140, Height: 24})
	m, _ = m.Update(boardLoadedMsg{items: testBoardItems(), fields: testBoardFields()})

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	currentCol := m.activeCol

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyLeft})

	if m.activeCol != currentCol-1 {
		t.Fatalf("activeCol after left arrow = %d, want %d", m.activeCol, currentCol-1)
	}
}

func TestArrowKeyNavigation_Down(t *testing.T) {
	t.Parallel()

	m := New(&github.MockClient{}, github.Project{ID: "PVT_1", Title: "Roadmap"})
	m, _ = m.Update(tea.WindowSizeMsg{Width: 140, Height: 24})
	m, _ = m.Update(boardLoadedMsg{items: testVerticalScrollItems(10), fields: testVerticalScrollFields()})

	initialCard := m.activeCard

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})

	if m.activeCard != initialCard+1 {
		t.Fatalf("activeCard after down arrow = %d, want %d", m.activeCard, initialCard+1)
	}
}

func TestArrowKeyNavigation_Up(t *testing.T) {
	t.Parallel()

	m := New(&github.MockClient{}, github.Project{ID: "PVT_1", Title: "Roadmap"})
	m, _ = m.Update(tea.WindowSizeMsg{Width: 140, Height: 24})
	m, _ = m.Update(boardLoadedMsg{items: testVerticalScrollItems(10), fields: testVerticalScrollFields()})

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	currentCard := m.activeCard

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})

	if m.activeCard != currentCard-1 {
		t.Fatalf("activeCard after up arrow = %d, want %d", m.activeCard, currentCard-1)
	}
}

func TestArrowKeyNavigation_ArrowsEquivalentToHJKL(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		arrowKey   tea.KeyType
		hjklKey    rune
		itemCount  int
		iterations int
	}{
		{
			name:       "right arrow equivalent to l",
			arrowKey:   tea.KeyRight,
			hjklKey:    'l',
			itemCount:  9,
			iterations: 3,
		},
		{
			name:       "down arrow equivalent to j",
			arrowKey:   tea.KeyDown,
			hjklKey:    'j',
			itemCount:  15,
			iterations: 5,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			m1 := New(&github.MockClient{}, github.Project{ID: "PVT_1", Title: "Roadmap"})
			m1, _ = m1.Update(tea.WindowSizeMsg{Width: 140, Height: 24})
			if tc.hjklKey == 'l' || tc.hjklKey == 'r' {
				m1, _ = m1.Update(boardLoadedMsg{items: testScrollItems(tc.itemCount), fields: testScrollFields(tc.itemCount)})
			} else {
				m1, _ = m1.Update(boardLoadedMsg{items: testVerticalScrollItems(tc.itemCount), fields: testVerticalScrollFields()})
			}

			for range tc.iterations {
				m1, _ = m1.Update(tea.KeyMsg{Type: tc.arrowKey})
			}

			m2 := New(&github.MockClient{}, github.Project{ID: "PVT_1", Title: "Roadmap"})
			m2, _ = m2.Update(tea.WindowSizeMsg{Width: 140, Height: 24})
			if tc.hjklKey == 'l' || tc.hjklKey == 'r' {
				m2, _ = m2.Update(boardLoadedMsg{items: testScrollItems(tc.itemCount), fields: testScrollFields(tc.itemCount)})
			} else {
				m2, _ = m2.Update(boardLoadedMsg{items: testVerticalScrollItems(tc.itemCount), fields: testVerticalScrollFields()})
			}

			for range tc.iterations {
				m2, _ = m2.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{tc.hjklKey}})
			}

			if tc.hjklKey == 'l' || tc.hjklKey == 'r' {
				if m1.activeCol != m2.activeCol {
					t.Fatalf("arrow key activeCol %d != hjkl activeCol %d", m1.activeCol, m2.activeCol)
				}
			} else {
				if m1.activeCard != m2.activeCard {
					t.Fatalf("arrow key activeCard %d != hjkl activeCard %d", m1.activeCard, m2.activeCard)
				}
			}
		})
	}
}

func TestSettingsUpdateMsgPersistsConfig(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmp, "config"))
	home := filepath.Join(tmp, "home")
	if err := os.MkdirAll(home, 0755); err != nil {
		t.Fatalf("MkdirAll(home): %v", err)
	}
	t.Setenv("HOME", home)

	m := New(&github.MockClient{}, github.Project{ID: "PVT_1", Title: "Roadmap"})
	m, _ = m.Update(tea.WindowSizeMsg{Width: 140, Height: 24})
	m, _ = m.Update(boardLoadedMsg{items: testBoardItems(), fields: testBoardFields()})

	// Fire SettingsUpdateMsg for DefaultOwner
	m, _ = m.Update(components.SettingsUpdateMsg{Field: "DefaultOwner", Value: "persistedorg"})

	// Load config from disk and verify DefaultOwner was persisted
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("config.Load(): %v", err)
	}
	if cfg.DefaultOwner != "persistedorg" {
		t.Fatalf("cfg.DefaultOwner = %q, want %q", cfg.DefaultOwner, "persistedorg")
	}

	// Fire SettingsUpdateMsg for DefaultView
	_, _ = m.Update(components.SettingsUpdateMsg{Field: "DefaultView", Value: "myview"})

	// Load config again and verify DefaultView was persisted
	cfg, err = config.Load()
	if err != nil {
		t.Fatalf("config.Load(): %v", err)
	}
	if cfg.DefaultView != "myview" {
		t.Fatalf("cfg.DefaultView = %q, want %q", cfg.DefaultView, "myview")
	}
}

func TestBoardFooterContainsSettingsHint(t *testing.T) {
	t.Parallel()

	m := New(&github.MockClient{}, github.Project{ID: "PVT_1", Title: "Roadmap"})
	m, _ = m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m, _ = m.Update(boardLoadedMsg{items: testBoardItems(), fields: testBoardFields()})

	view := m.View()
	if !strings.Contains(view, "s Settings") {
		t.Errorf("expected footer to contain 's Settings', got:\n%s", view)
	}
}

func TestApplyViewFilter_RepoAndLabelFilters(t *testing.T) {
	t.Parallel()

	// Set up test data: items from multiple repos, various labels, PR and DraftIssue
	fields := testBoardFields()
	items := []github.ProjectItem{
		// Issue with safe label
		{ID: "1", Title: "Issue1", Type: "Issue", StatusID: "todo", RepoOwner: "org", RepoName: "safe",
			Content: &github.Issue{Title: "Issue1", Number: 1, Labels: []github.Label{{Name: "bug"}}}},
		// Issue with excluded label "marketing"
		{ID: "2", Title: "Issue2", Type: "Issue", StatusID: "doing", RepoOwner: "org", RepoName: "safe",
			Content: &github.Issue{Title: "Issue2", Number: 2, Labels: []github.Label{{Name: "marketing"}}}},
		// Issue with excluded label "design"
		{ID: "3", Title: "Issue3", Type: "Issue", StatusID: "doing", RepoOwner: "org", RepoName: "safe",
			Content: &github.Issue{Title: "Issue3", Number: 3, Labels: []github.Label{{Name: "design"}}}},
		// Issue in excluded repo
		{ID: "4", Title: "Issue4", Type: "Issue", StatusID: "done", RepoOwner: "org", RepoName: "excluded",
			Content: &github.Issue{Title: "Issue4", Number: 4, Labels: []github.Label{{Name: "bug"}}}},
		// PR (should be excluded when any label filter is active)
		{ID: "5", Title: "PR1", Type: "PullRequest", StatusID: "done", RepoOwner: "org", RepoName: "safe",
			Content: &github.PullRequest{Title: "PR1", Number: 55}},
		// DraftIssue (should always pass through)
		{ID: "6", Title: "Draft1", Type: "DraftIssue", StatusID: "todo", Content: nil},
	}

	m := New(&github.MockClient{}, github.Project{ID: "PVT_1", Title: "Roadmap"})
	m, _ = m.Update(tea.WindowSizeMsg{Width: 140, Height: 24})
	m, _ = m.Update(boardLoadedMsg{items: items, fields: fields})

	// Set active view with status, repo, and label filters
	m.activeView = &github.ProjectView{Filter: "-status:Backlog -repo:org/excluded -label:marketing -label:design"}
	m.applyViewFilter()

	// Assert: "Backlog" column is removed (if it existed, but with testBoardFields it doesn't, so just verify no Backlog)
	columnNames := make(map[string]bool)
	for _, col := range m.viewColumns {
		columnNames[col.name] = true
	}
	if columnNames["Backlog"] {
		t.Errorf("applyViewFilter() unexpectedly kept 'Backlog' column")
	}

	// Flatten all items from remaining columns
	var allItems []github.ProjectItem
	for _, col := range m.viewColumns {
		allItems = append(allItems, col.items...)
	}

	// Assert: Items from excluded repo are not present
	for _, item := range allItems {
		if item.RepoOwner == "org" && item.RepoName == "excluded" {
			t.Errorf("applyViewFilter() unexpectedly kept item from excluded repo: %#v", item)
		}
	}

	// Assert: Items with excluded labels are not present (except DraftIssue)
	for _, item := range allItems {
		if item.Type == "Issue" && item.Content != nil {
			if issue, ok := item.Content.(*github.Issue); ok {
				for _, label := range issue.Labels {
					if strings.EqualFold(label.Name, "marketing") || strings.EqualFold(label.Name, "design") {
						t.Errorf("applyViewFilter() unexpectedly kept item with excluded label: %#v", item)
					}
				}
			}
		}
	}

	// Assert: PRs are excluded when label filter is active
	for _, item := range allItems {
		if item.Type == "PullRequest" {
			t.Errorf("applyViewFilter() unexpectedly kept PR when label filter is active: %#v", item)
		}
	}

	// Assert: DraftIssue is preserved
	hasDraft := false
	for _, item := range allItems {
		if item.Type == "DraftIssue" && item.Title == "Draft1" {
			hasDraft = true
			break
		}
	}
	if !hasDraft {
		t.Errorf("applyViewFilter() unexpectedly removed DraftIssue")
	}

	// Assert: Issue1 (with bug label in safe repo) is preserved
	hasIssue1 := false
	for _, item := range allItems {
		if item.ID == "1" {
			hasIssue1 = true
			break
		}
	}
	if !hasIssue1 {
		t.Errorf("applyViewFilter() unexpectedly removed Issue1 (should pass filters)")
	}
}

func TestApplyViewFilter_StatusOnlyRegression(t *testing.T) {
	t.Parallel()

	fields := testBoardFields()
	items := []github.ProjectItem{
		{ID: "1", Title: "Item1", Type: "Issue", StatusID: "todo",
			Content: &github.Issue{Title: "Item1", Number: 1}},
		{ID: "2", Title: "Item2", Type: "Issue", StatusID: "doing",
			Content: &github.Issue{Title: "Item2", Number: 2}},
		{ID: "3", Title: "Item3", Type: "Issue", StatusID: "done",
			Content: &github.Issue{Title: "Item3", Number: 3}},
	}

	m := New(&github.MockClient{}, github.Project{ID: "PVT_1", Title: "Roadmap"})
	m, _ = m.Update(tea.WindowSizeMsg{Width: 140, Height: 24})
	m, _ = m.Update(boardLoadedMsg{items: items, fields: fields})

	// Store original column/item counts
	origColCount := len(m.allColumns)
	origItemCount := 0
	for _, col := range m.allColumns {
		origItemCount += len(col.items)
	}

	// Set active view with status-only filter (no repo/label)
	m.activeView = &github.ProjectView{Filter: "-status:Backlog"}
	m.applyViewFilter()

	// Assert: no columns were removed (Backlog doesn't exist in testBoardFields)
	if len(m.viewColumns) != origColCount {
		t.Errorf("applyViewFilter() changed column count from %d to %d for status-only filter",
			origColCount, len(m.viewColumns))
	}

	// Assert: all items in remaining columns are preserved
	viewItemCount := 0
	for _, col := range m.viewColumns {
		viewItemCount += len(col.items)
	}
	if viewItemCount != origItemCount {
		t.Errorf("applyViewFilter() changed item count from %d to %d for status-only filter",
			origItemCount, viewItemCount)
	}
}

func TestApplyViewFilter_NoFilter(t *testing.T) {
	t.Parallel()

	fields := testBoardFields()
	items := []github.ProjectItem{
		{ID: "1", Title: "Item1", Type: "Issue", StatusID: "todo",
			Content: &github.Issue{Title: "Item1", Number: 1}},
		{ID: "2", Title: "Item2", Type: "Issue", StatusID: "doing",
			Content: &github.Issue{Title: "Item2", Number: 2}},
		{ID: "3", Title: "Item3", Type: "Issue", StatusID: "done",
			Content: &github.Issue{Title: "Item3", Number: 3}},
	}

	m := New(&github.MockClient{}, github.Project{ID: "PVT_1", Title: "Roadmap"})
	m, _ = m.Update(tea.WindowSizeMsg{Width: 140, Height: 24})
	m, _ = m.Update(boardLoadedMsg{items: items, fields: fields})

	// Store original counts
	origColCount := len(m.allColumns)
	origItemCount := 0
	for _, col := range m.allColumns {
		origItemCount += len(col.items)
	}

	// Set activeView to nil (no filter at all)
	m.activeView = nil
	m.applyViewFilter()

	// Assert: all columns preserved
	if len(m.viewColumns) != origColCount {
		t.Errorf("applyViewFilter() with nil activeView: expected %d columns, got %d",
			origColCount, len(m.viewColumns))
	}

	// Assert: all items preserved
	viewItemCount := 0
	for _, col := range m.viewColumns {
		viewItemCount += len(col.items)
	}
	if viewItemCount != origItemCount {
		t.Errorf("applyViewFilter() with nil activeView: expected %d items, got %d",
			origItemCount, viewItemCount)
	}
}
