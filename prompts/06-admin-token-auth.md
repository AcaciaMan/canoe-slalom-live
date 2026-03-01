# Prompt 06 — Admin Token Authentication Middleware

## Context

You are working on **canoe-slalom-live**, a Go web app for canoe/water slalom competitions. Stack: Go `net/http` (Go 1.22+ with pattern routing), `html/template`, SQLite via `github.com/mattn/go-sqlite3`.

Read `PLAN.md` for full architecture. Phase 1 is complete. The app has:

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
store/runs.go                    — CreateRun(), ListRunsByEntry(), GetEntryByID(), GetLeaderboard(), LeaderboardRow, RunResult structs
handler/public.go                — Deps struct {DB, Tmpls}, EventPage, AthletePage, LeaderboardPage, plus data structs
handler/judge.go                 — JudgePage, SubmitRun with form validation
handler/helpers.go               — renderError() helper
templates/layout.html            — shared shell with nav (logo, Event, Leaderboard, Judge links)
templates/event.html             — public event page with start list (tab bar: Start List active / Leaderboard)
templates/athlete.html           — athlete profile with bio, runs table
templates/leaderboard.html       — leaderboard with auto-refresh, tab bar (Start List / Leaderboard active)
templates/leaderboard_partial.html — leaderboard tables partial for AJAX refresh
templates/judge_run.html         — judge form: category radio → athlete radio → run number → raw time → stepper penalties → submit
templates/error.html             — styled error page
static/style.css                 — full stylesheet (557 lines)
static/app.js                    — auto-refresh leaderboard (every 15s, pause/resume)
```

**Current routes in main.go:**
```go
GET  /{$}                           → redirect to /events/demo-slalom-2026
GET  /events/{slug}                 → deps.EventPage
GET  /events/{slug}/leaderboard     → deps.LeaderboardPage
GET  /events/{slug}/athletes/{id}   → deps.AthletePage
GET  /judge/events/{slug}           → deps.JudgePage
POST /judge/events/{slug}/runs      → deps.SubmitRun
GET  /static/                       → file server
```

**Key handlers struct:**
```go
type Deps struct {
    DB    *sql.DB
    Tmpls map[string]*template.Template
}
```

There is currently NO authentication. Judge and admin pages are open to anyone.

## Goal

Add a simple admin token system so that `/judge/*` and `/admin/*` routes are protected. A single shared token is set via `ADMIN_TOKEN` environment variable. Users with the token get a session cookie so they don't need to keep passing it in the URL.

## What to build

### 1. Auth middleware — `handler/auth.go`

Create a new file `handler/auth.go`. This handles token validation and cookie session management.

**Approach:**
- Read `ADMIN_TOKEN` from environment at startup, store it in the `Deps` struct.
- Create a middleware function that wraps `http.HandlerFunc`.
- On each protected request, check (in order):
  1. Does the request have a cookie named `admin_session` with value matching a stored session token? If yes → allow.
  2. Does the URL query param `?token=` match `ADMIN_TOKEN`? If yes → set a secure cookie `admin_session` with a randomly generated session ID, store it in a session map, then redirect to the same URL with `?token=` stripped (so it doesn't linger in browser history/bookmarks). Allow.
  3. Neither → return 403 with a styled error page: "Access Denied — Please use an authorized link."

**Add to `Deps` struct:**
```go
type Deps struct {
    DB         *sql.DB
    Tmpls      map[string]*template.Template
    AdminToken string
    Sessions   map[string]bool  // session ID → valid
}
```

**Session generation:** Use `crypto/rand` to generate 32 random bytes, hex-encode them. This is the session cookie value. Store it in `d.Sessions` map.

**Cookie settings:**
- Name: `admin_session`
- Path: `/`
- HttpOnly: `true`
- SameSite: `Lax`
- MaxAge: 86400 (24 hours)
- Secure: `false` for localhost development (would be `true` in production)

**Middleware function signature:**
```go
func (d *Deps) RequireAuth(next http.HandlerFunc) http.HandlerFunc
```

Usage in route registration:
```go
mux.HandleFunc("GET /judge/events/{slug}", deps.RequireAuth(deps.JudgePage))
mux.HandleFunc("POST /judge/events/{slug}/runs", deps.RequireAuth(deps.SubmitRun))
```

**Edge cases:**
- If `ADMIN_TOKEN` env var is empty or not set, log a warning at startup but still start. In that case, disable auth (allow all requests through) — this lets the developer run locally without setting env vars. Log: `"WARNING: ADMIN_TOKEN not set, auth disabled for judge/admin routes"`.
- If someone hits a protected route without token and without cookie, return the 403 page immediately — don't redirect to a login form (no login form in MVP).

### 2. Update `Deps` initialization in `main.go`

```go
adminToken := os.Getenv("ADMIN_TOKEN")
if adminToken == "" {
    log.Println("WARNING: ADMIN_TOKEN not set, auth disabled for judge/admin routes")
}

deps := &handler.Deps{
    DB:         database,
    Tmpls:      tmpls,
    AdminToken: adminToken,
    Sessions:   make(map[string]bool),
}
```

### 3. Wrap judge routes with auth in `main.go`

Change:
```go
mux.HandleFunc("GET /judge/events/{slug}", deps.JudgePage)
mux.HandleFunc("POST /judge/events/{slug}/runs", deps.SubmitRun)
```
To:
```go
mux.HandleFunc("GET /judge/events/{slug}", deps.RequireAuth(deps.JudgePage))
mux.HandleFunc("POST /judge/events/{slug}/runs", deps.RequireAuth(deps.SubmitRun))
```

### 4. Update startup log messages in `main.go`

After the server starts, also log the authenticated judge URL when a token is set:
```go
if adminToken != "" {
    log.Printf("Judge panel (with token): http://localhost:%s/judge/events/demo-slalom-2026?token=%s", port, adminToken)
} else {
    log.Printf("Judge panel (no auth): http://localhost:%s/judge/events/demo-slalom-2026", port)
}
```

### 5. Add an "Access Denied" error template (reuse error.html)

The existing `templates/error.html` and `renderError()` function already support custom error codes and messages. Use them for the 403:
```go
d.renderError(w, 403, "Access Denied — Please use an authorized link to access the judge panel.")
```

No new template needed.

### 6. Update navigation — conditionally show judge link

In `templates/layout.html`, the judge nav link currently shows to all visitors. This is fine for now — clicking it without auth will show the 403 page, which tells them they need an authorized link. No changes needed to the nav for MVP.

However, update the judge nav link comment if desired to clarify it's protected.

## Verification

1. `go build ./...` — no errors.
2. Set env var and start: `set ADMIN_TOKEN=secret123` then `go run main.go`.
3. Open `http://localhost:8080/judge/events/demo-slalom-2026` — see 403 "Access Denied" page.
4. Open `http://localhost:8080/judge/events/demo-slalom-2026?token=secret123` — judge page loads. URL redirects to strip `?token=` param. An `admin_session` cookie is set.
5. Close and reopen the judge page URL (without `?token=`) — still works (cookie auth).
6. Open in a different browser / incognito (no cookie) — 403 again without token.
7. Clear `ADMIN_TOKEN` env var, restart server — warning logged, judge page accessible without auth.
8. Public pages (`/events/...`, `/events/.../leaderboard`, `/events/.../athletes/...`) — always accessible, no auth needed.
9. POST to `/judge/events/{slug}/runs` without auth → 403 (can test with curl).

## Files to create/modify

```
handler/auth.go      (create — middleware + session management)
handler/public.go    (modify — add AdminToken, Sessions fields to Deps struct)
main.go              (modify — init AdminToken/Sessions, wrap judge routes, update log)
```

## Important notes

- This is NOT production-grade auth. It's a shared secret token approach for a weekend demo. The session is in-memory (lost on server restart). That's fine.
- No login page, no user accounts, no password hashing. All of that is Phase 4.
- The `Sessions` map is not concurrent-safe. For MVP with 1–3 judges this won't matter. If you want, use `sync.RWMutex` — it's a few extra lines and future-proofs it. Recommended.
- Don't add admin routes in this prompt — those are not yet built. Just protect the judge routes.
- Keep the admin token out of logs in production, but for localhost MVP, printing it to the terminal is helpful.
