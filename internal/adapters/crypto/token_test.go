package crypto

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestTokenIssuerIssueValidate(t *testing.T) {
	now := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)
	issuer := NewTokenIssuer("secret", func() time.Time { return now })

	token, expiresAt, err := issuer.Issue(context.Background(), "user-1", time.Minute)
	if err != nil {
		t.Fatalf("issue error: %v", err)
	}
	if token == "" {
		t.Fatalf("expected token")
	}
	if expiresAt.Equal(now) {
		t.Fatalf("expected expiry in the future")
	}

	claims, err := issuer.Validate(context.Background(), token)
	if err != nil {
		t.Fatalf("validate error: %v", err)
	}
	if claims.Subject != "user-1" {
		t.Fatalf("unexpected subject: %s", claims.Subject)
	}
}

func TestTokenIssuerInvalidSignature(t *testing.T) {
	issuer := NewTokenIssuer("secret", time.Now)

	token, _, err := issuer.Issue(context.Background(), "user-1", time.Minute)
	if err != nil {
		t.Fatalf("issue error: %v", err)
	}

	badToken := token + "x"
	if _, err := issuer.Validate(context.Background(), badToken); err == nil {
		t.Fatalf("expected validation error")
	}
}

func TestTokenIssuerExpired(t *testing.T) {
	now := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)
	issuer := NewTokenIssuer("secret", func() time.Time { return now })

	token, _, err := issuer.Issue(context.Background(), "user-1", time.Minute)
	if err != nil {
		t.Fatalf("issue error: %v", err)
	}

	expiredIssuer := NewTokenIssuer("secret", func() time.Time { return now.Add(2 * time.Minute) })
	if _, err := expiredIssuer.Validate(context.Background(), token); err == nil {
		t.Fatalf("expected expiration error")
	}
}

func TestTokenIssuerInvalidToken(t *testing.T) {
	issuer := NewTokenIssuer("secret", time.Now)

	if _, err := issuer.Validate(context.Background(), "invalid"); err == nil {
		t.Fatalf("expected validation error")
	}
	if _, err := issuer.Validate(context.Background(), "a.b.c"); err == nil {
		t.Fatalf("expected validation error")
	}
	if _, err := issuer.Validate(context.Background(), "a."+strings.Repeat("x", 3)); err == nil {
		t.Fatalf("expected validation error")
	}
}
