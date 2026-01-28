package server

import (
	"bytes"
	"context"
	"log"
	"net/http"
	"testing"
	"time"

	"go.uber.org/fx"

	"gophkeeper/internal/config"
)

func TestNewHTTPServer(t *testing.T) {
	cfg := config.Config{Addr: ":9999"}
	handler := http.NewServeMux()

	srv := NewHTTPServer(cfg, handler, nil)
	if srv.Addr != ":9999" {
		t.Fatalf("unexpected addr: %s", srv.Addr)
	}
	if srv.Handler == nil {
		t.Fatalf("expected handler")
	}
	if srv.ReadTimeout != 10*time.Second {
		t.Fatalf("unexpected read timeout")
	}
}

type testLifecycle struct {
	hooks []fx.Hook
}

func (l *testLifecycle) Append(h fx.Hook) {
	l.hooks = append(l.hooks, h)
}

func TestRegisterHooks(t *testing.T) {
	logger := log.New(new(bytes.Buffer), "", 0)

	t.Run("plain", func(t *testing.T) {
		lc := &testLifecycle{}
		srv := &http.Server{Addr: "invalid"}

		RegisterHooks(lc, srv, logger, config.Config{Addr: "invalid"})
		if len(lc.hooks) != 1 {
			t.Fatalf("expected one hook")
		}
		if err := lc.hooks[0].OnStart(context.Background()); err != nil {
			t.Fatalf("unexpected start error: %v", err)
		}
		if err := lc.hooks[0].OnStop(context.Background()); err != nil {
			t.Fatalf("unexpected stop error: %v", err)
		}
	})

	t.Run("tls", func(t *testing.T) {
		lc := &testLifecycle{}
		srv := &http.Server{Addr: "invalid"}

		RegisterHooks(lc, srv, logger, config.Config{
			Addr:        "invalid",
			TLSEnabled:  true,
			TLSCertFile: "missing.crt",
			TLSKeyFile:  "missing.key",
		})
		if len(lc.hooks) != 1 {
			t.Fatalf("expected one hook")
		}
		if err := lc.hooks[0].OnStart(context.Background()); err != nil {
			t.Fatalf("unexpected start error: %v", err)
		}
		if err := lc.hooks[0].OnStop(context.Background()); err != nil {
			t.Fatalf("unexpected stop error: %v", err)
		}
	})
}
