package outbound

import (
	"context"

	"gophkeeper/internal/domain"
)

// UserRepository persists user accounts.
type UserRepository interface {
	Create(ctx context.Context, user domain.User, passwordHash string) (domain.User, error)
	FindByLogin(ctx context.Context, login string) (domain.User, string, error)
	FindByID(ctx context.Context, id domain.UserID) (domain.User, string, error)
	UpdatePasswordHash(ctx context.Context, id domain.UserID, passwordHash string) error
}

// RecordRepository persists encrypted private records.
type RecordRepository interface {
	Upsert(ctx context.Context, record domain.Record) (domain.Record, error)
	Get(ctx context.Context, id domain.RecordID) (domain.Record, error)
	List(ctx context.Context, ownerID domain.UserID, filter RecordFilter) ([]domain.Record, error)
	Delete(ctx context.Context, id domain.RecordID) error
}

// RecordFilter narrows repository queries.
type RecordFilter struct {
	Type   domain.RecordType
	Tag    string
	Query  string
	Limit  int
	Offset int
}

// ChangeLogRepository stores synchronization deltas.
type ChangeLogRepository interface {
	Append(ctx context.Context, change domain.RecordChange) error
	List(ctx context.Context, ownerID domain.UserID, cursor domain.SyncCursor, limit int) ([]domain.RecordChange, domain.SyncCursor, error)
}
