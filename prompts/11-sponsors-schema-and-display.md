# Prompt 11 — Sponsors Schema and Display

## Context

You are working on **canoe-slalom-live**, a Go web app for canoe/water slalom competitions. Stack: Go `net/http` (Go 1.22+ with pattern routing), `html/template`, SQLite via `github.com/mattn/go-sqlite3`.

Read `PLAN.md` for full architecture. Phases 1–2 are complete. The app has:

**Current codebase structure:**
```
main.go                          — entry point, template parsing, route registration, HTTP server
db/db.go                         — Open() and Seed() with go:embed for migrations.sql and seed.sql
db/migrations.sql                — CREATE TABLE for events, categories, athletes, entries, runs
db/seed.sql                      — demo data (1 event, 2 categories, 10 athletes, entries, runs)
domain/event.go                  — Event, Category structs
domain/athlete.go                — Athlete, Entry structs
domain/run.go                    — Run struct with RawTimeFormatted(), TotalTimeFormatted(), PenaltyDisplay(), FormatTime()
store/events.go                  — GetEventBySlug(), ListCategories()
store/athletes.go                — GetAthlete(), ListEntriesByCategory(), GetEntryByEventAndAthlete(), EntryWithAthlete struct
store/runs.go                    — CreateRun(), ListRunsByEntry(), GetEntryByID(), GetRunByID(), UpdateRun(), DeleteRun(), GetLeaderboard(), ListRecentRuns()
handler/public.go                — Deps struct {DB, Tmpls, AdminToken, Sessions}, EventPage, AthletePage, LeaderboardPage
handler/judge.go                 — JudgePage, SubmitRun, EditRunPage, UpdateRunHandler, DeleteRunHandler
handler/auth.go                  — RequireAuth middleware, SessionStore, cookie-based sessions
handler/helpers.go               — renderError() helper
handler/logging.go               — LoggingMiddleware, SecurityHeaders
templates/layout.html            — shared shell with nav + footer
templates/event.html             — public event page with start list
templates/athlete.html           — athlete profile with bio, runs table
templates/leaderboard.html       — leaderboard with auto-refresh bar
templates/leaderboard_partial.html — leaderboard tables partial for AJAX refresh (badges, sparklines, time-behind)
templates/judge_run.html         — judge form with confirmation step
templates/judge_edit_run.html    — edit run form
templates/error.html             — styled error page
static/style.css                 — full stylesheet (~740 lines)
static/app.js                    — auto-refresh leaderboard (10s, pause/resume, change highlight)
```

**Current routes in main.go:**
```go
GET  /{$}                                          → redirect to /events/demo-slalom-2026
GET  /events/{slug}                                → deps.EventPage
GET  /events/{slug}/leaderboard                    → deps.LeaderboardPage
GET  /events/{slug}/athletes/{id}                  → deps.AthletePage
GET  /judge/events/{slug}                          → deps.RequireAuth(deps.JudgePage)
POST /judge/events/{slug}/runs                     → deps.RequireAuth(deps.SubmitRun)
GET  /judge/events/{slug}/runs/{id}/edit           → deps.RequireAuth(deps.EditRunPage)
POST /judge/events/{slug}/runs/{id}                → deps.RequireAuth(deps.UpdateRunHandler)
POST /judge/events/{slug}/runs/{id}/delete         → deps.RequireAuth(deps.DeleteRunHandler)
GET  /static/                                      → file server
```

**Current footer in `templates/layout.html`:**
```html
<footer class="site-footer">
    <div class="container">
        Canoe Slalom Live — Grassroots timing made simple
    </div>
</footer>
```

## Goal

Add a `sponsors` table, seed demo sponsor data, create store/domain layer, and display sponsor logos on the event page footer, leaderboard footer, and in the shared site footer for the main sponsor "powered by" line.

## What to build

### 1. Domain struct — `domain/sponsor.go`

Create a new file `domain/sponsor.go`:

```go
package domain

type Sponsor struct {
    ID         int
    EventID    int
    Name       string
    LogoURL    string
    WebsiteURL string
    Tier       string // "main", "partner", "supporter"
    SortOrder  int
}
```

### 2. Database migration — `db/migrations.sql`

Add a new `CREATE TABLE IF NOT EXISTS` at the end of `migrations.sql`:

```sql
CREATE TABLE IF NOT EXISTS sponsors (
    id          INTEGER PRIMARY KEY,
    event_id    INTEGER REFERENCES events(id),
    name        TEXT NOT NULL,
    logo_url    TEXT NOT NULL,
    website_url TEXT,
    tier        TEXT NOT NULL DEFAULT 'supporter',
    sort_order  INTEGER DEFAULT 0
);
```

Tier values: `main` (1 per event, headline sponsor), `partner` (mid-tier), `supporter` (smallest logos).

### 3. Seed sponsor data — `db/seed.sql`

Add 4–5 seed sponsors at the end of `seed.sql`. Use realistic-sounding names and placeholder logo URLs (use `https://placehold.co/` service for demo logos with the sponsor name embedded):

```sql
-- Sponsors for Demo Slalom 2026
INSERT OR IGNORE INTO sponsors (id, event_id, name, logo_url, website_url, tier, sort_order)
VALUES (1, 1, 'WaterForce Energy', 'https://placehold.co/240x80/1e3a5f/ffffff?text=WaterForce+Energy', 'https://example.com/waterforce', 'main', 1);

INSERT OR IGNORE INTO sponsors (id, event_id, name, logo_url, website_url, tier, sort_order)
VALUES (2, 1, 'PaddleTech', 'https://placehold.co/160x60/2563eb/ffffff?text=PaddleTech', 'https://example.com/paddletech', 'partner', 2);

INSERT OR IGNORE INTO sponsors (id, event_id, name, logo_url, website_url, tier, sort_order)
VALUES (3, 1, 'Alpine Rapids Gear', 'https://placehold.co/160x60/059669/ffffff?text=Alpine+Rapids', 'https://example.com/alpinerapids', 'partner', 3);

INSERT OR IGNORE INTO sponsors (id, event_id, name, logo_url, website_url, tier, sort_order)
VALUES (4, 1, 'River City Tourism', 'https://placehold.co/120x45/6b7280/ffffff?text=River+City', 'https://example.com/rivercity', 'supporter', 4);

INSERT OR IGNORE INTO sponsors (id, event_id, name, logo_url, website_url, tier, sort_order)
VALUES (5, 1, 'CzechPaddle.cz', 'https://placehold.co/120x45/dc2626/ffffff?text=CzechPaddle', 'https://example.com/czechpaddle', 'supporter', 5);
```

### 4. Store queries — `store/sponsors.go`

Create a new file `store/sponsors.go`:

```go
package store

import (
    "database/sql"
    "canoe-slalom-live/domain"
)

// ListSponsorsByEvent returns all sponsors for an event, ordered by tier priority then sort_order.
func ListSponsorsByEvent(db *sql.DB, eventID int) ([]domain.Sponsor, error) {
    // Order: main first, then partner, then supporter
    rows, err := db.Query(`
        SELECT id, event_id, name, logo_url, website_url, tier, sort_order
        FROM sponsors
        WHERE event_id = ?
        ORDER BY
            CASE tier WHEN 'main' THEN 1 WHEN 'partner' THEN 2 WHEN 'supporter' THEN 3 ELSE 4 END,
            sort_order`,
        eventID,
    )
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var sponsors []domain.Sponsor
    for rows.Next() {
        var s domain.Sponsor
        if err := rows.Scan(&s.ID, &s.EventID, &s.Name, &s.LogoURL, &s.WebsiteURL, &s.Tier, &s.SortOrder); err != nil {
            return nil, err
        }
        sponsors = append(sponsors, s)
    }
    return sponsors, rows.Err()
}

// GetMainSponsor returns the main sponsor for an event (or empty if none).
func GetMainSponsor(db *sql.DB, eventID int) (domain.Sponsor, error) {
    var s domain.Sponsor
    err := db.QueryRow(`
        SELECT id, event_id, name, logo_url, website_url, tier, sort_order
        FROM sponsors
        WHERE event_id = ? AND tier = 'main'
        ORDER BY sort_order
        LIMIT 1`,
        eventID,
    ).Scan(&s.ID, &s.EventID, &s.Name, &s.LogoURL, &s.WebsiteURL, &s.Tier, &s.SortOrder)
    return s, err
}
```

### 5. Update handler data structs — `handler/public.go`

Add sponsor data to `EventPageData` and `LeaderboardPageData`:

```go
type EventPageData struct {
    Event      domain.Event
    Categories []CategoryWithEntries
    Sponsors   []domain.Sponsor      // all sponsors grouped for display
    Title      string
}

type LeaderboardPageData struct {
    Event        domain.Event
    Categories   []CategoryLeaderboard
    MainSponsor  *domain.Sponsor       // optional "powered by" line
    Title        string
}
```

### 6. Update `EventPage` handler

After fetching categories, also fetch sponsors:

```go
sponsors, err := store.ListSponsorsByEvent(d.DB, event.ID)
if err != nil {
    log.Printf("Error fetching sponsors: %v", err)
    // Don't fail the page for sponsors — just leave empty
}

data := EventPageData{
    Event:      event,
    Categories: catsWithEntries,
    Sponsors:   sponsors,
    Title:      event.Name + " — Canoe Slalom Live",
}
```

### 7. Update `LeaderboardPage` handler

After fetching leaderboard data, also fetch the main sponsor:

```go
mainSponsor, err := store.GetMainSponsor(d.DB, event.ID)
if err == nil {
    data.MainSponsor = &mainSponsor
}
// sql.ErrNoRows is fine — just means no main sponsor
```

### 8. Event page sponsor section — `templates/event.html`

Add a sponsor section at the bottom of the event page (before `{{end}}`):

```html
{{if .Sponsors}}
<section class="sponsors-section">
    <h3 class="sponsors-heading">Supported By</h3>
    <div class="sponsors-grid">
        {{range .Sponsors}}
        <a href="{{.WebsiteURL}}" target="_blank" rel="noopener" class="sponsor-card sponsor-{{.Tier}}" title="{{.Name}}">
            <img src="{{.LogoURL}}" alt="{{.Name}}" loading="lazy">
        </a>
        {{end}}
    </div>
</section>
{{end}}
```

### 9. Leaderboard sponsor line — `templates/leaderboard_partial.html`

At the very bottom of the `leaderboard-tables` block (after the `leaderboard-note` paragraph), add:

```html
{{if $.MainSponsor}}
<p class="sponsor-powered-by">
    Powered by
    <a href="{{$.MainSponsor.WebsiteURL}}" target="_blank" rel="noopener">
        <img src="{{$.MainSponsor.LogoURL}}" alt="{{$.MainSponsor.Name}}" class="sponsor-inline-logo">
    </a>
</p>
{{end}}
```

**Note:** The `MainSponsor` field must be available from the partial template's data. When rendering `?partial=1`, the handler passes the same `LeaderboardPageData` struct to the partial, so `$.MainSponsor` is accessible.

### 10. CSS — `static/style.css`

Add sponsor styling:

```css
/* === Sponsors === */
.sponsors-section {
    margin-top: 3rem;
    padding-top: 2rem;
    border-top: 1px solid #e5e7eb;
    text-align: center;
}

.sponsors-heading {
    font-size: 0.85rem;
    text-transform: uppercase;
    letter-spacing: 0.1em;
    color: #9ca3af;
    margin-bottom: 1rem;
    font-weight: 600;
}

.sponsors-grid {
    display: flex;
    flex-wrap: wrap;
    justify-content: center;
    align-items: center;
    gap: 1.5rem;
}

.sponsor-card {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    transition: opacity 0.2s;
}
.sponsor-card:hover {
    opacity: 0.7;
    text-decoration: none;
}

/* Tier-based sizing */
.sponsor-main img {
    max-height: 60px;
    max-width: 220px;
}
.sponsor-partner img {
    max-height: 45px;
    max-width: 160px;
}
.sponsor-supporter img {
    max-height: 32px;
    max-width: 120px;
    opacity: 0.7;
}
.sponsor-supporter:hover img {
    opacity: 1;
}

/* "Powered by" on leaderboard */
.sponsor-powered-by {
    text-align: center;
    font-size: 0.8rem;
    color: #9ca3af;
    margin-top: 1rem;
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 0.5rem;
}
.sponsor-powered-by a {
    display: inline-flex;
    align-items: center;
}
.sponsor-powered-by a:hover {
    opacity: 0.7;
    text-decoration: none;
}
.sponsor-inline-logo {
    max-height: 24px;
    max-width: 120px;
    vertical-align: middle;
}

/* Mobile: stack sponsors vertically */
@media (max-width: 500px) {
    .sponsors-grid {
        flex-direction: column;
        gap: 1rem;
    }
}
```

## Verification

1. `go build ./...` — no errors.
2. Delete `data.db` and re-seed: `Remove-Item -Force data.db; go run main.go -seed`.
3. Open `http://localhost:8080/events/demo-slalom-2026` — scroll down, see "Supported By" section with 5 sponsor logos at bottom. Main sponsor is largest. Supporter logos are smallest and slightly transparent.
4. Click a sponsor logo — opens `https://example.com/...` in new tab.
5. Open leaderboard page — see "Powered by [WaterForce Energy logo]" line at bottom below the tables.
6. Leaderboard partial refresh (`?partial=1`) — the "Powered by" line is included in the partial HTML.
7. Resize to mobile width — sponsor logos stack vertically.
8. Verify that removing all sponsor rows from the DB causes the sections to gracefully disappear (no broken layout, no empty headers).

## Files to create/modify

```
domain/sponsor.go        (create — Sponsor struct)
store/sponsors.go        (create — ListSponsorsByEvent, GetMainSponsor)
db/migrations.sql        (modify — add sponsors CREATE TABLE)
db/seed.sql              (modify — add sponsor seed data)
handler/public.go        (modify — add Sponsors/MainSponsor to data structs, fetch in handlers)
templates/event.html     (modify — add sponsors section)
templates/leaderboard_partial.html  (modify — add "powered by" line)
static/style.css         (modify — add sponsor CSS)
```

## Important notes

- Sponsors are seeded via SQL — no admin CRUD form for sponsors in this prompt. For the hackathon, editing sponsors means editing `seed.sql` and re-seeding, which is fine.
- Use `https://placehold.co/` placeholder images. In a real deployment these would be actual logo files or URLs.
- `WebsiteURL` can be empty — if so, render the logo without an `<a>` wrapper, or just skip the link. The template should handle `{{if .WebsiteURL}}`.
- Don't break the existing layout — the sponsor section should feel like a natural addition to the bottom of pages, not dominating anything.
- The partial leaderboard template receives `LeaderboardPageData` as its context, so `$.MainSponsor` works there.
