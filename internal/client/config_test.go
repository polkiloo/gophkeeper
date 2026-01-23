package client

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfigDefault(t *testing.T) {
	cfg, err := LoadConfig(ConfigPath(filepath.Join(os.TempDir(), "missing-client.yaml")))
	if err != nil {
		t.Fatalf("load config error: %v", err)
	}
	if cfg.BaseURL != DefaultBaseURL {
		t.Fatalf("unexpected baseurl: %s", cfg.BaseURL)
	}
}

func TestLoadConfigFromFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "client.yaml")

	if err := os.WriteFile(path, []byte("baseurl: http://example.com/api\ntransport_key: abc\ntls_ca_file: /tmp/ca.pem\ntls_insecure_skip_verify: true\n"), 0o600); err != nil {
		t.Fatalf("write config error: %v", err)
	}

	cfg, err := LoadConfig(ConfigPath(path))
	if err != nil {
		t.Fatalf("load config error: %v", err)
	}
	if cfg.BaseURL != "http://example.com/api" {
		t.Fatalf("unexpected baseurl: %s", cfg.BaseURL)
	}
	if cfg.TransportKey != "abc" {
		t.Fatalf("unexpected transport key: %s", cfg.TransportKey)
	}
	if cfg.TLSCAFile != "/tmp/ca.pem" {
		t.Fatalf("unexpected tls ca: %s", cfg.TLSCAFile)
	}
	if !cfg.TLSInsecureSkipVerify {
		t.Fatalf("expected tls insecure flag")
	}
}

func TestLoadConfigEnvOverridesTransportKey(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "client.yaml")

	if err := os.WriteFile(path, []byte("baseurl: http://example.com/api\ntransport_key: filekey\n"), 0o600); err != nil {
		t.Fatalf("write config error: %v", err)
	}

	t.Setenv("GOPHKEEPER_TRANSPORT_KEY", "envkey")
	t.Setenv("GOPHKEEPER_CLIENT_TLS_CA", "/env/ca.pem")
	t.Setenv("GOPHKEEPER_CLIENT_TLS_INSECURE", "true")
	cfg, err := LoadConfig(ConfigPath(path))
	if err != nil {
		t.Fatalf("load config error: %v", err)
	}
	if cfg.TransportKey != "envkey" {
		t.Fatalf("unexpected transport key: %s", cfg.TransportKey)
	}
	if cfg.TLSCAFile != "/env/ca.pem" {
		t.Fatalf("unexpected tls ca: %s", cfg.TLSCAFile)
	}
	if !cfg.TLSInsecureSkipVerify {
		t.Fatalf("expected tls insecure flag")
	}
}
