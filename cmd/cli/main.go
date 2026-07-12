package main

import (
	"os"

	"github.com/movietracker/movie-tracker/internal/app"
)

func main() {
	if err := app.RunCLI(); err != nil {
		os.Exit(1)
	}
}
