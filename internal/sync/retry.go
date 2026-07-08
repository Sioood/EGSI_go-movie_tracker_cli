package sync

import (
	"context"
	"errors"
	"net"
	"time"

	"github.com/movietracker/movie-tracker/internal/apperrors"
)

const (
	maxAttempts = 5
	initialWait = time.Second
	maxWait     = 30 * time.Second
)

// WithRetry runs fn with exponential backoff on retryable errors.
func WithRetry(ctx context.Context, fn func() error) error {
	wait := initialWait
	var lastErr error

	for attempt := 0; attempt < maxAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return err
		}

		lastErr = fn()
		if lastErr == nil {
			return nil
		}
		if !IsRetryable(lastErr) {
			return lastErr
		}
		if attempt == maxAttempts-1 {
			break
		}

		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}

		wait *= 2
		if wait > maxWait {
			wait = maxWait
		}
	}

	return lastErr
}

// IsRetryable reports whether a sync error should be retried.
func IsRetryable(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, apperrors.ErrNetwork) {
		return true
	}
	var netErr net.Error
	return errors.As(err, &netErr)
}
