package store

import (
	"database/sql"
	"fmt"
	"time"

	"fintracker/internal/finance"
	_ "modernc.org/sqlite"
)

const schema = `
create table if not exists transactions (
	id	integer primary key autoincrement,
	date	text not null,
	amount	integer not null,
	payee	text not null,
	account	text not null,
	category	text not null default'',
	
	unique(date, amount, payee, account)
);
`

type Store struct {
	db *sql.DB
}

func NewStore(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)

	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("creating schema: %w", err)
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
	return &Store{db: db}, nil
}

// UpsertTransactions upserts transactions into the Store
// The number of inserted rows is returned, 0 in case of error
func (s *Store) UpsertTransactions(txns []finance.Transaction) (inserted int, err error) {
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
			return inserted, fmt.Errorf("inserting %s/%s: %w", t.Date.Format("2006-01-02"), t.Payee, err)
		}

		n, _ := result.RowsAffected()
		if n > 0 {
			inserted++
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("comitting: %w", err)
	}

	return inserted, nil
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
