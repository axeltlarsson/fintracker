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

	statusFilter  lipgloss.Style
	statusMessage lipgloss.Style
	statusLeft    lipgloss.Style
	statusMiddle  lipgloss.Style
	statusRight   lipgloss.Style
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
			Foreground(t.Rose).
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
			Foreground(t.Subtle).
			Padding(0, 1),

		tableCell: lipgloss.NewStyle().
			Padding(0, 1),

		prompt: lipgloss.NewStyle().
			PaddingLeft(2),

		value: lipgloss.NewStyle().
			Bold(true),

		warning: lipgloss.NewStyle().
			Bold(true).
			Foreground(t.Love),

		statusLeft: lipgloss.NewStyle().
			Background(t.Surface).
			Foreground(t.Muted).
			PaddingLeft(2),

		statusMiddle: lipgloss.NewStyle().
			Background(t.Surface),

		statusRight: lipgloss.NewStyle().
			Background(t.Surface).
			Foreground(t.Muted).
			Align(lipgloss.Right).
			PaddingRight(2),

		statusFilter: lipgloss.NewStyle().
			Foreground(t.Iris).
			Bold(true),

		statusMessage: lipgloss.NewStyle().
			Foreground(t.Gold),
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

func (s styles) transactionStyleFuncFromIdx(transactions []finance.Transaction, accts map[int64]finance.Account, idx []int) TxnStyleFunc {
	return func(row, col int, selected bool) lipgloss.Style {
		base := s.tableCell

		if selected {
			base = base.Bold(true).Background(s.theme.HighlightLow).Foreground(s.theme.Rose)
		}

		if row < 0 || row >= len(idx) {
			return base
		}

		e := transactions[idx[row]]
		v := projectTransaction(e, accts)
		state := reviewStateOf(e, accts)

		// Uncleared rows recede - except under the cursor, where legibiilty beats consistency
		dim := state == stateReview && !selected
		if dim {
			base = base.Foreground(s.theme.Muted)
		}

		switch col {
		case colStatus:
			base = base.Foreground(s.statusColor(state))
			if state == stateReview {
				return base.Bold(true)
			}
			return base
		case colAmount:
			if dim {
				return base
			}
			return base.Foreground(s.amountColor(v.Amount))
		case colCategory:
			if dim {
				return base.Italic(true)
			}
			return base.Foreground(s.theme.Foam).Italic(true)
		}
		return base

	}
}

func (s styles) statusColor(r reviewState) color.Color {
	switch r {
	case stateReview:
		return s.theme.Gold
	case stateTransfer:
		return s.theme.Muted
	default:
		return s.theme.Pine
	}
}
