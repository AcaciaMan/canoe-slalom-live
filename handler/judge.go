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

// JudgeAthleteEntry enriches an entry with run status for the judge UI.
type JudgeAthleteEntry struct {
	store.EntryWithAthlete
	HasRun1  bool
	HasRun2  bool
	Run1Time string // formatted total time if exists, e.g. "1:34.37"
	Run2Time string // formatted total time if exists
}

// JudgeCategoryWithEntries pairs a category with enriched entries for the judge page.
type JudgeCategoryWithEntries struct {
	Category domain.Category
	Entries  []JudgeAthleteEntry
}

// JudgePageData is the template data for the judge run entry page.
type JudgePageData struct {
	Event       domain.Event
	Categories  []JudgeCategoryWithEntries
	RecentRuns  []store.RecentRun
	Success     string
	Error       string
	Title       string
	SelectedCat int // category ID to pre-select (from query param)
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

	var catsWithEntries []JudgeCategoryWithEntries
	for _, cat := range cats {
		entries, err := store.ListEntriesByCategory(d.DB, cat.ID)
		if err != nil {
			log.Printf("Error fetching entries for category %d: %v", cat.ID, err)
			d.renderError(w, 500, "Internal server error")
			return
		}

		var enriched []JudgeAthleteEntry
		for _, entry := range entries {
			jae := JudgeAthleteEntry{EntryWithAthlete: entry}
			runs, err := store.ListRunsByEntry(d.DB, entry.EntryID)
			if err != nil {
				log.Printf("Error fetching runs for entry %d: %v", entry.EntryID, err)
			} else {
				for _, run := range runs {
					if run.RunNumber == 1 {
						jae.HasRun1 = true
						jae.Run1Time = run.TotalTimeFormatted()
					}
					if run.RunNumber == 2 {
						jae.HasRun2 = true
						jae.Run2Time = run.TotalTimeFormatted()
					}
				}
			}
			enriched = append(enriched, jae)
		}

		catsWithEntries = append(catsWithEntries, JudgeCategoryWithEntries{
			Category: cat,
			Entries:  enriched,
		})
	}

	// Parse selected category from query param
	selectedCat, _ := strconv.Atoi(r.URL.Query().Get("cat"))

	// Fetch recent runs
	recentRuns, err := store.ListRecentRuns(d.DB, event.ID, 5)
	if err != nil {
		log.Printf("Error fetching recent runs: %v", err)
	}

	data := JudgePageData{
		Event:       event,
		Categories:  catsWithEntries,
		RecentRuns:  recentRuns,
		Success:     r.URL.Query().Get("success"),
		Error:       r.URL.Query().Get("error"),
		Title:       "Judge Panel — " + event.Name,
		SelectedCat: selectedCat,
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
	if r.FormValue("entry_id") == "" {
		redirectWithError("Please select an athlete")
		return
	}
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
	if r.FormValue("run_number") == "" {
		redirectWithError("Please select a run number")
		return
	}
	runNumber, err := strconv.Atoi(r.FormValue("run_number"))
	if err != nil || (runNumber != 1 && runNumber != 2) {
		redirectWithError("Run number must be 1 or 2")
		return
	}

	// Validate raw_time
	rawTimeStr := strings.TrimSpace(r.FormValue("raw_time"))
	if rawTimeStr == "" {
		redirectWithError("Please enter the raw time")
		return
	}
	rawTime, err := strconv.ParseFloat(rawTimeStr, 64)
	if err != nil || rawTime <= 0 {
		redirectWithError("Raw time must be a positive number (e.g. 94.37)")
		return
	}
	if rawTime < 30.0 || rawTime > 999.99 {
		redirectWithError("Raw time must be between 30.00 and 999.99 seconds")
		return
	}
	rawTimeMs := int(rawTime * 1000)

	// Validate touches
	touchesStr := strings.TrimSpace(r.FormValue("touches"))
	touches, err := strconv.Atoi(touchesStr)
	if err != nil || touches < 0 {
		redirectWithError("Touches must be 0 or more")
		return
	}
	if touches > 50 {
		redirectWithError("Gate touches seems too high (max 50)")
		return
	}

	// Validate misses
	missesStr := strings.TrimSpace(r.FormValue("misses"))
	misses, err := strconv.Atoi(missesStr)
	if err != nil || misses < 0 {
		redirectWithError("Misses must be 0 or more")
		return
	}
	if misses > 25 {
		redirectWithError("Missed gates seems too high (max 25)")
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
	http.Redirect(w, r, fmt.Sprintf("/judge/events/%s?success=%s&cat=%d", slug, url.QueryEscape(msg), entry.CategoryID), http.StatusSeeOther)
}

// EditRunPageData is the template data for the edit run page.
type EditRunPageData struct {
	Event      domain.Event
	Run        domain.Run
	Entry      store.EntryWithAthlete
	RawTimeSec string
	Title      string
	Error      string
}

// EditRunPage handles GET /judge/events/{slug}/runs/{id}/edit.
func (d *Deps) EditRunPage(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	runID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		d.renderError(w, 400, "Invalid run ID")
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

	run, err := store.GetRunByID(d.DB, runID)
	if err == sql.ErrNoRows {
		d.renderError(w, 404, "Run not found")
		return
	}
	if err != nil {
		log.Printf("Error fetching run: %v", err)
		d.renderError(w, 500, "Internal server error")
		return
	}

	entry, err := store.GetEntryByID(d.DB, run.EntryID)
	if err != nil {
		log.Printf("Error fetching entry: %v", err)
		d.renderError(w, 500, "Internal server error")
		return
	}
	if entry.EventID != event.ID {
		d.renderError(w, 403, "Run does not belong to this event")
		return
	}

	data := EditRunPageData{
		Event:      event,
		Run:        run,
		Entry:      entry,
		RawTimeSec: fmt.Sprintf("%.2f", float64(run.RawTimeMs)/1000.0),
		Title:      "Edit Run — " + event.Name,
		Error:      r.URL.Query().Get("error"),
	}

	if err := d.Tmpls["judge_edit"].ExecuteTemplate(w, "layout.html", data); err != nil {
		log.Printf("Error rendering edit run page: %v", err)
	}
}

// UpdateRunHandler handles POST /judge/events/{slug}/runs/{id}.
func (d *Deps) UpdateRunHandler(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	runID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		d.renderError(w, 400, "Invalid run ID")
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

	run, err := store.GetRunByID(d.DB, runID)
	if err == sql.ErrNoRows {
		d.renderError(w, 404, "Run not found")
		return
	}
	if err != nil {
		log.Printf("Error fetching run: %v", err)
		d.renderError(w, 500, "Internal server error")
		return
	}

	entry, err := store.GetEntryByID(d.DB, run.EntryID)
	if err != nil {
		log.Printf("Error fetching entry: %v", err)
		d.renderError(w, 500, "Internal server error")
		return
	}
	if entry.EventID != event.ID {
		d.renderError(w, 403, "Run does not belong to this event")
		return
	}

	redirectWithError := func(msg string) {
		http.Redirect(w, r, fmt.Sprintf("/judge/events/%s/runs/%d/edit?error=%s", slug, runID, url.QueryEscape(msg)), http.StatusSeeOther)
	}

	if err := r.ParseForm(); err != nil {
		redirectWithError("Invalid form data")
		return
	}

	// Validate raw_time
	rawTimeStr := strings.TrimSpace(r.FormValue("raw_time"))
	if rawTimeStr == "" {
		redirectWithError("Please enter the raw time")
		return
	}
	rawTime, err := strconv.ParseFloat(rawTimeStr, 64)
	if err != nil || rawTime <= 0 {
		redirectWithError("Raw time must be a positive number (e.g. 94.37)")
		return
	}
	if rawTime < 30.0 || rawTime > 999.99 {
		redirectWithError("Raw time must be between 30.00 and 999.99 seconds")
		return
	}
	rawTimeMs := int(rawTime * 1000)

	// Validate touches
	touchesStr := strings.TrimSpace(r.FormValue("touches"))
	touches, err := strconv.Atoi(touchesStr)
	if err != nil || touches < 0 {
		redirectWithError("Touches must be 0 or more")
		return
	}
	if touches > 50 {
		redirectWithError("Gate touches seems too high (max 50)")
		return
	}

	// Validate misses
	missesStr := strings.TrimSpace(r.FormValue("misses"))
	misses, err := strconv.Atoi(missesStr)
	if err != nil || misses < 0 {
		redirectWithError("Misses must be 0 or more")
		return
	}
	if misses > 25 {
		redirectWithError("Missed gates seems too high (max 25)")
		return
	}

	penaltySeconds := touches*2 + misses*50
	totalTimeMs := rawTimeMs + penaltySeconds*1000

	run.RawTimeMs = rawTimeMs
	run.PenaltyTouches = touches
	run.PenaltyMisses = misses
	run.PenaltySeconds = penaltySeconds
	run.TotalTimeMs = totalTimeMs
	run.Status = "ok"
	run.JudgedAt = time.Now().UTC().Format(time.RFC3339)

	if err := store.UpdateRun(d.DB, run); err != nil {
		log.Printf("Error updating run: %v", err)
		redirectWithError("Failed to update run")
		return
	}

	updatedRun := domain.Run{TotalTimeMs: totalTimeMs}
	msg := fmt.Sprintf("Run updated: %s Run %d — %s", entry.AthleteName, run.RunNumber, updatedRun.TotalTimeFormatted())
	http.Redirect(w, r, fmt.Sprintf("/judge/events/%s?success=%s&cat=%d", slug, url.QueryEscape(msg), entry.CategoryID), http.StatusSeeOther)
}

// DeleteRunHandler handles POST /judge/events/{slug}/runs/{id}/delete.
func (d *Deps) DeleteRunHandler(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	runID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		d.renderError(w, 400, "Invalid run ID")
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

	run, err := store.GetRunByID(d.DB, runID)
	if err == sql.ErrNoRows {
		d.renderError(w, 404, "Run not found")
		return
	}
	if err != nil {
		log.Printf("Error fetching run: %v", err)
		d.renderError(w, 500, "Internal server error")
		return
	}

	entry, err := store.GetEntryByID(d.DB, run.EntryID)
	if err != nil {
		log.Printf("Error fetching entry: %v", err)
		d.renderError(w, 500, "Internal server error")
		return
	}
	if entry.EventID != event.ID {
		d.renderError(w, 403, "Run does not belong to this event")
		return
	}

	if err := store.DeleteRun(d.DB, runID); err != nil {
		log.Printf("Error deleting run: %v", err)
		d.renderError(w, 500, "Failed to delete run")
		return
	}

	msg := fmt.Sprintf("Run deleted: %s Run %d", entry.AthleteName, run.RunNumber)
	http.Redirect(w, r, fmt.Sprintf("/judge/events/%s?success=%s&cat=%d", slug, url.QueryEscape(msg), entry.CategoryID), http.StatusSeeOther)
}
