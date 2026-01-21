// Package main wires the GophKeeper client application.
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/fx"

	"gophkeeper/internal/buildinfo"
	"gophkeeper/internal/client"
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

	var result client.RunResult
	app := fx.New(
		fx.Provide(func() context.Context { return ctx }),
		client.Module,
		fx.Populate(&result),
	)

	if err := run(ctx, app, result); err != nil {
		log.Printf("client stopped with error: %v", err)
	}
}

func run(ctx context.Context, app *fx.App, result client.RunResult) error {
	if err := app.Start(ctx); err != nil {
		return err
	}

	select {
	case err := <-result:
		if err != nil {
			_ = app.Stop(context.Background())
			return err
		}
	case <-ctx.Done():
	}

	return app.Stop(context.Background())
}
