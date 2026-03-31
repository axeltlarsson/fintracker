package tui

import (
	"fmt"
	"strings"

	lipgloss "charm.land/lipgloss/v2"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2/table"
	"fintracker/internal/finance"
	"sort"
)

const uncategorized = "(uncategorized)"

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
		footer := m.styles.help.Render("c categorise • ↑/↓ scroll • esc back")
		content = header + "\n" + body + "\n" + footer
	case summaryScreen:
		header := m.styles.title.Render("fintracker — summary")
		body := m.viewport.View()
		footer := m.styles.help.Render("↑/↓ scroll • esc back")
		content = header + "\n" + body + "\n" + footer

	case categoryScreen:
		txn := m.selectedTxn()
		if txn == nil {
			content = m.styles.warning.Render("cursor out of bounds")
			break
		}
		t := *txn
		header := m.styles.title.Render(fmt.Sprintf("Categorize: %s", t.Payee))
		prompt := m.styles.prompt.Render("Category: ")
		input := m.catInput.View()

		hint := m.styles.help.Render(m.help.View(m.catKeys))
		existing := m.styles.muted.PaddingLeft(2).MarginTop(1).
			Render("Existing: " + strings.Join(m.categories, ", "))

		content = header + "\n" + prompt + input + "\n" + existing + "\n\n" + hint

	}

	v := tea.NewView(content)
	v.AltScreen = true
	return v
}

func (m Model) renderDetail() string {
	txn := m.selectedTxn()
	if txn == nil {
		return m.styles.warning.Render("cursor out of bounds")
	}
	t := *txn

	var b strings.Builder
	row := func(label, value string) {
		b.WriteString(m.styles.label.Render(label))
		b.WriteString(m.styles.value.Render(value))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	row("Date", t.Date.Format("2006-01-02 (Monday)"))
	row("Payee", t.Payee)
	row("Amount", m.styles.amountStyle(t.Amount).Render(t.Amount.String()))
	row("Account", t.Account)

	if t.Category != "" {
		row("Category", m.styles.category.Render(t.Category))

	} else {
		row("Category", m.styles.uncategorized.Render(uncategorized))

	}
	b.WriteString("\n")

	// Show other transactions from the same payee
	b.WriteString(m.styles.sectionTitle.MarginTop(1).Render("Other transactions from " + t.Payee))
	b.WriteString("\n\n")

	count := 0
	for _, other := range m.transactions {
		if other.Payee == t.Payee && other.Date != t.Date {
			fmt.Fprintf(&b, " %s %s\n",
				other.Date.Format("2006-01-02"),
				m.styles.amountStyle(other.Amount).Render(other.Amount.String()),
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

// per-account and pe-category summary view
func (m Model) renderSummary() string {
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
	categoryRows = append(categoryRows, []string{"Total", m.styles.amountStyle(m.totalBalance).Render(m.totalBalance.String())})
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
	filtered := m.table.FilteredCount()
	structFiltered := len(m.visibleIdx)
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

	return m.styles.statusLeft.Width(leftW).Render(left) +
		m.styles.statusMiddle.Width(middleW).Render(middle) +
		m.styles.statusRight.Width(rightW).Render(right)

}
