package finance_test

import (
	"fintracker/internal/finance"
	"testing"
)

func TestEntryValidate(t *testing.T) {
	sek := func(n int64) finance.Posting {
		return finance.Posting{Amount: finance.Öre(n), Currency: "SEK"}
	}
	inr := func(n int64) finance.Posting {
		return finance.Posting{Amount: finance.Öre(n), Currency: "INR"}
	}

	tests := []struct {
		name    string
		entry   finance.Entry
		wantErr bool
	}{
		{
			name:    "balanced two postings",
			entry:   finance.Entry{Postings: []finance.Posting{sek(890_00), sek(-890_00)}},
			wantErr: false,
		},
		{
			name:    "unbalanced",
			entry:   finance.Entry{Postings: []finance.Posting{sek(890_00), sek(-800_00)}},
			wantErr: true,
		},
		{
			name:    "single posting",
			entry:   finance.Entry{Postings: []finance.Posting{inr(1_890_00)}},
			wantErr: true,
		},
		{
			name:    "no postings",
			entry:   finance.Entry{},
			wantErr: true,
		},
		{
			name: "multi-currency both balanced",
			entry: finance.Entry{Postings: []finance.Posting{
				sek(500_00), sek(-500_00), inr(100_00), inr(-100_00)}},
			wantErr: false,
		},
		{
			name: "multi-currency one leg unbalanced",
			entry: finance.Entry{Postings: []finance.Posting{
				sek(500_00), sek(-500_00), inr(100_00), inr(-90_00)}},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.entry.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
