package main

import "testing"

func TestCategorize(t *testing.T) {
	rules := []Rule{
		{PayeeContains: "ICA", Category: "Livsmedel"},
		{PayeeContains: "SJ", Category: "Transport"},
	}
	txns := []Transaction{
		{Payee: "ICA Nära Skanör"},
		{Payee: "SJ Biljett"},
		{Payee: "Spotify"},
		{Payee: "ica maxi", Category: ""},
	}

	matched := categorize(txns, rules)

	if matched != 3 {
		t.Errorf("matched = %d, want 3", matched)
	}
	if txns[0].Category != "Livsmedel" {
		t.Errorf("txns[0].Category = %q, want %q", txns[0].Category, "Livsmedel")
	}
	if txns[2].Category != "" {
		t.Errorf("txns[2] (Spotify) should un uncategorized, got %q", txns[2].Category)
	}

	// already categorised transactions should be skipped
	txns[2].Category = "Underhållning"
	matched = categorize(txns, rules)
	if matched != 0 {
		t.Errorf("second pass matched = %d, want 0 (all already categorised)", matched)
	}
}

func TestLoadRules(t *testing.T) {
	rules, err := loadRules("testdata/rules.yaml")
	if err != nil {
		t.Fatalf("loadRules: %v", err)
	}
	if len(rules) != 7 {
		t.Fatalf("got %d rules, want 7", len(rules))
	}

	// spot check
	if rules[0].PayeeContains != "ICA" {
		t.Errorf("rules[0].PayeeContains = %q, want %q", rules[0].PayeeContains, "ICA")
	}
	if rules[0].Category != "Groceries" {
		t.Errorf("rules[0].Category = %q, want %q", rules[0].Category, "Groceries")
	}

	// test missing file

	_, err = loadRules("testdata/nonexistent.yaml")
	if err == nil {
		t.Error("Expected error for missing file, got nil")
	}
}
