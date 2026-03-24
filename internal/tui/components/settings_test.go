package components

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestSettingsModelRendersOwnerProjectAndViewValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		owner     string
		project   int
		view      string
		fragments []string
	}{
		{
			name:      "renders all new settings fields",
			owner:     "myorg",
			project:   42,
			view:      "sprint",
			fragments: []string{"myorg", "42", "sprint"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			m := NewSettingsModel(true, false, tt.owner, tt.project, tt.view)
			got := m.View()

			for _, fragment := range tt.fragments {
				if !strings.Contains(got, fragment) {
					t.Fatalf("View() missing %q in %q", fragment, got)
				}
			}
		})
	}
}

func TestSettingsEditOwnerFieldEmitsSettingsUpdateMsg(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		initial   string
		typed     string
		expected  string
		fieldName string
	}{
		{
			name:      "confirm owner edit emits update",
			initial:   "oldorg",
			typed:     "neworg",
			expected:  "neworg",
			fieldName: "DefaultOwner",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			m := NewSettingsModel(true, false, tt.initial, 0, "")

			updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
			m = updated.(SettingsModel)
			updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
			m = updated.(SettingsModel)

			updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
			m = updated.(SettingsModel)

			if m.editingField != tt.fieldName {
				t.Fatalf("editingField = %q, want %q", m.editingField, tt.fieldName)
			}

			for range len(tt.initial) {
				updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
				m = updated.(SettingsModel)
			}

			updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tt.typed)})
			m = updated.(SettingsModel)

			updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
			m = updated.(SettingsModel)

			if m.editingField != "" {
				t.Fatalf("editingField = %q, want empty", m.editingField)
			}

			if cmd == nil {
				t.Fatalf("Update() cmd = nil, want SettingsUpdateMsg cmd")
			}

			msg, ok := cmd().(SettingsUpdateMsg)
			if !ok {
				t.Fatalf("cmd() type = %T, want SettingsUpdateMsg", cmd())
			}

			if msg.Field != tt.fieldName {
				t.Fatalf("SettingsUpdateMsg.Field = %q, want %q", msg.Field, tt.fieldName)
			}

			if msg.Value != tt.expected {
				t.Fatalf("SettingsUpdateMsg.Value = %q, want %q", msg.Value, tt.expected)
			}
		})
	}
}

func TestSettingsCancelEditViaEscDoesNotEmitMessage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		initial string
		typed   string
	}{
		{
			name:    "esc cancels owner edit",
			initial: "oldorg",
			typed:   "x",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			m := NewSettingsModel(true, false, tt.initial, 0, "")

			updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
			m = updated.(SettingsModel)
			updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
			m = updated.(SettingsModel)

			updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
			m = updated.(SettingsModel)

			if m.editingField != "DefaultOwner" {
				t.Fatalf("editingField = %q, want %q", m.editingField, "DefaultOwner")
			}

			updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tt.typed)})
			m = updated.(SettingsModel)

			updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
			m = updated.(SettingsModel)

			if m.editingField != "" {
				t.Fatalf("editingField = %q, want empty", m.editingField)
			}

			if cmd != nil {
				if _, ok := cmd().(SettingsUpdateMsg); ok {
					t.Fatalf("esc emitted SettingsUpdateMsg unexpectedly")
				}
			}
		})
	}
}
