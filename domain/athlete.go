package domain

// Athlete represents a competitor.
type Athlete struct {
	ID        int
	Name      string
	Club      string
	Nation    string
	Bio       string
	PhotoURL  string
	CreatedAt string
}

// Entry links an athlete to an event and category with a bib number and start position.
type Entry struct {
	ID            int
	EventID       int
	CategoryID    int
	AthleteID     int
	BibNumber     int
	StartPosition int
}
