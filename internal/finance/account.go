package finance

import (
	"fmt"
	"strings"
	"time"
)

type AccountType string

const (
	Assets      AccountType = "Assets"
	Liabilities AccountType = "Liabilities"
	Income      AccountType = "Income"
	Expenses    AccountType = "Expenses"
	Equity      AccountType = "Equity"
)

type Account struct {
	ID       int64
	Path     string
	Type     AccountType
	Currency string
	OpenedAt *time.Time
	ClosedAt *time.Time
}

func (a Account) Name() string {
	s := strings.Split(a.Path, ":")
	return s[len(s)-1]
}
func (a Account) Parent() string {
	s := strings.Split(a.Path, ":")
	return strings.Join(s[:(len(s)-1)], ":")
}
func (a Account) Depth() int {
	return len(strings.Split(a.Path, ":")) - 1
}

func AccountTypeFromPath(path string) (AccountType, error) {
	before, _, _ := strings.Cut(path, ":")

	switch t := AccountType(before); t {
	case Assets, Equity, Liabilities, Income, Expenses:
		return t, nil
	default:
		return "", fmt.Errorf("invalid account type: %q", before)
	}

}
