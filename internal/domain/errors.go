package domain

import "errors"

var (
	// ErrNotFound indicates a missing entity.
	ErrNotFound = errors.New("not found")
	// ErrUnauthorized indicates missing or invalid credentials.
	ErrUnauthorized = errors.New("unauthorized")
	// ErrConflict indicates a version or state conflict.
	ErrConflict = errors.New("conflict")
)
