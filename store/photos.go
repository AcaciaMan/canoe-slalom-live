package store

import (
	"database/sql"

	"canoe-slalom-live/domain"
)

// PhotoWithAthlete pairs a photo with the athlete's name (if linked).
type PhotoWithAthlete struct {
	domain.Photo
	AthleteName string // empty if no athlete linked
}

// ListPhotosByEvent returns all photos for an event, ordered by created_at DESC.
func ListPhotosByEvent(db *sql.DB, eventID int) ([]PhotoWithAthlete, error) {
	rows, err := db.Query(`
		SELECT p.id, p.event_id, COALESCE(p.athlete_id, 0), p.image_url, COALESCE(p.caption, ''),
		       COALESCE(p.photographer_name, ''), COALESCE(p.created_at, ''), COALESCE(a.name, '')
		FROM photos p
		LEFT JOIN athletes a ON a.id = p.athlete_id
		WHERE p.event_id = ?
		ORDER BY p.created_at DESC`,
		eventID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var photos []PhotoWithAthlete
	for rows.Next() {
		var pa PhotoWithAthlete
		if err := rows.Scan(&pa.ID, &pa.EventID, &pa.AthleteID,
			&pa.ImageURL, &pa.Caption, &pa.PhotographerName,
			&pa.CreatedAt, &pa.AthleteName); err != nil {
			return nil, err
		}
		photos = append(photos, pa)
	}
	return photos, rows.Err()
}

// ListPhotosByAthlete returns photos for a specific athlete in an event.
func ListPhotosByAthlete(db *sql.DB, eventID, athleteID int) ([]domain.Photo, error) {
	rows, err := db.Query(`
		SELECT id, event_id, athlete_id, image_url, COALESCE(caption, ''),
		       COALESCE(photographer_name, ''), COALESCE(created_at, '')
		FROM photos
		WHERE event_id = ? AND athlete_id = ?
		ORDER BY created_at DESC`,
		eventID, athleteID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var photos []domain.Photo
	for rows.Next() {
		var p domain.Photo
		if err := rows.Scan(&p.ID, &p.EventID, &p.AthleteID,
			&p.ImageURL, &p.Caption, &p.PhotographerName, &p.CreatedAt); err != nil {
			return nil, err
		}
		photos = append(photos, p)
	}
	return photos, rows.Err()
}
