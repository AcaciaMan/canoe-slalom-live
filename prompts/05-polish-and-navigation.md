# Prompt 05 — Polish, Navigation, Auto-Refresh, and End-to-End Verification

## Context

You are working on **canoe-slalom-live**, a Go web app for canoe/water slalom competitions. Stack: Go `net/http`, `html/template`, SQLite.

Read `PLAN.md` for full architecture. Previous steps built the complete Phase 1 core loop:
- **Prompt 01**: Database + seed data + domain + store
- **Prompt 02**: HTTP server, event page with start list, athlete profiles, CSS
- **Prompt 03**: Judge run entry form with stepper penalties
- **Prompt 04**: Leaderboard ranking athletes by best total time with penalty display

The app now has: 3 public pages (event/start list, athlete profile, leaderboard), 1 judge page, and 2 POST endpoints (submit run, with redirect).

This prompt **polishes the MVP** into a cohesive, demo-ready state: consistent navigation, auto-refreshing leaderboard, mobile responsiveness, error handling, and a quick smoke test pass.

## Goal

Turn the working-but-rough prototype into something you'd confidently demo to a canoe slalom organizer. No new features — just cohesion, reliability, and visual quality.

## What to build

### 1. Consistent navigation across all pages

Update `templates/layout.html` header nav to include:
- **Logo/title**: "🛶 Canoe Slalom Live" → links to `/`.
- **Nav links** (right-aligned or after title):
  - "Event" → `/events/demo-slalom-2026` (hardcoded slug for MVP is fine; later this becomes a dropdown).
  - "Leaderboard" → `/events/demo-slalom-2026/leaderboard`
  - "Judge" → `/judge/events/demo-slalom-2026` (visually distinct — maybe a different color or with a 🏁 icon, signaling it's a functional/staff link)

On mobile (< 600px), collapse nav links into a simple list that wraps below the title (no hamburger menu needed — just flex-wrap).

### 2. Tab bar consistency

Both the event page and leaderboard page share a tab bar with "Start List" and "Leaderboard" tabs. Ensure:
- The correct tab is visually active (underline/bold/colored) on each page.
- Consider extracting the tab bar into a shared template block or just duplicate it in both templates with the active class toggled. Duplication is fine for 2 templates.

### 3. Auto-refreshing leaderboard via JS

Update `static/app.js` to add leaderboard auto-refresh:

```javascript
// Auto-refresh leaderboard every 15 seconds
(function() {
    const container = document.getElementById('leaderboard-content');
    if (!container) return;  // only run on leaderboard page

    const slug = container.dataset.slug;  // set data-slug on the div in template
    let refreshInterval = 15000;
    let paused = false;

    const statusEl = document.getElementById('refresh-status');

    async function refresh() {
        if (paused) return;
        try {
            const resp = await fetch(`/events/${slug}/leaderboard?partial=1`);
            if (resp.ok) {
                container.innerHTML = await resp.text();
                if (statusEl) statusEl.textContent = 'Updated ' + new Date().toLocaleTimeString();
            }
        } catch (e) {
            if (statusEl) statusEl.textContent = 'Refresh failed — retrying...';
        }
    }

    setInterval(refresh, refreshInterval);

    // Pause/resume toggle
    const toggleBtn = document.getElementById('refresh-toggle');
    if (toggleBtn) {
        toggleBtn.addEventListener('click', function() {
            paused = !paused;
            toggleBtn.textContent = paused ? '▶ Resume' : '⏸ Pause';
            if (statusEl) statusEl.textContent = paused ? 'Auto-refresh paused' : 'Auto-refresh active';
        });
    }
})();
```

Add to `templates/leaderboard.html`:
- `<div id="leaderboard-content" data-slug="{{.Event.Slug}}">` wrapping all leaderboard tables.
- Below the tab bar: a small status line: `<span id="refresh-status">Auto-refresh active</span>` and a button `<button id="refresh-toggle">⏸ Pause</button>`. Style these as small, unobtrusive controls (gray text, small font).

### 4. Mobile-first CSS pass

Review and improve `static/style.css` for real-world mobile use (judge uses a phone, spectators use phones):

**Global:**
- Body font size 16px (prevents iOS zoom on input focus).
- All form inputs: `font-size: 16px` minimum (again, prevents iOS zoom).

**Tables:**
- Wrap each table in a `<div class="table-wrapper">` with `overflow-x: auto` so wide tables scroll horizontally on phones.
- Leaderboard table: on screens < 500px, consider hiding the "Nation" column (use a CSS media query with `display: none`).

**Judge form:**
- On mobile, stack all form sections vertically (they should already be if using flex/block layout).
- Radio cards should fill available width.
- Stepper `+`/`−` buttons minimum 48×48px.
- Submit button full width, 56px height.

**Event page:**
- Start list table: compact on mobile. Bib column 50px. Name wraps.

**Athlete page:**
- Photo floats right on desktop, full width on mobile (above bio text).

### 5. Error page

Create `templates/error.html`:
```
{{define "content"}}
<div class="error-page">
    <h1>{{.Code}}</h1>
    <p>{{.Message}}</p>
    <a href="/">← Back to events</a>
</div>
{{end}}
```

Add a helper to `handler/public.go` (or a shared `handler/helpers.go`):
```go
func (d *Deps) renderError(w http.ResponseWriter, code int, message string) {
    w.WriteHeader(code)
    d.Tmpls["error"].ExecuteTemplate(w, "layout.html", map[string]interface{}{
        "Code":    code,
        "Message": message,
        "Title":   fmt.Sprintf("%d — %s", code, message),
    })
}
```

Use this in all handlers where you currently return 404 or other errors. Replace bare `http.NotFound(w, r)` calls with `d.renderError(w, 404, "Event not found")` etc.

Register template:
```go
"error": template.Must(template.ParseFiles("templates/layout.html", "templates/error.html")),
```

### 6. Graceful server startup and shutdown

Update `main.go`:
- Print the URL at startup: `log.Printf("Server running at http://localhost:%s", port)`.
- Print the judge URL: `log.Printf("Judge panel: http://localhost:%s/judge/events/demo-slalom-2026", port)`.
- Handle `Ctrl+C` gracefully: use `signal.Notify` to catch `os.Interrupt` and call `server.Shutdown(ctx)` with a 5-second timeout. Close the database on shutdown.

### 7. `.gitignore` update

Ensure `.gitignore` includes:
```
data.db
*.exe
```

### 8. README update

Update `README.md` with a concise quick-start:

```markdown
# 🛶 Canoe Slalom Live

Live timing and results for canoe/water slalom competitions.

## Quick Start

```bash
# Prerequisites: Go 1.22+, GCC (for SQLite CGo driver)
go mod tidy
go run main.go -seed

# Opens at http://localhost:8080
# Judge panel at http://localhost:8080/judge/events/demo-slalom-2026
```

## Stack

- Go (net/http, html/template)
- SQLite (embedded)
- Vanilla HTML/CSS/JS — no build step

## Pages

- **Event page**: `/events/{slug}` — start list with athlete bios
- **Leaderboard**: `/events/{slug}/leaderboard` — live rankings with penalty breakdown
- **Judge panel**: `/judge/events/{slug}` — record runs with time + penalties
- **Athlete profile**: `/events/{slug}/athletes/{id}` — bio and run history
```

### 9. End-to-end smoke test checklist

After all changes, manually verify this exact flow:

1. Delete `data.db` if it exists.
2. `go run main.go -seed` — starts clean with seed data.
3. `http://localhost:8080/` → redirects to event page.
4. Event page: see "Demo Slalom 2026", "Troja Whitewater Course, Prague", two category sections, 10 athletes.
5. Click an athlete name → see their bio. If seed runs exist, see run results table.
6. Back to event → click "Leaderboard" tab → see ranked results with penalty colors.
7. Auto-refresh status shows "Auto-refresh active". Wait 15s — "Updated HH:MM:SS" appears.
8. Click "Pause" → text changes to "Auto-refresh paused".
9. Open judge page in another tab → select K1M → pick an athlete without Run 2 → enter `91.50`, 1 touch, 0 misses → submit.
10. Green flash confirms the run.
11. Switch to leaderboard tab → within 15 seconds, new run should appear and rankings should update.
12. Try submitting a duplicate run (same athlete, same run number) → red error message.
13. Open on phone-sized viewport (Chrome DevTools, 375px wide) → all pages usable, no horizontal overflow on main content, judge buttons easily tappable.
14. Open a non-existent event slug → see styled 404 page with back link.

If all 14 checks pass, Phase 1 is complete and demo-ready.

## Files to create/modify

```
templates/layout.html          (modify: nav links)
templates/event.html           (modify: tab bar active state)
templates/leaderboard.html     (modify: auto-refresh wiring, tab bar)
templates/error.html           (create)
static/style.css               (modify: mobile pass, nav styles)
static/app.js                  (modify: auto-refresh logic)
handler/public.go              (modify: renderError helper, error handling)
main.go                        (modify: graceful shutdown, startup logs)
.gitignore                     (modify: add data.db)
README.md                      (rewrite: quick start guide)
```

## Important notes

- This prompt adds NO new features — only polish. Resist scope creep.
- The auto-refresh uses the `?partial=1` endpoint built in Prompt 04. If that wasn't implemented, add it now: the leaderboard handler should return just the `leaderboard_partial.html` content when `?partial=1` is in the query string.
- Keep total JS under 80 lines (auto-refresh + judge stepper + category filter from Prompt 03).
- After this prompt, the app should feel like a real (if simple) product, not a prototype.
