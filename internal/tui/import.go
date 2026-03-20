package tui

import (
	"context"
	"fintracker/internal/finance"
	"fintracker/internal/parser"
	"fmt"
	"golang.org/x/sync/errgroup"
	"os"
)

type ImportFileProgress struct {
	Account string
	Count   int // transactions parsed from this file
}

func parseAllFiles(ctx context.Context, specs []ImportSpec, progress chan<- ImportFileProgress) ([]finance.Transaction, error) {
	// errgroup.WithContext gives you:
	// - g the group
	// - ctx - a derivec context that cancels if any goroutine errors out
	g, ctx := errgroup.WithContext(ctx)

	// one slice per goroutine - no mutext needed
	results := make([][]finance.Transaction, len(specs))

	for i, spec := range specs {
		// launch a goroutine for each spec using
		g.Go(func() error {
			// open the file at spec.Path
			f, err := os.Open(spec.Path)
			if err != nil {
				return fmt.Errorf("opening %s: %w", spec.Path, err)
			}
			defer f.Close()
			txns, err := parser.ParseTransactions(f, spec.Account)
			if err != nil {
				return fmt.Errorf("parsing %s: %w", spec.Path, err)
			}

			results[i] = txns

			// signal progress
			progress <- ImportFileProgress{
				Account: spec.Account,
				Count:   len(txns),
			}
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	// flatten results
	var all []finance.Transaction
	for _, txns := range results {
		all = append(all, txns...)
	}

	return all, nil

}
