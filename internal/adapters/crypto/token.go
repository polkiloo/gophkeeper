package crypto

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"gophkeeper/internal/ports/outbound"
)

type tokenPayload struct {
	Subject string    `json:"sub"`
	Expires time.Time `json:"exp"`
}

// TokenIssuer implements a simple HMAC-signed token format.
type TokenIssuer struct {
	secret []byte
	clock  func() time.Time
}

// NewTokenIssuer constructs a token issuer using the provided secret.
func NewTokenIssuer(secret string, clock func() time.Time) *TokenIssuer {
	return &TokenIssuer{secret: []byte(secret), clock: clock}
}

// Issue creates a signed token for the subject.
func (t *TokenIssuer) Issue(_ context.Context, subject string, ttl time.Duration) (string, time.Time, error) {
	if subject == "" {
		return "", time.Time{}, errors.New("subject is required")
	}
	if ttl <= 0 {
		return "", time.Time{}, errors.New("ttl must be positive")
	}

	expires := t.clock().Add(ttl)
	payload := tokenPayload{Subject: subject, Expires: expires}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", time.Time{}, err
	}

	payloadEnc := base64.RawURLEncoding.EncodeToString(payloadBytes)
	mac := hmac.New(sha256.New, t.secret)
	_, _ = mac.Write([]byte(payloadEnc))
	signature := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	return payloadEnc + "." + signature, expires, nil
}

// Validate parses and verifies a token.
func (t *TokenIssuer) Validate(_ context.Context, token string) (outbound.TokenClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return outbound.TokenClaims{}, errors.New("invalid token")
	}
	payloadEnc := parts[0]
	signatureEnc := parts[1]

	sig, err := base64.RawURLEncoding.DecodeString(signatureEnc)
	if err != nil {
		return outbound.TokenClaims{}, errors.New("invalid token")
	}

	mac := hmac.New(sha256.New, t.secret)
	_, _ = mac.Write([]byte(payloadEnc))
	if !hmac.Equal(sig, mac.Sum(nil)) {
		return outbound.TokenClaims{}, errors.New("invalid token")
	}

	payloadBytes, err := base64.RawURLEncoding.DecodeString(payloadEnc)
	if err != nil {
		return outbound.TokenClaims{}, errors.New("invalid token")
	}

	var payload tokenPayload
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return outbound.TokenClaims{}, errors.New("invalid token")
	}
	if t.clock().After(payload.Expires) {
		return outbound.TokenClaims{}, errors.New("token expired")
	}

	return outbound.TokenClaims{Subject: payload.Subject, ExpiresAt: payload.Expires}, nil
}
