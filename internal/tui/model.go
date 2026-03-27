package tui

import (
	"context"
	"fmt"
	"sort"
	"strings"

	tea "charm.land/bubbletea/v2"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"

	"fintracker/internal/finance"
	"fintracker/internal/store"
)

type screen int

const appTitle = "fintracker"
const (
	listScreen screen = iota
	detailScreen
	summaryScreen
	categoryScreen
	categorySummaryScreen // TODO might not need it as is right now
)

type ImportSpec struct {
	Path    string
	Account string
}

type Model struct {
	// Data
	transactions    []finance.Transaction
	visibleIdx      []int // indices into transactions
	totalBalance    finance.Öre
	accountSummary  map[string]finance.Öre
	categorySummary map[string]finance.Öre
	rules           []finance.Rule
	categories      []string
	store           *store.Store

	// Import state
	importSpecs  []ImportSpec
	importing    bool
	importStatus string

	// UI components - each is a Bubble with its own state
	table    TxnTable
	viewport viewport.Model
	catInput textinput.Model
	help     help.Model
	keys     keyMap         // list/global keymap
	catKeys  categoryKeyMap // category screen keybindings

	// UI state
	screen        screen
	filterAccount string
	accounts      []string
	isDark        bool
	width         int
	height        int
	ready         bool // true once we've received the first WindowSizeMsg

	// Theming
	theme  Theme
	styles styles
}

func InitialModelFromStore(store *store.Store, rules []finance.Rule, specs []ImportSpec) (Model, error) {
	txns, err := store.LoadTransactions()

	if err != nil {
		return Model{}, err
	}

	theme := RoséPineMain // default to dark
	st := newStyles(theme)

	// Apply rules to any uncategorised transactions
	if len(txns) > 0 {
		if matched := finance.Categorize(txns, rules); matched > 0 {
			if _, err := store.UpsertTransactions(txns); err != nil {
				return Model{}, fmt.Errorf("saving categorized transactions: %w", err)
			}
		}
	}

	cols := buildDefaultColumns()
	visibleIdx := initialVisibleIdx(len(txns))
	rows := buildRowsFromIdx(txns, visibleIdx)

	t := NewTxnTable(
		// Data
		WithTxnColumns(cols),
		WithTxnRows(rows),
		// Layout
		WithTxnFocused(true),
		WithTxnHeight(20),
		// Appearance - from styles, model just wires it up
		WithTxnStyleFunc(st.transactionStyleFuncFromIdx(txns, visibleIdx)),
		WithTxnHeaderStyle(st.tableHeader),
		WithTxnBorderStyle(st.tableBorder),
	)

	// Category text input
	ti := textinput.New()
	ti.Placeholder = "New category..."
	ti.CharLimit = 50
	ti.ShowSuggestions = true

	keys := newKeyMap()
	catKeys := newCategoryKeyMap()

	help := help.New()
	help.Styles = newHelpStyles(theme)

	return Model{
		transactions:    txns,
		visibleIdx:      visibleIdx,
		totalBalance:    finance.CalculateBalance(txns),
		accountSummary:  buildAccountSummary(txns),
		categorySummary: buildCategorySummary(txns),
		rules:           rules,
		categories:      collectCategories(txns, rules),
		store:           store,
		importSpecs:     specs,
		table:           t,
		catInput:        ti,
		help:            help,
		keys:            keys,
		catKeys:         catKeys,
		accounts:        collectAccounts(txns),
		theme:           theme,
		styles:          st,
	}, nil
}

func initialVisibleIdx(n int) []int {
	idx := make([]int, n)
	for i := range idx {
		idx[i] = i
	}
	return idx
}

func collectCategories(txns []finance.Transaction, rules []finance.Rule) []string {

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

func collectAccounts(txns []finance.Transaction) []string {
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

func buildAccountSummary(txns []finance.Transaction) map[string]finance.Öre {
	summary := make(map[string]finance.Öre)

	for _, t := range txns {
		summary[t.Account] += t.Amount
	}
	return summary
}

func buildCategorySummary(txns []finance.Transaction) map[string]finance.Öre {

	// amount per category
	summary := make(map[string]finance.Öre)

	for _, t := range txns {
		c := t.Category
		if t.Category == "" {
			c = "(uncategorised)"
		}
		summary[c] += t.Amount
	}
	return summary
}

func (m Model) Init() tea.Cmd {
	cmds := []tea.Cmd{tea.RequestBackgroundColor}
	if len(m.importSpecs) > 0 {
		cmds = append(cmds, func() tea.Msg {
			return ImportStartMsg{FileCount: len(m.importSpecs)}
		})
	}
	return tea.Batch(cmds...)
}

func (m Model) importAllCmd() tea.Cmd {
	specs := m.importSpecs
	s := m.store
	progress := make(chan ImportFileProgress, len(specs)) // buffered channel

	doImportCmd := func() tea.Msg {
		defer close(progress)
		txns, err := parseAllFiles(context.Background(), specs, progress)
		if err != nil {
			return ImportErrMsg{Err: err}
		}
		inserted, err := s.UpsertTransactions(txns)
		if err != nil {
			return ImportErrMsg{Err: fmt.Errorf("storing transactions: %w", err)}
		}
		return ImportDoneMsg{Total: len(txns), Inserted: inserted}
	}

	listenProgressCmd := listenForProgress(progress)

	return tea.Batch(doImportCmd, listenProgressCmd)

}

// returns a Cmd that reads ONE msg from the channel
// When update receives the msg, it re-calls this to get the next one
func listenForProgress(progress <-chan ImportFileProgress) tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-progress
		if !ok {
			return nil // channel closed, no more progress
		}

		return ImportProgressMsg{
			Account:  msg.Account,
			Count:    msg.Count,
			Progress: progress, // carry the channel forward
		}
	}
}

// Import messages - sent from background tea.Cmd to Update()
type ImportStartMsg struct {
	FileCount int
}

type ImportProgressMsg struct {
	Account  string
	Count    int
	Progress <-chan ImportFileProgress
}
type ImportDoneMsg struct {
	Total    int // total txns imported
	Inserted int // new rows inserted
}

type ImportErrMsg struct {
	Err error
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {

	case tea.BackgroundColorMsg:
		m.isDark = msg.IsDark()
		if m.isDark {
			m.theme = RoséPineMain
		} else {
			m.theme = RoséPineDawn
		}
		m.styles = newStyles(m.theme)
		m.table.SetStyleFunc(m.styles.transactionStyleFuncFromIdx(m.transactions, m.visibleIdx))
		m.help.Styles = newHelpStyles(m.theme)
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		headerHeight := 0
		footerHeight := 2

		if !m.ready {
			m.viewport = viewport.New()
			m.ready = true
		}

		m.table.SetWidth(msg.Width)
		m.table.SetHeight(msg.Height - 6) // room for title, help, borders
		m.catInput.SetWidth(msg.Width - 4)
		m.viewport.SetWidth(msg.Width - 4)
		m.viewport.SetHeight(msg.Height - headerHeight - footerHeight)
		m.help.SetWidth(msg.Width)

		return m, nil

	case ImportStartMsg:
		m.importing = true
		m.importStatus = "Importing files..."
		// kick off the first file
		return m, m.importAllCmd()

	case ImportProgressMsg:
		m.importStatus = fmt.Sprintf("Parsed %s: %d transactions", msg.Account, msg.Count)
		// re-subscribe for next progress update
		return m, listenForProgress(msg.Progress)

	case ImportDoneMsg:
		// reload transactions from store to include newly import ones
		txns, err := m.store.LoadTransactions()
		if err != nil {
			return m, tea.Quit
		}
		// re-apply rules
		if matched := finance.Categorize(txns, m.rules); matched > 0 {
			m.store.UpsertTransactions(txns)
		}
		m.transactions = txns
		m.totalBalance = finance.CalculateBalance(txns)
		m.accountSummary = buildAccountSummary(txns)
		m.categorySummary = buildCategorySummary(txns)
		m.categories = collectCategories(txns, m.rules)
		m.accounts = collectAccounts(txns)
		m.refreshTable()

		m.importStatus = fmt.Sprintf("Imported %d transactions (%d new)", msg.Total, msg.Inserted)
		return m, nil

	case ImportErrMsg:
		m.importStatus = fmt.Sprintf("Import error: %v", msg.Err)
		return m, nil

	case tea.KeyPressMsg:
		if key.Matches(msg, m.keys.Quit) && m.screen != listScreen {
			m.screen = listScreen
			return m, nil
		}
	}

	// Dispatch to screen-specific update

	switch m.screen {
	case listScreen:
		return m.updateList(msg)
	case detailScreen:
		return m.updateDetail(msg)
	case summaryScreen:
		return m.updateSummary(msg)
	case categoryScreen:
		return m.updateCategory(msg)

	}
	return m, tea.Batch(cmds...)
}

func (m Model) updateList(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		// Don't intercept keys when the list is filtering
		// TODO: implement filter here again?
		// if m.table.FilterState() == list.Filtering {
		// 	break
		// }
		switch {
		case key.Matches(msg, m.keys.Enter):
			m.screen = detailScreen
			m.viewport.SetContent(m.renderDetail())
			m.viewport.GotoTop()
			return m, nil

		case key.Matches(msg, m.keys.Summary):
			m.screen = summaryScreen
			m.viewport.SetContent(m.renderSummary())
			m.viewport.GotoTop()
			return m, nil

		case key.Matches(msg, m.keys.Filter):
			m.filterAccount = m.nextAccount()
			m.refreshTable()
			return m, nil

		case key.Matches(msg, m.keys.Category):
			m.screen = categoryScreen
			m.catInput.SetValue("")
			m.catInput.SetSuggestions(m.categories)
			cmd := m.catInput.Focus()
			return m, cmd

		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit

		}
	}

	// forward everything else to the list component
	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	return m, cmd

}

func buildDefaultColumns() []TxnColumn {
	return []TxnColumn{
		{Title: "Date", Width: 12},
		{Title: "Payee", Width: 25},
		{Title: "Amount", Width: 14},
		{Title: "Account", Width: 12},
		{Title: "Category", Width: 18},
	}
}

func buildRowsFromIdx(txns []finance.Transaction, idx []int) [][]string {
	rows := make([][]string, 0, len(idx))
	for _, i := range idx {
		t := txns[i]
		cat := t.Category
		if cat == "" {
			cat = uncategorized
		}
		rows = append(rows, []string{
			t.Date.Format("2006-01-02"),
			t.Payee,
			t.Amount.String(),
			t.Account,
			cat,
		})
	}
	return rows
}

// selectedTxn returns a pointer to the transactions under the cursor
func (m *Model) selectedTxn() *finance.Transaction {
	if len(m.visibleIdx) == 0 {
		return nil
	}
	return &m.transactions[m.visibleIdx[m.table.Cursor()]]
}
func (m *Model) refreshTable() {
	m.visibleIdx = m.visibleIdx[:0]
	for i, t := range m.transactions {
		if m.filterAccount != "" && t.Account != m.filterAccount {
			continue
		}
		m.visibleIdx = append(m.visibleIdx, i)
	}
	m.table.SetRows(buildRowsFromIdx(m.transactions, m.visibleIdx))
	m.table.SetStyleFunc(m.styles.transactionStyleFuncFromIdx(m.transactions, m.visibleIdx))
}

func (m Model) nextAccount() string {
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

func (m Model) updateDetail(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, m.keys.Back):
			m.screen = listScreen
			return m, nil
		case key.Matches(msg, m.keys.Category):
			m.screen = categoryScreen
			m.catInput.SetValue("")
			m.catInput.SetSuggestions(m.categories)
			cmd := m.catInput.Focus()
			return m, cmd
		}
	}

	// forward to viewport for scrolling
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m Model) updateCategory(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, m.catKeys.Back):
			m.catInput.Blur()
			m.screen = detailScreen
			return m, nil

		case key.Matches(msg, m.catKeys.Confirm):
			value := strings.TrimSpace(m.catInput.Value())
			if value == "" {
				return m, nil
			}

			// apply category
			txn := m.selectedTxn()
			txn.Category = value

			// persist to database
			if m.store != nil {
				if err := m.store.UpdateCategory(*txn); err != nil {
					_ = err // for now silently ignore
				}
			}

			// update derived state
			m.accountSummary = buildAccountSummary(m.transactions)
			m.categorySummary = buildCategorySummary(m.transactions)
			if !contains(m.categories, value) {
				m.categories = append(m.categories, value)
				sort.Strings(m.categories)
			}

			m.refreshTable()

			m.catInput.SetSuggestions(m.categories)
			m.catInput.Blur()
			m.screen = detailScreen
			m.viewport.SetContent(m.renderDetail())
			return m, nil
		}

	}
	// forward to text input
	var cmd tea.Cmd
	m.catInput, cmd = m.catInput.Update(msg)

	return m, cmd
}

func contains(ss []string, s string) bool {
	for _, v := range ss {
		if v == s {
			return true
		}
	}
	return false
}

func (m Model) updateSummary(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, m.keys.Back):
			m.screen = listScreen
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}
