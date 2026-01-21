package remote

import (
	"context"

	"gophkeeper/internal/domain"
	"gophkeeper/internal/ports/inbound"
)

// ServerAPI represents the server endpoints available to a client.
type ServerAPI interface {
	Register(ctx context.Context, input inbound.RegisterInput) (domain.User, error)
	Login(ctx context.Context, input inbound.LoginInput) (domain.Session, error)
	Validate(ctx context.Context, token string) (domain.Session, error)
	Upsert(ctx context.Context, record domain.Record) (domain.Record, error)
	Get(ctx context.Context, id domain.RecordID) (domain.Record, error)
	List(ctx context.Context, filter inbound.RecordFilter) ([]domain.Record, error)
	Delete(ctx context.Context, id domain.RecordID) error
	Pull(ctx context.Context, cursor domain.SyncCursor, limit int) (domain.SyncBatch, error)
	Push(ctx context.Context, changes []domain.RecordChange) (domain.SyncResult, error)
}
