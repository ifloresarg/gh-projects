package components

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type NotifKind int

const (
	KindSuccess NotifKind = iota
	KindError
	KindInfo
)

// DismissMsg is sent after the auto-dismiss timer fires for success notifications.
type DismissMsg struct{}

// Notification holds a transient notification message.
type Notification struct {
	msg     string
	kind    NotifKind
	visible bool
}

// Show creates a notification and returns the model + cmd.
// For success messages, returns a Tick cmd to auto-dismiss after 3s.
// For error/info messages, returns nil cmd (persistent until manually dismissed).
func Show(msg string, kind NotifKind) (Notification, tea.Cmd) {
	n := Notification{msg: msg, kind: kind, visible: true}
	if kind == KindSuccess {
		cmd := tea.Tick(3*time.Second, func(time.Time) tea.Msg { return DismissMsg{} })
		return n, cmd
	}
	return n, nil
}

// Update handles DismissMsg to clear the notification.
func (n *Notification) Update(msg tea.Msg) (Notification, tea.Cmd) {
	if _, ok := msg.(DismissMsg); ok {
		n.visible = false
	}
	return *n, nil
}

// View renders the notification or empty string if not visible.
func (n Notification) View() string {
	if !n.visible || n.msg == "" {
		return ""
	}

	var color string
	switch n.kind {
	case KindSuccess:
		color = "10" // green
	case KindError:
		color = "9" // red
	default:
		color = "14" // cyan
	}

	return lipgloss.NewStyle().
		Foreground(lipgloss.Color(color)).
		Render(n.msg)
}

// Hide clears the notification.
func (n *Notification) Hide() {
	n.visible = false
}
