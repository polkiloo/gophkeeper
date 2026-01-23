package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	cfg, err := Load("")
	if err != nil {
		t.Fatalf("load config error: %v", err)
	}
	if cfg.Addr == "" || cfg.TokenSecret == "" {
		t.Fatalf("expected defaults")
	}
	if cfg.TokenTTL == 0 {
		t.Fatalf("expected token ttl")
	}
	if cfg.AuthProvider == "" {
		t.Fatalf("expected auth provider")
	}
	if cfg.Keycloak.BaseURL == "" || cfg.Keycloak.Realm == "" || cfg.Keycloak.ClientID == "" {
		t.Fatalf("expected keycloak defaults")
	}
	if cfg.StorageBackend == "" {
		t.Fatalf("expected storage backend")
	}
	if cfg.TransportKey != "" {
		t.Fatalf("expected empty transport key by default")
	}
	if cfg.TLSEnabled {
		t.Fatalf("expected tls disabled by default")
	}
}

func TestLoadFromYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "server.yaml")

	data := []byte("addr: :9999\npostgres_dsn: postgres://user:pass@localhost/db\nkeycloak:\n  realm: demo\n")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write config error: %v", err)
	}

	cfg, err := Load(ConfigPath(path))
	if err != nil {
		t.Fatalf("load config error: %v", err)
	}
	if cfg.Addr != ":9999" {
		t.Fatalf("unexpected addr: %s", cfg.Addr)
	}
	if cfg.PostgresDSN != "postgres://user:pass@localhost/db" {
		t.Fatalf("unexpected dsn: %s", cfg.PostgresDSN)
	}
	if cfg.Keycloak.Realm != "demo" {
		t.Fatalf("unexpected realm: %s", cfg.Keycloak.Realm)
	}
}

func TestLoadEnvOverridesYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "server.yaml")

	if err := os.WriteFile(path, []byte("addr: :9999\n"), 0o600); err != nil {
		t.Fatalf("write config error: %v", err)
	}

	t.Setenv("GOPHKEEPER_ADDR", ":7777")
	cfg, err := Load(ConfigPath(path))
	if err != nil {
		t.Fatalf("load config error: %v", err)
	}
	if cfg.Addr != ":7777" {
		t.Fatalf("unexpected addr: %s", cfg.Addr)
	}
}
