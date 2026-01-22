// Package config loads runtime configuration.
package config

import (
	"os"
	"time"
)

// Config stores server configuration values.
type Config struct {
	Addr         string
	TokenSecret  string
	TokenTTL     time.Duration
	AuthProvider string
	Keycloak     KeycloakConfig
}

// KeycloakConfig configures Keycloak authentication.
type KeycloakConfig struct {
	BaseURL           string
	Realm             string
	ClientID          string
	ClientSecret      string
	AdminUser         string
	AdminPassword     string
	AdminClientID     string
	AdminClientSecret string
}

// Load reads configuration from environment variables.
func Load() Config {
	return Config{
		Addr:         getenv("GOPHKEEPER_ADDR", ":8080"),
		TokenSecret:  getenv("GOPHKEEPER_TOKEN_SECRET", "dev-secret"),
		TokenTTL:     parseDuration(getenv("GOPHKEEPER_TOKEN_TTL", "1h"), time.Hour),
		AuthProvider: getenv("GOPHKEEPER_AUTH_PROVIDER", "local"),
		Keycloak: KeycloakConfig{
			BaseURL:           getenv("KEYCLOAK_BASE_URL", "http://localhost:8081"),
			Realm:             getenv("KEYCLOAK_REALM", "gophkeeper"),
			ClientID:          getenv("KEYCLOAK_CLIENT_ID", "gophkeeper-cli"),
			ClientSecret:      getenv("KEYCLOAK_CLIENT_SECRET", ""),
			AdminUser:         getenv("KEYCLOAK_ADMIN_USER", "admin"),
			AdminPassword:     getenv("KEYCLOAK_ADMIN_PASSWORD", "admin"),
			AdminClientID:     getenv("KEYCLOAK_ADMIN_CLIENT_ID", "admin-cli"),
			AdminClientSecret: getenv("KEYCLOAK_ADMIN_CLIENT_SECRET", ""),
		},
	}
}

func getenv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func parseDuration(value string, fallback time.Duration) time.Duration {
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}
	return parsed
}
