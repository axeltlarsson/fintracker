# Accounting Model Roadmap

Implementing plain-text accounting concepts (double-entry bookkeeping) in fintracker.
This replaces the current one-to-one CSV row → DB row model with a rigorous journal-based model.

---

## Core concepts to implement

### Accounts

An account is a named bucket that tracks value over time. Every financial event moves value
between accounts — never into a void, never from nowhere.

Accounts have a **type** that determines their normal sign polarity:

| Type        | Tracks              | Normal balance          |
|-------------|---------------------|-------------------------|
| Assets      | Things you own      | Positive = you have it  |
| Liabilities | Things you owe      | Negative = you owe it   |
| Income      | Value flowing in    | Negative = you earned it |
| Expenses    | Value consumed      | Positive = you spent it |
| Equity      | Opening/net worth   | Residual                |

The invariant that must never break: `Assets + Liabilities + Equity + Income + Expenses = 0`
(when all are signed correctly). Every transaction preserves this.

Account paths are colon-delimited strings representing a hierarchy:

```
Assets:Checking:SEB
Assets:Checking:Handelsbanken
Assets:Receivable:Parents
Assets:Receivable:Erik
Liabilities:Creditcard:SEB
Liabilities:Creditcard:Amex
Expenses:Food:Groceries
Expenses:Food:Restaurants
Expenses:Travel:India2026
Expenses:Travel:London2026
Expenses:Subscriptions
Income:Salary:Axel
Income:Salary:Wife
Equity:Opening
```

Querying a subtree is a prefix match: `Expenses:Travel:%` gives all travel spend.
Querying a specific trip: `Expenses:Travel:India2026` gives just that trip.

**Schema:**

```sql
CREATE TABLE accounts (
    id         INTEGER PRIMARY KEY,
    path       TEXT NOT NULL UNIQUE,       -- full colon-delimited path
    type       TEXT NOT NULL,              -- Assets | Liabilities | Income | Expenses | Equity
    currency   TEXT NOT NULL DEFAULT 'SEK',
    opened_at  DATE,
    closed_at  DATE
);
```

**Go model:**

```go
type AccountType string

const (
    Assets      AccountType = "Assets"
    Liabilities AccountType = "Liabilities"
    Income      AccountType = "Income"
    Expenses    AccountType = "Expenses"
    Equity      AccountType = "Equity"
)

type Account struct {
    ID       int64
    Path     string      // e.g. "Expenses:Travel:India2026"
    Type     AccountType
    Currency string
    OpenedAt *time.Time
    ClosedAt *time.Time
}

func (a Account) Name() string {
    parts := strings.Split(a.Path, ":")
    return parts[len(parts)-1]
}

func (a Account) Parent() string {
    parts := strings.Split(a.Path, ":")
    if len(parts) < 2 {
        return ""
    }
    return strings.Join(parts[:len(parts)-1], ":")
}

func (a Account) Depth() int {
    return len(strings.Split(a.Path, ":")) - 1
}
```

---

### Postings

A posting is a single line in a transaction: one account affected by one amount.
It is the atomic unit of accounting. Everything else — balances, reports — is just
aggregating postings per account.

**Schema:**

```sql
CREATE TABLE postings (
    id             INTEGER PRIMARY KEY,
    transaction_id INTEGER NOT NULL REFERENCES transactions(id) ON DELETE CASCADE,
    account_id     INTEGER NOT NULL REFERENCES accounts(id),
    amount         INTEGER NOT NULL,   -- minor units (öre). NEVER float.
    currency       TEXT NOT NULL DEFAULT 'SEK'
);
```

**Go model:**

```go
type Posting struct {
    ID            int64
    TransactionID int64
    AccountID     int64
    Account       *Account  // populated on read
    Amount        int64     // öre — positive or negative
    Currency      string
}
```

**On signed amounts vs debit/credit:**

Plain-text accounting replaces the debit/credit terminology with signed amounts.
The rule is simple:

- If the account balance should go **up**: use a **positive** amount
- If the account balance should go **down**: use a **negative** amount

Examples:
- Spending 500 SEK on food: `Expenses:Food:Restaurants +500` (expense goes up)
- Charging it to credit card: `Liabilities:Creditcard:SEB -500` (liability goes more negative = more debt)
- Receiving salary: `Assets:Checking:SEB +25000`, `Income:Salary:Axel -25000`

Never use `REAL` or `DECIMAL` for money in SQLite. Store amounts as `INTEGER` in öre (minor units).
100 SEK = 10000 öre. Format for display: `amount / 100`.

---

### Transactions

A transaction is a dated, described group of postings that must balance to zero.
The sum of all posting amounts across all currencies must equal zero.

**Schema:**

```sql
CREATE TABLE transactions (
    id          INTEGER PRIMARY KEY,
    date        DATE NOT NULL,
    payee       TEXT,                   -- normalized display name, e.g. "ICA Maxi"
    raw_payee   TEXT,                   -- from bank export, e.g. "ICA MAXI HUSIE 1247"
    memo        TEXT,                   -- human annotation, written during review
    cleared     INTEGER NOT NULL DEFAULT 0,  -- 0 = pending/unreviewed, 1 = cleared
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
    imported_at DATETIME                -- when the CSV row was imported
);
```

**Go model:**

```go
type Transaction struct {
    ID        int64
    Date      time.Time
    Payee     string
    RawPayee  string
    Memo      string
    Cleared   bool
    Postings  []Posting
    Tags      []string
}

// Validate checks the double-entry invariant.
// Every valid transaction must sum to zero per currency.
func (t Transaction) Validate() error {
    sums := make(map[string]int64)
    for _, p := range t.Postings {
        sums[p.Currency] += p.Amount
    }
    for currency, sum := range sums {
        if sum != 0 {
            return fmt.Errorf("transaction does not balance: %s sum is %d öre", currency, sum)
        }
    }
    if len(t.Postings) < 2 {
        return fmt.Errorf("transaction must have at least 2 postings")
    }
    return nil
}
```

**Example — simple grocery purchase:**

```
2026-03-10 * ICA Maxi
    Expenses:Food:Groceries    +89000   ; 890 SEK
    Liabilities:Creditcard:SEB -89000
    sum = 0 ✓
```

**Example — split transaction (hotel, parents' share as receivable):**

```
2026-03-07 * Radisson Blu Delhi    ; Hotel Delhi, 2 rooms
    Assets:Receivable:Parents      +370000   ; 3 700 SEK — parents owe us
    Expenses:Travel:India2026      +370000   ; our half
    Liabilities:Creditcard:SEB     -740000   ; total charge
    sum = 0 ✓
```

**Example — settlement (Swish from parents):**

```
2026-03-09 * Swish from parents
    Assets:Checking:SEB            +370000
    Assets:Receivable:Parents      -370000   ; receivable closes to zero
    sum = 0 ✓
```

---

### Tags

Tags provide orthogonal grouping that cuts across the account hierarchy.
A trip to India may have postings in `Expenses:Travel:India2026`, `Expenses:Food:Restaurants`,
and `Expenses:Health` — all of which belong together conceptually under a tag.

Tags are many-to-many on transactions (simpler) or postings (more precise).
Recommendation: support both, default to transaction-level tagging.

**Schema:**

```sql
CREATE TABLE tags (
    id   INTEGER PRIMARY KEY,
    name TEXT NOT NULL UNIQUE
);

CREATE TABLE transaction_tags (
    transaction_id INTEGER REFERENCES transactions(id) ON DELETE CASCADE,
    tag_id         INTEGER REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (transaction_id, tag_id)
);

-- Optional: posting-level tags for fine-grained split tagging
CREATE TABLE posting_tags (
    posting_id INTEGER REFERENCES postings(id) ON DELETE CASCADE,
    tag_id     INTEGER REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (posting_id, tag_id)
);
```

**Usage examples:**

- `india2026` — everything spent on the India trip (flights, hotel, food, insurance)
- `flights` — all flight purchases across all trips
- `reimbursable` — transactions where someone owes you money back
- `household` vs `personal` — shared vs individual spend

Querying "total India trip cost" = sum all postings on transactions tagged `india2026`
across all expense accounts. This is more powerful than account hierarchy alone because
it crosses account boundaries.

---

### Memos and payee normalization

Two distinct concepts, both on the transaction:

**Memo** — your human annotation written during the review session. Free text.
Answers "why / context I will forget." Stored in `transactions.memo`.

Examples:
- "Paid for Erik's lunch, he'll Swish me"
- "Annual subscription, paid in March"
- "Parents' hotel share — awaiting Swish"

**Payee normalization** — mapping raw bank strings to clean display names.
Solved at import time via a rules table. Write the rule once, never think about
that payee again.

**Schema:**

```sql
CREATE TABLE payee_rules (
    id                 INTEGER PRIMARY KEY,
    pattern            TEXT NOT NULL,          -- regex matched against raw_payee
    normalized_payee   TEXT NOT NULL,          -- display name to use
    default_account_id INTEGER REFERENCES accounts(id),  -- suggested account for auto-fill
    priority           INTEGER NOT NULL DEFAULT 0
);
```

**Example rules:**

| pattern                  | normalized_payee | default_account             |
|--------------------------|------------------|-----------------------------|
| `ICA MAXI.*`             | ICA Maxi         | Expenses:Food:Groceries     |
| `SPOTIFY AB`             | Spotify          | Expenses:Subscriptions      |
| `SAS SCANDINAVIAN.*`     | SAS              | Expenses:Travel             |
| `SWISH \+46.*`           | Swish            | (manual — varies)           |

At import time: apply rules in priority order, set `payee` and optionally pre-fill
the account on the generated postings. Unmatched transactions go into the review queue.

---

### Balance assertions

A balance assertion states that an account's computed balance on a given date
must equal a specific value. This is the reconciliation mechanism — how the journal
stays honest against real bank statements over time.

**Schema:**

```sql
CREATE TABLE balance_assertions (
    id         INTEGER PRIMARY KEY,
    account_id INTEGER NOT NULL REFERENCES accounts(id),
    date       DATE NOT NULL,
    expected   INTEGER NOT NULL,    -- expected balance in minor units
    currency   TEXT NOT NULL DEFAULT 'SEK',
    UNIQUE (account_id, date)
);
```

**Go validation:**

```go
func (l *Ledger) CheckBalanceAssertions() []AssertionError {
    var errs []AssertionError
    for _, assertion := range l.Assertions {
        actual := l.BalanceAt(assertion.AccountID, assertion.Date)
        if actual != assertion.Expected {
            errs = append(errs, AssertionError{
                Assertion: assertion,
                Actual:    actual,
                Diff:      actual - assertion.Expected,
            })
        }
    }
    return errs
}
```

**Workflow:** After your monthly session, add a balance assertion for each account
you can verify against a bank statement or credit card bill. If a future import
causes a mismatch, `fintracker verify` will tell you exactly which account is off
and by how much.

---

## Import pipeline

The recommended flow for `fintracker import seb.csv`:

```
1. Parse CSV rows → raw transaction structs
2. Deduplicate against existing transactions (date + raw_payee + amount)
3. Apply payee_rules → set normalized payee + suggested account
4. Generate skeleton transactions with two postings each:
       [suggested account OR unknown]  +amount
       [source liability/asset account] -amount
5. Write to DB with cleared=false
6. Present uncleared transactions in TUI review queue
```

During TUI review:
- Confirm or change the suggested account
- Add tags
- Write a memo if needed
- For splits: edit postings, add more postings, verify sum=0
- Mark as cleared on confirm

---

## Journal export

After a review session, export cleared transactions to hledger-compatible journal format
for archival and git history.

**Format:**

```ledger
2026-03-05 * SAS
    Expenses:Travel:India2026    4200 SEK  ; flights, india2026
    Liabilities:Creditcard:SEB

2026-03-07 * Radisson Blu Delhi  ; Hotel Delhi, 2 rooms — parents + us
    Assets:Receivable:Parents    3700 SEK  ; hotel
    Expenses:Travel:India2026    3700 SEK  ; hotel
    Liabilities:Creditcard:SEB  -7400 SEK
```

Notes:
- The last posting's amount can be omitted (hledger infers it to balance)
- Tags go in semicolon comments on the posting line
- `*` = cleared, `!` = pending
- Amounts in display units (SEK), not öre

**Suggested workflow after monthly session:**

```bash
fintracker export --month 2026-03 > ~/finance/journal/2026-03.journal
cd ~/finance && git add journal/2026-03.journal && git commit -m "march 2026"
```

---

## Implementation order

1. **Schema migration** — accounts, transactions, postings, tags, balance_assertions, payee_rules
2. **Account management** — create/list/search accounts in TUI, account path autocomplete
3. **Transaction + posting model** — replace current flat row model, implement `Validate()`
4. **Import pipeline** — CSV → raw transactions → payee rule matching → review queue
5. **Review TUI** — account picker, tag toggler, memo input, split mode, confirm/clear
6. **Balance assertions** — `fintracker verify` command
7. **Journal export** — `fintracker export` command + hledger format writer
8. **Payee rules** — rule editor in TUI, auto-apply on import
9. **Reporting** — balance by account, spending by tag, trip totals, net worth

---

## Key invariants to enforce in Go (not SQL)

- Every `Transaction` must have `len(Postings) >= 2`
- Sum of `Posting.Amount` across all postings must equal `0` per currency
- An account must exist (be opened) before a posting references it
- A closed account must not receive new postings after its `closed_at` date
- `Posting.Amount` is always stored in minor units (öre) as `int64` — never `float64`
