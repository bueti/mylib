// Package auth provides password hashing and session token utilities
// for mylib. It has no HTTP surface area — handlers live in
// internal/api.
package auth

import (
	"crypto/rand"
	"encoding/hex"
	"errors"

	"golang.org/x/crypto/bcrypt"
)

// ErrBadPassword is returned when a supplied password doesn't match
// the stored hash.
var ErrBadPassword = errors.New("invalid username or password")

// HashPassword produces a bcrypt hash suitable for storage.
func HashPassword(plaintext string) (string, error) {
	if plaintext == "" {
		return "", errors.New("empty password")
	}
	h, err := bcrypt.GenerateFromPassword([]byte(plaintext), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(h), nil
}

// VerifyPassword checks plaintext against a stored hash.
func VerifyPassword(hash, plaintext string) error {
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(plaintext)); err != nil {
		return ErrBadPassword
	}
	return nil
}

// NewSessionToken returns a 256-bit random, hex-encoded string.
func NewSessionToken() (string, error) {
	var buf [32]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf[:]), nil
}
