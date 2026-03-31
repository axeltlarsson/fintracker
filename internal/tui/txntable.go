package tui

import (
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	lipgloss "charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/table"
	"strings"
)

// TxnStyleFunc returs a style for a given cell
// row is the aboslute index into the data (not the visible window)
// col is the column index.
// selected is true if this row is the cursor row
type TxnStyleFunc func(row, col int, selected bool) lipgloss.Style

// TxnColumn defines a column in the transaction table
type TxnColumn struct {
	Title string
	Width int
	Align lipgloss.Position
}

// TxnTable is an interactive table that uses lipgloss/table for rendering
// and manages its own cursor, scrolling, and keyboard navigation
type TxnTable struct {
	cols        []TxnColumn
	rows        [][]string
	cursor      int
	offset      int // first visible row
	visibleRows int
	width       int
	focused     bool
	styleFunc   TxnStyleFunc
	keyMap      TxnTableKeyMap
	border      lipgloss.Border
	borderStyle lipgloss.Style
	headerStyle lipgloss.Style
	// no selectedRow style?

	// Searching
	query     string
	searchIdx []int // indices into t.rows that match the query

}

type TxnTableKeyMap struct {
	Up       key.Binding
	Down     key.Binding
	PageUp   key.Binding
	PageDown key.Binding
	Top      key.Binding
	Bottom   key.Binding
}

func defaultTxnTableKeyMap() TxnTableKeyMap {
	return TxnTableKeyMap{
		Up:       key.NewBinding(key.WithKeys("up", "k")),
		Down:     key.NewBinding(key.WithKeys("down", "j")),
		PageUp:   key.NewBinding(key.WithKeys("pgup", "ctrl+u")),
		PageDown: key.NewBinding(key.WithKeys("pgdown", "ctrl+d")),
		Top:      key.NewBinding(key.WithKeys("home", "g")),
		Bottom:   key.NewBinding(key.WithKeys("end", "G")),
	}
}

func (t *TxnTable) resetSearchIdx() {
	t.searchIdx = make([]int, len(t.rows))
	for i := range t.searchIdx {
		t.searchIdx[i] = i
	}
}

// Option pattern for construction
type TxnTableOption func(*TxnTable)

func WithTxnColumns(cols []TxnColumn) TxnTableOption {
	return func(t *TxnTable) { t.cols = cols }
}

func WithTxnRows(rows [][]string) TxnTableOption {
	return func(t *TxnTable) { t.rows = rows }
}

func WithTxnHeight(h int) TxnTableOption {
	return func(t *TxnTable) { t.visibleRows = h }
}

func WithTxnFocused(f bool) TxnTableOption {
	return func(t *TxnTable) { t.focused = f }
}

func WithTxnStyleFunc(f TxnStyleFunc) TxnTableOption {
	return func(t *TxnTable) { t.styleFunc = f }
}

func WithTxnBorder(b lipgloss.Border) TxnTableOption {
	return func(t *TxnTable) { t.border = b }
}

func WithTxnBorderStyle(s lipgloss.Style) TxnTableOption {
	return func(t *TxnTable) { t.borderStyle = s }
}

func WithTxnHeaderStyle(s lipgloss.Style) TxnTableOption {
	return func(t *TxnTable) { t.headerStyle = s }
}

func NewTxnTable(opts ...TxnTableOption) TxnTable {
	t := TxnTable{
		visibleRows: 20,
		keyMap:      defaultTxnTableKeyMap(),
	}
	for _, opt := range opts {
		opt(&t)
	}
	t.resetSearchIdx()
	return t
}

// --- State access ---

func (t TxnTable) Cursor() int {
	if len(t.searchIdx) == 0 {
		return 0
	}
	return t.searchIdx[t.cursor]

}
func (t TxnTable) SelectedRow() []string {
	if t.cursor < 0 || t.cursor >= len(t.searchIdx) {
		return nil
	}
	return t.rows[t.searchIdx[t.cursor]]
}

// --- State mutation ---

func (t *TxnTable) SetRows(rows [][]string) {
	t.rows = rows
	t.resetSearchIdx()
	if t.cursor >= len(t.searchIdx) {
		t.cursor = max(len(t.searchIdx)-1, 0)
	}
	t.clampOffset()
}

func (t *TxnTable) SetColumns(cols []TxnColumn) { t.cols = cols }
func (t *TxnTable) SetHeight(h int)             { t.visibleRows = h; t.clampOffset() }
func (t *TxnTable) SetWidth(w int)              { t.width = w }
func (t *TxnTable) SetStyleFunc(f TxnStyleFunc) { t.styleFunc = f }
func (t *TxnTable) Focus()                      { t.focused = true }
func (t *TxnTable) Blur()                       { t.focused = false }

func (t *TxnTable) SetSearch(query string) {
	t.query = strings.ToLower(query)
	t.applyFilter()
}

func (t *TxnTable) ClearSearch() {
	t.query = ""
	t.resetSearchIdx()
	t.cursor = clamp(t.cursor, 0, max(len(t.searchIdx)-1, 0))
	t.clampOffset()
}

func (t TxnTable) SearchedCount() int { return len(t.searchIdx) }

func (t *TxnTable) applyFilter() {
	t.searchIdx = t.searchIdx[:0]
	for i, row := range t.rows {
		if t.query == "" || matchRow(row, t.query) {
			t.searchIdx = append(t.searchIdx, i)
		}
	}
	t.cursor = clamp(t.cursor, 0, max(len(t.searchIdx)-1, 0))
	t.clampOffset()
}

func matchRow(row []string, query string) bool {
	for _, cell := range row {
		if strings.Contains(strings.ToLower(cell), query) {
			return true
		}
	}
	return false
}

// --  Navigation ---
func (t *TxnTable) MoveUp(n int) {
	t.cursor = clamp(t.cursor-n, 0, max(len(t.searchIdx)-1, 0))
	// scroll up if cursor is above the visible window
	if t.cursor < t.offset {
		t.offset = t.cursor
	}
}

func (t *TxnTable) MoveDown(n int) {
	t.cursor = clamp(t.cursor+n, 0, max(len(t.searchIdx)-1, 0))
	// Scroll down if cursor is below the visible window
	if t.cursor >= t.offset+t.visibleRows {
		t.offset = t.cursor - t.visibleRows + 1
	}
}

func (t *TxnTable) GotoTop() {
	t.cursor = 0
	t.offset = 0
}

func (t *TxnTable) GotoBottom() {
	t.cursor = max(len(t.searchIdx)-1, 0)
	t.clampOffset()
}

func (t *TxnTable) clampOffset() {
	maxOffset := max(len(t.searchIdx)-t.visibleRows, 0)
	t.offset = clamp(t.offset, 0, maxOffset)
	// Ensure cursor is still visible after offset clamp
	t.offset = min(t.offset, t.cursor)
	if t.cursor >= t.offset+t.visibleRows && t.visibleRows > 0 {
		t.offset = t.cursor - t.visibleRows + 1
	}
}

// --- Bubble tea interface ---

func (t TxnTable) Update(msg tea.Msg) (TxnTable, tea.Cmd) {

	if !t.focused {
		return t, nil
	}
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, t.keyMap.Up):
			t.MoveUp(1)
		case key.Matches(msg, t.keyMap.Down):
			t.MoveDown(1)
		case key.Matches(msg, t.keyMap.PageUp):
			t.MoveUp(t.visibleRows)
		case key.Matches(msg, t.keyMap.PageDown):
			t.MoveDown(t.visibleRows)
		case key.Matches(msg, t.keyMap.Top):
			t.GotoTop()
		case key.Matches(msg, t.keyMap.Bottom):
			t.GotoBottom()
		}
	}
	return t, nil
}

func (t TxnTable) View() string {
	if len(t.cols) == 0 {
		return ""
	}

	// Compute visible window
	end := min(t.offset+t.visibleRows, len(t.searchIdx))
	visible := make([][]string, 0, end-t.offset)
	for _, idx := range t.searchIdx[t.offset:end] {
		visible = append(visible, t.rows[idx])
	}

	// Extract column headers
	headers := make([]string, len(t.cols))
	for i, c := range t.cols {
		headers[i] = c.Title
	}

	// Build lipgloss table rows
	tableRows := make([][]string, len(visible))
	copy(tableRows, visible)

	// Capture offset and cursor for the closure
	offset := t.offset
	cursor := t.cursor
	styleFunc := t.styleFunc
	filtered := t.searchIdx

	lt := table.New().
		Headers(headers...).
		Rows(tableRows...).
		StyleFunc(func(row, col int) lipgloss.Style {
			align := t.cols[col].Align
			if row == table.HeaderRow {
				return t.headerStyle.Align(align)
			}
			absRow := row + offset
			origRow := filtered[absRow]
			isSelected := absRow == cursor
			if styleFunc != nil {
				return styleFunc(origRow, col, isSelected).Align(align)
			}

			// Fallback: default style
			s := lipgloss.NewStyle().Padding(0, 1)
			if isSelected {
				s = s.Bold(true)
			}
			return s.Align(align)
		})
	if t.width > 0 {
		lt = lt.Width(t.width)
	}

	// Apply border if set
	lt = lt.Border(t.border).BorderStyle(t.borderStyle)

	rendered := lt.Render()
	// let table take up full height to prevent shrinking and status bar jumping when filtering rows
	targetHeight := t.visibleRows + 3 // rows + header + top/bottom border
	return lipgloss.PlaceVertical(targetHeight, lipgloss.Top, rendered)

}

func clamp(v, low, high int) int {
	return min(max(v, low), high)
}

// TODO: does this location make sense?
// Ideally want to statically tie to the Transaction I guess
const (
	colDate     = 0
	colPayee    = 1
	colAmount   = 2
	colAccount  = 3
	colCategory = 4
)
