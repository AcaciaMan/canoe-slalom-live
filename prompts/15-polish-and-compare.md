# Prompt 15 — Phase 3 Polish and Integration

## Context

You are working on **canoe-slalom-live**, a Go web app for canoe/water slalom competitions. Stack: Go `net/http` (Go 1.22+), `html/template`, SQLite via `github.com/mattn/go-sqlite3`.

Read `PLAN.md` (Phase 3). Previous Phase 3 prompts:
- **Prompt 11**: Sponsors — schema, seed data, event page footer, leaderboard "powered by" line.
- **Prompt 12**: Photo gallery — photos table, gallery page, athlete photos on profile.
- **Prompt 13**: Commentator view — big-screen commentator page with latest run, top 3, auto-refresh.
- **Prompt 14**: Penalty sparklines — CSS sparkline bars on leaderboard run columns.

**Current codebase structure after Prompt 14:**
```
main.go                          — entry point, templates, routes
db/db.go                         — Open() and Seed()
db/migrations.sql                — 7 tables: events, categories, athletes, entries, runs, sponsors, photos
db/seed.sql                      — demo data for all tables
domain/event.go                  — Event, Category
domain/athlete.go                — Athlete, Entry
domain/run.go                    — Run struct, formatting helpers
domain/sponsor.go                — Sponsor struct
domain/photo.go                  — Photo struct
store/events.go                  — GetEventBySlug(), ListCategories()
store/athletes.go                — GetAthlete(), ListEntriesByCategory(), GetEntryByEventAndAthlete()
store/runs.go                    — CRUD, GetLeaderboard(), GetLatestRun(), ListRecentRuns()
store/sponsors.go                — ListSponsorsByEvent(), GetMainSponsor()
store/photos.go                  — ListPhotosByEvent(), ListPhotosByAthlete()
handler/public.go                — EventPage, AthletePage, LeaderboardPage, GalleryPage, CommentatorPage
handler/judge.go                 — JudgePage, SubmitRun, EditRunPage, UpdateRunHandler, DeleteRunHandler
handler/auth.go                  — RequireAuth, SessionStore
handler/helpers.go               — renderError()
handler/logging.go               — LoggingMiddleware, SecurityHeaders
templates/layout.html            — shared shell (nav + footer)
templates/event.html             — event page with start list, sponsors, tab bar
templates/athlete.html           — athlete profile (bio, runs, photos)
templates/leaderboard.html       — leaderboard with auto-refresh
templates/leaderboard_partial.html — leaderboard tables with sparklines
templates/gallery.html           — photo gallery grid
templates/commentator.html       — commentator container
templates/commentator_partial.html — commentator inner content
templates/judge_run.html         — judge form
templates/judge_edit_run.html    — edit run form
templates/error.html             — error page
static/style.css                 — full stylesheet
static/app.js                    — auto-refresh for leaderboard + commentator
```

**Current routes:**
```go
GET  /{$}                                          → redirect
GET  /events/{slug}                                → deps.EventPage
GET  /events/{slug}/leaderboard                    → deps.LeaderboardPage
GET  /events/{slug}/athletes/{id}                  → deps.AthletePage
GET  /events/{slug}/photos                         → deps.GalleryPage
GET  /events/{slug}/commentator                    → deps.CommentatorPage
GET  /judge/events/{slug}                          → deps.RequireAuth(deps.JudgePage)
POST /judge/events/{slug}/runs                     → deps.RequireAuth(deps.SubmitRun)
GET  /judge/events/{slug}/runs/{id}/edit           → deps.RequireAuth(deps.EditRunPage)
POST /judge/events/{slug}/runs/{id}                → deps.RequireAuth(deps.UpdateRunHandler)
POST /judge/events/{slug}/runs/{id}/delete         → deps.RequireAuth(deps.DeleteRunHandler)
GET  /static/                                      → file server
```

## Goal

Final polish pass for Phase 3. Fix any cross-feature integration issues, enrich the athlete profile with cross-event history awareness, add a head-to-head comparison page (the second "wow" detail from PLAN.md), and do a comprehensive CSS consistency pass.

## What to build

### 1. Head-to-head comparison page

**Route:** `GET /events/{slug}/compare?a={id1}&b={id2}`

This is the second "wow" detail from PLAN.md: pick two athletes and see their runs side by side with differences highlighted.

#### Store query — `store/runs.go`

No new store function needed. Reuse:
- `store.GetAthlete(db, athleteID)` for both athletes.
- `store.GetEntryByEventAndAthlete(db, eventID, athleteID)` for both entries.
- `store.ListRunsByEntry(db, entryID)` for both run sets.

#### Handler — `handler/public.go`

```go
// ComparePageData is the template data for the head-to-head comparison page.
type ComparePageData struct {
    Event    domain.Event
    AthleteA CompareAthlete
    AthleteB CompareAthlete
    Title    string
}

// CompareAthlete holds one side of the comparison.
type CompareAthlete struct {
    Athlete domain.Athlete
    Entry   store.EntryWithAthlete
    Run1    *domain.Run
    Run2    *domain.Run
    BestMs  int // best total_time_ms
}

// ComparePage handles GET /events/{slug}/compare?a={id1}&b={id2}.
func (d *Deps) ComparePage(w http.ResponseWriter, r *http.Request) {
    slug := r.PathValue("slug")

    event, err := store.GetEventBySlug(d.DB, slug)
    if err == sql.ErrNoRows {
        d.renderError(w, 404, "Event not found")
        return
    }
    if err != nil {
        log.Printf("Error fetching event: %v", err)
        d.renderError(w, 500, "Internal server error")
        return
    }

    aID, err := strconv.Atoi(r.URL.Query().Get("a"))
    if err != nil || aID <= 0 {
        d.renderError(w, 400, "Missing or invalid athlete A (?a= parameter)")
        return
    }
    bID, err := strconv.Atoi(r.URL.Query().Get("b"))
    if err != nil || bID <= 0 {
        d.renderError(w, 400, "Missing or invalid athlete B (?b= parameter)")
        return
    }
    if aID == bID {
        d.renderError(w, 400, "Cannot compare an athlete to themselves")
        return
    }

    loadAthlete := func(athleteID int) (CompareAthlete, error) {
        var ca CompareAthlete
        athlete, err := store.GetAthlete(d.DB, athleteID)
        if err != nil {
            return ca, err
        }
        ca.Athlete = athlete

        entry, err := store.GetEntryByEventAndAthlete(d.DB, event.ID, athleteID)
        if err != nil {
            return ca, err
        }
        ca.Entry = entry

        runs, err := store.ListRunsByEntry(d.DB, entry.EntryID)
        if err != nil {
            return ca, err
        }

        for i := range runs {
            if runs[i].RunNumber == 1 {
                ca.Run1 = &runs[i]
            }
            if runs[i].RunNumber == 2 {
                ca.Run2 = &runs[i]
            }
        }

        // Compute best time
        if ca.Run1 != nil && ca.Run1.Status == "ok" {
            ca.BestMs = ca.Run1.TotalTimeMs
        }
        if ca.Run2 != nil && ca.Run2.Status == "ok" {
            if ca.BestMs == 0 || ca.Run2.TotalTimeMs < ca.BestMs {
                ca.BestMs = ca.Run2.TotalTimeMs
            }
        }
        return ca, nil
    }

    athleteA, err := loadAthlete(aID)
    if err != nil {
        d.renderError(w, 404, "Athlete A not found or not entered in this event")
        return
    }
    athleteB, err := loadAthlete(bID)
    if err != nil {
        d.renderError(w, 404, "Athlete B not found or not entered in this event")
        return
    }

    data := ComparePageData{
        Event:    event,
        AthleteA: athleteA,
        AthleteB: athleteB,
        Title:    athleteA.Athlete.Name + " vs " + athleteB.Athlete.Name + " — " + event.Name,
    }

    if err := d.Tmpls["compare"].ExecuteTemplate(w, "layout.html", data); err != nil {
        log.Printf("Error rendering compare page: %v", err)
    }
}
```

#### Template — `templates/compare.html`

```html
{{define "content"}}
<nav class="breadcrumb">
    <a href="/events/{{.Event.Slug}}">{{.Event.Name}}</a> → Head-to-Head
</nav>

<h1 class="compare-title">
    <span class="compare-name-a">{{.AthleteA.Athlete.Name}}</span>
    <span class="compare-vs">vs</span>
    <span class="compare-name-b">{{.AthleteB.Athlete.Name}}</span>
</h1>

<div class="compare-grid">
    {{/* Athlete A column */}}
    <div class="compare-column compare-col-a">
        <div class="compare-athlete-header">
            <span class="compare-bib">#{{.AthleteA.Entry.BibNumber}}</span>
            <a href="/events/{{.Event.Slug}}/athletes/{{.AthleteA.Athlete.ID}}">{{.AthleteA.Athlete.Name}}</a>
            <div class="compare-meta">{{.AthleteA.Entry.Club}} · {{.AthleteA.Athlete.Nation}}</div>
        </div>

        <div class="compare-run-block">
            <h3>Run 1</h3>
            {{if .AthleteA.Run1}}
            <div class="compare-time">{{.AthleteA.Run1.TotalTimeFormatted}}</div>
            <div class="compare-detail">Raw: {{.AthleteA.Run1.RawTimeFormatted}} | {{.AthleteA.Run1.PenaltyDisplay}}</div>
            {{else}}<div class="compare-no-run">Not yet</div>{{end}}
        </div>

        <div class="compare-run-block">
            <h3>Run 2</h3>
            {{if .AthleteA.Run2}}
            <div class="compare-time">{{.AthleteA.Run2.TotalTimeFormatted}}</div>
            <div class="compare-detail">Raw: {{.AthleteA.Run2.RawTimeFormatted}} | {{.AthleteA.Run2.PenaltyDisplay}}</div>
            {{else}}<div class="compare-no-run">Not yet</div>{{end}}
        </div>

        <div class="compare-best-block">
            <h3>Best Time</h3>
            {{if gt .AthleteA.BestMs 0}}
            <div class="compare-best-time {{if and (gt .AthleteA.BestMs 0) (gt .AthleteB.BestMs 0) (le .AthleteA.BestMs .AthleteB.BestMs)}}compare-winner{{end}}">
                {{formatTime .AthleteA.BestMs}}
            </div>
            {{else}}<div class="compare-no-run">—</div>{{end}}
        </div>
    </div>

    {{/* Center divider */}}
    <div class="compare-divider">
        <div class="compare-vs-badge">VS</div>
        {{if and (gt .AthleteA.BestMs 0) (gt .AthleteB.BestMs 0)}}
        <div class="compare-diff">
            {{if lt .AthleteA.BestMs .AthleteB.BestMs}}
                <span class="compare-diff-value">{{.AthleteA.Athlete.Name}} by {{formatTime (sub .AthleteB.BestMs .AthleteA.BestMs)}}</span>
            {{else if lt .AthleteB.BestMs .AthleteA.BestMs}}
                <span class="compare-diff-value">{{.AthleteB.Athlete.Name}} by {{formatTime (sub .AthleteA.BestMs .AthleteB.BestMs)}}</span>
            {{else}}
                <span class="compare-diff-value">Tied!</span>
            {{end}}
        </div>
        {{end}}
    </div>

    {{/* Athlete B column */}}
    <div class="compare-column compare-col-b">
        <div class="compare-athlete-header">
            <span class="compare-bib">#{{.AthleteB.Entry.BibNumber}}</span>
            <a href="/events/{{.Event.Slug}}/athletes/{{.AthleteB.Athlete.ID}}">{{.AthleteB.Athlete.Name}}</a>
            <div class="compare-meta">{{.AthleteB.Entry.Club}} · {{.AthleteB.Athlete.Nation}}</div>
        </div>

        <div class="compare-run-block">
            <h3>Run 1</h3>
            {{if .AthleteB.Run1}}
            <div class="compare-time">{{.AthleteB.Run1.TotalTimeFormatted}}</div>
            <div class="compare-detail">Raw: {{.AthleteB.Run1.RawTimeFormatted}} | {{.AthleteB.Run1.PenaltyDisplay}}</div>
            {{else}}<div class="compare-no-run">Not yet</div>{{end}}
        </div>

        <div class="compare-run-block">
            <h3>Run 2</h3>
            {{if .AthleteB.Run2}}
            <div class="compare-time">{{.AthleteB.Run2.TotalTimeFormatted}}</div>
            <div class="compare-detail">Raw: {{.AthleteB.Run2.RawTimeFormatted}} | {{.AthleteB.Run2.PenaltyDisplay}}</div>
            {{else}}<div class="compare-no-run">Not yet</div>{{end}}
        </div>

        <div class="compare-best-block">
            <h3>Best Time</h3>
            {{if gt .AthleteB.BestMs 0}}
            <div class="compare-best-time {{if and (gt .AthleteA.BestMs 0) (gt .AthleteB.BestMs 0) (le .AthleteB.BestMs .AthleteA.BestMs)}}compare-winner{{end}}">
                {{formatTime .AthleteB.BestMs}}
            </div>
            {{else}}<div class="compare-no-run">—</div>{{end}}
        </div>
    </div>
</div>

<div class="compare-footer">
    <a href="/events/{{.Event.Slug}}/leaderboard" class="back-link">← Back to Leaderboard</a>
</div>
{{end}}
```

**Note:** The template uses a `sub` template function (subtract two ints). Add this to `funcMap` in `main.go`:

```go
"sub": func(a, b int) int { return a - b },
"le":  func(a, b int) bool { return a <= b },
"lt":  func(a, b int) bool { return a < b },
```

#### Register in `main.go`

Template:
```go
"compare": template.Must(template.New("layout.html").Funcs(funcMap).ParseFiles("templates/layout.html", "templates/compare.html")),
```

Route:
```go
mux.HandleFunc("GET /events/{slug}/compare", deps.ComparePage)
```

### 2. Compare links on the leaderboard

Add a subtle "Compare" feature on the leaderboard. On each athlete row, add a checkbox-based selection mechanism. When two athletes are checked, show a "Compare" button.

**Simpler alternative (recommended for hackathon):** Don't add checkboxes to the leaderboard. Instead, on the **athlete profile page** (`templates/athlete.html`), add a "Compare with..." dropdown that lists other athletes in the same category. This is simpler and less disruptive to the leaderboard layout.

In `templates/athlete.html`, add after the runs section:

```html
<section class="compare-section">
    <h2>⚔️ Head-to-Head</h2>
    <form action="/events/{{.Event.Slug}}/compare" method="GET" class="compare-form">
        <input type="hidden" name="a" value="{{.Athlete.ID}}">
        <label for="compare-with">Compare with:</label>
        <select name="b" id="compare-with" required>
            <option value="">Select athlete...</option>
            <!-- Populated from other athletes in same event -->
        </select>
        <button type="submit" class="btn-compare">Compare ⚔️</button>
    </form>
</section>
```

To populate the dropdown, add a list of other athletes to `AthletePageData`:

```go
type AthletePageData struct {
    Event       domain.Event
    Athlete     domain.Athlete
    Entry       store.EntryWithAthlete
    Runs        []domain.Run
    Photos      []domain.Photo
    OtherEntries []store.EntryWithAthlete  // other athletes in same category
    Title       string
}
```

In the `AthletePage` handler, fetch other entries in the same category:

```go
// After fetching entry...
otherEntries, err := store.ListEntriesByCategory(d.DB, entry.CategoryID)
if err != nil {
    log.Printf("Error fetching other entries: %v", err)
}
// Filter out current athlete
var filtered []store.EntryWithAthlete
for _, oe := range otherEntries {
    if oe.AthleteID != athleteID {
        filtered = append(filtered, oe)
    }
}
data.OtherEntries = filtered
```

Then in the template dropdown:
```html
{{range .OtherEntries}}
<option value="{{.AthleteID}}">#{{.BibNumber}} {{.AthleteName}}</option>
{{end}}
```

### 3. CSS for compare page — `static/style.css`

```css
/* === Head-to-Head Compare === */
.compare-title {
    text-align: center;
    font-size: 1.5rem;
    margin-bottom: 2rem;
}

.compare-vs {
    display: inline-block;
    margin: 0 0.75rem;
    color: #9ca3af;
    font-weight: 400;
    font-size: 1rem;
}

.compare-grid {
    display: grid;
    grid-template-columns: 1fr auto 1fr;
    gap: 1.5rem;
    align-items: start;
}

.compare-column {
    background: #f9fafb;
    border: 1px solid #e5e7eb;
    border-radius: 10px;
    padding: 1.5rem;
}

.compare-athlete-header {
    margin-bottom: 1.5rem;
    text-align: center;
}

.compare-bib {
    color: #6b7280;
    font-weight: 600;
    font-size: 0.9rem;
    margin-right: 0.25rem;
}

.compare-athlete-header a {
    font-size: 1.2rem;
    font-weight: 700;
}

.compare-meta {
    font-size: 0.85rem;
    color: #6b7280;
    margin-top: 0.25rem;
}

.compare-run-block {
    border-top: 1px solid #e5e7eb;
    padding: 1rem 0;
}
.compare-run-block h3 {
    font-size: 0.8rem;
    text-transform: uppercase;
    letter-spacing: 0.1em;
    color: #9ca3af;
    margin-bottom: 0.5rem;
}

.compare-time {
    font-size: 1.8rem;
    font-weight: 700;
    color: #1e3a5f;
    font-variant-numeric: tabular-nums;
    text-align: center;
}

.compare-detail {
    font-size: 0.85rem;
    color: #6b7280;
    text-align: center;
    margin-top: 0.25rem;
}

.compare-no-run {
    text-align: center;
    color: #9ca3af;
    font-style: italic;
    padding: 0.75rem 0;
}

.compare-best-block {
    border-top: 2px solid #1e3a5f;
    padding: 1rem 0;
    text-align: center;
}
.compare-best-block h3 {
    font-size: 0.8rem;
    text-transform: uppercase;
    letter-spacing: 0.1em;
    color: #1e3a5f;
    margin-bottom: 0.5rem;
}

.compare-best-time {
    font-size: 2rem;
    font-weight: 800;
    font-variant-numeric: tabular-nums;
    color: #1e3a5f;
}

.compare-winner {
    color: #059669;
}
.compare-winner::after {
    content: " ✓";
    font-size: 1rem;
}

/* Center divider */
.compare-divider {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    padding: 2rem 0;
}

.compare-vs-badge {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 48px;
    height: 48px;
    border-radius: 50%;
    background: #1e3a5f;
    color: #fff;
    font-weight: 800;
    font-size: 0.9rem;
    margin-bottom: 1rem;
}

.compare-diff {
    text-align: center;
}

.compare-diff-value {
    font-size: 0.85rem;
    color: #059669;
    font-weight: 600;
}

.compare-footer {
    margin-top: 2rem;
    text-align: center;
}

/* Compare form on athlete page */
.compare-section {
    margin-top: 2rem;
    padding-top: 1rem;
    border-top: 1px solid #e5e7eb;
}

.compare-form {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    flex-wrap: wrap;
}

.compare-form label {
    font-weight: 600;
    font-size: 0.9rem;
}

.compare-form select {
    padding: 0.5rem;
    border: 1px solid #d1d5db;
    border-radius: 6px;
    font-size: 0.9rem;
    min-width: 200px;
}

.btn-compare {
    background: #1e3a5f;
    color: #fff;
    border: none;
    padding: 0.5rem 1rem;
    border-radius: 6px;
    font-weight: 600;
    cursor: pointer;
    font-size: 0.9rem;
}
.btn-compare:hover {
    background: #2d5a8e;
}

/* Mobile: compare stacks vertically */
@media (max-width: 700px) {
    .compare-grid {
        grid-template-columns: 1fr;
    }
    .compare-divider {
        flex-direction: row;
        padding: 1rem 0;
    }
    .compare-vs-badge {
        margin-bottom: 0;
        margin-right: 1rem;
    }
    .compare-time, .compare-best-time {
        font-size: 1.5rem;
    }
}
```

### 4. Consistency check — tab bar standardization

Ensure all pages that belong to the event context have a consistent tab bar. Currently:
- `event.html`: Start List | Leaderboard | Photos
- `leaderboard.html`: Start List | Leaderboard | Photos
- `gallery.html`: Start List | Leaderboard | Photos

Verify all three `tab-bar` sections match the same structure and active class logic.

### 5. Update `leaderboard_partial.html` — add leaderboard note text update

If not already present after Prompt 14, ensure the leaderboard note mentions the sparkline:

```html
<p class="leaderboard-note">Ranked by best single-run time. Penalties: gate touch = 2s, missed gate = 50s. Bars show raw time (navy) vs penalties (amber/red).</p>
```

### 6. Verify sponsors appear on the correct pages

After all prompts, sponsors should appear:
- **Event page** (`event.html`): "Supported By" section with all sponsor logos at bottom.
- **Leaderboard** (`leaderboard_partial.html`): "Powered by [main sponsor]" line at bottom.
- **Commentator view** (`commentator_partial.html`): Main sponsor logo in the standings panel.

If any of these were missed, add them now.

## Verification

1. `go build ./...` — no errors.
2. Delete `data.db` and re-seed: `Remove-Item -Force data.db; go run main.go -seed`.
3. **Compare page**: Open `http://localhost:8080/events/demo-slalom-2026/compare?a=1&b=2` (Jan Rohan vs Oliver Bennett). See side-by-side comparison with Run 1, Run 2, and Best Time for each. The winner's best time is highlighted in green with a checkmark. The difference is shown in the center divider.
4. **Compare from athlete page**: Open Jan Rohan's profile → see "Head-to-Head" section with dropdown of other K1M athletes → select Oliver Bennett → submit → goes to compare page.
5. **Error handling**: Try `/compare?a=1` (missing b) → 400 error. Try `/compare?a=1&b=1` (same athlete) → 400 error. Try `/compare?a=1&b=999` (non-existent) → 404 error.
6. **Tab bar consistency**: Check that Start List, Leaderboard, and Photos tabs appear identically on all three pages with the correct one highlighted as active.
7. **Sponsors integration**: Event page has sponsor logos. Leaderboard has "Powered by" line. Commentator has sponsor logo.
8. **Mobile**: Compare page stacks columns vertically on narrow screens.
9. **All pages work**: Run through every page (event, leaderboard, athlete, gallery, commentator, judge, compare) to verify no regressions.
10. **CSS file size**: Should be under ~1000 lines total. Run `Get-Content static/style.css | Measure-Object -Line` to verify.

## Files to create/modify

```
templates/compare.html       (create — head-to-head comparison page)
handler/public.go            (modify — add ComparePage handler, ComparePageData, CompareAthlete structs, add OtherEntries to AthletePageData)
main.go                      (modify — add compare template, add route, add sub/le/lt template functions)
templates/athlete.html       (modify — add compare form section)
templates/leaderboard_partial.html  (modify — update leaderboard note text if needed)
static/style.css             (modify — add compare page CSS)
```

## Important notes

- The compare page is a "shareable screenshot" feature. Keep it visually clean for social media sharing. The side-by-side layout with clear winner highlighting makes for a great screenshot.
- The `sub` template function is needed for computing time differences in the template. This is a simple `func(a, b int) int { return a - b }`.
- The `le` and `lt` template functions are needed for comparison logic in the template. Go's `html/template` only has `eq`, `ne`, `gt`, `ge`, `lt`, `le` as built-in comparison — actually `le` and `lt` ARE built-in. Check if they work with the syntax `{{if le .A .B}}`. If they're available as built-ins, don't add them to `funcMap`. Test and verify.
- The compare form on the athlete page uses a simple `<select>` dropdown. It only shows athletes in the same category, which makes the most sense for comparison.
- Don't add compare checkboxes to the leaderboard — it adds complexity and clutters the table. The athlete profile dropdown is cleaner.
- This is the final prompt of Phase 3. After completion, the app should have: sponsors, photo gallery, commentator view, penalty sparklines, and head-to-head comparison. That's a compelling feature set for a hackathon demo.
