package domain

type Photo struct {
	ID               int
	EventID          int
	AthleteID        int // 0 if not linked to a specific athlete
	ImageURL         string
	Caption          string
	PhotographerName string
	CreatedAt        string
}
