package main

import (
	"fmt"
	"strings"

	lipgloss "charm.land/lipgloss/v2"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2/table"
	"sort"
)

const uncategorized = "(uncategorized)"

func (m model) View() tea.View {

	if !m.ready {
		return tea.NewView("  loading...")
	}

	var content string

	switch m.screen {
	case listScreen:
		content = m.list.View()
	case detailScreen:
		header := titleStyle.Render("fintracker — detail")
		body := m.viewport.View()
		footer := helpStyle.Render("c categorise • ↑/↓ scroll • esc back")
		content = header + "\n" + body + "\n" + footer
	case summaryScreen:
		header := titleStyle.Render("fintracker — summary")
		body := m.viewport.View()
		footer := helpStyle.Render("↑/↓ scroll • esc back")
		content = header + "\n" + body + "\n" + footer

	case categoryScreen:
		t := m.transactions[m.selectedIndex]
		header := titleStyle.Render(fmt.Sprintf("Categorize: %s", t.Payee))
		prompt := lipgloss.NewStyle().PaddingLeft(2).Render("Category: ")
		input := m.catInput.View()

		hint := helpStyle.Render(m.help.View(m.catKeys))
		existing := lipgloss.NewStyle().
			Foreground(colorMuted).PaddingLeft(2).MarginTop(1).
			Render("Existing: " + strings.Join(m.categories, ", "))

		content = header + "\n" + prompt + input + "\n" + existing + "\n\n" + hint

	}

	v := tea.NewView(content)
	v.AltScreen = true
	return v
}

func (m model) renderDetail() string {
	t := m.transactions[m.selectedIndex]

	labelStyle := lipgloss.NewStyle().
		Width(14).
		Foreground(colorMuted).
		PaddingLeft(2)

	valueStyle := lipgloss.NewStyle().Bold(true)

	var b strings.Builder
	row := func(label, value string) {
		b.WriteString(labelStyle.Render(label))
		b.WriteString(valueStyle.Render(value))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	row("Date", t.Date.Format("2006-01-02 (Monday)"))
	row("Payee", t.Payee)
	row("Amount", amountStyle(t.Amount).Render(t.Amount.String()))
	row("Account", t.Account)

	if t.Category != "" {
		row("Category", categoryStyle.Render(t.Category))

	} else {
		row("Category", uncategorizedStyle.Render(uncategorized))

	}
	b.WriteString("\n")

	// Show other transactions from the same payee
	b.WriteString(lipgloss.NewStyle().
		Bold(true).PaddingLeft(2).MarginTop(1).Render("Other transactions from " + t.Payee))
	b.WriteString("\n\n")

	count := 0
	for _, other := range m.transactions {
		if other.Payee == t.Payee && other.Date != t.Date {
			fmt.Fprintf(&b, " %s %s\n",
				other.Date.Format("2006-01-02"),
				amountStyle(other.Amount).Render(other.Amount.String()),
			)
			count++
			if count >= 10 {
				b.WriteString(" ....\n")
				break
			}

		}
	}

	if count == 0 {
		b.WriteString(lipgloss.NewStyle().
			Padding(2).Foreground(colorMuted).Render("No other transactions"))
		b.WriteString("\n")
	}

	return b.String()
}

// per-account and pe-category summary view
func (m model) renderSummary() string {
	var b strings.Builder

	// Accounts table
	b.WriteString(lipgloss.NewStyle().
		Bold(true).
		PaddingLeft(2).
		Render("Accounts"))
	b.WriteString("\n\n")

	var accountRows [][]string
	for _, acc := range sortedKeys(m.accountSummary) {
		amount := m.accountSummary[acc]
		accountRows = append(accountRows, []string{acc, amount.String()})
	}

	at := table.New().
		Border(lipgloss.RoundedBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(colorMuted)).
		Headers("Account", "Balance").
		Rows(accountRows...).
		StyleFunc(func(row, col int) lipgloss.Style {
			s := lipgloss.NewStyle().PaddingRight(3)
			if row == table.HeaderRow {
				return s.Bold(true).Foreground(colorPrimary)
			}
			if col == 1 {
				return s.Align(lipgloss.Right)
			}
			return s
		})
	b.WriteString(at.Render())
	b.WriteString("\n\n")

	// Category table
	b.WriteString(lipgloss.NewStyle().
		Bold(true).
		PaddingLeft(2).
		Render("Categories"))
	b.WriteString("\n\n")

	var categoryRows [][]string
	for _, acc := range sortedKeys(m.categorySummary) {
		amount := m.categorySummary[acc]
		categoryRows = append(categoryRows, []string{acc, amount.String()})
	}

	// add last row with total
	categoryRows = append(categoryRows, []string{"Total", amountStyle(m.totalBalance).Render(m.totalBalance.String())})
	ct := table.New().
		Border(lipgloss.RoundedBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(colorMuted)).
		Headers("Category", "Balance").
		Rows(categoryRows...).
		StyleFunc(func(row, col int) lipgloss.Style {
			s := lipgloss.NewStyle().PaddingRight(3)
			if row == table.HeaderRow {
				s = s.Bold(true).Foreground(colorPrimary)
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

	b.WriteString(helpStyle.Render("esc back • q quit"))

	return b.String()
}

func sortedKeys(m map[string]Öre) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func (m model) viewCategorySummaryScreen() tea.View {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Categories"))

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
