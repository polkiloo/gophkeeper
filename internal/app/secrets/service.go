package secrets

import (
	"context"
	"errors"

	"gophkeeper/internal/app"
	"gophkeeper/internal/domain"
	"gophkeeper/internal/ports/inbound"
	"gophkeeper/internal/ports/outbound"
)

// Service implements operations over private records.
type Service struct {
	records outbound.RecordRepository
	changes outbound.ChangeLogRepository
	clock   outbound.Clock
	idGen   outbound.IDGenerator
}

// NewService constructs a secrets service.
func NewService(records outbound.RecordRepository, changes outbound.ChangeLogRepository, clock outbound.Clock, idGen outbound.IDGenerator) *Service {
	return &Service{
		records: records,
		changes: changes,
		clock:   clock,
		idGen:   idGen,
	}
}

// Upsert creates or updates a record for the authenticated user.
func (s *Service) Upsert(ctx context.Context, record domain.Record) (domain.Record, error) {
	ownerID, ok := app.UserIDFromContext(ctx)
	if !ok {
		return domain.Record{}, domain.ErrUnauthorized
	}

	record.OwnerID = ownerID
	record.UpdatedAt = s.clock.Now()

	if err := validateRecord(record); err != nil {
		return domain.Record{}, err
	}

	if record.ID == "" {
		return s.createRecord(ctx, record)
	}

	return s.updateRecord(ctx, ownerID, record)
}

// Create stores a new record for the authenticated user.
func (s *Service) Create(ctx context.Context, record domain.Record) (domain.Record, error) {
	ownerID, ok := app.UserIDFromContext(ctx)
	if !ok {
		return domain.Record{}, domain.ErrUnauthorized
	}

	record.OwnerID = ownerID
	record.UpdatedAt = s.clock.Now()

	if err := validateRecord(record); err != nil {
		return domain.Record{}, err
	}

	return s.createRecord(ctx, record)
}

// Update stores an existing record for the authenticated user.
func (s *Service) Update(ctx context.Context, record domain.Record) (domain.Record, error) {
	ownerID, ok := app.UserIDFromContext(ctx)
	if !ok {
		return domain.Record{}, domain.ErrUnauthorized
	}

	record.OwnerID = ownerID
	record.UpdatedAt = s.clock.Now()

	if err := validateRecord(record); err != nil {
		return domain.Record{}, err
	}

	if record.ID == "" {
		return domain.Record{}, errors.New("record id is required")
	}

	return s.updateRecord(ctx, ownerID, record)
}

// Get fetches a record by id for the authenticated user.
func (s *Service) Get(ctx context.Context, id domain.RecordID) (domain.Record, error) {
	ownerID, ok := app.UserIDFromContext(ctx)
	if !ok {
		return domain.Record{}, domain.ErrUnauthorized
	}

	record, err := s.records.Get(ctx, id)
	if err != nil {
		return domain.Record{}, err
	}
	if record.OwnerID != ownerID {
		return domain.Record{}, domain.ErrUnauthorized
	}
	return record, nil
}

// List returns records for the authenticated user.
func (s *Service) List(ctx context.Context, filter inbound.RecordFilter) ([]domain.Record, error) {
	ownerID, ok := app.UserIDFromContext(ctx)
	if !ok {
		return nil, domain.ErrUnauthorized
	}

	outFilter := outbound.RecordFilter{
		Type:           filter.Type,
		Tag:            filter.Tag,
		Query:          filter.Query,
		Limit:          filter.Limit,
		Offset:         filter.Offset,
		IncludeDeleted: filter.IncludeDeleted,
	}

	iter, err := s.records.List(ctx, ownerID, outFilter)
	if err != nil {
		return nil, err
	}
	return outbound.Collect(ctx, iter)
}

// Delete removes a record for the authenticated user.
func (s *Service) Delete(ctx context.Context, id domain.RecordID) error {
	ownerID, ok := app.UserIDFromContext(ctx)
	if !ok {
		return domain.ErrUnauthorized
	}

	record, err := s.records.Get(ctx, id)
	if err != nil {
		return err
	}
	if record.OwnerID != ownerID {
		return domain.ErrUnauthorized
	}

	if err := s.records.Delete(ctx, id); err != nil {
		return err
	}

	record.Version++
	record.UpdatedAt = s.clock.Now()
	return s.appendChange(ctx, record, domain.ChangeDelete)
}

func (s *Service) appendChange(ctx context.Context, record domain.Record, changeType domain.ChangeType) error {
	return s.changes.Append(ctx, domain.RecordChange{
		RecordID:   record.ID,
		OwnerID:    record.OwnerID,
		Type:       record.Type,
		Change:     changeType,
		Payload:    record.Payload,
		Meta:       record.Meta,
		Version:    record.Version,
		HappenedAt: s.clock.Now(),
	})
}

func validateRecord(record domain.Record) error {
	if record.Type == "" {
		return errors.New("record type is required")
	}
	if record.Payload == nil {
		return errors.New("record payload is required")
	}
	return nil
}

func (s *Service) createRecord(ctx context.Context, record domain.Record) (domain.Record, error) {
	record.ID = domain.RecordID(s.idGen.NewID())
	if record.Version <= 0 {
		record.Version = 1
	}
	return s.storeAndLog(ctx, record)
}

func (s *Service) updateRecord(ctx context.Context, ownerID domain.UserID, record domain.Record) (domain.Record, error) {
	current, err := s.records.Get(ctx, record.ID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			if record.Version <= 0 {
				record.Version = 1
			}
			return s.storeAndLog(ctx, record)
		}
		return domain.Record{}, err
	}

	if current.OwnerID != ownerID {
		return domain.Record{}, domain.ErrUnauthorized
	}

	if record.Version == 0 {
		record.Version = current.Version + 1
	}
	if record.Version <= current.Version {
		return domain.Record{}, domain.ErrConflict
	}

	return s.storeAndLog(ctx, record)
}

func (s *Service) storeAndLog(ctx context.Context, record domain.Record) (domain.Record, error) {
	saved, err := s.records.Upsert(ctx, record)
	if err != nil {
		return domain.Record{}, err
	}
	if err := s.appendChange(ctx, saved, domain.ChangeUpsert); err != nil {
		return domain.Record{}, err
	}
	return saved, nil
}
