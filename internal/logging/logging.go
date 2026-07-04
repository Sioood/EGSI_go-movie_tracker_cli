package logging

import (
	"log/slog"
	"os"
)

func New(service string) *slog.Logger {
	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})
	return slog.New(handler).With("service", service)
}
