package store

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"fintracker/internal/finance"
	_ "modernc.org/sqlite"
)

// returned by InsertTransaction when transaction with same hash already exists
var ErrDuplicateTransaction = errors.New("duplicate transaction")

type Store struct {
	db *sql.DB
}

func NewStore(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)

	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}
	// sqlite performs better with these params
	_, err = db.Exec(`
	PRAGMA journal_mode=WAL;
	PRAGMA foreign_keys=ON;
	`)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("setting pragmas: %w", err)
	}
	if err := migrate(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrating schema: %w", err)
	}

	return &Store{db: db}, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

// each migration runs inside a transaction
// The function receives the *sql.Tx to execute statements against
var migrations = []func(*sql.Tx) error{
	// 0 -> 1
	func(tx *sql.Tx) error {
		_, err := tx.Exec(`
			create table accounts (
				id integer primary key autoincrement,
				path text not null unique, -- right choice - other choices are less ergonomic
				type text not null,
				currency text not null default 'SEK',
				opened_at text, -- why text?
				closed_at text
			);
			create table transactions (
				id integer primary key autoincrement,
				date text not null,
				payee text not null default '',
				raw_payee text not null default '',
				memo text not null default '',
				cleared integer not null default 0 -- sqlite has no bool type
			);
			create table postings (
				id integer primary key autoincrement,
				transaction_id integer not null references transactions(id) on delete cascade,
				account_id integer not null references accounts(id) on delete restrict,
				amount integer not null,
				currency text not null default 'SEK'
			);
			create table transaction_tags (
				transaction_id integer not null references transactions(id) on delete cascade,
				tag text not null,
				primary key (transaction_id, tag)
			);
		`)
		return err
	},

	// 1 -> 2
	func(tx *sql.Tx) error {
		_, err := tx.Exec(`
			create table payee_rules (
				id integer primary key autoincrement,
				pattern text not null,
				normalized_payee text not null default '',
				default_account_id integer references accounts(id) on delete set null,
				priority integer not null default 0
			);
		`)
		return err
	},

	// 2 → 3
	func(tx *sql.Tx) error {
		_, err := tx.Exec(`
		alter table transactions add column import_hash text;
		create unique index idx_transaction_import_hash on transactions(import_hash);
		`)
		return err
	},
}

func migrate(db *sql.DB) error {
	var current int
	if err := db.QueryRow("PRAGMA user_version").Scan(&current); err != nil {
		return fmt.Errorf("reading schema version: %w", err)
	}

	for i := current; i < len(migrations); i++ {
		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("beginning migration %d: %w", i+1, err)
		}
		if err := migrations[i](tx); err != nil {
			tx.Rollback()
			return fmt.Errorf("migration %d: %w", i+1, err)
		}
		// PRAGMA cannot be parameterized, but i+1 is always an int we control
		if _, err := tx.Exec(fmt.Sprintf("PRAGMA user_version = %d", i+1)); err != nil {
			tx.Rollback()
			return fmt.Errorf("setting version %d: %w", i+1, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("committing migration: %d %w", i+1, err)
		}
	}
	return nil
}

func (s *Store) InsertAccount(acc finance.Account) (int64, error) {
	return insertAccount(s.db, acc)
}

func (s *Store) LoadAccounts() ([]finance.Account, error) {
	rows, err := s.db.Query(`
	select id, path, type, currency, opened_at, closed_at
	from accounts
	order by path
	`)
	if err != nil {
		return nil, fmt.Errorf("querying accounts: %w", err)
	}

	defer rows.Close()

	var accounts []finance.Account
	for rows.Next() {
		var a finance.Account
		if err := rows.Scan(&a.ID, &a.Path, &a.Type, &a.Currency, &a.OpenedAt, &a.ClosedAt); err != nil {
			return nil, fmt.Errorf("scanning accounts: %w", err)
		}
		accounts = append(accounts, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating accounts: %w", err)
	}

	return accounts, nil
}

func (s *Store) InsertTransaction(e finance.Transaction) (int64, error) {
	if err := e.Validate(); err != nil {
		return 0, fmt.Errorf("invalid transaction: %w", err)
	}

	tx, err := s.db.Begin()
	if err != nil {
		return 0, fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback()

	// 1 Insert transaction itself
	result, err := tx.Exec(`
		insert into transactions (date, payee, raw_payee, memo, cleared, import_hash)
		values (?, ?, ?, ?, ?, ?)
		on conflict (import_hash) do nothing
	`, e.Date.Format("2006-01-02"), e.Payee, e.RawPayee, e.Memo, e.Cleared,
		sql.NullString{String: e.ImportHash, Valid: e.ImportHash != ""})
	if err != nil {
		return 0, fmt.Errorf("inserting transaction: %w", err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("checking rows affected: %w", err)
	}
	if n == 0 {
		return 0, fmt.Errorf("transaction %s/%s: %w", e.Date.Format("2006-01-02"), e.RawPayee, ErrDuplicateTransaction)
	}
	txnID, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("getting transaction ID: %w", err)
	}

	// 2 insert postings
	postStmt, err := tx.Prepare(`
		insert into postings (transaction_id, account_id, amount, currency)
		values (?, ?, ?, ?)
	`)
	if err != nil {
		return 0, fmt.Errorf("preparing postings insert: %w", err)
	}
	defer postStmt.Close() // must close prepared statement resource

	for _, p := range e.Postings {
		if _, err := postStmt.Exec(txnID, p.AccountID, int64(p.Amount), p.Currency); err != nil {
			return 0, fmt.Errorf("inserting posting: %w", err)
		}
	}

	// 3. Insert tags
	for _, tag := range e.Tags {
		if _, err := tx.Exec(`
			insert into transaction_tags (transaction_id, tag) values (?, ?)
	        `, txnID, tag); err != nil {
			return 0, fmt.Errorf("inserting tag %q: %w", tag, err)

		}
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("committing transaction: %w", err)
	}

	return txnID, nil

}

func (s *Store) UpdateTransaction(e finance.Transaction) error {
	_, err := s.db.Exec(`
		UPDATE transactions
		SET payee = ?, memo = ?, cleared = ?
		WHERE id = ?
	`, e.Payee, e.Memo, e.Cleared, e.ID)
	if err != nil {
		return fmt.Errorf("updating transaction %d: %w", e.ID, err)
	}
	return nil
}

func (s *Store) LoadTransactions() ([]finance.Transaction, error) {
	// 1. Load all transactions
	rows, err := s.db.Query(`
		select id, date, payee, raw_payee, memo, cleared, import_hash
		from transactions
		order by date, id
	`)
	if err != nil {
		return nil, fmt.Errorf("querying transactions: %w", err)
	}
	defer rows.Close()

	txnMap := make(map[int64]*finance.Transaction)
	var txnOrder []int64 // preserve query order

	for rows.Next() {
		var e finance.Transaction
		var dateStr string
		var hash sql.NullString
		if err := rows.Scan(&e.ID, &dateStr, &e.Payee, &e.RawPayee, &e.Memo, &e.Cleared, &hash); err != nil {
			return nil, fmt.Errorf("scanning transaction: %w", err)
		}
		e.ImportHash = hash.String
		e.Date, err = time.Parse("2006-01-02", dateStr)
		if err != nil {
			return nil, fmt.Errorf("parsing date %q: %w", dateStr, err)
		}
		txnMap[e.ID] = &e
		txnOrder = append(txnOrder, e.ID)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating transactions: %w", err)
	}

	// 2. Load all postings
	pRows, err := s.db.Query(`
		select id, transaction_id, account_id, amount, currency
		from postings
		order by transaction_id, id
	`)

	if err != nil {
		return nil, fmt.Errorf("querying postings: %w", err)
	}
	defer pRows.Close()

	for pRows.Next() {
		var p finance.Posting
		if err := pRows.Scan(&p.ID, &p.TransactionID, &p.AccountID, &p.Amount, &p.Currency); err != nil {
			return nil, fmt.Errorf("scanning posting: %w", err)
		}
		if e, ok := txnMap[p.TransactionID]; ok {
			e.Postings = append(e.Postings, p)
		}
	}
	if err := pRows.Err(); err != nil {
		return nil, fmt.Errorf("iterating postings: %w", err)
	}

	// 3. Load all tags
	tRows, err := s.db.Query(`
		select transaction_id, tag
		from transaction_tags
		order by transaction_id, tag
	`)

	if err != nil {
		return nil, fmt.Errorf("querying tags: %w", err)
	}
	defer tRows.Close()

	for tRows.Next() {
		var txnID int64
		var tag string
		if err = tRows.Scan(&txnID, &tag); err != nil {
			return nil, fmt.Errorf("scanning tag: %w", err)
		}
		if e, ok := txnMap[txnID]; ok {
			e.Tags = append(e.Tags, tag)
		}
	}

	if err := tRows.Err(); err != nil {
		return nil, fmt.Errorf("iterating tags: %w", err)
	}

	// 4. Assemble in order
	txns := make([]finance.Transaction, 0, len(txnOrder))
	for _, id := range txnOrder {
		txns = append(txns, *txnMap[id])
	}
	return txns, nil

}

func (s *Store) InsertPayeeRule(r finance.PayeeRule) (int64, error) {
	result, err := s.db.Exec(`
		insert into payee_rules (pattern, normalized_payee, default_account_id, priority)
		values (?, ?, ?, ?)
	`, r.Pattern, r.NormalizedPayee, r.DefaultAccountID, r.Priority)
	if err != nil {
		return 0, fmt.Errorf("inserting payee rule %q: %w", r.Pattern, err)
	}
	return result.LastInsertId()
}

func (s *Store) LoadPayeeRules() ([]finance.PayeeRule, error) {
	rows, err := s.db.Query(`
		select id, pattern, normalized_payee, default_account_id, priority
		from payee_rules
		order by priority, id
	`)
	if err != nil {
		return nil, fmt.Errorf("querying payee rules: %w", err)
	}
	defer rows.Close()

	var rules []finance.PayeeRule
	for rows.Next() {
		var r finance.PayeeRule
		var accountID sql.NullInt64
		if err := rows.Scan(&r.ID, &r.Pattern, &r.NormalizedPayee, &accountID, &r.Priority); err != nil {
			return nil, fmt.Errorf("scanning payee rules: %w", err)
		}
		if accountID.Valid {
			r.DefaultAccountID = &accountID.Int64
		}
		rules = append(rules, r)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating payee rules: %w", err)
	}

	return rules, nil

}

func (s *Store) SeedPayeeRules(defaults []finance.PayeeRule) (int, error) {
	var count int
	err := s.db.QueryRow("select count(*) from payee_rules").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("counting payee rules: %w", err)
	}
	if count > 0 {
		return 0, nil
	}
	for _, r := range defaults {
		if _, err := s.InsertPayeeRule(r); err != nil {
			return 0, fmt.Errorf("seeding rule %q: %w", r.Pattern, err)
		}
		count++
	}
	return count, nil
}

// EnsureAccount returns the ID of the account at path, creating it
// (with type inferred from the first path segment) if it doesn't exist
func (s *Store) EnsureAccount(path string) (int64, error) {
	return ensureAccount(s.db, path)
}

func (s *Store) UpdatePosting(postingID, accountID int64) error {
	return updatePosting(s.db, postingID, accountID)
}

func updatePosting(q dbtx, postingID, accountID int64) error {
	result, err := q.Exec(`
		update postings set account_id = ? where id = ?
		`, accountID, postingID,
	)

	if err != nil {
		return fmt.Errorf("updating posting %d: %w", postingID, err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking rows affected: %w", err)
	}
	if n == 0 {
		return fmt.Errorf("updating posting %d not found", postingID)
	}
	return nil

}

// dbtx is satisfied by both *sql.DB and *sql.TX, so helpers can run
// standalone or inside a transaction
type dbtx interface {
	Exec(query string, args ...any) (sql.Result, error)
	QueryRow(query string, args ...any) *sql.Row
}

func insertAccount(q dbtx, acc finance.Account) (int64, error) {
	result, err := q.Exec(`
		insert into accounts (path, type, currency, opened_at, closed_at)
		values (?, ?, ?, ?, ?)
	`, acc.Path, acc.Type, acc.Currency, acc.OpenedAt, acc.ClosedAt)
	if err != nil {
		return 0, fmt.Errorf("inserting account %q: %w", acc.Path, err)
	}
	return result.LastInsertId()
}

func ensureAccount(q dbtx, path string) (int64, error) {
	typ, err := finance.AccountTypeFromPath(path)
	if err != nil {
		return 0, fmt.Errorf("ensuring account %q: %w", path, err)
	}
	var id int64
	err = q.QueryRow("select id from accounts where path = ?", path).Scan(&id)
	switch {
	case err == nil:
		return id, nil
	case errors.Is(err, sql.ErrNoRows):
		return insertAccount(q, finance.Account{Path: path, Type: typ, Currency: "SEK"})
	default:
		return 0, fmt.Errorf("look up account %q: %w", path, err)
	}
}

func (s *Store) ReviewTransaction(txID, postingID int64, accountPath string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("starting transaction: %w", err)
	}
	defer tx.Rollback()

	accountID, err := ensureAccount(tx, accountPath)
	if err != nil {
		return fmt.Errorf("ensureAccount: %w", err)
	}
	if err := updatePosting(tx, postingID, accountID); err != nil {
		return err
	}

	// clear transaction
	result, err := tx.Exec(`
		update transactions set cleared = 1 where id = ?
	`, txID)
	if err != nil {
		return fmt.Errorf("clearing transaction %d: %w", txID, err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("getting rows affected: %w", err)
	}
	if n == 0 {
		return fmt.Errorf("no transaction %d to update", txID)
	}

	return tx.Commit()

}

func (s *Store) EnsurePayeeRule(r finance.PayeeRule) (created bool, err error) {
	if r.DefaultAccountID == nil {
		return false, fmt.Errorf("rule is missing default account id")
	}
	// query for an existing rule with same pattern and default_account_id
	var id int64
	err = s.db.QueryRow(`
		select id
		from payee_rules
		where pattern = ?
		and default_account_id = ?
	`, r.Pattern, *r.DefaultAccountID).Scan(&id)
	switch {
	case err == nil:
		return false, nil
	case errors.Is(err, sql.ErrNoRows):
		if _, err := s.InsertPayeeRule(r); err != nil {
			return false, err
		}
		return true, nil
	default:
		return false, fmt.Errorf("look up payee_rule (%q, %d): %w", r.Pattern, *r.DefaultAccountID, err)
	}

}

func (s *Store) RepointPostings(postingIDs []int64, accountID int64) error {
	tx, err := s.db.Begin()

	if err != nil {
		return fmt.Errorf("starting transaction: %w", err)
	}
	defer tx.Rollback()

	for _, id := range postingIDs {
		if err := updatePosting(tx, id, accountID); err != nil {
			return err
		}
	}

	return tx.Commit()

}
