package importer

import (
	_ "embed"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"fintracker/internal/finance"

	"crypto/sha256"
	"encoding/hex"
	"gopkg.in/yaml.v3"
)

//go:embed default_rules.yaml
var defaultRules []byte

// RawRow is a parsed but unprocessed transaction row from a bank CSV export
type RawRow struct {
	Date     time.Time
	Amount   finance.Öre
	RawPayee string
	Hash     string // import dedup hash, stamped after parse
}

// BankFormat knows how to parse a specific bank's CSV export
type BankFormat interface {
	Parse(r io.Reader) ([]RawRow, error)
}

// ImportResult splits rows into matched transactions (ready to insert) and
// unmatched rows (no rul fired, need manual account assignment in the TUI)
type ImportResult struct {
	Transactions []finance.Transaction
	Unmatched    []RawRow
}

func Import(r io.Reader, format BankFormat, sourceAccountID int64, rules []finance.PayeeRule) (ImportResult, error) {
	rows, err := format.Parse(r)
	if err != nil {
		return ImportResult{}, fmt.Errorf("parsing: %w", err)
	}
	stampHashes(rows)

	sorted := make([]finance.PayeeRule, len(rules))
	copy(sorted, rules)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Priority < sorted[j].Priority
	})

	var result ImportResult
	for _, row := range rows {
		rule, ok := matchRule(row.RawPayee, sorted)
		if !ok || rule.DefaultAccountID == nil {
			result.Unmatched = append(result.Unmatched, row)
			continue
		}
		result.Transactions = append(result.Transactions, finance.Transaction{
			Date:     row.Date,
			Payee:    rule.NormalizedPayee,
			RawPayee: row.RawPayee,
			Postings: []finance.Posting{
				{AccountID: sourceAccountID, Amount: row.Amount, Currency: "SEK"},
				{AccountID: *rule.DefaultAccountID, Amount: -row.Amount, Currency: "SEK"},
			},
			ImportHash: row.Hash,
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

// yamlRule is the on-disk shape of a payee rule in default_rules.yaml
// Separate from finance.PayeeRule because YAML has no ID or account reference
type yamlRule struct {
	Pattern         string `yaml:"pattern"`
	NormalizedPayee string `yaml:"normalized_payee"`
	Priority        int    `yaml:"priority"`
}

// DefaultRules parses the embedded default_rules.yaml into PayeeRules
// DefaultAccountID is nil - the user assigns accounts after seeding
func DefaultRules() ([]finance.PayeeRule, error) {
	var raw []yamlRule
	if err := yaml.Unmarshal(defaultRules, &raw); err != nil {
		return nil, fmt.Errorf("parsing default rules: %w", err)
	}

	rules := make([]finance.PayeeRule, len(raw))
	for i, r := range raw {
		rules[i] = finance.PayeeRule{
			Pattern:         r.Pattern,
			NormalizedPayee: r.NormalizedPayee,
			Priority:        r.Priority,
		}
	}
	return rules, nil
}

func PlaceholderTransactions(rows []RawRow, sourceAccountID, placeholderAccountID int64) []finance.Transaction {
	txns := make([]finance.Transaction, 0, len(rows))
	for _, row := range rows {
		txn := finance.Transaction{
			Date:     row.Date,
			Payee:    "", // unmatched - review TUI to fill in
			RawPayee: row.RawPayee,
			Postings: []finance.Posting{
				{AccountID: sourceAccountID, Amount: row.Amount, Currency: "SEK"},
				{AccountID: placeholderAccountID, Amount: -row.Amount, Currency: "SEK"},
			},
			ImportHash: row.Hash,
		}
		txns = append(txns, txn)
	}

	return txns

}

func stampHashes(rows []RawRow) {
	seen := make(map[string]int)
	for i, row := range rows {
		key := fmt.Sprintf("%s|%d|%s", row.Date.Format("2006-01-02"), row.Amount, row.RawPayee)
		seen[key]++
		keyWithCount := fmt.Sprintf("%s|%d", key, seen[key])
		shaSum := sha256.Sum256([]byte(keyWithCount))
		rows[i].Hash = hex.EncodeToString(shaSum[:])
	}

}
