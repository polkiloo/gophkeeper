package crypto

import (
	"context"
	"strings"
	"testing"
)

func TestPasswordHasherHashCompare(t *testing.T) {
	hasher := PasswordHasher{}

	hash, err := hasher.Hash(context.Background(), "secret")
	if err != nil {
		t.Fatalf("hash error: %v", err)
	}
	if !strings.Contains(hash, ":") {
		t.Fatalf("expected hash with salt separator")
	}

	if err := hasher.Compare(context.Background(), "secret", hash); err != nil {
		t.Fatalf("compare failed: %v", err)
	}
	if err := hasher.Compare(context.Background(), "wrong", hash); err == nil {
		t.Fatalf("expected compare failure")
	}
}

func TestPasswordHasherCompareInvalidHash(t *testing.T) {
	hasher := PasswordHasher{}
	if err := hasher.Compare(context.Background(), "secret", "bad"); err == nil {
		t.Fatalf("expected error for invalid hash")
	}
}
