package domain

import "time"

type User struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type Movie struct {
	ID         string    `json:"id"`
	UserID     string    `json:"user_id"`
	Title      string    `json:"title"`
	Year       int       `json:"year"`
	ExternalID string    `json:"external_id"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type WatchEntry struct {
	ID          string     `json:"id"`
	MovieID     string     `json:"movie_id"`
	Watched     bool       `json:"watched"`
	Rating      *float64   `json:"rating"`
	RatingScale int        `json:"rating_scale"`
	Review      string     `json:"review"`
	WatchedAt   *time.Time `json:"watched_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
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
	Year  int        `json:"year"`
	Month time.Month `json:"month"`
	Count int        `json:"count"`
}

type MovieRating struct {
	Movie  Movie   `json:"movie"`
	Rating float64 `json:"rating"`
}

type Stats struct {
	TotalMovies   int           `json:"total_movies"`
	TotalWatched  int           `json:"total_watched"`
	TotalRated    int           `json:"total_rated"`
	AverageRating float64       `json:"average_rating"`
	BestMovies    []MovieRating `json:"best_movies"`
	WorstMovies   []MovieRating `json:"worst_movies"`
	ByMonth       []MonthBucket `json:"by_month"`
}
