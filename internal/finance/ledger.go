package finance

import (
	"time"
)

// TODO will become Transaction
type Entry struct {
	ID int64
	Date time.Time
	Payee string
	RawPayee string
	Memo string
	Cleared bool
	Postings []Posting
	Tags []string
}

type Posting struct {
	ID int64
	EntryID int64
	AccountID int64
	Amount Öre
	Currency string // TODO: should be a stricter type imo
}

func (e Entry) Validate() error {
	// TODO implement
	return nil
}


