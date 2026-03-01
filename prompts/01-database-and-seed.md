# Prompt 01 — Database Layer, Domain Structs, and Seed Data

## Context

You are working on **canoe-slalom-live**, a small Go web app for canoe/water slalom competitions. The stack is Go (idiomatic `net/http`), server-rendered HTML templates, and SQLite (embedded). The Go module is already initialized (`canoe-slalom-live`, Go 1.25).

Read `PLAN.md` in the repo root for full architecture and data model details. This prompt implements **Phase 1, Step 1**.

## Goal

Set up the SQLite database layer, domain structs, store (data access) functions, and seed data so the app can start up, create its schema, and have realistic demo data ready to query.

## What to build

### 1. SQLite driver dependency

Use `github.com/mattn/go-sqlite3` (CGo-based SQLite driver). Run `go get github.com/mattn/go-sqlite3`. If CGo is problematic on the machine, fall back to `modernc.org/sqlite` (pure Go) — but try `mattn` first.

### 2. `db/migrations.sql`

A single SQL file with `CREATE TABLE IF NOT EXISTS` for all five tables. Follow the schema exactly as specified in PLAN.md Section 2:

- **`events`**: `id` INTEGER PRIMARY KEY, `slug` TEXT UNIQUE NOT NULL, `name` TEXT NOT NULL, `date` TEXT, `location` TEXT, `status` TEXT DEFAULT 'draft', `created_at` TEXT.
- **`categories`**: `id` INTEGER PRIMARY KEY, `event_id` INTEGER REFERENCES events(id), `code` TEXT NOT NULL, `name` TEXT NOT NULL, `sort_order` INTEGER DEFAULT 0, `num_runs` INTEGER DEFAULT 2.
- **`athletes`**: `id` INTEGER PRIMARY KEY, `name` TEXT NOT NULL, `club` TEXT, `nation` TEXT, `bio` TEXT, `photo_url` TEXT, `created_at` TEXT.
- **`entries`**: `id` INTEGER PRIMARY KEY, `event_id` INTEGER REFERENCES events(id), `category_id` INTEGER REFERENCES categories(id), `athlete_id` INTEGER REFERENCES athletes(id), `bib_number` INTEGER, `start_position` INTEGER. UNIQUE constraint on `(event_id, athlete_id)`.
- **`runs`**: `id` INTEGER PRIMARY KEY, `entry_id` INTEGER REFERENCES entries(id), `run_number` INTEGER, `raw_time_ms` INTEGER, `penalty_touches` INTEGER DEFAULT 0, `penalty_misses` INTEGER DEFAULT 0, `penalty_seconds` INTEGER DEFAULT 0, `total_time_ms` INTEGER, `status` TEXT DEFAULT 'ok', `judged_at` TEXT. UNIQUE constraint on `(entry_id, run_number)`.

Enable foreign keys with `PRAGMA foreign_keys = ON;` at the top.

### 3. `db/seed.sql`

Insert realistic canoe slalom seed data:

- **1 event**: "Demo Slalom 2026", slug `demo-slalom-2026`, date "2026-06-15", location "Troja Whitewater Course, Prague", status "active".
- **2 categories** for that event: K1M ("Kayak Single Men", sort_order 1, num_runs 2), C1W ("Canoe Single Women", sort_order 2, num_runs 2).
- **10 athletes** with realistic names, clubs, nations, and short bios. Mix of nations (CZE, GBR, FRA, GER, AUS, SVK, ESP, POL). Use real-sounding club names from the canoe slalom world (e.g., "USK Praha", "Lee Valley CC", "Pau Canoe-Kayak"). Each athlete needs a 1–2 sentence bio with a "commentator fact" (e.g., "Junior European champion 2024. Known for aggressive lines through upstream gates.").
- **10 entries**: 6 athletes in K1M, 4 athletes in C1W. Assign bib numbers (101–106 for K1M, 201–204 for C1W) and start positions sequentially.
- **No runs yet** — those will be added via the judge interface.

### 4. `db/db.go`

Package `db`. Exports:

- `func Open(dbPath string) (*sql.DB, error)` — opens SQLite file, enables foreign keys (`PRAGMA foreign_keys = ON`), reads and executes `db/migrations.sql` using `go:embed`. Returns the `*sql.DB`.
- `func Seed(database *sql.DB) error` — reads and executes `db/seed.sql` using `go:embed`. Should be idempotent (use `INSERT OR IGNORE` in seed.sql so re-running doesn't fail on unique constraints).

Use `//go:embed migrations.sql` and `//go:embed seed.sql` to embed the SQL files.

### 5. Domain structs — `domain/event.go`

```
type Event struct {
    ID        int
    Slug      string
    Name      string
    Date      string
    Location  string
    Status    string
    CreatedAt string
}

type Category struct {
    ID        int
    EventID   int
    Code      string
    Name      string
    SortOrder int
    NumRuns   int
}
```

### 6. Domain structs — `domain/athlete.go`

```
type Athlete struct {
    ID        int
    Name      string
    Club      string
    Nation    string
    Bio       string
    PhotoURL  string
    CreatedAt string
}

type Entry struct {
    ID            int
    EventID       int
    CategoryID    int
    AthleteID     int
    BibNumber     int
    StartPosition int
}
```

### 7. Domain structs — `domain/run.go`

```
type Run struct {
    ID              int
    EntryID         int
    RunNumber       int
    RawTimeMs       int
    PenaltyTouches  int
    PenaltyMisses   int
    PenaltySeconds  int
    TotalTimeMs     int
    Status          string
    JudgedAt        string
}
```

Add helper methods on Run:
- `func (r Run) RawTimeFormatted() string` — formats `RawTimeMs` as `MM:SS.xx` (e.g., 94370 → "01:34.37").
- `func (r Run) TotalTimeFormatted() string` — same format for `TotalTimeMs`.
- `func (r Run) PenaltyDisplay() string` — returns e.g. "2T + 1M = 52s" or "Clean" if zero penalties.

### 8. Store layer — `store/events.go`

Package `store`. Functions (all take `*sql.DB` as first arg):

- `func GetEventBySlug(db *sql.DB, slug string) (domain.Event, error)`
- `func ListCategories(db *sql.DB, eventID int) ([]domain.Category, error)` — ordered by `sort_order`.

### 9. Store layer — `store/athletes.go`

- `func GetAthlete(db *sql.DB, id int) (domain.Athlete, error)`
- `func ListEntriesByCategory(db *sql.DB, categoryID int) ([]EntryWithAthlete, error)` — returns entries joined with athlete data, ordered by `start_position`. Define `EntryWithAthlete` struct that embeds/combines Entry + Athlete fields needed for display (bib, name, club, nation).

### 10. Store layer — `store/runs.go`

- `func CreateRun(db *sql.DB, run domain.Run) (int64, error)` — inserts a run, returns the new ID.
- `func ListRunsByEntry(db *sql.DB, entryID int) ([]domain.Run, error)`
- `func GetLeaderboard(db *sql.DB, categoryID int) ([]LeaderboardRow, error)` — query that joins entries, athletes, and runs to produce ranked results. Each `LeaderboardRow` should have: Rank, BibNumber, AthleteName, AthleteNation, Run1 (nullable Run data), Run2 (nullable Run data), BestTotalTimeMs. Order by best `total_time_ms` ascending. Athletes with no runs sort last.

Define the `LeaderboardRow` struct in this file.

### 11. `main.go` (minimal for this step)

- Parse a `-seed` bool flag.
- Open the database with `db.Open("data.db")`.
- If `-seed` flag is set, call `db.Seed(database)`.
- Print "Database ready" and exit (HTTP server wiring comes in the next prompt).

Make sure `main.go` compiles and runs: `go run main.go -seed` should create `data.db` with all tables and seed data.

## Verification

After implementing, these should work:
1. `go build ./...` — no errors.
2. `go run main.go -seed` — creates `data.db`.
3. Open `data.db` with any SQLite tool and verify: 1 event, 2 categories, 10 athletes, 10 entries, 0 runs.

## Files to create

```
db/db.go
db/migrations.sql
db/seed.sql
domain/event.go
domain/athlete.go
domain/run.go
store/events.go
store/athletes.go
store/runs.go
main.go
```

## Important notes

- Add `data.db` to `.gitignore`.
- All times are stored in milliseconds as integers (not floats).
- Use `go:embed` for SQL files — don't read from disk at runtime.
- Keep functions simple: no interfaces, no generics, no constructor patterns. Plain functions and structs.
