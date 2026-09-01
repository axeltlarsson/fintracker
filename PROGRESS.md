# PROGRESS.md — fintracker learning progress

> This file is a living document. Claude suggests updates at the end of each session; Axel approves them after which Claude applies it.

## Project state

fintracker is a Bubble Tea v2 TUI for personal finance tracking across multiple Swedish bank accounts.

### Current architecture

```
fintracker/
├── cmd/fintracker/
│   ├── main.go            # entry point, flag parsing, orchestration, rule seeding
│   └── args.go            # CLI argument parsing (Account:Path=file format)
├── internal/
│   ├── finance/           # domain leaf: Transaction, Posting, Account, Öre, PayeeRule, ParseAmount
│   │   ├── ledger.go      # Transaction (double-entry), Posting, Validate, DisplayPayee
│   │   ├── account.go     # Account, AccountType, path helpers, AccountTypeFromPath
│   │   ├── ore.go         # Öre monetary type + String()
│   │   ├── parse.go       # ParseAmount (moved here from the retired parser package)
│   │   ├── payee_rule.go  # PayeeRule
│   │   └── *_test.go
│   ├── importer/          # CSV → []Transaction: BankFormat strategy, Import, dedup hashing
│   │   ├── importer.go    # Import, PlaceholderEntries, matchRule, stampHashes, DefaultRules
│   │   ├── seb.go         # SEBFormat (BankFormat impl)
│   │   ├── default_rules.yaml  # //go:embed default payee rules
│   │   └── importer_test.go
│   ├── store/             # SQLite persistence (modernc.org/sqlite)
│   │   ├── store.go       # migrations, InsertTransaction, LoadTransactions, EnsureAccount, payee rules
│   │   └── store_test.go
│   └── tui/               # Bubble Tea model, views, keys, styles, custom components
│       ├── model.go       # Bubble Tea Model, Update, Init, import orchestration
│       ├── views.go       # View rendering (detail, summary screens)
│       ├── keys.go        # key.Binding definitions
│       ├── styles.go      # design tokens: styles struct, StyleFunc, theme → style mapping
│       ├── theme.go       # Rosé Pine palette (15 colors × 3 variants)
│       ├── txntable.go    # custom interactive table (lipgloss rendering + cursor/scroll/chromeHeight)
│       ├── transactionview.go  # transactionView + projectTransaction (domain → view-model projection)
│       ├── import.go      # parallel CSV parse with errgroup + progress channel (pure, no DB)
│       └── *_test.go
├── testdata/
│   └── seb.csv            # sample CSV for manual testing
├── go.mod
└── flake.nix
```

Dependency graph: `cmd/fintracker → tui, finance, importer, store` · `tui → finance, importer, store` · `importer → finance` · `store → finance` · `finance → (nothing)`. The `parser` package was retired — `ParseAmount` folded into `finance`, CSV parsing owned by `importer`'s `BankFormat` strategy.

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
- `Transaction` — double-entry journal entry: Date, Payee, RawPayee, Memo, Cleared, Postings, Tags, ImportHash. `Validate()` enforces ≥2 postings summing to zero per currency. `DisplayPayee()` prefers normalized Payee, falls back to RawPayee.
- `Posting` — one leg: AccountID, Amount (Öre), Currency (+ TransactionID FK)
- `Account` — typed hierarchical path (`Assets:Bank:SEB`); AccountType enum; `AccountTypeFromPath`
- `PayeeRule` — Pattern, NormalizedPayee, DefaultAccountID (*int64, nullable), Priority
- `Store` — wraps *sql.DB; NewStore, Close, InsertTransaction, LoadTransactions, UpdateTransaction, EnsureAccount, Insert/LoadAccounts, Insert/LoadPayeeRules, SeedPayeeRules
- `Model` — Bubble Tea model; source of truth is `transactions []finance.Transaction` + `accountsByID`; `transactionView`/`projectTransaction` is the render-time projection

### Design principles

- Value types for data (Transaction, Öre), pointer types for resources (*Store, *sql.DB)
- Composition over inheritance — Bubbles components embedded in model
- io.Reader/io.Writer for I/O abstraction
- Errors propagated upward with fmt.Errorf wrapping (%w)
- View is a pure render function; computation happens in Update
- Maps for sets (map[string]bool), sorted key extraction for stable iteration
- Layout is measured, not assumed — a component reports its own chrome (`TxnTable.chromeHeight`) and `Model.relayout` is the single owner of sizing

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
- [~] Custom error types (errors.Is ✓ via sql.ErrNoRows; errors.As + custom types still to cover)
- [ ] Benchmarking (go test -bench, pprof)
- [ ] Build tags
- [ ] go generate
- [x] //go:embed
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
**Follow-up:** live theme switching + robust detection → Phase 26.

### Phase 13: Accounting model (double-entry foundation)
**Feature:** Core domain types for a proper journal-based model — `Account`, `Posting`, `Entry` (journal transaction), `Validate()`. Replaces the current flat `Transaction` model over subsequent phases. See `ACCOUNTING_ROADMAP.md` for full design.
**Go concepts:** custom error types (`errors.As`/`errors.Is`), TDD for pure business logic, named types for type safety, table-driven tests as specification.
**Design:**
- `Account` — typed, hierarchical colon-delimited paths (`Assets:Checking:SEB`, `Expenses:Food:Groceries`). Type: Assets | Liabilities | Income | Expenses | Equity.
- `Posting` — atomic unit: one account, one `Öre` amount (signed), one currency.
- `Entry` — group of `Posting`s that must sum to zero per currency. Renamed to `Transaction` once flat model is retired.
- `Validate()` — enforces the double-entry invariant: `len(Postings) >= 2`, sum per currency == 0.
- New files: `internal/finance/account.go`, `internal/finance/ledger.go`.
- Approach: red-green TDD — write tests first, implement to pass.
**Note on naming:** `Entry` is used during migration to avoid conflict with the existing flat `Transaction`. Will be renamed `Transaction` when the old model is retired in Phase 14.

### Phase 14: Import pipeline v2
**Feature:** CSV → `Entry`+`Posting` pairs. `payee_rules` table in SQLite replaces YAML categorization rules. Skeleton transaction generation: two postings per import row (source account + suggested expense account). Review queue: imported entries with `cleared=false`.
**Go concepts:** SQL schema migrations, strategy pattern for bank format parsers, `//go:embed` for default payee rules or schema SQL.
**Design:**
- `payee_rules` table: `pattern` (regex), `normalized_payee`, `default_account_id`, `priority`.
- Import: apply rules in priority order → set payee + pre-fill account on postings. Unmatched → review queue.
- `cleared=false` is the "needs review" signal — replaces the old category=="" heuristic.
- Migrate existing YAML rules into `payee_rules` table as part of this phase.

### Phase 15: Review TUI
**Feature:** Interactive review of uncleared entries. Account picker (hierarchical), memo input, tag toggler, split posting editor (add postings, verify sum=0), mark cleared on confirm.
**Go concepts:** complex multi-step state machines, custom input components, constraint enforcement in UI.

### Phase 16: Period & batch workflow
**Feature:** Monthly batch review built on `cleared` flag. Period as first-class concept.
**Go concepts:** `time.Time` range filtering, custom `DateRange` type, `sort.Slice` with multi-key comparisons.
**Design:**
- Period shown in title bar: "fintracker · March 2026"
- `[` / `]` navigate prev/next month
- Progress counter: "15/47 cleared"
- Smart default on startup: auto-select latest period with uncleared entries
- Sort: uncleared entries first (work queue), cleared sink below visual divider
- Period is global state — affects all views

### Phase 17: Structured filters
**Feature:** `f` opens filter mode with account path prefix matching, tag filter, cleared/uncleared status, date range. `F` clears all filters.
**Go concepts:** trie or prefix matching for account paths, composable predicates, builder pattern for filter chains.
**Design:**
- `f` → `(a)ccount (t)ag (u)ncleared (d)ate range`. Filters shown as pills in status bar. Multiple filters stack.
- Account filter uses prefix match: `Expenses:Food` matches all subaccounts.
- All filters compose through `refreshTable`: period → structured filters → text search → `filteredTxns`.
- `/` fuzzy search (already implemented) remains for quick text matching.

### Phase 18: Multi-view architecture
**Feature:** `tab` switches between views: Transactions (review/detail) and Reports (charts/KPIs).
**Go concepts:** interface-based view abstraction, per-view state vs global state, Model-per-view pattern in Bubble Tea.
**Design:**
- Period is global — switching views keeps the same period
- Each view has its own filter state and screen navigation
- Architecture: `Model { entries, period, activeView, txnView, reportsView }`
- Single source of truth means clearing in Transactions view is instantly reflected in Reports

### Phase 19: Statistics & reporting
**Feature:** Spending analytics built on the accounting model.
**Go concepts:** aggregate SQL queries, text-based chart rendering, formatting tables with computed columns.
**Ideas:**
- Balance by account (and subtree: `Expenses:Food` total)
- Spending by tag (trip totals across account boundaries)
- Month-over-month comparison
- Net worth over time (Assets − Liabilities)
- Top payees by spend

### Phase 20: Balance assertions
**Feature:** `fintracker verify` — check computed account balances against known bank statement values.
**Go concepts:** custom error types, accumulator pattern, CLI subcommands.
**Design:** `balance_assertions` table: `account_id`, `date`, `expected`. `verify` checks each assertion against summed postings.

### Phase 21: Journal export
**Feature:** `fintracker export --month 2026-03` — write cleared entries to hledger-compatible journal format.
**Go concepts:** `io.Writer` for output abstraction, text formatting, CLI flags.
**Format:** hledger journal (`2026-03-10 * ICA Maxi \n  Expenses:Food:Groceries  890 SEK\n  Liabilities:Creditcard:SEB`).

### Phase 22: Multiple bank format support
**Feature:** parse CSV from SEB, Swedbank, Nordea, ICA Banken.
**Go concepts:** strategy pattern via interfaces, factory functions, `//go:embed` for default configs.

### Phase 23: HTTP & APIs
**Feature:** GoCardless integration or local HTTP API.
**Go concepts:** `net/http`, `http.Client`, JSON, context with timeouts, middleware.

### Phase 24: Configuration
**Feature:** config file for database path, bank formats, rules, theme, keybindings.
**Go concepts:** `//go:embed`, config hierarchy (defaults → file → env → flags), XDG conventions.

### Phase 25: Distribution
**Feature:** installable via Nix, goreleaser, `go install`.
**Go concepts:** ldflags for version embedding, `buildGoModule` in flake.nix, goreleaser.

### Phase 26: Live theme awareness (DEC mode 2031 + OSC 11)
**Feature:** react to the terminal's light/dark theme flipping *while fintracker is running*, and harden the initial background-colour query.

**Current state:** `Init` fires `tea.RequestBackgroundColor` (an OSC 11 query) exactly once; `tea.BackgroundColorMsg` picks `RoséPineMain` vs `RoséPineDawn` (`internal/tui/model.go:85`, `:323`). Two gaps: nothing happens if the terminal theme changes mid-session, and there is no fallback if the terminal never answers the query.

**Simpler alternative — weigh this first (may make the whole phase moot):** stop shipping absolute hex colours and instead style everything from the **16 ANSI palette slots** (`lipgloss.Color("1")`…`"15"`, i.e. `ansi.Red`, `ansi.BrightBlack` …). The terminal already owns those slots and remaps them the instant its own theme flips — light/dark tracking, per-user themes and tmux all come for free, with zero escape sequences, no probe, no fallback chain and no terminal state to restore. Cost: we give up the exact Rosé Pine hues (users who theme their terminal in Rosé Pine get them anyway) and we're down to ~8 semantic slots, so the design-token layer has to be honest about which *roles* it really needs (accent / positive / negative / muted / border / highlight) rather than which 15 named colours. The `styles` struct is exactly the seam for that — views never touch `Theme`, so this is a rewrite of `theme.go` only. **Deferred: pick ANSI-only vs. the mode-2031 machinery below when we get here; ANSI-only currently looks like the elegant answer.**

**Design:**
- **Enable mode 2031 on startup** — `tea.Raw(ansi.SetModeLightDark)` (`\x1b[?2031h`). Supporting terminals then send an *unsolicited* OSC 11 report on every theme change, which bubbletea surfaces as another `tea.BackgroundColorMsg`. The existing handler already does the right thing with it.
- **Probe support first** — `tea.Raw(ansi.RequestModeLightDark)` (DECRQM, `\x1b[?2031$p`) → `tea.ModeReportMsg`; treat as supported when `msg.Mode == ansi.ModeLightDark && !msg.Value.IsNotRecognized()`. Store `m.supportsLightDark` for the help/debug view.
- **Reset on exit** — `ansi.ResetModeLightDark`. DEC modes are *terminal-global* state, not process state; leaving 2031 set leaks into whatever runs in that shell next.
- **Fallback chain** for initial detection: OSC 11 reply → `COLORFGBG` env var → config override → assume dark. The OSC 11 query needs a timeout — terminals that don't implement it simply never reply.
- **Luminance, not `IsDark()` alone** — compute relative luminance from the returned `color.Color` so a mid-grey background picks the right variant; leaves room for `RoséPineMoon` as "dark but softer".
- **Config hook (ties into Phase 24):** `theme = "auto" | "main" | "moon" | "dawn"`; an explicit choice must win over detection.
- **Consolidate the refresh** — the handler currently re-derives `styles`, the table style func and `help.Styles` inline. Factor into a single `func (m *Model) applyTheme(t Theme)` so a future style-carrying component can't be forgotten.

**Go concepts:** raw escape sequences as commands (`tea.Raw`), terminal state as a resource that must be restored (same discipline as `defer f.Close()`), message-driven state refresh, environment-variable fallback chains, a method that consolidates an invariant.

**Testing:** `Update` is pure — feed it a synthetic `tea.BackgroundColorMsg` / `tea.ModeReportMsg` and assert on `m.theme`. No PTY required.

**Note:** 2031 is implemented by Ghostty, WezTerm, contour, foot and recent iTerm2; Terminal.app and older tmux will fall through to the probe-failed path, which is exactly why the probe exists.


### Key binding plan (end state)
```
tab          switch view (Transactions ↔ Reports)
/            fuzzy search (text filter)
f            structured filter (account path, tag, cleared status)
F            clear all filters
[ / ]        prev / next period (month)
enter        drill into detail / review entry
esc          back / clear search
c            categorize / confirm posting account
m            add/edit memo
t            toggle tag
j/k          navigate
g/G          top / bottom
?            help toggle
q            quit
```

### Ongoing topics to weave in opportunistically

- Generics (utility functions, type constraints)
- Benchmarking (go test -bench, pprof)
- Linting (golangci-lint, staticcheck, exhaustive)
- Documentation (godoc, doc.go, example tests)
- Build tags (platform-specific, integration tests)
- Reflection (how struct tags work)
- Channel patterns (fan-out/fan-in, pipelines)
- sync package (Once, Pool, Map, atomics)
- go generate
- Payee normalization rules (migrate from YAML to DB-backed payee_rules)

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

**Remaining polish (carried forward):**
- Help as full screen (instead of inline toggle)
- Search input still jumps slightly (status line layout needs more tuning)

### Session 7 — rename + accounting model foundations (Phase 13 start)
**Date:** 2026-04-01
**Covered:** Naming cleanup, accounting roadmap planning, and first TDD cycle for the double-entry model.

**Rename:** `visibleIdx` → `filteredTxns` (Model field), `filtered` → `searchIdx` (TxnTable field), `SetFilter`/`ClearFilter` → `SetSearch`/`ClearSearch`, `FilteredCount` → `SearchedCount`. Pure mechanical rename — no logic changes. Used gopls LSP rename, verified with `go build ./...`.

**ACCOUNTING_ROADMAP.md:** Axel added a detailed accounting model design doc (double-entry bookkeeping: Account hierarchy, Posting, Transaction/Entry, Tags, payee_rules, balance assertions, journal export). Analysed fit with existing codebase and agreed on:
- Use `Öre` (not bare `int64`) for `Posting.Amount` — keeps type safety
- Roadmap phases 13–21 rewritten around the accounting model (see updated roadmap above)
- `cleared` flag (from accounting model) unifies with the old "batch review" concept from Phase 14
- YAML categorization rules will be migrated to `payee_rules` table in DB

**Phase 13 — TDD, core accounting types:**
- `internal/finance/ledger.go` — `Entry` (future `Transaction`), `Posting`, `Validate()`
- `internal/finance/account.go` — `Account`, `AccountType` constants, `Name()`/`Parent()`/`Depth()` methods
- Tests in `ledger_test.go` and `account_test.go` — full red-green cycle, all passing
- `Validate()` uses map accumulator `map[string]Öre` — same pattern as `buildAccountSummary`, now the canonical Go idiom for groupBy+fold
- `Parent()` on a root account (`"Equity"`) returns `""` for free via `strings.Join(s[:0], ":")` — zero value covers the edge case

**Concepts practiced:** TDD red-green cycle, pure function testing as specification, named types for type safety (`Öre` vs `int64`), map accumulator pattern, `strings.Split`/`Join` for path manipulation, zero value correctness.

**Next:** Phase 13 continues — DB schema migration: `accounts` table + new `transactions`/`postings` tables in `internal/store/store.go`. Will cover SQL migrations in Go (schema versioning with `PRAGMA user_version` or a migrations table).

### Session 8 — schema migrations + store methods (Phase 13 continued)
**Date:** 2026-04-13 to 2026-04-14
**Covered:** Schema migration system and store layer for the double-entry model.

**Migration system:** Replaced inline `CREATE TABLE IF NOT EXISTS` with a versioned migration runner using `PRAGMA user_version`. Migrations are a `[]func(*sql.Tx) error` slice — index is version number, each runs in its own transaction. `NewStore` calls `migrate(db)` which reads current version and runs pending migrations in order. Design discussion: `PRAGMA user_version` (SQLite-specific, simple) vs migrations table (portable, more flexible) — chose the former for an embedded SQLite app.

**Migration 1→2:** Added `accounts`, `entries`, `postings`, and `entry_tags` tables. Design decisions:
- String paths for account hierarchy (`Assets:Checking:SEB`) over adjacency list — simpler queries (`LIKE 'Expenses:%'`), matches hledger notation, Go code already works with `strings.Split`. Tradeoff: no referential integrity on parent segments, but enforced in Go layer.
- `entry_tags` junction table over JSON array in TEXT column — normalized, queryable, idiomatic relational modeling.
- `ON DELETE CASCADE` for postings/tags (aggregate root: entry owns them), `ON DELETE RESTRICT` for accounts (independent existence).
- `cleared INTEGER` — SQLite has no boolean type, 0/1 convention.
- Composite primary key `(entry_id, tag)` on `entry_tags` — no synthetic ID needed, prevents duplicate tags, free unique index.

**Store methods:**
- `InsertAccount` / `LoadAccounts` — straightforward CRUD. `*time.Time` for nullable dates maps to SQL NULL automatically. `ORDER BY path` naturally groups parent before children.
- `InsertEntry` — coordinates three inserts (entry, postings, tags) in one `sql.Tx`. Calls `Validate()` before touching the DB — domain invariant enforced at store boundary. Prepared statement for postings (amortized), inline exec for tags.
- `LoadEntries` — map-assemble pattern: three independent queries (entries, postings, tags), assemble in Go via `map[int64]*finance.Entry`. Pointer values in map so `append` to Postings/Tags mutates the right entry. `entryOrder []int64` preserves query ordering since maps are unordered.

**Concepts practiced:** `PRAGMA user_version` for schema versioning, migration-per-transaction pattern, `defer tx.Rollback()` as no-op after commit, `sql.Stmt` vs `sql.Result` (Close semantics), `sql.Rows` resource lifecycle, map-assemble pattern for avoiding complex JOINs, aggregate root in DB design (CASCADE vs RESTRICT), `defer` evaluation semantics (receiver captured at defer-time, not at function exit).

**Next:** Phase 13 wrap-up — potentially add `UpdateEntry`, test edge cases (duplicate account path, foreign key violation on bad account_id). Then Phase 14: import pipeline v2 (CSV → Entry+Posting pairs, payee_rules table).

### Session 9 — Phase 13 wrap-up + Phase 14 start (import pipeline v2)
**Date:** 2026-04-21
**Covered:** Wrapped up Phase 13, then built the foundation of Phase 14.

**Phase 13 wrap-up:**
- Fixed stray `2` in `Validate()` error message (`%d2` → `%d`) — caught by code review, `go vet` would have caught the `%d`/struct mismatch variant
- `UpdateEntry(e finance.Entry) error` — updates mutable header fields (payee, memo, cleared) by ID. Design: whole-entity update (caller holds full entry, mutates field, passes back) vs per-field methods — whole-entity is idiomatic for CRUD where the caller always has the full struct.
- Edge case tests: `TestInsertAccountDuplicate` (UNIQUE constraint), `TestInsertEntryValidationFail` (Go-layer reject before DB), `TestInsertEntryForeignKeyViolation` (FK constraint with `PRAGMA foreign_keys=ON`). The FK test used balanced postings with nonexistent account IDs — intentionally isolates the DB-layer constraint from the Go-layer `Validate()`.
- Layered constraint model: Go enforces business invariants (`Validate()`), SQLite enforces data integrity (UNIQUE, FK). Tests should know which layer they're testing.

**Phase 14 — import pipeline foundation:**
- `payee_rules` migration (2→3): `ON DELETE SET NULL` — third FK strategy: referenced row deletion nulls the FK column rather than cascading or blocking. Right choice for rules that survive account deletion.
- Strategy pattern: `BankFormat` interface (`Parse(io.Reader) ([]RawRow, error)`), `SEBFormat` implementation. `RawRow` is the decoupled intermediate type — parser knows nothing about accounts or rules. `account` parameter dropped from `parseRow` because `RawRow` doesn't carry source account — that's injected by the caller at `Import` time.
- Value vs pointer receiver: `SEBFormat` is stateless — value receiver, so both `SEBFormat{}` and `&SEBFormat{}` satisfy the interface.
- `ParseAmount` duplication resolved: `importer/seb.go` calls `parser.ParseAmount` — `importer → parser → finance` is a valid dependency chain.
- `//go:embed`: compiler directive embeds `default_rules.yaml` into the binary as `[]byte`. Must be on the line immediately before the `var`. Requires `_ "embed"` blank import (side-effect import, same pattern as `_ "modernc.org/sqlite"`). File becomes a build dependency — deleting it breaks the build.
- `Import` function: `format.Parse` → sort rules by priority → match each row → build `Entry` with two postings (source + counter). Signing: CSV amount goes to source posting unchanged; counter posting gets `-amount`. Sum = 0, `Validate()` passes. `ImportResult{Entries, Unmatched}` splits matched vs unmatched rows cleanly.
- `matchRule`: case-insensitive `strings.Contains`, first match wins (rules pre-sorted). Mirrors `Categorize` in `finance/categorize.go`.
- `copy(sorted, rules)` before `sort.Slice` — defensive copy prevents mutating the caller's slice.

**Concepts practiced:** `ON DELETE SET NULL`, strategy pattern via interfaces, `//go:embed` + blank import, value vs pointer receivers, double-entry signing convention, `ImportResult` as a result type splitting two outcome classes, defensive copy before sort.

**Next:** Phase 14 continues — `LoadPayeeRules`/`InsertPayeeRule` store methods, YAML→DB rule migration, wire `Import` into the TUI import flow (replace old `parseAllFiles` → `UpsertTransactions` chain).

### Session 10 — Phase 14 continued (store methods, seeding, `*int64` nullable pattern)
**Date:** 2026-06-03 to 2026-06-07
**Covered:** Store CRUD for payee rules, nullable FK pattern, embedded YAML seeding.

**`PayeeRule` moved to `finance` package:** Originally in `importer` — but `store` also needs it for CRUD. Moving to `finance` avoids `store → importer` dependency; both `store` and `importer` already import `finance`. Domain types belong in the leaf package.

**`*int64` for nullable FK:** `DefaultAccountID *int64` replaces `int64` — `ON DELETE SET NULL` can produce NULL, and `0` is an ambiguous sentinel. Mirrors `*time.Time` for nullable dates on `Account`. Cascading fix: `matchRule` returns `*int64`, posting construction dereferences with `*rule.DefaultAccountID`, test literals need a `ptr(n int64) *int64` helper since Go can't take the address of numeric literals.

**`sql.NullInt64`:** Standard library bridge for nullable integer columns. `{Int64 int64; Valid bool}` — scan into it, check `.Valid`, convert to `*int64`. Predates generics; Go 1.22 added `sql.Null[T]` but `NullInt64` is still idiomatic.

**Store methods:**
- `InsertPayeeRule` / `LoadPayeeRules` — standard CRUD with `sql.NullInt64` scan for nullable FK. `ORDER BY priority, id` for deterministic ordering.
- `SeedPayeeRules(defaults []finance.PayeeRule)` — idempotent: checks `COUNT(*)`, inserts only if table is empty. Schema migrations are for structure, seeding is for data — kept separate so `default_rules.yaml` can change without new migrations.

**Embedded YAML seeding:**
- `yamlRule` struct with struct tags — separate from `finance.PayeeRule` because YAML has no `id` or `default_account_id`. Bridge function `DefaultRules()` parses `defaultRules` bytes and converts.
- Fixed typo in `default_rules.yaml` (`patter` → `pattern`) — YAML silently ignores unknown keys, so the rule would have had an empty pattern matching everything.

**Concepts practiced:** Nullable FK modelling (`*int64` vs sentinel), `sql.NullInt64` for scanning nullable columns, Go's inability to take address of literals (and the `ptr()` helper pattern), idempotent seeding vs schema migration, YAML struct tags as bridge types, package dependency direction (domain types in leaf package).

**Next:** Phase 14 wrap-up — wire `Import` into the TUI import flow, replacing `parseAllFiles` → `UpsertTransactions` with `BankFormat.Parse` → `Import` → `InsertEntry`. Call `SeedPayeeRules` + `LoadPayeeRules` at startup.

### Session 11 — Phase 14 continued (account resolution, placeholder entries, errors.Is)
**Date:** 2026-06-07
**Covered:** Pre-wiring design decisions and implementation: CLI spec format, account resolution, unmatched-row policy.

**Design decision — CLI spec separator:** `account:path` broke once accounts became hierarchical paths (`Assets:Bank:SEB` contains colons). Chose `=` separator: `Assets:Bank:SEB=testdata/seb.csv`. One-line change in `args.go` (`SplitN(arg, "=", 2)`).

**Design decision — account resolution at import:** validated get-or-create. `AccountTypeFromPath(path)` (package-level function in `finance/account.go`) validates the first path segment against the five `AccountType` constants and infers the type — typos in the first segment fail loudly. `EnsureAccount(path)` in store does lookup-or-insert.

**`errors.Is` and sentinel errors (finally checked off):** `sql.ErrNoRows` as the canonical sentinel error. `EnsureAccount` distinguishes three outcomes: found (return ID), not-found (`errors.Is(err, sql.ErrNoRows)` → take the create branch), real error (propagate). Modern Go rule: never `==` on errors — `errors.Is` walks the `%w` unwrap chain. Bare `switch { case ... }` as a cleaner if/else-if chain.

**Design decision — unmatched rows policy:** Unmatched rows can't become entries directly (no counter account, `Validate()` requires two balanced postings) but must not be lost on quit. Chose placeholder account parking: unmatched rows become real entries with counter posting → `Equity:Uncategorized`, persisted `cleared=false`.
- Why Equity (Axel's catch): unmatched rows aren't necessarily expenses — a salary parked in `Expenses:Uncategorized` would show as negative expense. Equity is accounting's type-neutral "no claim made" bucket (cf. `Equity:Opening-Balances`); pollutes neither expense nor income reports, and a non-zero Equity balance is a loud "review me" signal rather than plausible-looking wrong data.
- Review-everything: ALL imported entries get `cleared=false` (matched rules are suggestions, not truth). Already the behavior via zero value — `Import` never sets `Cleared`.
- Mechanism in importer, policy in caller: `PlaceholderEntries(rows, sourceAccountID, placeholderAccountID)` is a pure function; the caller decides whether/which placeholder account. Sign-splitting later = caller filters by sign, calls twice.

**Test discipline:** subtest auto-naming (`#00` when name is empty string), test-data bugs vs code bugs (failing test where the test table was wrong, twice), regression test with both amount signs pinning the salary case.

**Naming:** `PlaceHolderEntries` → `PlaceholderEntries` — "placeholder" is one word; mid-word caps are like writing `DataBase`.

**Known gap (next session opener):** import idempotency is LOST in the new pipeline. Old `transactions` table had `unique(date, amount, payee, account)` + upsert; `entries` has no unique constraint and identifying data spans two tables. Re-importing the same CSV duplicates everything. Needs design (import hash? dedup query? import log?) + migration BEFORE wiring, or first re-import double-counts balances.

**Concepts practiced:** sentinel errors and `errors.Is`, conditionless `switch`, get-or-create pattern, pure-function design for testability, zero value correctness (`Cleared`), policy/mechanism separation, accounting equity semantics.

**Next:** dedup design discussion → then the wiring: `SeedPayeeRules` in `main.go`, rework `importAllCmd`/`parseAllFiles` (EnsureAccount sequential before fan-out, parse parallel, InsertEntry sequential after join — no DB in goroutines, SQLITE_BUSY), display shim for entries.

### Session 12 — Phase 14 continued (import dedup, custom sentinel error)
**Date:** 2026-06-13
**Covered:** The import-idempotency design problem and its full implementation.

**The identity problem:** SEB CSV gives only date/amount/payee — no bank transaction ID. "Don't import twice" requires defining what makes two rows the same, and the killer test case is two legitimately-distinct identical rows (two 45 kr coffees, same day). Realized the OLD `unique(date,amount,payee,account)` upsert had been silently *merging* the second coffee all along — a latent data-loss bug in the flat model.

**Chosen design — content hash + occurrence counter (option C):** `hash(date | amount | raw_payee | n)` where `n` counts prior identical rows *within the same file*. Re-import same file → identical hashes → deduped. Overlapping rolling-window exports → shared rows get same hashes → deduped. Two distinct coffees → different `n` → both kept. This is how OFX FITID synthesis works when banks omit IDs. Rejected: porting old unique constraint (fails coffee test), whole-file hashing (fails on overlapping exports).

**`stampHashes(rows []RawRow)`:** unexported step of `Import`, runs over the WHOLE file before the matched/unmatched split (so the counter sees every row). `map[string]int` occurrence accumulator (same pattern as `Validate()` currency sums). `crypto/sha256` + `encoding/hex`. Count is the last `|`-delimited token and always digits → no collision even with `|` in payee. Increment-before-read (1,2,3…) vs after (0,1,2…) doesn't matter — only determinism + distinctness do, and both hold. Mutates via `rows[i].Hash` (Session 6: `range` value copy can't mutate the slice).

**Schema (migration 3→4):** `alter table entries add column import_hash text` + `create unique index idx_entries_import_hash`. SQLite UNIQUE indexes permit unlimited NULLs → manual entries (no hash) never collide.

**`Entry.ImportHash string` — why string not `*string`:** both nullable, but choose pointer only when the zero value could be mistaken for real data. Account ID `0` looks plausible → needed `*int64`. Empty string is never a valid SHA hex digest → `""` is self-evidently absent. Rule refined: "nullable → pointer" is wrong; "ambiguous zero value → pointer" is right. NULL conversion happens at the store boundary, domain type stays pointer-free.

**Custom sentinel error `store.ErrDuplicateEntry` (producer side of the pattern):** `var ErrDuplicateEntry = errors.New(...)`, package-level, exported, `Err` prefix. `InsertEntry` insert grows `on conflict (import_hash) do nothing` (precise — suppresses ONLY hash collisions, other constraints still error, unlike `insert or ignore`), checks `RowsAffected()==0`, returns the sentinel WRAPPED with context (`%w`) so `errors.Is` still finds it through the chain. Caller treats it as "skipped" not "failed". This session covered BOTH sides of the sentinel pattern: consuming `sql.ErrNoRows` (EnsureAccount) + producing `ErrDuplicateEntry`.

**`sql.NullString` both directions:** write side — `sql.NullString{String: e.ImportHash, Valid: e.ImportHash != ""}` so `""`→NULL. Read side (`LoadEntries`) — scan into `sql.NullString` intermediary, `e.ImportHash = hash.String` (`""` when NULL, no guard needed unlike the `*int64` address-taking case). Found via test failure: `converting NULL to string is unsupported`.

**Test discipline:** store dedup test uses a FIXED string hash (`"test-hash-1"`), NOT a computed one — store enforces uniqueness, doesn't care about hash content; computation is tested separately in importer. Caught a test-data bug (postings referencing phantom account IDs → FK violation; fixed by `InsertAccount` first like `TestInsertAndLoadEntry`). "Impossible" failure message (error printing two EQUAL strings) = stale test binary / inverted condition — check what actually ran.

**Concepts practiced:** content-addressable dedup / synthetic IDs, occurrence-counter accumulator, custom sentinel errors (producing, not just consuming), `errors.Is` through `%w` wrapping, `sql.NullString` read+write, pointer-vs-value field choice by zero-value ambiguity, `on conflict do nothing` vs `insert or ignore`, test isolation (don't couple a store test to an importer algorithm).

**Next:** the actual TUI wiring. `SeedPayeeRules(DefaultRules())` in `main.go` after `NewStore`. Rework `importAllCmd`/`parseAllFiles`: `EnsureAccount` (source paths + `Equity:Uncategorized`) sequential before fan-out, `importer.Import` + `PlaceholderEntries` parallel in goroutines (pure, no DB), `InsertEntry` loop sequential after join (SQLITE_BUSY — no concurrent writers), count `ErrDuplicateEntry` as skipped in `ImportDoneMsg`. Then the display question: views consume flat `Transaction` — need an `Entry`→display shim or start migrating views (Phase 15 territory).

### Session 13 — TUI migrated to the double-entry model + import pipeline wired (Phase 14, near-complete)
**Date:** 2026-06-14 to 2026-06-17
**Decision:** No display shim — migrate the Model + views to consume `[]finance.Entry` directly, retire the flat `Transaction`, then rename `Entry` → `Transaction` (the roadmap's intended end state). Big blast radius, done in sequenced steps; package compiled green at the end of each step.

**`entryView` + `projectEntry` (`internal/tui/entryview.go`):** the chokepoint that flattens a header+postings `Entry` into the one row the UI shows. View-model, NOT a shim — Model's source of truth is `[]finance.Entry`; `entryView` is a throwaway render-time projection (domain model vs view model = healthy split). Rule: view from the **asset/liability** posting → it gives Account + Amount; the contra posting gives Category; `"(split)"` when >2 postings. `projectEntry` finds the FIRST asset/liability posting (break — don't let later asset postings clobber, matters for transfers; import convention puts source posting first and the DB preserves order via `order by id`). Fully tested incl. order-independence + transfer + split.

**Model migration (`model.go`, `views.go`, `styles.go`):** `transactions []finance.Transaction` → `entries []finance.Entry`; added `accountsByID map[int64]finance.Account` (resolves posting AccountID → path/type, loaded once via `LoadAccounts`); `filteredTxns`→`filteredEntries`; `selectedTxn`→`selectedEntry` (returns `*finance.Entry`); `transactionStyleFuncFromIdx`→`entryStyleFuncFromIdx`. All display helpers (`buildRowsFromIdx`, summaries, `collectAccounts`, styleFunc, `renderDetail`) now project entries. **Field `totalBalance` renamed `netWorth`** — summing ALL postings in a balanced ledger is always 0, so total must mean something specific: net of Asset+Liability postings. Summaries split by account type: `buildAccountSummary` = Asset/Liability balances, `buildCategorySummary` = Income/Expense balances (every account has a balance; "spent on groceries" = balance of `Expenses:Food:Groceries`). styleFunc dims the category cell on `!e.Cleared` (review state) instead of the old `Category==""`.

**Import pipeline rewired (`import.go`, `importAllCmd` in `model.go`):** `parseAllFiles`→`importAllFiles` returning `[]finance.Entry`, PURE (no DB) — same errgroup/`select`+`ctx.Done()` leak guard from Sessions 3–4, carried forward unchanged. Orchestration shape in `importAllCmd`: (1) `EnsureAccount` for `Equity:Uncategorized` + each spec's source path — DB WRITES, sequential, BEFORE fan-out; (2) `LoadPayeeRules`; (3) `importAllFiles` parallel parse+`importer.Import`+`PlaceholderEntries` (pure); (4) `InsertEntry` loop sequential AFTER join, `errors.Is(err, store.ErrDuplicateEntry)`→skip (count `skipped`), other err→`ImportErrMsg`, else `inserted`. No DB writes in goroutines — SQLite tolerates concurrent readers, not writers. `ImportDoneMsg` gained `Skipped int`; status line shows "(N new, M duplicates)".

**Bug caught & fixed (`importer.Import`):** matched rule with `nil DefaultAccountID` (every freshly-seeded rule, since `DefaultRules()` sets no account) → `*rule.DefaultAccountID` panic. Fix: `if !ok || rule.DefaultAccountID == nil { → Unmatched }`. Account-less match routes to review queue (placeholder), losing normalized payee until Phase 15 assigns the account. This is the `*int64` (Session 10) optional-as-pointer choice billing us: every `*T` deref across a boundary is a nil-check Go won't enforce.

**Concepts practiced:** domain-model vs view-model projection, asset-side ledger register semantics, net worth vs naive sum, summaries as per-account-type balances, pure-function fan-out (DB writes bracket the parallel section), `errors.Is` sentinel-skip in a real loop, nil-pointer discipline on optional `*T` fields, sequenced big-refactor (compile-green per step), Go field/function namespace separation (`m.netWorth` field + `netWorth()` func coexist).

**Status:** all packages build; `go test ./...` green; working tree committed at `550d723` "Wire up new double-entry ledger into tui".

### Session 14 — Phase 14 COMPLETE: seed rules, verify pipeline, retire flat model, rename Entry → Transaction
**Date:** 2026-06-17 to 2026-07-07
**Covered:** Worked the entire Phase-14-wrap handoff to completion. Phase 14 is now done.

**1. `main.go` rule seeding (handoff step 1):** Added `importer.DefaultRules()` + `s.SeedPayeeRules(defaults)` after `NewStore` (idempotent — `COUNT(*)` guard). Seeding ≠ migration: migrations version *structure* (immutable), seeding inserts *data* (safe every startup). Caught a missing `os.Exit(1)` — a printed error is not a handled error; at `main()` top-level, fail loud.

**2. End-to-end verification (handoff step 2) — found and fixed real bugs:**
- **Empty `accountsByID` after import:** the `ImportDoneMsg` handler loaded accounts but never populated the map (drifted from `InitialModelFromStore`). Map zero-value trap: `accts[id]` on an absent key returns a zero `Account` silently — no panic, just blank cells. This is deferred-polish item (c) biting: duplicated reload sites diverge. Fixed with the populate loop; full `reload()` DRY still deferred.
- **Latent `selec count(*)` SQL typo** in `SeedPayeeRules` (dead since Session 10) — surfaced the moment step 1 *called* it. Lesson: wiring up dead code exposes latent bugs; SQL-in-strings is invisible to the Go compiler.
- **Empty payee column** turned out to be *correct* behavior (all rows unmatched → review queue → normalized `Payee` blank by design) but a real UX gap. Added `Transaction.DisplayPayee()` (value receiver): prefer curated `Payee`, fall back to `RawPayee`. Swapped 5 display sites. Subtlety: line-104 grouping comparison is *logic*, not display — using `DisplayPayee()` there also fixed the empty-payee-groups-everything bug (deferred item b). First flipped the condition wrong (`RawPayee == ""` → would show raw forever even after curation); corrected to test the preferred field.
- Verified via `sqlite3`: 8 transactions, 16 postings, all balanced (sum=0), all `cleared=0`, unmatched parked on `Equity:Uncategorized`; re-import → "0 new, 8 duplicates" (hash dedup works).

**3. Retire the flat model (handoff step 3) — compiler-driven, top-down:** Deleted consumers before definitions so `go build ./...` stayed green at each step (Go tolerates unused *exported* funcs — only unused *locals*/*imports* error). Order: tui+main → store → parser → finance. Removed: the fake `c` categorize screen entirely (Axel's call — the flat "category string" is the wrong abstraction; Phase 15 needs a hierarchical account picker), `rules`/`categories` fields, `collectCategories`, `catInput`/`catKeys`; store `Upsert/Load/UpdateCategory` methods; `finance.Transaction`(flat)/`CalculateBalance`/`categorize.go`. **Bonus:** the `parser` package collapsed to just `ParseAmount` (the `BankFormat` strategy in `importer` had replaced `ParseTransactions`) — folded `ParseAmount` into `finance` and *deleted the parser package*. Renamed `transaction.go` → `ore.go`. Since nothing is deployed, also rewrote history: deleted the old `transactions`-table migration (index-addressed slice → all versions shift down; fresh DB now ends at `user_version=3`; `rm` local dev DBs).
- **Blind-spot theme (recurred 3×):** the compiler catches unused locals/imports but **not unused struct fields** (`catInput` survived the first pass); gopls/vet don't see inside **strings** (the `selec` typo, a stale `projectEntry()` error-message literal) or **comments**. A grep for the old name after any rename is cheap insurance.

**4. Rename `Entry` → `Transaction` (handoff step 4):** gopls LSP rename on the type — rewrites the declaration, both method receivers, and all `finance.Entry` refs across 4 packages atomically, leaving substring-only identifiers alone. Axel propagated peripheral names too: store methods (reusing the freed `LoadTransactions`), `transactionView`/`projectTransaction`, `selectedTransaction`, `filteredTransactions`, and — by hand, since gopls can't see SQL — the schema (`entries`→`transactions`, `entry_tags`→`transaction_tags`, `entry_id`→`transaction_id`).

**Concepts practiced:** map zero-value trap, value-receiver derived accessor, display-vs-logic distinction, `os.Exit` as top-level error handling, end-to-end verification catching what unit tests miss, compiler-driven (follow-the-compiler) refactoring, unused-import/local vs unused-field blind spots, tools-see-AST-not-strings, index-addressed migration renumbering, strategy pattern superseding a package, semantic rename via gopls.

**Status:** all packages build; `go test ./...`, `go vet` green. Phase 14 pipeline verified end-to-end against `testdata/seb.csv`. Committed in three units: 🐛 seed+display fixes, 🔥 retire flat model, ♻️ rename Entry → Transaction.

---

### Session 15 — Phase 15 MVP: the Review TUI (with atomic commit)
**Date:** 2026-07-17 to 2026-07-22
**Covered:** Built the review flow end-to-end — turning imported `cleared=false` transactions into cleared, categorized ones — then hardened the commit into a single atomic DB transaction.

**Warm-up — `reload()` extraction (deferred item c, now safe):** `func (m *Model) reload() error` — the single store→model sync path (load transactions+accounts, build `accountsByID`, recompute netWorth/summaries/accounts, `refreshTable`). Pointer receiver (mutates Model in place; Go auto-addresses the value copy in `Update`). Wired into both `InitialModelFromStore` (constructor now assembles the UI shell only, then calls `reload()` — table built with empty rows, `refreshTable` fills them; `initialVisibleIdx` deleted) and the `ImportDoneMsg` handler (~20 lines → 3). Kills the divergence class that caused Session 14's empty-`accountsByID` bug.

**Store — `UpdatePosting` + the `dbtx` pattern:** `UpdatePosting(postingID, accountID)` repoints one posting (`RowsAffected()==0` → not-found guard). Then the atomicity refinement introduced `dbtx` — an interface (`Exec`/`QueryRow`) satisfied by BOTH `*sql.DB` and `*sql.Tx` via structural typing (no stdlib equivalent; you define your own). Extracted `ensureAccount`/`insertAccount`/`updatePosting` as free functions taking `dbtx`; public methods delegate with `s.db`. `ReviewTransaction(txID, postingID, path)` composes `ensureAccount(tx,…)` + `updatePosting(tx,…)` + clear inside ONE `sql.Tx` with `defer tx.Rollback()` — crash-safe: a mid-way failure rolls back even a freshly-created account. Extract-on-second-caller (updatePosting has 2 callers → helper; the one-line clear stays inline, YAGNI).

**TUI — the review screen:** new `reviewScreen`, entered with `c` (the freed key) on an uncleared row (guards: not cleared, has a contra posting). `reviewInput` is a `textinput` with `ShowSuggestions` (autocomplete over `accountPaths(m.accountsByID)`). `contraPosting(t, accts) (Posting, bool)` finds the non-asset/liability posting to categorize (the `accts` map IS the resolver — no store call). `updateReview` handles esc/enter; enter calls `ReviewTransaction` then `reload()`. `renderReview` shows txn + input + inline `reviewErr`.

**Bugs caught (the session's recurring themes):**
- **Grouped-param type gotcha:** `func(txID, postingID, accountPath string)` typed ALL three as string — `txID`/`postingID` must be `int64` (same "grouping applies to all" as struct fields). Also the `si`/`ri` sibling: `ri := textinput.New()` was configured but never assigned into the `Model` literal → zero-value textinput → nil cursor → panic on `Focus()`. Value semantics: the local and the struct field are two values.
- **`defer tx.Rollback()` before the Begin error-check** → nil-`tx` Rollback panic masking the real error. Defer must come AFTER the validity check.
- **Missing `UpdateTransaction`** in the first (pre-atomic) confirm handler → `cleared` set in memory then clobbered by `reload()`; the category flipped (persisted via `UpdatePosting`) so it *looked* done. Lesson: verify the invariant, not a proxy.
- **`reviewErr` state-lifecycle leak:** cleared on retry+success but not on entry+cancel → stale error rendered on the next review. Ephemeral UI state needs a reset wired to EVERY transition. Same root cause as the (still-deferred) `importStatus` that never clears.
- **Inverted test assertion** (`if err != nil` where the not-found case *wants* an error) → a correct method failed the test. When the message and the condition disagree, re-derive.

**Concepts practiced:** pointer-receiver mutation + Go-1.22 per-iteration loop vars (`&acc` now safe), the `dbtx`/querier interface (structural typing over `*sql.DB`/`*sql.Tx`), multi-statement atomic transactions (`Begin`/`defer Rollback`/`Commit`), extract-on-second-caller, `textinput` autocomplete (`ShowSuggestions`), TUI screen state machine + ephemeral-state lifecycle, `t.Run` subtests, atomic-rollback as a test assertion, `go vet` printf verbs (`%d` vs `%q`).

**Status:** all packages build; `go test ./...`, `go vet` green. Review flow verified in the running TUI. Pre-existing `gofmt` drift in `ledger.go`/`ledger_test.go`/`importer.go` (untouched this session) — `gofmt -w internal/` to tidy.

---

### Session 16 — Phase 15 ext (c): write a payee_rule on review (loop closed & verified)
**Date:** 2026-07-24
**Covered:** Closed the Phase-14 loop — a categorization made during review now becomes a persisted `payee_rule`, so future imports auto-match. Store method + a new editable-pattern prompt screen, debugged the state-transition bugs, verified end-to-end against a scratch DB.

**Store — `EnsurePayeeRule(r) (created bool, err error)`:** get-or-create keyed on `(pattern, default_account_id)`, using the `errors.Is(err, sql.ErrNoRows)` sentinel pattern (same as `EnsureAccount`). Rejects a nil `DefaultAccountID` up front. Test covers create → idempotent no-op → nil-account error. Surfaced a `%q`-on-int in the test message (`want %q` with `1` → `'\x01'`); `go vet` doesn't flag it because `%q` is *valid* for integers (rune-quoting) — tools check the type, not the intent (recurring Session-14 theme).

**TUI — the rule-prompt screen (editable pattern):** new `rulePromptScreen` + `ruleInput textinput` + `ruleErr`. After a successful `ReviewTransaction`, `updateReview` routes to the prompt (unless the account is the placeholder or the payee is blank), seeding `ruleInput` with the raw payee for the user to trim (`ICA MAXI MALMÖ` → `ICA`). `updateRulePrompt` resolves path→ID via `EnsureAccount`, calls `EnsurePayeeRule`, reports created/exists in the status line. Chose editable-pattern over y/n because SEB raw payees carry per-transaction junk — a verbatim-raw-payee pattern would almost never re-fire.

**Bugs caught (all state-transition bugs — the class we predicted going in):** (1) success path set the status but never returned to `listScreen` nor reset state — the Session-15 `reviewErr` lifecycle lesson, missed on the *happy* path because it "feels done" once the save succeeds; (2) empty-pattern guard set the error but forgot to `return`, so it fell through and would have saved a `Pattern==""` catch-all (the Session-10 empty-pattern trap); (3) `created` branch inverted; (4) transition returned `m, nil` without `m.ruleInput.Focus()` — bubbles `textinput.Update` early-returns when unfocused, so the "editable" field silently dropped every keystroke (Focus is the keyboard-routing signal, not just a cursor); (5) error paths tore down `pendingRuleAccount`/blurred the input while staying on-screen, killing retry — teardown belongs on the *leaving* paths (success + skip), not error.

**Growing hack noted:** the global Quit guard now excludes `reviewScreen && rulePromptScreen` by name — deferred-item-(c) state-machine smell scaling exactly as predicted; a `mode` enum would collapse it.

**End-to-end verification (scratch DB, isolated binary):** imported `testdata/seb.csv` (all rows → `Equity:Uncategorized`, `cleared=0`), reviewed one ICA row → `Expenses:Food:Groceries`, trimmed pattern to `ICA`, saved. Confirmed via `sqlite3`: rule row `pattern=ICA, priority=0, account=Groceries` sits ahead of the seeded `ICA/priority=10/nil`; reviewed txn repointed + `cleared=1`, balanced. Then imported a *fresh* file (`ICA NÄRA LUND`) → rule fired automatically (counter → Groceries), `PRESSBYRÅN` → Equity (control). Loop closed.

**Precedence finding:** user rules get `Priority: 0` (zero value) < seeded `10`, so `Import`'s ascending-priority sort + first-match puts them ahead — the loop closes *because* of that ordering. Fragile: `matchRule` returns the first *pattern* match, then `Import` rejects it if the account is nil — so a same-priority nil-account rule could shadow a usable one and wrongly send a row to the queue. Design smell for later.

**Temporal-application realization (drove the next decision):** rules fire only at import time, so rows imported *before* a rule was written stay in the queue (and content-hash dedup blocks re-import from fixing them). Axel hit this immediately ("but the other ICA rows aren't categorized!"). Decided: **re-point on save** — saving a rule immediately re-points matching *uncleared, placeholder-parked* transactions to the rule's account (still `cleared=0` — fill blanks, never overwrite intent). Deferred to next session.

**Concepts practiced:** get-or-create + `errors.Is`/`sql.ErrNoRows` sentinel, screen state-machine + ephemeral-state lifecycle (reset on every *leaving* path), `textinput` Focus as the keyboard-routing signal, `%q`-vs-`%d` and tools-see-types-not-intent, rule precedence via priority sort, content-hash dedup × re-import interaction, end-to-end DB verification catching what unit tests miss.

**Status:** all packages build; `go test ./internal/tui ./internal/store` green; `go vet` clean. Loop verified end-to-end. Committed at `95dea79` "Add ruleInputPrompt screen" (incl. +4 `testdata/seb.csv` rows for richer manual testing).

---

### Session 17 — re-point on save (retroactive rule application) + review-indicator WIP
**Date:** 2026-07-27 to 2026-08-10
**Covered:** Built and verified re-point-on-save (the Phase-15-ext-(c) refinement). Then started a table review-state indicator (in progress, uncommitted).

**Re-point on save — 3 steps, all done & committed (`bd6c025`):**
1. **`finance.PayeeRule.Matches(rawPayee string) bool`** — lifted the case-insensitive-`Contains` predicate out of `importer.matchRule` into the domain type (value receiver, like `DisplayPayee`); `matchRule` now delegates. Behavior-preserving extraction — the existing importer match tests going green *was* the proof nothing changed. One source of truth so importer and TUI can't diverge (matters the day patterns become regex).
2. **`store.RepointPostings(postingIDs []int64, accountID int64) error`** — bulk re-point in one `sql.Tx`, looping the `updatePosting(dbtx, …)` helper (Session 15); no clearing. The `updatePosting` not-found guard means **one bad ID aborts the whole batch** — the atomicity we want. Test includes the rollback-as-assertion case (batch with a bad ID leaves the good row unmoved). Empty slice → deliberately a no-op (opens+commits an empty tx; not worth a guard).
3. **Wired into `updateRulePrompt` success path** — build a match probe `finance.PayeeRule{Pattern: pattern}`, loop `m.transactions` collecting `contra.ID` where **all** of `!t.Cleared`, `rule.Matches(t.RawPayee)`, and `m.accountsByID[contra.AccountID].Path == placeholderAccount` hold; then `RepointPostings` + `reload()` + fold count into `importStatus`.

**Bug caught (skip vs abort):** the sweep first wrote `if !ok { m.ruleErr = …; return }` on `contraPosting`'s second return — Axel flagged it himself with `TODO: not sure makes sense?`. A `!ok` means a **transfer** (all postings on Asset/Liability, no contra leg) — a normal non-candidate, not an error. As written, a single transfer anywhere in the ledger would abort *every* rule-save. Fix: `continue`. Lesson: `contraPosting`→false belongs in the same bucket as the `!t.Cleared`/`Matches` filters (skip this item), not with "the operation is broken" (`return`).

**End-to-end verified (scratch DB):** imported `testdata/seb.csv` (4 ICA rows, all → `Equity:Uncategorized`, `cleared=0`), reviewed **one** ICA row → `Expenses:Food:Groceries`, trimmed pattern to `ICA`, saved. The **other three** March ICA rows flipped to Groceries instantly (`cleared=0`), every non-ICA row untouched on Equity. Confirmed via `sqlite3`. Status line: `Saved rule: "ICA" — categorized 3 in queue`.

**Review-state indicator — IN PROGRESS, UNCOMMITTED (see handoff):** noticed cleared vs uncleared is invisible in the table. Root cause: `transactionStyleFuncFromIdx` keys its review styling on `v.Category == ""`, a **dead branch** — `projectTransaction` always sets `Category` to a contra account *path*, never `""`. Drift from the Session-13 intent (`!e.Cleared`). The `cleared` flag has zero table representation. Chosen design (via AskUserQuestion): a **status marker column** (`*` cleared / `!` review) **+ fade the whole uncleared row + a legend** in the help footer.

**Dead-config discovery:** `TxnColumn.Width` is **never applied** in the render path — only `.Title` and `.Align` are read. The table auto-sizes columns to content, then `lt.Width(t.width)` stretches them to fill the terminal, so the 1-char marker column absorbs stretch slack → visible gap between glyph and date. `Align` is honored, `Width` is a lie the struct tells. Fix options (unresolved): right-align the marker (uses the honored `Align`), or prefix the glyph onto the date cell (drop the column), or make `Width` real in the StyleFunc (a layout refactor — not mid-stream).

**Concepts practiced:** value-receiver domain method + behavior-preserving refactor proven by existing tests, bulk atomic transaction + rollback-as-assertion test, skip-vs-abort control flow (a filter is not an error), fill-blanks-never-overwrite scope guard, dead-configuration-field smell (`Width` vs `Align` asymmetry).

**Status:** re-point committed at `bd6c025` "Automatically repoint uncleared postings with new rules"; `go test ./...` + `go vet` green there. Review-indicator WIP is **uncommitted** (`model.go`, `txntable.go`, `views.go`).

---

### Session 18 — review-state indicator finished + layout truthfulness (Phase 15 polish)
**Date:** 2026-09-01
**Covered:** Closed out the Session-17 WIP indicator (marker column, faded rows, coloured glyphs) and fixed the four layout lies it exposed. Also recorded the Phase-26 ANSI-palette alternative.

**Method worth reusing — the offline render harness.** Copied `go.mod`/`go.sum` into a scratch dir under `/tmp` (renamed the module), reimplemented `TxnTable.View`'s essentials, and rendered labelled variants at fixed widths with a column ruler. No PTY, no DB, no TUI restart — layout questions answered in seconds and *proven* rather than argued. Also read the vendored `lipgloss/v2@v2.0.2/table` source directly instead of guessing at behaviour. `tmux capture-pane -p [-e]` on the running app closed the loop: `-e` keeps the SGR sequences, so the rendered colours can be asserted (`\e[38;2;40;105;131m` = `#286983` = Pine) without eyeballing a screenshot.

**Layout findings (all four were dead or lying config):**
1. **`TxnColumn.Width` was never applied** — lipgloss *does* support fixed columns, but only via `style.GetWidth()` read off the `StyleFunc` return (`resizing.go:93`); every expand/shrink loop then skips columns where `width == fixedWidth`. Fix: apply the width in both branches of `TxnTable.View`'s style closure. Only the marker column declares one — making *all* columns fixed removes the slack absorber and `Category` wraps instead of the table filling the terminal.
2. **The gap before the date** was the expansion loop (`resizing.go:213-225`) repeatedly growing the *shortest* non-fixed column — the 1-char marker column was always the target. Fixed width solves it; `Align: Right` only hides it.
3. **`Width` includes padding.** `{Width: 1}` against `tableCell`'s frame of 4 truncated the glyph to nothing — silent, no error (the markers vanished for one round-trip because of exactly this). Settled on `Padding(0, 1)` for both `tableCell` and `tableHeader` (was 2 each), i.e. frame 2 everywhere, `Width: 3` for the marker.
4. **Header padding inflated every column's minimum**, because `column.xPadding` is a max over *all* rows including the header (`resizing.go:92`) — 6 columns × 2 extra chars was the difference between fitting and wrapping at 94 cols.

**`Wrap(false)` — the height contract.** `table.wrap` defaults true (`table.go:90`), so a long payee silently made a row 2+ lines tall: measured **13 lines for 5 data rows**, against a budget that assumes 1 line per row. It broke the layout from *inside* the component, which is why it's hard-coded rather than exposed as an option. `Wrap(false)` truncates with `…` (`table.go:555`) and degrades cleanly down to 60 cols.

**Border.** `WithTxnBorder` had never been called, so the border was the zero `lipgloss.Border{}` — top border and header separator still rendered as **blank lines** (measured: chrome = rows + 3 either way, rows + 4 with a real border). Switched to `RoundedBorder()`, and replaced the duplicated `3` (one in `model.go`'s budget, one in `View`) with `TxnTable.chromeHeight()`, guarded on `t.border.Bottom != ""` so it stays correct for both configurations.

**styleFunc (the point of the exercise):** `dim := !e.Cleared && !selected` fades uncleared rows, `case colStatus` colours `*` Pine / `!` Gold+bold and *deliberately ignores* `dim` so the marker survives a faded row. Precedence rule chosen: **background encodes selection, foreground encodes semantics** — every `return base.Foreground(...)` inherits the cursor's `HighlightLow` background, and the cursor row is never dimmed (Muted-on-HighlightLow is the worst contrast in the table for the one row you're about to act on). New token `styles.statusColor(cleared bool) color.Color` alongside `amountColor`. Removed two dead branches: `v.Category == ""` (never true — `projectTransaction` always sets a path; the Session-13 intent was `!e.Cleared`) and a stray `.Align(lipgloss.Right)` that `TxnTable.View` overwrites on every frame.

**Status line.** `lipgloss` `Width(n)` is a *minimum* that wraps; `MaxWidth(n)` clips per line *after* wrapping (so `Width+MaxWidth` alone still yields two clipped lines). Fixed with `clampToWidth(s, style, w)` = `style.Width(w).Render(ansi.Truncate(s, w-style.GetHorizontalFrameSize(), "…"))` — ANSI-aware truncation, the same function the lipgloss table uses; promoted `github.com/charmbracelet/x/ansi` from indirect to direct. `ansi.Truncate` with a negative length returns `""` rather than panicking (probed). `statusRight` gained `Align(lipgloss.Right)` — alignment is inert without a width, which is the same reason the column `Align` only works because the table assigns widths.

**Legend: built, then deleted.** Rendered fine but had nowhere to live — 20 cells the help row doesn't have at 94 cols (it clipped), and the status line's middle third is reserved for messages. Decision: it belongs in the full help; deferred (see handoff (b)). `glyphCleared`/`glyphReview` consts kept and used by `buildRowsFromIdx`.

**`relayout()` — the session's real architectural fix.** Pressing `?` made the status line vanish: `helpH = 2` hard-coded "the help is one line", `FullHelp()` renders three, and nothing recomputed sizing outside `WindowSizeMsg`. Extracted all sizing into `func (m *Model) relayout()` with `helpH := lipgloss.Height(m.styles.help.Render(m.help.View(m.keys)))` — measured, so it covers both help states — guarded by `if !m.ready { return }` so it's safe to call from anywhere. Three callers: resize, help toggle, and `esc`. Axel extended `esc` to also collapse expanded help, so `esc` now uniformly means "back out of every transient view state" (search + status message + expanded help), and folded the duplicated search teardown into `func (m *Model) clearSearch()`.

**Go idiom noted:** pointer-receiver helpers (`relayout`, `clearSearch`) called from value-receiver `Update` methods — Go auto-takes the address of the addressable local `m`, so it reads as mutation while `Update` stays a pure function of the incoming model.

**Also recorded:** Phase 26 now leads with the **ANSI-palette alternative** — style from the 16 terminal palette slots and light/dark tracking, per-user themes and tmux come free, with no OSC 11 probe, no fallback chain and no terminal state to restore. Deferred, but currently the more elegant answer than the mode-2031 machinery.

**Concepts practiced:** reading a dependency's source to settle behaviour instead of guessing, offline render harness as a debugging tool, fixed-vs-flexible layout budgeting, style composition (colours compose onto a base, whole styles replace it), a full reset (`\e[0m`) killing an outer background — styled strings compose by construction not concatenation, ANSI-aware truncation vs byte/rune slicing, derived state recomputed by every input that affects it, dead-config and dead-branch smells.

**Exercise done the same evening — the third review state.** The `!` glyph was lying on transfers: a transaction whose legs are all Asset/Liability has no contra posting, so `c` silently did nothing on a row painted attention-Gold. Replaced the boolean with `reviewState` (`stateCleared`/`stateReview`/`stateTransfer`, `iota`) + `reviewStateOf(t, accts)` + a `glyph()` method in `transactionview.go`, and `statusColor` now switches on the state (Gold = act, Pine = done, Muted = nothing to do, bold reserved for the one actionable state). Transfers are **not** dimmed: the review queue *is* `Equity:Uncategorized`'s balance and a transfer was never in it. `dim` became `state == stateReview && !selected`. Payoff: `updateList`'s Review guard collapsed from two checks into `reviewStateOf(*t, m.accountsByID) != stateReview`, so what the marker says and what `c` does are now one expression. Go gotcha met on the way: in a const block a type does *not* carry to the next line when an expression is given (`glyphCleared glyph = "*"` then `glyphReview = "!"` leaves the second untyped) — repetition only happens when the expression is omitted, which is why the `iota` idiom works.

**Transfer fixture:** `testdata/seb.csv` gained `ÖVERFÖRING Sparkonto` / `ÖVERFÖRING Investering` rows (field 4 is ignored by `parseRow`). Note that CSV rows alone can't produce a transfer — `PlaceholderTransactions` parks every contra leg on Equity, so both import as `!`. A transfer appears when the contra leg lands on an Asset/Liability account, i.e. review *one* row to `Assets:Bank:Sparkonto` and save the rule `ÖVERFÖRING`; the Session-17 sweep re-points the other row while leaving `cleared = 0` → `stateTransfer`. **Not yet run end-to-end** (see handoff).

**Status:** `go build ./...`, `go vet ./...`, `go test ./...`, `gofmt -l` all clean. No `ci` script in this repo. Verified in the running app at 94×28.

---

## 🔧 HANDOFF — pick up here

**⚠️ UNCOMMITTED but green:** the `reviewState` work (`internal/tui/{transactionview,styles,model}.go`) and the two `ÖVERFÖRING` rows in `testdata/seb.csv`. `go build`/`go vet`/`go test ./...`/`gofmt -l` all clean; **not yet exercised in the running app**. Commit or stash before switching machines. Everything before it is committed at `b296266`.

**Step 1 — trivial:** restore the truncated comment in `transactionview.go`: `Account string // the asset/liability ("from") account path`.

**Step 2 — prefill the review input (decided, snippets agreed, not yet typed).** Two edits, one idea: *the review screen should show what the transaction currently claims, and only ask to remember when you changed its mind.*
- `updateList`'s Review case: invert the guard to `if t == nil || reviewStateOf(*t, m.accountsByID) == stateTransfer { return m, nil }` — "cleared" was never the reason to refuse; a missing contra leg is. This makes **re-review of cleared rows** possible, which the store already supports (`ReviewTransaction` repoints whatever posting ID it's given and sets `cleared = 1`; it makes no placeholder assumption).
- Seed `m.reviewInput` with the current contra path (skipping `placeholderAccount`, then `CursorEnd()`) instead of `SetValue("")`. This is what makes a **rule-fired row a one-keystroke confirm** — its contra leg already *is* the rule's account.
- In `updateReview`'s Enter branch, capture `wasPath := m.accountsByID[contra.AccountID].Path` *before* `ReviewTransaction`, and extend the rule-prompt skip to `payee == "" || path == placeholderAccount || path == wasPath`. Derived, not stored — a `Model` field would need resetting on every leaving path (the Session 15–16 bug family). Flows: first review → prompt; re-review with a change → prompt; confirm a rule's guess → straight back to the list.

**Step 3 — verify the transfer glyph end-to-end:**
```sh
rm -f /tmp/ft-transfer.db
go run ./cmd/fintracker -db /tmp/ft-transfer.db "Assets:Bank:SEB=testdata/seb.csv"
# c on ÖVERFÖRING Sparkonto → Assets:Bank:Sparkonto → rule pattern "ÖVERFÖRING"
sqlite3 /tmp/ft-transfer.db "select t.id, t.cleared, a.path, p.amount from transactions t
  join postings p on p.transaction_id = t.id join accounts a on a.id = p.account_id
  where t.raw_payee like 'ÖVERFÖRING%' order by t.id, p.id;"
```
Expect: reviewed row `cleared=1` → `*` Pine; the swept row `cleared=0` with a leg on `Assets:Bank:Sparkonto` → `→` Muted, undimmed.

**Two design decisions waiting (both surfaced by the third state):**
1. **`cleared` is overloaded** — it means both "a human confirmed this" and "this has a real contra account". A rule-repointed transfer satisfies the second without the first, and since `c` correctly refuses transfers, *nothing in the UI can ever clear it*: it sits as `→` forever and the queue count lies. Options: (i) let `c` on a transfer confirm-without-recategorizing, (ii) have the re-point sweep clear rows it moves off the placeholder, (iii) treat "no contra leg" as inherently cleared. Decide before the queue count matters.
2. **Rule pattern collisions on re-review** — `EnsurePayeeRule` is keyed on `(pattern, default_account_id)`, so correcting `ICA → Groceries` to `ICA → Restaurants` creates a *second* `ICA` rule, both `Priority: 0`; `matchRule` takes the first pattern match in ascending-priority order, so the old wrong rule can keep winning while the UI reports success. Options: upsert on pattern alone (one pattern → one account — matches how you'd describe the feature out loud, but changes `EnsurePayeeRule`'s contract and tests), detect-and-offer-"update", or skip the prompt when a rule for that pattern exists. Step 2's `path == wasPath` skip disarms only the *confirm* path; the genuine-correction path still needs this.

**Deferred polish (note, don't block):**
- (a) ~~status line never clears / wraps on a narrow window~~ — **DONE** Session 18.
- (b) **status-marker legend** — has no home yet. *Not* the status line's middle third (reserved for messages) and *not* the help row (20 cells it doesn't have at 94 cols — it clipped). Home: the **full help** (`?`), affordable now that `relayout` measures help height. Either append a legend line to the rendered help in `views.go` when `m.help.ShowAll`, or express it as `key.Binding`s in `FullHelp()` — note a binding built without `WithKeys` is `Enabled() == false` and the help component skips it. Three states to explain now, not two: `glyphCleared`/`glyphReview`/`glyphTransfer` consts live in `views.go`, and `styles.statusColor(reviewState)` gives the matching colours.
- (c) summary view appends `netWorth` as the "Total" row of the *Category* table — semantically wrong (different account types); relabel/reposition.
- (d) proper **state machine** for screens/input-modes — `search` (a bool on `listScreen`) vs real screens still forces a `reviewScreen`/`rulePromptScreen` exclusion list in the global Quit handler; `ctrl-c` won't hard-quit from review.
- (e) unify `contraPosting` with `projectTransaction`'s asset/contra detection (minor dup).
- (f) empty `internal/finance/testdata/` dir can be `rmdir`'d.
- (g) `TxnColumn.Width` is honest now, but only the marker column sets one. Rule: **a fixed width must cover the cell's own frame** — `Width: 3` = 1 glyph + `tableCell`'s `Padding(0, 1)`. Declare it smaller than the padding and lipgloss truncates the content to nothing, silently.
- (h) the rounded table border is new and provisional — `WithTxnBorder` is one line to drop and `chromeHeight()` adapts on its own.

**Phase 15 status:** MVP done (`c` on an uncleared row → type a contra account with autocomplete → enter → atomically repointed + cleared + reloaded; `Equity:Uncategorized`'s balance *is* the review-queue length). ✅ (c) payee_rule written on review + re-point-on-save. Not built: (a) memo / tags / normalized-payee editing, (b) **split editor** for >2 postings — `contraPosting` returns only the first and the picker can't split, so `ReviewTransaction` needs a wholesale posting-rewrite variant (delete + reinsert in one tx); this is also the blocker for editing a transfer's legs, (d) visible filtered pick-list instead of inline autocomplete.

**Still-uncovered Go concepts (opportunistic):** `errors.As` + a custom error *struct* type (an FK violation on a bad `account_id` is the natural trigger — `errors.Is` already covered via `ErrDuplicateTransaction`/`sql.ErrNoRows`); generics; benchmarking; `iter` / range-over-func; `sync` primitives; `net/http`; JSON.
