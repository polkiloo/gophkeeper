// Package config loads runtime configuration.
package config

import (
	"os"
	"time"
)

// Config stores server configuration values.
type Config struct {
	Addr        string
	TokenSecret string
	TokenTTL    time.Duration
}

// Load reads configuration from environment variables.
func Load() Config {
	return Config{
		Addr:        getenv("GOPHKEEPER_ADDR", ":8080"),
		TokenSecret: getenv("GOPHKEEPER_TOKEN_SECRET", "dev-secret"),
		TokenTTL:    parseDuration(getenv("GOPHKEEPER_TOKEN_TTL", "1h"), time.Hour),
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
