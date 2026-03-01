package store

import (
	"database/sql"

	"canoe-slalom-live/domain"
)

// GetEventBySlug returns a single event matching the given slug.
func GetEventBySlug(db *sql.DB, slug string) (domain.Event, error) {
	var e domain.Event
	err := db.QueryRow(
		`SELECT id, slug, name, date, location, status, created_at FROM events WHERE slug = ?`,
		slug,
	).Scan(&e.ID, &e.Slug, &e.Name, &e.Date, &e.Location, &e.Status, &e.CreatedAt)
	return e, err
}

// ListCategories returns all categories for an event, ordered by sort_order.
func ListCategories(db *sql.DB, eventID int) ([]domain.Category, error) {
	rows, err := db.Query(
		`SELECT id, event_id, code, name, sort_order, num_runs FROM categories WHERE event_id = ? ORDER BY sort_order`,
		eventID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []domain.Category
	for rows.Next() {
		var c domain.Category
		if err := rows.Scan(&c.ID, &c.EventID, &c.Code, &c.Name, &c.SortOrder, &c.NumRuns); err != nil {
			return nil, err
		}
		categories = append(categories, c)
	}
	return categories, rows.Err()
}
