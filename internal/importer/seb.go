package importer

import (
	"encoding/csv"
	"fmt"
	"io"
	"strings"
	"time"

	"fintracker/internal/parser"
)

type SEBFormat struct {
}

func (s SEBFormat) Parse(r io.Reader) ([]RawRow, error) {
	cr := csv.NewReader(r)
	cr.Comma = ';'

	var txns []RawRow

	for {
		record, err := cr.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("reading csv: %w", err)
		}

		if len(record) < 3 {
			continue // skip malformed rows
		}
		t, err := parseRow(record)
		if err != nil {
			return nil, fmt.Errorf("parsing row %v: %w", record, err)
		}

		txns = append(txns, t)
	}
	return txns, nil
}

func parseRow(fields []string) (RawRow, error) {
	date, err := time.Parse("2006-01-02", strings.TrimSpace(fields[0]))
	if err != nil {
		return RawRow{}, fmt.Errorf("bad date %q: %w", fields[0], err)
	}

	amount, err := parser.ParseAmount(fields[1])
	if err != nil {
		return RawRow{}, fmt.Errorf("bad amount %q: %w", fields[1], err)
	}

	return RawRow{
		Date:     date,
		Amount:   amount,
		RawPayee: strings.TrimSpace(fields[2]),
	}, nil
}
