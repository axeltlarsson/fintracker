package importer

import (
	_ "embed"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"fintracker/internal/finance"
)

//go:embed default_rules.yaml
var defaultRules []byte

// RawRow is a parsed but unprocessed transaction row from a bank CSV export
type RawRow struct {
	Date     time.Time
	Amount   finance.Öre
	RawPayee string
}

// BankFormat knows how to parse a specific bank's CSV export
type BankFormat interface {
	Parse(r io.Reader) ([]RawRow, error)
}

// ImportResult splits rows into matched entries (ready to insert) and
// unmatched rows (no rul fired, need manual account assignment in the TUI)
type ImportResult struct {
	Entries   []finance.Entry
	Unmatched []RawRow
}

func Import(r io.Reader, format BankFormat, sourceAccountID int64, rules []finance.PayeeRule) (ImportResult, error) {
	rows, err := format.Parse(r)
	if err != nil {
		return ImportResult{}, fmt.Errorf("parsing: %w", err)
	}

	sorted := make([]finance.PayeeRule, len(rules))
	copy(sorted, rules)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Priority < sorted[j].Priority
	})

	var result ImportResult
	for _, row := range rows {
		rule, ok := matchRule(row.RawPayee, sorted)
		if !ok {
			result.Unmatched = append(result.Unmatched, row)
			continue
		}
		result.Entries = append(result.Entries, finance.Entry{
			Date:     row.Date,
			Payee:    rule.NormalizedPayee,
			RawPayee: row.RawPayee,
			Postings: []finance.Posting{
				{AccountID: sourceAccountID, Amount: row.Amount, Currency: "SEK"},
				{AccountID: rule.DefaultAccountID, Amount: -row.Amount, Currency: "SEK"},
			},
		})
	}
	return result, nil

}

func matchRule(payee string, rules []finance.PayeeRule) (finance.PayeeRule, bool) {
	for _, rule := range rules {

		if strings.Contains(
			strings.ToUpper(payee),
			strings.ToUpper(rule.Pattern),
		) {
			return rule, true
		}
	}
	return finance.PayeeRule{}, false

}

func (s *Store) InsertPayeeRule(r finance.PayeeRule) (int64, error) {

}
