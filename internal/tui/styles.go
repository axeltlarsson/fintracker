package tui

import (
	lipgloss "charm.land/lipgloss/v2"
	"fintracker/internal/finance"
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
}

func newStyles(t Theme) styles {
	return styles{
		theme: t,

		title: lipgloss.NewStyle().
			Bold(true).
			Foreground(t.Iris).
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
			Background(t.Surface).
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
	}
}

func (s styles) amountStyle(amount finance.Öre) lipgloss.Style {
	if amount >= 0 {
		return lipgloss.NewStyle().Foreground(s.theme.Pine).Background(s.theme.Surface)
	}
	return lipgloss.NewStyle().Foreground(s.theme.Love).Background(s.theme.Surface)
}


