package tui

import (
	"context"
	"errors"
	"fmt"
	"sort"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

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
	categorySummaryScreen // TODO might not need it as is right now
)

type ImportSpec struct {
	Path    string
	Account string
}

type Model struct {
	// Data
	entries         []finance.Entry
	accountsByID    map[int64]finance.Account
	filteredEntries []int
	netWorth        finance.Öre
	accountSummary  map[string]finance.Öre
	categorySummary map[string]finance.Öre
	store           *store.Store

	// Import state
	importSpecs  []ImportSpec
	importing    bool
	importStatus string

	// UI components - each is a Bubble with its own state
	table    TxnTable
	viewport viewport.Model
	help     help.Model
	keys     keyMap // list/global keymap

	// UI state
	screen        screen
	filterAccount string
	accounts      []string
	isDark        bool
	width         int
	height        int
	ready         bool // true once we've received the first WindowSizeMsg
	searching     bool // TODO: would it make sense to have a proper state enum/state machine?
	searchInput   textinput.Model

	// Theming
	theme  Theme
	styles styles
}

func InitialModelFromStore(store *store.Store, specs []ImportSpec) (Model, error) {
	entries, err := store.LoadEntries()

	if err != nil {
		return Model{}, err
	}

	accounts, err := store.LoadAccounts()
	if err != nil {
		return Model{}, err
	}
	accountsByID := make(map[int64]finance.Account, len(accounts))
	for _, a := range accounts {
		accountsByID[a.ID] = a
	}

	theme := RoséPineMain // default to dark
	st := newStyles(theme)

	cols := buildCols()
	visibleIdx := initialVisibleIdx(len(entries))
	rows := buildRowsFromIdx(entries, accountsByID, visibleIdx)

	t := NewTxnTable(
		// Data
		WithTxnColumns(cols),
		WithTxnRows(rows),
		// Layout
		WithTxnFocused(true),
		WithTxnHeight(20),
		// Appearance - from styles, model just wires it up
		WithTxnStyleFunc(st.entryStyleFuncFromIdx(entries, accountsByID, visibleIdx)),
		WithTxnHeaderStyle(st.tableHeader),
		WithTxnBorderStyle(st.tableBorder),
	)

	// Search input
	si := textinput.New()
	si.Placeholder = "search..."
	si.CharLimit = 100

	keys := newKeyMap()

	help := help.New()
	help.Styles = newHelpStyles(theme)

	return Model{
		// Data
		entries:         entries,
		accountsByID:    accountsByID,
		netWorth:        netWorth(entries, accountsByID),
		accountSummary:  buildAccountSummary(entries, accountsByID),
		categorySummary: buildCategorySummary(entries, accountsByID),
		store:           store,

		// Import state
		importSpecs: specs,

		// UI components
		table: t,
		help:  help,
		keys:  keys,

		// UI state
		accounts:    collectAccounts(entries, accountsByID),
		searchInput: si,

		// Theming
		theme:  theme,
		styles: st,
	}, nil
}

func initialVisibleIdx(n int) []int {
	idx := make([]int, n)
	for i := range idx {
		idx[i] = i
	}
	return idx
}

func collectAccounts(entries []finance.Entry, accts map[int64]finance.Account) []string {
	seen := make(map[string]bool)
	for _, e := range entries {
		seen[projectEntry(e, accts).Account] = true
	}
	accs := make([]string, 0, len(seen))
	for a := range seen {
		accs = append(accs, a)
	}
	sort.Strings(accs)
	return accs

}

// sums postings on Asset and Liability accounts
func netWorth(entries []finance.Entry, accts map[int64]finance.Account) finance.Öre {
	var total finance.Öre
	for _, e := range entries {
		for _, p := range e.Postings {
			switch accts[p.AccountID].Type {
			case finance.Assets, finance.Liabilities:
				total += p.Amount
			}
		}
	}
	return total
}

// returns the balance of each Asset/Liability account
func buildAccountSummary(entries []finance.Entry, accts map[int64]finance.Account) map[string]finance.Öre {
	summary := make(map[string]finance.Öre)
	for _, e := range entries {
		for _, p := range e.Postings {
			a := accts[p.AccountID]
			switch a.Type {
			case finance.Assets, finance.Liabilities:
				summary[a.Path] += p.Amount
			}
		}
	}
	return summary
}

func buildCategorySummary(entries []finance.Entry, accts map[int64]finance.Account) map[string]finance.Öre {
	summary := make(map[string]finance.Öre)
	for _, e := range entries {
		for _, p := range e.Postings {
			a := accts[p.AccountID]
			switch a.Type {
			case finance.Income, finance.Expenses:
				summary[a.Path] += p.Amount
			}
		}
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
		// 1. resolve accounts (DB writes) before the parallel fan out
		placeholderID, err := s.EnsureAccount("Equity:Uncategorized")
		if err != nil {
			return ImportErrMsg{Err: fmt.Errorf("placeholder account: %w", err)}
		}
		sourceIDs := make([]int64, len(specs))
		for i, spec := range specs {
			id, err := s.EnsureAccount(spec.Account)
			if err != nil {
				return ImportErrMsg{Err: fmt.Errorf("source account %q: %w", spec.Account, err)}
			}
			sourceIDs[i] = id
		}

		// 2. load payee rules (read)
		rules, err := s.LoadPayeeRules()
		if err != nil {
			return ImportErrMsg{Err: err}
		}

		// 3. parse + transform in parallel (pure, no db)
		entries, err := importAllFiles(context.Background(), specs, sourceIDs, placeholderID, rules, progress)
		if err != nil {
			return ImportErrMsg{Err: err}
		}

		// 4. insert entries sequentially (DB writes)
		var count, skipped int
		for _, e := range entries {
			_, err := s.InsertEntry(e)
			if errors.Is(err, store.ErrDuplicateEntry) {
				// skip duplicate entries
				skipped++
				continue
			}
			if err != nil {
				return ImportErrMsg{Err: err}
			}
			count++
		}

		return ImportDoneMsg{Total: len(entries), Inserted: count, Duplicates: skipped}
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
	Total      int // total txns imported
	Inserted   int // new rows inserted
	Duplicates int // skipped rows (duplicates)
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
		m.table.SetStyleFunc(m.styles.entryStyleFuncFromIdx(m.entries, m.accountsByID, m.filteredEntries))
		m.help.Styles = newHelpStyles(m.theme)
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// Layout budget - each element's fixed height
		const (
			titleH       = 3 // title text + margin bottom
			tableBorderH = 3 // header row + top/bottom border
			statusLineH  = 1
			helpH        = 2 // help text + margin top
		)
		chrome := titleH + tableBorderH + statusLineH + helpH
		tableRows := max(msg.Height-chrome, 1)
		if !m.ready {
			m.viewport = viewport.New()
			m.ready = true
		}

		m.table.SetWidth(msg.Width)
		m.table.SetHeight(tableRows)
		m.searchInput.SetWidth(msg.Width / 3) // reasonable width for status line context
		m.viewport.SetWidth(msg.Width - 4)
		m.viewport.SetHeight(msg.Height - 2) // detail/summary screens: minimal chrome
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
		// reload entries from store to include newly import ones
		entries, err := m.store.LoadEntries()
		if err != nil {
			return m, tea.Quit
		}
		accounts, err := m.store.LoadAccounts()
		if err != nil {
			return m, tea.Quit
		}
		accountsByID := make(map[int64]finance.Account, len(accounts))
		for _, a := range accounts {
			accountsByID[a.ID] = a
		}
		m.entries = entries
		m.accountsByID = accountsByID
		m.netWorth = netWorth(entries, accountsByID)
		m.accountSummary = buildAccountSummary(entries, accountsByID)
		m.categorySummary = buildCategorySummary(entries, accountsByID)
		m.accounts = collectAccounts(entries, accountsByID)
		m.refreshTable()

		m.importStatus = fmt.Sprintf("Imported %d transactions (%d new, %d duplicates)", msg.Total, msg.Inserted, msg.Duplicates)
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
	}
	return m, tea.Batch(cmds...)
}

func (m Model) updateList(msg tea.Msg) (tea.Model, tea.Cmd) {

	if m.searching {
		return m.updateSearch(msg)
	}
	switch msg := msg.(type) {
	case tea.KeyPressMsg:

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

		case key.Matches(msg, m.keys.Search):
			m.searching = true
			m.searchInput.SetValue("")
			cmd := m.searchInput.Focus()
			return m, cmd

		case key.Matches(msg, m.keys.Help):
			m.help.ShowAll = !m.help.ShowAll
			return m, nil

		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit

		}
	}

	// forward everything else to the list component
	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	return m, cmd

}
func (m Model) updateSearch(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, m.keys.Back):
			m.searching = false
			m.searchInput.Blur()
			m.searchInput.SetValue("")
			m.table.ClearSearch()
			return m, nil
		case key.Matches(msg, m.keys.Enter):
			m.searching = false
			m.searchInput.Blur()
			// keep filter active, just exit input mode
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.searchInput, cmd = m.searchInput.Update(msg)
	// refresh table on every keystroke
	m.table.SetSearch(m.searchInput.Value())
	return m, cmd
}

func buildCols() []TxnColumn {
	return []TxnColumn{
		{Title: "Date", Width: 12},
		{Title: "Payee", Width: 25},
		{Title: "Amount", Width: 14, Align: lipgloss.Right},
		{Title: "Account", Width: 12},
		{Title: "Category", Width: 18},
	}
}

func buildRowsFromIdx(entries []finance.Entry, accts map[int64]finance.Account, idx []int) [][]string {
	rows := make([][]string, 0, len(idx))
	for _, i := range idx {
		e := entries[i]
		v := projectEntry(e, accts)
		rows = append(rows, []string{
			e.Date.Format("2006-01-02"),
			e.DisplayPayee(),
			v.Amount.String(),
			v.Account,
			v.Category,
		})
	}
	return rows
}

// selectedEntry returns a pointer to the transactions under the cursor
func (m *Model) selectedEntry() *finance.Entry {
	if len(m.filteredEntries) == 0 {
		return nil
	}
	return &m.entries[m.filteredEntries[m.table.Cursor()]]
}
func (m *Model) refreshTable() {
	m.filteredEntries = m.filteredEntries[:0]
	for i, e := range m.entries {
		if m.filterAccount != "" && projectEntry(e, m.accountsByID).Account != m.filterAccount {
			continue
		}
		m.filteredEntries = append(m.filteredEntries, i)
	}
	m.table.SetRows(buildRowsFromIdx(m.entries, m.accountsByID, m.filteredEntries))
	m.table.SetStyleFunc(m.styles.entryStyleFuncFromIdx(m.entries, m.accountsByID, m.filteredEntries))
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
		}
	}

	// forward to viewport for scrolling
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
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
