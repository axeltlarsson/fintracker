package tui

import (
	lipgloss "charm.land/lipgloss/v2"
	"fintracker/internal/finance"

	"charm.land/bubbles/v2/help"
	// "charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/table"
)

type styles struct {
	theme         Theme
	title         lipgloss.Style
	help          lipgloss.Style
	cursor        lipgloss.Style
	selectedRow   lipgloss.Style
	category      lipgloss.Style
	uncategorized lipgloss.Style
	label         lipgloss.Style
	muted         lipgloss.Style
	sectionTitle  lipgloss.Style
	tableBorder   lipgloss.Style
	tableHeader   lipgloss.Style
	tableCell     lipgloss.Style
	prompt        lipgloss.Style
	value         lipgloss.Style
	warning       lipgloss.Style
}

func newStyles(t Theme) styles {
	return styles{
		theme: t,

		title: lipgloss.NewStyle().
			Bold(true).
			Foreground(t.Rose).
			MarginBottom(1).
			PaddingLeft(2),

		help: lipgloss.NewStyle().
			Foreground(t.Muted).
			PaddingLeft(2).
			MarginTop(1),

		cursor: lipgloss.NewStyle().
			Foreground(t.Iris).
			Bold(true),

		selectedRow: lipgloss.NewStyle().
			Background(t.HighlightLow).
			Foreground(t.Gold).
			Bold(true),

		category: lipgloss.NewStyle().
			Foreground(t.Foam).
			Italic(true),

		uncategorized: lipgloss.NewStyle().
			Foreground(t.Muted).
			Italic(true),

		label: lipgloss.NewStyle().
			Width(14).
			Foreground(t.Muted).
			PaddingLeft(2),

		muted: lipgloss.NewStyle().
			Foreground(t.Muted),

		sectionTitle: lipgloss.NewStyle().
			Bold(true).
			PaddingLeft(2),

		tableBorder: lipgloss.NewStyle().
			Foreground(t.Muted),

		tableHeader: lipgloss.NewStyle().
			Bold(true).
			Foreground(t.Iris).
			PaddingRight(3),

		tableCell: lipgloss.NewStyle().
			PaddingRight(3),

		prompt: lipgloss.NewStyle().
			PaddingLeft(2),

		value: lipgloss.NewStyle().
			Bold(true),

		warning: lipgloss.NewStyle().
			Bold(true).
			Foreground(t.Love),
	}
}

func (s styles) amountStyle(amount finance.Öre) lipgloss.Style {
	if amount >= 0 {
		return lipgloss.NewStyle().Foreground(s.theme.Pine)
	}
	return lipgloss.NewStyle().Foreground(s.theme.Love)
}

func newHelpStyles(t Theme) help.Styles {
	return help.Styles{
		Ellipsis:       lipgloss.NewStyle().Foreground(t.Muted),
		ShortKey:       lipgloss.NewStyle().Foreground(t.Subtle),
		ShortDesc:      lipgloss.NewStyle().Foreground(t.Muted),
		ShortSeparator: lipgloss.NewStyle().Foreground(t.Muted),
		FullKey:        lipgloss.NewStyle().Foreground(t.Subtle),
		FullDesc:       lipgloss.NewStyle().Foreground(t.Muted),
		FullSeparator:  lipgloss.NewStyle().Foreground(t.Muted),
	}
}

func (s styles) tableStyles() table.Styles {
	return table.Styles{
		Header:   lipgloss.NewStyle().Bold(true).Foreground(s.theme.Iris).Padding(0, 1),
		Cell:     lipgloss.NewStyle().Padding(0, 1),
		Selected: s.selectedRow,
	}
}

// styles for Bubbles list
/* func newListStyles(t Theme) list.Styles {

	s := list.Styles{}

	s.TitleBar = lipgloss.NewStyle().Padding(0, 0, 1, 2)
	s.Title = lipgloss.NewStyle().
		Background(t.Iris).
		Foreground(t.Base).
		Padding(0, 1)

	s.Spinner = lipgloss.NewStyle().Foreground(t.Subtle)
	s.StatusBar = lipgloss.NewStyle().
		Foreground(t.Subtle).
		Padding(0, 0, 1, 2)
	s.StatusEmpty = lipgloss.NewStyle().Foreground(t.Muted)
	s.StatusBarActiveFilter = lipgloss.NewStyle().Foreground(t.Text)
	s.StatusBarFilterCount = lipgloss.NewStyle().Foreground(t.Muted)
	s.NoItems = lipgloss.NewStyle().Foreground(t.Muted)
	s.PaginationStyle = lipgloss.NewStyle().PaddingLeft(2)
	s.HelpStyle = lipgloss.NewStyle().Padding(1, 0, 0, 2)
	s.ActivePaginationDot = lipgloss.NewStyle().
		Foreground(t.Subtle).SetString("•")
	s.InactivePaginationDot = lipgloss.NewStyle().
		Foreground(t.Muted).SetString("•")
	s.DividerDot = lipgloss.NewStyle().
		Foreground(t.Muted).SetString(" • ")
	s.DefaultFilterCharacterMatch = lipgloss.NewStyle().Underline(true)

	return s

}

func newItemStyles(t Theme) list.DefaultItemStyles {
	s := list.DefaultItemStyles{}

	s.NormalTitle = lipgloss.NewStyle().
		Foreground(t.Text).
		Padding(0, 0, 0, 2)
	s.NormalDesc = s.NormalTitle.
		Foreground(t.Subtle)

	s.SelectedTitle = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, false, false, true).
		BorderForeground(t.Iris).
		Foreground(t.Iris).
		Padding(0, 0, 0, 1)
	s.SelectedDesc = s.SelectedTitle.
		Foreground(t.Subtle)

	s.DimmedTitle = lipgloss.NewStyle().
		Foreground(t.Muted).
		Padding(0, 0, 0, 2)
	s.DimmedDesc = s.DimmedTitle.
		Foreground(t.HighlightHigh)

	s.FilterMatch = lipgloss.NewStyle().Underline(true)

	return s
} */
