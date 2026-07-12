package tui

import (
	"log/slog"

	"github.com/movietracker/movie-tracker/internal/logging"
)

var uiLog = logging.New("tui")

func logPersistError(msg string, err error) {
	if err != nil {
		uiLog.Warn(msg, slog.Any("err", err))
	}
}
