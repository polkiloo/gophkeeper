package migrate

import (
	"context"
	"database/sql"
	"testing"

	"go.uber.org/fx"

	"gophkeeper/internal/config"
)

type testLifecycle struct {
	hooks []fx.Hook
}

func (l *testLifecycle) Append(h fx.Hook) {
	l.hooks = append(l.hooks, h)
}

func TestRegisterMigrationsSkip(t *testing.T) {
	lc := &testLifecycle{}
	registerMigrations(lc, config.Config{StorageBackend: "memory"}, &sql.DB{})
	if len(lc.hooks) != 1 {
		t.Fatalf("expected hook")
	}
	if err := lc.hooks[0].OnStart(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRegisterMigrationsMissingDB(t *testing.T) {
	lc := &testLifecycle{}
	registerMigrations(lc, config.Config{StorageBackend: "postgres"}, nil)
	if len(lc.hooks) != 1 {
		t.Fatalf("expected hook")
	}
	if err := lc.hooks[0].OnStart(context.Background()); err == nil {
		t.Fatalf("expected error")
	}
}

func TestUsePostgres(t *testing.T) {
	if !usePostgres(config.Config{StorageBackend: "Postgres"}) {
		t.Fatalf("expected postgres true")
	}
	if usePostgres(config.Config{StorageBackend: "memory"}) {
		t.Fatalf("expected postgres false")
	}
}
