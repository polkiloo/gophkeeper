package memory

import (
	"context"
	"testing"
	"time"

	"gophkeeper/internal/domain"
)

func TestUserRepositoryCRUD(t *testing.T) {
	repo := NewUserRepository()
	user := domain.User{ID: "u1", Login: "login", CreatedAt: time.Now()}

	created, err := repo.Create(context.Background(), user, "hash")
	if err != nil {
		t.Fatalf("create error: %v", err)
	}
	if created.ID != user.ID {
		t.Fatalf("unexpected user id")
	}

	byLogin, hash, err := repo.FindByLogin(context.Background(), "login")
	if err != nil {
		t.Fatalf("find by login error: %v", err)
	}
	if byLogin.ID != user.ID || hash != "hash" {
		t.Fatalf("unexpected user data")
	}

	byID, hash, err := repo.FindByID(context.Background(), "u1")
	if err != nil {
		t.Fatalf("find by id error: %v", err)
	}
	if byID.ID != user.ID || hash != "hash" {
		t.Fatalf("unexpected user data")
	}

	if err := repo.UpdatePasswordHash(context.Background(), "u1", "hash2"); err != nil {
		t.Fatalf("update hash error: %v", err)
	}
	_, hash, _ = repo.FindByID(context.Background(), "u1")
	if hash != "hash2" {
		t.Fatalf("expected updated hash")
	}
}

func TestUserRepositoryConflictAndNotFound(t *testing.T) {
	repo := NewUserRepository()
	user := domain.User{ID: "u1", Login: "login"}

	if _, err := repo.Create(context.Background(), user, "hash"); err != nil {
		t.Fatalf("create error: %v", err)
	}
	if _, err := repo.Create(context.Background(), domain.User{ID: "u2", Login: "login"}, "hash"); err == nil {
		t.Fatalf("expected conflict")
	}

	if _, _, err := repo.FindByLogin(context.Background(), "missing"); err == nil {
		t.Fatalf("expected not found")
	}
	if _, _, err := repo.FindByID(context.Background(), "missing"); err == nil {
		t.Fatalf("expected not found")
	}
	if err := repo.UpdatePasswordHash(context.Background(), "missing", "hash"); err == nil {
		t.Fatalf("expected not found")
	}
}
