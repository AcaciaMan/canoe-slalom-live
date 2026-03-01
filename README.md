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

To enable authentication for judge routes:

```bash
ADMIN_TOKEN=secret123 go run main.go -seed
# Judge panel: http://localhost:8080/judge/events/demo-slalom-2026?token=secret123
```

## Stack

- Go (net/http, html/template)
- SQLite (embedded, WAL mode for concurrent access)
- Vanilla HTML/CSS/JS — no build step

## Pages

- **Event page**: `/events/{slug}` — start list with athlete bios
- **Leaderboard**: `/events/{slug}/leaderboard` — live rankings with penalty breakdown
- **Judge panel**: `/judge/events/{slug}` — record runs with time + penalties
- **Athlete profile**: `/events/{slug}/athletes/{id}` — bio and run history

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
- Security headers (X-Content-Type-Options, X-Frame-Options, Referrer-Policy)
- Request logging for debugging during live events
