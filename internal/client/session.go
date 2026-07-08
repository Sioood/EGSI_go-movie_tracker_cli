package client

import (
	"context"
	"fmt"

	"github.com/movietracker/movie-tracker/internal/config"
)

// RestoredSession holds updated tokens and user info after a successful restore.
type RestoredSession struct {
	Session config.Session
}

// RestoreSession validates stored tokens via /api/me, refreshing if needed.
func RestoreSession(ctx context.Context, auth *AuthClient, sess config.Session) (RestoredSession, error) {
	if sess.RefreshToken == "" {
		return RestoredSession{}, fmt.Errorf("no refresh token")
	}

	access := sess.AccessToken
	if access != "" {
		info, err := auth.Me(ctx, access)
		if err == nil {
			return RestoredSession{Session: config.Session{
				AccessToken:  access,
				RefreshToken: sess.RefreshToken,
				ServerUserID: info.ID,
				Email:        info.Email,
			}}, nil
		}
		if !IsUnauthorized(err) {
			return RestoredSession{}, err
		}
	}

	pair, err := auth.Refresh(ctx, sess.RefreshToken)
	if err != nil {
		return RestoredSession{}, err
	}

	info, err := auth.Me(ctx, pair.AccessToken)
	if err != nil {
		return RestoredSession{}, err
	}

	return RestoredSession{Session: config.Session{
		AccessToken:  pair.AccessToken,
		RefreshToken: pair.RefreshToken,
		ServerUserID: info.ID,
		Email:        info.Email,
	}}, nil
}
