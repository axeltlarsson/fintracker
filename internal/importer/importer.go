package importer

import (
	_ "embed"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"fintracker/internal/finance"

	"gopkg.in/yaml.v3"
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
				{AccountID: *rule.DefaultAccountID, Amount: -row.Amount, Currency: "SEK"},
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
