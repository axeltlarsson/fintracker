package tui

import "fintracker/internal/finance"

// transactionView is the flat projection of an Transaction for table/detail display
// derived on demand from the postings - never stored
type transactionView struct {
	Account  string // the asset/liability ("from") account path)
	Amount   finance.Öre
	Category string // contra account path; "(split)" when >2 postings
}

// projectTransaction flattens a Transaction into the single row the UI shows, viewed
// from the asset/liability side. accts resolved posting AccountIDs to paths
func projectTransaction(e finance.Transaction, accts map[int64]finance.Account) transactionView {
	// get the asset or liability posting w/o relying on order of postings
	assPosting := e.Postings[0]   // default if no asset/liability posting found
	otherPosting := e.Postings[1] // a valid Transaction always has >= postings
	for _, posting := range e.Postings {
		a := accts[posting.AccountID].Type
		if a == finance.Assets || a == finance.Liabilities {
			assPosting = posting
			break
		} else {
			otherPosting = posting
		}
	}

	acct := accts[assPosting.AccountID]

	var cat string
	if len(e.Postings) > 2 {
		cat = "(split)"
	} else {
		cat = accts[otherPosting.AccountID].Path
	}

	return transactionView{
		Account:  acct.Path,
		Amount:   assPosting.Amount,
		Category: cat,
	}

}

// Return the contra posting of transaction:
// the first posting whose resolved account Type is neither Liabilities nor Assets
func contraPosting(t finance.Transaction, accts map[int64]finance.Account) (finance.Posting, bool) {
	// iterate the postings, return the first one whose resolved account Type is neither
	// Liabilities, nor Assets
	for _, posting := range t.Postings {
		account := accts[posting.AccountID]
		if account.Type != finance.Assets && account.Type != finance.Liabilities {
			return posting, true
		}
	}
	return finance.Posting{}, false
}
