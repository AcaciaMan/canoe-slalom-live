package handler

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"canoe-slalom-live/domain"
	"canoe-slalom-live/store"
)

// JudgePageData is the template data for the judge run entry page.
type JudgePageData struct {
	Event      domain.Event
	Categories []CategoryWithEntries
	Success    string
	Error      string
	Title      string
}

// JudgePage handles GET /judge/events/{slug}.
func (d *Deps) JudgePage(w http.ResponseWriter, r *http.Request) {
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

	data := JudgePageData{
		Event:      event,
		Categories: catsWithEntries,
		Success:    r.URL.Query().Get("success"),
		Error:      r.URL.Query().Get("error"),
		Title:      "Judge Panel — " + event.Name,
	}

	if err := d.Tmpls["judge"].ExecuteTemplate(w, "layout.html", data); err != nil {
		log.Printf("Error rendering judge page: %v", err)
	}
}

// SubmitRun handles POST /judge/events/{slug}/runs.
func (d *Deps) SubmitRun(w http.ResponseWriter, r *http.Request) {
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

	redirectWithError := func(msg string) {
		http.Redirect(w, r, fmt.Sprintf("/judge/events/%s?error=%s", slug, url.QueryEscape(msg)), http.StatusSeeOther)
	}

	// Parse form
	if err := r.ParseForm(); err != nil {
		redirectWithError("Invalid form data")
		return
	}

	// Validate entry_id
	entryID, err := strconv.Atoi(r.FormValue("entry_id"))
	if err != nil || entryID <= 0 {
		redirectWithError("Please select an athlete")
		return
	}

	// Validate entry belongs to this event
	entry, err := store.GetEntryByID(d.DB, entryID)
	if err == sql.ErrNoRows {
		redirectWithError("Invalid athlete selection")
		return
	}
	if err != nil {
		log.Printf("Error fetching entry: %v", err)
		redirectWithError("Server error")
		return
	}
	if entry.EventID != event.ID {
		redirectWithError("Athlete entry does not belong to this event")
		return
	}

	// Validate run_number
	runNumber, err := strconv.Atoi(r.FormValue("run_number"))
	if err != nil || (runNumber != 1 && runNumber != 2) {
		redirectWithError("Run number must be 1 or 2")
		return
	}

	// Validate raw_time
	rawTimeStr := strings.TrimSpace(r.FormValue("raw_time"))
	rawTime, err := strconv.ParseFloat(rawTimeStr, 64)
	if err != nil || rawTime <= 0 {
		redirectWithError("Raw time must be a positive number (e.g. 94.37)")
		return
	}
	rawTimeMs := int(rawTime * 1000)

	// Validate touches
	touches, err := strconv.Atoi(r.FormValue("touches"))
	if err != nil || touches < 0 {
		redirectWithError("Touches must be 0 or more")
		return
	}

	// Validate misses
	misses, err := strconv.Atoi(r.FormValue("misses"))
	if err != nil || misses < 0 {
		redirectWithError("Misses must be 0 or more")
		return
	}

	// Compute penalties
	penaltySeconds := touches*2 + misses*50
	totalTimeMs := rawTimeMs + penaltySeconds*1000

	run := domain.Run{
		EntryID:        entryID,
		RunNumber:      runNumber,
		RawTimeMs:      rawTimeMs,
		PenaltyTouches: touches,
		PenaltyMisses:  misses,
		PenaltySeconds: penaltySeconds,
		TotalTimeMs:    totalTimeMs,
		Status:         "ok",
		JudgedAt:       time.Now().UTC().Format(time.RFC3339),
	}

	_, err = store.CreateRun(d.DB, run)
	if err != nil {
		// Check for unique constraint violation (duplicate run)
		if strings.Contains(err.Error(), "UNIQUE constraint failed") || strings.Contains(err.Error(), "unique") {
			redirectWithError(fmt.Sprintf("Run %d already recorded for this athlete. Delete it first to re-enter.", runNumber))
			return
		}
		log.Printf("Error creating run: %v", err)
		redirectWithError("Failed to save run")
		return
	}

	// Format total time for success message
	successRun := domain.Run{TotalTimeMs: totalTimeMs}
	msg := fmt.Sprintf("Run recorded: %s — %s", entry.AthleteName, successRun.TotalTimeFormatted())
	http.Redirect(w, r, fmt.Sprintf("/judge/events/%s?success=%s", slug, url.QueryEscape(msg)), http.StatusSeeOther)
}
