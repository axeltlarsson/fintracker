package store

import (
	"database/sql"
	"fmt"
	"time"

	"fintracker/internal/finance"
	_ "modernc.org/sqlite"
)

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

// UpsertTransactions upserts transactions into the Store
// The number of upserted rows is returned, 0 in case of error
func (s *Store) UpsertTransactions(txns []finance.Transaction) (upserted int, err error) {
	tx, err := s.db.Begin()
	if err != nil {
		return 0, fmt.Errorf("beginning transaction: %w", err)
	}

	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		insert into transactions (date, amount, payee, account, category)
		values (?, ?, ?, ?, ?)
		on conflict (date, amount, payee, account) do update
		set category = case
			when excluded.category != '' then excluded.category
			when transactions.category != '' then transactions.category
			else ''
		end
		where excluded.category != transactions.category
	`)

	if err != nil {
		return 0, fmt.Errorf("preparing statement: %w", err)
	}
	defer stmt.Close()

	for _, t := range txns {
		result, err := stmt.Exec(
			t.Date.Format("2006-01-02"),
			int64(t.Amount),
			t.Payee,
			t.Account,
			t.Category,
		)
		if err != nil {
			return upserted, fmt.Errorf("inserting %s/%s: %w", t.Date.Format("2006-01-02"), t.Payee, err)
		}

		n, _ := result.RowsAffected()
		if n > 0 {
			upserted++
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("comitting: %w", err)
	}

	return upserted, nil
}

// Load transactions from the Store
func (s *Store) LoadTransactions() ([]finance.Transaction, error) {
	rows, err := s.db.Query(`
		select id, date, amount , payee, account, category
		from transactions
		order by date, id
	`)
	if err != nil {
		return nil, fmt.Errorf("querying transactions: %w", err)
	}
	defer rows.Close()

	var txns []finance.Transaction
	for rows.Next() {
		var (
			id       int64
			dateStr  string
			amount   int64
			payee    string
			account  string
			category string
		)
		if err := rows.Scan(&id, &dateStr, &amount, &payee, &account, &category); err != nil {
			return nil, fmt.Errorf("scanning row: %w", err)
		}

		date, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			return nil, fmt.Errorf("parsing date %q: %w", dateStr, err)
		}

		txns = append(txns, finance.Transaction{
			ID:       id,
			Date:     date,
			Amount:   finance.Öre(amount),
			Payee:    payee,
			Account:  account,
			Category: category,
		})
	}

	// rows.Err() catches errors that occurred during iteration
	// (e.g. connection lost mid-read). rows.Next() would have
	// returned false, so the loop ends silently without this check.
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating rows: %w", err)
	}

	return txns, nil
}

func (s *Store) UpdateCategory(t finance.Transaction) error {
	_, err := s.db.Exec(`
		update transactions
		set category = ?
		where id = ?
	`, t.Category, t.ID)
	if err != nil {
		return fmt.Errorf("updating category: %w", err)
	}

	return nil
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
			CREATE table if not exists transactions (
				id integer primary key autoincrement,
				date text not null,
				amount integer not null,
				payee text not null,
				account text not null,
				category text not null default '',
				unique(date, amount, payee, account)
			);
			`)
		return err
	},

	// 1 -> 2
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
			create table entries (
				id integer primary key autoincrement,
				date text not null,
				payee text not null default '',
				raw_payee text not null default '',
				memo text not null default '',
				cleared integer not null default 0 -- sqlite has no bool type
			);
			create table postings (
				id integer primary key autoincrement,
				entry_id integer not null references entries(id) on delete cascade,
				account_id integer not null references accounts(id) on delete restrict,
				amount integer not null,
				currency text not null default 'SEK'
			);
			create table entry_tags (
				entry_id integer not null references entries(id) on delete cascade,
				tag text not null,
				primary key (entry_id, tag)
			);
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
	result, err := s.db.Exec(`
		insert into accounts (path, type, currency, opened_at, closed_at)
		values (?, ?, ?, ?, ?)
	`, acc.Path, acc.Type, acc.Currency, acc.OpenedAt, acc.ClosedAt)
	if err != nil {
		return 0, fmt.Errorf("inserting account %q: %w", acc.Path, err)
	}
	return result.LastInsertId()
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

func (s *Store) InsertEntry(e finance.Entry) (int64, error) {
	if err := e.Validate(); err != nil {
		return 0, fmt.Errorf("invalid entry: %w", err)
	}

	tx, err := s.db.Begin()
	if err != nil {
		return 0, fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback()

	// 1 Insert entry itself
	result, err := tx.Exec(`
		insert into entries (date, payee, raw_payee, memo, cleared)
		values (?, ?, ?, ?, ?)
	`, e.Date.Format("2006-01-02"), e.Payee, e.RawPayee, e.Memo, e.Cleared)
	if err != nil {
		return 0, fmt.Errorf("inserting entry: %w", err)
	}
	entryID, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("getting entry ID: %w", err)
	}

	// 2 insert postings
	postStmt, err := tx.Prepare(`
		insert into postings (entry_id, account_id, amount, currency)
		values (?, ?, ?, ?)
	`)
	if err != nil {
		return 0, fmt.Errorf("preparing postings insert: %w", err)
	}
	defer postStmt.Close() // must close prepared statement resource

	for _, p := range e.Postings {
		if _, err := postStmt.Exec(entryID, p.AccountID, int64(p.Amount), p.Currency); err != nil {
			return 0, fmt.Errorf("inserting posting: %w", err)
		}
	}

	// 3. Insert tags
	for _, tag := range e.Tags {
		if _, err := tx.Exec(`
			insert into entry_tags (entry_id, tag) values (?, ?)
	        `, entryID, tag); err != nil {
			return 0, fmt.Errorf("inserting tag %q: %w", tag, err)

		}
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("committing entry: %w", err)
	}

	return entryID, nil

}

func (s *Store) UpdateEntry(e finance.Entry) error {
	_, err := s.db.Exec(`
		UPDATE entries
		SET payee = ?, memo = ?, cleared = ?
		WHERE id = ?
	`, e.Payee, e.Memo, e.Cleared, e.ID)
	if err != nil {
		return fmt.Errorf("updating entry %d: %w", e.ID, err)
	}
	return nil
}

func (s *Store) LoadEntries() ([]finance.Entry, error) {
	// 1. Load all entries
	rows, err := s.db.Query(`
		select id, date, payee, raw_payee, memo, cleared
		from entries
		order by date, id
	`)
	if err != nil {
		return nil, fmt.Errorf("querying entries: %w", err)
	}
	defer rows.Close()

	entryMap := make(map[int64]*finance.Entry)
	var entryOrder []int64 // preserve query order

	for rows.Next() {
		var e finance.Entry
		var dateStr string
		if err := rows.Scan(&e.ID, &dateStr, &e.Payee, &e.RawPayee, &e.Memo, &e.Cleared); err != nil {
			return nil, fmt.Errorf("scanning entry: %w", err)
		}
		e.Date, err = time.Parse("2006-01-02", dateStr)
		if err != nil {
			return nil, fmt.Errorf("parsing date %q: %w", dateStr, err)
		}
		entryMap[e.ID] = &e
		entryOrder = append(entryOrder, e.ID)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating entries: %w", err)
	}

	// 2. Load all postings
	pRows, err := s.db.Query(`
		select id, entry_id, account_id, amount, currency
		from postings
		order by entry_id, id
	`)

	if err != nil {
		return nil, fmt.Errorf("querying postings: %w", err)
	}
	defer pRows.Close()

	for pRows.Next() {
		var p finance.Posting
		if err := pRows.Scan(&p.ID, &p.EntryID, &p.AccountID, &p.Amount, &p.Currency); err != nil {
			return nil, fmt.Errorf("scanning posting: %w", err)
		}
		if e, ok := entryMap[p.EntryID]; ok {
			e.Postings = append(e.Postings, p)
		}
	}
	if err := pRows.Err(); err != nil {
		return nil, fmt.Errorf("iterating postings: %w", err)
	}

	// 3. Load all tags
	tRows, err := s.db.Query(`
		select entry_id, tag
		from entry_tags
		order by entry_id, tag
	`)

	if err != nil {
		return nil, fmt.Errorf("querying tags: %w", err)
	}
	defer tRows.Close()

	for tRows.Next() {
		var entryID int64
		var tag string
		if err = tRows.Scan(&entryID, &tag); err != nil {
			return nil, fmt.Errorf("scanning tag: %w", err)
		}
		if e, ok := entryMap[entryID]; ok {
			e.Tags = append(e.Tags, tag)
		}
	}

	if err := tRows.Err(); err != nil {
		return nil, fmt.Errorf("iterating tags: %w", err)
	}

	// 4. Assemble in order
	entries := make([]finance.Entry, 0, len(entryOrder))
	for _, id := range entryOrder {
		entries = append(entries, *entryMap[id])
	}
	return entries, nil

}
