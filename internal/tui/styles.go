package tui

import (
	lipgloss "charm.land/lipgloss/v2"
	"fintracker/internal/finance"
)

var (
	// colors
	colorPrimary   = lipgloss.Color("#7D56F4")
	colorSecondary = lipgloss.Color("#04B575")
	colorMuted     = lipgloss.Color("#626262")
	colorDanger    = lipgloss.Color("#FF4672")
	colorIncome    = lipgloss.Color("#04B575")
	colorExpense   = lipgloss.Color("#FF4672")

	// base styles

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorPrimary).
			MarginBottom(1).
			PaddingLeft(2)

	helpStyle = lipgloss.NewStyle().
			Foreground(colorMuted).
			PaddingLeft(2).
			MarginTop(1)

	cursorStyle = lipgloss.NewStyle().
			Foreground(colorPrimary).
			Bold(true)

	selectedRowStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("#2D2D2D")).
				Bold(true)
	categoryStyle = lipgloss.NewStyle().
			Foreground(colorSecondary).
			Italic(true)

	uncategorizedStyle = lipgloss.NewStyle().
				Foreground(colorMuted).
				Italic(true)
)

func amountStyle(amount finance.Öre) lipgloss.Style {
	if amount >= 0 {
		return lipgloss.NewStyle().Foreground(colorIncome)
	}
	return lipgloss.NewStyle().Foreground(colorExpense)
}
