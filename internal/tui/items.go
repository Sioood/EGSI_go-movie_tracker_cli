package tui

import (
	"strconv"

	"github.com/movietracker/movie-tracker/internal/domain"
)

type menuItem struct {
	title       string
	description string
	route       Route
}

func (i menuItem) Title() string {
	return i.title
}

func (i menuItem) Description() string {
	return i.description
}

func (i menuItem) FilterValue() string {
	return i.title + " " + i.description
}

type movieItem struct {
	movie  domain.Movie
	status string
}

func (i movieItem) Title() string {
	if i.movie.Year > 0 {
		return i.movie.Title + " (" + strconv.Itoa(i.movie.Year) + ")"
	}
	return i.movie.Title
}

func (i movieItem) Description() string {
	return i.status
}

func (i movieItem) FilterValue() string {
	return i.movie.Title + " " + i.status
}
