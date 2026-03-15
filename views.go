package main

import (
	"fmt"
	"strings"

	lipgloss "charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/table"

	tea "charm.land/bubbletea/v2"
	"sort"
)

const uncategorized = "(uncategorized)"

func (m model) View() tea.View {

	switch m.screen {
	case listScreen:
		return m.viewList()
	case detailScreen:
		return m.viewDetail()
	case summaryScreen:
		return m.viewSummary()
	case categoryScreen:
		return m.viewCategory()
	case categorySummaryScreen:
		return m.viewCategorySummaryScreen()

	default:
		return tea.NewView("unknown screen")
	}
}

func (m model) viewList() tea.View {
	var b strings.Builder

	b.WriteString(titleStyle.Render("fintracker"))
	b.WriteString("\n")

	// Build table rows
	headers := []string{"", "Date", "Payee", "Amount", "Balance", "Category"}

	var rows [][]string
	var running Öre

	for i, t := range m.transactions {
		if m.filterAccount != "" && t.Account != m.filterAccount {
			continue
		}
		running += t.Amount

		cursor := " "
		if i == m.cursor {
			cursor = ">"
		}

		cat := t.Category
		if cat == "" {
			cat = "-"
		}

		rows = append(rows, []string{
			cursor,
			t.Date.Format("2006-01-02"),
			t.Payee,
			t.Amount.String(),
			running.String(),
			cat,
		})
	}

	t := table.New().
		Border(lipgloss.RoundedBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(colorMuted)).
		Headers(headers...).
		Rows(rows...).
		StyleFunc(func(row, col int) lipgloss.Style {
			base := lipgloss.NewStyle().PaddingRight(2)

			if row == table.HeaderRow {
				return base.Bold(true).Foreground(colorPrimary)
			}

			// Highlight selected row
			if row < len(rows) && rows[row][0] == ">" {
				base = base.Bold(true)
			}

			switch col {
			case 0: // cursor
				return base.Foreground(colorPrimary).Width(2)
			case 3, 4: // amount, balance
				return base.Align(lipgloss.Right)
			case 5: // category
				if row < len(rows) && rows[row][5] == "-" {
					return base.Foreground(colorMuted).Italic(true)
				}
				return base.Foreground(colorSecondary)
			}
			return base
		})
	b.WriteString(t.Render())

	// Footer
	filter := "all accounts"
	if m.filterAccount != "" {
		filter = m.filterAccount
	}
	footer := fmt.Sprintf(" %s • %d transactions • balance: %s",
		filter, len(rows), m.totalBalance)
	b.WriteString("\n" + lipgloss.NewStyle().Foreground(colorMuted).Render(footer))

	help := "  j/k navigate • enter detail • summary • c categories • tab filter account • q quit"
	b.WriteString("\n" + helpStyle.Render(help))

	return tea.NewView(b.String())

}

func (m model) viewDetail() tea.View {
	t := m.transactions[m.cursor]

	var b strings.Builder

	b.WriteString(titleStyle.Render("fintracker — detail"))
	b.WriteString("\n\n")

	labelStyle := lipgloss.NewStyle().
		Width(12).
		Foreground(colorMuted).
		PaddingLeft(2)

	valueStyle := lipgloss.NewStyle().Bold(true)

	row := func(label, value string) {
		b.WriteString(labelStyle.Render(label))
		b.WriteString(valueStyle.Render(value))
		b.WriteString("\n")

	}

	row("Date", t.Date.Format("2006-01-02"))
	row("Payee", t.Payee)
	row("Amount", amountStyle(t.Amount).Render(t.Amount.String()))
	row("Account", t.Account)

	if t.Category != "" {
		row("Category", categoryStyle.Render(t.Category))

	} else {
		row("Category", uncategorizedStyle.Render(uncategorized))

	}

	b.WriteString("\n")
	b.WriteString(helpStyle.Render("  c categorize • esc back • q quit"))

	return tea.NewView(b.String())
}

// per-account and pe-category summary view
func (m model) viewSummary() tea.View {
	var b strings.Builder
	b.WriteString(titleStyle.Render("fintracker — summary"))
	b.WriteString("\n")


	// Accounts
	b.WriteString(lipgloss.NewStyle().
		Bold(true).
		PaddingLeft(2).
		MarginBottom(1).
		Render("Accounts"))
	b.WriteString("\n")

	keys := make([]string, 0, len(m.accountSummary))
	for k := range m.accountSummary {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, account := range keys {
		amount := m.accountSummary[account]
		fmt.Fprintf(&b, " %-20s %12s\n",
			lipgloss.NewStyle().PaddingLeft(2).Render(account), amountStyle(amount).Render(amount.String()),
		)
	}

	// Categories
	catSummary := make(map[string]Öre)
	for _, t := range m.transactions {
		cat := t.Category
		if cat == "" {
			cat = uncategorized
		}
		catSummary[cat] += t.Amount
	}
	b.WriteString("\n")
	b.WriteString(lipgloss.NewStyle().
		Bold(true).
		PaddingLeft(2).
		MarginBottom(1).
		Render("Categories"))
	b.WriteString("\n")

	catKeys := make([]string, 0, len(catSummary))
	for k := range catSummary {
		catKeys = append(catKeys, k)
	}
	sort.Strings(catKeys)

	for _, cat := range catKeys {
		amount := catSummary[cat]
		style := categoryStyle.PaddingLeft(2)
		if cat == uncategorized {
			style = uncategorizedStyle.PaddingLeft(2)
		}
		fmt.Fprintf(&b, " %-50s %12s\n",
			style.Render(cat),
			amountStyle(amount).Render(amount.String()),
		)
	}

	b.WriteString("\n")
	totalLabel := lipgloss.NewStyle().Bold(true).PaddingLeft(2).Render("Total")
	fmt.Fprintf(&b, "%-20s %s\n", totalLabel,
		amountStyle(m.totalBalance).Render(m.totalBalance.String()))
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("  esc back • q quit"))

	return tea.NewView(b.String())
}

func (m model) viewCategory() tea.View {
	t := m.transactions[m.cursor]

	var b strings.Builder

	title := fmt.Sprintf("Categorize: %s — %s",
		t.Payee, amountStyle(t.Amount).Render(t.Amount.String()))
	b.WriteString(titleStyle.Render(title))
	b.WriteString("\n\n")

	for i, cat := range m.categories {
		cursor := " "
		if i == m.catCursor {
			cursor = cursorStyle.Render("> ")
		}

		style := lipgloss.NewStyle()
		if cat == t.Category {
			style = style.Foreground(colorSecondary).Bold(true)
		}

		marker := ""
		if cat == t.Category {
			marker = " ✓"
		}

		fmt.Fprintf(&b, "%s%s%s\n", cursor, style.Render(cat), marker)

	}
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("  j/k navigate • enter select • esc back\n"))

	return tea.NewView(b.String())
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
