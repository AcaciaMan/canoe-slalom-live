# Prompt 04 — Leaderboard Page

## Context

You are working on **canoe-slalom-live**, a Go web app for canoe/water slalom competitions. Stack: Go `net/http`, `html/template`, SQLite.

Read `PLAN.md` for full architecture. Previous steps built:
- **Prompt 01**: Database + domain structs + store layer + seed data
- **Prompt 02**: HTTP server, public event page with start list, athlete profiles, CSS
- **Prompt 03**: Judge interface to record runs with time + penalties

The app runs on `localhost:8080`. The judge can record runs for athletes. Runs are stored with `raw_time_ms`, `penalty_touches`, `penalty_misses`, `penalty_seconds`, `total_time_ms`, and `status`.

This prompt implements **Phase 1, Step 4**: the leaderboard that ranks athletes by best total time.

## Goal

Visit `http://localhost:8080/events/demo-slalom-2026/leaderboard` to see athletes ranked by their best run, grouped by category. The leaderboard updates every time a new run is recorded (on page refresh). It also integrates into the event page as the second tab.

## What to build

### 1. Store query — `store/runs.go`: `GetLeaderboard`

This is the most important query in the app. If it doesn't already exist or is incomplete, implement it properly.

**Function signature:**
```go
func GetLeaderboard(db *sql.DB, categoryID int) ([]LeaderboardRow, error)
```

**`LeaderboardRow` struct:**
```go
type LeaderboardRow struct {
    Rank           int
    BibNumber      int
    AthleteID      int
    AthleteName    string
    AthleteNation  string
    Run1           *RunResult  // nil if no Run 1 recorded
    Run2           *RunResult  // nil if no Run 2 recorded
    BestTotalTimeMs int        // best of Run1/Run2 total_time_ms; 0 if no runs
    HasRuns        bool
}

type RunResult struct {
    RawTimeMs      int
    PenaltyTouches int
    PenaltyMisses  int
    PenaltySeconds int
    TotalTimeMs    int
    Status         string
}
```

**Query logic:**
- Join `entries` with `athletes` to get athlete info.
- LEFT JOIN `runs` twice (once for `run_number = 1`, once for `run_number = 2`) to get both runs.
- Filter by `entries.category_id = ?`.
- Order by `start_position` for athletes without runs (they go last).
- For athletes with runs, rank by `BestTotalTimeMs` ascending.
- Handle `dns`, `dnf`, `dsq` statuses: these sort after all valid finishes.

**Suggested SQL approach:**
```sql
SELECT 
    e.bib_number,
    e.athlete_id,
    a.name,
    a.nation,
    r1.raw_time_ms, r1.penalty_touches, r1.penalty_misses, r1.penalty_seconds, r1.total_time_ms, r1.status,
    r2.raw_time_ms, r2.penalty_touches, r2.penalty_misses, r2.penalty_seconds, r2.total_time_ms, r2.status
FROM entries e
JOIN athletes a ON a.id = e.athlete_id
LEFT JOIN runs r1 ON r1.entry_id = e.id AND r1.run_number = 1
LEFT JOIN runs r2 ON r2.entry_id = e.id AND r2.run_number = 2
WHERE e.category_id = ?
ORDER BY 
    CASE WHEN COALESCE(r1.total_time_ms, 0) = 0 AND COALESCE(r2.total_time_ms, 0) = 0 THEN 1 ELSE 0 END,
    MIN(
        CASE WHEN r1.status = 'ok' THEN r1.total_time_ms ELSE 999999999 END,
        CASE WHEN r2.status = 'ok' THEN r2.total_time_ms ELSE 999999999 END
    ),
    e.start_position
```

After fetching rows, compute `BestTotalTimeMs` and `Rank` in Go code (simpler than doing complex ranking in SQL):
1. Iterate rows. For each, find the minimum `total_time_ms` from valid (status = `ok`) runs.
2. Separate into two lists: athletes with valid runs, athletes without.
3. Sort the "with runs" list by `BestTotalTimeMs` ascending.
4. Assign ranks: 1, 2, 3... for sorted athletes. Athletes with equal `BestTotalTimeMs` get the same rank.
5. Append "without runs" athletes at the end with Rank = 0 (displayed as "—").

**Add formatting helpers** to `RunResult` or as template functions:
- `FormatTime(ms int) string` — converts milliseconds to `MM:SS.xx` format. Examples: 94370 → `1:34.37`, 182050 → `3:02.05`, 59990 → `0:59.99`. Use this function from `domain/run.go` if it already exists, or add it as a shared utility.

### 2. Handler — add to `handler/public.go`

**LeaderboardPage handler** (`GET /events/{slug}/leaderboard`):
1. Extract slug, fetch event. 404 if not found.
2. Fetch categories for this event.
3. For each category, call `store.GetLeaderboard(d.DB, cat.ID)`.
4. Check for `?partial=1` query param. If set, render ONLY the leaderboard content (no layout wrapper) — this is for future AJAX refresh. If not set, render full page with layout.
5. Render with data:

```go
LeaderboardPageData struct {
    Event      domain.Event
    Categories []CategoryLeaderboard
    Title      string
}

CategoryLeaderboard struct {
    Category domain.Category
    Rows     []store.LeaderboardRow
}
```

### 3. Template — `templates/leaderboard.html`

Layout within `{{define "content"}}`:

**Event header** (same as event page — event name, date, location).

**Tab bar**: "Start List" (links to `/events/{slug}`), "Leaderboard" (active).

**Leaderboard content div** (`id="leaderboard-content"` — for future JS refresh):

For each category:
- **Category heading**: e.g., "K1M — Kayak Single Men"
- **Results table** with columns:

| Rank | Bib | Athlete | Nation | Run 1 | Run 2 | Best Time |
|------|-----|---------|--------|-------|-------|-----------|

- **Rank column**: Show `#1`, `#2`, `#3` etc. Show "—" for athletes with no runs.
- **Athlete column**: Name as a link to athlete profile page.
- **Nation column**: 3-letter code.
- **Run 1 / Run 2 columns**: Show as `{rawFormatted} + {penaltySec}s = {totalFormatted}` if the run exists. Example: `1:34.37 + 4s = 1:38.37`. If no penalties: just `1:34.37`. If run doesn't exist: `—`. Color the penalty text:
  - 0 penalties (clean run): show time in green.
  - Touches only: show penalty part in yellow/amber.
  - Any misses: show penalty part in red.
  - DSQ/DNF/DNS: show status text in red, no time.
- **Best Time column**: The best total time formatted as `MM:SS.xx` in bold. If no valid runs: "—".

**Bottom note**: "Ranked by best single-run time. Penalties: gate touch = 2s, missed gate = 50s."

**If no runs exist for any category**: Show a message "No results yet — check back during the competition!" instead of an empty table.

### 4. Wire leaderboard into event page

Update `templates/event.html`:
- The "Leaderboard" tab in the tab bar should now link to `/events/{{.Event.Slug}}/leaderboard` (replace any placeholder).

### 5. Template — `templates/leaderboard_partial.html` (optional but recommended)

Create a separate partial that contains ONLY the leaderboard tables (no layout, no header). This is what gets returned when `?partial=1` is requested.

The handler checks `?partial=1`:
- If set: render `leaderboard_partial.html` directly (no layout wrapping).
- If not set: render `leaderboard.html` with `layout.html`.

This enables future auto-refresh via JS without duplicating template code. For now, the partial isn't used by any JS yet — it's just wired up ready.

Register the partial template separately in `main.go`:
```go
"leaderboard_partial": template.Must(template.ParseFiles("templates/leaderboard_partial.html")),
```

### 6. Route registration in `main.go`

Add:
```
GET /events/{slug}/leaderboard → deps.LeaderboardPage
```

Add template to the template map:
```go
"leaderboard": template.Must(template.ParseFiles("templates/layout.html", "templates/leaderboard.html")),
```

### 7. CSS additions — `static/style.css`

Append leaderboard-specific styles:

- `.leaderboard-table` — clean table styling, slightly different from start list. Rank column narrow (40px). Best Time column bold.
- `.rank-1`, `.rank-2`, `.rank-3` — optional: gold/silver/bronze accent for top 3 rows (subtle left border or background tint).
- `.penalty-clean` — green text color (#16a34a).
- `.penalty-touch` — amber text color (#d97706).
- `.penalty-miss` — red text color (#dc2626).
- `.status-dnf`, `.status-dsq` — red text, italic.
- `.no-runs` — gray italic text for "—" cells.
- `.leaderboard-note` — small gray text for the bottom explanation.

### 8. Seed some runs for demo

Add to `db/seed.sql` (or create a separate `db/seed_runs.sql`): insert 8–12 runs for the seeded athletes so the leaderboard has data to display. Include variety:
- Some athletes with 2 runs (best of 2 displayed).
- Some athletes with only 1 run.
- Some clean runs (0 penalties).
- Some runs with touches (2–4 touches).
- One run with a missed gate (50s penalty — this athlete should rank poorly).
- Use realistic raw times for canoe slalom: 85–110 seconds range for K1M, 95–120 seconds range for C1W.

Example seed runs (adjust athlete/entry IDs based on your seed data):
- Entry 1, Run 1: raw 92.45s, 0 touches, 0 misses → total 92.45s
- Entry 1, Run 2: raw 89.71s, 2 touches, 0 misses → total 93.71s (Run 1 is better)
- Entry 2, Run 1: raw 95.33s, 1 touch, 0 misses → total 97.33s
- Entry 3, Run 1: raw 88.12s, 0 touches, 1 miss → total 138.12s (ouch)
- Entry 3, Run 2: raw 91.44s, 0 touches, 0 misses → total 91.44s (great recovery)
- etc.

Update `db/db.go` `Seed()` function to also execute the run seeds (can be in the same `seed.sql` file, just appended).

## Verification

1. `go build ./...` — no errors.
2. Re-seed: `go run main.go -seed` (or delete `data.db` first, then seed fresh).
3. Open `http://localhost:8080/events/demo-slalom-2026/leaderboard`.
4. See athletes ranked by best run time. Top athlete has the lowest total time.
5. Penalty colors visible: clean runs in green, touches in amber, misses in red.
6. Both Run 1 and Run 2 columns populated for athletes with 2 runs.
7. Athletes without runs show "—" and sort to the bottom.
8. Click "Start List" tab → goes back to event page.
9. Go to judge page → record a new run for an athlete → refresh leaderboard → new run appears and ranking updates.
10. Click athlete name in leaderboard → goes to athlete profile → shows their runs in the results table.
11. Open `/events/demo-slalom-2026/leaderboard?partial=1` → returns just the table HTML, no layout.

## Files to create/modify

```
store/runs.go                  (modify: implement/improve GetLeaderboard + LeaderboardRow)
handler/public.go              (modify: add LeaderboardPage handler)
templates/leaderboard.html     (create)
templates/leaderboard_partial.html (create)
templates/event.html           (modify: fix leaderboard tab link)
main.go                        (modify: add route + template)
static/style.css               (modify: add leaderboard styles)
db/seed.sql                    (modify: add seed runs)
```

## Important notes

- Time precision: always display hundredths of seconds (2 decimal places). `1:34.37`, not `1:34.370` or `1:34.4`.
- Milliseconds to formatted time: `ms / 60000` = minutes, `(ms % 60000) / 1000` = seconds, `(ms % 1000) / 10` = hundredths. Format: `%d:%02d.%02d`.
- Equal times = equal rank (standard sports ranking). If two athletes tie at rank 2, the next athlete is rank 4 (not 3).
- The leaderboard is the **most important page** of the app. If it looks good with realistic data and correct math, the demo is convincing.
