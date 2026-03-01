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
	Sponsors   []domain.Sponsor
	Title      string
}

// AthletePageData is the template data for the athlete profile page.
type AthletePageData struct {
	Event        domain.Event
	Athlete      domain.Athlete
	Entry        store.EntryWithAthlete
	Runs         []domain.Run
	Photos       []domain.Photo
	OtherEntries []store.EntryWithAthlete
	Title        string
}

// ComparePageData is the template data for the head-to-head comparison page.
type ComparePageData struct {
	Event    domain.Event
	AthleteA CompareAthlete
	AthleteB CompareAthlete
	Title    string
}

// CompareAthlete holds one side of the comparison.
type CompareAthlete struct {
	Athlete domain.Athlete
	Entry   store.EntryWithAthlete
	Run1    *domain.Run
	Run2    *domain.Run
	BestMs  int
}

// GalleryPageData is the template data for the photo gallery page.
type GalleryPageData struct {
	Event  domain.Event
	Photos []store.PhotoWithAthlete
	Title  string
}

// Top3Row is a simplified leaderboard row for the commentator's top-3 display.
type Top3Row struct {
	Rank            int
	BibNumber       int
	AthleteName     string
	AthleteNation   string
	BestTotalTimeMs int
}

// CategoryTop3 pairs a category with its top 3 athletes.
type CategoryTop3 struct {
	Category domain.Category
	Top3     []Top3Row
}

// CommentatorPageData is the template data for the commentator view.
type CommentatorPageData struct {
	Event       domain.Event
	LatestRun   *store.LatestRunDetail
	CatTop3     []CategoryTop3
	MainSponsor *domain.Sponsor
	Title       string
}

// CategoryLeaderboard pairs a category with its ranked leaderboard rows.
type CategoryLeaderboard struct {
	Category domain.Category
	Rows     []store.LeaderboardRow
}

// LeaderboardPageData is the template data for the leaderboard page.
type LeaderboardPageData struct {
	Event       domain.Event
	Categories  []CategoryLeaderboard
	MainSponsor *domain.Sponsor
	Title       string
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

	sponsors, err := store.ListSponsorsByEvent(d.DB, event.ID)
	if err != nil {
		log.Printf("Error fetching sponsors: %v", err)
		// Don't fail the page for sponsors — just leave empty
	}

	data := EventPageData{
		Event:      event,
		Categories: catsWithEntries,
		Sponsors:   sponsors,
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

	photos, err := store.ListPhotosByAthlete(d.DB, event.ID, athleteID)
	if err != nil {
		log.Printf("Error fetching athlete photos: %v", err)
		// Don't fail page — just leave photos empty
	}

	// Fetch other athletes in same category for compare dropdown
	var otherEntries []store.EntryWithAthlete
	if entry.CategoryID > 0 {
		allEntries, err := store.ListEntriesByCategory(d.DB, entry.CategoryID)
		if err != nil {
			log.Printf("Error fetching other entries: %v", err)
		} else {
			for _, oe := range allEntries {
				if oe.AthleteID != athleteID {
					otherEntries = append(otherEntries, oe)
				}
			}
		}
	}

	data := AthletePageData{
		Event:        event,
		Athlete:      athlete,
		Entry:        entry,
		Runs:         runs,
		Photos:       photos,
		OtherEntries: otherEntries,
		Title:        athlete.Name + " — " + event.Name + " — Canoe Slalom Live",
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

	mainSponsor, err := store.GetMainSponsor(d.DB, event.ID)
	if err == nil {
		data.MainSponsor = &mainSponsor
	}
	// sql.ErrNoRows is fine — just means no main sponsor

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

// GalleryPage handles GET /events/{slug}/photos.
func (d *Deps) GalleryPage(w http.ResponseWriter, r *http.Request) {
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

	photos, err := store.ListPhotosByEvent(d.DB, event.ID)
	if err != nil {
		log.Printf("Error fetching photos: %v", err)
		d.renderError(w, 500, "Internal server error")
		return
	}

	data := GalleryPageData{
		Event:  event,
		Photos: photos,
		Title:  "Photos — " + event.Name + " — Canoe Slalom Live",
	}

	if err := d.Tmpls["gallery"].ExecuteTemplate(w, "layout.html", data); err != nil {
		log.Printf("Error rendering gallery page: %v", err)
	}
}

// CommentatorPage handles GET /events/{slug}/commentator.
func (d *Deps) CommentatorPage(w http.ResponseWriter, r *http.Request) {
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

	// Fetch latest run
	var latestRun *store.LatestRunDetail
	lr, err := store.GetLatestRun(d.DB, event.ID)
	if err == nil {
		latestRun = &lr
	}
	// sql.ErrNoRows is fine — no runs recorded yet

	// Fetch top 3 per category
	cats, err := store.ListCategories(d.DB, event.ID)
	if err != nil {
		log.Printf("Error fetching categories: %v", err)
		d.renderError(w, 500, "Internal server error")
		return
	}

	var catTop3 []CategoryTop3
	for _, cat := range cats {
		rows, err := store.GetLeaderboard(d.DB, cat.ID)
		if err != nil {
			log.Printf("Error fetching leaderboard for category %d: %v", cat.ID, err)
			continue
		}
		var top3 []Top3Row
		for _, row := range rows {
			if row.Rank >= 1 && row.Rank <= 3 {
				top3 = append(top3, Top3Row{
					Rank:            row.Rank,
					BibNumber:       row.BibNumber,
					AthleteName:     row.AthleteName,
					AthleteNation:   row.AthleteNation,
					BestTotalTimeMs: row.BestTotalTimeMs,
				})
			}
		}
		catTop3 = append(catTop3, CategoryTop3{
			Category: cat,
			Top3:     top3,
		})
	}

	// Fetch main sponsor
	var mainSponsor *domain.Sponsor
	ms, err := store.GetMainSponsor(d.DB, event.ID)
	if err == nil {
		mainSponsor = &ms
	}

	data := CommentatorPageData{
		Event:       event,
		LatestRun:   latestRun,
		CatTop3:     catTop3,
		MainSponsor: mainSponsor,
		Title:       "Commentator — " + event.Name,
	}

	// Support partial refresh (AJAX)
	if r.URL.Query().Get("partial") == "1" {
		if err := d.Tmpls["commentator_partial"].ExecuteTemplate(w, "commentator-content", data); err != nil {
			log.Printf("Error rendering commentator partial: %v", err)
		}
		return
	}

	if err := d.Tmpls["commentator"].ExecuteTemplate(w, "layout.html", data); err != nil {
		log.Printf("Error rendering commentator page: %v", err)
	}
}

// ComparePage handles GET /events/{slug}/compare?a={id1}&b={id2}.
func (d *Deps) ComparePage(w http.ResponseWriter, r *http.Request) {
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

	aID, err := strconv.Atoi(r.URL.Query().Get("a"))
	if err != nil || aID <= 0 {
		d.renderError(w, 400, "Missing or invalid athlete A (?a= parameter)")
		return
	}
	bID, err := strconv.Atoi(r.URL.Query().Get("b"))
	if err != nil || bID <= 0 {
		d.renderError(w, 400, "Missing or invalid athlete B (?b= parameter)")
		return
	}
	if aID == bID {
		d.renderError(w, 400, "Cannot compare an athlete to themselves")
		return
	}

	loadAthlete := func(athleteID int) (CompareAthlete, error) {
		var ca CompareAthlete
		athlete, err := store.GetAthlete(d.DB, athleteID)
		if err != nil {
			return ca, err
		}
		ca.Athlete = athlete

		entry, err := store.GetEntryByEventAndAthlete(d.DB, event.ID, athleteID)
		if err != nil {
			return ca, err
		}
		ca.Entry = entry

		runs, err := store.ListRunsByEntry(d.DB, entry.EntryID)
		if err != nil {
			return ca, err
		}

		for i := range runs {
			if runs[i].RunNumber == 1 {
				ca.Run1 = &runs[i]
			}
			if runs[i].RunNumber == 2 {
				ca.Run2 = &runs[i]
			}
		}

		// Compute best time
		if ca.Run1 != nil && ca.Run1.Status == "ok" {
			ca.BestMs = ca.Run1.TotalTimeMs
		}
		if ca.Run2 != nil && ca.Run2.Status == "ok" {
			if ca.BestMs == 0 || ca.Run2.TotalTimeMs < ca.BestMs {
				ca.BestMs = ca.Run2.TotalTimeMs
			}
		}
		return ca, nil
	}

	athleteA, err := loadAthlete(aID)
	if err != nil {
		d.renderError(w, 404, "Athlete A not found or not entered in this event")
		return
	}
	athleteB, err := loadAthlete(bID)
	if err != nil {
		d.renderError(w, 404, "Athlete B not found or not entered in this event")
		return
	}

	data := ComparePageData{
		Event:    event,
		AthleteA: athleteA,
		AthleteB: athleteB,
		Title:    athleteA.Athlete.Name + " vs " + athleteB.Athlete.Name + " — " + event.Name,
	}

	if err := d.Tmpls["compare"].ExecuteTemplate(w, "layout.html", data); err != nil {
		log.Printf("Error rendering compare page: %v", err)
	}
}
