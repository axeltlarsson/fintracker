package main

import (
	"fmt"
	"time"
)

type Öre int64

func (ö Öre) String() string {
	sign := ""
	v := int64(ö)

	if v < 0 {
		sign = "-"
		v = -v
	}

	return fmt.Sprintf("%s%d,%02d kr", sign, v/100, v%100)
}

type Transaction struct {
	ID       int64
	Date     time.Time
	Amount   Öre
	Payee    string
	Account  string
	Category string // empty until caetgorised
}

func CalculateBalance(txns []Transaction) Öre {
	// calculate the balance from a slice of transactions
	var balance Öre
	for _, t := range txns {
		balance += t.Amount
	}
	return balance
}

// Satisfy list.DefaultItem so we can show Transactions in bubbbles/list

func (t Transaction) Title() string {
	return fmt.Sprintf("%s %s", t.Date.Format("2006-01-02"), t.Payee)
}

func (t Transaction) Description() string {
	cat := t.Category
	if cat == "" {
		cat = uncategorized
	}

	return fmt.Sprintf("%12s • %s • %s", t.Amount, t.Account, cat)
}

// Satisfy list.Item - fuzzy search
func (t Transaction) FilterValue() string {
	return t.Payee + " " + t.Category + " " + t.Account
}
