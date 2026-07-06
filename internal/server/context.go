package server

import (
	"context"

	"github.com/movietracker/movie-tracker/internal/auth"
)

type contextKey string

const claimsKey contextKey = "jwt_claims"

func withClaims(ctx context.Context, c *auth.Claims) context.Context {
	return context.WithValue(ctx, claimsKey, c)
}

func claimsFromContext(ctx context.Context) (*auth.Claims, bool) {
	c, ok := ctx.Value(claimsKey).(*auth.Claims)
	return c, ok && c != nil
}
