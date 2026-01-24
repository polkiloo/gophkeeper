package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
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

func TestConfigPathFromEnv(t *testing.T) {
	t.Setenv("GOPHKEEPER_CONFIG", "/tmp/server.yaml")
	if ConfigPathFromEnv() != ConfigPath("/tmp/server.yaml") {
		t.Fatalf("unexpected config path")
	}
}

func TestLoadInvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "server.yaml")
	if err := os.WriteFile(path, []byte(":"), 0o600); err != nil {
		t.Fatalf("write config error: %v", err)
	}

	if _, err := Load(ConfigPath(path)); err == nil {
		t.Fatalf("expected error")
	}
}

func TestParseDurationFallback(t *testing.T) {
	if got := parseDuration("nope", 2*time.Hour); got != 2*time.Hour {
		t.Fatalf("expected fallback")
	}
}

func TestParseBoolInvalid(t *testing.T) {
	if parseBool("not-bool") {
		t.Fatalf("expected false")
	}
}

func TestParseHelpers(t *testing.T) {
	if got := parseDuration("30m", time.Hour); got != 30*time.Minute {
		t.Fatalf("unexpected duration")
	}
	if !parseBool("true") {
		t.Fatalf("expected true")
	}
	if ConfigPathFromEnv() != "" {
		t.Fatalf("expected empty env path")
	}
}

func TestLoadTLSEnvOverrides(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "server.yaml")
	if err := os.WriteFile(path, []byte("addr: :8081\n"), 0o600); err != nil {
		t.Fatalf("write config error: %v", err)
	}

	t.Setenv("GOPHKEEPER_TLS_ENABLE", "true")
	t.Setenv("GOPHKEEPER_TLS_CERT", "/tmp/server.crt")
	t.Setenv("GOPHKEEPER_TLS_KEY", "/tmp/server.key")
	t.Setenv("GOPHKEEPER_TLS_CA", "/tmp/ca.pem")
	cfg, err := Load(ConfigPath(path))
	if err != nil {
		t.Fatalf("load config error: %v", err)
	}
	if !cfg.TLSEnabled {
		t.Fatalf("expected tls enabled")
	}
	if cfg.TLSCertFile != "/tmp/server.crt" || cfg.TLSKeyFile != "/tmp/server.key" || cfg.TLSCAFile != "/tmp/ca.pem" {
		t.Fatalf("unexpected tls paths")
	}
}

func TestLoadKeycloakEnvOverrides(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "server.yaml")
	if err := os.WriteFile(path, []byte("addr: :8081\n"), 0o600); err != nil {
		t.Fatalf("write config error: %v", err)
	}

	t.Setenv("KEYCLOAK_BASE_URL", "http://kc")
	t.Setenv("KEYCLOAK_REALM", "realm")
	t.Setenv("KEYCLOAK_CLIENT_ID", "client")
	t.Setenv("KEYCLOAK_CLIENT_SECRET", "secret")
	t.Setenv("KEYCLOAK_ADMIN_USER", "admin")
	t.Setenv("KEYCLOAK_ADMIN_PASSWORD", "pass")
	t.Setenv("KEYCLOAK_ADMIN_CLIENT_ID", "admin-client")
	t.Setenv("KEYCLOAK_ADMIN_CLIENT_SECRET", "admin-secret")
	t.Setenv("GOPHKEEPER_STORAGE", "postgres")
	t.Setenv("GOPHKEEPER_PG_DSN", "dsn")

	cfg, err := Load(ConfigPath(path))
	if err != nil {
		t.Fatalf("load config error: %v", err)
	}
	if cfg.Keycloak.BaseURL != "http://kc" || cfg.Keycloak.Realm != "realm" || cfg.Keycloak.ClientID != "client" {
		t.Fatalf("unexpected keycloak config")
	}
	if cfg.StorageBackend != "postgres" || cfg.PostgresDSN != "dsn" {
		t.Fatalf("unexpected storage config")
	}
}
