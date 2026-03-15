package main

import (
	"fmt"
	"os"
	"slices"
)

func loadTransactions(specs []importSpec) ([]Transaction, error) {
	var all []Transaction

	for _, spec := range specs {
		txns, err := loadFile(spec)
		if err != nil {
			return nil, fmt.Errorf("loading %s (%s): %w", spec.account, spec.path, err)
		}
		all = append(all, txns...)

	}

	slices.SortFunc(all, func(a, b Transaction) int {
		return a.Date.Compare(b.Date)
	})

	return all, nil

}

func loadFile(spec importSpec) ([]Transaction, error) {
	f, err := os.Open(spec.path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return parseTransactions(f, spec.account)
}
