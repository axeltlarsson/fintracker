package main

import (
	"flag"
	"fmt"
	"os"
	"sort"

	tea "charm.land/bubbletea/v2"
)

type importSpec struct {
	path    string
	account string
}

type screen int

const (
	listScreen screen = iota
	detailScreen
	summaryScreen
	categoryScreen
	categorySummaryScreen
)

type model struct {
	transactions    []Transaction
	cursor          int
	totalBalance    Öre
	accountSummary  map[string]Öre
	screen          screen
	categorySummary map[string]Öre
	rules           []Rule
	categories      []string
	catCursor       int
	store           *Store
	filterAccount   string
	accounts        []string
	width           int
	height          int
}

func collectCategories(txns []Transaction, rules []Rule) []string {
	seen := make(map[string]bool)

	for _, r := range rules {
		seen[r.Category] = true
	}

	for _, t := range txns {
		if t.Category != "" {
			seen[t.Category] = true
		}
	}

	cats := make([]string, 0, len(seen))
	for c := range seen {
		cats = append(cats, c)
	}
	sort.Strings(cats)
	return cats
}

func collectAccounts(txns []Transaction) []string {
	seen := make(map[string]bool)
	for _, t := range txns {
		seen[t.Account] = true
	}
	accs := make([]string, 0, len(seen))
	for a := range seen {
		accs = append(accs, a)
	}
	sort.Strings(accs)
	return accs
}

func buildAccountSummary(txns []Transaction) map[string]Öre {
	summary := make(map[string]Öre)

	for _, t := range txns {
		summary[t.Account] += t.Amount
	}
	return summary
}

func buildCategorySummary(txns []Transaction) map[string]Öre {
	// amount per category
	summary := make(map[string]Öre)

	for _, t := range txns {
		c := t.Category
		if t.Category == "" {
			c = "(uncategorised)"
		}
		summary[c] += t.Amount
	}
	return summary
}

func initialModelFromStore(store *Store, rules []Rule) (model, error) {
	txns, err := store.LoadTransactions()

	if err != nil {
		return model{}, err
	}

	if len(txns) == 0 {
		return model{}, fmt.Errorf("no transaction found")
	}

	// Apply rules to any uncategorised transactions
	if matched := categorize(txns, rules); matched > 0 {
		if _, err := store.UpsertTransactions(txns); err != nil {
			return model{}, fmt.Errorf("saving categorized transactions: %w", err)
		}
		fmt.Fprintf(os.Stderr, "categorized %d transactions from rules\n", matched)
	}

	return model{
		transactions:    txns,
		totalBalance:    CalculateBalance(txns),
		accountSummary:  buildAccountSummary(txns),
		categorySummary: buildCategorySummary(txns),
		rules:           rules,
		categories:      collectCategories(txns, rules),
		store:           store,
		accounts:        collectAccounts(txns),
	}, nil
}

func (m model) Init() tea.Cmd {
	// no initial commands to run
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyPressMsg:
		// global keys that work on every screen
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit

		}

		// screen-specific keys

		switch m.screen {
		case listScreen:
			return m.updateList(msg)
		case detailScreen:
			return m.updateDetail(msg)
		case summaryScreen:
			return m.updateSummary(msg)
		case categoryScreen:
			return m.updateCategory(msg)
		case categorySummaryScreen:
			return m.updateCategorySummary(msg)

		}
	}
	return m, nil
}

func (m model) updateList(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q":
		return m, tea.Quit
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.transactions)-1 {
			m.cursor++
		}
	case "enter":
		m.screen = detailScreen
	case "s":
		m.screen = summaryScreen
	case "c":
		m.screen = categorySummaryScreen
	case "tab":
		m.filterAccount = m.nextAccount()
	}
	return m, nil
}

func (m model) nextAccount() string {
	if m.filterAccount == "" {
		// currently showing all - switch to first account
		if len(m.accounts) > 0 {
			return m.accounts[0]
		}
		return ""
	}

	// find current account, advance to next
	for i, a := range m.accounts {
		if a == m.filterAccount {
			if i+1 < len(m.accounts) {
				return m.accounts[i+1]
			}
			return "" // wrap around to "all"
		}
	}
	return ""
}

func (m model) updateDetail(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.screen = listScreen
	case "c":
		m.catCursor = 0
		m.screen = categoryScreen
	}

	return m, nil

}

func (m model) updateCategory(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.screen = detailScreen
	case "up", "k":
		if m.catCursor > 0 {
			m.catCursor--
		}
	case "down", "j":
		if m.catCursor < len(m.categories)-1 {
			m.catCursor++
		}
	case "enter":
		m.transactions[m.cursor].Category = m.categories[m.catCursor]
		m.accountSummary = buildAccountSummary(m.transactions)
		m.categorySummary = buildCategorySummary(m.transactions)

		// persist to database
		if m.store != nil {
			if err := m.store.UpdateCategory(m.transactions[m.cursor]); err != nil {
				_ = err // for now silently ignore
			}
		}
		m.screen = detailScreen
	}
	return m, nil
}

func (m model) updateSummary(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.screen = listScreen
	}
	return m, nil
}

func (m model) updateCategorySummary(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.screen = listScreen
	}

	return m, nil
}

func main() {
	rulesPath := flag.String("rules", "", "path to categorization rules YAML")
	dbPath := flag.String("db", "fintracker.db", "path to database")
	flag.Parse()

	store, err := NewStore(*dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	defer store.Close()

	args := flag.Args()

	// Import CSV:s if provided
	if len(args) > 0 {
		specs, err := parseArgs(args)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}

		txns, err := loadTransactions(specs)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}

		inserted, err := store.UpsertTransactions(txns)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "imported %d new transactions\n", inserted)

	}
	var rules []Rule
	if *rulesPath != "" {
		rules, err = loadRules(*rulesPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	}

	m, err := initialModelFromStore(store, rules)
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
