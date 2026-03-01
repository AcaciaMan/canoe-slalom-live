# Prompt 02 — Public Event Page with Start List

## Context

You are working on **canoe-slalom-live**, a Go web app for canoe/water slalom competitions. Stack: Go `net/http`, `html/template`, SQLite.

Read `PLAN.md` for full architecture. The previous step (Prompt 01) built:
- `db/db.go` — opens SQLite, runs migrations via `go:embed`
- `db/migrations.sql` — schema for events, categories, athletes, entries, runs
- `db/seed.sql` — demo event with 10 athletes across 2 categories
- `domain/*.go` — structs: Event, Category, Athlete, Entry, Run (with time formatting helpers)
- `store/*.go` — query functions: `GetEventBySlug`, `ListCategories`, `ListEntriesByCategory`, `GetAthlete`
- `main.go` — opens DB, optionally seeds, then exits

This prompt implements **Phase 1, Step 2**: the public event page showing start list and athlete profiles, plus wiring up the HTTP server.

## Goal

A visitor can open `http://localhost:8080/events/demo-slalom-2026` in a browser and see the event name, date, location, and a start list grouped by category. Clicking an athlete name shows their profile/bio.

## What to build

### 1. HTTP server wiring in `main.go`

Update `main.go` to:
- After opening the DB (and optionally seeding), start an HTTP server on `:8080` (or from `PORT` env var).
- Use Go 1.22+ routing patterns with `http.NewServeMux()`.
- Serve static files: `GET /static/` → serves from `static/` directory.
- Register handler routes (details below).
- Log "Server starting on :8080" to stdout.

Route registration (for this prompt, only the public routes):
```
GET /                           → redirect to /events/demo-slalom-2026 (hardcoded for MVP)
GET /events/{slug}              → handler/public.go: EventPage
GET /events/{slug}/athletes/{id} → handler/public.go: AthletePage
```

### 2. Template layout — `templates/layout.html`

A shared HTML shell used by all pages. Must:
- Set `<!DOCTYPE html>`, charset UTF-8, viewport meta for mobile.
- `<title>` based on a `.Title` field from template data, defaulting to "Canoe Slalom Live".
- Link to `/static/style.css`.
- Header bar with: site name "🛶 Canoe Slalom Live" linking to `/`, navigation placeholder.
- A `{{block "content" .}}{{end}}` where page content goes.
- Footer with "Canoe Slalom Live — Grassroots timing made simple".
- Load `/static/app.js` at end of body (file can be empty for now).

### 3. Event page — `templates/event.html`

Template data struct (define in handler or as a nested struct):
```
PageData {
    Event      domain.Event
    Categories []CategoryWithEntries
    Title      string
}

CategoryWithEntries {
    Category domain.Category
    Entries  []store.EntryWithAthlete
}
```

Layout:
- **Event header**: Event name (h1), date (formatted nicely, e.g., "15 June 2026"), location, status badge (e.g., "🟢 Active").
- **Tab bar** with two tabs: "Start List" (active by default), "Leaderboard" (links to `/events/{slug}/leaderboard` — will be built in Prompt 04, for now just a dead link or a placeholder message).
- **Start list**, grouped by category:
  - Category heading: e.g., "K1M — Kayak Single Men"
  - Table with columns: Bib | Athlete | Club | Nation
  - Each athlete name is a link to `/events/{slug}/athletes/{athleteID}`
  - Nation shown as 3-letter code (e.g., "CZE")
  - Rows ordered by `start_position`

### 4. Athlete profile page — `templates/athlete.html`

Template data:
```
AthletePageData {
    Event   domain.Event
    Athlete domain.Athlete
    Entry   store.EntryWithAthlete  // for bib, category context
    Runs    []domain.Run            // may be empty in Phase 1 step 2
    Title   string
}
```

Layout:
- Breadcrumb: Event name (link) → Athlete name
- Athlete name (h1), bib number badge
- Club and nation
- Bio paragraph (the "commentator facts")
- Photo (if `photo_url` is set, show `<img>`; otherwise skip)
- **Runs section** (header "Results" with a table): will be empty until judge records runs. Show "No runs recorded yet." if empty. Table columns when data exists: Run # | Raw Time | Penalties | Total Time | Status. Use the Run formatting helpers from `domain/run.go`.
- Back link to event page

### 5. Handler — `handler/public.go`

Package `handler`. Needs access to `*sql.DB` and parsed templates.

**Approach for template + DB access:** Create a simple struct to hold shared dependencies:
```go
type Deps struct {
    DB        *sql.DB
    Templates *template.Template
}
```

Parse all templates once at startup (in `main.go`) using `template.ParseGlob("templates/*.html")` or by explicitly parsing layout + each page. Pass the `Deps` struct to handler functions. Use method receivers: `func (d *Deps) EventPage(w http.ResponseWriter, r *http.Request)`.

**EventPage handler** (`GET /events/{slug}`):
1. Extract `slug` from `r.PathValue("slug")`.
2. Fetch event via `store.GetEventBySlug(d.DB, slug)`. If not found, return 404.
3. Fetch categories via `store.ListCategories(d.DB, event.ID)`.
4. For each category, fetch entries via `store.ListEntriesByCategory(d.DB, cat.ID)`.
5. Build `PageData` and render `event.html` within `layout.html`.

**AthletePage handler** (`GET /events/{slug}/athletes/{id}`):
1. Extract `slug` and `id` from path.
2. Fetch event (for context/breadcrumb).
3. Fetch athlete via `store.GetAthlete(d.DB, id)`. 404 if not found.
4. Fetch entry for this athlete in this event (you may need a new store function: `store.GetEntryByEventAndAthlete(db, eventID, athleteID)`).
5. Fetch runs via `store.ListRunsByEntry(d.DB, entry.ID)`.
6. Render `athlete.html`.

If `store.GetEntryByEventAndAthlete` doesn't exist yet, add it to `store/athletes.go`:
```
func GetEntryByEventAndAthlete(db *sql.DB, eventID, athleteID int) (EntryWithAthlete, error)
```

### 6. Static files — `static/style.css`

Create a clean, minimal stylesheet. Key design:
- **System font stack**: `-apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif`.
- **Color palette**: White background, dark text (#1a1a1a), accent blue (#2563eb) for links and active tabs, light gray (#f3f4f6) for table striping, green (#16a34a) for "Active" badge.
- **Layout**: Max-width 900px centered container. Generous padding.
- **Header**: Blue background (#1e3a5f — a "water" navy blue), white text, sticky top.
- **Tables**: Full width, left-aligned, striped rows, no heavy borders. Bib column smaller width.
- **Mobile responsive**: Table scrolls horizontally on small screens. Stack layout elements vertically below 600px.
- **Tab bar**: Horizontal tabs using inline-block or flexbox. Active tab has bottom border accent.
- **Athlete page**: Bio text in a slightly indented blockquote style. Photo max 200px wide, float right.
- Keep it under 150 lines. No CSS framework.

### 7. Static files — `static/app.js`

Create an empty file with a comment: `// Canoe Slalom Live — JS will be added in later phases`.

## Template rendering approach

Use Go's `html/template` with explicit template names. Recommended pattern:

In `main.go`, parse templates:
```go
tmpl := template.Must(template.ParseGlob("templates/*.html"))
```

In handlers, execute with:
```go
tmpl.ExecuteTemplate(w, "layout.html", data)
```

Each page template should `{{define "content"}}...{{end}}` and `layout.html` should `{{block "content" .}}{{end}}`.

**Important**: With `ParseGlob`, if `layout.html` uses `{{block "content"}}` and multiple page templates define `"content"`, only the last one parsed wins. To handle this properly, either:
- Parse each page separately: `template.Must(template.ParseFiles("templates/layout.html", "templates/event.html"))` — store each as a separate `*template.Template` in `Deps`.
- Or use `{{template "event-content" .}}` with unique names per page.

**Recommended approach**: Store templates as a map in Deps:
```go
type Deps struct {
    DB    *sql.DB
    Tmpls map[string]*template.Template  // key: "event", "athlete", etc.
}
```

Build each one:
```go
tmpls := map[string]*template.Template{
    "event":   template.Must(template.ParseFiles("templates/layout.html", "templates/event.html")),
    "athlete": template.Must(template.ParseFiles("templates/layout.html", "templates/athlete.html")),
}
```

Render via `d.Tmpls["event"].ExecuteTemplate(w, "layout.html", data)`.

## Verification

1. `go build ./...` — no errors.
2. `go run main.go -seed` — seeds DB and starts server.
3. Open `http://localhost:8080/` — redirects to event page.
4. Event page shows: event header, two category sections, 10 athletes with bibs.
5. Click an athlete name → athlete page with bio, empty runs table.
6. Back link returns to event page.
7. Page looks clean on both desktop and mobile (phone-width browser).

## Files to create/modify

```
main.go              (modify: add HTTP server, template parsing, route registration)
handler/public.go    (create)
templates/layout.html (create)
templates/event.html  (create)
templates/athlete.html (create)
static/style.css      (create)
static/app.js         (create)
```

May also need to add to `store/athletes.go`: `GetEntryByEventAndAthlete` function.

## Important notes

- Do NOT use any external template engine or CSS framework.
- All navigation links should use the event slug in URLs, not IDs.
- Athlete profile links use athlete ID (integer) in the URL: `/events/{slug}/athletes/{id}`.
- Handle 404 gracefully: if event or athlete not found, return a simple "Not Found" page, don't panic.
- The leaderboard tab link should point to `/events/{slug}/leaderboard` but can show "Coming soon" if clicked (that page is built in Prompt 04).
