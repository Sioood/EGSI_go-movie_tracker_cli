package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims is the JWT payload for both access and refresh tokens.
type Claims struct {
	UserID string `json:"uid"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

const (
	AccessTokenDuration  = 15 * time.Minute
	RefreshTokenDuration = 7 * 24 * time.Hour
)

// ErrInvalidToken is returned when a JWT cannot be parsed or is expired.
var ErrInvalidToken = errors.New("invalid token")

// GenerateAccessToken creates a short-lived signed JWT.
func GenerateAccessToken(userID, email string, secret []byte) (string, error) {
	return signToken(userID, email, secret, AccessTokenDuration)
}

// GenerateRefreshToken creates a long-lived signed JWT.
func GenerateRefreshToken(userID, email string, secret []byte) (string, error) {
	return signToken(userID, email, secret, RefreshTokenDuration)
}

func signToken(userID, email string, secret []byte, ttl time.Duration) (string, error) {
	claims := Claims{
		UserID: userID,
		Email:  email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := tok.SignedString(secret)
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}
	return signed, nil
}

// ParseToken validates the signature and expiry of a signed JWT and returns its claims.
func ParseToken(tokenStr string, secret []byte) (*Claims, error) {
	tok, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("%w: unexpected signing method", ErrInvalidToken)
		}
		return secret, nil
	})
	if err != nil || !tok.Valid {
		return nil, ErrInvalidToken
	}
	claims, ok := tok.Claims.(*Claims)
	if !ok {
		return nil, ErrInvalidToken
	}
	return claims, nil
}
