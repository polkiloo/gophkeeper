package domain

import "time"

// User represents a registered account.
type User struct {
	ID        UserID
	Login     string
	CreatedAt time.Time
}

// Session represents an authenticated user session.
type Session struct {
	UserID    UserID
	Token     string
	ExpiresAt time.Time
}
