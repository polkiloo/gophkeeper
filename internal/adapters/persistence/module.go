package persistence

import (
	"database/sql"
	"errors"
	"strings"

	"go.uber.org/fx"

	"gophkeeper/internal/adapters/memory"
	"gophkeeper/internal/adapters/postgres"
	"gophkeeper/internal/config"
	"gophkeeper/internal/ports/outbound"
)

// Module provides repositories based on configuration.
var Module = fx.Options(
	memory.Module,
	postgres.Module,
	fx.Provide(selectUserRepository),
	fx.Provide(selectRecordRepository),
	fx.Provide(selectChangeLogRepository),
)

type repoParams struct {
	fx.In
	Cfg  config.Config
	DB   *sql.DB
	MemU *memory.UserRepository
	MemR *memory.RecordRepository
	MemC *memory.ChangeLogRepository
}

func selectUserRepository(p repoParams) (outbound.UserRepository, error) {
	if usePostgres(p.Cfg) {
		if p.DB == nil {
			return nil, errors.New("postgres db is not configured")
		}
		return postgres.NewUserRepository(p.DB), nil
	}
	return p.MemU, nil
}

func selectRecordRepository(p repoParams) (outbound.RecordRepository, error) {
	if usePostgres(p.Cfg) {
		if p.DB == nil {
			return nil, errors.New("postgres db is not configured")
		}
		return postgres.NewRecordRepository(p.DB), nil
	}
	return p.MemR, nil
}

func selectChangeLogRepository(p repoParams) (outbound.ChangeLogRepository, error) {
	if usePostgres(p.Cfg) {
		if p.DB == nil {
			return nil, errors.New("postgres db is not configured")
		}
		return postgres.NewChangeLogRepository(p.DB), nil
	}
	return p.MemC, nil
}

func usePostgres(cfg config.Config) bool {
	return strings.EqualFold(cfg.StorageBackend, "postgres")
}
