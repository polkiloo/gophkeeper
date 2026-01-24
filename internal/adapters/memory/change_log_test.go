package memory

import (
	"context"
	"testing"
	"time"

	"gophkeeper/internal/domain"
)

func TestChangeLogAppendList(t *testing.T) {
	repo := NewChangeLogRepository()

	change1 := domain.RecordChange{RecordID: "r1", OwnerID: "u1", Change: domain.ChangeUpsert, Version: 1, HappenedAt: time.Now()}
	change2 := domain.RecordChange{RecordID: "r2", OwnerID: "u1", Change: domain.ChangeUpsert, Version: 2, HappenedAt: time.Now()}

	if err := repo.Append(context.Background(), change1); err != nil {
		t.Fatalf("append error: %v", err)
	}
	if err := repo.Append(context.Background(), change2); err != nil {
		t.Fatalf("append error: %v", err)
	}

	changes, cursor, err := repo.List(context.Background(), "u1", "", 1)
	if err != nil {
		t.Fatalf("list error: %v", err)
	}
	if len(changes) != 1 || cursor == "" {
		t.Fatalf("expected first page")
	}

	changes, cursor, err = repo.List(context.Background(), "u1", cursor, 10)
	if err != nil {
		t.Fatalf("list error: %v", err)
	}
	if len(changes) != 1 || cursor == "" {
		t.Fatalf("expected second page")
	}
}

func TestChangeLogEmpty(t *testing.T) {
	repo := NewChangeLogRepository()

	changes, cursor, err := repo.List(context.Background(), "u1", "", 10)
	if err != nil {
		t.Fatalf("list error: %v", err)
	}
	if len(changes) != 0 || cursor == "" {
		t.Fatalf("expected empty list")
	}
}
