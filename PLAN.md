# Canoe Slalom Live — Technical Plan

## 1. High-Level Architecture

### Project Layout

```
canoe-slalom-live/
├── main.go                  # Entry point: starts HTTP server, opens SQLite
├── db/
│   ├── db.go                # Open/migrate SQLite, expose *sql.DB
│   └── migrations.sql       # Single SQL file with CREATE TABLE statements
├── domain/
│   ├── event.go             # Structs: Event, Category
│   ├── athlete.go           # Structs: Athlete, Entry (event+athlete link)
│   └── run.go               # Structs: Run (includes penalties, computed time)
├── store/
│   ├── events.go            # SQL queries for events/categories
│   ├── athletes.go          # SQL queries for athletes/entries
│   └── runs.go              # SQL queries for runs
├── handler/
│   ├── public.go            # Visitor-facing: event page, leaderboard, athlete profile
│   ├── admin.go             # Organizer: create event, manage categories, manage start list
│   └── judge.go             # Judge: record runs
├── templates/
│   ├── layout.html          # Shared shell (nav, head, footer)
│   ├── event.html           # Public event page (start list + leaderboard tabs)
│   ├── athlete.html         # Athlete profile / bio page
│   ├── leaderboard.html     # Leaderboard partial (also used for full page)
│   ├── admin_event.html     # Create/edit event form
│   ├── admin_startlist.html # Manage entries for an event
│   └── judge_run.html       # Record a run
├── static/
│   ├── style.css
│   └── app.js               # Minimal JS: auto-refresh, form helpers
└── README.md
```

### Key Structural Decisions

**Routing.** Use Go's `net/http` default mux (Go 1.22+ supports method+pattern routing natively: `GET /events/{slug}`). No framework needed. Group handlers into three files by audience: public, admin, judge.

**Templates.** `html/template` with a shared layout via `template.ParseFiles("layout.html", "page.html")`. Each page is a named `{{define "content"}}` block. Server-rendered HTML means zero build step, no JS bundler.

**Database.** Single SQLite file (`data.db`). Use `modernc.org/sqlite` (pure Go, no CGo) or `mattn/go-sqlite3` (CGo, faster). One `migrations.sql` file run at startup with `IF NOT EXISTS` guards — no migration framework for MVP.

**Store layer.** Each file in `store/` exposes plain functions taking `*sql.DB` and returning domain structs. No ORM. Queries are hand-written SQL strings. This keeps it transparent and easy to debug.

**IDs and slugs.** Every table gets an integer `id` primary key (SQLite `INTEGER PRIMARY KEY` gives auto-increment). Events also get a unique `slug` (text) used in URLs. All public URLs use slugs: `/events/prague-open-2026`. Internal/admin/judge URLs can use IDs where convenience matters.

**URL structure:**

| URL | Purpose |
|-----|---------|
| `GET /` | List of events (or redirect to single event for MVP) |
| `GET /events/{slug}` | Event page: start list + leaderboard |
| `GET /events/{slug}/leaderboard` | Leaderboard (standalone or partial for JS refresh) |
| `GET /events/{slug}/athletes/{id}` | Athlete profile within event context |
| `GET /admin/events/new` | Create event form |
| `POST /admin/events` | Submit new event |
| `GET /admin/events/{slug}/startlist` | Manage start list |
| `GET /judge/events/{slug}` | Judge run entry page |
| `POST /judge/events/{slug}/runs` | Submit a run |

**Admin protection (MVP).** A single token set via environment variable (`ADMIN_TOKEN`). Admin and judge URLs require `?token=xyz` query param or a cookie set after first visit with token. Not secure for production — sufficient for a weekend demo where you share the link with 2–3 judges.

---

## 2. Data Model (Initial Schema)

### Table: `events`

| Column | Type | Notes |
|--------|------|-------|
| `id` | INTEGER PK | Auto-increment |
| `slug` | TEXT UNIQUE | URL-safe identifier, e.g. `prague-open-2026` |
| `name` | TEXT | Display name: "Prague Open 2026" |
| `date` | TEXT | ISO date `2026-06-15` |
| `location` | TEXT | Venue name, e.g. "Troja Whitewater Course" |
| `status` | TEXT | `draft`, `active`, `finished` — controls visibility |
| `created_at` | TEXT | ISO timestamp |

### Table: `categories`

| Column | Type | Notes |
|--------|------|-------|
| `id` | INTEGER PK | |
| `event_id` | INTEGER FK → events | |
| `code` | TEXT | `K1M`, `K1W`, `C1M`, `C1W` — standard ICF codes |
| `name` | TEXT | "Kayak Men", "Canoe Women" |
| `sort_order` | INTEGER | Display ordering on event page |
| `num_runs` | INTEGER | How many runs count (default 2 for standard slalom) |

Why a separate table: different events can have different category sets, and later you'll want per-category settings (number of gates, run count, etc.).

### Table: `athletes`

| Column | Type | Notes |
|--------|------|-------|
| `id` | INTEGER PK | |
| `name` | TEXT | Full name |
| `club` | TEXT | Club or national federation |
| `nation` | TEXT | 3-letter country code (CZE, GBR, etc.) |
| `bio` | TEXT | Short "commentator facts" blurb |
| `photo_url` | TEXT | Optional URL to headshot image |
| `created_at` | TEXT | |

Athletes are global (not per-event). The same athlete can enter multiple events. This avoids duplication and maps to how the sport actually works — athletes belong to clubs and travel between competitions.

### Table: `entries`

Links athletes to events+categories with a start position.

| Column | Type | Notes |
|--------|------|-------|
| `id` | INTEGER PK | |
| `event_id` | INTEGER FK → events | |
| `category_id` | INTEGER FK → categories | |
| `athlete_id` | INTEGER FK → athletes | |
| `bib_number` | INTEGER | Bib/start number for this event |
| `start_position` | INTEGER | Order in start list |

UNIQUE constraint on `(event_id, athlete_id)` — one entry per athlete per event. An athlete enters one category per event (standard in slalom).

### Table: `runs`

One row per run attempt by an athlete.

| Column | Type | Notes |
|--------|------|-------|
| `id` | INTEGER PK | |
| `entry_id` | INTEGER FK → entries | |
| `run_number` | INTEGER | 1 or 2 (ties to `categories.num_runs`) |
| `raw_time_ms` | INTEGER | Raw time in milliseconds (avoids float rounding) |
| `penalty_touches` | INTEGER | Count of 2-second gate touches |
| `penalty_misses` | INTEGER | Count of 50-second missed gates |
| `penalty_seconds` | INTEGER | Computed: `touches * 2 + misses * 50` |
| `total_time_ms` | INTEGER | Computed: `raw_time_ms + penalty_seconds * 1000` |
| `status` | TEXT | `ok`, `dns` (did not start), `dnf` (did not finish), `dsq` (disqualified) |
| `judged_at` | TEXT | Timestamp when recorded |

UNIQUE constraint on `(entry_id, run_number)`.

**Why store computed fields?** `penalty_seconds` and `total_time_ms` are derived but storing them makes leaderboard queries trivial (`ORDER BY total_time_ms`). The handler computes them on insert — single source of truth is the touch/miss counts.

### Design-for-future notes (not built yet)

- **`sponsors` table**: `id, event_id, name, logo_url, tier, link_url`. Will add in Phase 3. The event page template already has a placeholder `<div>` where sponsor logos will go.
- **`photos` table**: `id, event_id, athlete_id, url, photographer_name, created_at`. Phase 3 gallery feature.
- **`users` table**: `id, email, password_hash, role`. Phase 4 proper auth. For now, the admin token avoids this entirely.

---

## 3. Phase 1 — Core Skeleton and Happy Path

**Goal:** Run one event end-to-end on localhost. Create event → add athletes → start list → judge records runs → leaderboard shows results.

### Build order (each step should be a working commit):

### Step 1: Database and seed data

- Write `migrations.sql` with all five tables.
- Write a seed script (`seed.sql` or a Go func) that inserts one event ("Demo Slalom 2026"), two categories (K1M, K1W), and 6–8 athletes with realistic names/clubs/bios.
- Auto-generate entries linking athletes to categories with bib numbers.
- **Outcome:** `data.db` exists and has data you can query with `sqlite3` CLI.

### Step 2: Public event page — start list

- `GET /events/{slug}` renders the event page.
- Shows event name, date, location at top.
- Shows start list grouped by category, each athlete row: bib, name, club, nation.
- Clicking an athlete name goes to `GET /events/{slug}/athletes/{id}` showing their bio and photo (if any).
- **Pages:** Event page, Athlete profile page.
- **Endpoints:** `GET /events/{slug}`, `GET /events/{slug}/athletes/{id}`.

### Step 3: Judge run entry

- `GET /judge/events/{slug}` shows the judge form.
- Form flow: select category → shows list of athletes in that category → pick athlete → enter run number (1 or 2) → enter raw time (MM:SS.ms format, parsed to milliseconds server side) → enter touch count and miss count → submit.
- `POST /judge/events/{slug}/runs` validates and inserts the run. Computes `penalty_seconds` and `total_time_ms` before insert.
- On success, redirects back to judge page with a flash message ("Run recorded: Athlete X — 98.45s + 4s penalties = 102.45s").
- **Pages:** Judge run form.
- **Endpoints:** `GET /judge/events/{slug}`, `POST /judge/events/{slug}/runs`.

### Step 4: Leaderboard

- `GET /events/{slug}/leaderboard` shows results grouped by category.
- Each row: rank, bib, athlete name, best run time (or both runs with best highlighted), penalty breakdown, total time.
- Ranking logic for MVP: rank by best single-run `total_time_ms`. Athletes with `dnf`/`dsq` sort to bottom. Athletes with no runs yet show as "—".
- Embed or link the leaderboard into the event page (tab or section below start list).
- **Pages:** Leaderboard (standalone page and embedded in event page).
- **Endpoints:** `GET /events/{slug}/leaderboard` (returns full page or HTML partial based on `Accept` header or query param `?partial=1` for JS refresh).

### Step 5: Admin — create event and manage start list

- `GET /admin/events/new` → form for event name, slug, date, location.
- `POST /admin/events` → creates event, redirects to start list management.
- `GET /admin/events/{slug}/startlist` → shows current entries, form to add athlete (select existing or create new inline), assign bib and category.
- `POST /admin/events/{slug}/entries` → adds an entry.
- **Pages:** New event form, Start list manager.
- **Endpoints:** `GET /admin/events/new`, `POST /admin/events`, `GET /admin/events/{slug}/startlist`, `POST /admin/events/{slug}/entries`, `POST /admin/athletes` (create new athlete).

### Navigation structure (MVP)

```
┌─────────────────────────────────────────────────────┐
│  Header: "Canoe Slalom Live"   [Events] [Admin▾]   │
├─────────────────────────────────────────────────────┤
│                                                     │
│  /events/{slug}                                     │
│  ┌─────────────┬──────────────┐                     │
│  │ Start List  │ Leaderboard  │  ← tab navigation   │
│  └─────────────┴──────────────┘                     │
│                                                     │
│  Athlete name links → /events/{slug}/athletes/{id}  │
│  Admin dropdown → /admin/events/new                 │
│                    /admin/events/{slug}/startlist    │
│                    /judge/events/{slug}              │
│                                                     │
└─────────────────────────────────────────────────────┘
```

---

## 4. Phase 2 — Improve UX and Robustness

**Goal:** Make it usable at an actual riverside event, not just a localhost demo.

### Leaderboard UX

- **Auto-refresh:** Add a small `app.js` snippet that fetches `GET /events/{slug}/leaderboard?partial=1` every 10 seconds and replaces the leaderboard `<div>` innerHTML. A `<meta>` tag or JS toggle lets the user pause refresh.
- **Visual polish:** Highlight the current leader. Show "NEW" badge on runs recorded in the last 60 seconds. Use color coding: green = clean run (0 penalties), yellow = touches, red = misses or DSQ.
- **Time formatting:** Always display as `MM:SS.xx` (e.g., `01:42.37`). Show penalty breakdown inline: `92.37 + 4 = 96.37`.

### Judge UI for outdoor use

- **Large tap targets:** Big buttons for common actions. Minimum 48×48px touch targets.
- **Penalty entry by tapping:** Instead of typing a number, show a gate-by-gate grid. Each gate is a button: tap once = touch (2s), tap twice = miss (50s), no tap = clean. This matches how judges actually work — they mark per-gate. For MVP, a simple `+` / `−` stepper for touches and misses is the realistic middle ground.
- **Confirmation step:** After entering all data, show a summary ("Athlete: Jan Rohan | Raw: 94.12 | Touches: 2 | Total: 98.12") with a big green "Confirm" button. Prevents fat-finger errors.
- **Offline resilience (light):** Wrap the `POST` in JS that retries on network failure and shows "Saved" / "Pending" status. Not full offline sync — just graceful retry.

### Safety and validation

- **Admin token:** Check `?token=` query param on all `/admin/*` and `/judge/*` routes. Set via `ADMIN_TOKEN` env var. If invalid, return 403. Store token in a session cookie after first valid access so the URL can be shared once and then bookmarked cleanly.
- **Input validation:** Server-side: raw time must be positive, touches/misses must be ≥ 0, run number must be 1 or 2 and not already recorded for that entry. Return the form with error messages on failure (standard POST-redirect-GET with flash messages).
- **Duplicate run protection:** UNIQUE constraint on `(entry_id, run_number)` catches this at DB level. Handler should catch the constraint error and show a clear message ("Run 1 already recorded for this athlete. Edit or delete it first.").
- **Run edit/delete:** Add `PUT /judge/events/{slug}/runs/{id}` and `DELETE /judge/events/{slug}/runs/{id}` behind admin token. Needed because judges make mistakes.

### Keeping codebase small

- Resist adding packages. Templates, `net/http`, `database/sql`, and one SQLite driver should be the only dependencies.
- Total Go code target: under 1,500 lines across all `.go` files at end of Phase 2. If it's growing past that, you're over-engineering.
- Keep all SQL in the `store/` files as string constants. No query builder.

---

## 5. Phase 3 — Community and Storytelling Features

**Goal:** Make the event page something you'd actually share on social media or show on a big screen at the venue.

### Richer athlete profiles

- **Schema addition:** Add `nationality_flag` (emoji or small image URL) to `athletes`. Add an `athlete_facts` table (`id, athlete_id, fact_text, sort_order`) for structured commentator facts that can be shown as bullet points.
- **Profile page improvements:** Show past results from previous events on the same instance. A query joining `entries → runs` across events gives a simple competition history.
- **Photo display:** Use the existing `photo_url` column. For MVP, these are external URLs (club websites, social media). A later phase could add upload.

### Commentator view

- **New page:** `GET /events/{slug}/commentator` — designed for a second screen or tablet at the commentary booth.
- Shows: the athlete currently on course (based on start order and which runs have been recorded), their bio/facts, their Run 1 result (if doing Run 2), and the current top 3.
- Big fonts, high contrast, auto-advances when a new run is recorded.
- **Implementation cost:** One new template, one new handler. Reuses existing queries. Realistic for one evening.

### Sponsor visibility

- **Schema addition:** `sponsors` table: `id, event_id, name, logo_url, website_url, tier` (values: `main`, `partner`, `supporter`).
- **Display:** Event page footer shows sponsor logos grouped by tier. Commentator view shows main sponsor logo in corner. Leaderboard page shows "powered by {main sponsor}" line.
- **Admin:** Simple form at `GET /admin/events/{slug}/sponsors` to add/remove sponsors. Or just seed them in SQL for hackathon speed.

### Photo gallery (basic)

- **Schema addition:** `photos` table: `id, event_id, athlete_id (nullable), photographer_name, image_url, caption, created_at`.
- **Page:** `GET /events/{slug}/photos` — grid of images linked to athletes. Click photo to see full size + photographer credit.
- **Admin upload:** For MVP, just a form where photographer pastes image URLs. Actual file upload is Phase 4.

### "Wow" details (pick one or two)

1. **Run comparison sparkline.** On the leaderboard, show a tiny inline bar for each athlete's run: green segment = raw time, red segment = penalty time. Gives an instant visual sense of who was fast-but-sloppy versus slow-but-clean. Implementable with pure CSS (two `<span>`s with percentage widths inside a fixed-width `<div>`). One evening of work.

2. **Head-to-head comparison.** `GET /events/{slug}/compare?a={id1}&b={id2}` — pick two athletes, see their runs side by side with differences highlighted. Simple table layout. Good for social media screenshots. Two hours of work (one handler, one template).

---

## 6. Phase 4 — Post-Hackathon Evolution

### Proper auth and roles

- Add a `users` table with `email`, `password_hash` (bcrypt), and `role` (`organizer`, `judge`, `photographer`, `admin`).
- Use session cookies (Go's `gorilla/sessions` or hand-rolled with `crypto/rand` + a `sessions` table in SQLite).
- Role-based middleware: wrap handler groups. `requireRole("judge")` checks session.
- **Migration path:** The `?token=` approach maps cleanly to this — replace token check with session check. No handler logic changes needed, only the auth middleware swap.

### Multi-event, multi-club support

- Events already have their own IDs and slugs, so multi-event works from day one.
- Add a `clubs` table and a `club_members` join table to formalize club→athlete relationships.
- Add an `organizations` table if you want multi-tenant (multiple organizing bodies). Each event gets an `organization_id`. This is a bigger architectural decision — delay it unless there's real demand.

### Advanced penalty rules / formats

- **Gate-level penalties:** Replace the simple `penalty_touches` / `penalty_misses` integers with a `gate_penalties` table: `id, run_id, gate_number, penalty_type` (clean/touch/miss). Allows per-gate analysis and matches ICF scoring sheets.
- **Different formats:** Add `format` field to `categories`: `best-of-2`, `combined`, `single-run`, `qualification+final`. The leaderboard query switches ranking logic based on format.
- **Why to delay:** MVP's simple touch/miss counts cover 90% of grassroots competitions. Gate-level tracking is needed for national-level events but adds significant judge UI complexity.

### Design decisions to protect now

| Decision | Why it matters later |
|----------|---------------------|
| Athletes are global, not per-event | Enables cross-event history without migration |
| Times stored in milliseconds as integers | Avoids float precision issues when you add split times |
| Entries table separates athlete from event | Clean many-to-many; adding qualification rounds = new entry rows |
| Slugs on events | URLs stay stable when you add organizations or seasons |
| Computed `total_time_ms` stored in runs | Leaderboard queries stay fast even with 1000+ runs; recomputable |
| Templates use `{{define "content"}}` blocks | Adding new layouts (e.g., TV overlay, mobile) = new layout files, same content blocks |

---

## 7. Risk and Scope Check

### Absolutely essential for "feels real" demo

These are non-negotiable for a demo that impresses someone from the canoe slalom community:

1. **Seed data with realistic athletes, clubs, and nations.** Use real-sounding names and actual club names. An empty app is unconvincing.
2. **Judge flow works end-to-end.** Enter a run → leaderboard updates → rank changes visible. This is the core loop. If this doesn't work smoothly, nothing else matters.
3. **Leaderboard shows correct ranking with penalties.** The penalty calculation (2s per touch, 50s per miss) must be correct. Anyone from the sport will spot wrong math instantly.
4. **Time display in MM:SS.xx format.** The community reads times this way. Showing seconds-only or wrong precision breaks immersion.

### Nice-to-have but cuttable

| Feature | Cut impact |
|---------|------------|
| Admin create-event form | Seed via SQL instead. Saves 2–3 hours. |
| Athlete profile page | Show info inline on start list. Saves 1–2 hours. |
| Auto-refresh leaderboard | Manual browser refresh works. Saves 1 hour. |
| Multiple categories per event | Seed one category only. Saves complexity in judge form. |
| DNS/DNF/DSQ status handling | Assume all athletes complete their runs. Saves edge-case handling. |

### Fallback: Minimum Demo Flow (guaranteed finish in one weekend)

If time is tight, build exactly this and nothing else:

**Saturday:**
1. SQLite schema + seed script with one event, one category (K1M), 8 athletes.
2. `GET /events/{slug}` — shows start list (name, bib, club, nation). Server-rendered, static HTML.
3. `GET /judge/events/{slug}` — form with dropdowns (athlete, run number) and inputs (raw time, touches, misses). Posts to server.
4. `POST /judge/events/{slug}/runs` — saves run, redirects back to judge page with success message.

**Sunday:**
5. `GET /events/{slug}/leaderboard` — ranked list showing best run per athlete, penalty breakdown, total time.
6. CSS pass: make it look clean on mobile (judge will use a phone). Use a system font stack, simple table layout, and green/red penalty coloring.
7. Seed 10–15 runs with varied times and penalties so the leaderboard looks populated in screenshots.

**That's it.** Three pages, two POST endpoints, one database. Fits in ~600 lines of Go and ~200 lines of HTML/CSS. Demonstrably understands the sport, and everything built extends cleanly into the full plan above.

### Time budget estimate

| Block | Estimated time |
|-------|---------------|
| Schema + seed + DB layer | 2–3 hours |
| Public event page + start list | 2 hours |
| Judge form + run recording | 3 hours |
| Leaderboard with ranking | 2–3 hours |
| CSS/mobile polish | 2 hours |
| Admin event/startlist forms | 3 hours (cut if needed) |
| Auto-refresh + JS | 1 hour |
| Testing + bug fixing | 2 hours |
| **Total** | **17–19 hours** |

The fallback flow is roughly **11–13 hours** — achievable in a focused weekend.
