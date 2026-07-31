package store

import (
	"errors"
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

func TestInsertAndLoadTransaction(t *testing.T) {
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

	tx := finance.Transaction{
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

	txID, err := s.InsertTransaction(tx)
	if err != nil {
		t.Fatalf("InsertTransaction: %v", err)
	}
	if txID == 0 {
		t.Fatalf("expected non-zero transaction ID")
	}

	txns, err := s.LoadTransactions()
	if err != nil {
		t.Fatalf("LoadTransactions: %v", err)
	}

	if len(txns) != 1 {
		t.Fatalf("expected 1 transaction, got %d", len(txns))
	}

	got := txns[0]
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
		t.Errorf("loaded transaction doesn't validate: %v", err)
	}
}

func TestInsertAccountDuplicate(t *testing.T) {
	s := newTestStore(t)
	acc := finance.Account{Path: "Assets:Bank:SEB", Type: finance.Assets, Currency: "SEK"}

	if _, err := s.InsertAccount(acc); err != nil {
		t.Fatalf("first insert: %v", err)
	}

	_, err := s.InsertAccount(acc)
	if err == nil {
		t.Fatalf("expected error on duplicate path, got nil")
	}
}

func TestInsertTransactionValidationFail(t *testing.T) {
	s := newTestStore(t)
	bad := finance.Transaction{
		Date: time.Now(),
		Postings: []finance.Posting{
			{AccountID: 1, Amount: 100_00, Currency: "SEK"},
			// deliberately unbalanced
		},
	}
	_, err := s.InsertTransaction(bad)
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
}

func TestInsertTransactionForeignKeyViolation(t *testing.T) {
	s := newTestStore(t)

	e := finance.Transaction{
		Date: time.Now(),
		Postings: []finance.Posting{
			{AccountID: 123, Amount: 100_00, Currency: "SEK"},
			{AccountID: 124, Amount: -100_00, Currency: "SEK"},
		},
	}
	_, err := s.InsertTransaction(e)
	if err == nil {
		t.Fatal("expected account_id foreign key to fail, got nil")
	}

}

func TestInsertTransactionDuplicate(t *testing.T) {
	s := newTestStore(t)

	// create an account reference
	accID, err := s.InsertAccount(finance.Account{
		Path: "Expenses:Food", Type: finance.Expenses, Currency: "SEK",
	})
	if err != nil {
		t.Fatalf("InsertAccount: %v", err)
	}

	e := finance.Transaction{
		Date:       time.Now(),
		Payee:      "Ica",
		ImportHash: "test-hash-1",
		Postings: []finance.Posting{
			{AccountID: accID, Amount: 100_00, Currency: "SEK"},
			{AccountID: accID, Amount: -100_00, Currency: "SEK"},
		},
	}
	_, err = s.InsertTransaction(e)
	if err != nil {
		t.Errorf("error inserting transaction: %v", err)
	}

	// insert same duplicate transaction
	_, err = s.InsertTransaction(e)
	if !errors.Is(err, ErrDuplicateTransaction) {
		t.Error("expected ErrDuplicateTransaction")
	}

	// transactions without ImportHash don't cause ErrDuplicateTransaction

	e2 := finance.Transaction{
		Date:       time.Now(),
		Payee:      "Ica",
		ImportHash: "",
		Postings: []finance.Posting{
			{AccountID: accID, Amount: 100_00, Currency: "SEK"},
			{AccountID: accID, Amount: -100_00, Currency: "SEK"},
		},
	}
	_, err2 := s.InsertTransaction(e2)
	if err2 != nil {
		t.Errorf("unexpected error for duplicate transaction with nil importhash")
	}

}

func TestInsertAndLoadPayeeResult(t *testing.T) {
	s := newTestStore(t)

	// create an account reference
	accID, err := s.InsertAccount(finance.Account{
		Path: "Expenses:Food", Type: finance.Expenses, Currency: "SEK",
	})
	if err != nil {
		t.Fatalf("InsertAccount: %v", err)
	}

	// Rule with acccount
	_, err = s.InsertPayeeRule(finance.PayeeRule{
		Pattern:          "ICA",
		NormalizedPayee:  "ICA",
		DefaultAccountID: &accID,
		Priority:         10,
	})
	if err != nil {
		t.Fatalf("InsertPayeeRule: %v", err)
	}

	// Rule without account (nil DefaultAccountID)
	_, err = s.InsertPayeeRule(finance.PayeeRule{
		Pattern:         "SPOTIFY",
		NormalizedPayee: "Spotify",
		Priority:        20,
	})
	if err != nil {
		t.Fatalf("InsertPayeeRule: %v", err)
	}

	rules, err := s.LoadPayeeRules()
	if err != nil {
		t.Fatalf("LoadPayeeRules: %v", err)
	}
	if len(rules) != 2 {
		t.Fatalf("got %d rules, want 2", len(rules))
	}

	// Ordered by priority - ICA (10) first
	if rules[0].Pattern != "ICA" {
		t.Errorf("rules[0].Pattern = %q, want %q", rules[0].Pattern, "ICA")
	}
	if rules[0].DefaultAccountID == nil || *rules[0].DefaultAccountID != accID {
		t.Errorf("rules[0].DefaultAccountID = %v, want %d", rules[0].DefaultAccountID, accID)
	}
	if rules[1].DefaultAccountID != nil {
		t.Errorf("rules[1].DefaultAccountID = %v, want nil", rules[1].DefaultAccountID)
	}

}

func TestEnsureAccount(t *testing.T) {
	s := newTestStore(t)

	// garbage part errors
	_, err := s.EnsureAccount("bnk:SEB")
	if err == nil {
		t.Error("incorrect path should error")
	}
	// idempotency test - calling something twise - same result (ID) and create only one row
	id, err := s.EnsureAccount("Assets:SEB")
	if err != nil {
		t.Fatalf("s.EnsureAccount %v", err)
	}
	id2, err := s.EnsureAccount("Assets:SEB")
	if err != nil {
		t.Fatalf("s.EnsureAccount called twice err: %v", err)
	}
	if id != id2 {
		t.Errorf("s.EnsureAccount is not idempotent id (%q) != id2 (%q)", id, id2)
	}
	// assert inferred type round-trips via load accounts
	accs, err := s.LoadAccounts()
	if err != nil {
		t.Fatalf("s.LoadAccounts() %v", err)
	}

	if accs[0].ID != id {
		t.Errorf("accs[0].ID got %q want %q", accs[0].ID, id)
	}
	if accs[0].Type != finance.Assets {
		t.Errorf("accs[0].Type got %q want 'Assets'", accs[0].Type)
	}

}

func TestUpdatePosting(t *testing.T) {
	s := newTestStore(t)
	// Set up accounts first
	sebID, err := s.InsertAccount(finance.Account{
		Path: "Assets:Bank:SEB", Type: finance.Assets, Currency: "SEK",
	})
	if err != nil {
		t.Fatalf("InsertAccount SEB: %v", err)
	}
	contraAccID, err := s.InsertAccount(finance.Account{
		Path: "Expenses:Food:Groceries", Type: finance.Expenses, Currency: "SEK",
	})
	if err != nil {
		t.Fatalf("InsertAccount Groceries: %v", err)
	}

	tx := finance.Transaction{
		Date:     time.Date(2026, 4, 14, 10, 0, 10, 10, time.UTC),
		Payee:    "Malmborgs",
		RawPayee: "Ica Malmborgs Eriklust",
		Memo:     "Veckans mat",
		Tags:     []string{"fest", "april"},
		Postings: []finance.Posting{
			{AccountID: sebID, Amount: 649_50, Currency: "SEK"}, // will update and test
			{AccountID: sebID, Amount: -649_50, Currency: "SEK"},
		},
	}

	txID, err := s.InsertTransaction(tx)
	if err != nil {
		t.Fatalf("InsertTransaction: %v", err)
	}
	if txID == 0 {
		t.Fatalf("expected non-zero transaction ID")
	}

	txns, err := s.LoadTransactions()
	if err != nil {
		t.Fatalf("load transactions: %v", err)
	}
	if len(txns) != 1 {
		t.Errorf("expected 1 transaction got %d", len(txns))
	}
	contraPostID := txns[0].Postings[1].ID
	err = s.UpdatePosting(contraPostID, contraAccID)
	if err != nil {
		t.Fatalf("update posting %v", err)
	}
	// load transactions again
	txns, err = s.LoadTransactions()
	if err != nil {
		t.Fatalf("load transactions: %v", txns)
	}

	posting := txns[0].Postings[1]
	// verify posting was updated to correct contra account
	if posting.AccountID != contraAccID {
		t.Errorf("posting not updated to correct account")
	}

	err = s.UpdatePosting(999999, contraAccID)
	if err == nil {
		t.Fatalf("expected non-nil error for updating non-existing posting")
	}

}

func TestRepointPostings(t *testing.T) {
	s := newTestStore(t)

	// Set up accounts first
	sebID, err := s.InsertAccount(finance.Account{
		Path: "Assets:Bank:SEB", Type: finance.Assets, Currency: "SEK",
	})
	if err != nil {
		t.Fatalf("InsertAccount SEB: %v", err)
	}
	contraAccID, err := s.InsertAccount(finance.Account{
		Path: "Expenses:Food:Groceries", Type: finance.Expenses, Currency: "SEK",
	})
	if err != nil {
		t.Fatalf("InsertAccount Groceries: %v", err)
	}

	tx := finance.Transaction{
		Date:     time.Date(2026, 4, 14, 10, 0, 10, 10, time.UTC),
		Payee:    "Malmborgs",
		RawPayee: "Ica Malmborgs Eriklust",
		Memo:     "Veckans mat",
		Tags:     []string{"fest", "april"},
		Postings: []finance.Posting{
			{AccountID: sebID, Amount: 649_50, Currency: "SEK"}, // will update and test
			{AccountID: sebID, Amount: -649_50, Currency: "SEK"},
		},
	}

	txID, err := s.InsertTransaction(tx)
	if err != nil {
		t.Fatalf("InsertTransaction: %v", err)
	}
	if txID == 0 {
		t.Fatalf("expected non-zero transaction ID")
	}

	txns, err := s.LoadTransactions()
	if err != nil {
		t.Fatalf("load transactions: %v", err)
	}
	if len(txns) != 1 {
		t.Errorf("expected 1 transaction got %d", len(txns))
	}
	contraPostID := txns[0].Postings[1].ID

	// re-point the second posting to Groceries
	if err := s.RepointPostings([]int64{contraPostID}, contraAccID); err != nil {
		t.Fatalf("RepointPostings: %v", err)
	}

	txns, err = s.LoadTransactions()
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if got := txns[0].Postings[1].AccountID; got != contraAccID {
		t.Errorf("re-pointed posting account = %d, want %d (Groceries)", got, contraAccID)
	}
	if got := txns[0].Postings[0].AccountID; got != sebID {
		t.Errorf("untouched posting account = %d, want %d (unchanged)", got, sebID)
	}

	// atomicity: a batch with one bad ID must roll back entirely
	t.Run("rolls back on bad id", func(t *testing.T) {
		if err := s.RepointPostings([]int64{contraPostID, 999999}, sebID); err == nil {
			t.Fatal("want error for non-existent posting ID, got nil")
		}
		txns, err := s.LoadTransactions()
		if err != nil {
			t.Fatalf("reload: %v", err)
		}
		if got := txns[0].Postings[1].AccountID; got != contraAccID {
			t.Errorf("posting moved despite rollback: account = %d, want %d", got, contraAccID)
		}
	})

}

func TestReviewTransaction(t *testing.T) {
	s := newTestStore(t)
	// insert a source acc, a transaction with two balanced postings (source + a contra on some account), cleared = false
	sebID, err := s.InsertAccount(finance.Account{
		Path: "Assets:Bank:SEB", Type: finance.Assets, Currency: "SEK",
	})
	if err != nil {
		t.Fatalf("InsertAccount SEB: %v", err)
	}

	tx := finance.Transaction{
		Date:     time.Date(2026, 4, 14, 10, 0, 10, 10, time.UTC),
		Payee:    "Malmborgs",
		RawPayee: "Ica Malmborgs Eriklust",
		Memo:     "Veckans mat",
		Tags:     []string{"fest", "april"},
		Postings: []finance.Posting{
			{AccountID: sebID, Amount: 649_50, Currency: "SEK"}, // will update and test
			{AccountID: sebID, Amount: -649_50, Currency: "SEK"},
		},
	}

	txID, err := s.InsertTransaction(tx)
	if err != nil {
		t.Fatalf("InsertTransaction: %v", err)
	}
	if txID == 0 {
		t.Fatalf("expected non-zero transaction ID")
	}

	// LoadTransactions -> grab tx id and contra posting's ID
	txns, err := s.LoadTransactions()
	if err != nil {
		t.Fatalf("load transactions: %v", err)
	}
	if len(txns) != 1 {
		t.Errorf("expected 1 transaction got %d", len(txns))
	}
	contraPostingID := txns[0].Postings[1].ID

	// call ReviewTransaction(txID, contraPostingID, "Expenses:Food:Groceries") - a path that doesn't yet exist
	if err := s.ReviewTransaction(txID, contraPostingID, "Expenses:Food:Groceries"); err != nil {
		t.Fatalf("review transaction failed: %v", err)
	}
	// reload and assert
	txns, err = s.LoadTransactions()
	if err != nil {
		t.Fatalf("load transactions: %v", err)
	}
	if len(txns) != 1 {
		t.Errorf("expected 1 transaction got %d", len(txns))
	}
	if !txns[0].Cleared {
		t.Errorf("ReviewTransaction did not clear transaction")
	}
	// contraPosting's account should exist and  resolve to "Expenses:Food:Groceries"
	accounts, err := s.LoadAccounts()
	if err != nil {
		t.Fatalf("could not load accounts %v", err)
	}
	if len(accounts) != 2 {
		t.Errorf("have %d accounts, want 2", len(accounts))
	}
	contraPostingAccID := txns[0].Postings[1].AccountID
	var contraPostingAcc *finance.Account = nil
	for _, acc := range accounts {
		if acc.ID == contraPostingAccID {
			contraPostingAcc = &acc
		}
	}
	if contraPostingAcc == nil {
		t.Errorf("contra posting account could not be found")
	}
	if contraPostingAcc.Path != "Expenses:Food:Groceries" {
		t.Errorf("contra posting's account resolves to %q want %q", contraPostingAcc.Path, "Expenses:Food:Groceries")
	}

	t.Run("rolls back on bad posting", func(t *testing.T) {
		// Calling ReviewTransaction on valid transaction ID, but inexistent posting, should atomically roll back
		err = s.ReviewTransaction(txID, 9999 /* nonexistent posting */, "Expenses:New:Rollback")
		if err == nil {
			t.Errorf("ReviewTransaction should return an error for nonexistent posting")
		}
		// check that no spurious accounts were created
		accounts, err = s.LoadAccounts()
		if err != nil {
			t.Fatalf("could not load accounts %v", err)
		}
		if len(accounts) != 2 {
			t.Errorf("have %d accounts, want 2", len(accounts))
		}
		for _, acc := range accounts {
			if acc.Path == "Expenses:New:Rollback" {
				t.Errorf("ReviewTransaction is not atomic, found %q in accounts", "Expenses:New:Rollback")
			}
		}
	})

}

func TestEnsurePayeeRule(t *testing.T) {
	s := newTestStore(t)
	accId, err := s.InsertAccount(finance.Account{
		Path: "Assets:Bank:SEB", Type: finance.Assets, Currency: "SEK",
	})

	if err != nil {
		t.Fatalf("insert account: %v", err)
	}

	created, err := s.EnsurePayeeRule(finance.PayeeRule{Pattern: "ICA", DefaultAccountID: &accId})
	if err != nil {
		t.Errorf("EnsurePayeeRule: %v", err)
	}
	if !created {
		t.Errorf("payee rule not `created`")
	}

	created, err = s.EnsurePayeeRule(finance.PayeeRule{Pattern: "ICA", DefaultAccountID: &accId})
	if err != nil {
		t.Errorf("EnsurePayeeRule: %v", err)
	}
	if created {
		t.Errorf("payee rule `created` %t, want false", created)
	}

	rules, err := s.LoadPayeeRules()
	if err != nil {
		t.Fatalf("could not load payee rules: %v", err)
	}
	if len(rules) != 1 {
		t.Errorf("have %d rules, want %d", len(rules), 1)
	}

	t.Run("no default account id", func(t *testing.T) {
		created, err = s.EnsurePayeeRule(finance.PayeeRule{Pattern: "ICA"})
		if err == nil {
			t.Errorf("want err, got non")
		}

	})
}
