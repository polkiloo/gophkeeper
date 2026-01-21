package outbound

import (
	"context"
	"time"
)

// PasswordHasher hashes and verifies passwords.
type PasswordHasher interface {
	Hash(ctx context.Context, password string) (string, error)
	Compare(ctx context.Context, password, hash string) error
}

// TokenClaims represents verified token data.
type TokenClaims struct {
	Subject   string
	ExpiresAt time.Time
}

// TokenIssuer signs and validates access tokens.
type TokenIssuer interface {
	Issue(ctx context.Context, subject string, ttl time.Duration) (string, time.Time, error)
	Validate(ctx context.Context, token string) (TokenClaims, error)
}

// Encryptor encrypts and decrypts record payloads.
type Encryptor interface {
	Encrypt(ctx context.Context, plaintext []byte, keyID string) ([]byte, error)
	Decrypt(ctx context.Context, ciphertext []byte, keyID string) ([]byte, error)
}
