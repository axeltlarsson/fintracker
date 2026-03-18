package main

import (
	"flag"
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"

	"fintracker/internal/finance"
	"fintracker/internal/store"
	"fintracker/internal/tui"
)

func main() {
	rulesPath := flag.String("rules", "", "path to categorization rules YAML")
	dbPath := flag.String("db", "fintracker.db", "path to database")
	flag.Parse()

	s, err := store.NewStore(*dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	defer s.Close()

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
	var rules []finance.Rule
	if *rulesPath != "" {
		rules, err = finance.LoadRules(*rulesPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	}

	m, err := tui.InitialModelFromStore(s, rules, specs)
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
