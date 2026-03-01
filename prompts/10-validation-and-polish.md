# Prompt 10 — Input Validation, Error Handling, and Phase 2 Polish

## Context

You are working on **canoe-slalom-live**, a Go web app for canoe/water slalom competitions. Stack: Go `net/http` (Go 1.22+), `html/template`, SQLite.

Read `PLAN.md` (Phase 2 — Safety and validation, Keeping codebase small). Previous Phase 2 prompts:
- **Prompt 06**: Admin token auth middleware (`handler/auth.go`, `RequireAuth` wrapping judge routes).
- **Prompt 07**: Judge UI overhaul (confirmation step, run status indicators, recent runs, category persistence).
- **Prompt 08**: Run edit/delete (edit form, POST delete, `GetRunByID`, `UpdateRun`, `DeleteRun`, `CategoryID` on entries).
- **Prompt 09**: Leaderboard UX (medals, NEW badges, time-behind, simplified penalty display, `penaltyClass` template func, 10s refresh).

**What's still fragile or missing at this point:**
1. Input validation in `SubmitRun` and `UpdateRunHandler` catches basic errors but doesn't cover all edge cases.
2. No protection against unreasonable values (e.g., raw time of 0.01s or 99999s).
3. HTTP error responses could include more helpful detail.
4. Some potential panics if templates fail or database queries return unexpected results.
5. No request logging — hard to debug issues during a live event.
6. The server doesn't serve a favicon, causing noisy 404s in browser console.
7. codebase may have inconsistencies across the prompts — this cleanup pass catches them.

## Goal

Harden the app for a real riverside event. Add sensible input bounds, consistent error handling, request logging, and final polish. After this prompt, the app should be robust enough that a non-technical judge can't break it by entering bad data.

## What to build

### 1. Input validation hardening — `handler/judge.go`

**In `SubmitRun` and `UpdateRunHandler`, add bounds validation:**

```go
// Raw time bounds: must be between 30.00 and 999.99 seconds
// (A canoe slalom run is typically 80-130 seconds; 30s absolute floor for extreme sprints,
//  999.99s ceiling catches obviously wrong values)
if rawTime < 30.0 || rawTime > 999.99 {
    redirectWithError("Raw time must be between 30.00 and 999.99 seconds")
    return
}

// Penalty bounds: touches 0-50 (a course has ~25 gates, max 2 touches per gate is impossible but generous)
if touches > 50 {
    redirectWithError("Gate touches seems too high (max 50)")
    return
}

// Misses 0-25 (max number of gates)
if misses > 25 {
    redirectWithError("Missed gates seems too high (max 25)")
    return
}
```

**Trim whitespace on all text inputs** before parsing:
```go
rawTimeStr = strings.TrimSpace(r.FormValue("raw_time"))
```
This is already done for `raw_time`, but ensure `touches` and `misses` form values are also trimmed.

**Handle empty form values explicitly** — currently, `strconv.Atoi("")` returns an error, which maps to a generic message. Add specific messages:
```go
if r.FormValue("entry_id") == "" {
    redirectWithError("Please select an athlete")
    return
}
if r.FormValue("run_number") == "" {
    redirectWithError("Please select a run number")
    return
}
if rawTimeStr == "" {
    redirectWithError("Please enter the raw time")
    return
}
```

### 2. Request logging middleware — `handler/logging.go`

Create a simple logging middleware that logs each request:

```go
package handler

import (
    "log"
    "net/http"
    "time"
)

func LoggingMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()

        // Wrap ResponseWriter to capture status code
        lrw := &loggingResponseWriter{ResponseWriter: w, statusCode: 200}
        next.ServeHTTP(lrw, r)

        log.Printf("%s %s %d %s", r.Method, r.URL.Path, lrw.statusCode, time.Since(start).Round(time.Millisecond))
    })
}

type loggingResponseWriter struct {
    http.ResponseWriter
    statusCode int
}

func (lrw *loggingResponseWriter) WriteHeader(code int) {
    lrw.statusCode = code
    lrw.ResponseWriter.WriteHeader(code)
}
```

**Apply in `main.go`:**
```go
server := &http.Server{
    Addr:    ":" + port,
    Handler: handler.LoggingMiddleware(mux),
}
```

This logs every request — useful for debugging during a live event.

### 3. Favicon and robots.txt

**Favicon:** Add a simple text-based handler so browsers don't get 404:
```go
mux.HandleFunc("GET /favicon.ico", func(w http.ResponseWriter, r *http.Request) {
    // Serve a canoe emoji as a favicon via SVG
    w.Header().Set("Content-Type", "image/svg+xml")
    w.Write([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100"><text y=".9em" font-size="90">🛶</text></svg>`))
})
```

**Robots.txt:**
```go
mux.HandleFunc("GET /robots.txt", func(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "text/plain")
    w.Write([]byte("User-agent: *\nAllow: /\n"))
})
```

### 4. Template rendering error recovery

In all handlers that call `ExecuteTemplate`, the current pattern is:
```go
if err := d.Tmpls["event"].ExecuteTemplate(w, "layout.html", data); err != nil {
    log.Printf("Error rendering ...: %v", err)
}
```

The problem: if `ExecuteTemplate` starts writing to `w` and then errors partway through, the user sees a partial page with no error. And the status code is already 200 (headers sent).

**For MVP, this is acceptable.** Don't add complexity (render to buffer → check → write). But do ensure every handler logs the error. Scan all handlers to verify each has the `if err != nil` check.

### 5. Consistent error page usage

Audit all handlers and ensure they use `d.renderError()` instead of bare `http.Error()` or `http.NotFound()`. Specifically check:
- `EventPage` — should use `renderError(w, 404, "Event not found")` ✓
- `AthletePage` — should use `renderError` ✓
- `LeaderboardPage` — should use `renderError` ✓
- `JudgePage` — should use `renderError` ✓
- `EditRunPage` — should use `renderError`
- `UpdateRunHandler` — redirects on error, but edge cases (invalid run ID format) should use `renderError`
- `DeleteRunHandler` — same as above

### 6. Database timeout and WAL mode

Update `db/db.go` to set pragmas that improve SQLite behavior under concurrent reads (auto-refresh polling + judge writing simultaneously):

```go
func Open(dbPath string) (*sql.DB, error) {
    database, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
    if err != nil {
        return nil, err
    }
    // ... rest of function
}
```

- `_journal_mode=WAL` — Write-Ahead Logging allows concurrent reads and writes (critical when the leaderboard is auto-refreshing every 10 seconds while a judge is submitting runs).
- `_busy_timeout=5000` — Wait up to 5 seconds if the database is locked, instead of returning an error immediately.

Alternatively, set these via PRAGMA after opening:
```go
database.Exec("PRAGMA journal_mode=WAL")
database.Exec("PRAGMA busy_timeout=5000")
```

### 7. HTTP security headers

Add basic security headers via middleware or in the `LoggingMiddleware`:

```go
func securityHeaders(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("X-Content-Type-Options", "nosniff")
        w.Header().Set("X-Frame-Options", "DENY")
        w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
        next.ServeHTTP(w, r)
    })
}
```

Compose with logging:
```go
Handler: securityHeaders(handler.LoggingMiddleware(mux)),
```

Or combine both into a single middleware for simplicity.

### 8. Consistent time display on all pages

Audit all templates to ensure times are always displayed using the `formatTime` template function (from `domain.FormatTime`):
- Athlete profile page runs table: uses `.RawTimeFormatted` and `.TotalTimeFormatted` methods — these use the same underlying `formatTimeMs` function. ✓
- Leaderboard: uses `{{formatTime .Run1.TotalTimeMs}}` etc. ✓
- Judge success flash: uses `successRun.TotalTimeFormatted()` in Go string. ✓
- Edit run page: shows `RawTimeSec` as seconds string for input. This is correct for the input field.
- Recent runs in judge page: should display time via `formatTime`. Verify the template uses it.

### 9. Final CSS polish pass

Review and add any missing styles:

**Form inputs focus state:**
```css
input:focus, select:focus {
    outline: 2px solid #2563eb;
    outline-offset: 2px;
}
```

**Print styles (for leaderboard printout at venue):**
```css
@media print {
    .site-header, .site-footer, .site-nav, .tab-bar, .refresh-bar, .judge-panel { display: none; }
    .container { max-width: 100%; }
    .badge-new { animation: none; }
    body { font-size: 12pt; }
}
```

**High-contrast mode for outdoor readability:**
The existing blue/white scheme is already decent. No dark mode needed. But ensure:
- All text has minimum contrast ratio of 4.5:1 against background.
- The `.penalty-touch` amber (#d97706) on white background meets WCAG AA. (It does — ratio is 4.6:1.)
- The `.time-behind` gray (#9ca3af) on white is 2.8:1 — below AA. Darken it to `#6b7280` (4.6:1). Update in CSS.

### 10. Update README with Phase 2 features

Append to `README.md`:

```markdown
## Features

### For Spectators
- Live leaderboard with auto-refresh (10s)
- Medal indicators for top 3
- "NEW" badges on freshly recorded runs
- Time-behind-leader display
- Athlete profiles with bios and run history

### For Judges
- Mobile-friendly run entry with large buttons
- Confirmation step before saving
- Penalty steppers (touch/miss counters)
- Run status indicators (which athletes still need runs)
- Edit and delete runs to correct mistakes
- Recent runs feed

### Security
- Admin token authentication for judge/admin routes
- Session cookies (no repeated token entry)
- Server-side input validation with bounds checking
```

### 11. End-to-end Phase 2 smoke test

After all changes, verify:

1. **Auth flow**: Start with `ADMIN_TOKEN=secret123`. Public pages work without token. Judge page shows 403 without token. With `?token=secret123` → cookie set, access granted.
2. **Judge form**: Select K1M → athlete shows run status → enter data → "Review Run →" → confirmation panel → "Confirm ✓" → saved. Flash message shows. Category stays selected.
3. **Run edit**: Click ✏️ on recent run → edit form pre-filled → change touches → "Update Run ✓" → leaderboard reflects change.
4. **Run delete**: Edit page → "🗑 Delete This Run" → confirm dialog → deleted → flash message → leaderboard updates.
5. **Leaderboard**: Auto-refreshes every 10s. NEW badges pulse on fresh runs. Medals for top 3. Time behind shown. Penalty badges clean/amber/red. Content-change flash animation.
6. **Validation**: Enter raw time "0" → error. Enter raw time "abc" → error. Leave athlete unselected → error. Duplicate run → error.
7. **Request logging**: Terminal shows `GET /events/demo-slalom-2026 200 12ms` etc. for all requests.
8. **Mobile**: All pages usable on 375px viewport. Judge buttons ≥ 48px. Tables scroll horizontally. No content overflow.
9. **Print**: Print leaderboard page → clean output, no nav/footer.
10. **Error pages**: Navigate to `/events/nonexistent` → styled 404. Navigate to `/judge/events/demo-slalom-2026` without auth → styled 403.
11. **Concurrent access**: Open leaderboard in one tab (auto-refreshing), submit runs in another tab (judge). No database lock errors.

## Files to create/modify

```
handler/judge.go         (modify — strengthen validation bounds)
handler/logging.go       (create — LoggingMiddleware, securityHeaders)
handler/public.go        (modify — audit renderError usage)
db/db.go                 (modify — add WAL mode and busy_timeout)
main.go                  (modify — apply logging/security middleware, favicon, robots.txt)
static/style.css         (modify — focus states, print styles, contrast fixes)
README.md                (modify — add features section)
```

## Important notes

- This is the FINAL prompt of Phase 2. After this, the app should be robust enough for a real demo event.
- Do NOT add new features. Only harden, validate, polish, and fix.
- Total Go code should be under 1,500 lines across all `.go` files. If it's significantly over, consolidate.
- Run `go vet ./...` and `go build ./...` — zero warnings, zero errors.
- The WAL mode change is important: without it, simultaneous auto-refresh reads and judge writes will cause "database is locked" errors. This is the single most impactful reliability fix.
