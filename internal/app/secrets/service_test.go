package secrets

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"

	"gophkeeper/internal/app"
	"gophkeeper/internal/domain"
	"gophkeeper/internal/ports/inbound"
	"gophkeeper/internal/ports/outbound"
)

type mockRecordRepo struct{ mock.Mock }

type mockChangeRepo struct{ mock.Mock }

type mockClock struct{ mock.Mock }

type mockIDGen struct{ mock.Mock }

func (m *mockRecordRepo) Upsert(ctx context.Context, record domain.Record) (domain.Record, error) {
	args := m.Called(ctx, record)
	return args.Get(0).(domain.Record), args.Error(1)
}

func (m *mockRecordRepo) Get(ctx context.Context, id domain.RecordID) (domain.Record, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(domain.Record), args.Error(1)
}

func (m *mockRecordRepo) List(ctx context.Context, ownerID domain.UserID, filter outbound.RecordFilter) ([]domain.Record, error) {
	args := m.Called(ctx, ownerID, filter)
	return args.Get(0).([]domain.Record), args.Error(1)
}

func (m *mockRecordRepo) Delete(ctx context.Context, id domain.RecordID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *mockChangeRepo) Append(ctx context.Context, change domain.RecordChange) error {
	args := m.Called(ctx, change)
	return args.Error(0)
}

func (m *mockChangeRepo) List(ctx context.Context, ownerID domain.UserID, cursor domain.SyncCursor, limit int) ([]domain.RecordChange, domain.SyncCursor, error) {
	args := m.Called(ctx, ownerID, cursor, limit)
	return args.Get(0).([]domain.RecordChange), args.Get(1).(domain.SyncCursor), args.Error(2)
}

func (m *mockClock) Now() time.Time {
	args := m.Called()
	return args.Get(0).(time.Time)
}

func (m *mockIDGen) NewID() string {
	args := m.Called()
	return args.String(0)
}

func TestServiceUpsertNewRecord(t *testing.T) {
	records := new(mockRecordRepo)
	changes := new(mockChangeRepo)
	clock := new(mockClock)
	idGen := new(mockIDGen)

	now := time.Date(2024, 4, 4, 10, 0, 0, 0, time.UTC)
	payload := domain.TextPayload{Text: "hello"}
	input := domain.Record{Type: domain.RecordTypeText, Payload: payload, Meta: domain.Metadata{Title: "t"}}

	idGen.On("NewID").Return("rec-1")
	clock.On("Now").Return(now)

	expected := domain.Record{ID: "rec-1", OwnerID: "user-1", Type: domain.RecordTypeText, Payload: payload, Meta: input.Meta, Version: 1, UpdatedAt: now}
	records.On("Upsert", mock.Anything, expected).Return(expected, nil)
	changes.On("Append", mock.Anything, mock.MatchedBy(func(change domain.RecordChange) bool {
		return change.RecordID == "rec-1" && change.Change == domain.ChangeUpsert
	})).Return(nil)

	svc := NewService(records, changes, clock, idGen)
	ctx := app.WithUserID(context.Background(), "user-1")
	stored, err := svc.Upsert(ctx, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stored.ID != "rec-1" || stored.Version != 1 {
		t.Fatalf("unexpected record")
	}
}

func TestServiceUpsertUpdateRecord(t *testing.T) {
	records := new(mockRecordRepo)
	changes := new(mockChangeRepo)
	clock := new(mockClock)

	now := time.Date(2024, 5, 5, 10, 0, 0, 0, time.UTC)
	payload := domain.TextPayload{Text: "updated"}
	current := domain.Record{ID: "rec-1", OwnerID: "user-1", Version: 3}
	input := domain.Record{ID: "rec-1", Type: domain.RecordTypeText, Payload: payload, Meta: domain.Metadata{}, Version: 0}

	records.On("Get", mock.Anything, domain.RecordID("rec-1")).Return(current, nil)
	clock.On("Now").Return(now)

	expected := domain.Record{ID: "rec-1", OwnerID: "user-1", Type: domain.RecordTypeText, Payload: payload, Meta: input.Meta, Version: 4, UpdatedAt: now}
	records.On("Upsert", mock.Anything, expected).Return(expected, nil)
	changes.On("Append", mock.Anything, mock.Anything).Return(nil)

	svc := NewService(records, changes, clock, new(mockIDGen))
	ctx := app.WithUserID(context.Background(), "user-1")
	stored, err := svc.Upsert(ctx, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stored.Version != 4 {
		t.Fatalf("expected version 4, got %d", stored.Version)
	}
}

func TestServiceUpsertConflict(t *testing.T) {
	records := new(mockRecordRepo)
	changes := new(mockChangeRepo)
	clock := new(mockClock)
	current := domain.Record{ID: "rec-1", OwnerID: "user-1", Version: 5}

	records.On("Get", mock.Anything, domain.RecordID("rec-1")).Return(current, nil)
	clock.On("Now").Return(time.Date(2024, 5, 5, 10, 0, 0, 0, time.UTC))

	svc := NewService(records, changes, clock, new(mockIDGen))
	ctx := app.WithUserID(context.Background(), "user-1")
	_, err := svc.Upsert(ctx, domain.Record{ID: "rec-1", Type: domain.RecordTypeText, Payload: domain.TextPayload{Text: "x"}, Version: 5})
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("expected conflict, got %v", err)
	}
}

func TestServiceUpsertUnauthorized(t *testing.T) {
	svc := NewService(new(mockRecordRepo), new(mockChangeRepo), new(mockClock), new(mockIDGen))
	_, err := svc.Upsert(context.Background(), domain.Record{})
	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("expected unauthorized, got %v", err)
	}
}

func TestServiceUpsertMissingType(t *testing.T) {
	clock := new(mockClock)
	clock.On("Now").Return(time.Date(2024, 8, 9, 11, 0, 0, 0, time.UTC))

	svc := NewService(new(mockRecordRepo), new(mockChangeRepo), clock, new(mockIDGen))
	ctx := app.WithUserID(context.Background(), "user-1")
	_, err := svc.Upsert(ctx, domain.Record{})
	if err == nil {
		t.Fatalf("expected validation error")
	}
}

func TestServiceUpsertMissingPayload(t *testing.T) {
	clock := new(mockClock)
	clock.On("Now").Return(time.Date(2024, 8, 9, 12, 0, 0, 0, time.UTC))

	svc := NewService(new(mockRecordRepo), new(mockChangeRepo), clock, new(mockIDGen))
	ctx := app.WithUserID(context.Background(), "user-1")
	_, err := svc.Upsert(ctx, domain.Record{Type: domain.RecordTypeText})
	if err == nil {
		t.Fatalf("expected validation error")
	}
}

func TestServiceUpsertNotFoundCreates(t *testing.T) {
	records := new(mockRecordRepo)
	changes := new(mockChangeRepo)
	clock := new(mockClock)

	now := time.Date(2024, 6, 6, 10, 0, 0, 0, time.UTC)
	payload := domain.TextPayload{Text: "hello"}
	input := domain.Record{ID: "rec-1", Type: domain.RecordTypeText, Payload: payload}

	records.On("Get", mock.Anything, domain.RecordID("rec-1")).Return(domain.Record{}, domain.ErrNotFound)
	clock.On("Now").Return(now)

	expected := domain.Record{ID: "rec-1", OwnerID: "user-1", Type: domain.RecordTypeText, Payload: payload, Meta: domain.Metadata{}, Version: 1, UpdatedAt: now}
	records.On("Upsert", mock.Anything, expected).Return(expected, nil)
	changes.On("Append", mock.Anything, mock.Anything).Return(nil)

	svc := NewService(records, changes, clock, new(mockIDGen))
	ctx := app.WithUserID(context.Background(), "user-1")
	stored, err := svc.Upsert(ctx, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stored.Version != 1 {
		t.Fatalf("expected version 1")
	}
}

func TestServiceGet(t *testing.T) {
	records := new(mockRecordRepo)
	records.On("Get", mock.Anything, domain.RecordID("rec-1")).Return(domain.Record{ID: "rec-1", OwnerID: "user-1"}, nil)

	svc := NewService(records, new(mockChangeRepo), new(mockClock), new(mockIDGen))
	ctx := app.WithUserID(context.Background(), "user-1")
	_, err := svc.Get(ctx, "rec-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestServiceGetUnauthorized(t *testing.T) {
	records := new(mockRecordRepo)
	records.On("Get", mock.Anything, domain.RecordID("rec-1")).Return(domain.Record{ID: "rec-1", OwnerID: "user-2"}, nil)

	svc := NewService(records, new(mockChangeRepo), new(mockClock), new(mockIDGen))
	ctx := app.WithUserID(context.Background(), "user-1")
	_, err := svc.Get(ctx, "rec-1")
	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("expected unauthorized, got %v", err)
	}
}

func TestServiceGetNoContext(t *testing.T) {
	svc := NewService(new(mockRecordRepo), new(mockChangeRepo), new(mockClock), new(mockIDGen))
	_, err := svc.Get(context.Background(), "rec-1")
	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("expected unauthorized, got %v", err)
	}
}

func TestServiceList(t *testing.T) {
	records := new(mockRecordRepo)
	filter := inbound.RecordFilter{Type: domain.RecordTypeText, Tag: "t", Query: "q", Limit: 1, Offset: 2, IncludeDeleted: true}

	records.On("List", mock.Anything, domain.UserID("user-1"), outbound.RecordFilter{Type: filter.Type, Tag: filter.Tag, Query: filter.Query, Limit: 1, Offset: 2, IncludeDeleted: true}).Return([]domain.Record{}, nil)

	svc := NewService(records, new(mockChangeRepo), new(mockClock), new(mockIDGen))
	ctx := app.WithUserID(context.Background(), "user-1")
	_, err := svc.List(ctx, filter)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestServiceListUnauthorized(t *testing.T) {
	svc := NewService(new(mockRecordRepo), new(mockChangeRepo), new(mockClock), new(mockIDGen))
	_, err := svc.List(context.Background(), inbound.RecordFilter{})
	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("expected unauthorized, got %v", err)
	}
}

func TestServiceDeleteNoContext(t *testing.T) {
	svc := NewService(new(mockRecordRepo), new(mockChangeRepo), new(mockClock), new(mockIDGen))
	if err := svc.Delete(context.Background(), "rec-1"); !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("expected unauthorized, got %v", err)
	}
}

func TestServiceDelete(t *testing.T) {
	records := new(mockRecordRepo)
	changes := new(mockChangeRepo)
	clock := new(mockClock)

	now := time.Date(2024, 7, 7, 10, 0, 0, 0, time.UTC)
	records.On("Get", mock.Anything, domain.RecordID("rec-1")).Return(domain.Record{ID: "rec-1", OwnerID: "user-1", Version: 2}, nil)
	records.On("Delete", mock.Anything, domain.RecordID("rec-1")).Return(nil)
	clock.On("Now").Return(now)
	changes.On("Append", mock.Anything, mock.MatchedBy(func(change domain.RecordChange) bool {
		return change.Change == domain.ChangeDelete && change.Version == 3
	})).Return(nil)

	svc := NewService(records, changes, clock, new(mockIDGen))
	ctx := app.WithUserID(context.Background(), "user-1")
	if err := svc.Delete(ctx, "rec-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestServiceDeleteUnauthorized(t *testing.T) {
	records := new(mockRecordRepo)
	records.On("Get", mock.Anything, domain.RecordID("rec-1")).Return(domain.Record{ID: "rec-1", OwnerID: "user-2"}, nil)

	svc := NewService(records, new(mockChangeRepo), new(mockClock), new(mockIDGen))
	ctx := app.WithUserID(context.Background(), "user-1")
	if err := svc.Delete(ctx, "rec-1"); !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("expected unauthorized, got %v", err)
	}
}

func TestServiceUpsertRepositoryError(t *testing.T) {
	records := new(mockRecordRepo)
	clock := new(mockClock)
	idGen := new(mockIDGen)

	now := time.Date(2024, 8, 8, 10, 0, 0, 0, time.UTC)
	clock.On("Now").Return(now)
	idGen.On("NewID").Return("rec-2")
	records.On("Upsert", mock.Anything, mock.Anything).Return(domain.Record{}, errors.New("store failed"))

	svc := NewService(records, new(mockChangeRepo), clock, idGen)
	ctx := app.WithUserID(context.Background(), "user-1")
	_, err := svc.Upsert(ctx, domain.Record{Type: domain.RecordTypeText, Payload: domain.TextPayload{Text: "x"}})
	if err == nil || err.Error() != "store failed" {
		t.Fatalf("expected store error, got %v", err)
	}
}

func TestServiceUpsertChangeAppendError(t *testing.T) {
	records := new(mockRecordRepo)
	changes := new(mockChangeRepo)
	clock := new(mockClock)
	idGen := new(mockIDGen)

	now := time.Date(2024, 8, 9, 10, 0, 0, 0, time.UTC)
	clock.On("Now").Return(now)
	idGen.On("NewID").Return("rec-3")
	saved := domain.Record{ID: "rec-3", OwnerID: "user-1", Type: domain.RecordTypeText, Payload: domain.TextPayload{Text: "x"}, Version: 1, UpdatedAt: now}
	records.On("Upsert", mock.Anything, mock.Anything).Return(saved, nil)
	changes.On("Append", mock.Anything, mock.Anything).Return(errors.New("append failed"))

	svc := NewService(records, changes, clock, idGen)
	ctx := app.WithUserID(context.Background(), "user-1")
	_, err := svc.Upsert(ctx, domain.Record{Type: domain.RecordTypeText, Payload: domain.TextPayload{Text: "x"}})
	if err == nil || err.Error() != "append failed" {
		t.Fatalf("expected append error, got %v", err)
	}
}

func TestServiceUpsertPreserveVersion(t *testing.T) {
	records := new(mockRecordRepo)
	changes := new(mockChangeRepo)
	clock := new(mockClock)
	idGen := new(mockIDGen)

	now := time.Date(2024, 8, 14, 10, 0, 0, 0, time.UTC)
	clock.On("Now").Return(now)
	idGen.On("NewID").Return("rec-4")

	input := domain.Record{Type: domain.RecordTypeText, Payload: domain.TextPayload{Text: "x"}, Version: 7}
	expected := domain.Record{ID: "rec-4", OwnerID: "user-1", Type: domain.RecordTypeText, Payload: input.Payload, Meta: input.Meta, Version: 7, UpdatedAt: now}
	records.On("Upsert", mock.Anything, expected).Return(expected, nil)
	changes.On("Append", mock.Anything, mock.Anything).Return(nil)

	svc := NewService(records, changes, clock, idGen)
	ctx := app.WithUserID(context.Background(), "user-1")
	stored, err := svc.Upsert(ctx, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stored.Version != 7 {
		t.Fatalf("expected version 7, got %d", stored.Version)
	}
}

func TestServiceUpsertNotFoundUpsertError(t *testing.T) {
	records := new(mockRecordRepo)
	clock := new(mockClock)

	clock.On("Now").Return(time.Date(2024, 8, 15, 10, 0, 0, 0, time.UTC))
	records.On("Get", mock.Anything, domain.RecordID("rec-8")).Return(domain.Record{}, domain.ErrNotFound)
	records.On("Upsert", mock.Anything, mock.Anything).Return(domain.Record{}, errors.New("store failed"))

	svc := NewService(records, new(mockChangeRepo), clock, new(mockIDGen))
	ctx := app.WithUserID(context.Background(), "user-1")
	_, err := svc.Upsert(ctx, domain.Record{ID: "rec-8", Type: domain.RecordTypeText, Payload: domain.TextPayload{Text: "x"}, Version: 2})
	if err == nil || err.Error() != "store failed" {
		t.Fatalf("expected store error, got %v", err)
	}
}

func TestServiceUpsertNotFoundAppendError(t *testing.T) {
	records := new(mockRecordRepo)
	changes := new(mockChangeRepo)
	clock := new(mockClock)

	now := time.Date(2024, 8, 16, 10, 0, 0, 0, time.UTC)
	clock.On("Now").Return(now)
	records.On("Get", mock.Anything, domain.RecordID("rec-9")).Return(domain.Record{}, domain.ErrNotFound)
	saved := domain.Record{ID: "rec-9", OwnerID: "user-1", Type: domain.RecordTypeText, Payload: domain.TextPayload{Text: "x"}, Version: 2, UpdatedAt: now}
	records.On("Upsert", mock.Anything, mock.Anything).Return(saved, nil)
	changes.On("Append", mock.Anything, mock.Anything).Return(errors.New("append failed"))

	svc := NewService(records, changes, clock, new(mockIDGen))
	ctx := app.WithUserID(context.Background(), "user-1")
	_, err := svc.Upsert(ctx, domain.Record{ID: "rec-9", Type: domain.RecordTypeText, Payload: domain.TextPayload{Text: "x"}, Version: 2})
	if err == nil || err.Error() != "append failed" {
		t.Fatalf("expected append error, got %v", err)
	}
}

func TestServiceUpsertUpdateAppendError(t *testing.T) {
	records := new(mockRecordRepo)
	changes := new(mockChangeRepo)
	clock := new(mockClock)

	now := time.Date(2024, 8, 17, 10, 0, 0, 0, time.UTC)
	clock.On("Now").Return(now)
	records.On("Get", mock.Anything, domain.RecordID("rec-10")).Return(domain.Record{ID: "rec-10", OwnerID: "user-1", Version: 1}, nil)
	saved := domain.Record{ID: "rec-10", OwnerID: "user-1", Type: domain.RecordTypeText, Payload: domain.TextPayload{Text: "x"}, Version: 2, UpdatedAt: now}
	records.On("Upsert", mock.Anything, mock.Anything).Return(saved, nil)
	changes.On("Append", mock.Anything, mock.Anything).Return(errors.New("append failed"))

	svc := NewService(records, changes, clock, new(mockIDGen))
	ctx := app.WithUserID(context.Background(), "user-1")
	_, err := svc.Upsert(ctx, domain.Record{ID: "rec-10", Type: domain.RecordTypeText, Payload: domain.TextPayload{Text: "x"}, Version: 2})
	if err == nil || err.Error() != "append failed" {
		t.Fatalf("expected append error, got %v", err)
	}
}

func TestServiceUpsertGetError(t *testing.T) {
	records := new(mockRecordRepo)
	clock := new(mockClock)

	clock.On("Now").Return(time.Date(2024, 8, 10, 10, 0, 0, 0, time.UTC))
	records.On("Get", mock.Anything, domain.RecordID("rec-4")).Return(domain.Record{}, errors.New("get failed"))

	svc := NewService(records, new(mockChangeRepo), clock, new(mockIDGen))
	ctx := app.WithUserID(context.Background(), "user-1")
	_, err := svc.Upsert(ctx, domain.Record{ID: "rec-4", Type: domain.RecordTypeText, Payload: domain.TextPayload{Text: "x"}, Version: 2})
	if err == nil || err.Error() != "get failed" {
		t.Fatalf("expected get error, got %v", err)
	}
}

func TestServiceUpsertOwnerMismatch(t *testing.T) {
	records := new(mockRecordRepo)
	clock := new(mockClock)

	clock.On("Now").Return(time.Date(2024, 8, 11, 10, 0, 0, 0, time.UTC))
	records.On("Get", mock.Anything, domain.RecordID("rec-5")).Return(domain.Record{ID: "rec-5", OwnerID: "user-2", Version: 1}, nil)

	svc := NewService(records, new(mockChangeRepo), clock, new(mockIDGen))
	ctx := app.WithUserID(context.Background(), "user-1")
	_, err := svc.Upsert(ctx, domain.Record{ID: "rec-5", Type: domain.RecordTypeText, Payload: domain.TextPayload{Text: "x"}, Version: 2})
	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("expected unauthorized, got %v", err)
	}
}

func TestServiceUpsertUpdateStoreError(t *testing.T) {
	records := new(mockRecordRepo)
	clock := new(mockClock)

	clock.On("Now").Return(time.Date(2024, 8, 12, 10, 0, 0, 0, time.UTC))
	records.On("Get", mock.Anything, domain.RecordID("rec-6")).Return(domain.Record{ID: "rec-6", OwnerID: "user-1", Version: 1}, nil)
	records.On("Upsert", mock.Anything, mock.Anything).Return(domain.Record{}, errors.New("update failed"))

	svc := NewService(records, new(mockChangeRepo), clock, new(mockIDGen))
	ctx := app.WithUserID(context.Background(), "user-1")
	_, err := svc.Upsert(ctx, domain.Record{ID: "rec-6", Type: domain.RecordTypeText, Payload: domain.TextPayload{Text: "x"}, Version: 2})
	if err == nil || err.Error() != "update failed" {
		t.Fatalf("expected update error, got %v", err)
	}
}

func TestServiceGetRepositoryError(t *testing.T) {
	records := new(mockRecordRepo)
	records.On("Get", mock.Anything, domain.RecordID("rec-1")).Return(domain.Record{}, errors.New("read failed"))

	svc := NewService(records, new(mockChangeRepo), new(mockClock), new(mockIDGen))
	ctx := app.WithUserID(context.Background(), "user-1")
	_, err := svc.Get(ctx, "rec-1")
	if err == nil || err.Error() != "read failed" {
		t.Fatalf("expected read error, got %v", err)
	}
}

func TestServiceDeleteGetError(t *testing.T) {
	records := new(mockRecordRepo)
	records.On("Get", mock.Anything, domain.RecordID("rec-1")).Return(domain.Record{}, errors.New("read failed"))

	svc := NewService(records, new(mockChangeRepo), new(mockClock), new(mockIDGen))
	ctx := app.WithUserID(context.Background(), "user-1")
	if err := svc.Delete(ctx, "rec-1"); err == nil || err.Error() != "read failed" {
		t.Fatalf("expected read error, got %v", err)
	}
}

func TestServiceDeleteRemoveError(t *testing.T) {
	records := new(mockRecordRepo)
	records.On("Get", mock.Anything, domain.RecordID("rec-1")).Return(domain.Record{ID: "rec-1", OwnerID: "user-1", Version: 2}, nil)
	records.On("Delete", mock.Anything, domain.RecordID("rec-1")).Return(errors.New("delete failed"))

	svc := NewService(records, new(mockChangeRepo), new(mockClock), new(mockIDGen))
	ctx := app.WithUserID(context.Background(), "user-1")
	if err := svc.Delete(ctx, "rec-1"); err == nil || err.Error() != "delete failed" {
		t.Fatalf("expected delete error, got %v", err)
	}
}

func TestServiceDeleteAppendError(t *testing.T) {
	records := new(mockRecordRepo)
	changes := new(mockChangeRepo)
	clock := new(mockClock)

	now := time.Date(2024, 8, 13, 10, 0, 0, 0, time.UTC)
	records.On("Get", mock.Anything, domain.RecordID("rec-1")).Return(domain.Record{ID: "rec-1", OwnerID: "user-1", Version: 2}, nil)
	records.On("Delete", mock.Anything, domain.RecordID("rec-1")).Return(nil)
	clock.On("Now").Return(now)
	changes.On("Append", mock.Anything, mock.Anything).Return(errors.New("append failed"))

	svc := NewService(records, changes, clock, new(mockIDGen))
	ctx := app.WithUserID(context.Background(), "user-1")
	if err := svc.Delete(ctx, "rec-1"); err == nil || err.Error() != "append failed" {
		t.Fatalf("expected append error, got %v", err)
	}
}
