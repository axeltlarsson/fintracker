package tui

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
	Search   key.Binding
}

type categoryKeyMap struct {
	Confirm key.Binding
	Tab     key.Binding
	Back    key.Binding
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Enter, k.Category, k.Summary, k.Search, k.Filter, k.Help, k.Quit}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Enter},
		{k.Category, k.Summary, k.Filter, k.Search},
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
			key.WithHelp("esc", "back"),
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
		Search: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "search"),
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

func (k categoryKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Confirm, k.Tab, k.Back}
}

func (k categoryKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Confirm, k.Tab},
		{k.Back},
	}
}
