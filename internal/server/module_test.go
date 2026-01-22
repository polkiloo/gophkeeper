package server

import (
	"net/http"
	"testing"
	"time"

	"gophkeeper/internal/config"
)

func TestNewHTTPServer(t *testing.T) {
	cfg := config.Config{Addr: ":9999"}
	handler := http.NewServeMux()

	srv := NewHTTPServer(cfg, handler)
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
