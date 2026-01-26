package server

import (
	"testing"

	"go.uber.org/fx"

	"gophkeeper/internal/adapters/crypto"
	"gophkeeper/internal/adapters/memory"
	"gophkeeper/internal/adapters/system"
	"gophkeeper/internal/app/auth"
	"gophkeeper/internal/app/secrets"
	"gophkeeper/internal/app/sync"
	"gophkeeper/internal/config"
	"gophkeeper/internal/logger"
	"gophkeeper/internal/transport/httpapi"
)

func TestFxGraphValid(t *testing.T) {
	app := fx.New(
		logger.Module,
		config.Module,
		system.Module,
		memory.Module,
		crypto.Module,
		auth.Module,
		secrets.Module,
		sync.Module,
		httpapi.Module,
		Module,
	)
	if err := app.Err(); err != nil {
		t.Fatalf("fx graph invalid: %v", err)
	}
}
