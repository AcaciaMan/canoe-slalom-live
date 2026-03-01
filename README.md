# 🛶 Canoe Slalom Live

Live timing and results for canoe/water slalom competitions. Built for grassroots events — run it on a laptop at the riverside, let spectators follow on their phones.

## Quick Start

```bash
# Prerequisites: Go 1.22+, GCC (for SQLite CGo driver)
go mod tidy
go run main.go -seed

# Opens at http://localhost:8080
# Judge panel at http://localhost:8080/judge/events/demo-slalom-2026
```

To enable authentication for judge routes:

```bash
# Windows
set ADMIN_TOKEN=secret123
go run main.go -seed

# Linux/macOS
ADMIN_TOKEN=secret123 go run main.go -seed

# Judge panel: http://localhost:8080/judge/events/demo-slalom-2026?token=secret123
```

## Stack

- **Go** — `net/http` with Go 1.22+ pattern routing, `html/template` for server-rendered HTML
- **SQLite** — embedded via `github.com/mattn/go-sqlite3` (CGo), WAL mode, single `data.db` file
- **Vanilla HTML/CSS/JS** — no build step, no framework, no bundler

## Pages

| Route | Page | Description |
|-------|------|-------------|
| `/events/{slug}` | Event | Start list grouped by category with athlete bios |
| `/events/{slug}/leaderboard` | Leaderboard | Live rankings with penalty sparklines and auto-refresh |
| `/events/{slug}/photos` | Photo Gallery | Grid of event and athlete action photos |
| `/events/{slug}/commentator` | Commentator View | Big-screen display for commentary booth |
| `/events/{slug}/compare?a={id}&b={id}` | Head-to-Head | Side-by-side athlete comparison |
| `/events/{slug}/athletes/{id}` | Athlete Profile | Bio, run history, photos, head-to-head links |
| `/judge/events/{slug}` | Judge Panel | Record runs with time + penalties (auth required) |
| `/judge/events/{slug}/runs/{id}/edit` | Edit Run | Correct mistakes on recorded runs (auth required) |

## Features

### For Spectators
- Live leaderboard with auto-refresh (10s polling via `?partial=1`)
- 🥇🥈🥉 Medal indicators for top 3
- "NEW" badges on runs recorded within the last 60 seconds
- Time-behind-leader display on each row
- CSS penalty sparklines — tiny inline bars showing raw time (navy) vs penalties (amber/red)
- Athlete profiles with bios, run history, and linked photos
- Photo gallery with photographer credits and athlete tagging
- Head-to-head comparison page with winner highlighting and time difference
- Sponsor logos on event page and "Powered by" on leaderboard

### For Commentators
- Dedicated commentator view at `/events/{slug}/commentator`
- Shows most recently judged athlete with bio, photo, and run result in large font
- Current top 3 per category with medal indicators
- Auto-refreshes every 5 seconds for near-real-time updates
- Designed for projection — big fonts, high contrast layout
- Main sponsor logo displayed in corner

### For Judges
- Mobile-friendly run entry with large tap targets (48×48px minimum)
- Confirmation step before saving to prevent fat-finger errors
- Penalty steppers (touch +2s / miss +50s counters)
- Run status indicators showing which athletes still need runs
- Edit and delete runs to correct mistakes
- Recent runs feed on judge page
- Category auto-selection after recording a run

### Security
- Admin token authentication via `ADMIN_TOKEN` environment variable
- Session cookies after first token use (no repeated URL token entry)
- Server-side input validation with bounds checking (time range, max penalties)
- Duplicate run protection via `UNIQUE(entry_id, run_number)` constraint
- Security headers (X-Content-Type-Options, X-Frame-Options, Referrer-Policy)
- Request logging with method, path, status code, and duration

## Project Structure

```
canoe-slalom-live/
├── main.go                    # Entry point, templates, routes, server
├── db/
│   ├── db.go                  # SQLite open/migrate/seed (go:embed)
│   ├── migrations.sql         # 7 tables: events, categories, athletes, entries, runs, sponsors, photos
│   └── seed.sql               # Demo data: 1 event, 2 categories, 10 athletes, runs, sponsors, photos
├── domain/
│   ├── event.go               # Event, Category structs
│   ├── athlete.go             # Athlete, Entry structs
│   ├── run.go                 # Run struct, time formatting helpers
│   ├── sponsor.go             # Sponsor struct
│   └── photo.go               # Photo struct
├── store/
│   ├── events.go              # Event/category queries
│   ├── athletes.go            # Athlete/entry queries
│   ├── runs.go                # Run CRUD, leaderboard ranking, latest run
│   ├── sponsors.go            # Sponsor queries by event/tier
│   └── photos.go              # Photo queries by event/athlete
├── handler/
│   ├── public.go              # Public handlers + Deps struct
│   ├── judge.go               # Judge form, run submit/edit/delete
│   ├── auth.go                # Token auth middleware, session store
│   ├── helpers.go             # renderError helper
│   └── logging.go             # Logging + security headers middleware
├── templates/                 # html/template files (12 templates)
│   ├── layout.html            # Shared shell (nav, footer)
│   ├── event.html             # Start list with tab bar
│   ├── leaderboard.html       # Leaderboard container
│   ├── leaderboard_partial.html # Leaderboard tables (AJAX-refreshable)
│   ├── athlete.html           # Athlete profile
│   ├── gallery.html           # Photo gallery grid
│   ├── commentator.html       # Commentator view container
│   ├── commentator_partial.html # Commentator content (AJAX-refreshable)
│   ├── compare.html           # Head-to-head comparison
│   ├── judge_run.html         # Judge run entry form
│   ├── judge_edit_run.html    # Edit existing run
│   └── error.html             # Styled error page
└── static/
    ├── style.css              # Full stylesheet (responsive, mobile-first)
    └── app.js                 # Auto-refresh for leaderboard + commentator
```

## Data Model

7 SQLite tables with foreign keys:

- **events** — competition info (slug, name, date, location, status)
- **categories** — competition classes per event (K1M, C1W, etc.)
- **athletes** — competitors (name, club, nation, bio, photo)
- **entries** — links athletes to events+categories with bib numbers
- **runs** — individual run attempts (raw time, penalties, computed total)
- **sponsors** — event sponsors with tier-based display (main/partner/supporter)
- **photos** — event and athlete photos with captions and photographer credits

## Scoring

Standard ICF canoe slalom rules:
- **Gate touch**: +2 seconds penalty
- **Missed gate**: +50 seconds penalty
- **Total time** = raw time + (touches × 2s) + (misses × 50s)
- Ranked by best single-run total time (2 runs per athlete)
- Equal times receive equal rank

## Development

```bash
# Build
go build ./...

# Run with demo data
go run main.go -seed

# Run with auth enabled
set ADMIN_TOKEN=mytoken
go run main.go

# Reset database
Remove-Item -Force data.db; go run main.go -seed
```

## Implementation Status

- [x] **Phase 1** — Core skeleton: database, start list, judge form, leaderboard
- [x] **Phase 2** — UX & robustness: auth, judge UI overhaul, run edit/delete, leaderboard polish, validation
- [x] **Phase 3** — Community & storytelling: sponsors, photo gallery, commentator view, penalty sparklines, head-to-head comparison
- [ ] **Phase 4** — Post-hackathon: proper auth/roles, multi-event, gate-level penalties

See [PLAN.md](PLAN.md) for the full technical plan.
