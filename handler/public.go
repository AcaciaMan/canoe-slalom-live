package handler

import (
	"database/sql"
	"html/template"
	"log"
	"net/http"
	"strconv"

	"canoe-slalom-live/domain"
	"canoe-slalom-live/store"
)

// Deps holds shared dependencies for handlers.
type Deps struct {
	DB         *sql.DB
	Tmpls      map[string]*template.Template
	AdminToken string
	Sessions   *SessionStore
}

// CategoryWithEntries pairs a category with its start list entries.
type CategoryWithEntries struct {
	Category domain.Category
	Entries  []store.EntryWithAthlete
}

// EventPageData is the template data for the event page.
type EventPageData struct {
	Event      domain.Event
	Categories []CategoryWithEntries
	Title      string
}

// AthletePageData is the template data for the athlete profile page.
type AthletePageData struct {
	Event   domain.Event
	Athlete domain.Athlete
	Entry   store.EntryWithAthlete
	Runs    []domain.Run
	Title   string
}

// CategoryLeaderboard pairs a category with its ranked leaderboard rows.
type CategoryLeaderboard struct {
	Category domain.Category
	Rows     []store.LeaderboardRow
}

// LeaderboardPageData is the template data for the leaderboard page.
type LeaderboardPageData struct {
	Event      domain.Event
	Categories []CategoryLeaderboard
	Title      string
}

// EventPage handles GET /events/{slug}.
func (d *Deps) EventPage(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")

	event, err := store.GetEventBySlug(d.DB, slug)
	if err == sql.ErrNoRows {
		d.renderError(w, 404, "Event not found")
		return
	}
	if err != nil {
		log.Printf("Error fetching event: %v", err)
		d.renderError(w, 500, "Internal server error")
		return
	}

	cats, err := store.ListCategories(d.DB, event.ID)
	if err != nil {
		log.Printf("Error fetching categories: %v", err)
		d.renderError(w, 500, "Internal server error")
		return
	}

	var catsWithEntries []CategoryWithEntries
	for _, cat := range cats {
		entries, err := store.ListEntriesByCategory(d.DB, cat.ID)
		if err != nil {
			log.Printf("Error fetching entries for category %d: %v", cat.ID, err)
			d.renderError(w, 500, "Internal server error")
			return
		}
		catsWithEntries = append(catsWithEntries, CategoryWithEntries{
			Category: cat,
			Entries:  entries,
		})
	}

	data := EventPageData{
		Event:      event,
		Categories: catsWithEntries,
		Title:      event.Name + " — Canoe Slalom Live",
	}

	if err := d.Tmpls["event"].ExecuteTemplate(w, "layout.html", data); err != nil {
		log.Printf("Error rendering event page: %v", err)
	}
}

// AthletePage handles GET /events/{slug}/athletes/{id}.
func (d *Deps) AthletePage(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	idStr := r.PathValue("id")

	athleteID, err := strconv.Atoi(idStr)
	if err != nil {
		d.renderError(w, 400, "Invalid athlete ID")
		return
	}

	event, err := store.GetEventBySlug(d.DB, slug)
	if err == sql.ErrNoRows {
		d.renderError(w, 404, "Event not found")
		return
	}
	if err != nil {
		log.Printf("Error fetching event: %v", err)
		d.renderError(w, 500, "Internal server error")
		return
	}

	athlete, err := store.GetAthlete(d.DB, athleteID)
	if err == sql.ErrNoRows {
		d.renderError(w, 404, "Athlete not found")
		return
	}
	if err != nil {
		log.Printf("Error fetching athlete: %v", err)
		d.renderError(w, 500, "Internal server error")
		return
	}

	entry, err := store.GetEntryByEventAndAthlete(d.DB, event.ID, athleteID)
	if err != nil && err != sql.ErrNoRows {
		log.Printf("Error fetching entry: %v", err)
		d.renderError(w, 500, "Internal server error")
		return
	}

	var runs []domain.Run
	if err == nil {
		runs, err = store.ListRunsByEntry(d.DB, entry.EntryID)
		if err != nil {
			log.Printf("Error fetching runs: %v", err)
			d.renderError(w, 500, "Internal server error")
			return
		}
	}

	data := AthletePageData{
		Event:   event,
		Athlete: athlete,
		Entry:   entry,
		Runs:    runs,
		Title:   athlete.Name + " — " + event.Name + " — Canoe Slalom Live",
	}

	if err := d.Tmpls["athlete"].ExecuteTemplate(w, "layout.html", data); err != nil {
		log.Printf("Error rendering athlete page: %v", err)
	}
}

// LeaderboardPage handles GET /events/{slug}/leaderboard.
func (d *Deps) LeaderboardPage(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")

	event, err := store.GetEventBySlug(d.DB, slug)
	if err == sql.ErrNoRows {
		d.renderError(w, 404, "Event not found")
		return
	}
	if err != nil {
		log.Printf("Error fetching event: %v", err)
		d.renderError(w, 500, "Internal server error")
		return
	}

	cats, err := store.ListCategories(d.DB, event.ID)
	if err != nil {
		log.Printf("Error fetching categories: %v", err)
		d.renderError(w, 500, "Internal server error")
		return
	}

	var catLeaderboards []CategoryLeaderboard
	for _, cat := range cats {
		rows, err := store.GetLeaderboard(d.DB, cat.ID)
		if err != nil {
			log.Printf("Error fetching leaderboard for category %d: %v", cat.ID, err)
			d.renderError(w, 500, "Internal server error")
			return
		}
		catLeaderboards = append(catLeaderboards, CategoryLeaderboard{
			Category: cat,
			Rows:     rows,
		})
	}

	data := LeaderboardPageData{
		Event:      event,
		Categories: catLeaderboards,
		Title:      "Leaderboard — " + event.Name + " — Canoe Slalom Live",
	}

	// Support partial rendering for AJAX refresh
	if r.URL.Query().Get("partial") == "1" {
		if err := d.Tmpls["leaderboard_partial"].ExecuteTemplate(w, "leaderboard-tables", data); err != nil {
			log.Printf("Error rendering leaderboard partial: %v", err)
		}
		return
	}

	if err := d.Tmpls["leaderboard"].ExecuteTemplate(w, "layout.html", data); err != nil {
		log.Printf("Error rendering leaderboard page: %v", err)
	}
}
