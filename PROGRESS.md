# PROGRESS.md — fintracker learning progress

> This file is a living document. Claude suggests updates at the end of each session; Axel approves them after which Claude applies it.

## Project state

fintracker is a Bubble Tea v2 TUI for personal finance tracking across multiple Swedish bank accounts.

### Current architecture

```
fintracker/
├── cmd/fintracker/
│   ├── main.go            # entry point, flag parsing, orchestration
│   ├── args.go            # CLI argument parsing (account:path format)
│   └── load.go            # file loading, calls parser
├── internal/
│   ├── finance/           # Transaction, Öre, Rule, Categorize
│   │   ├── transaction.go
│   │   ├── categorize.go
│   │   ├── transaction_test.go
│   │   └── categorize_test.go
│   ├── parser/            # CSV parsing (io.Reader → []Transaction)
│   │   ├── parse.go
│   │   └── parse_test.go
│   ├── store/             # SQLite persistence (modernc.org/sqlite)
│   │   ├── store.go
│   │   └── store_test.go
│   └── tui/               # Bubble Tea model, views, keys, styles, custom components
│       ├── model.go       # Bubble Tea Model, Update, Init, orchestration
│       ├── views.go       # View rendering (detail, summary, category screens)
│       ├── keys.go        # key.Binding definitions
│       ├── styles.go      # design tokens: styles struct, StyleFunc, theme → style mapping
│       ├── theme.go       # Rosé Pine palette (15 colors × 3 variants)
│       ├── txntable.go    # custom interactive table (lipgloss rendering + cursor/scroll)
│       ├── import.go      # parallel CSV import with errgroup + progress channel
│       └── import_test.go
├── testdata/
│   └── seb.csv            # sample CSV for manual testing
├── go.mod
└── flake.nix
```

Dependency graph: `cmd/fintracker → tui, finance, parser, store` · `tui → finance, store` · `store → finance` · `parser → finance` · `finance → (nothing)`

### Tech stack

- Go (latest stable)
- charm.land/bubbletea/v2 — TUI framework (Elm Architecture)
- charm.land/lipgloss/v2 — terminal styling
- charm.land/bubbles/v2 — TUI components (list, viewport, textinput, help, table)
- charm.land/lipgloss/v2/table — styled tables
- modernc.org/sqlite — pure Go SQLite driver (no CGo)
- gopkg.in/yaml.v3 — YAML parsing for categorization rules
- database/sql — standard library SQL interface

### Key types

- `Öre int64` — monetary amount in öre (1/100 SEK), satisfies fmt.Stringer
- `Transaction` — Date, Amount (Öre), Payee, Account, Category; satisfies list.Item
- `Store` — wraps *sql.DB; NewStore, Close, UpsertTransactions, LoadTransactions, UpdateCategory
- `Rule` — PayeeContains, Category; loaded from YAML with struct tags
- `model` — Bubble Tea model; embeds list.Model, viewport.Model, textinput.Model, help.Model

### Design principles

- Value types for data (Transaction, Öre), pointer types for resources (*Store, *sql.DB)
- Composition over inheritance — Bubbles components embedded in model
- io.Reader/io.Writer for I/O abstraction
- Errors propagated upward with fmt.Errorf wrapping (%w)
- View is a pure render function; computation happens in Update
- Maps for sets (map[string]bool), sorted key extraction for stable iteration

---

## Go concepts covered

- [x] Modules, packages, imports, go.mod
- [x] Structs, defined types, struct tags
- [x] Methods, value vs pointer receivers
- [x] Interfaces (structural typing, io.Reader, io.Writer, fmt.Stringer, tea.Model, list.Item)
- [x] Error handling ((value, error) pattern, %w wrapping, error chains)
- [x] Slices (append, range, sub-slicing, length vs capacity, make)
- [x] Maps (as sets, accumulators, iteration order randomness)
- [x] Closures and first-class functions
- [x] defer for resource cleanup
- [x] database/sql, prepared statements, rows.Err()
- [x] String building (strings.Builder, fmt.Fprintf)
- [x] Time parsing (reference time layout)
- [x] Visibility (exported/unexported via capitalization)
- [x] Type switches, type assertions
- [x] iota for enums
- [x] Blank imports for side effects
- [x] Composition pattern (forwarding Update to sub-components, tea.Batch)
- [x] Zero values
- [x] Functional options pattern (WithTxn* constructors, variadic opts)
- [ ] Generics (mentioned, not deeply used)
- [ ] iter package and range-over-function (mentioned, not used)
- [x] Testing (table-driven, subtests, coverage, race detector)
- [x] Fuzz testing
- [x] Goroutines (via errgroup.Go, tea.Batch)
- [x] Channels (buffered, directional types chan<-/<-chan, close, range-over-channel)
- [x] select (cancellable channel send with ctx.Done())
- [x] context.Context (errgroup.WithContext, context.Background, cancellation via ctx.Done())
- [ ] sync package (WaitGroup, Mutex, Once)
- [x] errgroup (golang.org/x/sync/errgroup)
- [ ] net/http (client and server)
- [ ] JSON encoding/decoding
- [ ] Custom error types (errors.As, errors.Is)
- [ ] Benchmarking (go test -bench, pprof)
- [ ] Build tags
- [ ] go generate
- [ ] //go:embed
- [ ] Reflection (struct tags under the hood)

---

## Roadmap

### Phase 8: Testing
**Feature:** comprehensive test suite for existing code.
**Go concepts:** testing package, go test, table-driven tests, subtests (t.Run), testify vs stdlib assertions, test fixtures, golden files, _test.go file convention, test coverage (go test -cover), the -race flag, testing.Short() for skipping slow tests, TestMain for setup/teardown.
**Exercise ideas:**
- Table-driven tests for parseAmount (edge cases: negative, no decimals, thousand separators, garbage input)
- Test parseTransactions with a strings.Reader (io.Reader payoff)
- Test categorize with various rule/transaction combinations
- Test Store with a temporary in-memory SQLite database
- Test Öre.String() formatting

### Phase 9: Fuzz testing
**Feature:** find parsing bugs with fuzzing.
**Go concepts:** go test -fuzz, corpus seeding, writing fuzz targets, interpreting crashes.
**Exercise ideas:**
- Fuzz parseAmount — discover edge cases in decimal handling
- Fuzz the CSV parser with malformed input
- Fix any bugs the fuzzer finds

### Phase 10: Project structure & packages
**Feature:** split into internal packages.
**Go concepts:** internal/ directory, package design (accept interfaces return structs), circular dependency prevention, godoc.
**Suggested structure:**
```
fintracker/
├── cmd/fintracker/main.go
├── internal/
│   ├── finance/        # Transaction, Öre, categorization
│   ├── store/          # SQLite persistence
│   ├── importer/       # CSV parsing, bank format handling
│   └── tui/            # Bubble Tea model, views, keys, styles
├── testdata/
└── go.mod
```

### Phase 11: Concurrency
**Feature:** background CSV import with progress, parallel bank data fetching.
**Go concepts:** goroutines, channels, select, sync.WaitGroup, context.Context, errgroup, tea.Cmd as goroutine abstraction.

### Phase 12: Rosé Pine theme
**Feature:** proper theming with Rosé Pine palette (main, moon, dawn variants).
**Go concepts:** functional options, config structs, embedding for theme inheritance, color profile detection.

### Phase 13: Search & filter
**Feature:** `/` fuzzy search (text match, temporary), `f` structured filters (account, category, status facets, persistent), `F` clear all filters.
**Go concepts:** string matching, state machines (filter mode), composable predicates, builder pattern for filter chains.
**Design:**
- `/` narrows visibleIdxs by substring match on payee + category + account. `esc` clears, `enter` locks filter.
- `f` opens filter mode: `(a)ccount (c)ategory (u)ncategorized (d)ate range`. Filters shown as pills in status line. Multiple filters stack.
- `F` clears all active filters (like Linear).
- All filters compose through `refreshTable`: period → structured filters → text search → visibleIdxs.
- Current `tab` account cycling is temporary — will be replaced by `f` → `a` once structured filters land.

### Phase 14: Period & batch workflow
**Feature:** monthly batch categorization workflow with period as a first-class concept.
**Go concepts:** time.Time range filtering, custom types (DateRange), sort.Slice with multi-key comparisons.
**Design:**
- Period shown in title bar: "fintracker · March 2026"
- `[` / `]` navigate prev/next month
- Progress counter: "15/47 categorized"
- Smart default on startup: auto-select latest period with uncategorized transactions
- Sort: uncategorized transactions first (work queue at top), categorized sink below a visual divider
- Period is global state — affects all views

### Phase 15: Multi-view architecture
**Feature:** `tab` switches between views: Transactions (list/categorize/detail) and Statistics (charts/KPIs).
**Go concepts:** interface-based view abstraction, per-view state vs global state, the Model-per-view pattern in Bubble Tea.
**Design:**
- Views: Transactions (current functionality) and Statistics (spending breakdown, trends, budgets)
- Period is global — switching views keeps the same period, data stays in sync
- Each view has its own filter state and screen navigation
- Current `screen` enum becomes per-view state
- Architecture: `Model { transactions, period, activeView, txnView, statsView }`
- Single source of truth (`m.transactions`) means categorizing in Transactions view is instantly reflected in Statistics

### Phase 16: Statistics view
**Feature:** spending analytics with charts and KPIs.
**Go concepts:** text-based chart rendering, aggregate queries, formatting tables with computed columns.
**Ideas:**
- Category breakdown (bar chart or table with percentages)
- Month-over-month spending comparison
- Top payees by spend
- Income vs expenses summary
- Budget vs actual (if budgets are configured)

### Phase 17: Vim-style navigation
**Feature:** vim modes (normal, insert, command) in the TUI.
**Go concepts:** state machines, dynamic key binding enable/disable, rune handling, Unicode.

### Phase 18: Multiple bank format support
**Feature:** parse CSV from SEB, Swedbank, Nordea, ICA Banken.
**Go concepts:** Strategy pattern via interfaces, factory functions, //go:embed for default configs.

### Phase 19: HTTP & APIs
**Feature:** GoCardless integration or local HTTP API.
**Go concepts:** net/http, http.Client, JSON, context with timeouts, middleware.

### Phase 20: Configuration
**Feature:** config file for database path, bank formats, rules, theme, keybindings.
**Go concepts:** //go:embed, config hierarchy (defaults → file → env → flags), XDG conventions.

### Phase 21: Distribution
**Feature:** installable via Nix, goreleaser, go install.
**Go concepts:** ldflags for version embedding, buildGoModule in flake.nix, goreleaser.

### Key binding plan (end state)
```
tab          switch view (Transactions ↔ Statistics)
/            fuzzy search (temporary text filter)
f            structured filter (faceted, persistent)
F            clear all filters
[ / ]        prev / next period (month)
enter        drill into detail
esc          back / clear search
c            categorize
j/k          navigate
g/G          top / bottom
?            help toggle
q            quit
```

### Ongoing topics to weave in opportunistically

- Custom error types (errors.As, errors.Is)
- Generics (utility functions, type constraints)
- Benchmarking (go test -bench, pprof)
- Linting (golangci-lint, staticcheck, exhaustive)
- Documentation (godoc, doc.go, example tests)
- Build tags (platform-specific, integration tests)
- Reflection (how struct tags work)
- Channel patterns (fan-out/fan-in, pipelines)
- sync package (Once, Pool, Map, atomics)
- go generate
- Category name normalization (case-insensitive dedup: "entertainment" == "Entertainment") — let Axel drive this one with minimal guidance

---

## Session log

### Session 1 — initial conversation (pre-Claude Code)
**Date:** 2026-03-17
**Where:** claude.ai
**Covered:** Phases 1–7 of fintracker. Built the full application from scratch: TUI with Bubble Tea v2, CSV parsing, YAML categorization rules, SQLite persistence with modernc.org/sqlite, Lip Gloss styling, Bubbles components (list, viewport, textinput, help, table). Also covered: Go module system, Nix integration (devShell, buildGoModule vs gomod2nix), CGo tradeoffs, cross-compilation, sum type workarounds, FP primitives in Go, struct embedding/shadowing rules.
**Notes:** Project name is fintracker. Axel prefers project-first learning (build the thing, learn concepts as needed). Created CLAUDE.md and PROGRESS.md for continuing in Claude Code.

### Session 2 — testing (Phase 8)
**Date:** 2026-03-18
**Covered:** Phase 8 (testing). Table-driven tests with subtests for parseAmount, Öre.String(), CalculateBalance, parseTransactions, categorize, loadRules, and Store (upsert round-trip with in-memory SQLite). Test helper with t.Helper()/t.Cleanup(). Coverage profiling (go test -cover, -coverprofile). Race detector (-race). Found and fixed a bug: parseAmount silently truncated >2 decimal digits, now returns an error. Learned fmt.Errorf verbs (%v, %q, %w) and error wrapping.
**Next:** Phase 9 (fuzz testing) — fuzz parseAmount to discover more edge cases.

### Session 2b — fuzz testing (Phase 9)
**Date:** 2026-03-18
**Covered:** Phase 9 (fuzz testing). Fuzz targets for parseAmount and parseTransactions using testing.F. Seed corpus design, property-based assertions (no panics, invariant checks), -fuzztime flag. ~1.6M inputs tested across both targets with no crashes. Learned how fuzz corpus entries become permanent regression tests.
**Next:** Phase 10 (project structure & packages) or pick from roadmap.

### Session 2c — project structure (Phase 10)
**Date:** 2026-03-18
**Covered:** Phase 10 (project structure). Split flat `package main` into `cmd/fintracker/` + `internal/{finance,parser,store,tui}`. Learned: `internal/` compiler enforcement, `cmd/` convention, `testdata/` per-package scope, Go file naming conventions (verb not actor), Go's procedural-with-interfaces paradigm. Created `TransactionItem` wrapper using struct embedding to satisfy `list.Item` across package boundaries. Dependency graph flows inward with `finance` as the leaf package. All tests pass across packages with `go test ./...`.
**Next:** Phase 11 (concurrency) or pick from roadmap.

### Session 3 — concurrency with errgroup (Phase 11, part 1)
**Date:** 2026-03-20
**Covered:** Phase 11 (concurrency), first part. Replaced sequential file-by-file import chain with parallel parsing using `errgroup`. Created `internal/tui/import.go` with `parseAllFiles` using index-partitioned results (no mutex needed). Added buffered progress channel (`chan ImportFileProgress`) with directional types. Implemented Bubble Tea's recursive Cmd pattern for progress reporting: `listenForProgress` reads one message, carries the `<-chan` forward in the `ImportProgressMsg`, `Update` re-subscribes. Used `tea.Batch` to run import + progress listener concurrently. Fixed goroutine leak with `defer close(progress)` on error paths. Discussed why Bubble Tea v2 has no Elm-style `Sub` — commands are the only abstraction, subscriptions are manual re-issue.
**Concepts taught:** errgroup pattern, buffered vs unbuffered channels, channel direction types, goroutine leak prevention, recursive Cmd as subscription, `tea.Batch` for concurrent commands, closure capture of Model fields for goroutine safety.
**Next:** Continue Phase 11 — test `parseAllFiles`, `g.SetLimit`, `select`, context cancellation, or `sync` primitives.

### Session 4 — testing concurrent code, select (Phase 11, part 2)
**Date:** 2026-03-21
**Covered:** Tested `parseAllFiles` — happy path (2 files, parallel), empty specs, and bad file path. The bad-path test exposed a real goroutine leak: successful goroutine blocked on unbuffered progress channel send while errgroup waited for all goroutines. Fixed with `select`/`ctx.Done()` pattern — cancellable channel operations. Full suite passes with `-race`.
**Concepts taught:** `select` statement for multiplexing channel operations, `ctx.Done()` for cancellation-aware sends, testing concurrent code to find real bugs, goroutine leak detection via test timeouts.
**Next:** Phase 12 (Rosé Pine theme) or pick from roadmap.

### Session 5 — Rosé Pine theme + TxnTable component (Phase 12)
**Date:** 2026-03-23 to 2026-03-24
**Covered:** Phase 12 (theming) + custom TUI component.

**Theming:** Designed and implemented a three-layer design token system: Theme (primitive palette, 15 colors × 3 variants) → styles struct (semantic mapping) → views (consumers). Full Rosé Pine palette (Main, Moon, Dawn) with docs from rosepinetheme.com. Theme struct uses `color.Color` (lipgloss v2 breaking change). Views never touch the theme directly — all appearance comes from `m.styles`. Discussed API design at length: encapsulation vs exposed palette, semantic aliases as methods vs fields, option D (raw palette, styles layer maps to semantics). Color decision methods (`amountColor`, `categoryColor`) as single source of truth shared between pre-built styles and StyleFunc.

**Table migration:** Migrated from `bubbles/list` → `bubbles/table` → custom `TxnTable`. Discovered ANSI nesting limitation: pre-rendered ANSI codes in cell data contain reset sequences that kill outer backgrounds. The bubbles interactive table lacks `StyleFunc` (per-cell styling); the lipgloss rendering table has it but no interactivity. Solution: composed lipgloss table (rendering) with custom state management (cursor, scrolling) in `TxnTable` (~250 lines). Per-cell styling via `TxnStyleFunc` closure — all styling at render time, no pre-rendered ANSI in row data. Selected row background now spans all columns correctly.

**Architecture decisions:**
- Model is the orchestration layer — wires data + styles via functional options, never constructs lipgloss values
- styles.go owns all appearance decisions, exposes values and methods for model to pass through
- TxnTable is generic (knows nothing about transactions) — domain styling comes via StyleFunc closure
- Column index constants (`colDate`, `colAmount`, etc.) for type-safe StyleFunc switch cases

**Concepts taught:** design tokens, `color.Color` interface, lipgloss v2 API, functional options pattern (`With*` constructors, variadic opts), ANSI nesting limitations, composition over forking, Go's lack of named/keyword arguments, closure-based per-cell styling, `min`/`max` builtins (Go 1.21+).

**Remaining polish (for next session):**
- Table padding and column alignment refinement
- Help keymap display on list screen
- Filter indicator (show which account is filtered)
- Fuzzy search/filter in the table
- Color tuning (reduce Iris overuse, balance palette)
- Status messages (import progress, errors) without bubbles list

### Session 6 — UI polish + search (Phase 12 polish + Phase 13 start)
**Date:** 2026-03-27
**Covered:** Bug fixes, data structure refactor, UI polish, search implementation.

**Bug fix — category not showing immediately:** Traced to value-type mutation bug: `visibleTxns` held copies of transactions, not references. Assigning category to the copy didn't affect `m.transactions`. Classic Go gotcha with value types in slices.

**Data structure refactor — index slice pattern:** Replaced `visibleTxns []finance.Transaction` (copies) with `visibleIdx []int` (indices into `m.transactions`). Added `selectedTxn()` helper returning `*finance.Transaction` for direct mutation. Eliminated data duplication and the entire class of stale-copy bugs. This pattern became the foundation for all subsequent filtering work.

**Color tuning:** Table headers from Iris → Subtle (less visual noise). Selected row: HighlightLow + Rose. Iris reserved for active filter indicators. Gold for transient status messages. Follows Rosé Pine palette roles.

**Status line:** Added contextual status bar with three fixed-width columns (statusLeft, statusMiddle, statusRight). Shows filter state (left), search/import status (middle), transaction count (right). Each column carries its own Background(Surface) to avoid ANSI nesting issues. Learned the hard way that gap-calculation layout with ANSI strings causes jumping — fixed-width columns solve it.

**Table alignment:** Added `Align lipgloss.Position` field to `TxnColumn`. Table View reads alignment from column definition — domain knowledge stays in `buildCols`, generic component respects it. "Data, not code" principle.

**Help toggle:** Wired `?` to toggle `m.help.ShowAll`. Added Filter and Search to ShortHelp. Noted that full help expanding inline is awkward (pushes layout around) — future improvement: help as full screen.

**Search — controlled component pattern:** Built `/` fuzzy search. Key architectural decision: TxnTable owns filtering logic (`SetFilter`, `ClearFilter`, `applyFilter`, `matchRow`) but NOT the textinput. Model owns the textinput and calls `table.SetFilter(query)`. This is the "controlled component" pattern — table filters on string data without knowing where the query comes from. `matchRow` is generic (matches query against all cells), no Transaction domain knowledge in the table. TxnTable's `filtered []int` indices (same pattern as `visibleIdx`) map display positions to original row indices. `Cursor()` maps through `filtered` transparently.

**Cleanup:** Deleted dead code: `item.go` (unused TransactionItem from bubbles/list era), `viewCategorySummaryScreen`, `categorySummaryScreen` constant. Fixed `expandHome` bug (path[:2] → path[2:]).

**Layout:** Fixed-height table rendering (pad with newlines when fewer rows than allocated height) to prevent layout jumping during search. Named layout budget constants (titleHeight, tableBorderH, statusLineH, helpH) replacing magic numbers.

**Roadmap updates:** Expanded phases 13–21 with detailed design for search/filter, period/batch workflow, multi-view architecture, statistics view. Added key binding plan (end state). Core insight: period (monthly batch) is a first-class concept, not just a filter. `tab` will switch views (Transactions ↔ Statistics), `f` for structured filters (Linear-style), `/` for fuzzy search.

**Concepts practiced:** Value vs pointer semantics in slices, index slice pattern, controlled vs uncontrolled components, ANSI nesting pitfalls, fixed-width terminal layout, separation of filtering logic from input management, design token discipline (views consume styles, never construct them).

**Remaining polish:**
- Rename for clarity: Model's `visibleIdx` → `filteredTxns`, TxnTable's `filtered` → `searchIdx`, `SetFilter`/`ClearFilter` → `SetSearch`/`ClearSearch`
- Help as full screen (instead of inline toggle)
- Search input still jumps slightly (status line layout needs more tuning)
- `ctrl-d`/`ctrl-u` are swapped (PageUp/PageDown reversed from vim convention)

**Next:** Phase 13 (structured filters with `f` key) or Phase 14 (period/batch workflow), or tackle remaining polish.

