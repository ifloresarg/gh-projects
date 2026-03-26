package help

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestHelpShowHideAndView(t *testing.T) {
	t.Parallel()

	m := New(80, 24)
	if m.IsVisible() {
		t.Fatal("new help model should start hidden")
	}

	m.Show()
	if !m.IsVisible() {
		t.Fatal("Show() did not mark help as visible")
	}

	view := m.View()
	for _, fragment := range []string{"Global", "Board", "Detail", "Press any key to close"} {
		if !strings.Contains(view, fragment) {
			t.Fatalf("View() missing %q in %q", fragment, view)
		}
	}

	m.Hide()
	if got := m.View(); got != "" {
		t.Fatalf("hidden help View() = %q, want empty string", got)
	}
}

func TestHelpUpdateDismissesOnKeyAndResizes(t *testing.T) {
	t.Parallel()

	m := New(10, 10)
	m.Show()

	updated, _ := (&m).Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	if updated.width != 100 || updated.height != 40 {
		t.Fatalf("Update(WindowSizeMsg) = (%d, %d), want (100, 40)", updated.width, updated.height)
	}

	updated, _ = (&updated).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	if updated.IsVisible() {
		t.Fatal("key update did not dismiss help overlay")
	}
}

func TestHelpViewContainsSettingsKeybinding(t *testing.T) {
	t.Parallel()

	m := New(80, 24)
	m.Show()

	view := m.View()
	if !strings.Contains(view, "Settings") {
		t.Errorf("expected help view to contain 'Settings', got:\n%s", view)
	}
}

func TestHelpViewContainsEditBodyKeybinding(t *testing.T) {
	t.Parallel()

	m := New(80, 24)
	m.Show()

	view := m.View()
	if !strings.Contains(view, "  e         Edit body") {
		t.Fatalf("expected help view to contain edit body keybinding, got:\n%s", view)
	}
}
