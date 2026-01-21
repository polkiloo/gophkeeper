package sync

import (
	"context"
	"errors"

	"gophkeeper/internal/app"
	"gophkeeper/internal/domain"
	"gophkeeper/internal/ports/outbound"
)

// Service implements synchronization use cases.
type Service struct {
	records outbound.RecordRepository
	changes outbound.ChangeLogRepository
	clock   outbound.Clock
}

// NewService constructs a sync service.
func NewService(records outbound.RecordRepository, changes outbound.ChangeLogRepository, clock outbound.Clock) *Service {
	return &Service{
		records: records,
		changes: changes,
		clock:   clock,
	}
}

// Pull returns new changes for the authenticated user.
func (s *Service) Pull(ctx context.Context, cursor domain.SyncCursor, limit int) (domain.SyncBatch, error) {
	ownerID, ok := app.UserIDFromContext(ctx)
	if !ok {
		return domain.SyncBatch{}, domain.ErrUnauthorized
	}

	changes, nextCursor, err := s.changes.List(ctx, ownerID, cursor, limit)
	if err != nil {
		return domain.SyncBatch{}, err
	}

	return domain.SyncBatch{Changes: changes, Cursor: nextCursor}, nil
}

// Push applies incoming changes for the authenticated user.
func (s *Service) Push(ctx context.Context, changes []domain.RecordChange) (domain.SyncResult, error) {
	ownerID, ok := app.UserIDFromContext(ctx)
	if !ok {
		return domain.SyncResult{}, domain.ErrUnauthorized
	}

	result := domain.SyncResult{}

	for _, change := range changes {
		if change.OwnerID != ownerID {
			result.Rejected++
			result.Conflicts = append(result.Conflicts, change.RecordID)
			continue
		}

		_, err := s.applyChange(ctx, change)
		if err != nil {
			if errors.Is(err, domain.ErrConflict) {
				result.Rejected++
				result.Conflicts = append(result.Conflicts, change.RecordID)
				continue
			}
			return domain.SyncResult{}, err
		}

		result.Applied++
	}

	return result, nil
}

func (s *Service) applyChange(ctx context.Context, change domain.RecordChange) (bool, error) {
	current, err := s.records.Get(ctx, change.RecordID)
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return false, err
	}

	if err == nil {
		if current.OwnerID != change.OwnerID {
			return false, domain.ErrUnauthorized
		}
		if change.Version <= current.Version {
			return false, domain.ErrConflict
		}
	}

	happenedAt := change.HappenedAt
	if happenedAt.IsZero() {
		happenedAt = s.clock.Now()
	}

	switch change.Change {
	case domain.ChangeDelete:
		if err := s.records.Delete(ctx, change.RecordID); err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				break
			}
			return false, err
		}
	case domain.ChangeUpsert:
		record := domain.Record{
			ID:        change.RecordID,
			OwnerID:   change.OwnerID,
			Type:      change.Type,
			Payload:   change.Payload,
			Meta:      change.Meta,
			Version:   change.Version,
			UpdatedAt: happenedAt,
		}
		if _, err := s.records.Upsert(ctx, record); err != nil {
			return false, err
		}
	default:
		return false, errors.New("unknown change type")
	}

	change.HappenedAt = happenedAt
	if err := s.changes.Append(ctx, change); err != nil {
		return false, err
	}

	return true, nil
}
