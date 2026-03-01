package store

import (
	"database/sql"
	"sort"
	"time"

	"canoe-slalom-live/domain"
)

// RunResult holds run data for leaderboard display.
type RunResult struct {
	RawTimeMs      int
	PenaltyTouches int
	PenaltyMisses  int
	PenaltySeconds int
	TotalTimeMs    int
	Status         string
	JudgedAt       string
}

// LeaderboardRow represents a single row in the category leaderboard.
type LeaderboardRow struct {
	Rank            int
	BibNumber       int
	AthleteID       int
	AthleteName     string
	AthleteNation   string
	Run1            *RunResult
	Run2            *RunResult
	BestTotalTimeMs int
	HasRuns         bool
	TimeBehindMs    int
	Run1IsBest      bool
	Run2IsBest      bool
	Run1IsNew       bool
	Run2IsNew       bool
}

// GetEntryByID returns an entry with athlete info by entry ID.
func GetEntryByID(db *sql.DB, id int) (EntryWithAthlete, error) {
	var ea EntryWithAthlete
	err := db.QueryRow(`
		SELECT e.id, e.bib_number, e.start_position, a.id, a.name, a.club, a.nation, e.event_id, e.category_id
		FROM entries e
		JOIN athletes a ON a.id = e.athlete_id
		WHERE e.id = ?`,
		id,
	).Scan(&ea.EntryID, &ea.BibNumber, &ea.StartPosition, &ea.AthleteID, &ea.AthleteName, &ea.Club, &ea.Nation, &ea.EventID, &ea.CategoryID)
	return ea, err
}

// CreateRun inserts a new run and returns the new row ID.
func CreateRun(db *sql.DB, run domain.Run) (int64, error) {
	result, err := db.Exec(`
		INSERT INTO runs (entry_id, run_number, raw_time_ms, penalty_touches, penalty_misses, penalty_seconds, total_time_ms, status, judged_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		run.EntryID, run.RunNumber, run.RawTimeMs,
		run.PenaltyTouches, run.PenaltyMisses, run.PenaltySeconds,
		run.TotalTimeMs, run.Status, run.JudgedAt,
	)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

// ListRunsByEntry returns all runs for a given entry, ordered by run_number.
func ListRunsByEntry(db *sql.DB, entryID int) ([]domain.Run, error) {
	rows, err := db.Query(`
		SELECT id, entry_id, run_number, raw_time_ms, penalty_touches, penalty_misses, penalty_seconds, total_time_ms, status, judged_at
		FROM runs WHERE entry_id = ? ORDER BY run_number`,
		entryID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var runs []domain.Run
	for rows.Next() {
		var r domain.Run
		if err := rows.Scan(&r.ID, &r.EntryID, &r.RunNumber, &r.RawTimeMs,
			&r.PenaltyTouches, &r.PenaltyMisses, &r.PenaltySeconds,
			&r.TotalTimeMs, &r.Status, &r.JudgedAt); err != nil {
			return nil, err
		}
		runs = append(runs, r)
	}
	return runs, rows.Err()
}

// GetLeaderboard returns ranked results for a category.
// Athletes with no runs sort last. Ranked by best total_time_ms ascending.
// Equal times get equal rank; next rank skips accordingly.
func GetLeaderboard(db *sql.DB, categoryID int) ([]LeaderboardRow, error) {
	rows, err := db.Query(`
		SELECT
			e.bib_number,
			e.athlete_id,
			a.name,
			a.nation,
			r1.raw_time_ms, r1.penalty_touches, r1.penalty_misses, r1.penalty_seconds, r1.total_time_ms, r1.status, r1.judged_at,
			r2.raw_time_ms, r2.penalty_touches, r2.penalty_misses, r2.penalty_seconds, r2.total_time_ms, r2.status, r2.judged_at
		FROM entries e
		JOIN athletes a ON a.id = e.athlete_id
		LEFT JOIN runs r1 ON r1.entry_id = e.id AND r1.run_number = 1
		LEFT JOIN runs r2 ON r2.entry_id = e.id AND r2.run_number = 2
		WHERE e.category_id = ?
		ORDER BY e.start_position`,
		categoryID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var allRows []LeaderboardRow
	for rows.Next() {
		var lr LeaderboardRow

		var r1Raw, r1Touches, r1Misses, r1Penalty, r1Total sql.NullInt64
		var r1Status, r1JudgedAt sql.NullString
		var r2Raw, r2Touches, r2Misses, r2Penalty, r2Total sql.NullInt64
		var r2Status, r2JudgedAt sql.NullString

		if err := rows.Scan(
			&lr.BibNumber, &lr.AthleteID, &lr.AthleteName, &lr.AthleteNation,
			&r1Raw, &r1Touches, &r1Misses, &r1Penalty, &r1Total, &r1Status, &r1JudgedAt,
			&r2Raw, &r2Touches, &r2Misses, &r2Penalty, &r2Total, &r2Status, &r2JudgedAt,
		); err != nil {
			return nil, err
		}

		if r1Total.Valid {
			lr.Run1 = &RunResult{
				RawTimeMs:      int(r1Raw.Int64),
				PenaltyTouches: int(r1Touches.Int64),
				PenaltyMisses:  int(r1Misses.Int64),
				PenaltySeconds: int(r1Penalty.Int64),
				TotalTimeMs:    int(r1Total.Int64),
				Status:         r1Status.String,
				JudgedAt:       r1JudgedAt.String,
			}
		}

		if r2Total.Valid {
			lr.Run2 = &RunResult{
				RawTimeMs:      int(r2Raw.Int64),
				PenaltyTouches: int(r2Touches.Int64),
				PenaltyMisses:  int(r2Misses.Int64),
				PenaltySeconds: int(r2Penalty.Int64),
				TotalTimeMs:    int(r2Total.Int64),
				Status:         r2Status.String,
				JudgedAt:       r2JudgedAt.String,
			}
		}

		// Compute best time from valid (status=ok) runs
		best := 0
		if lr.Run1 != nil && lr.Run1.Status == "ok" {
			best = lr.Run1.TotalTimeMs
		}
		if lr.Run2 != nil && lr.Run2.Status == "ok" {
			if best == 0 || lr.Run2.TotalTimeMs < best {
				best = lr.Run2.TotalTimeMs
			}
		}
		lr.BestTotalTimeMs = best
		lr.HasRuns = lr.Run1 != nil || lr.Run2 != nil

		allRows = append(allRows, lr)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Separate into athletes with valid times and those without
	var withRuns []LeaderboardRow
	var noRuns []LeaderboardRow
	for _, lr := range allRows {
		if lr.BestTotalTimeMs > 0 {
			withRuns = append(withRuns, lr)
		} else {
			noRuns = append(noRuns, lr)
		}
	}

	// Sort athletes with runs by best time ascending
	sort.Slice(withRuns, func(i, j int) bool {
		return withRuns[i].BestTotalTimeMs < withRuns[j].BestTotalTimeMs
	})

	// Assign ranks with ties
	for i := range withRuns {
		if i == 0 {
			withRuns[i].Rank = 1
		} else if withRuns[i].BestTotalTimeMs == withRuns[i-1].BestTotalTimeMs {
			withRuns[i].Rank = withRuns[i-1].Rank
		} else {
			withRuns[i].Rank = i + 1
		}
	}

	// No-run athletes get Rank = 0 (displayed as "—")
	result := append(withRuns, noRuns...)

	// Compute time behind leader
	if len(withRuns) > 0 {
		leaderTime := withRuns[0].BestTotalTimeMs
		for i := range result {
			if result[i].BestTotalTimeMs > 0 {
				result[i].TimeBehindMs = result[i].BestTotalTimeMs - leaderTime
			}
		}
	}

	// Compute best-run and NEW badges
	now := time.Now()
	for i := range result {
		row := &result[i]
		if row.Run1 != nil && row.Run1.Status == "ok" && row.Run1.TotalTimeMs == row.BestTotalTimeMs {
			row.Run1IsBest = true
		}
		if row.Run2 != nil && row.Run2.Status == "ok" && row.Run2.TotalTimeMs == row.BestTotalTimeMs {
			row.Run2IsBest = true
		}
		if row.Run1 != nil && row.Run1.JudgedAt != "" {
			if t, err := time.Parse(time.RFC3339, row.Run1.JudgedAt); err == nil {
				row.Run1IsNew = now.Sub(t) < 60*time.Second
			}
		}
		if row.Run2 != nil && row.Run2.JudgedAt != "" {
			if t, err := time.Parse(time.RFC3339, row.Run2.JudgedAt); err == nil {
				row.Run2IsNew = now.Sub(t) < 60*time.Second
			}
		}
	}

	return result, nil
}

// GetRunByID returns a single run by ID.
func GetRunByID(db *sql.DB, id int) (domain.Run, error) {
	var r domain.Run
	err := db.QueryRow(`
		SELECT id, entry_id, run_number, raw_time_ms, penalty_touches, penalty_misses, penalty_seconds, total_time_ms, status, judged_at
		FROM runs WHERE id = ?`,
		id,
	).Scan(&r.ID, &r.EntryID, &r.RunNumber, &r.RawTimeMs,
		&r.PenaltyTouches, &r.PenaltyMisses, &r.PenaltySeconds,
		&r.TotalTimeMs, &r.Status, &r.JudgedAt)
	return r, err
}

// UpdateRun updates the time/penalty/status fields of an existing run.
func UpdateRun(db *sql.DB, run domain.Run) error {
	_, err := db.Exec(`
		UPDATE runs SET raw_time_ms = ?, penalty_touches = ?, penalty_misses = ?, penalty_seconds = ?, total_time_ms = ?, status = ?, judged_at = ?
		WHERE id = ?`,
		run.RawTimeMs, run.PenaltyTouches, run.PenaltyMisses, run.PenaltySeconds,
		run.TotalTimeMs, run.Status, run.JudgedAt, run.ID,
	)
	return err
}

// DeleteRun deletes a run by ID.
func DeleteRun(db *sql.DB, id int) error {
	_, err := db.Exec(`DELETE FROM runs WHERE id = ?`, id)
	return err
}

// RecentRun holds data for the recent runs feed on the judge page.
type RecentRun struct {
	ID             int
	AthleteName    string
	BibNumber      int
	CategoryCode   string
	RunNumber      int
	TotalTimeMs    int
	PenaltySeconds int
	Status         string
	JudgedAt       string
}

// ListRecentRuns returns the most recent runs for an event, ordered by judged_at DESC.
func ListRecentRuns(db *sql.DB, eventID int, limit int) ([]RecentRun, error) {
	rows, err := db.Query(`
		SELECT r.id, a.name, e.bib_number, c.code, r.run_number, r.total_time_ms, r.penalty_seconds, r.status, r.judged_at
		FROM runs r
		JOIN entries e ON e.id = r.entry_id
		JOIN athletes a ON a.id = e.athlete_id
		JOIN categories c ON c.id = e.category_id
		WHERE e.event_id = ?
		ORDER BY r.judged_at DESC
		LIMIT ?`,
		eventID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var recent []RecentRun
	for rows.Next() {
		var rr RecentRun
		if err := rows.Scan(&rr.ID, &rr.AthleteName, &rr.BibNumber, &rr.CategoryCode, &rr.RunNumber, &rr.TotalTimeMs, &rr.PenaltySeconds, &rr.Status, &rr.JudgedAt); err != nil {
			return nil, err
		}
		recent = append(recent, rr)
	}
	return recent, rows.Err()
}
