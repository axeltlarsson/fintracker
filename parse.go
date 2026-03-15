package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
)

func parseTransactions(r io.Reader, account string) ([]Transaction, error) {
	cr := csv.NewReader(r)
	cr.Comma = ';'

	var txns []Transaction

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
		t, err := parseRow(record, account)
		if err != nil {
			return nil, fmt.Errorf("parsing row %v: %w", record, err)
		}

		txns = append(txns, t)
	}
	return txns, nil
}

func parseRow(fields []string, account string) (Transaction, error) {
	date, err := time.Parse("2006-01-02", strings.TrimSpace(fields[0]))
	if err != nil {
		return Transaction{}, fmt.Errorf("bad date %q: %w", fields[0], err)
	}

	amount, err := parseAmount(fields[1])
	if err != nil {
		return Transaction{}, fmt.Errorf("bad amount %q: %w", fields[1], err)
	}

	return Transaction{
		Date:    date,
		Amount:  amount,
		Payee:   strings.TrimSpace(fields[2]),
		Account: account,
	}, nil
}

func parseAmount(s string) (Öre, error) {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, " ", "")

	parts := strings.SplitN(s, ",", 2)

	kronor, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, err
	}

	var öre int64
	if len(parts) == 2 {
		// pad or truncate to exactly 2 digits
		örePart := parts[1]
		if len(örePart) == 1 {
			örePart += "0"
		} else if len(örePart) > 2 {
			örePart = örePart[:2]
		}
		öre, err = strconv.ParseInt(örePart, 10, 64)
		if err != nil {
			return 0, err
		}
	}

	total := kronor*100 + öre
	if kronor < 0 {
		total = kronor*100 - öre
	}
	return Öre(total), nil

}
