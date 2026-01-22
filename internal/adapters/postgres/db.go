package postgres

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	_ "github.com/lib/pq"

	"gophkeeper/internal/config"
)

// NewDB opens a PostgreSQL connection.
func NewDB(cfg config.Config) (*sql.DB, error) {
	if cfg.StorageBackend != "" && !strings.EqualFold(cfg.StorageBackend, "postgres") {
		return nil, nil
	}
	if cfg.PostgresDSN == "" {
		return nil, errors.New("postgres dsn is required")
	}

	db, err := sql.Open("postgres", cfg.PostgresDSN)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}

	return db, nil
}
