package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	TokenTypeAccess  = "access"
	TokenTypeRefresh = "refresh"
)

// Claims is the JWT payload for access and refresh tokens.
type Claims struct {
	UserID    string `json:"uid"`
	Email     string `json:"email"`
	TokenType string `json:"typ"`
	jwt.RegisteredClaims
}

const (
	AccessTokenDuration  = 15 * time.Minute
	RefreshTokenDuration = 7 * 24 * time.Hour
)

// ErrInvalidToken is returned when a JWT cannot be parsed or is expired.
var ErrInvalidToken = errors.New("invalid token")

// ErrWrongTokenType is returned when a token has an unexpected typ claim.
var ErrWrongTokenType = errors.New("wrong token type")

// GenerateAccessToken creates a short-lived signed JWT.
func GenerateAccessToken(userID, email string, secret []byte) (string, error) {
	return signToken(userID, email, TokenTypeAccess, secret, AccessTokenDuration)
}

// GenerateRefreshToken creates a long-lived signed JWT.
func GenerateRefreshToken(userID, email string, secret []byte) (string, error) {
	return signToken(userID, email, TokenTypeRefresh, secret, RefreshTokenDuration)
}

func signToken(userID, email, tokenType string, secret []byte, ttl time.Duration) (string, error) {
	claims := Claims{
		UserID:    userID,
		Email:     email,
		TokenType: tokenType,
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

// ParseAccessToken validates an access JWT and returns its claims.
func ParseAccessToken(tokenStr string, secret []byte) (*Claims, error) {
	return parseTokenWithType(tokenStr, secret, TokenTypeAccess)
}

// ParseRefreshToken validates a refresh JWT and returns its claims.
func ParseRefreshToken(tokenStr string, secret []byte) (*Claims, error) {
	return parseTokenWithType(tokenStr, secret, TokenTypeRefresh)
}

func parseTokenWithType(tokenStr string, secret []byte, wantType string) (*Claims, error) {
	claims, err := parseToken(tokenStr, secret)
	if err != nil {
		return nil, err
	}
	if claims.TokenType != wantType {
		return nil, ErrWrongTokenType
	}
	return claims, nil
}

// ParseToken validates the signature and expiry of a signed JWT and returns its claims.
// Prefer ParseAccessToken or ParseRefreshToken for endpoint-specific validation.
func ParseToken(tokenStr string, secret []byte) (*Claims, error) {
	return parseToken(tokenStr, secret)
}

func parseToken(tokenStr string, secret []byte) (*Claims, error) {
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
