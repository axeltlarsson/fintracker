package tui

import (
	"context"
	"fmt"
	"os"

	"golang.org/x/sync/errgroup"

	"fintracker/internal/finance"
	"fintracker/internal/importer"
)

type ImportFileProgress struct {
	Account string
	Count   int // transactions parsed from this file
}

// parses and transforms each CSV into ledger entries in parallel
func importAllFiles(
	ctx context.Context,
	specs []ImportSpec,
	sourceIDs []int64, // sourceID[i] is the resolved account ID for specs[i]
	placeholderID int64,
	rules []finance.PayeeRule,
	progress chan<- ImportFileProgress,
) ([]finance.Entry, error) {
	g, ctx := errgroup.WithContext(ctx)
	results := make([][]finance.Entry, len(specs))

	for i, spec := range specs {
		g.Go(func() error {
			f, err := os.Open(spec.Path)
			if err != nil {
				return fmt.Errorf("opening %s: %w", spec.Path, err)
			}
			defer f.Close()

			res, err := importer.Import(f, importer.SEBFormat{}, sourceIDs[i], rules)
			if err != nil {
				return fmt.Errorf("importing %s: %w", spec.Path, err)
			}
			entries := res.Entries
			entries = append(entries, importer.PlaceholderEntries(res.Unmatched, sourceIDs[i], placeholderID)...)
			results[i] = entries

			select {
			case progress <- ImportFileProgress{Account: spec.Account, Count: len(entries)}:
			case <-ctx.Done():
				return ctx.Err()
			}
			return nil

		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	var all []finance.Entry
	for _, e := range results {
		all = append(all, e...)
	}

	return all, nil
}
