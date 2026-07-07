package tui

import "fintracker/internal/finance"

// entryView is the flat projection of an Entry for table/detail display
// derived on demand from the postings - never stored
type entryView struct {
	Account  string // the asset/liability ("from") account path)
	Amount   finance.Öre
	Category string // contra account path; "(split)" when >2 postings
}

// projectEntry flattens an Entry into the single row the UI shows, viewed
// from the asset/liability side. accts resolved posting AccountIDs to paths
func projectEntry(e finance.Entry, accts map[int64]finance.Account) entryView {
	// get the asset or liability posting w/o relying on order of postings
	assPosting := e.Postings[0]   // default if no asset/liability posting found
	otherPosting := e.Postings[1] // a valid Entry always has >= postings
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

	return entryView{
		Account:  acct.Path,
		Amount:   assPosting.Amount,
		Category: cat,
	}

}
