package config

import "testing"

func TestLoadDefaults(t *testing.T) {
	cfg := Load()
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
}
