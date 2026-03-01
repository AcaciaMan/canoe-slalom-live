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
