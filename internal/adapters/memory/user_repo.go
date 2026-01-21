// Package memory provides in-memory persistence adapters.
package memory

import (
	"context"
	"sync"

	"gophkeeper/internal/domain"
)

type userEntry struct {
	user         domain.User
	passwordHash string
}

// UserRepository stores users in memory.
type UserRepository struct {
	mu      sync.RWMutex
	byID    map[domain.UserID]userEntry
	byLogin map[string]domain.UserID
}

// NewUserRepository constructs an in-memory user repository.
func NewUserRepository() *UserRepository {
	return &UserRepository{
		byID:    make(map[domain.UserID]userEntry),
		byLogin: make(map[string]domain.UserID),
	}
}

// Create stores a new user account.
func (r *UserRepository) Create(_ context.Context, user domain.User, passwordHash string) (domain.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.byLogin[user.Login]; exists {
		return domain.User{}, domain.ErrConflict
	}

	r.byID[user.ID] = userEntry{user: user, passwordHash: passwordHash}
	r.byLogin[user.Login] = user.ID
	return user, nil
}

// FindByLogin finds a user by login.
func (r *UserRepository) FindByLogin(_ context.Context, login string) (domain.User, string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	id, ok := r.byLogin[login]
	if !ok {
		return domain.User{}, "", domain.ErrNotFound
	}
	entry := r.byID[id]
	return entry.user, entry.passwordHash, nil
}

// FindByID finds a user by id.
func (r *UserRepository) FindByID(_ context.Context, id domain.UserID) (domain.User, string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	entry, ok := r.byID[id]
	if !ok {
		return domain.User{}, "", domain.ErrNotFound
	}
	return entry.user, entry.passwordHash, nil
}

// UpdatePasswordHash updates a user's stored password hash.
func (r *UserRepository) UpdatePasswordHash(_ context.Context, id domain.UserID, passwordHash string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	entry, ok := r.byID[id]
	if !ok {
		return domain.ErrNotFound
	}
	entry.passwordHash = passwordHash
	r.byID[id] = entry
	return nil
}
