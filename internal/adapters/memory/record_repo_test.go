package memory

import (
	"context"
	"testing"
	"time"

	"gophkeeper/internal/domain"
	"gophkeeper/internal/ports/outbound"
)

func TestRecordRepositoryCRUD(t *testing.T) {
	repo := NewRecordRepository()

	record := domain.Record{
		ID:        "r1",
		OwnerID:   "u1",
		Type:      domain.RecordTypeText,
		Payload:   domain.TextPayload{Text: "hello"},
		Meta:      domain.Metadata{Title: "title", Tags: []string{"tag"}, Attributes: map[string]string{"site": "example.com"}},
		Version:   1,
		UpdatedAt: time.Now(),
	}

	if _, err := repo.Upsert(context.Background(), record); err != nil {
		t.Fatalf("upsert error: %v", err)
	}
	stored, err := repo.Get(context.Background(), "r1")
	if err != nil {
		t.Fatalf("get error: %v", err)
	}
	if stored.ID != record.ID {
		t.Fatalf("unexpected record")
	}

	list, err := repo.List(context.Background(), "u1", outbound.RecordFilter{Type: domain.RecordTypeText, Tag: "tag"})
	if err != nil {
		t.Fatalf("list error: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected record list")
	}

	list, err = repo.List(context.Background(), "u1", outbound.RecordFilter{Query: "example"})
	if err != nil {
		t.Fatalf("list query error: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected query match")
	}

	if err := repo.Delete(context.Background(), "r1"); err != nil {
		t.Fatalf("delete error: %v", err)
	}
	if _, err := repo.Get(context.Background(), "r1"); err == nil {
		t.Fatalf("expected not found after delete")
	}
	if err := repo.Delete(context.Background(), "r1"); err == nil {
		t.Fatalf("expected not found on second delete")
	}
}

func TestRecordRepositoryIncludeDeleted(t *testing.T) {
	repo := NewRecordRepository()

	record := domain.Record{ID: "r1", OwnerID: "u1", Type: domain.RecordTypeText, Payload: domain.TextPayload{Text: "x"}, Version: 1}
	if _, err := repo.Upsert(context.Background(), record); err != nil {
		t.Fatalf("upsert error: %v", err)
	}
	if err := repo.Delete(context.Background(), "r1"); err != nil {
		t.Fatalf("delete error: %v", err)
	}

	list, err := repo.List(context.Background(), "u1", outbound.RecordFilter{})
	if err != nil {
		t.Fatalf("list error: %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("expected empty list without deleted")
	}

	list, err = repo.List(context.Background(), "u1", outbound.RecordFilter{IncludeDeleted: true})
	if err != nil {
		t.Fatalf("list error: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected deleted record")
	}
}
