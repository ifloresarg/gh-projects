package help

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Model represents the help overlay.
type Model struct {
	width   int
	height  int
	visible bool
}

// New creates a new help model with given dimensions.
func New(width, height int) Model {
	return Model{
		width:   width,
		height:  height,
		visible: false,
	}
}

// Init returns no initial command.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles messages. Any key press sets visible to false.
func (m *Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return *m, nil
	case tea.KeyMsg:
		// Any key dismisses the help overlay
		m.visible = false
		return *m, nil
	}
	return *m, nil
}

// View renders the help overlay as a dark box with keybindings.
func (m Model) View() string {
	if !m.visible {
		return ""
	}

	// Help content organized by sections
	sections := []struct {
		title string
		items []string
	}{
		{
			title: "Global",
			items: []string{
				"  q         Quit",
				"  ?         Help",
				"  R         Refresh",
				"  Esc       Back",
			},
		},
		{
			title: "Board",
			items: []string{
				"  h / l     Move between columns",
				"  j / k     Move between cards",
				"  Enter     Open issue detail",
				"  g / G     First / last card",
				"  < / >     Move card left / right",
				"  /         Search / filter",
				"  s         Settings",
				"  v         Switch view",
				"  p         Switch project",
			},
		},
		{
			title: "Picker",
			items: []string{
				"  j / k     Move between projects",
				"  Enter     Select project",
				"  /         Search",
				"  o         Switch owner",
			},
		},
		{
			title: "Detail",
			items: []string{
				"  c         Comments",
				"  a         Assign / Unassign",
				"  L         Add / Remove Labels",
				"  e         Edit body",
				"  x         Close issue",
				"  X         Reopen issue",
				"  p         Link PR to project",
				"  Esc       Back to board",
			},
		},
		{
			title: "Comments",
			items: []string{
				"  Ctrl+S    Submit comment",
				"  Esc       Back to detail",
			},
		},
	}

	// Build the help box content
	boxContent := []string{}

	for _, section := range sections {
		boxContent = append(boxContent, lipgloss.NewStyle().Bold(true).Render(section.title))
		boxContent = append(boxContent, section.items...)
		boxContent = append(boxContent, "")
	}

	boxContent = append(boxContent, lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render("Press any key to close"))

	// Style the box
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("8")).
		Background(lipgloss.Color("235")).
		Foreground(lipgloss.Color("7")).
		Padding(1, 2)

	helpBox := boxStyle.Render(strings.Join(boxContent, "\n"))

	// Place the box in the center of the screen
	if m.width <= 0 || m.height <= 0 {
		return helpBox
	}

	// Place the help box in the center of a dark background
	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		helpBox,
	)
}

// Show sets the help overlay to visible.
func (m *Model) Show() {
	m.visible = true
}

// Hide sets the help overlay to hidden.
func (m *Model) Hide() {
	m.visible = false
}

// IsVisible returns whether the help overlay is currently visible.
func (m Model) IsVisible() bool {
	return m.visible
}
