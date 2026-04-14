package store

import (
	"fintracker/internal/finance"
	"testing"
	"time"
)

// newTestStore creates a in-memory Store that auto-closes after test ends
func newTestStore(t *testing.T) *Store {
	t.Helper()
	store, err := NewStore(":memory:")
	if err != nil {
		t.Fatalf("creating test store: %v", err)
	}
	t.Cleanup(func() { store.Close() })
	return store
}

func TestUpsertAndLoad(t *testing.T) {
	store := newTestStore(t)

	txns := []finance.Transaction{
		{
			Date:    time.Date(2026, 1, 15, 19, 30, 0, 0, time.UTC),
			Amount:  -49_50,
			Payee:   "ICA Nära",
			Account: "SEB",
		},
		{
			Date:     time.Date(2026, 1, 17, 18, 20, 0, 0, time.UTC),
			Amount:   25000_00,
			Payee:    "Lön",
			Account:  "SEB",
			Category: "Inkomst",
		},
	}

	upserted, err := store.UpsertTransactions(txns)

	if err != nil {
		t.Fatalf("UpsertTransactions: %v", err)
	}
	if upserted != 2 {
		t.Fatalf("upserted = %d, want 2", upserted)
	}

	// Round-trip - load it back
	loaded, err := store.LoadTransactions()
	if err != nil {
		t.Fatalf("LoadTransactions: %v", err)
	}
	if len(loaded) != 2 {
		t.Fatalf("LoadTransactions got %d, expected 2", len(loaded))
	}

	if loaded[0].Payee != "ICA Nära" {
		t.Errorf("loaded[0].Payee = %q, want %q", loaded[0].Payee, "ICA Nära")
	}
	if loaded[1].Category != "Inkomst" {
		t.Errorf("loaded[1].Category = %q, want %q", loaded[1].Category, "Inkomst")
	}

	// test upsert - inserting same transaction again

	upserted, err = store.UpsertTransactions(txns[:1])
	if err != nil {
		t.Fatalf("UpsertTransactions transaction 0: %v", err)
	}
	if upserted != 0 {
		t.Errorf("upserted = %d, want 0", upserted)
	}

	// modify a transaction and upsert
	txns[0].Category = "Groceries"
	upserted, err = store.UpsertTransactions(txns[:1])
	if err != nil {
		t.Fatalf("UpsertTransactions with changed data: %v", err)
	}
	if upserted != 1 {
		t.Fatalf("upserted = %d, want 1", upserted)
	}
	// load the transaction and verify correct data stored

	loaded, err = store.LoadTransactions()
	if err != nil {
		t.Fatalf("LoadTransactions: %v", err)
	}
	if len(loaded) != 2 {
		t.Errorf("LoadTransactions got %d, want 2", len(loaded))
	}
	// check data
	if loaded[0].Category != "Groceries" {
		t.Errorf("loaded[0].Category = %q, want %q", loaded[0].Category, "Groceries")
	}

	// update category of existing and upsert
	txns[0].Category = "livsmedel"
	upserted, err = store.UpsertTransactions(txns[:1])
	if err != nil {
		t.Fatalf("UpsertTransactions with changed data: %v", err)
	}
	if upserted != 1 {
		t.Fatalf("upserted = %d, want 1", upserted)
	}
	// load the transaction and verify correct data stored

	loaded, err = store.LoadTransactions()
	if err != nil {
		t.Fatalf("LoadTransactions: %v", err)
	}
	if len(loaded) != 2 {
		t.Errorf("LoadTransactions got %d, want 2", len(loaded))
	}
	// check data
	if loaded[0].Category != "livsmedel" {
		t.Errorf("loaded[0].Category = %q, want %q", loaded[0].Category, "livsmedel")
	}

	// for transactions with existing category, setting it to empty string should preserve old category
	txns[0].Category = ""
	upserted, err = store.UpsertTransactions(txns[:1])
	if err != nil {
		t.Fatalf("UpsertTransactions with changed data: %v", err)
	}
	if upserted != 1 {
		t.Fatalf("upserted = %d, want 1", upserted)
	}
	// load the transaction and verify correct data stored

	loaded, err = store.LoadTransactions()
	if err != nil {
		t.Fatalf("LoadTransactions: %v", err)
	}
	if len(loaded) != 2 {
		t.Errorf("LoadTransactions got %d, want 2", len(loaded))
	}
	// check data
	if loaded[0].Category != "livsmedel" {
		t.Errorf("loaded[0].Category = %q, want %q", loaded[0].Category, "livsmedel")
	}

}

func TestInsertAndLoadAccount(t *testing.T) {
	s := newTestStore(t)

	acc := finance.Account{
		Path:     "Assets:Bank:SEB",
		Type:     finance.Assets,
		Currency: "SEK",
	}

	id, err := s.InsertAccount(acc)
	if err != nil {
		t.Fatalf("InsertAccount: %v", err)
	}
	if id == 0 {
		t.Fatal("expected non-zero ID")
	}

	got, err := s.LoadAccounts()
	if err != nil {
		t.Fatalf("LoadAccounts: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 account, got %d", len(got))
	}
	if got[0].Path != "Assets:Bank:SEB" {
		t.Errorf("path = %q, want %q", got[0].Path, "Assets:Bank:SEB")
	}
	if got[0].Type != finance.Assets {
		t.Errorf("type = %q, want %q", got[0].Type, finance.Assets)
	}
	if got[0].ID != id {
		t.Errorf("ID = %d, want %d", got[0].ID, id)
	}

}

func TestInsertAndLoadEntry(t *testing.T) {
	s := newTestStore(t)

	// Set up accounts first
	sebID, err := s.InsertAccount(finance.Account{
		Path: "Assets:Bank:SEB", Type: finance.Assets, Currency: "SEK",
	})
	if err != nil {
		t.Fatalf("InsertAccount SEB: %v", err)
	}
	grocID, err := s.InsertAccount(finance.Account{
		Path: "Expenses:Food:Groceries", Type: finance.Expenses, Currency: "SEK",
	})
	if err != nil {
		t.Fatalf("InsertAccount Groceries: %v", err)
	}

	entry := finance.Entry{
		Date:     time.Date(2026, 4, 14, 10, 0, 10, 10, time.UTC),
		Payee:    "Malmborgs",
		RawPayee: "Ica Malmborgs Eriklust",
		Memo:     "Veckans mat",
		Tags:     []string{"fest", "april"},
		Postings: []finance.Posting{
			{AccountID: grocID, Amount: 649_50, Currency: "SEK"},
			{AccountID: sebID, Amount: -649_50, Currency: "SEK"},
		},
	}

	entryID, err := s.InsertEntry(entry)
	if err != nil {
		t.Fatalf("InsertEntry: %v", err)
	}
	if entryID == 0 {
		t.Fatalf("expected non-zero entry ID")
	}

	entries, err := s.LoadEntries()
	if err != nil {
		t.Fatalf("LoadEntries: %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}

	got := entries[0]
	if got.Payee != "Malmborgs" {
		t.Errorf("payee = %q, want %q", got.Payee, "Malmborgs")
	}
	if got.Memo != "Veckans mat" {
		t.Errorf("memo = %q, want %q", got.Memo, "Veckans mat")
	}
	if len(got.Tags) != 2 {
		t.Fatalf("tags count = %d, want 2", len(got.Tags))
	}
	if len(got.Postings) != 2 {
		t.Fatalf("postings count = %d, want 2", len(got.Postings))
	}
	if err = got.Validate(); err != nil {
		t.Errorf("loaded entry doesn't validate: %v", err)
	}
}
