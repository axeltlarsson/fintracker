package finance_test

import (
	"fintracker/internal/finance"
	"testing"
)

func TestAccountName(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"Assets:Bank:SEB", "SEB"},
		{"Expenses:Food:Groceries", "Groceries"},
		{"Equity", "Equity"},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			a := finance.Account{Path: tt.path}
			if got := a.Name(); got != tt.want {
				t.Errorf("Name() = %q, want %q", got, tt.want)
			}
		})
	}

}

func TestAccountParent(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"Assets:Bank:SEB", "Assets:Bank"},
		{"Expenses:Food", "Expenses"},
		{"Equity", ""},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			a := finance.Account{Path: tt.path}
			if got := a.Parent(); got != tt.want {
				t.Errorf("Parent() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestAccountDepth(t *testing.T) {
	tests := []struct {
		path string
		want int
	}{
		{"Equity", 0},
		{"Assets:Bank", 1},
		{"Assets:Bank:SEB", 2},
		{"Expenses:Food:Restaurants", 2},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			a := finance.Account{Path: tt.path}
			if got := a.Depth(); got != tt.want {
				t.Errorf("Depth() = %d, want %d", got, tt.want)
			}
		})
	}
}
