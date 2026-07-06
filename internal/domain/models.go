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

type MovieFilter string

const (
	MovieFilterAll       MovieFilter = "all"
	MovieFilterWatched   MovieFilter = "watched"
	MovieFilterUnwatched MovieFilter = "unwatched"
	MovieFilterRated     MovieFilter = "rated"
	MovieFilterUnrated   MovieFilter = "unrated"
)

type MovieSort string

const (
	MovieSortTitle  MovieSort = "title"
	MovieSortDate   MovieSort = "date"
	MovieSortRating MovieSort = "rating"
)

type MovieSearchParams struct {
	UserID string
	Query  string
	Filter MovieFilter
	Sort   MovieSort
}

type MonthBucket struct {
	Year  int
	Month time.Month
	Count int
}

type MovieRating struct {
	Movie  Movie
	Rating float64
}

type Stats struct {
	TotalMovies   int
	TotalWatched  int
	TotalRated    int
	AverageRating float64
	BestMovies    []MovieRating
	WorstMovies   []MovieRating
	ByMonth       []MonthBucket
}
