package tui

import (
	"context"
	"testing"
)

func TestParseAllFiles(t *testing.T) {
	specs := []ImportSpec{
		{Path: "testdata/bank_a.csv", Account: "SEB"},
		{Path: "testdata/bank_b.csv", Account: "Nordea"},
	}

	progress := make(chan ImportFileProgress, len(specs))

	txns, err := parseAllFiles(context.Background(), specs, progress)
	if err != nil {
		t.Fatalf("parseAllFiles: %v", err)
	}

	// drain progress channel
	close(progress)
	var progressMsgs []ImportFileProgress
	for msg := range progress {
		progressMsgs = append(progressMsgs, msg)
	}
	if len(progressMsgs) != 2 {
		t.Errorf("got %d progress messages, want 2", len(progressMsgs))
	}
	if len(txns) != 5 {
		t.Errorf("got %d transactions, want 5", len(txns))
	}

	// Verify both accounts are present
	accounts := make(map[string]int)
	for _, tx := range txns {
		accounts[tx.Account]++
	}
	if accounts["SEB"] != 2 {
		t.Errorf("SEB transactions = %d, want 2", accounts["SEB"])
	}
	if accounts["Nordea"] != 3 {
		t.Errorf("Nordea transactions = %d, want 3", accounts["Nordea"])
	}

}

func TestParseAllFilesEmpty(t *testing.T) {
	progress := make(chan ImportFileProgress)

	txns, err := parseAllFiles(context.Background(), nil, progress)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(txns) != 0 {
		t.Errorf("got %d transations, wawnt 0", len(txns))
	}
}

func TestParseAllFilesBadPath(t *testing.T) {
	specs := []ImportSpec{
		{Path: "testdata/bank_a.csv", Account: "SEB"},
		{Path: "testdata/nonexistent.csv", Account: "Ghost"},
	}

	progress := make(chan ImportFileProgress)

	_, err := parseAllFiles(context.Background(), specs, progress)

	if err == nil {
		t.Fatalf("expected error for missing file, got nil")
	}
}
