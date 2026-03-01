# Prompt 08 — Run Edit and Delete

## Context

You are working on **canoe-slalom-live**, a Go web app for canoe/water slalom competitions. Stack: Go `net/http` (Go 1.22+), `html/template`, SQLite.

Read `PLAN.md` for full context. Phase 2 is underway. Previous prompts in this phase:
- **Prompt 06**: Admin token auth middleware (`handler/auth.go`). Judge routes protected by `RequireAuth` middleware. `Deps` struct now has `AdminToken`, `Sessions`, and a mutex.
- **Prompt 07**: Judge UI overhaul — confirmation step, athlete run status indicators, auto-select run number, recent runs feed, category persistence across submissions.

**Current state of run management:**
- Runs can only be **created** via `POST /judge/events/{slug}/runs` (in `handler/judge.go` `SubmitRun`).
- There is NO way to edit or delete a run once recorded.
- The `UNIQUE(entry_id, run_number)` constraint prevents duplicate runs. If a judge makes a mistake, they're currently stuck.
- `store/runs.go` has: `CreateRun()`, `ListRunsByEntry()`, `GetEntryByID()`, `GetLeaderboard()`, `ListRecentRuns()`.

## Goal

Add the ability to edit and delete runs from the judge panel. Judges make mistakes — wrong time, wrong penalty count, wrong athlete. They need to fix these without asking a developer to edit the database.

## What to build

### 1. Store functions — `store/runs.go`

Add these new functions:

**`GetRunByID`:**
```go
func GetRunByID(db *sql.DB, id int) (domain.Run, error)
```
Query: `SELECT id, entry_id, run_number, raw_time_ms, penalty_touches, penalty_misses, penalty_seconds, total_time_ms, status, judged_at FROM runs WHERE id = ?`

**`UpdateRun`:**
```go
func UpdateRun(db *sql.DB, run domain.Run) error
```
Query: `UPDATE runs SET raw_time_ms = ?, penalty_touches = ?, penalty_misses = ?, penalty_seconds = ?, total_time_ms = ?, status = ?, judged_at = ? WHERE id = ?`

Only updates the time/penalty/status fields. Does NOT allow changing `entry_id` or `run_number` (if entry or run number was wrong, the judge should delete and re-enter).

**`DeleteRun`:**
```go
func DeleteRun(db *sql.DB, id int) error
```
Query: `DELETE FROM runs WHERE id = ?`

### 2. Handlers — `handler/judge.go`

Add three new handler methods on `Deps`:

**`EditRunPage`** (`GET /judge/events/{slug}/runs/{id}/edit`):
1. Extract `slug` and `id` from path.
2. Fetch event by slug. 404 if not found.
3. Fetch run by ID via `store.GetRunByID`. 404 if not found.
4. Fetch the entry for context (athlete name, bib, category) via `store.GetEntryByID(d.DB, run.EntryID)`.
5. Verify the entry belongs to this event (`entry.EventID == event.ID`). 403 if not.
6. Render an edit form pre-populated with the existing run data.
7. Template data:

```go
type EditRunPageData struct {
    Event    domain.Event
    Run      domain.Run
    Entry    store.EntryWithAthlete
    RawTimeSec string  // pre-computed: run.RawTimeMs / 1000.0, formatted as "94.37"
    Title    string
    Error    string
}
```

**`UpdateRunHandler`** (`POST /judge/events/{slug}/runs/{id}`):
1. Extract slug and id.
2. Fetch event and existing run. Validate run belongs to event.
3. Parse form: `raw_time` (seconds string), `touches` (int), `misses` (int).
4. Validate inputs (same rules as `SubmitRun`).
5. Recompute `penalty_seconds` and `total_time_ms`.
6. Call `store.UpdateRun()`.
7. Redirect to judge page with success message: `?success=Run+updated:+{athleteName}+Run+{N}+—+{totalFormatted}&cat={categoryID}`

**`DeleteRunHandler`** (`POST /judge/events/{slug}/runs/{id}/delete`):
1. Extract slug and id.
2. Fetch event and run. Validate run belongs to event.
3. Get entry for the athlete name (for the success message).
4. Call `store.DeleteRun()`.
5. Redirect to judge page: `?success=Run+deleted:+{athleteName}+Run+{runNumber}&cat={categoryID}`

Use POST for delete (not DELETE verb) because HTML forms can only POST. Use a dedicated `/delete` sub-path to differentiate from update.

### 3. Route registration — `main.go`

Add these routes (all protected by auth):
```go
mux.HandleFunc("GET /judge/events/{slug}/runs/{id}/edit", deps.RequireAuth(deps.EditRunPage))
mux.HandleFunc("POST /judge/events/{slug}/runs/{id}", deps.RequireAuth(deps.UpdateRunHandler))
mux.HandleFunc("POST /judge/events/{slug}/runs/{id}/delete", deps.RequireAuth(deps.DeleteRunHandler))
```

### 4. Edit run template — `templates/judge_edit_run.html`

Create a new template for the edit run form. It should be similar to the judge run entry form but simpler (no category/athlete selection — those are fixed):

```
{{define "content"}}
<div class="judge-panel">
    <h1>✏️ Edit Run</h1>
    <p class="event-subtitle">{{.Event.Name}}</p>

    {{if .Error}}
    <div class="flash flash-error">✗ {{.Error}}</div>
    {{end}}

    <!-- Run context (read-only) -->
    <div class="edit-run-context">
        <div class="confirm-row">
            <span class="confirm-label">Athlete</span>
            <span class="confirm-value">#{{.Entry.BibNumber}} {{.Entry.AthleteName}}</span>
        </div>
        <div class="confirm-row">
            <span class="confirm-label">Run</span>
            <span class="confirm-value">Run {{.Run.RunNumber}}</span>
        </div>
    </div>

    <form action="/judge/events/{{.Event.Slug}}/runs/{{.Run.ID}}" method="POST" class="judge-form">

        <!-- Raw Time -->
        <fieldset class="form-section">
            <legend>Raw Time (seconds)</legend>
            <input type="text" name="raw_time" inputmode="decimal" value="{{.RawTimeSec}}" class="time-input" required autocomplete="off">
        </fieldset>

        <!-- Penalties — reuse stepper pattern -->
        <fieldset class="form-section">
            <legend>Penalties</legend>

            <div class="penalty-row">
                <span class="penalty-label">Gate Touches (2s each)</span>
                <div class="stepper">
                    <button type="button" class="stepper-btn" onclick="adjustStepper('touches', -1)">−</button>
                    <span id="touches-display" class="stepper-value">{{.Run.PenaltyTouches}}</span>
                    <button type="button" class="stepper-btn" onclick="adjustStepper('touches', 1)">+</button>
                </div>
                <input type="hidden" name="touches" id="touches-input" value="{{.Run.PenaltyTouches}}">
            </div>

            <div class="penalty-row">
                <span class="penalty-label">Missed Gates (50s each)</span>
                <div class="stepper">
                    <button type="button" class="stepper-btn" onclick="adjustStepper('misses', -1)">−</button>
                    <span id="misses-display" class="stepper-value">{{.Run.PenaltyMisses}}</span>
                    <button type="button" class="stepper-btn" onclick="adjustStepper('misses', 1)">+</button>
                </div>
                <input type="hidden" name="misses" id="misses-input" value="{{.Run.PenaltyMisses}}">
            </div>

            <div class="penalty-preview" id="penalty-preview">...</div>
        </fieldset>

        <div class="edit-actions">
            <button type="submit" class="btn-submit">Update Run ✓</button>
        </div>
    </form>

    <!-- Delete form (separate) -->
    <form action="/judge/events/{{.Event.Slug}}/runs/{{.Run.ID}}/delete" method="POST" class="delete-form"
          onsubmit="return confirm('Delete Run {{.Run.RunNumber}} for {{.Entry.AthleteName}}? This cannot be undone.');">
        <button type="submit" class="btn-delete">🗑 Delete This Run</button>
    </form>

    <a href="/judge/events/{{.Event.Slug}}" class="back-link">← Back to Judge Panel</a>
</div>

<!-- Reuse stepper JS -->
<script>
(function() {
    window.adjustStepper = function(name, delta) {
        var input = document.getElementById(name + '-input');
        var display = document.getElementById(name + '-display');
        var val = parseInt(input.value) + delta;
        if (val < 0) val = 0;
        input.value = val;
        display.textContent = val;
        updatePenaltyPreview();
    };

    function updatePenaltyPreview() {
        var touches = parseInt(document.getElementById('touches-input').value) || 0;
        var misses = parseInt(document.getElementById('misses-input').value) || 0;
        var total = touches * 2 + misses * 50;
        var el = document.getElementById('penalty-preview');
        if (total === 0) {
            el.textContent = 'Penalty: 0s (Clean)';
        } else {
            var parts = [];
            if (touches > 0) parts.push(touches + '×2');
            if (misses > 0) parts.push(misses + '×50');
            el.textContent = 'Penalty: ' + parts.join(' + ') + ' = ' + total + 's';
        }
    }

    updatePenaltyPreview();
})();
</script>
{{end}}
```

### 5. Add edit/delete links to existing UI

**Recent runs in judge page** (`templates/judge_run.html`):
In the recent runs section, each run item should include a small "Edit" link:
```html
<a href="/judge/events/{{$.Event.Slug}}/runs/{{.ID}}/edit" class="edit-link">✏️</a>
```

This requires adding `ID` (run ID) to the `RecentRun` struct in `store/runs.go`. Update `ListRecentRuns` to also select `runs.id`.

**Athlete profile page** (`templates/athlete.html`):
In the runs results table, each run row could optionally show an "Edit" link — but this is a public page, and editing requires auth. For now, skip adding edit links on the public page. The judge uses the judge panel.

### 6. Register template in `main.go`

Add to template map:
```go
"judge_edit": template.Must(template.New("layout.html").Funcs(funcMap).ParseFiles("templates/layout.html", "templates/judge_edit_run.html")),
```

### 7. CSS additions — `static/style.css`

```css
/* Edit run context */
.edit-run-context {
    background: #f0f9ff;
    border: 2px solid #bfdbfe;
    border-radius: 8px;
    padding: 1rem;
    margin-bottom: 1.5rem;
}

/* Edit actions */
.edit-actions {
    margin-bottom: 1rem;
}

/* Delete button */
.delete-form {
    margin-top: 1rem;
    margin-bottom: 1rem;
}
.btn-delete {
    width: 100%;
    height: 48px;
    font-size: 1rem;
    font-weight: 600;
    color: #991b1b;
    background: #fff;
    border: 2px solid #fecaca;
    border-radius: 8px;
    cursor: pointer;
    transition: background 0.15s;
}
.btn-delete:hover {
    background: #fee2e2;
}

/* Edit link in recent runs */
.edit-link {
    font-size: 0.8rem;
    text-decoration: none;
    margin-left: 0.25rem;
}
```

### 8. Add `CategoryID` to `EntryWithAthlete` if missing

Check `store/athletes.go` — the `EntryWithAthlete` struct currently has: `EntryID, EventID, BibNumber, StartPosition, AthleteID, AthleteName, Club, Nation`. It does NOT have `CategoryID`.

Add `CategoryID int` to `EntryWithAthlete`. Update ALL queries that populate `EntryWithAthlete` to also SELECT and Scan `e.category_id`:
- `GetEntryByID` in `store/runs.go`
- `ListEntriesByCategory` in `store/athletes.go`
- `GetEntryByEventAndAthlete` in `store/athletes.go`

This is needed so the redirect after update/delete can include `?cat={categoryID}`.

## Verification

1. `go build ./...` — no errors.
2. Start server with auth: `set ADMIN_TOKEN=secret123` then `go run main.go -seed`.
3. Open judge page → record a run → see it in "Recent Runs".
4. Click ✏️ edit icon on a recent run → edit page loads with pre-filled data.
5. Change touches from 2 to 3 → click "Update Run ✓" → redirects to judge page with "Run updated" message.
6. Verify leaderboard reflects the updated penalty.
7. Go back to edit page → click "🗑 Delete This Run" → browser confirm dialog → click OK → redirect with "Run deleted" message.
8. Athlete's run status in judge list updates (R1: — if deleted).
9. Leaderboard reflects the deletion.
10. Try to edit a run from a different event slug → 403 or 404.
11. Try to edit without auth cookie → 403 (access denied).

## Files to create/modify

```
store/runs.go                     (modify — add GetRunByID, UpdateRun, DeleteRun; add ID to RecentRun)
store/athletes.go                 (modify — add CategoryID to EntryWithAthlete, update all queries)
handler/judge.go                  (modify — add EditRunPage, UpdateRunHandler, DeleteRunHandler)
templates/judge_edit_run.html     (create)
templates/judge_run.html          (modify — add edit links to recent runs)
main.go                           (modify — add routes, add template)
static/style.css                  (modify — add edit/delete styles)
```

## Important notes

- Delete uses `POST` to a `/delete` sub-path, not HTTP `DELETE` verb. HTML forms don't support DELETE.
- Delete has a `confirm()` JavaScript dialog. This is intentionally simple. A fancier modal can come later.
- Update does NOT allow changing the athlete or run number. If those were wrong, delete and re-enter. This avoids complex validation around unique constraints.
- The `RawTimeSec` field on `EditRunPageData` should format `run.RawTimeMs / 1000.0` as a string with 2 decimal places (e.g., `"94.37"` not `"94.36999..."`). Use `fmt.Sprintf("%.2f", float64(run.RawTimeMs)/1000.0)`.
- All edit/delete routes MUST be wrapped in `RequireAuth` middleware.
