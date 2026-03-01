package store

import (
	"database/sql"

	"canoe-slalom-live/domain"
)

// ListSponsorsByEvent returns all sponsors for an event, ordered by tier priority then sort_order.
func ListSponsorsByEvent(db *sql.DB, eventID int) ([]domain.Sponsor, error) {
	rows, err := db.Query(`
		SELECT id, event_id, name, logo_url, COALESCE(website_url, ''), tier, sort_order
		FROM sponsors
		WHERE event_id = ?
		ORDER BY
			CASE tier WHEN 'main' THEN 1 WHEN 'partner' THEN 2 WHEN 'supporter' THEN 3 ELSE 4 END,
			sort_order`,
		eventID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sponsors []domain.Sponsor
	for rows.Next() {
		var s domain.Sponsor
		if err := rows.Scan(&s.ID, &s.EventID, &s.Name, &s.LogoURL, &s.WebsiteURL, &s.Tier, &s.SortOrder); err != nil {
			return nil, err
		}
		sponsors = append(sponsors, s)
	}
	return sponsors, rows.Err()
}

// GetMainSponsor returns the main sponsor for an event (or error if none).
func GetMainSponsor(db *sql.DB, eventID int) (domain.Sponsor, error) {
	var s domain.Sponsor
	err := db.QueryRow(`
		SELECT id, event_id, name, logo_url, COALESCE(website_url, ''), tier, sort_order
		FROM sponsors
		WHERE event_id = ? AND tier = 'main'
		ORDER BY sort_order
		LIMIT 1`,
		eventID,
	).Scan(&s.ID, &s.EventID, &s.Name, &s.LogoURL, &s.WebsiteURL, &s.Tier, &s.SortOrder)
	return s, err
}
