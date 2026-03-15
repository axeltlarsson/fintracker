package main

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type Rule struct {
	PayeeContains string `yaml:"payee_contains"`
	Category      string `yaml:"category"`
}

func loadRules(path string) ([]Rule, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading rules: %w", err)
	}

	var rules []Rule
	if err := yaml.Unmarshal(data, &rules); err != nil {
		return nil, fmt.Errorf("parsing rules: %w", err)
	}

	return rules, nil
}

func categorize(txns []Transaction, rules []Rule) int {

	matched := 0
	for i := range txns {
		if txns[i].Category != "" {
			continue // already categorised, skip
		}
		for _, rule := range rules {
			if strings.Contains(
				strings.ToUpper(txns[i].Payee),
				strings.ToUpper(rule.PayeeContains),
			) {
				txns[i].Category = rule.Category
				matched++
				break // first match wins
			}
		}
	}
	return matched
}
