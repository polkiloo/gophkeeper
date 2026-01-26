// Package server wires the HTTP server lifecycle.
package server

import (
	"context"
	"log"
	"net/http"
	"time"

	"go.uber.org/fx"

	"gophkeeper/internal/config"
)

// Module provides the HTTP server lifecycle.
var Module = fx.Options(
	fx.Provide(NewHTTPServer),
	fx.Invoke(RegisterHooks),
)

// NewHTTPServer constructs an HTTP server instance.
func NewHTTPServer(cfg config.Config, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:         cfg.Addr,
		Handler:      handler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  30 * time.Second,
	}
}

// RegisterHooks starts and stops the HTTP server.
func RegisterHooks(lc fx.Lifecycle, srv *http.Server, logger *log.Logger) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go func() {
				if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
					logger.Printf("server failed: %v", err)
				}
			}()
			logger.Printf("gophkeeper server listening on %s", srv.Addr)
			return nil
		},
		OnStop: func(ctx context.Context) error {
			return srv.Shutdown(ctx)
		},
	})
}
