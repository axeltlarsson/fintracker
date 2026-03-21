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
		content = m.list.View()
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
		t := m.transactions[m.selectedIndex]
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
	t := m.transactions[m.selectedIndex]

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

func (m Model) viewCategorySummaryScreen() tea.View {
	var b strings.Builder
	b.WriteString(m.styles.title.Render("Categories"))

	keys := make([]string, 0, len(m.categorySummary))
	for k := range m.categorySummary {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, category := range keys {
		fmt.Fprintf(&b, " %-25s %12s\n", category, m.categorySummary[category])
	}

	b.WriteString("\n esc back\n")

	return tea.NewView(b.String())
}
