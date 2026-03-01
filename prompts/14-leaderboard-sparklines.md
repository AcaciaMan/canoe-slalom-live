# Prompt 14 — Leaderboard Penalty Sparklines

## Context

You are working on **canoe-slalom-live**, a Go web app for canoe/water slalom competitions. Stack: Go `net/http` (Go 1.22+), `html/template`, SQLite via `github.com/mattn/go-sqlite3`.

Read `PLAN.md` (Phase 3 — "Wow" details: run comparison sparkline). Previous Phase 3 prompts:
- **Prompt 11**: Sponsors schema and display.
- **Prompt 12**: Photo gallery page.
- **Prompt 13**: Commentator view with auto-refresh.

**Current leaderboard state:**
- `templates/leaderboard_partial.html` renders `{{define "leaderboard-tables"}}` used by full page and AJAX partial.
- Each row in the leaderboard table has: Rank | Bib | Athlete (with time-behind) | Nation | Run 1 | Run 2 | Best Time.
- `store.LeaderboardRow` has `Run1 *RunResult`, `Run2 *RunResult`, each with `RawTimeMs`, `PenaltyTouches`, `PenaltyMisses`, `PenaltySeconds`, `TotalTimeMs`, `Status`.
- `Run1IsBest`/`Run2IsBest` booleans mark which run is the athlete's best.
- Penalty CSS classes exist: `.penalty-clean` (green), `.penalty-touch` (amber), `.penalty-miss` (red).
- Template function `penaltyClass` returns the correct class for a `*RunResult`.

**Current Run 1 / Run 2 column rendering (in `leaderboard_partial.html`):**
```html
<td class="col-run">
    {{if .Run1}}
        {{if eq .Run1.Status "ok"}}
            {{if .Run1IsNew}}<span class="badge-new">NEW</span>{{end}}
            <span class="run-total {{if .Run1IsBest}}run-best{{end}}">{{formatTime .Run1.TotalTimeMs}}</span>
            {{if gt .Run1.PenaltySeconds 0}}<span class="run-penalty {{penaltyClass .Run1}}">+{{.Run1.PenaltySeconds}}s</span>{{end}}
        {{else}}<span class="status-dnf">{{.Run1.Status}}</span>{{end}}
    {{else}}<span class="no-runs">—</span>{{end}}
</td>
```

**Existing template functions in `main.go` funcMap:**
```go
funcMap := template.FuncMap{
    "formatTime": domain.FormatTime,
    "penaltyClass": func(r *store.RunResult) string { ... },
}
```

## Goal

Add a "wow" visual detail to the leaderboard: a tiny inline CSS sparkline bar for each run that visually shows the proportion of raw time vs penalty time. This gives spectators an instant visual sense of who was fast-but-sloppy vs slow-but-clean. Pure CSS — no JS charting library, no canvas, no SVG. Just HTML `<span>` elements with percentage widths inside a fixed-width container.

## What to build

### 1. Template functions for sparkline — `main.go`

Add two new template functions to the `funcMap` in `main.go`:

```go
// sparkRawPct computes the percentage of total time that is raw time.
// Returns an int from 0 to 100. Used for CSS sparkline width.
"sparkRawPct": func(r *store.RunResult) int {
    if r == nil || r.TotalTimeMs <= 0 {
        return 100
    }
    pct := (r.RawTimeMs * 100) / r.TotalTimeMs
    if pct > 100 {
        pct = 100
    }
    if pct < 1 {
        pct = 1
    }
    return pct
},

// sparkPenPct computes the percentage of total time that is penalty time.
"sparkPenPct": func(r *store.RunResult) int {
    if r == nil || r.TotalTimeMs <= 0 || r.PenaltySeconds <= 0 {
        return 0
    }
    penMs := r.PenaltySeconds * 1000
    pct := (penMs * 100) / r.TotalTimeMs
    if pct < 1 {
        pct = 1
    }
    if pct > 99 {
        pct = 99
    }
    return pct
},
```

### 2. Update leaderboard partial template — `templates/leaderboard_partial.html`

Add a sparkline bar below each run's time display in the Run 1 and Run 2 columns. The sparkline should appear only for runs with status "ok".

**Updated Run column rendering (for both Run 1 and Run 2):**

Replace the existing Run 1 `<td>` with:

```html
<td class="col-run">
    {{if .Run1}}
        {{if eq .Run1.Status "ok"}}
            {{if .Run1IsNew}}<span class="badge-new">NEW</span>{{end}}
            <span class="run-total {{if .Run1IsBest}}run-best{{end}}">{{formatTime .Run1.TotalTimeMs}}</span>
            {{if gt .Run1.PenaltySeconds 0}}<span class="run-penalty {{penaltyClass .Run1}}">+{{.Run1.PenaltySeconds}}s</span>{{end}}
            <div class="sparkline" title="Raw: {{formatTime .Run1.RawTimeMs}} | Penalties: +{{.Run1.PenaltySeconds}}s">
                <span class="spark-raw" style="width: {{sparkRawPct .Run1}}%"></span>
                {{if gt .Run1.PenaltySeconds 0}}<span class="spark-penalty {{penaltyClass .Run1}}" style="width: {{sparkPenPct .Run1}}%"></span>{{end}}
            </div>
        {{else}}<span class="status-dnf">{{.Run1.Status}}</span>{{end}}
    {{else}}<span class="no-runs">—</span>{{end}}
</td>
```

Do the same for Run 2 (replace `.Run1` with `.Run2`, `.Run1IsBest` with `.Run2IsBest`, `.Run1IsNew` with `.Run2IsNew`):

```html
<td class="col-run">
    {{if .Run2}}
        {{if eq .Run2.Status "ok"}}
            {{if .Run2IsNew}}<span class="badge-new">NEW</span>{{end}}
            <span class="run-total {{if .Run2IsBest}}run-best{{end}}">{{formatTime .Run2.TotalTimeMs}}</span>
            {{if gt .Run2.PenaltySeconds 0}}<span class="run-penalty {{penaltyClass .Run2}}">+{{.Run2.PenaltySeconds}}s</span>{{end}}
            <div class="sparkline" title="Raw: {{formatTime .Run2.RawTimeMs}} | Penalties: +{{.Run2.PenaltySeconds}}s">
                <span class="spark-raw" style="width: {{sparkRawPct .Run2}}%"></span>
                {{if gt .Run2.PenaltySeconds 0}}<span class="spark-penalty {{penaltyClass .Run2}}" style="width: {{sparkPenPct .Run2}}%"></span>{{end}}
            </div>
        {{else}}<span class="status-dnf">{{.Run2.Status}}</span>{{end}}
    {{else}}<span class="no-runs">—</span>{{end}}
</td>
```

### 3. CSS — `static/style.css`

Add sparkline styling:

```css
/* === Sparkline Bars === */
.sparkline {
    display: flex;
    height: 4px;
    width: 100%;
    max-width: 120px;
    margin-top: 3px;
    border-radius: 2px;
    overflow: hidden;
    background: #f3f4f6;
}

.spark-raw {
    display: block;
    height: 100%;
    background: #1e3a5f;
    border-radius: 2px 0 0 2px;
}

.spark-penalty {
    display: block;
    height: 100%;
}

/* Sparkline penalty color matches existing penalty classes */
.spark-penalty.penalty-touch {
    background: #f59e0b;
}
.spark-penalty.penalty-miss {
    background: #dc2626;
}
.spark-penalty.penalty-clean {
    background: #059669;
}
```

**Design rationale:**
- The sparkline is a 4px high bar placed below the time text.
- The dark navy segment (`.spark-raw`) represents raw time proportion.
- The colored segment represents penalty time: amber for touches, red for misses.
- A clean run shows a full navy bar (100% raw, 0% penalty).
- A run with a 50-second missed gate penalty will show a dramatic red segment — this is the visual "wow" that instantly communicates "that hurt."
- Max width is 120px — fits neatly under the time text in the column.
- The `title` attribute on `.sparkline` gives a tooltip with the raw/penalty breakdown on hover.
- On mobile, the sparkline scales with the column width.

### 4. Verify sparkline proportions with seed data

Looking at the seed data for visual validation:

| Athlete | Run | RawTimeMs | PenaltySeconds | TotalTimeMs | Raw % | Penalty % | Expected visual |
|---------|-----|-----------|----------------|-------------|-------|-----------|-----------------|
| Jan Rohan | R1 | 92,450 | 0 | 92,450 | 100% | 0% | Full navy bar |
| Jan Rohan | R2 | 89,710 | 4 | 93,710 | 96% | 4% | Navy bar with tiny amber sliver |
| Mathieu Deschamps | R1 | 88,120 | 50 | 138,120 | 64% | 36% | Navy 64% + large red 36% — dramatic! |
| Felix Brauer | R1 | 93,880 | 6 | 99,880 | 94% | 6% | Navy with small amber segment |
| Anna Březinová | R1 | 97,650 | 50 | 147,650 | 66% | 34% | Navy 66% + large red 34% |
| Anna Březinová | R2 | 103,440 | 8 | 111,440 | 93% | 7% | Navy with small amber segment |

The 50-second misses (Deschamps R1, Březinová R1) will create visually striking red segments — exactly the impact we want. Clean runs are a solid navy bar. Touches create small amber marks.

### 5. Optional: sparkline on commentator view

If desired, you can also add a sparkline to the commentator view's run result display. This is optional but would look great on the big-screen projection:

In `templates/commentator_partial.html`, after the time breakdown div inside `.commentator-run-result`, add:

```html
{{if .LatestRun}}
<div class="commentator-sparkline">
    <div class="sparkline sparkline-large">
        <span class="spark-raw" style="width: {{sparkRawPct .LatestRun}}%"></span>
        {{if gt .LatestRun.PenaltySeconds 0}}
        <span class="spark-penalty {{if gt .LatestRun.PenaltyMisses 0}}penalty-miss{{else}}penalty-touch{{end}}" style="width: {{sparkPenPct .LatestRun}}%"></span>
        {{end}}
    </div>
</div>
{{end}}
```

**Note:** For this to work, add template functions `sparkRawPct` and `sparkPenPct` that also accept `*store.LatestRunDetail`. The simplest approach is to create a helper interface or adapt the functions:

```go
"sparkRawPct": func(v interface{}) int {
    switch r := v.(type) {
    case *store.RunResult:
        if r == nil || r.TotalTimeMs <= 0 { return 100 }
        pct := (r.RawTimeMs * 100) / r.TotalTimeMs
        if pct > 100 { pct = 100 }
        if pct < 1 { pct = 1 }
        return pct
    case *store.LatestRunDetail:
        if r == nil || r.TotalTimeMs <= 0 { return 100 }
        pct := (r.RawTimeMs * 100) / r.TotalTimeMs
        if pct > 100 { pct = 100 }
        if pct < 1 { pct = 1 }
        return pct
    default:
        return 100
    }
},
```

Do the same for `sparkPenPct`. Alternatively, keep it simple and only use the sparkline on the leaderboard — the commentator view already has a text breakdown.

Add CSS for the larger sparkline variant:
```css
.sparkline-large {
    height: 8px;
    max-width: 300px;
    margin: 1rem auto 0;
    border-radius: 4px;
}
```

## Verification

1. `go build ./...` — no errors.
2. Delete `data.db` and re-seed: `Remove-Item -Force data.db; go run main.go -seed`.
3. Open `http://localhost:8080/events/demo-slalom-2026/leaderboard`.
4. Each run cell now shows the time + an inline sparkline bar below it.
5. Clean runs (Jan Rohan R1, Liam Crawford both runs, Elena Martínez R1, Sophie Leclerc both runs) — show a full solid navy bar.
6. Runs with gate touches (Jan Rohan R2, Oliver Bennett R1, Felix Brauer R1, Katarzyna Nowak R1, Elena Martínez R2, Anna Březinová R2) — show navy + amber segment. The amber segment is small (2–8% of total).
7. Runs with missed gates (Mathieu Deschamps R1, Anna Březinová R1) — show navy + large red segment (~34–36% of total). This is visually dramatic — the "wow" detail.
8. Hover over a sparkline — tooltip shows "Raw: 1:28.12 | Penalties: +50s" breakdown.
9. The sparkline fits neatly within the column width and doesn't push other content.
10. Auto-refresh leaderboard — sparklines render correctly after AJAX refresh.
11. Mobile view — sparklines scale proportionally.
12. Athletes with no runs — show "—" as before, no sparkline.

## Files to modify

```
main.go                              (modify — add sparkRawPct, sparkPenPct template functions)
templates/leaderboard_partial.html   (modify — add sparkline divs to Run 1 and Run 2 columns)
static/style.css                     (modify — add sparkline CSS)
```

## Important notes

- This is purely a CSS visual — no JavaScript, no charting library, no SVG. Just `<span>` elements with `style="width: X%"` inside a flex container. Minimal footprint.
- The sparkline should NOT disturb the table layout. Keep it as an additive element below the time text, not replacing anything.
- The sparkline `title` attribute provides an accessible tooltip. Screen readers will see the time text above and can use the title for extra context.
- The 4px bar height is deliberate — thin enough to be a visual accent, not a chart. It's a "sparkline" not a "bar chart."
- Clean runs still get a sparkline (100% navy) — this makes the visual consistent and shows that clean runs have no penalty segment, which is also informative.
- The `max-width: 120px` keeps sparklines from becoming absurdly wide on desktop while still being visible.
- Make sure the template functions handle nil pointers gracefully (return 100 for raw%, 0 for penalty%).
