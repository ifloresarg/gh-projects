package components

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type SettingsToggleMsg struct {
	Field string
	Value bool
}

type SettingsCloseMsg struct{}

type SettingsUpdateMsg struct {
	Field string
	Value string
}

type SettingsModel struct {
	cursor          int
	showLabels      bool
	showClosedItems bool
	editingField    string
	editInput       textinput.Model
	owner           string
	defaultProject  int
	defaultView     string
	width           int
	height          int
}

func NewSettingsModel(showLabels, showClosedItems bool, owner string, defaultProject int, defaultView string) SettingsModel {
	input := textinput.New()
	input.Width = 30

	return SettingsModel{
		cursor:          0,
		showLabels:      showLabels,
		showClosedItems: showClosedItems,
		editInput:       input,
		owner:           owner,
		defaultProject:  defaultProject,
		defaultView:     defaultView,
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
		if m.editingField != "" {
			switch msg.String() {
			case "enter":
				value := m.editInput.Value()
				switch m.editingField {
				case "DefaultOwner":
					m.owner = value
				case "DefaultProject":
					trimmed := strings.TrimSpace(value)
					if trimmed == "" || trimmed == "0" {
						m.defaultProject = 0
					} else if project, err := strconv.Atoi(trimmed); err == nil {
						m.defaultProject = project
					}
				case "DefaultView":
					m.defaultView = value
				}
				m.editingField = ""
				m.editInput.Blur()
				return m, func() tea.Msg { return SettingsUpdateMsg{Field: fieldName(m.cursor), Value: value} }
			case "esc":
				m.editingField = ""
				m.editInput.Blur()
				return m, nil
			default:
				var cmd tea.Cmd
				m.editInput, cmd = m.editInput.Update(msg)
				return m, cmd
			}
		}

		switch msg.String() {
		case "k", "up":
			if m.cursor > 0 {
				m.cursor--
			}
		case "j", "down":
			if m.cursor < 4 {
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
			case 2, 3, 4:
				m.editingField = fieldName(m.cursor)
				m.editInput.SetValue(m.currentFieldValue())
				cmd := m.editInput.Focus()
				return m, cmd
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
	items := []struct {
		label string
		value string
	}{
		{"Show Labels", checkboxValue(m.showLabels)},
		{"Show Closed", checkboxValue(m.showClosedItems)},
		{"Default Owner", emptyValue(m.owner)},
		{"Default Project", projectValue(m.defaultProject)},
		{"Default View", emptyValue(m.defaultView)},
	}

	rows := make([]string, 0, len(items)+2)
	rows = append(rows, lipgloss.NewStyle().Bold(true).Render("Settings"))

	for i, item := range items {
		cursor := "  "
		if m.cursor == i {
			cursor = "▸"
		}

		value := item.value
		field := fieldName(i)
		if field != "" && m.editingField == field {
			value = m.editInput.View()
		}

		rows = append(rows, fmt.Sprintf("%s %-15s %s", cursor, item.label, value))
	}

	footer := "esc: close"
	if m.editingField != "" {
		footer = "enter: save • esc: cancel"
	}
	rows = append(rows, "", lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render(footer))

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("8")).
		Background(lipgloss.Color("235")).
		Foreground(lipgloss.Color("7")).
		Padding(1, 2).
		Width(52).
		Render(strings.Join(rows, "\n"))

	return box
}

func checkboxValue(value bool) string {
	if value {
		return "[✓]"
	}

	return "[ ]"
}

func emptyValue(value string) string {
	if value == "" {
		return "None"
	}

	return value
}

func projectValue(value int) string {
	if value == 0 {
		return "None"
	}

	return strconv.Itoa(value)
}

func fieldName(cursor int) string {
	switch cursor {
	case 2:
		return "DefaultOwner"
	case 3:
		return "DefaultProject"
	case 4:
		return "DefaultView"
	default:
		return ""
	}
}

func (m SettingsModel) currentFieldValue() string {
	switch m.editingField {
	case "DefaultOwner":
		return m.owner
	case "DefaultProject":
		return strconv.Itoa(m.defaultProject)
	case "DefaultView":
		return m.defaultView
	default:
		return ""
	}
}
