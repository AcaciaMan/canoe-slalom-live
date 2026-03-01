# Prompt 07 — Judge UI Overhaul for Outdoor Use

## Context

You are working on **canoe-slalom-live**, a Go web app for canoe/water slalom competitions. Stack: Go `net/http` (Go 1.22+), `html/template`, SQLite.

Read `PLAN.md` (Phase 2 — Judge UI for outdoor use) for design intent.

**Current state after Prompt 06:** Admin token auth middleware is in place. Judge routes (`/judge/*`) are now protected by `RequireAuth` middleware. The `Deps` struct has `AdminToken string` and `Sessions map[string]bool` fields (with `sync.RWMutex` if added).

**Current judge form** (`templates/judge_run.html`) works but is designed for desktop use:
- Category selection via radio cards
- Athlete selection via radio cards filtered by category (JS)
- Run number selection via inline radio cards
- Raw time as a text input (`inputmode="decimal"`)
- Penalty steppers (+/− buttons with hidden inputs) and live penalty preview
- Submit button ("Record Run ✓")
- Flash messages for success/error via URL query params

**Problems for real outdoor use:**
1. No confirmation step — a judge can accidentally submit with wrong data (fat-finger on a wet phone screen).
2. Athlete list is long — no indication of which athletes already have runs recorded, making it hard to pick the right one.
3. No way to quickly see a summary of recent runs entered.
4. The form resets fully after each submit, losing context (category selection resets).

## Goal

Redesign the judge UI to be usable in a noisy, wet outdoor environment on a phone. Add a confirmation step, show run status per athlete, and preserve category context across submissions.

## What to build

### 1. Show run status per athlete in the selection list

Update the judge page data to include **which runs already exist** for each athlete.

**Backend changes — `handler/judge.go`:**

Add a new struct for judge-specific athlete display:
```go
type JudgeAthleteEntry struct {
    store.EntryWithAthlete
    HasRun1  bool
    HasRun2  bool
    Run1Time string  // formatted total time if exists, e.g. "1:34.37"
    Run2Time string  // formatted total time if exists
}
```

In the `JudgePage` handler, after fetching entries for each category, query runs for each entry to determine which runs exist. Use `store.ListRunsByEntry()` (already exists). Build `[]JudgeAthleteEntry` slices.

Update `JudgePageData` to use the enriched entries:
```go
type JudgeCategoryWithEntries struct {
    Category domain.Category
    Entries  []JudgeAthleteEntry
}

type JudgePageData struct {
    Event      domain.Event
    Categories []JudgeCategoryWithEntries
    Success    string
    Error      string
    Title      string
    SelectedCat int  // category ID to pre-select (from query param)
}
```

Add `?cat=` query param support: after recording a run, the redirect should include `?cat={categoryID}` so the judge stays in the same category. Read it in `JudgePage` and pass as `SelectedCat`.

**Update the redirect in `SubmitRun`** to include the category:
- Look up which category the entry belongs to (from `store.GetEntryByID` — the `EntryWithAthlete` already has `EventID`, but you'll need `CategoryID`). 
- Add `category_id` to `EntryWithAthlete` struct if it's not already there (check `store/athletes.go` — the current `GetEntryByID` query doesn't select `category_id`). Add it.
- Redirect to: `/judge/events/{slug}?success={msg}&cat={categoryID}`

### 2. Update athlete radio cards in template

Update `templates/judge_run.html` athlete section to show run status:

Each athlete card should display:
- Bib number and name (existing)
- Run status indicators: "R1: ✓ 1:34.37" or "R1: —" for each run
- If both runs are recorded, visually dim the card (lower opacity, gray border) to make it obvious this athlete is "done"
- If both runs exist, the radio button should still be selectable (in case of corrections) but visually de-emphasized

Example athlete card layout:
```html
<label class="radio-card athlete-option {{if and .HasRun1 .HasRun2}}athlete-done{{end}}" data-category-id="...">
    <input type="radio" name="entry_id" value="...">
    <div class="radio-card-content">
        <span class="radio-card-label"><strong>#101</strong> Jan Rohan</span>
        <span class="athlete-run-status">
            <span class="{{if .HasRun1}}run-done{{else}}run-pending{{end}}">R1: {{if .HasRun1}}✓ {{.Run1Time}}{{else}}—{{end}}</span>
            <span class="{{if .HasRun2}}run-done{{else}}run-pending{{end}}">R2: {{if .HasRun2}}✓ {{.Run2Time}}{{else}}—{{end}}</span>
        </span>
    </div>
</label>
```

### 3. Auto-select run number based on athlete status

When the athlete selection changes (JS), automatically select the next available run number:
- If athlete has no runs → auto-select "Run 1"
- If athlete has Run 1 but not Run 2 → auto-select "Run 2"
- If both exist → leave current selection (likely editing a previous run)

Add `data-has-run1` and `data-has-run2` attributes to each athlete radio card. JS reads these on change.

### 4. Confirmation step

Instead of submitting immediately, add a two-phase flow:

**Phase 1 (form):** Judge fills in the form as before (all fields visible).

**Phase 2 (confirmation):** When the judge clicks "Review Run →" (renamed from "Record Run ✓"), JavaScript:
1. Validates all fields are filled (client-side).
2. Hides the form.
3. Shows a large confirmation panel with:

```
┌──────────────────────────────────┐
│       CONFIRM RUN ENTRY          │
│                                  │
│  Athlete:  #101 Jan Rohan        │
│  Category: K1M                   │
│  Run:      Run 1                 │
│                                  │
│  Raw Time:     1:34.37           │
│  Touches:      2 × 2s = 4s      │
│  Misses:       0                 │
│  Total Penalty: 4s              │
│  ────────────────────────        │
│  TOTAL TIME:   1:38.37           │
│                                  │
│  ┌────────────┐ ┌──────────────┐ │
│  │  ← BACK    │ │ CONFIRM  ✓  │ │
│  │  (edit)     │ │ (save run)  │ │
│  └────────────┘ └──────────────┘ │
└──────────────────────────────────┘
```

- "← Back" button shows the form again (no data lost — fields retain values).
- "Confirm ✓" button actually submits the `<form>`.
- The confirmation panel uses very large text (1.5rem+, bold) and big buttons (64px height).
- Total time is computed client-side for preview: `rawTimeSeconds + penaltySeconds`. Format as `M:SS.xx`.

**Implementation approach:**
- Keep the existing `<form>` element and submit flow unchanged on the server side.
- Add a `<div id="confirmation-panel" class="confirmation-panel" style="display:none">` after the form.
- The form's submit button becomes `type="button"` (not `type="submit"`) and clicks trigger the confirmation JS.
- In the confirmation panel, the "Confirm ✓" button calls `document.querySelector('.judge-form').submit()` to actually POST.
- The "← Back" button toggles visibility back.

### 5. Recent runs feed

Add a "Recent Runs" section at the top of the judge page, below the flash message, showing the last 5 runs recorded for this event. This gives judges a quick reference.

**Backend:**
Add a new store function `store/runs.go`:
```go
func ListRecentRuns(db *sql.DB, eventID int, limit int) ([]RecentRun, error)
```

```go
type RecentRun struct {
    AthleteName    string
    BibNumber      int
    CategoryCode   string
    RunNumber      int
    TotalTimeMs    int
    PenaltySeconds int
    Status         string
    JudgedAt       string
}
```

Query: join `runs → entries → athletes → categories` where `entries.event_id = ?`, order by `runs.judged_at DESC`, limit to `limit`.

Add `RecentRuns []store.RecentRun` field to `JudgePageData`. Populate in `JudgePage` handler.

**Template:** Show as a compact horizontal-scrollable card row or simple list:
```
Recent: #101 K1M R1 1:38.37 (4s pen) | #203 C1W R1 1:52.01 (Clean) | ...
```

### 6. Update CSS — `static/style.css`

Add/modify styles:

```css
/* Athlete run status in judge form */
.radio-card-content {
    display: flex;
    flex-direction: column;
    gap: 0.25rem;
    width: 100%;
}
.athlete-run-status {
    display: flex;
    gap: 1rem;
    font-size: 0.8rem;
}
.run-done { color: #16a34a; }
.run-pending { color: #9ca3af; }
.athlete-done {
    opacity: 0.5;
    border-color: #d1d5db;
}
.athlete-done:has(input:checked) {
    opacity: 1;
}

/* Confirmation panel */
.confirmation-panel {
    background: #f0f9ff;
    border: 3px solid #2563eb;
    border-radius: 12px;
    padding: 2rem;
    max-width: 500px;
    margin: 0 auto;
}
.confirmation-panel h2 {
    text-align: center;
    font-size: 1.3rem;
    margin-bottom: 1.5rem;
    color: #1e3a5f;
}
.confirm-row {
    display: flex;
    justify-content: space-between;
    padding: 0.5rem 0;
    border-bottom: 1px solid #e0e7ff;
    font-size: 1.1rem;
}
.confirm-row:last-of-type {
    border-bottom: 2px solid #2563eb;
    font-weight: 700;
    font-size: 1.3rem;
}
.confirm-label { color: #6b7280; }
.confirm-value { font-weight: 600; }
.confirm-buttons {
    display: flex;
    gap: 1rem;
    margin-top: 1.5rem;
}
.btn-back {
    flex: 1;
    height: 64px;
    font-size: 1.1rem;
    font-weight: 600;
    background: #f3f4f6;
    color: #374151;
    border: 2px solid #d1d5db;
    border-radius: 8px;
    cursor: pointer;
}
.btn-confirm {
    flex: 2;
    height: 64px;
    font-size: 1.2rem;
    font-weight: 700;
    background: #16a34a;
    color: #fff;
    border: none;
    border-radius: 8px;
    cursor: pointer;
}
.btn-confirm:hover { background: #15803d; }
.btn-back:hover { background: #e5e7eb; }

/* Recent runs */
.recent-runs {
    margin-bottom: 1.5rem;
    padding: 0.75rem;
    background: #f9fafb;
    border-radius: 8px;
    font-size: 0.85rem;
}
.recent-runs h3 {
    font-size: 0.9rem;
    color: #6b7280;
    margin-bottom: 0.5rem;
}
.recent-run-item {
    display: inline-block;
    padding: 0.25rem 0.5rem;
    margin: 0.15rem;
    background: #fff;
    border: 1px solid #e5e7eb;
    border-radius: 4px;
    white-space: nowrap;
}
```

### 7. JavaScript updates — `templates/judge_run.html` inline script

Extend the existing inline script in `judge_run.html`:

**Category pre-selection:** On page load, if `SelectedCat > 0`, auto-check that category radio and trigger the filter. Pass `SelectedCat` as a `data-selected-cat` attribute on a container div, read it from JS.

**Auto-select run number:** When an athlete radio is selected, read `data-has-run1` / `data-has-run2` attributes, then programmatically check the appropriate run number radio.

**Confirmation flow:**
- Change submit button to `type="button"` with `onclick="showConfirmation()"`.
- `showConfirmation()` validates fields, reads selected athlete name/bib from the checked radio's label, computes total time, populates the confirmation panel, and toggles visibility.
- "← Back" calls `hideConfirmation()` to toggle back.
- "Confirm ✓" calls `document.querySelector('.judge-form').submit()`.
- Time formatting in JS: convert seconds decimal to `M:SS.xx` format (same as server-side `FormatTime`).

Keep total inline JS under 120 lines.

## Verification

1. `go build ./...` — no errors.
2. Start with `ADMIN_TOKEN=secret123 go run main.go` (or without token for easy testing).
3. Open judge page. Athletes now show run status icons (R1: —, R2: —).
4. Record a run for an athlete → page redirects, stays on same category, athlete now shows "R1: ✓ 1:34.37".
5. Select that athlete again → run number auto-selects "Run 2".
6. Fill form → click "Review Run →" → confirmation panel shows with all details and computed total.
7. Click "← Back" → back to form with values preserved.
8. Click "Confirm ✓" → run saved, redirect with success.
9. "Recent Runs" section at top shows the last few recorded runs.
10. Record runs for all athletes → athletes with both runs show dimmed cards.
11. Test on mobile viewport (375px) — all buttons ≥ 48px, confirmation panel readable, no overflow.

## Files to create/modify

```
handler/judge.go            (modify — JudgeAthleteEntry, JudgeCategoryWithEntries, enrich entries with run data, SelectedCat, redirect with cat)
store/runs.go               (modify — add ListRecentRuns, RecentRun struct)
store/athletes.go           (modify — possibly add category_id to EntryWithAthlete if missing from GetEntryByID)
templates/judge_run.html    (modify — run status display, confirmation panel, updated JS)
static/style.css            (modify — add confirmation, run-status, recent-runs styles)
```

## Important notes

- Do NOT change the form's server-side POST handling (`SubmitRun` in handler/judge.go). The form still POSTs the same fields. The confirmation step is 100% client-side JS.
- The recent runs query should be efficient — it's a single SQL query with JOINs and LIMIT, no N+1.
- Keep the existing penalty stepper logic exactly as-is. Only add to it, don't rewrite.
- The `data-has-run1` / `data-has-run2` attributes on athlete cards enable JS auto-selection without extra API calls. No need for a separate endpoint.
