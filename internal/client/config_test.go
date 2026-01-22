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

	if err := os.WriteFile(path, []byte("baseurl: http://example.com/api\n"), 0o600); err != nil {
		t.Fatalf("write config error: %v", err)
	}

	cfg, err := LoadConfig(ConfigPath(path))
	if err != nil {
		t.Fatalf("load config error: %v", err)
	}
	if cfg.BaseURL != "http://example.com/api" {
		t.Fatalf("unexpected baseurl: %s", cfg.BaseURL)
	}
}
