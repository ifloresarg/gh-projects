package components

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type SettingsToggleMsg struct {
	Field string
	Value bool
}

type SettingsCloseMsg struct{}

type SettingsModel struct {
	cursor          int
	showLabels      bool
	showClosedItems bool
	width           int
	height          int
}

func NewSettingsModel(showLabels bool, showClosedItems bool) SettingsModel {
	return SettingsModel{
		cursor:          0,
		showLabels:      showLabels,
		showClosedItems: showClosedItems,
	}
}

func (m SettingsModel) Init() tea.Cmd { return nil }

func (m *SettingsModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

func (m SettingsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "k", "up":
			if m.cursor > 0 {
				m.cursor--
			}
		case "j", "down":
			if m.cursor < 1 {
				m.cursor++
			}
		case "enter", " ":
			switch m.cursor {
			case 0:
				m.showLabels = !m.showLabels
				return m, func() tea.Msg { return SettingsToggleMsg{Field: "ShowLabels", Value: m.showLabels} }
			case 1:
				m.showClosedItems = !m.showClosedItems
				return m, func() tea.Msg { return SettingsToggleMsg{Field: "ShowClosedItems", Value: m.showClosedItems} }
			}
		case "esc":
			return m, func() tea.Msg { return SettingsCloseMsg{} }
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}

	return m, nil
}

func (m SettingsModel) View() string {
	box := m.PopupView()

	if m.width > 0 && m.height > 0 {
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
	}

	return box
}

func (m SettingsModel) PopupView() string {
	toggles := []struct {
		label string
		value bool
	}{
		{"Show Labels", m.showLabels},
		{"Show Closed", m.showClosedItems},
	}

	rows := make([]string, 0, len(toggles)+2)
	rows = append(rows, lipgloss.NewStyle().Bold(true).Render("Settings"))

	for i, toggle := range toggles {
		cursor := "  "
		if m.cursor == i {
			cursor = "▸"
		}

		checkbox := "[ ]"
		if toggle.value {
			checkbox = "[✓]"
		}

		rows = append(rows, fmt.Sprintf("%s %-12s  %s", cursor, toggle.label, checkbox))
	}

	rows = append(rows, "", lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render("esc: close"))

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("8")).
		Background(lipgloss.Color("235")).
		Foreground(lipgloss.Color("7")).
		Padding(1, 2).
		Width(21).
		Render(strings.Join(rows, "\n"))

	return box
}
