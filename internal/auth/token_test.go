package auth

import (
	"errors"
	"testing"
	"time"
)

func TestAccessAndRefreshTokensHaveDistinctTypes(t *testing.T) {
	secret := []byte("test-secret-key-for-jwt-types")
	access, err := GenerateAccessToken("user-1", "a@example.com", secret)
	if err != nil {
		t.Fatalf("generate access: %v", err)
	}
	refresh, err := GenerateRefreshToken("user-1", "a@example.com", secret)
	if err != nil {
		t.Fatalf("generate refresh: %v", err)
	}

	accessClaims, err := ParseAccessToken(access, secret)
	if err != nil {
		t.Fatalf("parse access: %v", err)
	}
	if accessClaims.TokenType != TokenTypeAccess {
		t.Fatalf("want access typ, got %q", accessClaims.TokenType)
	}

	refreshClaims, err := ParseRefreshToken(refresh, secret)
	if err != nil {
		t.Fatalf("parse refresh: %v", err)
	}
	if refreshClaims.TokenType != TokenTypeRefresh {
		t.Fatalf("want refresh typ, got %q", refreshClaims.TokenType)
	}
}

func TestAccessTokenRejectedAsRefresh(t *testing.T) {
	secret := []byte("test-secret-key-for-jwt-types")
	access, err := GenerateAccessToken("user-1", "a@example.com", secret)
	if err != nil {
		t.Fatalf("generate access: %v", err)
	}

	_, err = ParseRefreshToken(access, secret)
	if !errors.Is(err, ErrWrongTokenType) {
		t.Fatalf("want ErrWrongTokenType, got %v", err)
	}
}

func TestRefreshTokenRejectedAsAccess(t *testing.T) {
	secret := []byte("test-secret-key-for-jwt-types")
	refresh, err := GenerateRefreshToken("user-1", "a@example.com", secret)
	if err != nil {
		t.Fatalf("generate refresh: %v", err)
	}

	_, err = ParseAccessToken(refresh, secret)
	if !errors.Is(err, ErrWrongTokenType) {
		t.Fatalf("want ErrWrongTokenType, got %v", err)
	}
}

func TestExpiredTokenRejected(t *testing.T) {
	secret := []byte("test-secret-key-for-jwt-types")
	// Sign a token that is already expired by using a negative TTL via direct sign.
	expired, err := signToken("user-1", "a@example.com", TokenTypeAccess, secret, -time.Minute)
	if err != nil {
		t.Fatalf("sign expired: %v", err)
	}

	_, err = ParseAccessToken(expired, secret)
	if !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("want ErrInvalidToken, got %v", err)
	}
}
