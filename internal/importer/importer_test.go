package importer

import (
	"fintracker/internal/finance"
	"strings"
	"testing"
	"time"
)

func ptr(n int64) *int64 { return &n }

func TestImport(t *testing.T) {
	input := strings.NewReader(
		"2026-04-01;-490,00; ICA Skanör\n" +
			"2026-04-02;-99,00;SPOTIFY AB\n" +
			"2026-04-03;-250,00;OKÄND BUTIK\n",
	)

	rules := []finance.PayeeRule{
		{Pattern: "ICA", NormalizedPayee: "ICA", DefaultAccountID: ptr(10), Priority: 0},
		{Pattern: "SPOTIFY", NormalizedPayee: "Spotify", DefaultAccountID: ptr(11), Priority: 0},
	}
	result, err := Import(input, SEBFormat{}, 1, rules)
	if err != nil {
		t.Fatalf("Import: %v", err)
	}

	if len(result.Entries) != 2 {
		t.Errorf("got %d entries, want 2", len(result.Entries))
	}
	if len(result.Unmatched) != 1 {
		t.Errorf("got %d unmatched entries want 1", len(result.Unmatched))
	}

	if result.Entries[0].Payee != "ICA" {
		t.Errorf("Entries[0].payee got %q want %q", result.Entries[0].Payee, "ICA")
	}

	ps := result.Entries[0].Postings
	if ps[0].Amount != -490_00 {
		t.Errorf("source posting got %v want %v", ps[0].Amount, finance.Öre(-490_00))
	}
	if ps[1].Amount != 490_00 {
		t.Errorf("counter posting got %v want %v", ps[1].Amount, finance.Öre(490_00))
	}

	if result.Entries[0].Validate() != nil {
		t.Errorf("posting should Validate successfully")
	}
}

func TestDefaultRules(t *testing.T) {
	rules, err := DefaultRules()
	if err != nil {
		t.Fatalf("DefaultRules: %v", err)
	}
	if len(rules) < 2 {
		t.Fatalf("expected at least 2 default rules, got %d", len(rules))
	}
	for _, r := range rules {
		if r.Pattern == "" {
			t.Errorf("rules %q has empty pattern", r.NormalizedPayee)
		}
	}
}

func TestPlaceholderEntries(t *testing.T) {
	rows := []RawRow{
		{Date: time.Date(2026, 4, 3, 0, 0, 0, 0, time.UTC), Amount: -250_00, RawPayee: "OKÄND butik"},
		{Date: time.Date(2026, 4, 25, 0, 0, 0, 0, time.UTC), Amount: 25_000_00, RawPayee: "Arbetsgivare"},
	}

	entries := PlaceholderEntries(rows, 1, 7)

	if len(entries) != len(rows) {
		t.Fatalf("got %d entries, want %d", len(entries), len(rows))
	}

	for i, e := range entries {
		if err := e.Validate(); err != nil {
			t.Fatalf("entry %d: Validate: %v", i, err)
		}
		if e.Cleared {
			t.Errorf("entry %d: should not be cleared", i)
		}
		if e.Payee != "" {
			t.Errorf("entry %d: Payee = %q, want empty", i, e.Payee)
		}
		if e.RawPayee != rows[i].RawPayee {
			t.Errorf("entry %d: RawPayee = %q, want %q", i, e.RawPayee, rows[i].RawPayee)
		}
		if e.Postings[0].AccountID != 1 || e.Postings[1].AccountID != 7 {
			t.Errorf("entry %d: account IDs = %d,%d, want 1,7",
				i, e.Postings[0].AccountID, e.Postings[1].AccountID)
		}
		if e.Postings[0].Amount != rows[i].Amount {
			t.Errorf("entry %d: source amount = %v, want %v", i, e.Postings[0].Amount, e.Postings[i].Amount)
		}
	}
}

func TestStampHashes(t *testing.T) {
	rows := []RawRow{
		{Date: time.Date(2026, 4, 12, 0, 0, 0, 0, time.UTC), Amount: -250_00, RawPayee: "Espresso house"},
		{Date: time.Date(2026, 4, 12, 0, 0, 0, 0, time.UTC), Amount: -250_00, RawPayee: "Espresso house"},
		{Date: time.Date(2026, 4, 14, 0, 0, 0, 0, time.UTC), Amount: -250_00, RawPayee: "Espresso house"}, // Distinct

	}
	stampHashes(rows)

	// every hash non empty
	// two identical rows get different hashes
	if rows[0].Hash == rows[1].Hash {
		t.Fatalf("different hashes expected for rows[0].Hash and rows[1].Hash: %q == %q", rows[0].Hash, rows[1].Hash)
	}
	if rows[1].Hash == rows[2].Hash {
		t.Error("distinct row should get different hash")
	}
	// hash reproduces
	freshRows := []RawRow{
		{Date: time.Date(2026, 4, 12, 0, 0, 0, 0, time.UTC), Amount: -250_00, RawPayee: "Espresso house"},
		{Date: time.Date(2026, 4, 12, 0, 0, 0, 0, time.UTC), Amount: -250_00, RawPayee: "Espresso house"},
		{Date: time.Date(2026, 4, 14, 0, 0, 0, 0, time.UTC), Amount: -250_00, RawPayee: "Espresso house"},
	}
	stampHashes(freshRows)

	if rows[0].Hash != freshRows[0].Hash {
		t.Errorf("hash did not reproduce %q != %q", rows[0].Hash, freshRows[0].Hash)
	}
}
