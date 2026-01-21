package app

import (
	"context"
	"testing"

	"gophkeeper/internal/domain"
)

func TestUserIDContext(t *testing.T) {
	ctx := context.Background()
	if _, ok := UserIDFromContext(ctx); ok {
		t.Fatalf("expected empty context to return ok=false")
	}

	ctx = WithUserID(ctx, domain.UserID("user-1"))
	value, ok := UserIDFromContext(ctx)
	if !ok {
		t.Fatalf("expected user id to be present")
	}
	if value != "user-1" {
		t.Fatalf("unexpected user id: %s", value)
	}
}
