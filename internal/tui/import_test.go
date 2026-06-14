package tui

import (
	"context"
	"testing"
)

func TestImportAllFiles(t *testing.T) {
	specs := []ImportSpec{
		{Path: "testdata/bank_a.csv", Account: "SEB"},
		{Path: "testdata/bank_b.csv", Account: "Nordea"},
	}

	sourceIDs := []int64{1, 2}
	const placeholderID = 99

	progress := make(chan ImportFileProgress, len(specs))

	// nil rules -> every row is unmatched -> placeholder entries
	entries, err := importAllFiles(context.Background(), specs, sourceIDs, placeholderID, nil, progress)
	if err != nil {
		t.Fatalf("importAllFiles: %v", err)
	}
	close(progress)

	var msgs []ImportFileProgress
	for msg := range progress {
		msgs = append(msgs, msg)
	}
	if len(msgs) != 2 {
		t.Errorf("got %d progress messages, want 2", len(msgs))
	}
	if len(entries) != 5 {
		t.Errorf("got %d entries, want 5", len(entries))
	}

	// each entry: balanced double posting, source posting + placeholder counter
	bySource := make(map[int64]int)
	for _, e := range entries {
		if err := e.Validate(); err != nil {
			t.Errorf("entry doesn't validate: %v", err)
		}
		if len(e.Postings) != 2 {
			t.Fatalf("entry has %d postings, want 2", len(e.Postings))
		}
		bySource[e.Postings[0].AccountID]++
		if e.Postings[1].AccountID != placeholderID {
			t.Errorf("counter account != %d, want placeholder %d", e.Postings[1].AccountID, placeholderID)
		}
	}
	if bySource[1] != 2 {
		t.Errorf("SEB transactions = %d, want 2", bySource[1])
	}
	if bySource[2] != 3 {
		t.Errorf("Nordea transactions = %d, want 3", bySource[2])
	}

}

func TestImportAllFilesEmpty(t *testing.T) {
	progress := make(chan ImportFileProgress)

	entries, err := importAllFiles(context.Background(), nil, nil, 99, nil, progress)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("got %d entries, want 0", len(entries))
	}
}

func TestImportAllFilesBadPath(t *testing.T) {
	specs := []ImportSpec{
		{Path: "testdata/bank_a.csv", Account: "SEB"},
		{Path: "testdata/nonexistent.csv", Account: "Ghost"},
	}

	sourceIDs := []int64{1, 2}
	progress := make(chan ImportFileProgress) // unbuffered

	_, err := importAllFiles(context.Background(), specs, sourceIDs, 99, nil, progress)

	if err == nil {
		t.Fatalf("expected error for missing file, got nil")
	}
}
