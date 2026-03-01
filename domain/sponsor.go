package domain

type Sponsor struct {
	ID         int
	EventID    int
	Name       string
	LogoURL    string
	WebsiteURL string
	Tier       string // "main", "partner", "supporter"
	SortOrder  int
}
