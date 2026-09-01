package tui

import (
	"fmt"
	"strings"

	lipgloss "charm.land/lipgloss/v2"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2/table"
	"fintracker/internal/finance"
	"github.com/charmbracelet/x/ansi"
	"sort"
)

const uncategorized = "(uncategorized)"

const (
	glyphCleared  = "*"
	glyphReview   = "!"
	glyphTransfer = "→"
)

func (m Model) View() tea.View {

	if !m.ready {
		return tea.NewView("  loading...")
	}

	var content string

	switch m.screen {
	case listScreen:
		content = strings.Join(
			[]string{
				m.styles.title.Render(appTitle),
				m.styles.help.Render(m.help.View(m.keys)),
				m.table.View(),
				m.renderStatusLine(),
			}, "\n")
	case detailScreen:
		header := m.styles.title.Render("fintracker — detail")
		body := m.viewport.View()
		footer := m.styles.help.Render("↑/↓ scroll • esc back")
		content = header + "\n" + body + "\n" + footer
	case summaryScreen:
		header := m.styles.title.Render("fintracker — summary")
		body := m.viewport.View()
		footer := m.styles.help.Render("↑/↓ scroll • esc back")
		content = header + "\n" + body + "\n" + footer

	case reviewScreen:
		content = m.renderReview()

	case rulePromptScreen:
		content = m.renderRulePrompt()

	}

	v := tea.NewView(content)
	v.AltScreen = true
	return v
}

func (m Model) renderDetail() string {
	e := m.selectedTransaction()
	if e == nil {
		return m.styles.warning.Render("cursor out of bounds")
	}
	v := projectTransaction(*e, m.accountsByID)

	var b strings.Builder
	row := func(label, value string) {
		b.WriteString(m.styles.label.Render(label))
		b.WriteString(m.styles.value.Render(value))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	row("Date", e.Date.Format("2006-01-02 (Monday)"))
	row("Payee", e.DisplayPayee())
	row("Amount", m.styles.amountStyle(v.Amount).Render(v.Amount.String()))
	row("Account", v.Account)

	if e.Cleared {
		row("Category", m.styles.category.Render(v.Category))

	} else {
		row("Category", m.styles.uncategorized.Render(v.Category+" (unreviewed)"))

	}
	b.WriteString("\n")

	// Show other transactions from the same payee
	b.WriteString(m.styles.sectionTitle.MarginTop(1).Render("Other transactions from " + e.DisplayPayee()))
	b.WriteString("\n\n")

	count := 0
	for i := range m.transactions {
		other := m.transactions[i]
		if other.DisplayPayee() == e.DisplayPayee() && !other.Date.Equal(e.Date) {
			ov := projectTransaction(other, m.accountsByID)
			fmt.Fprintf(&b, " %s %s\n",
				other.Date.Format("2006-01-02"),
				m.styles.amountStyle(ov.Amount).Render(ov.Amount.String()),
			)
			count++
			if count >= 10 {
				b.WriteString(" ....\n")
				break
			}

		}
	}

	if count == 0 {
		b.WriteString(m.styles.muted.Render("No other transactions"))
		b.WriteString("\n")
	}

	return b.String()
}

func (m Model) renderReview() string {
	t := m.selectedTransaction()
	if t == nil {
		return m.styles.warning.Render("cursor out of bound")
	}
	v := projectTransaction(*t, m.accountsByID)

	var b strings.Builder
	b.WriteString(m.styles.title.Render("Review: " + t.DisplayPayee()))
	b.WriteString("\n\n")
	b.WriteString(m.styles.label.Render("Amount"))
	b.WriteString(m.styles.amountStyle(v.Amount).Render(v.Amount.String()))
	b.WriteString("\n")
	b.WriteString(m.styles.label.Render("Currently"))
	b.WriteString(m.styles.value.Render(v.Category))
	b.WriteString("\n\n")
	b.WriteString(m.styles.label.Render("Contra account "))
	b.WriteString(m.reviewInput.View())
	b.WriteString("\n\n")
	if m.reviewErr != "" {
		b.WriteString(m.styles.warning.Render(m.reviewErr))
	}
	b.WriteString(m.styles.help.Render("tab complete • enter confirm • esc cancel"))
	return b.String()
}

func (m Model) renderRulePrompt() string {
	var b strings.Builder

	b.WriteString(m.styles.title.Render("Remember this payee?"))
	b.WriteString("\n\n")
	b.WriteString(m.styles.value.Render("Future imports matching this pattern auto-categorise to:"))

	b.WriteString("\n\n")
	b.WriteString(m.styles.label.Render("Account"))
	b.WriteString(m.styles.value.Render(m.pendingRuleAccount))

	b.WriteString("\n\n")
	b.WriteString(m.styles.label.Render("Pattern "))
	b.WriteString(m.ruleInput.View())
	b.WriteString("\n\n")
	if m.ruleErr != "" {
		b.WriteString(m.styles.warning.Render(m.ruleErr))
		b.WriteString("\n")
	}
	b.WriteString(m.styles.help.Render("enter save • esc skip"))
	return b.String()
}

// per-account and pe-category summary view
func (m Model) renderSummary() string {
	// NOTE: net worth is not the sum of the category rows anymore should fix!
	var b strings.Builder

	// Accounts table
	b.WriteString(m.styles.sectionTitle.Render("Accounts"))
	b.WriteString("\n\n")

	var accountRows [][]string
	for _, acc := range sortedKeys(m.accountSummary) {
		amount := m.accountSummary[acc]
		accountRows = append(accountRows, []string{acc, m.styles.amountStyle(amount).Render(amount.String())})
	}

	at := table.New().
		Border(lipgloss.RoundedBorder()).
		BorderStyle(m.styles.tableBorder).
		Headers("Account", "Balance").
		Rows(accountRows...).
		StyleFunc(func(row, col int) lipgloss.Style {
			s := m.styles.tableCell
			if row == table.HeaderRow {
				return m.styles.tableHeader
			}
			if col == 1 {
				return s.Align(lipgloss.Right)
			}
			return s
		})
	b.WriteString(at.Render())
	b.WriteString("\n\n")

	// Category table
	b.WriteString(m.styles.sectionTitle.Render("Categories"))
	b.WriteString("\n\n")

	var categoryRows [][]string
	for _, acc := range sortedKeys(m.categorySummary) {
		amount := m.categorySummary[acc]
		categoryRows = append(categoryRows, []string{acc, m.styles.amountStyle(amount).Render(amount.String())})
	}

	// add last row with total
	categoryRows = append(categoryRows, []string{"Total", m.styles.amountStyle(m.netWorth).Render(m.netWorth.String())})
	ct := table.New().
		Border(lipgloss.RoundedBorder()).
		BorderStyle(m.styles.tableBorder).
		Headers("Category", "Balance").
		Rows(categoryRows...).
		StyleFunc(func(row, col int) lipgloss.Style {
			s := m.styles.tableCell
			if row == table.HeaderRow {
				s = m.styles.tableHeader
			}
			if col == 1 && row != table.HeaderRow {
				s = s.Align(lipgloss.Right)
			}
			if row == len(categoryRows)-1 {
				// last row is total row
				s = s.Bold(true)
			}
			return s
		})
	b.WriteString(ct.Render())

	b.WriteString(m.styles.help.Render("esc back • q quit"))

	return b.String()
}

func sortedKeys(m map[string]finance.Öre) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// renderStatusLine renders the contextual status bar between table and help.
func (m Model) renderStatusLine() string {
	var left, middle, right string

	// Left: filter state
	if m.filterAccount == "" {
		left = "All accounts"
	} else {
		// Find position in cycle
		pos := 0
		for i, a := range m.accounts {
			if a == m.filterAccount {
				pos = i + 1
				break
			}
		}
		left = m.styles.statusFilter.Render(
			fmt.Sprintf("%s (%d/%d)", m.filterAccount, pos, len(m.accounts)),
		)
	}

	// Middle: search or import status
	if m.searching || m.searchInput.Value() != "" {
		middle = m.styles.statusFilter.Render("search: ") + m.searchInput.View()
	} else {
		middle = m.styles.statusMessage.Render(m.importStatus)
	}

	// Right: transaction count
	total := len(m.transactions)
	filtered := m.table.SearchedCount()
	structFiltered := len(m.filteredTransactions)
	var msg string
	if filtered < structFiltered {
		msg = fmt.Sprintf("%d of %d transactions", filtered, total)
	} else if structFiltered < total {
		msg = fmt.Sprintf("%d of %d transactions", structFiltered, total)
	} else {
		msg = fmt.Sprintf("%d transactions", total)
	}
	right = m.styles.muted.Render(msg)

	// Layout: left -- middle -- right - fixed-width columns
	leftW := m.width / 3
	rightW := m.width / 3
	middleW := m.width - leftW - rightW

	return clampToWidth(left, m.styles.statusLeft, leftW) +
		clampToWidth(middle, m.styles.statusMiddle, middleW) +
		clampToWidth(right, m.styles.statusRight, rightW)

}

// clampToWidth renders s at exactly w cells, truncating so a long
// message can never wrap the status line onto a second row
func clampToWidth(s string, st lipgloss.Style, w int) string {
	return st.Width(w).Render(ansi.Truncate(s, w-st.GetHorizontalFrameSize(), "…"))
}
