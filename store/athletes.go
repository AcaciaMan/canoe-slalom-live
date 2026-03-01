package store

import (
	"database/sql"

	"canoe-slalom-live/domain"
)

// EntryWithAthlete combines entry and athlete data for display purposes.
type EntryWithAthlete struct {
	EntryID       int
	EventID       int
	CategoryID    int
	BibNumber     int
	StartPosition int
	AthleteID     int
	AthleteName   string
	Club          string
	Nation        string
}

// GetAthlete returns a single athlete by ID.
func GetAthlete(db *sql.DB, id int) (domain.Athlete, error) {
	var a domain.Athlete
	err := db.QueryRow(
		`SELECT id, name, club, nation, bio, photo_url, created_at FROM athletes WHERE id = ?`,
		id,
	).Scan(&a.ID, &a.Name, &a.Club, &a.Nation, &a.Bio, &a.PhotoURL, &a.CreatedAt)
	return a, err
}

// ListEntriesByCategory returns entries joined with athlete data, ordered by start_position.
func ListEntriesByCategory(db *sql.DB, categoryID int) ([]EntryWithAthlete, error) {
	rows, err := db.Query(`
		SELECT e.id, e.event_id, e.category_id, e.bib_number, e.start_position, a.id, a.name, a.club, a.nation
		FROM entries e
		JOIN athletes a ON a.id = e.athlete_id
		WHERE e.category_id = ?
		ORDER BY e.start_position`,
		categoryID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []EntryWithAthlete
	for rows.Next() {
		var ea EntryWithAthlete
		if err := rows.Scan(&ea.EntryID, &ea.EventID, &ea.CategoryID, &ea.BibNumber, &ea.StartPosition, &ea.AthleteID, &ea.AthleteName, &ea.Club, &ea.Nation); err != nil {
			return nil, err
		}
		entries = append(entries, ea)
	}
	return entries, rows.Err()
}

// GetEntryByEventAndAthlete returns the entry for a specific athlete in a specific event.
func GetEntryByEventAndAthlete(db *sql.DB, eventID, athleteID int) (EntryWithAthlete, error) {
	var ea EntryWithAthlete
	err := db.QueryRow(`
		SELECT e.id, e.event_id, e.category_id, e.bib_number, e.start_position, a.id, a.name, a.club, a.nation
		FROM entries e
		JOIN athletes a ON a.id = e.athlete_id
		WHERE e.event_id = ? AND e.athlete_id = ?`,
		eventID, athleteID,
	).Scan(&ea.EntryID, &ea.EventID, &ea.CategoryID, &ea.BibNumber, &ea.StartPosition, &ea.AthleteID, &ea.AthleteName, &ea.Club, &ea.Nation)
	return ea, err
}
