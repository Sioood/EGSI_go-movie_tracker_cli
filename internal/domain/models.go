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
	ID              string    `json:"id"`
	UserID          string    `json:"user_id"`
	Title           string    `json:"title"`
	Year            int       `json:"year"`
	ExternalID      string    `json:"external_id"`
	UpdatedByDevice string    `json:"updated_by_device,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type WatchEntry struct {
	ID              string     `json:"id"`
	MovieID         string     `json:"movie_id"`
	Watched         bool       `json:"watched"`
	Rating          *float64   `json:"rating"`
	RatingScale     int        `json:"rating_scale"`
	Review          string     `json:"review"`
	WatchedAt       *time.Time `json:"watched_at"`
	UpdatedByDevice string     `json:"updated_by_device,omitempty"`
	UpdatedAt       time.Time  `json:"updated_at"`
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

type SyncConflict struct {
	ID             string     `json:"id"`
	EntityType     string     `json:"entity_type"`
	EntityID       string     `json:"entity_id"`
	LocalJSON      string     `json:"local_json"`
	RemoteJSON     string     `json:"remote_json"`
	LocalDeviceID  string     `json:"local_device_id"`
	RemoteDeviceID string     `json:"remote_device_id"`
	DetectedAt     time.Time  `json:"detected_at"`
	ResolvedAt     *time.Time `json:"resolved_at,omitempty"`
}

const (
	SyncEntityMovie      = "movie"
	SyncEntityWatchEntry = "watch_entry"
	ConflictChoiceLocal  = "local"
	ConflictChoiceRemote = "remote"
)
