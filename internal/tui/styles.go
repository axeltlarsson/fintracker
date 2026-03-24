package tui

import (
	lipgloss "charm.land/lipgloss/v2"
	"fintracker/internal/finance"

	"charm.land/bubbles/v2/help"
	"image/color"
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

func (s styles) amountColor(amount finance.Öre) color.Color {
	if amount >= 0 {
		return s.theme.Pine
	}
	return s.theme.Love

}
func (s styles) amountStyle(amount finance.Öre) lipgloss.Style {
	return lipgloss.NewStyle().Foreground(s.amountColor(amount))
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

func (s styles) transactionStyleFunc(txns []finance.Transaction) TxnStyleFunc {
	return func(row, col int, selected bool) lipgloss.Style {
		base := lipgloss.NewStyle().Padding(0, 1)

		if selected {
			base = base.Bold(true).Background(s.theme.HighlightLow)
		}

		if row < 0 || row >= len(txns) {
			return base
		}

		switch col {
		case colAmount: // Amount
			if txns[row].Amount >= 0 {
				return base.Foreground(s.theme.Pine).Align(lipgloss.Right)
			}
			return base.Foreground(s.theme.Love).Align(lipgloss.Right)
		case colCategory:
			if txns[row].Category == "" {
				// TODO should use uncategorized style?
				return base.Foreground(s.theme.Muted).Italic(true)
			}
			return base.Foreground(s.theme.Foam).Italic(true)
		}
		if selected {
			// TODO: use selectedRow style?
			return base.Foreground(s.theme.Gold)
		}
		return base

	}
}
