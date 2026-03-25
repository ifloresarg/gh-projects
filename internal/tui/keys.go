package tui

import "github.com/charmbracelet/bubbles/key"

type KeyMap struct {
	Up        key.Binding
	Down      key.Binding
	Left      key.Binding
	Right     key.Binding
	Select    key.Binding
	Back      key.Binding
	Quit      key.Binding
	Help      key.Binding
	Refresh   key.Binding
	Search    key.Binding
	Settings  key.Binding
	MoveLeft  key.Binding
	MoveRight key.Binding
	Projects  key.Binding
	Owners    key.Binding
}

var DefaultKeyMap = KeyMap{
	Up:        key.NewBinding(key.WithKeys("k", "up"), key.WithHelp("k/↑", "up")),
	Down:      key.NewBinding(key.WithKeys("j", "down"), key.WithHelp("j/↓", "down")),
	Left:      key.NewBinding(key.WithKeys("h", "left"), key.WithHelp("h/←", "left")),
	Right:     key.NewBinding(key.WithKeys("l", "right"), key.WithHelp("l/→", "right")),
	Select:    key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "select")),
	Back:      key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
	Quit:      key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
	Help:      key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
	Refresh:   key.NewBinding(key.WithKeys("R"), key.WithHelp("R", "refresh")),
	Search:    key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "search")),
	Settings:  key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "settings")),
	MoveLeft:  key.NewBinding(key.WithKeys("H"), key.WithHelp("H", "move left")),
	MoveRight: key.NewBinding(key.WithKeys("L"), key.WithHelp("L", "move right")),
	Projects:  key.NewBinding(key.WithKeys("p"), key.WithHelp("p", "switch project")),
	Owners:    key.NewBinding(key.WithKeys("o"), key.WithHelp("o", "switch owner")),
}
