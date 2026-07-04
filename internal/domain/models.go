package domain

import "time"

type User struct {
	ID           string
	Email        string
	PasswordHash string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type Movie struct {
	ID         string
	UserID     string
	Title      string
	Year       int
	ExternalID string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type WatchEntry struct {
	ID          string
	MovieID     string
	Watched     bool
	Rating      *float64
	RatingScale int
	Review      string
	WatchedAt   *time.Time
	UpdatedAt   time.Time
}
