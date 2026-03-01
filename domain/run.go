package domain

import "fmt"

// Run represents a single run attempt by an athlete.
type Run struct {
	ID             int
	EntryID        int
	RunNumber      int
	RawTimeMs      int
	PenaltyTouches int
	PenaltyMisses  int
	PenaltySeconds int
	TotalTimeMs    int
	Status         string
	JudgedAt       string
}

// RawTimeFormatted formats RawTimeMs as MM:SS.xx (e.g. 94370 → "01:34.37").
func (r Run) RawTimeFormatted() string {
	return formatTimeMs(r.RawTimeMs)
}

// TotalTimeFormatted formats TotalTimeMs as MM:SS.xx.
func (r Run) TotalTimeFormatted() string {
	return formatTimeMs(r.TotalTimeMs)
}

// PenaltyDisplay returns a human-readable penalty summary, e.g. "2T + 1M = 52s" or "Clean".
func (r Run) PenaltyDisplay() string {
	if r.PenaltyTouches == 0 && r.PenaltyMisses == 0 {
		return "Clean"
	}
	parts := ""
	if r.PenaltyTouches > 0 {
		parts += fmt.Sprintf("%dT", r.PenaltyTouches)
	}
	if r.PenaltyMisses > 0 {
		if parts != "" {
			parts += " + "
		}
		parts += fmt.Sprintf("%dM", r.PenaltyMisses)
	}
	return fmt.Sprintf("%s = %ds", parts, r.PenaltySeconds)
}

// formatTimeMs converts milliseconds to MM:SS.xx format.
func formatTimeMs(ms int) string {
	if ms <= 0 {
		return "0:00.00"
	}
	minutes := ms / 60000
	seconds := (ms % 60000) / 1000
	hundredths := (ms % 1000) / 10
	return fmt.Sprintf("%d:%02d.%02d", minutes, seconds, hundredths)
}

// FormatTime is a public helper to format milliseconds as M:SS.xx.
func FormatTime(ms int) string {
	return formatTimeMs(ms)
}
