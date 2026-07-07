package main

import (
	"flag"
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"

	"fintracker/internal/importer"
	"fintracker/internal/store"
	"fintracker/internal/tui"
)

func main() {
	dbPath := flag.String("db", "fintracker.db", "path to database")
	flag.Parse()

	s, err := store.NewStore(*dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	defer s.Close()

	// Seed default payee rules on first run (idempotent)
	defaults, err := importer.DefaultRules()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	if _, err := s.SeedPayeeRules(defaults); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	args := flag.Args()

	var specs []tui.ImportSpec

	// Import CSV:s if provided
	if len(args) > 0 {
		specs, err = parseArgs(args)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}

	}
	m, err := tui.InitialModelFromStore(s, specs)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	p := tea.NewProgram(m)
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
