package persistence

import (
	"database/sql"
	"testing"

	"gophkeeper/internal/adapters/memory"
	"gophkeeper/internal/config"
)

func TestSelectRepositoriesMemory(t *testing.T) {
	params := repoParams{
		Cfg:  config.Config{StorageBackend: "memory"},
		MemU: memory.NewUserRepository(),
		MemR: memory.NewRecordRepository(),
		MemC: memory.NewChangeLogRepository(),
	}

	userRepo, err := selectUserRepository(params)
	if err != nil || userRepo == nil {
		t.Fatalf("expected user repo")
	}
	recordRepo, err := selectRecordRepository(params)
	if err != nil || recordRepo == nil {
		t.Fatalf("expected record repo")
	}
	changeRepo, err := selectChangeLogRepository(params)
	if err != nil || changeRepo == nil {
		t.Fatalf("expected change repo")
	}
}

func TestSelectRepositoriesPostgresMissingDB(t *testing.T) {
	params := repoParams{
		Cfg:  config.Config{StorageBackend: "postgres"},
		MemU: memory.NewUserRepository(),
		MemR: memory.NewRecordRepository(),
		MemC: memory.NewChangeLogRepository(),
	}
	if _, err := selectUserRepository(params); err == nil {
		t.Fatalf("expected error")
	}
	if _, err := selectRecordRepository(params); err == nil {
		t.Fatalf("expected error")
	}
	if _, err := selectChangeLogRepository(params); err == nil {
		t.Fatalf("expected error")
	}
}

func TestSelectRepositoriesPostgresWithDB(t *testing.T) {
	params := repoParams{
		Cfg:  config.Config{StorageBackend: "postgres"},
		DB:   &sql.DB{},
		MemU: memory.NewUserRepository(),
		MemR: memory.NewRecordRepository(),
		MemC: memory.NewChangeLogRepository(),
	}
	if _, err := selectUserRepository(params); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := selectRecordRepository(params); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := selectChangeLogRepository(params); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
