# Prompt 12 — Basic Photo Gallery

## Context

You are working on **canoe-slalom-live**, a Go web app for canoe/water slalom competitions. Stack: Go `net/http` (Go 1.22+), `html/template`, SQLite via `github.com/mattn/go-sqlite3`.

Read `PLAN.md` (Phase 3 — Photo gallery). Previous Phase 3 prompts:
- **Prompt 11**: Sponsors schema, seed data, display on event page and leaderboard.

**Current codebase state after Prompt 11:**
The app now has a `sponsors` table, `domain/sponsor.go` struct, `store/sponsors.go` queries, and sponsor display in event and leaderboard templates. All existing Phase 1–2 features remain intact.

**Current codebase structure:**
```
main.go                          — entry point, template parsing, route registration
db/db.go                         — Open() and Seed() with go:embed
db/migrations.sql                — CREATE TABLE for events, categories, athletes, entries, runs, sponsors
db/seed.sql                      — demo data
domain/event.go                  — Event, Category structs
domain/athlete.go                — Athlete, Entry structs
domain/run.go                    — Run struct with formatting helpers
domain/sponsor.go                — Sponsor struct
store/events.go                  — GetEventBySlug(), ListCategories()
store/athletes.go                — GetAthlete(), ListEntriesByCategory(), GetEntryByEventAndAthlete()
store/runs.go                    — CreateRun(), GetLeaderboard(), etc.
store/sponsors.go                — ListSponsorsByEvent(), GetMainSponsor()
handler/public.go                — Deps struct, EventPage, AthletePage, LeaderboardPage
handler/judge.go                 — JudgePage, SubmitRun, EditRunPage, UpdateRunHandler, DeleteRunHandler
handler/auth.go                  — RequireAuth middleware, SessionStore
handler/helpers.go               — renderError()
handler/logging.go               — LoggingMiddleware, SecurityHeaders
templates/layout.html            — shared shell with nav + footer
templates/event.html             — event page with start list + sponsors section
templates/athlete.html           — athlete profile
templates/leaderboard.html       — leaderboard page
templates/leaderboard_partial.html — leaderboard partial
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
GET  /judge/events/{slug}                          → deps.RequireAuth(deps.JudgePage)
POST /judge/events/{slug}/runs                     → deps.RequireAuth(deps.SubmitRun)
GET  /judge/events/{slug}/runs/{id}/edit           → deps.RequireAuth(deps.EditRunPage)
POST /judge/events/{slug}/runs/{id}                → deps.RequireAuth(deps.UpdateRunHandler)
POST /judge/events/{slug}/runs/{id}/delete         → deps.RequireAuth(deps.DeleteRunHandler)
GET  /static/                                      → file server
```

**Current nav in `templates/layout.html`:**
```html
<nav class="site-nav">
    <a href="/events/demo-slalom-2026">Event</a>
    <a href="/events/demo-slalom-2026/leaderboard">Leaderboard</a>
    <a href="/judge/events/demo-slalom-2026" class="nav-judge">🏁 Judge</a>
</nav>
```

## Goal

Add a `photos` table, seed demo photos, build a public gallery page at `GET /events/{slug}/photos`, and show recent photos on the athlete profile page. No file upload — just URL-based photo references (external image URLs or placeholder images).

## What to build

### 1. Domain struct — `domain/photo.go`

Create a new file `domain/photo.go`:

```go
package domain

type Photo struct {
    ID               int
    EventID          int
    AthleteID        int    // 0 if not linked to a specific athlete
    ImageURL         string
    Caption          string
    PhotographerName string
    CreatedAt        string
}
```

### 2. Database migration — `db/migrations.sql`

Add at the end of `migrations.sql`:

```sql
CREATE TABLE IF NOT EXISTS photos (
    id                INTEGER PRIMARY KEY,
    event_id          INTEGER REFERENCES events(id),
    athlete_id        INTEGER REFERENCES athletes(id),
    image_url         TEXT NOT NULL,
    caption           TEXT,
    photographer_name TEXT,
    created_at        TEXT
);
```

`athlete_id` is nullable — general event photos (crowd shots, course overview) don't link to an athlete.

### 3. Seed photo data — `db/seed.sql`

Add 8–10 seed photos at the end of `seed.sql`. Use `https://placehold.co/` service with descriptive text for demo purposes. Mix athlete-linked and general event photos:

```sql
-- Photos for Demo Slalom 2026
-- Athlete action shots
INSERT OR IGNORE INTO photos (id, event_id, athlete_id, image_url, caption, photographer_name, created_at)
VALUES (1, 1, 1, 'https://placehold.co/800x600/1e3a5f/ffffff?text=Jan+Rohan+Run+1', 'Jan Rohan navigating gate 12 on Run 1', 'Pavel Novák', '2026-06-15T10:06:00Z');

INSERT OR IGNORE INTO photos (id, event_id, athlete_id, image_url, caption, photographer_name, created_at)
VALUES (2, 1, 2, 'https://placehold.co/800x600/2563eb/ffffff?text=Oliver+Bennett+Start', 'Oliver Bennett at the start gate', 'James Wilson', '2026-06-15T10:09:00Z');

INSERT OR IGNORE INTO photos (id, event_id, athlete_id, image_url, caption, photographer_name, created_at)
VALUES (3, 1, 7, 'https://placehold.co/800x600/059669/ffffff?text=Elena+Martinez+Finish', 'Elena Martínez crossing the finish line', 'Carlos López', '2026-06-15T11:06:00Z');

INSERT OR IGNORE INTO photos (id, event_id, athlete_id, image_url, caption, photographer_name, created_at)
VALUES (4, 1, 3, 'https://placehold.co/800x600/7c3aed/ffffff?text=Mathieu+Deschamps', 'Mathieu Deschamps battling upstream gate 7', 'Pavel Novák', '2026-06-15T10:12:00Z');

INSERT OR IGNORE INTO photos (id, event_id, athlete_id, image_url, caption, photographer_name, created_at)
VALUES (5, 1, 10, 'https://placehold.co/800x600/dc2626/ffffff?text=Anna+Brezinova', 'Anna Březinová — home favourite gets the crowd roaring', 'Tereza Dvořáková', '2026-06-15T11:15:00Z');

-- General event photos (no athlete)
INSERT OR IGNORE INTO photos (id, event_id, athlete_id, image_url, caption, photographer_name, created_at)
VALUES (6, 1, NULL, 'https://placehold.co/800x600/f59e0b/1a1a1a?text=Troja+Course+Overview', 'Troja Whitewater Course — morning setup', 'Pavel Novák', '2026-06-15T08:30:00Z');

INSERT OR IGNORE INTO photos (id, event_id, athlete_id, image_url, caption, photographer_name, created_at)
VALUES (7, 1, NULL, 'https://placehold.co/800x600/10b981/ffffff?text=Finish+Area+Crowd', 'Spectators at the finish area', 'James Wilson', '2026-06-15T12:00:00Z');

INSERT OR IGNORE INTO photos (id, event_id, athlete_id, image_url, caption, photographer_name, created_at)
VALUES (8, 1, NULL, 'https://placehold.co/800x600/6366f1/ffffff?text=Award+Ceremony', 'K1M award ceremony', 'Tereza Dvořáková', '2026-06-15T17:00:00Z');
```

### 4. Store queries — `store/photos.go`

Create a new file `store/photos.go`:

```go
package store

import (
    "database/sql"
    "canoe-slalom-live/domain"
)

// PhotoWithAthlete pairs a photo with the athlete's name (if linked).
type PhotoWithAthlete struct {
    domain.Photo
    AthleteName string // empty if no athlete linked
}

// ListPhotosByEvent returns all photos for an event, ordered by created_at DESC.
func ListPhotosByEvent(db *sql.DB, eventID int) ([]PhotoWithAthlete, error) {
    rows, err := db.Query(`
        SELECT p.id, p.event_id, COALESCE(p.athlete_id, 0), p.image_url, p.caption,
               p.photographer_name, p.created_at, COALESCE(a.name, '')
        FROM photos p
        LEFT JOIN athletes a ON a.id = p.athlete_id
        WHERE p.event_id = ?
        ORDER BY p.created_at DESC`,
        eventID,
    )
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var photos []PhotoWithAthlete
    for rows.Next() {
        var pa PhotoWithAthlete
        if err := rows.Scan(&pa.ID, &pa.EventID, &pa.AthleteID,
            &pa.ImageURL, &pa.Caption, &pa.PhotographerName,
            &pa.CreatedAt, &pa.AthleteName); err != nil {
            return nil, err
        }
        photos = append(photos, pa)
    }
    return photos, rows.Err()
}

// ListPhotosByAthlete returns photos for a specific athlete in an event.
func ListPhotosByAthlete(db *sql.DB, eventID, athleteID int) ([]domain.Photo, error) {
    rows, err := db.Query(`
        SELECT id, event_id, athlete_id, image_url, caption, photographer_name, created_at
        FROM photos
        WHERE event_id = ? AND athlete_id = ?
        ORDER BY created_at DESC`,
        eventID, athleteID,
    )
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var photos []domain.Photo
    for rows.Next() {
        var p domain.Photo
        if err := rows.Scan(&p.ID, &p.EventID, &p.AthleteID,
            &p.ImageURL, &p.Caption, &p.PhotographerName, &p.CreatedAt); err != nil {
            return nil, err
        }
        photos = append(photos, p)
    }
    return photos, rows.Err()
}
```

### 5. Gallery handler — `handler/public.go`

Add a new `GalleryPage` handler and data struct:

```go
// GalleryPageData is the template data for the photo gallery page.
type GalleryPageData struct {
    Event  domain.Event
    Photos []store.PhotoWithAthlete
    Title  string
}

// GalleryPage handles GET /events/{slug}/photos.
func (d *Deps) GalleryPage(w http.ResponseWriter, r *http.Request) {
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

    photos, err := store.ListPhotosByEvent(d.DB, event.ID)
    if err != nil {
        log.Printf("Error fetching photos: %v", err)
        d.renderError(w, 500, "Internal server error")
        return
    }

    data := GalleryPageData{
        Event:  event,
        Photos: photos,
        Title:  "Photos — " + event.Name + " — Canoe Slalom Live",
    }

    if err := d.Tmpls["gallery"].ExecuteTemplate(w, "layout.html", data); err != nil {
        log.Printf("Error rendering gallery page: %v", err)
    }
}
```

### 6. Update `AthletePage` handler

After fetching runs, also fetch athlete photos:

```go
// After fetching runs...
photos, err := store.ListPhotosByAthlete(d.DB, event.ID, athleteID)
if err != nil {
    log.Printf("Error fetching athlete photos: %v", err)
    // Don't fail page — just leave photos empty
}
```

Add `Photos []domain.Photo` field to `AthletePageData`:

```go
type AthletePageData struct {
    Event   domain.Event
    Athlete domain.Athlete
    Entry   store.EntryWithAthlete
    Runs    []domain.Run
    Photos  []domain.Photo
    Title   string
}
```

### 7. Template — `templates/gallery.html`

Create a new template file `templates/gallery.html`:

```html
{{define "content"}}
<div class="event-header">
    <h1>{{.Event.Name}}</h1>
    <div class="event-meta">
        <span class="event-date">📅 {{.Event.Date}}</span>
        <span class="event-location">📍 {{.Event.Location}}</span>
    </div>
</div>

<div class="tab-bar">
    <a href="/events/{{.Event.Slug}}" class="tab">Start List</a>
    <a href="/events/{{.Event.Slug}}/leaderboard" class="tab">Leaderboard</a>
    <a href="/events/{{.Event.Slug}}/photos" class="tab active">📸 Photos</a>
</div>

{{if .Photos}}
<div class="photo-gallery">
    {{range .Photos}}
    <div class="photo-card">
        <a href="{{.ImageURL}}" target="_blank" rel="noopener" class="photo-link">
            <img src="{{.ImageURL}}" alt="{{.Caption}}" loading="lazy" class="photo-img">
        </a>
        <div class="photo-info">
            <p class="photo-caption">{{.Caption}}</p>
            <div class="photo-meta">
                {{if .AthleteName}}<span class="photo-athlete">🏅 {{.AthleteName}}</span>{{end}}
                {{if .PhotographerName}}<span class="photo-credit">📷 {{.PhotographerName}}</span>{{end}}
            </div>
        </div>
    </div>
    {{end}}
</div>
{{else}}
<p class="empty-state">No photos yet — check back during the competition!</p>
{{end}}
{{end}}
```

### 8. Update athlete profile template — `templates/athlete.html`

Add a photos section below the runs table (before the back link):

```html
{{if .Photos}}
<section class="athlete-photos-section">
    <h2>📸 Photos</h2>
    <div class="athlete-photo-grid">
        {{range .Photos}}
        <a href="{{.ImageURL}}" target="_blank" rel="noopener" class="athlete-photo-thumb">
            <img src="{{.ImageURL}}" alt="{{.Caption}}" loading="lazy" title="{{.Caption}} — 📷 {{.PhotographerName}}">
        </a>
        {{end}}
    </div>
</section>
{{end}}
```

### 9. Update tab bars in other templates

Add the Photos tab to the existing tab bars in `templates/event.html` and `templates/leaderboard.html`:

**In `event.html` tab bar (after Leaderboard tab):**
```html
<a href="/events/{{.Event.Slug}}/photos" class="tab">📸 Photos</a>
```

**In `leaderboard.html` tab bar (after Leaderboard tab):**
```html
<a href="/events/{{.Event.Slug}}/photos" class="tab">📸 Photos</a>
```

### 10. Update navigation — `templates/layout.html`

Add a Photos link to the site nav:

```html
<nav class="site-nav">
    <a href="/events/demo-slalom-2026">Event</a>
    <a href="/events/demo-slalom-2026/leaderboard">Leaderboard</a>
    <a href="/events/demo-slalom-2026/photos">📸 Photos</a>
    <a href="/judge/events/demo-slalom-2026" class="nav-judge">🏁 Judge</a>
</nav>
```

### 11. Register route and template in `main.go`

Add the template:
```go
"gallery": template.Must(template.New("layout.html").Funcs(funcMap).ParseFiles("templates/layout.html", "templates/gallery.html")),
```

Add the route:
```go
mux.HandleFunc("GET /events/{slug}/photos", deps.GalleryPage)
```

### 12. CSS — `static/style.css`

Add photo gallery styling:

```css
/* === Photo Gallery === */
.photo-gallery {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
    gap: 1.5rem;
    margin-top: 1rem;
}

.photo-card {
    border: 1px solid #e5e7eb;
    border-radius: 8px;
    overflow: hidden;
    background: #fff;
    transition: box-shadow 0.2s;
}
.photo-card:hover {
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
}

.photo-link {
    display: block;
}

.photo-img {
    width: 100%;
    height: 200px;
    object-fit: cover;
    display: block;
}

.photo-info {
    padding: 0.75rem;
}

.photo-caption {
    font-size: 0.9rem;
    color: #1a1a1a;
    margin-bottom: 0.5rem;
    line-height: 1.4;
}

.photo-meta {
    display: flex;
    flex-wrap: wrap;
    gap: 0.75rem;
    font-size: 0.8rem;
    color: #6b7280;
}

.photo-athlete {
    font-weight: 600;
    color: #2563eb;
}

/* Athlete profile photo thumbnails */
.athlete-photos-section {
    margin-top: 2rem;
}
.athlete-photos-section h2 {
    margin-bottom: 0.75rem;
}

.athlete-photo-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(150px, 1fr));
    gap: 0.75rem;
}

.athlete-photo-thumb {
    display: block;
    border-radius: 6px;
    overflow: hidden;
}

.athlete-photo-thumb img {
    width: 100%;
    height: 120px;
    object-fit: cover;
    display: block;
    transition: transform 0.2s;
}
.athlete-photo-thumb:hover img {
    transform: scale(1.05);
}

/* Mobile: single column gallery */
@media (max-width: 500px) {
    .photo-gallery {
        grid-template-columns: 1fr;
    }
    .athlete-photo-grid {
        grid-template-columns: repeat(auto-fill, minmax(120px, 1fr));
    }
    .photo-img {
        height: 180px;
    }
}
```

## Verification

1. `go build ./...` — no errors.
2. Delete `data.db` and re-seed: `Remove-Item -Force data.db; go run main.go -seed`.
3. Open `http://localhost:8080/events/demo-slalom-2026/photos` — see a grid of 8 photo cards with captions, athlete names, and photographer credits.
4. Click a photo — opens the full-size image in a new tab.
5. Photos linked to athletes show the athlete name badge. General event photos don't.
6. Navigate between tabs: Start List / Leaderboard / Photos — consistent tab bar across all three pages. Active tab is highlighted.
7. Open an athlete profile (e.g., Jan Rohan) — see a "Photos" section with thumbnail grid of their action shots.
8. Layout nav includes "📸 Photos" link.
9. Mobile view: photos stack in single column, thumbnails shrink but remain usable.
10. Gallery with no photos (e.g., a fresh event) — shows "No photos yet" empty state.

## Files to create/modify

```
domain/photo.go              (create — Photo struct)
store/photos.go              (create — ListPhotosByEvent, ListPhotosByAthlete)
templates/gallery.html       (create — photo gallery page)
db/migrations.sql            (modify — add photos CREATE TABLE)
db/seed.sql                  (modify — add photo seed data)
handler/public.go            (modify — add GalleryPage handler, add Photos to AthletePageData)
templates/event.html         (modify — add Photos tab)
templates/leaderboard.html   (modify — add Photos tab)
templates/athlete.html       (modify — add photos section)
templates/layout.html        (modify — add Photos nav link)
main.go                      (modify — add gallery template and route)
static/style.css             (modify — add gallery CSS)
```

## Important notes

- No file upload! Photos are external image URLs stored in the database. For a real deployment, organizers would paste URLs from their cloud storage or social media.
- The `https://placehold.co/` placeholder images are fine for the demo. They render instantly and clearly show what each "photo" represents.
- `athlete_id` is nullable (use `NULL` in SQL, `0` in Go). The Go struct uses `int` with 0 meaning "no athlete linked" — keep it simple.
- The gallery is purely public (no auth required). Anyone can view photos.
- Don't add a photo upload or admin form here. If needed later, that's a separate prompt. For now, photos are seeded.
- Keep the gallery simple — no lightbox, no carousel. Just a CSS grid with cards that link to full-size images. A lightbox is a nice-to-have for a future prompt.
