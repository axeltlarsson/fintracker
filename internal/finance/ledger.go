package finance

import (
	"fmt"
	"time"
)

// TODO will become Transaction
type Entry struct {
	ID       int64
	Date     time.Time
	Payee    string
	RawPayee string
	Memo     string
	Cleared  bool
	Postings []Posting
	Tags     []string
}

type Posting struct {
	ID        int64
	EntryID   int64
	AccountID int64
	Amount    Öre
	Currency  string // TODO: should be a stricter type imo
}

func (e Entry) Validate() error {
	// Two invariants:
	// 1. At least two postings in a transaction
	// 2. Sum of amounts per currency must equal zero
	if len(e.Postings) < 2 {
		return fmt.Errorf("transaction needs at least 2 postings, got %d2", len(e.Postings))
	}
	// collect sums by currency
	sums := make(map[string]Öre)
	for _, p := range e.Postings {
		sums[p.Currency] += p.Amount
	}
	// check sums by currency == 0
	for c, v := range sums {
		if v != 0 {
			return fmt.Errorf("transaction does not balance: %q sum is %v öre", c, v)
		}
	}
	return nil
}
