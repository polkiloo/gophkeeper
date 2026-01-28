package migrate

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"go.uber.org/fx"

	"gophkeeper/internal/config"
)

const migrationsPath = "file://migrations"

// Module applies database migrations on startup.
var Module = fx.Options(
	fx.Invoke(registerMigrations),
)

func registerMigrations(lc fx.Lifecycle, cfg config.Config, db *sql.DB) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			if !usePostgres(cfg) {
				return nil
			}
			if db == nil {
				return errors.New("postgres db is not configured")
			}
			driver, err := postgres.WithInstance(db, &postgres.Config{})
			if err != nil {
				return err
			}
			m, err := migrate.NewWithDatabaseInstance(migrationsPath, "postgres", driver)
			if err != nil {
				return err
			}
			if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
				return err
			}
			return nil
		},
	})
}

func usePostgres(cfg config.Config) bool {
	return strings.EqualFold(cfg.StorageBackend, "postgres")
}
