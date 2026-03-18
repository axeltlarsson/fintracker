package tui

import (
	"fmt"

	"fintracker/internal/finance"
)

// TransactionItem wraps a Transaction to satisfy list.DetaulItem
type TransactionItem struct {
	finance.Transaction
}


func (t TransactionItem) Title() string {
	return fmt.Sprintf("%s %s", t.Date.Format("2006-01-02"), t.Payee)
}

func (t TransactionItem) Description() string {
	cat := t.Category
	if cat == "" {
		cat = uncategorized
	}

	return fmt.Sprintf("%12s • %s • %s", t.Amount, t.Account, cat)
}

// Satisfy list.Item - fuzzy search
func (t TransactionItem) FilterValue() string {
	return t.Payee + " " + t.Category + " " + t.Account
}
