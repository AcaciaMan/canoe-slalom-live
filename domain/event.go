package domain

// Event represents a canoe slalom competition.
type Event struct {
	ID        int
	Slug      string
	Name      string
	Date      string
	Location  string
	Status    string
	CreatedAt string
}

// Category represents a competition class within an event (e.g. K1M, C1W).
type Category struct {
	ID        int
	EventID   int
	Code      string
	Name      string
	SortOrder int
	NumRuns   int
}
