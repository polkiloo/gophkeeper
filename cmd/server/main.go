// Package main wires the GophKeeper server application.
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/fx"

	"gophkeeper/internal/adapters/crypto"
	"gophkeeper/internal/adapters/persistence"
	"gophkeeper/internal/adapters/system"
	"gophkeeper/internal/app/auth"
	"gophkeeper/internal/app/auth/keycloak"
	"gophkeeper/internal/app/authprovider"
	"gophkeeper/internal/app/secrets"
	"gophkeeper/internal/app/sync"
	"gophkeeper/internal/buildinfo"
	"gophkeeper/internal/config"
	"gophkeeper/internal/logger"
	"gophkeeper/internal/migrate"
	"gophkeeper/internal/server"
	"gophkeeper/internal/transport/httpapi"
)

var (
	buildVersion = "dev"
	buildDate    = "unknown"
	buildCommit  = "none"
)

func main() {
	buildinfo.Print(os.Stdout, buildinfo.Info{Version: buildVersion, Date: buildDate, Commit: buildCommit})

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	defer stop()

	app := fx.New(
		fx.Provide(func() context.Context { return ctx }),
		logger.Module,
		config.Module,
		system.Module,
		persistence.Module,
		crypto.Module,
		auth.Module,
		keycloak.Module,
		authprovider.Module,
		secrets.Module,
		sync.Module,
		httpapi.Module,
		migrate.Module,
		server.Module,
	)

	if err := run(ctx, app); err != nil {
		log.Printf("server stopped with error: %v", err)
	}
}

func run(ctx context.Context, app *fx.App) error {
	if err := app.Start(ctx); err != nil {
		return err
	}
	<-ctx.Done()
	return app.Stop(context.Background())
}
