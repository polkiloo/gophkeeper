// Package main wires the GophKeeper server application.
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/fx"

	"gophkeeper/internal/buildinfo"
	"gophkeeper/internal/serverapp"
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
		serverapp.AppModule,
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
