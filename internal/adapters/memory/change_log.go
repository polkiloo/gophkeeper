package memory

import (
	"context"
	"strconv"
	"sync"

	"gophkeeper/internal/domain"
)

type changeStream struct {
	changes []domain.RecordChange
}

// ChangeLogRepository stores record changes in memory.
type ChangeLogRepository struct {
	mu      sync.RWMutex
	streams map[domain.UserID]*changeStream
}

// NewChangeLogRepository constructs a changelog repository.
func NewChangeLogRepository() *ChangeLogRepository {
	return &ChangeLogRepository{streams: make(map[domain.UserID]*changeStream)}
}

// Append adds a change to the stream.
func (r *ChangeLogRepository) Append(_ context.Context, change domain.RecordChange) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	stream := r.streams[change.OwnerID]
	if stream == nil {
		stream = &changeStream{}
		r.streams[change.OwnerID] = stream
	}
	stream.changes = append(stream.changes, change)
	return nil
}

// List returns changes after the cursor.
func (r *ChangeLogRepository) List(_ context.Context, ownerID domain.UserID, cursor domain.SyncCursor, limit int) ([]domain.RecordChange, domain.SyncCursor, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	stream := r.streams[ownerID]
	if stream == nil {
		return []domain.RecordChange{}, "0", nil
	}

	start := 0
	if cursor != "" {
		if parsed, err := strconv.Atoi(string(cursor)); err == nil && parsed > 0 {
			start = parsed
		}
	}
	if start >= len(stream.changes) {
		return []domain.RecordChange{}, domain.SyncCursor(strconv.Itoa(len(stream.changes))), nil
	}

	end := len(stream.changes)
	if limit > 0 && start+limit < end {
		end = start + limit
	}

	nextCursor := domain.SyncCursor(strconv.Itoa(end))
	return append([]domain.RecordChange(nil), stream.changes[start:end]...), nextCursor, nil
}
