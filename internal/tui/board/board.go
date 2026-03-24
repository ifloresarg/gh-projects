package board

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ifloresarg/gh-projects/internal/config"
	"github.com/ifloresarg/gh-projects/internal/github"
	"github.com/ifloresarg/gh-projects/internal/tui/components"
)

const columnWidth = 35

type boardLoadedMsg struct {
	items  []github.ProjectItem
	fields []github.ProjectField
	err    error
}

type moveResultMsg struct {
	item    github.ProjectItem
	fromCol int
	toCol   int
	err     error
}

type SwitchViewMsg struct{}

type column struct {
	name   string
	itemID string
	items  []github.ProjectItem
}

type Model struct {
	client           github.GitHubClient
	project          github.Project
	columns          []column
	fields           []github.ProjectField
	activeCol        int
	activeCard       int
	scrollOffset     int
	cardScrollOffset int
	selected         bool
	searchMode       bool
	searchQuery      string
	searchInput      textinput.Model
	allColumns       []column
	activeView       *github.ProjectView
	viewColumns      []column
	spinner          spinner.Model
	loading          bool
	moving           bool
	moveErr          string
	notif            components.Notification
	err              error
	width            int
	height           int
	config           config.Config
	showSettings     bool
	settingsModel    components.SettingsModel
	showLabels       bool
	showClosedItems  bool
}

func New(client github.GitHubClient, project github.Project) Model {
	cfg, err := config.Load()
	if err != nil {
		cfg = config.DefaultConfig()
	}

	s := spinner.New()
	s.Spinner = spinner.Dot
	si := textinput.New()
	si.Placeholder = "type to filter..."
	si.Width = 40
	si.TextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("15"))
	si.PromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("14"))

	return Model{
		client:          client,
		project:         project,
		searchInput:     si,
		spinner:         s,
		loading:         true,
		config:          cfg,
		showLabels:      cfg.ShowLabels,
		showClosedItems: cfg.ShowClosedItems,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, fetchBoard(m.client, m.project.ID))
}

func fetchBoard(client github.GitHubClient, projectID string) tea.Cmd {
	return func() tea.Msg {
		items, err := client.GetProjectItems(projectID)
		if err != nil {
			return boardLoadedMsg{err: err}
		}

		fields, err := client.GetProjectFields(projectID)
		return boardLoadedMsg{items: items, fields: fields, err: err}
	}
}

func buildColumns(items []github.ProjectItem, fields []github.ProjectField) []column {
	var statusField *github.ProjectField
	for i := range fields {
		if fields[i].Name == "Status" && fields[i].DataType == "SINGLE_SELECT" {
			statusField = &fields[i]
			break
		}
	}

	cols := make([]column, 0)
	if statusField != nil {
		for _, opt := range statusField.Options {
			cols = append(cols, column{name: opt.Name, itemID: opt.ID})
		}
	}

	for _, item := range items {
		for i := range cols {
			if cols[i].itemID == item.StatusID {
				cols[i].items = append(cols[i].items, item)
				break
			}
		}
	}

	return cols
}

func isClosedItem(item github.ProjectItem) bool {
	switch c := item.Content.(type) {
	case *github.Issue:
		return c.State == "CLOSED"
	case *github.PullRequest:
		return c.State == "CLOSED"
	default:
		return false
	}
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case components.SettingsToggleMsg:
		switch msg.Field {
		case "ShowLabels":
			m.showLabels = msg.Value
			m.config.ShowLabels = msg.Value
		case "ShowClosedItems":
			m.showClosedItems = msg.Value
			m.config.ShowClosedItems = msg.Value
		}

		cfg, err := config.Load()
		if err != nil {
			cfg = config.DefaultConfig()
		}
		cfg.ShowLabels = m.showLabels
		cfg.ShowClosedItems = m.showClosedItems
		if err := config.Save(cfg); err != nil {
			m.notif, _ = components.Show(fmt.Sprintf("Save failed: %v", err), components.KindError)
		} else {
			m.config = cfg
		}
		m.applyFilter()
		m.clampActiveCard()
		m.clampCardScrollOffset()
		return m, nil
	case components.SettingsUpdateMsg:
		cfg, err := config.Load()
		if err != nil {
			cfg = config.DefaultConfig()
		}

		switch msg.Field {
		case "DefaultOwner":
			m.config.DefaultOwner = msg.Value
			cfg.DefaultOwner = msg.Value
		case "DefaultProject":
			trimmed := strings.TrimSpace(msg.Value)
			project := 0
			if trimmed != "" && trimmed != "0" {
				parsed, parseErr := strconv.Atoi(trimmed)
				if parseErr != nil {
					m.notif, _ = components.Show(fmt.Sprintf("Save failed: %v", parseErr), components.KindError)
					return m, nil
				}
				project = parsed
			}
			m.config.DefaultProject = project
			cfg.DefaultProject = project
		case "DefaultView":
			m.config.DefaultView = msg.Value
			cfg.DefaultView = msg.Value
		}

		if err := config.Save(cfg); err != nil {
			m.notif, _ = components.Show(fmt.Sprintf("Save failed: %v", err), components.KindError)
			return m, nil
		}

		m.config = cfg
		return m, nil
	case components.SettingsCloseMsg:
		m.showSettings = false
		return m, nil
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if m.showSettings {
			m.settingsModel.SetSize(msg.Width, msg.Height)
		}
		m.clampScrollOffset()
		m.clampCardScrollOffset()
		return m, nil
	}

	if m.showSettings {
		newModel, cmd := m.settingsModel.Update(msg)
		if sm, ok := newModel.(components.SettingsModel); ok {
			m.settingsModel = sm
		}
		return m, cmd
	}

	switch msg := msg.(type) {
	case boardLoadedMsg:
		m.loading = false
		m.err = msg.err
		if msg.err != nil {
			return m, nil
		}

		m.fields = msg.fields
		m.allColumns = buildColumns(msg.items, msg.fields)
		m.applyViewFilter()
		m.applyFilter()
		m.activeCol = 0
		m.activeCard = 0
		m.clampScrollOffset()
		m.selected = false
		m.clampActiveCard()
		m.clampCardScrollOffset()
		return m, nil
	case moveResultMsg:
		m.moving = false
		var cmd tea.Cmd
		if msg.err != nil {
			m.notif, cmd = components.Show(fmt.Sprintf("Move failed: %v", msg.err), components.KindError)
			return m, cmd
		}

		m.moveErr = ""
		targetColName := ""
		if msg.toCol >= 0 && msg.toCol < len(m.columns) {
			targetColName = m.columns[msg.toCol].name
		}
		m.notif, cmd = components.Show(fmt.Sprintf("✓ Moved to %s", targetColName), components.KindSuccess)

		if msg.fromCol < 0 || msg.fromCol >= len(m.columns) || msg.toCol < 0 || msg.toCol >= len(m.columns) {
			return m, cmd
		}

		allItems := make([]github.ProjectItem, 0)
		for _, col := range m.allColumns {
			for _, it := range col.items {
				if it.ID != msg.item.ID {
					allItems = append(allItems, it)
				}
			}
		}

		moved := msg.item
		moved.StatusID = m.columns[msg.toCol].itemID
		allItems = append(allItems, moved)
		m.allColumns = buildColumns(allItems, m.fields)
		m.applyViewFilter()
		m.applyFilter()
		m.clampScrollOffset()
		m.clampActiveCard()
		m.clampCardScrollOffset()
		return m, cmd
	case spinner.TickMsg:
		if !m.loading && !m.moving {
			return m, nil
		}

		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case components.DismissMsg:
		m.notif, _ = m.notif.Update(msg)
		return m, nil
	case tea.KeyMsg:
		m.moveErr = ""
		m.notif.Hide()
		if m.loading || m.err != nil || len(m.columns) == 0 {
			return m, nil
		}

		if m.searchMode {
			switch msg.String() {
			case "esc":
				m.searchMode = false
				m.searchQuery = ""
				m.searchInput.SetValue("")
				m.searchInput.Blur()
				m.applyFilter()
				m.clampActiveCard()
				m.clampCardScrollOffset()
				return m, nil
			case "enter":
				m.searchMode = false
				m.searchInput.Blur()
				return m, nil
			default:
				var cmd tea.Cmd
				m.searchInput, cmd = m.searchInput.Update(msg)
				m.searchQuery = m.searchInput.Value()
				m.applyFilter()
				m.clampActiveCard()
				m.clampCardScrollOffset()
				return m, cmd
			}
		}

		switch msg.String() {
		case "R":
			m.loading = true
			m.err = nil
			m.moveErr = ""
			m.notif.Hide()
			// Invalidate cache if the client is a CachedClient
			if cc, ok := m.client.(*github.CachedClient); ok {
				cc.InvalidateAll()
			}
			return m, tea.Batch(m.spinner.Tick, fetchBoard(m.client, m.project.ID))
		case "/":
			if len(m.allColumns) == 0 {
				m.allColumns = cloneColumns(m.columns)
			}
			m.searchMode = true
			m.searchInput.SetValue(m.searchQuery)
			cmd := m.searchInput.Focus()
			return m, cmd
		case "s":
			m.showSettings = true
			m.settingsModel = components.NewSettingsModel(
				m.showLabels,
				m.showClosedItems,
				m.config.DefaultOwner,
				m.config.DefaultProject,
				m.config.DefaultView,
			)
			m.settingsModel.SetSize(m.width, m.height)
			return m, nil
		case "h", "left":
			if m.activeCol > 0 {
				m.activeCol--
				m.clampScrollOffset()
			}
			m.selected = false
			m.clampActiveCard()
			m.clampCardScrollOffset()
			return m, nil
		case "l", "right":
			if m.activeCol < len(m.columns)-1 {
				m.activeCol++
				m.clampScrollOffset()
			}
			m.selected = false
			m.clampActiveCard()
			m.clampCardScrollOffset()
			return m, nil
		case "j", "down":
			maxCard := m.currentColumnItemCount() - 1
			if maxCard < 0 {
				m.activeCard = 0
				m.clampCardScrollOffset()
				return m, nil
			}
			if m.activeCard < maxCard {
				m.activeCard++
			}
			m.clampCardScrollOffset()
			return m, nil
		case "k", "up":
			if m.activeCard > 0 {
				m.activeCard--
			}
			m.clampCardScrollOffset()
			return m, nil
		case "g":
			m.activeCard = 0
			m.clampActiveCard()
			m.clampCardScrollOffset()
			return m, nil
		case "G":
			m.activeCard = m.currentColumnItemCount() - 1
			m.clampActiveCard()
			m.clampCardScrollOffset()
			return m, nil
		case "enter":
			m.selected = m.currentColumnItemCount() > 0
			return m, nil
		case "v":
			return m, func() tea.Msg { return SwitchViewMsg{} }
		case "esc":
			m.selected = false
			return m, nil
		case "<":
			if !m.moving {
				cmd := m.moveCard(-1)
				if cmd != nil {
					return m, tea.Batch(m.spinner.Tick, cmd)
				}
			}
			return m, nil
		case ">":
			if !m.moving {
				cmd := m.moveCard(1)
				if cmd != nil {
					return m, tea.Batch(m.spinner.Tick, cmd)
				}
			}
			return m, nil
		}
	}

	return m, nil
}

func (m Model) View() string {
	if m.showSettings {
		return m.settingsModel.View()
	}

	if m.loading {
		msg := fmt.Sprintf("%s Loading board...", m.spinner.View())
		if m.width <= 0 || m.height <= 0 {
			return msg
		}

		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, msg)
	}

	if m.err != nil {
		errMsg := fmt.Sprintf("Error: %v (press q to quit)", m.err)
		if m.width <= 0 || m.height <= 0 {
			return errMsg
		}

		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, errMsg)
	}

	rendered := make([]string, 0, len(m.columns))
	maxVisible := max(m.width/(columnWidth+2), 1)
	start := max(m.scrollOffset, 0)
	end := min(start+maxVisible, len(m.columns))

	for i := start; i < end; i++ {
		rendered = append(rendered, m.renderColumn(m.columns[i], i == m.activeCol))
	}

	if len(rendered) == 0 {
		return ""
	}

	board := lipgloss.JoinHorizontal(lipgloss.Top, rendered...)

	parts := make([]string, 0, 4)
	if m.searchMode {
		parts = append(parts, lipgloss.NewStyle().Foreground(lipgloss.Color("14")).Render("Search: ")+m.searchInput.View())
	}
	parts = append(parts, board)

	var statusLine string
	if m.moving {
		statusLine = fmt.Sprintf("%s Moving card...", m.spinner.View())
	} else if notifView := m.notif.View(); notifView != "" {
		statusLine = notifView
	}

	if statusLine != "" {
		parts = append(parts, statusLine)
	}

	if m.searchMode || m.hasActiveFilter() {
		parts = append(parts, lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render(fmt.Sprintf("Filter: %d of %d items", m.matchCount(), m.totalItemCount())))
	}

	footerHints := m.renderFooterHints()
	if footerHints != "" {
		parts = append(parts, footerHints)
	}

	return strings.Join(parts, "\n")
}

func (m *Model) moveCard(direction int) tea.Cmd {
	if len(m.columns) == 0 || m.activeCol < 0 || m.activeCol >= len(m.columns) {
		return nil
	}

	col := m.columns[m.activeCol]
	if m.activeCard < 0 || m.activeCard >= len(col.items) {
		return nil
	}

	targetCol := m.activeCol + direction
	if targetCol < 0 || targetCol >= len(m.columns) {
		return nil
	}

	if m.columns[targetCol].itemID == "" {
		return nil
	}

	var statusFieldID string
	for _, f := range m.fields {
		if f.Name == "Status" && f.DataType == "SINGLE_SELECT" {
			statusFieldID = f.ID
			break
		}
	}

	if statusFieldID == "" {
		return nil
	}

	item := col.items[m.activeCard]
	fromCol := m.activeCol
	toCol := targetCol
	targetOptionID := m.columns[targetCol].itemID
	client := m.client
	projectID := m.project.ID

	m.moving = true
	return func() tea.Msg {
		err := client.MoveItem(projectID, item.ID, statusFieldID, targetOptionID)
		return moveResultMsg{item: item, fromCol: fromCol, toCol: toCol, err: err}
	}
}

func (m Model) renderColumn(col column, focused bool) string {
	header := truncate(fmt.Sprintf("%s (%d)", col.name, len(col.items)), columnWidth-4)

	headerStyle := lipgloss.NewStyle().Bold(true).Width(columnWidth-2).Padding(0, 1)
	if focused {
		headerStyle = headerStyle.Foreground(lipgloss.Color("12"))
	}

	height := max(m.height-4, 3)
	bodyHeight := max(height-2, 0)

	startIdx := 0
	if focused {
		startIdx = max(0, min(m.cardScrollOffset, len(col.items)))
	}

	showTopIndicator := focused && startIdx > 0
	indicatorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))

	type cardEntry struct {
		rendered string
		idx      int
	}

	renderVisibleCards := func(limit int) []cardEntry {
		if limit <= 0 {
			return nil
		}

		visible := make([]cardEntry, 0)
		used := 0
		for i := startIdx; i < len(col.items); i++ {
			cardFocused := focused && i == m.activeCard
			rendered := components.Card(col.items[i], cardFocused, columnWidth-2, m.showLabels)
			h := lipgloss.Height(rendered)
			if h <= 0 {
				h = 1
			}

			if used+h > limit {
				break
			}

			used += h
			visible = append(visible, cardEntry{rendered: rendered, idx: i})
		}

		return visible
	}

	reservedTop := 0
	if showTopIndicator {
		reservedTop = 1
	}

	visibleCards := renderVisibleCards(max(bodyHeight-reservedTop, 0))
	lastVisibleIdx := startIdx - 1
	if len(visibleCards) > 0 {
		lastVisibleIdx = visibleCards[len(visibleCards)-1].idx
	}
	belowCount := len(col.items) - (lastVisibleIdx + 1)
	showBottomIndicator := focused && belowCount > 0

	if showBottomIndicator {
		visibleCards = renderVisibleCards(max(bodyHeight-reservedTop-1, 0))
		lastVisibleIdx = startIdx - 1
		if len(visibleCards) > 0 {
			lastVisibleIdx = visibleCards[len(visibleCards)-1].idx
		}
		belowCount = len(col.items) - (lastVisibleIdx + 1)
		showBottomIndicator = belowCount > 0
	}

	lines := []string{headerStyle.Render(header), strings.Repeat("─", columnWidth-2)}
	if showTopIndicator {
		lines = append(lines, indicatorStyle.Render(fmt.Sprintf("▲ %d more", startIdx)))
	}
	for _, card := range visibleCards {
		lines = append(lines, card.rendered)
	}
	if showBottomIndicator {
		lines = append(lines, indicatorStyle.Render(fmt.Sprintf("▼ %d more", belowCount)))
	}

	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("8")).
		Width(columnWidth).
		Height(height)

	if focused {
		borderStyle = borderStyle.BorderForeground(lipgloss.Color("12"))
	}

	return borderStyle.Render(strings.Join(lines, "\n"))
}

func (m Model) renderFooterHints() string {
	if m.width <= 0 {
		return ""
	}

	hintStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("236")).
		Foreground(lipgloss.Color("8"))

	hints := "? Help  Enter Detail  < > Move  / Search  s Settings  v Views  q Quit"
	maxVisible := max(m.width/(columnWidth+2), 1)
	scrollHints := make([]string, 0, 2)
	if m.scrollOffset > 0 {
		scrollHints = append(scrollHints, fmt.Sprintf("◀ %d more", m.scrollOffset))
	}
	rightHidden := len(m.columns) - (m.scrollOffset + maxVisible)
	if rightHidden > 0 {
		scrollHints = append(scrollHints, fmt.Sprintf("%d more ▶", rightHidden))
	}
	if len(scrollHints) > 0 {
		hints = strings.Join(scrollHints, " | ") + "  " + hints
	}

	renderedHints := hintStyle.Render(hints)
	hintsWidth := lipgloss.Width(renderedHints)

	if hintsWidth < m.width {
		diff := m.width - hintsWidth
		renderedHints = hintStyle.Render(hints + strings.Repeat(" ", diff))
	} else if hintsWidth > m.width {
		runes := []rune(hints)
		if len(runes) > m.width {
			hints = string(runes[:m.width])
		}
		renderedHints = hintStyle.Render(hints)
	}

	return renderedHints
}

func (m *Model) clampActiveCard() {
	maxCard := m.currentColumnItemCount() - 1
	if maxCard < 0 {
		m.activeCard = 0
		return
	}

	if m.activeCard > maxCard {
		m.activeCard = maxCard
	}
	if m.activeCard < 0 {
		m.activeCard = 0
	}
}

func (m *Model) clampScrollOffset() {
	maxVisible := max(m.width/(columnWidth+2), 1)
	if len(m.columns) == 0 {
		m.scrollOffset = 0
		return
	}

	maxOffset := len(m.columns) - maxVisible
	maxOffset = max(maxOffset, 0)
	if m.scrollOffset > maxOffset {
		m.scrollOffset = maxOffset
	}
	m.scrollOffset = max(m.scrollOffset, 0)

	m.scrollOffset = min(m.scrollOffset, m.activeCol)
	if m.activeCol >= m.scrollOffset+maxVisible {
		m.scrollOffset = m.activeCol - maxVisible + 1
	}
}

func (m *Model) clampCardScrollOffset() {
	if m.activeCol < 0 || m.activeCol >= len(m.columns) {
		m.cardScrollOffset = 0
		return
	}

	items := m.columns[m.activeCol].items
	if len(items) == 0 {
		m.cardScrollOffset = 0
		return
	}

	availableHeight := max(max(m.height-4, 3)-2, 1)
	if availableHeight <= 0 {
		availableHeight = 1
	}

	maxOffset := len(items) - 1
	m.cardScrollOffset = max(0, min(m.cardScrollOffset, maxOffset))

	m.cardScrollOffset = min(m.cardScrollOffset, m.activeCard)

	lastVisibleFrom := func(start int) int {
		totalHeight := 0
		lastVisible := start
		for i := start; i < len(items); i++ {
			cardHeight := lipgloss.Height(components.Card(items[i], false, columnWidth-2, m.showLabels))
			if cardHeight <= 0 {
				cardHeight = 1
			}
			if totalHeight+cardHeight > availableHeight {
				if i == start {
					return start
				}
				break
			}
			totalHeight += cardHeight
			lastVisible = i
		}
		return lastVisible
	}

	lastVisible := lastVisibleFrom(m.cardScrollOffset)
	for m.activeCard > lastVisible && m.cardScrollOffset < m.activeCard {
		m.cardScrollOffset++
		lastVisible = lastVisibleFrom(m.cardScrollOffset)
	}

	m.cardScrollOffset = max(0, min(m.cardScrollOffset, maxOffset))
}

func (m Model) currentColumnItemCount() int {
	if m.activeCol < 0 || m.activeCol >= len(m.columns) {
		return 0
	}

	return len(m.columns[m.activeCol].items)
}

func truncate(s string, width int) string {
	if width <= 0 {
		return ""
	}

	if lipgloss.Width(s) <= width {
		return s
	}

	runes := []rune(s)
	if width <= 3 {
		if len(runes) < width {
			return string(runes)
		}
		return string(runes[:width])
	}

	maxRunes := min(width-3, len(runes))
	return string(runes[:maxRunes]) + "..."
}

func (m Model) SelectedItem() (github.ProjectItem, bool) {
	if !m.selected || len(m.columns) == 0 {
		return github.ProjectItem{}, false
	}

	if m.activeCol < 0 || m.activeCol >= len(m.columns) {
		return github.ProjectItem{}, false
	}

	col := m.columns[m.activeCol]
	if m.activeCard < 0 || m.activeCard >= len(col.items) {
		return github.ProjectItem{}, false
	}

	return col.items[m.activeCard], true
}

func (m Model) IsSelected() bool {
	return m.selected
}

func (m Model) Items() []github.ProjectItem {
	cols := m.allColumns
	if len(cols) == 0 {
		cols = m.columns
	}

	items := make([]github.ProjectItem, 0)
	for _, col := range cols {
		items = append(items, col.items...)
	}

	return items
}

func (m *Model) UpdateItem(item github.ProjectItem) {
	for ci := range m.allColumns {
		for ii := range m.allColumns[ci].items {
			if m.allColumns[ci].items[ii].ID == item.ID {
				m.allColumns[ci].items[ii] = item
			}
		}
	}

	for ci := range m.columns {
		for ii := range m.columns[ci].items {
			if m.columns[ci].items[ii].ID == item.ID {
				m.columns[ci].items[ii] = item
			}
		}
	}
}

func (m Model) IsShowingSettings() bool {
	return m.showSettings
}

func (m *Model) ClearSelection() {
	m.selected = false
}

func (m *Model) LoadItemsForTest(items []github.ProjectItem, fields []github.ProjectField) {
	m.fields = fields
	m.allColumns = buildColumns(items, fields)
	m.viewColumns = cloneColumns(m.allColumns)
	m.applyFilter()
	m.loading = false
	m.err = nil
	m.selected = false
	m.activeCol = 0
	m.activeCard = 0
	m.scrollOffset = 0
	m.cardScrollOffset = 0
	m.clampScrollOffset()
	m.clampActiveCard()
	m.clampCardScrollOffset()
}

func cloneColumns(cols []column) []column {
	cloned := make([]column, len(cols))
	for i, col := range cols {
		cloned[i] = column{
			name:   col.name,
			itemID: col.itemID,
			items:  append([]github.ProjectItem(nil), col.items...),
		}
	}

	return cloned
}

func itemTitle(item github.ProjectItem) string {
	switch c := item.Content.(type) {
	case *github.Issue:
		return c.Title
	case *github.PullRequest:
		return c.Title
	default:
		return item.Title
	}
}

func (m *Model) applyFilter() {
	if len(m.viewColumns) == 0 {
		m.columns = nil
		m.clampScrollOffset()
		return
	}

	query := strings.TrimSpace(m.searchQuery)
	query = strings.ToLower(query)
	filtered := make([]column, len(m.viewColumns))
	for i, col := range m.viewColumns {
		filtered[i] = column{name: col.name, itemID: col.itemID}
		for _, item := range col.items {
			if !m.showClosedItems && isClosedItem(item) {
				continue
			}
			if query == "" || strings.Contains(strings.ToLower(itemTitle(item)), query) {
				filtered[i].items = append(filtered[i].items, item)
			}
		}
	}

	m.columns = filtered
	m.clampScrollOffset()
}

func (m *Model) applyViewFilter() {
	if len(m.allColumns) == 0 {
		m.viewColumns = nil
		return
	}

	if m.activeView == nil || m.activeView.Filter == "" {
		m.viewColumns = cloneColumns(m.allColumns)
		return
	}

	excluded := ParseStatusFilter(m.activeView.Filter)
	m.viewColumns = FilterColumns(m.allColumns, excluded)
}

func (m *Model) SetActiveView(view *github.ProjectView) {
	m.activeView = view
	m.applyViewFilter()
	m.applyFilter()
	m.activeCol = 0
	m.activeCard = 0
	m.scrollOffset = 0
	m.cardScrollOffset = 0
	m.clampScrollOffset()
	m.clampActiveCard()
	m.clampCardScrollOffset()
}

func (m Model) hasActiveFilter() bool {
	if strings.TrimSpace(m.searchQuery) != "" {
		return true
	}
	return m.activeView != nil && m.activeView.Filter != ""
}

func (m Model) totalItemCount() int {
	total := 0
	for _, col := range m.allColumns {
		total += len(col.items)
	}

	return total
}

func (m Model) matchCount() int {
	total := 0
	for _, col := range m.columns {
		total += len(col.items)
	}

	return total
}
