package auth

import "github.com/alexedwards/argon2id"

// HashPassword returns an Argon2id PHC-encoded hash of the given password.
func HashPassword(password string) (string, error) {
	return argon2id.CreateHash(password, argon2id.DefaultParams)
}

// ComparePassword reports whether password matches the PHC-encoded hash.
func ComparePassword(password, hash string) (bool, error) {
	return argon2id.ComparePasswordAndHash(password, hash)
}
