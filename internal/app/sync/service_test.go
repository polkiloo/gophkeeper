package sync

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"

	"gophkeeper/internal/app"
	"gophkeeper/internal/domain"
	"gophkeeper/internal/ports/outbound"
)

type mockRecordRepo struct{ mock.Mock }

type mockChangeRepo struct{ mock.Mock }

type mockClock struct{ mock.Mock }

func (m *mockRecordRepo) Upsert(ctx context.Context, record domain.Record) (domain.Record, error) {
	args := m.Called(ctx, record)
	return args.Get(0).(domain.Record), args.Error(1)
}

func (m *mockRecordRepo) Get(ctx context.Context, id domain.RecordID) (domain.Record, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(domain.Record), args.Error(1)
}

func (m *mockRecordRepo) List(ctx context.Context, ownerID domain.UserID, filter outbound.RecordFilter) (outbound.Iterator[domain.Record], error) {
	args := m.Called(ctx, ownerID, filter)
	return args.Get(0).(outbound.Iterator[domain.Record]), args.Error(1)
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

func TestServicePullUnauthorized(t *testing.T) {
	svc := NewService(new(mockRecordRepo), new(mockChangeRepo), new(mockClock))
	_, err := svc.Pull(context.Background(), "", 10)
	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("expected unauthorized, got %v", err)
	}
}

func TestServicePullSuccess(t *testing.T) {
	changes := new(mockChangeRepo)
	changes.On("List", mock.Anything, domain.UserID("user-1"), domain.SyncCursor("c1"), 10).
		Return([]domain.RecordChange{{RecordID: "r1"}}, domain.SyncCursor("c2"), nil)

	svc := NewService(new(mockRecordRepo), changes, new(mockClock))
	ctx := app.WithUserID(context.Background(), "user-1")
	batch, err := svc.Pull(ctx, "c1", 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if batch.Cursor != "c2" || len(batch.Changes) != 1 {
		t.Fatalf("unexpected batch")
	}
}

func TestServicePullError(t *testing.T) {
	changes := new(mockChangeRepo)
	changes.On("List", mock.Anything, domain.UserID("user-1"), domain.SyncCursor("c1"), 10).
		Return([]domain.RecordChange(nil), domain.SyncCursor(""), errors.New("list failed"))

	svc := NewService(new(mockRecordRepo), changes, new(mockClock))
	ctx := app.WithUserID(context.Background(), "user-1")
	_, err := svc.Pull(ctx, "c1", 10)
	if err == nil || err.Error() != "list failed" {
		t.Fatalf("expected list error, got %v", err)
	}
}

func TestServicePushUnauthorized(t *testing.T) {
	svc := NewService(new(mockRecordRepo), new(mockChangeRepo), new(mockClock))
	_, err := svc.Push(context.Background(), nil)
	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("expected unauthorized, got %v", err)
	}
}

func TestServicePushOwnerMismatch(t *testing.T) {
	svc := NewService(new(mockRecordRepo), new(mockChangeRepo), new(mockClock))
	ctx := app.WithUserID(context.Background(), "user-1")
	result, err := svc.Push(ctx, []domain.RecordChange{{RecordID: "r1", OwnerID: "user-2"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Rejected != 1 || len(result.Conflicts) != 1 {
		t.Fatalf("expected conflict")
	}
}

func TestServicePushConflict(t *testing.T) {
	records := new(mockRecordRepo)
	records.On("Get", mock.Anything, domain.RecordID("r1")).Return(domain.Record{ID: "r1", OwnerID: "user-1", Version: 5}, nil)

	svc := NewService(records, new(mockChangeRepo), new(mockClock))
	ctx := app.WithUserID(context.Background(), "user-1")
	result, err := svc.Push(ctx, []domain.RecordChange{{RecordID: "r1", OwnerID: "user-1", Version: 4, Change: domain.ChangeUpsert}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Rejected != 1 {
		t.Fatalf("expected rejection")
	}
}

func TestServicePushUpsertNew(t *testing.T) {
	records := new(mockRecordRepo)
	changes := new(mockChangeRepo)
	clock := new(mockClock)

	now := time.Date(2024, 9, 1, 10, 0, 0, 0, time.UTC)
	records.On("Get", mock.Anything, domain.RecordID("r1")).Return(domain.Record{}, domain.ErrNotFound)
	records.On("Upsert", mock.Anything, mock.Anything).Return(domain.Record{ID: "r1"}, nil)
	clock.On("Now").Return(now)
	changes.On("Append", mock.Anything, mock.Anything).Return(nil)

	svc := NewService(records, changes, clock)
	ctx := app.WithUserID(context.Background(), "user-1")
	result, err := svc.Push(ctx, []domain.RecordChange{{RecordID: "r1", OwnerID: "user-1", Type: domain.RecordTypeText, Change: domain.ChangeUpsert, Version: 1, Payload: domain.TextPayload{Text: "hi"}}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Applied != 1 {
		t.Fatalf("expected applied")
	}
}

func TestServicePushUpsertError(t *testing.T) {
	records := new(mockRecordRepo)
	clock := new(mockClock)

	records.On("Get", mock.Anything, domain.RecordID("r8")).Return(domain.Record{}, domain.ErrNotFound)
	records.On("Upsert", mock.Anything, mock.Anything).Return(domain.Record{}, errors.New("upsert failed"))
	clock.On("Now").Return(time.Date(2024, 9, 5, 10, 0, 0, 0, time.UTC))

	svc := NewService(records, new(mockChangeRepo), clock)
	ctx := app.WithUserID(context.Background(), "user-1")
	_, err := svc.Push(ctx, []domain.RecordChange{{RecordID: "r8", OwnerID: "user-1", Change: domain.ChangeUpsert, Type: domain.RecordTypeText, Version: 1, Payload: domain.TextPayload{Text: "hi"}}})
	if err == nil || err.Error() != "upsert failed" {
		t.Fatalf("expected upsert error, got %v", err)
	}
}

func TestServicePushDeleteNotFound(t *testing.T) {
	records := new(mockRecordRepo)
	changes := new(mockChangeRepo)

	records.On("Get", mock.Anything, domain.RecordID("r2")).Return(domain.Record{}, domain.ErrNotFound)
	records.On("Delete", mock.Anything, domain.RecordID("r2")).Return(domain.ErrNotFound)
	changes.On("Append", mock.Anything, mock.Anything).Return(nil)

	svc := NewService(records, changes, new(mockClock))
	ctx := app.WithUserID(context.Background(), "user-1")
	result, err := svc.Push(ctx, []domain.RecordChange{{RecordID: "r2", OwnerID: "user-1", Change: domain.ChangeDelete, HappenedAt: time.Date(2024, 9, 2, 10, 0, 0, 0, time.UTC)}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Applied != 1 {
		t.Fatalf("expected applied")
	}
}

func TestServicePushDeleteError(t *testing.T) {
	records := new(mockRecordRepo)
	clock := new(mockClock)
	records.On("Get", mock.Anything, domain.RecordID("r3")).Return(domain.Record{}, domain.ErrNotFound)
	records.On("Delete", mock.Anything, domain.RecordID("r3")).Return(errors.New("delete failed"))
	clock.On("Now").Return(time.Date(2024, 9, 2, 11, 0, 0, 0, time.UTC))

	svc := NewService(records, new(mockChangeRepo), clock)
	ctx := app.WithUserID(context.Background(), "user-1")
	_, err := svc.Push(ctx, []domain.RecordChange{{RecordID: "r3", OwnerID: "user-1", Change: domain.ChangeDelete}})
	if err == nil || err.Error() != "delete failed" {
		t.Fatalf("expected delete error, got %v", err)
	}
}

func TestServicePushOwnerMismatchError(t *testing.T) {
	records := new(mockRecordRepo)
	records.On("Get", mock.Anything, domain.RecordID("r4")).Return(domain.Record{ID: "r4", OwnerID: "user-2", Version: 1}, nil)

	svc := NewService(records, new(mockChangeRepo), new(mockClock))
	ctx := app.WithUserID(context.Background(), "user-1")
	_, err := svc.Push(ctx, []domain.RecordChange{{RecordID: "r4", OwnerID: "user-1", Change: domain.ChangeUpsert, Version: 2}})
	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("expected unauthorized, got %v", err)
	}
}

func TestServicePushGetError(t *testing.T) {
	records := new(mockRecordRepo)
	records.On("Get", mock.Anything, domain.RecordID("r5")).Return(domain.Record{}, errors.New("get failed"))

	svc := NewService(records, new(mockChangeRepo), new(mockClock))
	ctx := app.WithUserID(context.Background(), "user-1")
	_, err := svc.Push(ctx, []domain.RecordChange{{RecordID: "r5", OwnerID: "user-1", Change: domain.ChangeUpsert, Version: 1}})
	if err == nil || err.Error() != "get failed" {
		t.Fatalf("expected get error, got %v", err)
	}
}

func TestServicePushUnknownChange(t *testing.T) {
	records := new(mockRecordRepo)
	clock := new(mockClock)
	records.On("Get", mock.Anything, domain.RecordID("r6")).Return(domain.Record{}, domain.ErrNotFound)
	clock.On("Now").Return(time.Date(2024, 9, 4, 10, 0, 0, 0, time.UTC))

	svc := NewService(records, new(mockChangeRepo), clock)
	ctx := app.WithUserID(context.Background(), "user-1")
	_, err := svc.Push(ctx, []domain.RecordChange{{RecordID: "r6", OwnerID: "user-1", Change: domain.ChangeType("weird")}})
	if err == nil || err.Error() != "unknown change type" {
		t.Fatalf("expected unknown change error, got %v", err)
	}
}

func TestServicePushAppendError(t *testing.T) {
	records := new(mockRecordRepo)
	changes := new(mockChangeRepo)
	clock := new(mockClock)

	now := time.Date(2024, 9, 3, 10, 0, 0, 0, time.UTC)
	records.On("Get", mock.Anything, domain.RecordID("r7")).Return(domain.Record{}, domain.ErrNotFound)
	records.On("Upsert", mock.Anything, mock.Anything).Return(domain.Record{ID: "r7"}, nil)
	clock.On("Now").Return(now)
	changes.On("Append", mock.Anything, mock.Anything).Return(errors.New("append failed"))

	svc := NewService(records, changes, clock)
	ctx := app.WithUserID(context.Background(), "user-1")
	_, err := svc.Push(ctx, []domain.RecordChange{{RecordID: "r7", OwnerID: "user-1", Change: domain.ChangeUpsert, Type: domain.RecordTypeText, Version: 1, Payload: domain.TextPayload{Text: "hi"}}})
	if err == nil || err.Error() != "append failed" {
		t.Fatalf("expected append error, got %v", err)
	}
}
