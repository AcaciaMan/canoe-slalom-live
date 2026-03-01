# Prompt 13 — Commentator View

## Context

You are working on **canoe-slalom-live**, a Go web app for canoe/water slalom competitions. Stack: Go `net/http` (Go 1.22+), `html/template`, SQLite via `github.com/mattn/go-sqlite3`.

Read `PLAN.md` (Phase 3 — Commentator view). Previous Phase 3 prompts:
- **Prompt 11**: Sponsors schema and display (event footer, leaderboard "powered by").
- **Prompt 12**: Photo gallery page, athlete photos.

**Current codebase state after Prompt 12:**
The app now has sponsors display, a photo gallery page at `GET /events/{slug}/photos`, and athlete photos on profile pages. All Phase 1–2 features remain intact.

**Current codebase structure:**
```
main.go                          — entry point, template parsing, route registration
db/db.go                         — Open() and Seed() with go:embed
db/migrations.sql                — CREATE TABLE for events, categories, athletes, entries, runs, sponsors, photos
db/seed.sql                      — demo data (1 event, 2 categories, 10 athletes, entries, runs, sponsors, photos)
domain/event.go                  — Event, Category structs
domain/athlete.go                — Athlete, Entry structs
domain/run.go                    — Run struct with formatting helpers
domain/sponsor.go                — Sponsor struct
domain/photo.go                  — Photo struct
store/events.go                  — GetEventBySlug(), ListCategories()
store/athletes.go                — GetAthlete(), ListEntriesByCategory(), GetEntryByEventAndAthlete()
store/runs.go                    — CreateRun(), GetLeaderboard(), ListRecentRuns(), etc.
store/sponsors.go                — ListSponsorsByEvent(), GetMainSponsor()
store/photos.go                  — ListPhotosByEvent(), ListPhotosByAthlete()
handler/public.go                — Deps struct, EventPage, AthletePage, LeaderboardPage, GalleryPage
handler/judge.go                 — JudgePage, SubmitRun, EditRunPage, UpdateRunHandler, DeleteRunHandler
handler/auth.go                  — RequireAuth middleware, SessionStore
handler/helpers.go               — renderError()
handler/logging.go               — LoggingMiddleware, SecurityHeaders
templates/layout.html            — shared shell with nav (Event, Leaderboard, Photos, Judge) + footer
templates/event.html             — event page with tab bar (Start List / Leaderboard / Photos)
templates/athlete.html           — athlete profile with bio, runs, photos
templates/leaderboard.html       — leaderboard with auto-refresh
templates/leaderboard_partial.html — leaderboard partial
templates/gallery.html           — photo gallery grid
templates/judge_run.html         — judge form
templates/judge_edit_run.html    — edit run form
templates/error.html             — error page
static/style.css                 — full stylesheet
static/app.js                    — auto-refresh leaderboard
```

**Current routes:**
```go
GET  /{$}                                          → redirect
GET  /events/{slug}                                → deps.EventPage
GET  /events/{slug}/leaderboard                    → deps.LeaderboardPage
GET  /events/{slug}/athletes/{id}                  → deps.AthletePage
GET  /events/{slug}/photos                         → deps.GalleryPage
GET  /judge/events/{slug}                          → deps.RequireAuth(deps.JudgePage)
POST /judge/events/{slug}/runs                     → deps.RequireAuth(deps.SubmitRun)
GET  /judge/events/{slug}/runs/{id}/edit           → deps.RequireAuth(deps.EditRunPage)
POST /judge/events/{slug}/runs/{id}                → deps.RequireAuth(deps.UpdateRunHandler)
POST /judge/events/{slug}/runs/{id}/delete         → deps.RequireAuth(deps.DeleteRunHandler)
GET  /static/                                      → file server
```

**Key existing queries:**
- `store.GetLeaderboard(db, categoryID)` returns `[]LeaderboardRow` ranked by best total time.
- `store.ListRecentRuns(db, eventID, limit)` returns `[]RecentRun` ordered by `judged_at DESC`.
- `store.GetAthlete(db, athleteID)` returns `domain.Athlete` (includes Bio, PhotoURL).
- `store.GetEntryByID(db, entryID)` returns `EntryWithAthlete` (includes CategoryID, BibNumber, AthleteName, Club, Nation).

**Key data available in `LeaderboardRow`:**
```go
type LeaderboardRow struct {
    Rank            int
    BibNumber       int
    AthleteID       int
    AthleteName     string
    AthleteNation   string
    Run1            *RunResult  // nil if no Run 1 yet
    Run2            *RunResult  // nil if no Run 2 yet
    BestTotalTimeMs int
    HasRuns         bool
    TimeBehindMs    int
    Run1IsBest      bool
    Run2IsBest      bool
    Run1IsNew       bool
    Run2IsNew       bool
}
```

## Goal

Build a commentator view page designed for a second screen or tablet at the commentary booth. It shows: the most recently judged athlete with their bio/facts, their run results, the current top 3 in each category, and the main sponsor logo. Auto-refreshes to stay current. Big fonts, high contrast, optimized for projection or large screens.

## What to build

### 1. New store query — `store/runs.go`

Add a function to get the most recently judged run with full context:

```go
// LatestRunDetail holds the most recent run with athlete and category context.
type LatestRunDetail struct {
    RunID          int
    EntryID        int
    RunNumber      int
    RawTimeMs      int
    PenaltyTouches int
    PenaltyMisses  int
    PenaltySeconds int
    TotalTimeMs    int
    Status         string
    JudgedAt       string
    AthleteID      int
    AthleteName    string
    AthleteClub    string
    AthleteNation  string
    AthleteBio     string
    AthletePhoto   string
    BibNumber      int
    CategoryCode   string
    CategoryName   string
    CategoryID     int
}

// GetLatestRun returns the most recently judged run for an event, with full athlete context.
func GetLatestRun(db *sql.DB, eventID int) (LatestRunDetail, error) {
    var d LatestRunDetail
    err := db.QueryRow(`
        SELECT r.id, r.entry_id, r.run_number, r.raw_time_ms,
               r.penalty_touches, r.penalty_misses, r.penalty_seconds,
               r.total_time_ms, r.status, r.judged_at,
               a.id, a.name, a.club, a.nation, COALESCE(a.bio, ''), COALESCE(a.photo_url, ''),
               e.bib_number, c.code, c.name, c.id
        FROM runs r
        JOIN entries e ON e.id = r.entry_id
        JOIN athletes a ON a.id = e.athlete_id
        JOIN categories c ON c.id = e.category_id
        WHERE e.event_id = ?
        ORDER BY r.judged_at DESC
        LIMIT 1`,
        eventID,
    ).Scan(&d.RunID, &d.EntryID, &d.RunNumber, &d.RawTimeMs,
        &d.PenaltyTouches, &d.PenaltyMisses, &d.PenaltySeconds,
        &d.TotalTimeMs, &d.Status, &d.JudgedAt,
        &d.AthleteID, &d.AthleteName, &d.AthleteClub, &d.AthleteNation,
        &d.AthleteBio, &d.AthletePhoto,
        &d.BibNumber, &d.CategoryCode, &d.CategoryName, &d.CategoryID)
    return d, err
}
```

### 2. Commentator handler — `handler/public.go`

Add a new handler and data struct:

```go
// Top3Row is a simplified leaderboard row for the commentator's top-3 display.
type Top3Row struct {
    Rank            int
    BibNumber       int
    AthleteName     string
    AthleteNation   string
    BestTotalTimeMs int
}

// CategoryTop3 pairs a category with its top 3 athletes.
type CategoryTop3 struct {
    Category domain.Category
    Top3     []Top3Row
}

// CommentatorPageData is the template data for the commentator view.
type CommentatorPageData struct {
    Event       domain.Event
    LatestRun   *store.LatestRunDetail
    CatTop3     []CategoryTop3
    MainSponsor *domain.Sponsor
    Title       string
}

// CommentatorPage handles GET /events/{slug}/commentator.
func (d *Deps) CommentatorPage(w http.ResponseWriter, r *http.Request) {
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

    // Fetch latest run
    var latestRun *store.LatestRunDetail
    lr, err := store.GetLatestRun(d.DB, event.ID)
    if err == nil {
        latestRun = &lr
    }
    // sql.ErrNoRows is fine — no runs recorded yet

    // Fetch top 3 per category
    cats, err := store.ListCategories(d.DB, event.ID)
    if err != nil {
        log.Printf("Error fetching categories: %v", err)
        d.renderError(w, 500, "Internal server error")
        return
    }

    var catTop3 []CategoryTop3
    for _, cat := range cats {
        rows, err := store.GetLeaderboard(d.DB, cat.ID)
        if err != nil {
            log.Printf("Error fetching leaderboard for category %d: %v", cat.ID, err)
            continue
        }
        var top3 []Top3Row
        for _, row := range rows {
            if row.Rank >= 1 && row.Rank <= 3 {
                top3 = append(top3, Top3Row{
                    Rank:            row.Rank,
                    BibNumber:       row.BibNumber,
                    AthleteName:     row.AthleteName,
                    AthleteNation:   row.AthleteNation,
                    BestTotalTimeMs: row.BestTotalTimeMs,
                })
            }
        }
        catTop3 = append(catTop3, CategoryTop3{
            Category: cat,
            Top3:     top3,
        })
    }

    // Fetch main sponsor
    var mainSponsor *domain.Sponsor
    ms, err := store.GetMainSponsor(d.DB, event.ID)
    if err == nil {
        mainSponsor = &ms
    }

    data := CommentatorPageData{
        Event:       event,
        LatestRun:   latestRun,
        CatTop3:     catTop3,
        MainSponsor: mainSponsor,
        Title:       "Commentator — " + event.Name,
    }

    // Support partial refresh (AJAX)
    if r.URL.Query().Get("partial") == "1" {
        if err := d.Tmpls["commentator_partial"].ExecuteTemplate(w, "commentator-content", data); err != nil {
            log.Printf("Error rendering commentator partial: %v", err)
        }
        return
    }

    if err := d.Tmpls["commentator"].ExecuteTemplate(w, "layout.html", data); err != nil {
        log.Printf("Error rendering commentator page: %v", err)
    }
}
```

### 3. Template — `templates/commentator.html`

Create a new template that wraps layout and includes the partial:

```html
{{define "content"}}
<div class="commentator-page">
    <div class="commentator-header">
        <h1>{{.Event.Name}} <span class="commentator-label">— Commentator View</span></h1>
        <div class="commentator-refresh-bar">
            <span id="commentator-status">Live</span>
            <button id="commentator-toggle" class="btn-refresh-toggle">⏸ Pause</button>
        </div>
    </div>

    <div id="commentator-content" data-slug="{{.Event.Slug}}">
        {{template "commentator-content" .}}
    </div>
</div>
{{end}}
```

### 4. Template — `templates/commentator_partial.html`

Create the partial template for AJAX refresh:

```html
{{define "commentator-content"}}
<div class="commentator-grid">
    {{/* Left column: latest run + athlete info */}}
    <div class="commentator-athlete-panel">
        {{if .LatestRun}}
        <div class="commentator-now-label">Latest Result</div>
        <div class="commentator-athlete-card">
            {{if .LatestRun.AthletePhoto}}
            <img src="{{.LatestRun.AthletePhoto}}" alt="{{.LatestRun.AthleteName}}" class="commentator-athlete-photo">
            {{end}}
            <div class="commentator-athlete-info">
                <h2 class="commentator-athlete-name">
                    <span class="commentator-bib">#{{.LatestRun.BibNumber}}</span>
                    {{.LatestRun.AthleteName}}
                </h2>
                <div class="commentator-athlete-meta">
                    <span>{{.LatestRun.AthleteClub}}</span>
                    <span>{{.LatestRun.AthleteNation}}</span>
                    <span class="commentator-category-badge">{{.LatestRun.CategoryCode}}</span>
                </div>
                {{if .LatestRun.AthleteBio}}
                <p class="commentator-bio">{{.LatestRun.AthleteBio}}</p>
                {{end}}
            </div>
        </div>

        <div class="commentator-run-result">
            <div class="commentator-run-label">Run {{.LatestRun.RunNumber}}</div>
            <div class="commentator-time-display">
                <span class="commentator-total-time">{{formatTime .LatestRun.TotalTimeMs}}</span>
            </div>
            <div class="commentator-time-breakdown">
                <span class="commentator-raw">Raw: {{formatTime .LatestRun.RawTimeMs}}</span>
                {{if gt .LatestRun.PenaltySeconds 0}}
                <span class="commentator-penalties">
                    +{{.LatestRun.PenaltySeconds}}s
                    ({{if gt .LatestRun.PenaltyTouches 0}}{{.LatestRun.PenaltyTouches}} touch{{if gt .LatestRun.PenaltyTouches 1}}es{{end}}{{end}}{{if and (gt .LatestRun.PenaltyTouches 0) (gt .LatestRun.PenaltyMisses 0)}}, {{end}}{{if gt .LatestRun.PenaltyMisses 0}}{{.LatestRun.PenaltyMisses}} miss{{if gt .LatestRun.PenaltyMisses 1}}es{{end}}{{end}})
                </span>
                {{else}}
                <span class="commentator-clean">✅ Clean run!</span>
                {{end}}
            </div>
        </div>
        {{else}}
        <div class="commentator-waiting">
            <h2>⏳ Waiting for first run...</h2>
            <p>The commentator view will update automatically once runs are recorded.</p>
        </div>
        {{end}}
    </div>

    {{/* Right column: top 3 per category */}}
    <div class="commentator-standings-panel">
        <div class="commentator-now-label">Current Standings</div>
        {{range .CatTop3}}
        <div class="commentator-category-standings">
            <h3>{{.Category.Code}} — {{.Category.Name}}</h3>
            {{if .Top3}}
            <table class="commentator-top3-table">
                <tbody>
                    {{range .Top3}}
                    <tr class="commentator-rank-{{.Rank}}">
                        <td class="commentator-rank-col">
                            {{if eq .Rank 1}}🥇{{else if eq .Rank 2}}🥈{{else if eq .Rank 3}}🥉{{end}}
                        </td>
                        <td class="commentator-name-col">
                            <span class="commentator-top3-bib">#{{.BibNumber}}</span>
                            {{.AthleteName}}
                            <span class="commentator-top3-nation">{{.AthleteNation}}</span>
                        </td>
                        <td class="commentator-time-col">{{formatTime .BestTotalTimeMs}}</td>
                    </tr>
                    {{end}}
                </tbody>
            </table>
            {{else}}
            <p class="commentator-no-results">No results yet</p>
            {{end}}
        </div>
        {{end}}

        {{if .MainSponsor}}
        <div class="commentator-sponsor">
            <a href="{{.MainSponsor.WebsiteURL}}" target="_blank" rel="noopener">
                <img src="{{.MainSponsor.LogoURL}}" alt="{{.MainSponsor.Name}}" class="commentator-sponsor-logo">
            </a>
        </div>
        {{end}}
    </div>
</div>
{{end}}
```

### 5. Register route and templates in `main.go`

Add templates:
```go
"commentator":         template.Must(template.New("layout.html").Funcs(funcMap).ParseFiles("templates/layout.html", "templates/commentator.html", "templates/commentator_partial.html")),
"commentator_partial": template.Must(template.New("commentator_partial.html").Funcs(funcMap).ParseFiles("templates/commentator_partial.html")),
```

Add route (public, no auth needed — commentators are at the venue):
```go
mux.HandleFunc("GET /events/{slug}/commentator", deps.CommentatorPage)
```

### 6. Auto-refresh for commentator — `static/app.js`

Add a second auto-refresh block for the commentator page. Refresh every 5 seconds (faster than leaderboard, since commentators need real-time updates):

```javascript
// Commentator view auto-refresh (every 5 seconds)
(function() {
    const container = document.getElementById('commentator-content');
    if (!container) return;

    const slug = container.dataset.slug;
    let paused = false;
    const statusEl = document.getElementById('commentator-status');

    async function refresh() {
        if (paused) return;
        try {
            const resp = await fetch(`/events/${slug}/commentator?partial=1`);
            if (resp.ok) {
                const oldText = container.innerText;
                container.innerHTML = await resp.text();
                const newText = container.innerText;
                if (oldText !== newText) {
                    container.classList.add('commentator-updated');
                    setTimeout(() => container.classList.remove('commentator-updated'), 1500);
                }
                if (statusEl) statusEl.textContent = '🟢 Live — ' + new Date().toLocaleTimeString();
            }
        } catch (e) {
            if (statusEl) statusEl.textContent = '🔴 Connection lost — retrying...';
        }
    }

    setInterval(refresh, 5000);

    const toggleBtn = document.getElementById('commentator-toggle');
    if (toggleBtn) {
        toggleBtn.addEventListener('click', function() {
            paused = !paused;
            toggleBtn.textContent = paused ? '▶ Resume' : '⏸ Pause';
            if (statusEl) statusEl.textContent = paused ? '⏸ Paused' : '🟢 Live';
        });
    }
})();
```

### 7. CSS — `static/style.css`

The commentator view needs a distinct visual style — bigger fonts, higher contrast, optimized for projection. Add these styles:

```css
/* === Commentator View === */
.commentator-page {
    max-width: 1200px;
    margin: 0 auto;
}

.commentator-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 1.5rem;
    flex-wrap: wrap;
    gap: 0.5rem;
}

.commentator-header h1 {
    font-size: 1.5rem;
}

.commentator-label {
    font-weight: 400;
    color: #6b7280;
    font-size: 1rem;
}

.commentator-refresh-bar {
    display: flex;
    align-items: center;
    gap: 0.75rem;
}

#commentator-status {
    font-size: 0.85rem;
    color: #6b7280;
}

.commentator-grid {
    display: grid;
    grid-template-columns: 1.2fr 0.8fr;
    gap: 2rem;
    align-items: start;
}

/* Left panel — athlete info */
.commentator-now-label {
    text-transform: uppercase;
    font-size: 0.75rem;
    font-weight: 700;
    letter-spacing: 0.1em;
    color: #9ca3af;
    margin-bottom: 0.75rem;
}

.commentator-athlete-card {
    display: flex;
    gap: 1.5rem;
    align-items: flex-start;
    margin-bottom: 1.5rem;
}

.commentator-athlete-photo {
    width: 120px;
    height: 120px;
    border-radius: 10px;
    object-fit: cover;
    flex-shrink: 0;
}

.commentator-athlete-name {
    font-size: 1.8rem;
    margin-bottom: 0.25rem;
}

.commentator-bib {
    color: #6b7280;
    font-weight: 600;
    font-size: 1.2rem;
    margin-right: 0.25rem;
}

.commentator-athlete-meta {
    display: flex;
    gap: 1rem;
    font-size: 1rem;
    color: #4b5563;
    margin-bottom: 0.5rem;
    flex-wrap: wrap;
}

.commentator-category-badge {
    background: #1e3a5f;
    color: #fff;
    padding: 0.15rem 0.5rem;
    border-radius: 4px;
    font-size: 0.8rem;
    font-weight: 700;
}

.commentator-bio {
    font-size: 1.05rem;
    color: #4b5563;
    line-height: 1.5;
    font-style: italic;
    margin-top: 0.5rem;
}

/* Run result display */
.commentator-run-result {
    background: #f9fafb;
    border: 2px solid #e5e7eb;
    border-radius: 10px;
    padding: 1.5rem;
    text-align: center;
}

.commentator-run-label {
    font-size: 0.85rem;
    text-transform: uppercase;
    font-weight: 700;
    letter-spacing: 0.1em;
    color: #6b7280;
    margin-bottom: 0.5rem;
}

.commentator-time-display {
    margin-bottom: 0.5rem;
}

.commentator-total-time {
    font-size: 3.5rem;
    font-weight: 800;
    color: #1e3a5f;
    font-variant-numeric: tabular-nums;
    letter-spacing: -0.02em;
}

.commentator-time-breakdown {
    display: flex;
    justify-content: center;
    gap: 1.5rem;
    font-size: 1rem;
    color: #6b7280;
}

.commentator-penalties {
    color: #dc2626;
    font-weight: 600;
}

.commentator-clean {
    color: #059669;
    font-weight: 600;
}

.commentator-raw {
    font-variant-numeric: tabular-nums;
}

/* Waiting state */
.commentator-waiting {
    text-align: center;
    padding: 3rem 1rem;
    color: #6b7280;
}
.commentator-waiting h2 {
    font-size: 1.5rem;
    margin-bottom: 0.5rem;
}

/* Right panel — standings */
.commentator-standings-panel {
    position: sticky;
    top: 5rem;
}

.commentator-category-standings {
    margin-bottom: 1.5rem;
}

.commentator-category-standings h3 {
    font-size: 1rem;
    color: #1e3a5f;
    margin-bottom: 0.5rem;
    padding-bottom: 0.25rem;
    border-bottom: 2px solid #e5e7eb;
}

.commentator-top3-table {
    width: 100%;
    border-collapse: collapse;
}

.commentator-top3-table tr {
    border-bottom: 1px solid #f3f4f6;
}

.commentator-rank-col {
    width: 2rem;
    font-size: 1.3rem;
    text-align: center;
    padding: 0.5rem 0.25rem;
}

.commentator-name-col {
    padding: 0.5rem 0.5rem;
    font-size: 1.05rem;
    font-weight: 600;
}

.commentator-top3-bib {
    color: #9ca3af;
    font-weight: 600;
    font-size: 0.85rem;
    margin-right: 0.25rem;
}

.commentator-top3-nation {
    font-weight: 400;
    color: #6b7280;
    font-size: 0.85rem;
    margin-left: 0.25rem;
}

.commentator-time-col {
    text-align: right;
    font-variant-numeric: tabular-nums;
    font-weight: 600;
    padding: 0.5rem 0.25rem;
    font-size: 1.05rem;
    color: #1e3a5f;
}

.commentator-no-results {
    font-size: 0.9rem;
    color: #9ca3af;
    font-style: italic;
}

/* Sponsor corner */
.commentator-sponsor {
    margin-top: 2rem;
    text-align: center;
    padding-top: 1rem;
    border-top: 1px solid #e5e7eb;
}
.commentator-sponsor-logo {
    max-height: 40px;
    max-width: 180px;
    opacity: 0.6;
}
.commentator-sponsor:hover .commentator-sponsor-logo {
    opacity: 1;
}

/* Update flash animation */
.commentator-updated {
    animation: commentator-flash 1.5s ease-out;
}
@keyframes commentator-flash {
    0% { background-color: #eff6ff; }
    100% { background-color: transparent; }
}

/* Mobile: single column */
@media (max-width: 768px) {
    .commentator-grid {
        grid-template-columns: 1fr;
    }
    .commentator-total-time {
        font-size: 2.5rem;
    }
    .commentator-athlete-name {
        font-size: 1.3rem;
    }
    .commentator-athlete-card {
        flex-direction: column;
        align-items: center;
        text-align: center;
    }
    .commentator-athlete-meta {
        justify-content: center;
    }
    .commentator-standings-panel {
        position: static;
    }
}
```

## Verification

1. `go build ./...` — no errors.
2. Delete `data.db` and re-seed: `Remove-Item -Force data.db; go run main.go -seed`.
3. Open `http://localhost:8080/events/demo-slalom-2026/commentator` — see the commentator view with:
   - Left side: most recently judged athlete with their photo (if set), name, bib, club, nation, category badge, bio, and their latest run result displayed in large font (total time, raw time, penalty breakdown).
   - Right side: top 3 standings per category with medal emoji, names, nations, and times.
   - Main sponsor logo at bottom-right.
4. The run result shows the time in large (3.5rem) bold font — easily readable from across a room.
5. Record a new run via the judge panel → the commentator view auto-refreshes within 5 seconds to show the new athlete and result.
6. Pause/Resume button works for the auto-refresh.
7. If no runs recorded yet, shows "Waiting for first run..." message.
8. Mobile view: single-column layout, athlete photo stacks above name.
9. The page works without auth (public route — commentators at the venue just need the URL).
10. The `?partial=1` param returns just the inner content (no layout wrapper).

## Files to create/modify

```
templates/commentator.html           (create — commentator page wrapper)
templates/commentator_partial.html   (create — commentator inner content partial)
store/runs.go                        (modify — add GetLatestRun query)
handler/public.go                    (modify — add CommentatorPage handler, Top3Row, CategoryTop3, CommentatorPageData)
main.go                              (modify — add commentator templates, add route)
static/app.js                        (modify — add commentator auto-refresh block)
static/style.css                     (modify — add commentator CSS)
```

## Important notes

- The commentator view is deliberately NOT added to the tab bar on the event/leaderboard/gallery pages. It's a separate page for internal use. Commentators get the direct URL.
- Don't add it to the main nav either. It's accessed via direct URL: `/events/{slug}/commentator`.
- The auto-refresh interval is 5 seconds (faster than the leaderboard's 10 seconds) because the commentator needs near-real-time updates.
- The `GetLatestRun` query is simple — it just gets the most recent run by `judged_at DESC`. If two runs are recorded simultaneously, one will show first. That's fine.
- The top-3 per category reuses the existing `GetLeaderboard` query and filters to rank ≤ 3 in Go. This is slightly inefficient (fetching all rows to show 3), but with 6–10 athletes per category it's negligible.
- The large time display (`3.5rem`) and high-contrast design are intentional — this page is meant to be projected or displayed on a large screen. Think sports arena scoreboard aesthetic.
- Consider adding `font-variant-numeric: tabular-nums` to all time displays so digits align vertically and don't shift when values change.
