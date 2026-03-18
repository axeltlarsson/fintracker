package main

import (
	"strings"
	"testing"
)

func TestParseAmount(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    Öre
		wantErr bool
	}{
		{name: "simple positive", input: "100,00", want: 100_00},
		{name: "negative", input: "-49,50", want: -49_50},
		{name: "no decimals", input: "200", want: 200_00},
		{name: "single decimal digit", input: "3,5", want: 3_50},
		{name: "truncates to two decimals", input: "1,99", want: 1_99},
		{name: "thousand separator", input: "1 000,00", want: 1000_00},
		{name: "leading/trailing whitespace", input: "  42,00  ", want: 42_00},
		{name: "zero", input: "0,00", want: 0},
		{name: "negative with öre", input: "-5,75", want: -5_75},
		{name: "incorrect decimal", input: "-5.75", wantErr: true},
		{name: "empty input", input: "", wantErr: true},
		{name: "weird input .", input: ".", wantErr: true},
		{name: "weird input ,", input: ",", wantErr: true},
		{name: "non digit input", input: "abc", wantErr: true},
		{name: "incorrect thousands sep", input: "1,200,000,000.00", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseAmount(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("parseAmount(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})

	}
}

func TestParseTransaction(t *testing.T) {
	input := "2006-01-15;-49,50;ICA Nära\n2026-01-16;1000,00;Lön\n"

	txns, err := parseTransactions(strings.NewReader(input), "SEB")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(txns) != 2 {
		t.Fatalf("got %d transactions, want 2", len(txns))
	}

	// spot-check first transaction
	if txns[0].Amount != -49_50 {
		t.Errorf("txns[0].Amount = %d, want %d", txns[0].Amount, Öre(-49_50))
	}
	if txns[0].Payee != "ICA Nära" {
		t.Errorf("txns[0].Payee = %q, want %q", txns[0].Payee, "ICA Nära")
	}
	if txns[0].Account != "SEB" {
		t.Errorf("txns[0].Account = %q, want %q", txns[0].Account, "SEB")
	}

}

func FuzzParseAmount(f *testing.F) {
	// seed corpus - give the fuzzer real-world examples to mutate from
	f.Add("100,00")
	f.Add("-49,50")
	f.Add("0")
	f.Add("1 000,00")
	f.Add("")
	f.Add(",")
	f.Add("abc")

	f.Fuzz(func(t *testing.T, s string) {
		öre, err := parseAmount(s)
		if err != nil {
			return // errors are fine, we're looking for panics
		}
		// Property: if parsing succeded, String() should not panic
		_ = öre.String()
	})

}
