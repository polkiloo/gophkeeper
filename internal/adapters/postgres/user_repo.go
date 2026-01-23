package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/lib/pq"

	"gophkeeper/internal/domain"
)

// UserRepository stores users in PostgreSQL.
type UserRepository struct {
	db *sql.DB
}

// NewUserRepository constructs a PostgreSQL-backed user repository.
func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

// Create stores a new user account.
func (r *UserRepository) Create(ctx context.Context, user domain.User, passwordHash string) (domain.User, error) {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO users (id, login, password_hash, created_at)
		VALUES ($1, $2, $3, $4)
	`, string(user.ID), user.Login, passwordHash, user.CreatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return domain.User{}, domain.ErrConflict
		}
		return domain.User{}, err
	}
	return user, nil
}

// FindByLogin finds a user by login.
func (r *UserRepository) FindByLogin(ctx context.Context, login string) (domain.User, string, error) {
	var user domain.User
	var hash string
	err := r.db.QueryRowContext(ctx, `
		SELECT id, login, password_hash, created_at
		FROM users
		WHERE login = $1
	`, login).Scan(&user.ID, &user.Login, &hash, &user.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.User{}, "", domain.ErrNotFound
	}
	if err != nil {
		return domain.User{}, "", err
	}
	return user, hash, nil
}

// FindByID finds a user by id.
func (r *UserRepository) FindByID(ctx context.Context, id domain.UserID) (domain.User, string, error) {
	var user domain.User
	var hash string
	err := r.db.QueryRowContext(ctx, `
		SELECT id, login, password_hash, created_at
		FROM users
		WHERE id = $1
	`, string(id)).Scan(&user.ID, &user.Login, &hash, &user.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.User{}, "", domain.ErrNotFound
	}
	if err != nil {
		return domain.User{}, "", err
	}
	return user, hash, nil
}

// UpdatePasswordHash updates a user's stored password hash.
func (r *UserRepository) UpdatePasswordHash(ctx context.Context, id domain.UserID, passwordHash string) error {
	res, err := r.db.ExecContext(ctx, `
		UPDATE users SET password_hash = $2 WHERE id = $1
	`, string(id), passwordHash)
	if err != nil {
		return err
	}
	updated, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if updated == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func isUniqueViolation(err error) bool {
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		return pqErr.Code == "23505"
	}
	return false
}
