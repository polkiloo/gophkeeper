// Package crypto provides security-related adapters.
package crypto

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
)

// PasswordHasher implements a salted SHA-256 password hasher.
type PasswordHasher struct{}

// Hash produces a salted hash string.
func (PasswordHasher) Hash(_ context.Context, password string) (string, error) {
	if password == "" {
		return "", errors.New("password is required")
	}

	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}

	result := sha256.Sum256(append(salt, []byte(password)...))
	return fmt.Sprintf("%s:%s", base64.StdEncoding.EncodeToString(salt), base64.StdEncoding.EncodeToString(result[:])), nil
}

// Compare verifies a password against a hash.
func (PasswordHasher) Compare(_ context.Context, password, hash string) error {
	parts := strings.Split(hash, ":")
	if len(parts) != 2 {
		return errors.New("invalid hash format")
	}
	salt, err := base64.StdEncoding.DecodeString(parts[0])
	if err != nil {
		return err
	}
	expected, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		return err
	}

	result := sha256.Sum256(append(salt, []byte(password)...))
	if !equalBytes(result[:], expected) {
		return errors.New("invalid password")
	}
	return nil
}

func equalBytes(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	var diff byte
	for i := range a {
		diff |= a[i] ^ b[i]
	}
	return diff == 0
}
