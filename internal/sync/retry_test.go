package sync

import (
	"context"
	"errors"
	"net"
	"testing"

	"github.com/movietracker/movie-tracker/internal/apperrors"
)

func TestIsRetryableNetworkError(t *testing.T) {
	if !IsRetryable(apperrors.ErrNetwork) {
		t.Fatal("ErrNetwork should be retryable")
	}
}

func TestIsRetryableTimeout(t *testing.T) {
	var netErr net.Error = &timeoutError{}
	if !IsRetryable(netErr) {
		t.Fatal("net timeout should be retryable")
	}
}

func TestIsRetryableUnauthorized(t *testing.T) {
	if IsRetryable(apperrors.ErrUnauthorized) {
		t.Fatal("ErrUnauthorized should not be retryable")
	}
}

func TestWithRetrySucceedsAfterTransientFailure(t *testing.T) {
	attempts := 0
	err := WithRetry(context.Background(), func() error {
		attempts++
		if attempts < 2 {
			return apperrors.ErrNetwork
		}
		return nil
	})
	if err != nil {
		t.Fatalf("want nil, got %v", err)
	}
	if attempts != 2 {
		t.Fatalf("want 2 attempts, got %d", attempts)
	}
}

func TestWithRetryStopsOnNonRetryableError(t *testing.T) {
	attempts := 0
	err := WithRetry(context.Background(), func() error {
		attempts++
		return apperrors.ErrUnauthorized
	})
	if !errors.Is(err, apperrors.ErrUnauthorized) {
		t.Fatalf("want ErrUnauthorized, got %v", err)
	}
	if attempts != 1 {
		t.Fatalf("want 1 attempt, got %d", attempts)
	}
}

type timeoutError struct{}

func (e *timeoutError) Error() string { return "timeout" }
func (e *timeoutError) Timeout() bool { return true }
func (e *timeoutError) Temporary() bool {
	return true
}
