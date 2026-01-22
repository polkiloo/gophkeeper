package inbound

import (
	"context"

	"gophkeeper/internal/domain"
)

// SyncUseCase defines change synchronization flows.
type SyncUseCase interface {
	Pull(ctx context.Context, cursor domain.SyncCursor, limit int) (domain.SyncBatch, error)
	Push(ctx context.Context, changes []domain.RecordChange) (domain.SyncResult, error)
}
