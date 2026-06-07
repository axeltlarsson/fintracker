package importer

import (
	"fintracker/internal/finance"
	"strings"
	"testing"
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
