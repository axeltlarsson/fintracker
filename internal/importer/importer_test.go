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
			"2026-04-03;-250,00;OKÄND BUTIK\n" +
			"2026-06-03;-250,00;temple of spices\n",
	)

	rules := []finance.PayeeRule{
		{Pattern: "ICA", NormalizedPayee: "ICA", DefaultAccountID: ptr(10), Priority: 0},
		{Pattern: "SPOTIFY", NormalizedPayee: "Spotify", DefaultAccountID: ptr(11), Priority: 0},
		{Pattern: "TEMPLE OF SPICES", NormalizedPayee: "Temple of Spices", DefaultAccountID: nil},
	}
	result, err := Import(input, SEBFormat{}, 1, rules)
	if err != nil {
		t.Fatalf("Import: %v", err)
	}

	if len(result.Transactions) != 2 {
		t.Errorf("got %d transactions, want 2", len(result.Transactions))
	}
	if len(result.Unmatched) != 2 {
		t.Errorf("got %d unmatched transactions want 2", len(result.Unmatched))
	}

	if result.Transactions[0].Payee != "ICA" {
		t.Errorf("Transactions[0].payee got %q want %q", result.Transactions[0].Payee, "ICA")
	}

	ps := result.Transactions[0].Postings
	if ps[0].Amount != -490_00 {
		t.Errorf("source posting got %v want %v", ps[0].Amount, finance.Öre(-490_00))
	}
	if ps[1].Amount != 490_00 {
		t.Errorf("counter posting got %v want %v", ps[1].Amount, finance.Öre(490_00))
	}

	if result.Transactions[0].Validate() != nil {
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

func TestPlaceholderTransactions(t *testing.T) {
	rows := []RawRow{
		{Date: time.Date(2026, 4, 3, 0, 0, 0, 0, time.UTC), Amount: -250_00, RawPayee: "OKÄND butik"},
		{Date: time.Date(2026, 4, 25, 0, 0, 0, 0, time.UTC), Amount: 25_000_00, RawPayee: "Arbetsgivare"},
	}

	txns := PlaceholderTransactions(rows, 1, 7)

	if len(txns) != len(rows) {
		t.Fatalf("got %d transations, want %d", len(txns), len(rows))
	}

	for i, tx := range txns {
		if err := tx.Validate(); err != nil {
			t.Fatalf("transaction %d: Validate: %v", i, err)
		}
		if tx.Cleared {
			t.Errorf("transaction %d: should not be cleared", i)
		}
		if tx.Payee != "" {
			t.Errorf("transaction %d: Payee = %q, want empty", i, tx.Payee)
		}
		if tx.RawPayee != rows[i].RawPayee {
			t.Errorf("transaction %d: RawPayee = %q, want %q", i, tx.RawPayee, rows[i].RawPayee)
		}
		if tx.Postings[0].AccountID != 1 || tx.Postings[1].AccountID != 7 {
			t.Errorf("transaction %d: account IDs = %d,%d, want 1,7",
				i, tx.Postings[0].AccountID, tx.Postings[1].AccountID)
		}
		if tx.Postings[0].Amount != rows[i].Amount {
			t.Errorf("transaction %d: source amount = %v, want %v", i, tx.Postings[0].Amount, tx.Postings[i].Amount)
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
