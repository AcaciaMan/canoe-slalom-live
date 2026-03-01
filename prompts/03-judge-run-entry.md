# Prompt 03 — Judge Run Entry Interface

## Context

You are working on **canoe-slalom-live**, a Go web app for canoe/water slalom competitions. Stack: Go `net/http`, `html/template`, SQLite.

Read `PLAN.md` for full architecture. Previous steps built:
- **Prompt 01**: Database layer (`db/`), domain structs (`domain/`), store queries (`store/`), seed data
- **Prompt 02**: HTTP server in `main.go`, public event page with start list (`handler/public.go`), athlete profile page, templates (`layout.html`, `event.html`, `athlete.html`), CSS (`static/style.css`)

The app runs on `localhost:8080`. Visiting `/events/demo-slalom-2026` shows the event with 10 athletes across 2 categories. No runs exist yet.

This prompt implements **Phase 1, Step 3**: the judge interface to record runs with time and penalties.

## Goal

A judge opens `http://localhost:8080/judge/events/demo-slalom-2026` and can:
1. Select a category (K1M or C1W).
2. Select an athlete from that category's start list.
3. Enter the run number (1 or 2).
4. Enter the raw time in seconds with hundredths (e.g., `94.37`).
5. Enter penalty touch count and missed gate count.
6. Submit → the app computes `penalty_seconds` (touches × 2 + misses × 50) and `total_time_ms`, saves the run, and redirects back with a success message.

## What to build

### 1. Handler — `handler/judge.go`

Package `handler`. Uses the same `Deps` struct (with `DB` and `Tmpls`) from `handler/public.go`.

**JudgePage handler** (`GET /judge/events/{slug}`):
1. Extract slug, fetch event. 404 if not found.
2. Fetch categories for this event.
3. For each category, fetch entries (with athlete names) using `store.ListEntriesByCategory`.
4. Check for a `?success=` query param to display a flash message (URL-encoded string).
5. Check for a `?error=` query param similarly.
6. Render `judge_run.html` with data:

```go
JudgePageData struct {
    Event      domain.Event
    Categories []CategoryWithEntries  // reuse from public handler or define shared
    Success    string                 // flash message
    Error      string                 // error message
    Title      string
}
```

**SubmitRun handler** (`POST /judge/events/{slug}/runs`):
1. Extract slug, fetch event. 404 if not found.
2. Parse form values: `entry_id`, `run_number`, `raw_time`, `touches`, `misses`.
3. **Validate inputs:**
   - `entry_id` must be a valid integer and must belong to this event (query: check `entries` table where `event_id` matches). If not, redirect back with error.
   - `run_number` must be 1 or 2.
   - `raw_time` must be a positive decimal number in seconds (e.g., `94.37`). Parse it to float64, then convert to milliseconds: `int(rawTime * 1000)`.
   - `touches` must be ≥ 0 integer.
   - `misses` must be ≥ 0 integer.
4. **Compute penalties:**
   - `penaltySeconds = touches * 2 + misses * 50`
   - `totalTimeMs = rawTimeMs + penaltySeconds * 1000`
5. **Create the run** via `store.CreateRun(d.DB, domain.Run{...})`.
6. **Handle duplicate run error**: If the UNIQUE constraint on `(entry_id, run_number)` is violated, catch the error and redirect back with message "Run {N} already recorded for this athlete. Delete it first to re-enter."
7. **On success**: Redirect to `GET /judge/events/{slug}?success=Run+recorded:+{athleteName}+—+{totalFormatted}` (URL-encode the message).

**Store additions needed** (add to `store/runs.go` or `store/athletes.go` if not already present):
- `func GetEntryByID(db *sql.DB, id int) (EntryWithAthlete, error)` — needed to validate entry belongs to event and to get athlete name for the success message.
- `func ValidateEntryBelongsToEvent(db *sql.DB, entryID, eventID int) (bool, error)` — or just do it inline in `GetEntryByID` by checking `event_id`.

### 2. Route registration in `main.go`

Add these routes to the mux:
```
GET  /judge/events/{slug}       → deps.JudgePage
POST /judge/events/{slug}/runs  → deps.SubmitRun
```

Add `"judge"` to the template map:
```go
"judge": template.Must(template.ParseFiles("templates/layout.html", "templates/judge_run.html")),
```

### 3. Template — `templates/judge_run.html`

This is the critical UI. It must be usable on a phone at a riverside with wet hands. Design for large targets and minimal typing.

```
{{define "content"}}
```

Layout:

**Header section:**
- Event name and "Judge Panel" title.
- If `Success` message exists, show it in a green banner div.
- If `Error` message exists, show it in a red banner div.

**Form** (`action="/judge/events/{{.Event.Slug}}/runs" method="POST"`):

**1. Category selector:**
- Radio buttons (not a dropdown — radio is faster to tap on mobile).
- Each radio is a large styled button showing the category code and name: `K1M — Kayak Single Men`.
- When a category is selected, filter the athlete list below using vanilla JS.

**2. Athlete selector:**
- Radio button list of athletes, each showing: bib number and name.
- Grouped/filtered by selected category (use JS `data-category` attributes on each radio to show/hide).
- Each radio button should be styled as a tappable card/row (not a tiny radio circle).
- The `value` of the selected radio maps to the `entry_id`.

**3. Run number:**
- Two large buttons (radio styled as buttons): "Run 1" / "Run 2".

**4. Raw time input:**
- Single text input with `inputmode="decimal"` for numeric keyboard on mobile.
- Placeholder: `94.37` (seconds with hundredths).
- Label: "Raw Time (seconds)".
- Large font size (at least 1.5rem).

**5. Penalties:**
- **Gate touches (2s each):** Show a number with a `−` and `+` button on either side. Starts at 0. Stepper pattern. Hidden `<input type="hidden" name="touches" value="0">` updated by JS.
- **Missed gates (50s each):** Same stepper pattern. Starts at 0.
- Show live-computed penalty preview below: "Penalty: {touches}×2 + {misses}×50 = {total}s" — update via JS as steppers change.

**6. Submit button:**
- Large green button: "Record Run ✓".
- Full width on mobile.

**JavaScript (inline or in a `<script>` tag at bottom):**
- Category radio change → filter athlete list (show/hide based on `data-category-id` attribute).
- Stepper buttons: increment/decrement the hidden input values, update display and penalty preview.
- Penalty preview: `document.getElementById('penalty-preview').textContent = ...`
- No JS framework. Vanilla DOM manipulation. Keep it under 60 lines.

### 4. CSS additions — `static/style.css`

Add judge-specific styles (append to existing file):

- `.judge-form` — max-width 500px, centered.
- `.radio-card` — styled radio buttons that look like tappable cards: border, rounded corners, padding 16px, min-height 48px. Selected state: blue background, white text.
- `.stepper` — flex row with `−` and `+` buttons (48×48px minimum, large font) and the count value centered between them.
- `.btn-submit` — large green button, full width, 56px height, bold white text.
- `.flash-success` — green background, white text, padding, margin bottom.
- `.flash-error` — red background, white text.
- `.penalty-preview` — gray background, monospace font, centered text.

### 5. Form data flow

The HTML form submits these fields via POST:
- `entry_id` (from athlete radio selection) — integer
- `run_number` (from run number radio) — 1 or 2
- `raw_time` (from text input) — string like "94.37", parsed server-side
- `touches` (from hidden input updated by stepper) — integer
- `misses` (from hidden input updated by stepper) — integer

The server computes everything else.

## Verification

1. `go build ./...` — no errors.
2. Start server: `go run main.go` (data already seeded from previous step).
3. Open `http://localhost:8080/judge/events/demo-slalom-2026`.
4. Select K1M → see 6 athletes → pick one → select Run 1 → enter `94.37` → set 2 touches, 0 misses → submit.
5. Green flash: "Run recorded: [Athlete Name] — 01:38.37" (or similar based on computed time).
6. Navigate to athlete page → should now show 1 run in the results table.
7. Submit same athlete + Run 1 again → red error: duplicate run.
8. Submit Run 2 for same athlete → success.
9. Switch to C1W category → shows 4 different athletes.

## Files to create/modify

```
handler/judge.go        (create)
templates/judge_run.html (create)
main.go                 (modify: add judge routes + template)
static/style.css        (modify: add judge-specific styles)
store/runs.go           (modify: add GetEntryByID if needed)
```

## Important notes

- Do NOT add any authentication for now. The judge page is open. Auth is Phase 2.
- Time input is in **seconds** (e.g., `94.37`), not MM:SS format. Seconds-only input is faster for the judge. Display can be MM:SS.xx elsewhere but input is raw seconds.
- Penalty computation: touches × 2 + misses × 50. This is the standard ICF rule. Hardcode it.
- After POST, always redirect (POST-Redirect-GET pattern). Never render a page directly from a POST handler.
- The flash message is passed via query param. Not ideal long-term, but fine for MVP.
