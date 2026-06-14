package tui

import (
	"testing"

	"fintracker/internal/finance"
)

func TestProjectEntry(t *testing.T) {
	accts := map[int64]finance.Account{
		1: {ID: 1, Path: "Assets:Bank:SEB", Type: finance.Assets},
		2: {ID: 2, Path: "Expenses:Food:Groceries", Type: finance.Expenses},
		3: {ID: 3, Path: "Assets:Bank:Savings", Type: finance.Assets},
		4: {ID: 4, Path: "Expenses:Transport", Type: finance.Expenses},
	}

	tests := []struct {
		name string
		e    finance.Entry
		want entryView
	}{
		{
			name: "expense, asset posting first",
			e: finance.Entry{Postings: []finance.Posting{
				{AccountID: 1, Amount: -490_00}, {AccountID: 2, Amount: 490_00},
			}},
			want: entryView{Account: "Assets:Bank:SEB", Amount: -490_00, Category: "Expenses:Food:Groceries"},
		},
		{
			name: "expense, asset posting second - order must not matter",
			e: finance.Entry{Postings: []finance.Posting{
				{AccountID: 2, Amount: 490_00}, {AccountID: 1, Amount: -490_00},
			}},
			want: entryView{Account: "Assets:Bank:SEB", Amount: -490_00, Category: "Expenses:Food:Groceries"},
		},
		{
			name: "tranfser between two assets",
			e: finance.Entry{Postings: []finance.Posting{
				{AccountID: 1, Amount: -1000_00}, {AccountID: 3, Amount: 1000_00},
			}},
			want: entryView{Account: "Assets:Bank:SEB", Amount: -1000_00, Category: "Assets:Bank:Savings"},
		},
		{
			name: "split - one asset, two expenses",
			e: finance.Entry{Postings: []finance.Posting{
				{AccountID: 1, Amount: -300_00}, {AccountID: 2, Amount: -3000_00}, {AccountID: 4, Amount: 100_00},
			}},
			want: entryView{Account: "Assets:Bank:SEB", Amount: -300_00, Category: "(split)"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := projectEntry(tt.e, accts)
			if got != tt.want {
				t.Errorf("projectEntry() = %+v, want %+v", got, tt.want)
			}
		})
	}
}
