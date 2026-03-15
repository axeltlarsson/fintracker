package main

import "charm.land/bubbles/v2/key"

type keyMap struct {
	Up       key.Binding
	Down     key.Binding
	Enter    key.Binding
	Back     key.Binding
	Quit     key.Binding
	Category key.Binding
	Summary  key.Binding
	Filter   key.Binding
	Help     key.Binding
}

type categoryKeyMap struct {
	Confirm key.Binding
	Tab     key.Binding
	Back    key.Binding
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Enter, k.Category, k.Summary, k.Quit}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Enter},
		{k.Category, k.Summary, k.Filter},
		{k.Back, k.Help, k.Quit},
	}
}

func newKeyMap() keyMap {

	return keyMap{
		Up: key.NewBinding(
			key.WithKeys("k", "up"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("j", "down"),
			key.WithHelp("↓/j", "down"),
		),
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "detail"),
		),
		Back: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "detail"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl-c"),
			key.WithHelp("q", "quit"),
		),
		Category: key.NewBinding(
			key.WithKeys("c"),
			key.WithHelp("c", "categorize"),
		),
		Summary: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "summary"),
		),
		Filter: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "filter accounts"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
	}
}

func newCategoryKeyMap() categoryKeyMap {
	return categoryKeyMap{
		Confirm: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "confirm"),
		),
		Tab: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "complete"),
		),
		Back: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "cancel"),
		),
	}

}
