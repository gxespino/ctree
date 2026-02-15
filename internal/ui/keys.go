package ui

import "github.com/charmbracelet/bubbles/key"

type keyMap struct {
	Enter        key.Binding
	NewWorkspace key.Binding
	Refresh      key.Binding
	Quit         key.Binding
	Escape       key.Binding
}

func defaultKeyMap() keyMap {
	return keyMap{
		Enter:        key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "jump")),
		NewWorkspace: key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "new")),
		Refresh:      key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh")),
		Quit:         key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
		Escape:       key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
	}
}
