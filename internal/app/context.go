package app

import (
	"context"

	"gophkeeper/internal/domain"
)

type contextKey string

const userIDKey contextKey = "user_id"

// WithUserID stores the authenticated user ID in the context.
func WithUserID(ctx context.Context, id domain.UserID) context.Context {
	return context.WithValue(ctx, userIDKey, id)
}

// UserIDFromContext loads the authenticated user ID from the context.
func UserIDFromContext(ctx context.Context) (domain.UserID, bool) {
	value := ctx.Value(userIDKey)
	id, ok := value.(domain.UserID)
	return id, ok
}
