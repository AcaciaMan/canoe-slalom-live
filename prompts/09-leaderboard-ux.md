# Prompt 09 — Leaderboard UX Improvements

## Context

You are working on **canoe-slalom-live**, a Go web app for canoe/water slalom competitions. Stack: Go `net/http` (Go 1.22+), `html/template`, SQLite.

Read `PLAN.md` (Phase 2 — Leaderboard UX). Previous Phase 2 prompts:
- **Prompt 06**: Admin token auth middleware.
- **Prompt 07**: Judge UI overhaul (confirmation step, run status, auto-select, recent runs feed).
- **Prompt 08**: Run edit/delete (edit form, delete with confirm, CategoryID on entries).

**Current leaderboard state:**
- `GET /events/{slug}/leaderboard` renders a full-page leaderboard with auto-refresh (15s via `static/app.js` fetching `?partial=1`).
- `templates/leaderboard.html` — wraps content with layout, event header, tab bar, refresh bar.
- `templates/leaderboard_partial.html` — defines `{{leaderboard-tables}}` block used by both the full page and the AJAX partial.
- Leaderboard table columns: Rank | Bib | Athlete | Nation | Run 1 | Run 2 | Best Time.
- Penalty colors: green for clean, amber for touches, red for misses.
- Top 3 get a subtle left border (gold/silver/bronze).
- `store.GetLeaderboard()` returns `[]LeaderboardRow` ranked by best `total_time_ms`.

**What the leaderboard is missing:**
1. No "NEW" indicator when a run was just recorded.
2. Run 1 / Run 2 columns are dense and hard to read — the inline `raw + penalty = total` format is cramped.
3. No indication of which run is the athlete's best.
4. No difference display (time behind leader).
5. The partial refresh replaces innerHTML but doesn't highlight what changed.

## Goal

Make the leaderboard visually compelling and informative for spectators watching on their phones or on a projected screen at the venue. Add "just recorded" badges, time-behind-leader display, best-run highlighting, and visual improvements.

## What to build

### 1. Add `JudgedAt` timestamp to leaderboard data

Update `store/runs.go` `GetLeaderboard()` query to also select `r1.judged_at` and `r2.judged_at`.

Add `JudgedAt string` field to `RunResult` struct.

This enables the template to show "NEW" badges for recently recorded runs.

### 2. Add time-behind-leader to `LeaderboardRow`

After computing ranks in the Go code (in `GetLeaderboard`), for each row calculate:
```go
TimeBehindMs int  // difference from rank 1's BestTotalTimeMs; 0 for leader
```

Add this field to `LeaderboardRow`. In the ranking loop:
```go
if len(withRuns) > 0 {
    leaderTime := withRuns[0].BestTotalTimeMs
    for i := range withRuns {
        withRuns[i].TimeBehindMs = withRuns[i].BestTotalTimeMs - leaderTime
    }
}
```

### 3. Update leaderboard partial template

Redesign the Run 1 / Run 2 columns to be cleaner and add new features.

**Updated row structure:**

```html
<tr class="{{rank class}} {{if .IsNew}}new-run-row{{end}}">
    <td class="col-rank">
        {{if eq .Rank 1}}🥇{{else if eq .Rank 2}}🥈{{else if eq .Rank 3}}🥉{{else if gt .Rank 0}}#{{.Rank}}{{else}}<span class="no-runs">—</span>{{end}}
    </td>
    <td class="col-bib">{{.BibNumber}}</td>
    <td>
        <a href="/events/{{$slug}}/athletes/{{.AthleteID}}">{{.AthleteName}}</a>
        {{if gt .TimeBehindMs 0}}<span class="time-behind">+{{formatTime .TimeBehindMs}}</span>{{end}}
    </td>
    <td class="col-nation">{{.AthleteNation}}</td>
    <td class="col-run">
        {{if .Run1}}
            {{if .Run1IsNew}}<span class="badge-new">NEW</span>{{end}}
            <span class="run-total {{if .Run1IsBest}}run-best{{end}}">
                {{formatTime .Run1.TotalTimeMs}}
            </span>
            {{if gt .Run1.PenaltySeconds 0}}
                <span class="run-penalty {{penaltyClass .Run1}}">+{{.Run1.PenaltySeconds}}s</span>
            {{end}}
        {{else}}<span class="no-runs">—</span>{{end}}
    </td>
    <td class="col-run">
        {{if .Run2}}
            {{if .Run2IsNew}}<span class="badge-new">NEW</span>{{end}}
            <span class="run-total {{if .Run2IsBest}}run-best{{end}}">
                {{formatTime .Run2.TotalTimeMs}}
            </span>
            {{if gt .Run2.PenaltySeconds 0}}
                <span class="run-penalty {{penaltyClass .Run2}}">+{{.Run2.PenaltySeconds}}s</span>
            {{end}}
        {{else}}<span class="no-runs">—</span>{{end}}
    </td>
    <td class="col-best">
        {{if gt .BestTotalTimeMs 0}}<strong>{{formatTime .BestTotalTimeMs}}</strong>{{else}}<span class="no-runs">—</span>{{end}}
    </td>
</tr>
```

**Key design changes:**
- **Run columns simplified:** Show only the total time (bold if best run), plus "+Xs" penalty badge if non-zero. Remove the `raw + penalty = total` inline equation — it was too dense. The raw time is visible on the athlete profile page.
- **Best run highlighted:** The run that is the athlete's best gets a `run-best` class (bold, slightly larger).
- **"NEW" badge:** If a run was recorded within the last 60 seconds, show a pulsing "NEW" badge.
- **Medals:** Replace `#1`, `#2`, `#3` text with 🥇🥈🥉 emoji for top 3.
- **Time behind:** Show `+0:02.14` next to athlete name for non-leaders.

### 4. Computed template fields

Several of the template fields above need pre-computation. The cleanest approach is to add methods or computed fields to `LeaderboardRow`:

**Option A (recommended):** Add computed fields in Go after building rows:

```go
type LeaderboardRow struct {
    // ... existing fields ...
    TimeBehindMs int
    Run1IsBest   bool
    Run2IsBest   bool
    Run1IsNew    bool
    Run2IsNew    bool
}
```

Compute in `GetLeaderboard()` after fetching and ranking:

```go
now := time.Now()
for i := range result {
    row := &result[i]
    // Best run marker
    if row.Run1 != nil && row.Run1.Status == "ok" && row.Run1.TotalTimeMs == row.BestTotalTimeMs {
        row.Run1IsBest = true
    }
    if row.Run2 != nil && row.Run2.Status == "ok" && row.Run2.TotalTimeMs == row.BestTotalTimeMs {
        row.Run2IsBest = true
    }
    // "NEW" badge (within 60 seconds)
    if row.Run1 != nil && row.Run1.JudgedAt != "" {
        if t, err := time.Parse(time.RFC3339, row.Run1.JudgedAt); err == nil {
            row.Run1IsNew = now.Sub(t) < 60*time.Second
        }
    }
    if row.Run2 != nil && row.Run2.JudgedAt != "" {
        if t, err := time.Parse(time.RFC3339, row.Run2.JudgedAt); err == nil {
            row.Run2IsNew = now.Sub(t) < 60*time.Second
        }
    }
}
```

### 5. Template function: `penaltyClass`

Add a template function to `main.go`'s `funcMap`:

```go
"penaltyClass": func(r *store.RunResult) string {
    if r == nil { return "" }
    if r.PenaltyMisses > 0 { return "penalty-miss" }
    if r.PenaltyTouches > 0 { return "penalty-touch" }
    return "penalty-clean"
},
```

This replaces the inline `{{if gt .Run1.PenaltyMisses 0}}penalty-miss{{else}}penalty-touch{{end}}` logic in the template.

### 6. CSS additions — `static/style.css`

```css
/* NEW badge */
.badge-new {
    display: inline-block;
    background: #2563eb;
    color: #fff;
    padding: 0.1rem 0.4rem;
    border-radius: 3px;
    font-size: 0.65rem;
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    animation: pulse-new 2s ease-in-out infinite;
    vertical-align: middle;
    margin-right: 0.25rem;
}
@keyframes pulse-new {
    0%, 100% { opacity: 1; }
    50% { opacity: 0.5; }
}

/* Best run highlight */
.run-best {
    font-weight: 700;
    color: #1e3a5f;
}

/* Run penalty badge (inline) */
.run-penalty {
    font-size: 0.75rem;
    font-weight: 600;
    margin-left: 0.15rem;
}

/* Run total in column */
.run-total {
    font-size: 0.9rem;
}

/* Time behind leader */
.time-behind {
    display: inline-block;
    font-size: 0.75rem;
    color: #9ca3af;
    margin-left: 0.5rem;
    font-weight: 400;
}

/* NEW row highlight (subtle background) */
.new-run-row {
    background: #eff6ff !important;
}
.new-run-row:hover {
    background: #dbeafe !important;
}

/* Medal ranks — remove left border, use medal emoji styling */
.rank-1 { border-left: none; }
.rank-2 { border-left: none; }
.rank-3 { border-left: none; }
.col-rank {
    font-size: 1.1rem;
}
```

### 7. Enhanced auto-refresh — highlight changes

Update `static/app.js` to briefly highlight rows that changed after a refresh:

```javascript
async function refresh() {
    if (paused) return;
    try {
        const resp = await fetch(`/events/${slug}/leaderboard?partial=1`);
        if (resp.ok) {
            const newHTML = await resp.text();
            // Detect changes by comparing inner text (simple approach)
            const oldText = container.innerText;
            container.innerHTML = newHTML;
            const newText = container.innerText;
            if (oldText !== newText) {
                // Flash the entire leaderboard briefly
                container.classList.add('leaderboard-updated');
                setTimeout(() => container.classList.remove('leaderboard-updated'), 1500);
            }
            if (statusEl) statusEl.textContent = 'Updated ' + new Date().toLocaleTimeString();
        }
    } catch (e) {
        if (statusEl) statusEl.textContent = 'Refresh failed — retrying...';
    }
}
```

Add CSS:
```css
.leaderboard-updated {
    animation: flash-update 1.5s ease-out;
}
@keyframes flash-update {
    0% { box-shadow: 0 0 0 3px rgba(37, 99, 235, 0.4); }
    100% { box-shadow: none; }
}
```

### 8. Reduce auto-refresh interval

Change from 15 seconds to 10 seconds in `static/app.js` for snappier updates during active competition.

## Verification

1. `go build ./...` — no errors.
2. Delete `data.db`, re-seed: `go run main.go -seed`.
3. Open leaderboard — see seeded runs with new design.
4. Top 3 show medal emoji (🥇🥈🥉).
5. Non-leaders show `+X:XX.XX` time behind.
6. For each athlete, the better run is bold (`.run-best`).
7. Penalty badges show `+4s` in amber or `+50s` in red (compact, no raw time).
8. Record a new run via judge page. Within 10 seconds, leaderboard auto-refreshes.
9. Newly recorded run shows "NEW" pulsing badge (disappears after 60 seconds / next refresh cycle).
10. Updated leaderboard container flashes a subtle blue shadow when content changes.
11. Click pause → auto-refresh stops. Click resume → resumes.
12. Open on mobile (375px) — columns still fit, Nation column hidden below 500px, time-behind text wraps gracefully.

## Files to create/modify

```
store/runs.go                      (modify — add JudgedAt to RunResult, TimeBehindMs/Run1IsBest/Run2IsBest/Run1IsNew/Run2IsNew to LeaderboardRow, compute them)
templates/leaderboard_partial.html (modify — redesigned row layout with medals, penalties, NEW badges, time-behind)
static/style.css                   (modify — add badge-new, run-best, time-behind, run-penalty, flash-update styles)
static/app.js                      (modify — change interval to 10s, add update highlight)
main.go                            (modify — add penaltyClass template func)
```

## Important notes

- The leaderboard partial template MUST remain a named block `{{define "leaderboard-tables"}}` so it's shared between the full page (`leaderboard.html`) and the AJAX partial.
- `penaltyClass` template function receives a `*store.RunResult` pointer. Handle `nil` case (return empty string).
- The "NEW" badge timer is computed server-side relative to `time.Now()`. On AJAX refresh, newly expired badges naturally disappear. No client-side timer needed.
- The simplified run columns (total + penalty badge instead of raw + penalty = total) are intentional. Spectators care about the total and whether there were penalties. The detailed breakdown is on the athlete profile page.
- Keep the leaderboard note at the bottom: "Ranked by best single-run time. Penalties: gate touch = 2s, missed gate = 50s."
