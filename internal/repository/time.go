package repository

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/movietracker/movie-tracker/internal/apperrors"
)

const sqliteTimeLayout = time.RFC3339Nano

func formatTime(t time.Time) string {
	return t.UTC().Format(sqliteTimeLayout)
}

func parseTime(value string) (time.Time, error) {
	t, err := time.Parse(sqliteTimeLayout, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("%w: parse time: %w", apperrors.ErrDB, err)
	}
	return t, nil
}

func parseNullableTime(value sql.NullString) (*time.Time, error) {
	if !value.Valid {
		return nil, nil
	}

	t, err := parseTime(value.String)
	if err != nil {
		return nil, err
	}
	return &t, nil
}
