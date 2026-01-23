// Package config loads runtime configuration.
package config

import (
	"os"
	"path/filepath"
	"strconv"
	"time"

	"gopkg.in/yaml.v3"
)

// Config stores server configuration values.
type Config struct {
	Addr           string         `yaml:"addr"`
	TokenSecret    string         `yaml:"token_secret"`
	TokenTTL       time.Duration  `yaml:"token_ttl"`
	AuthProvider   string         `yaml:"auth_provider"`
	Keycloak       KeycloakConfig `yaml:"keycloak"`
	StorageBackend string         `yaml:"storage_backend"`
	PostgresDSN    string         `yaml:"postgres_dsn"`
	TransportKey   string         `yaml:"transport_key"`
	TLSEnabled     bool           `yaml:"tls_enabled"`
	TLSCertFile    string         `yaml:"tls_cert_file"`
	TLSKeyFile     string         `yaml:"tls_key_file"`
	TLSCAFile      string         `yaml:"tls_ca_file"`
}

// KeycloakConfig configures Keycloak authentication.
type KeycloakConfig struct {
	BaseURL           string `yaml:"base_url"`
	Realm             string `yaml:"realm"`
	ClientID          string `yaml:"client_id"`
	ClientSecret      string `yaml:"client_secret"`
	AdminUser         string `yaml:"admin_user"`
	AdminPassword     string `yaml:"admin_password"`
	AdminClientID     string `yaml:"admin_client_id"`
	AdminClientSecret string `yaml:"admin_client_secret"`
}

// ConfigPath wraps configuration path for DI.
type ConfigPath string

// ConfigPathFromEnv returns the config path from environment or empty.
func ConfigPathFromEnv() ConfigPath {
	if value := os.Getenv("GOPHKEEPER_CONFIG"); value != "" {
		return ConfigPath(value)
	}
	return ""
}

// DefaultConfigPath builds the default config path.
func DefaultConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "gophkeeper.yaml"
	}
	return filepath.Join(home, ".config", "gophkeeper", "server.yaml")
}

// Load reads configuration from YAML and environment variables.
func Load(path ConfigPath) (Config, error) {
	cfg := defaultConfig()

	resolved := string(path)
	if resolved == "" {
		resolved = DefaultConfigPath()
	}

	data, err := os.ReadFile(resolved)
	if err != nil {
		if !os.IsNotExist(err) {
			return cfg, err
		}
	} else if err := yaml.Unmarshal(data, &cfg); err != nil {
		return cfg, err
	}

	applyEnvOverrides(&cfg)
	return cfg, nil
}

func defaultConfig() Config {
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
		StorageBackend: getenv("GOPHKEEPER_STORAGE", "memory"),
		PostgresDSN:    getenv("GOPHKEEPER_PG_DSN", ""),
		TransportKey:   getenv("GOPHKEEPER_TRANSPORT_KEY", ""),
		TLSEnabled:     parseBool(getenv("GOPHKEEPER_TLS_ENABLE", "")),
		TLSCertFile:    getenv("GOPHKEEPER_TLS_CERT", ""),
		TLSKeyFile:     getenv("GOPHKEEPER_TLS_KEY", ""),
		TLSCAFile:      getenv("GOPHKEEPER_TLS_CA", ""),
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

func parseBool(value string) bool {
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return false
	}
	return parsed
}

func applyEnvOverrides(cfg *Config) {
	if value := os.Getenv("GOPHKEEPER_ADDR"); value != "" {
		cfg.Addr = value
	}
	if value := os.Getenv("GOPHKEEPER_TOKEN_SECRET"); value != "" {
		cfg.TokenSecret = value
	}
	if value := os.Getenv("GOPHKEEPER_TOKEN_TTL"); value != "" {
		cfg.TokenTTL = parseDuration(value, cfg.TokenTTL)
	}
	if value := os.Getenv("GOPHKEEPER_AUTH_PROVIDER"); value != "" {
		cfg.AuthProvider = value
	}
	if value := os.Getenv("KEYCLOAK_BASE_URL"); value != "" {
		cfg.Keycloak.BaseURL = value
	}
	if value := os.Getenv("KEYCLOAK_REALM"); value != "" {
		cfg.Keycloak.Realm = value
	}
	if value := os.Getenv("KEYCLOAK_CLIENT_ID"); value != "" {
		cfg.Keycloak.ClientID = value
	}
	if value := os.Getenv("KEYCLOAK_CLIENT_SECRET"); value != "" {
		cfg.Keycloak.ClientSecret = value
	}
	if value := os.Getenv("KEYCLOAK_ADMIN_USER"); value != "" {
		cfg.Keycloak.AdminUser = value
	}
	if value := os.Getenv("KEYCLOAK_ADMIN_PASSWORD"); value != "" {
		cfg.Keycloak.AdminPassword = value
	}
	if value := os.Getenv("KEYCLOAK_ADMIN_CLIENT_ID"); value != "" {
		cfg.Keycloak.AdminClientID = value
	}
	if value := os.Getenv("KEYCLOAK_ADMIN_CLIENT_SECRET"); value != "" {
		cfg.Keycloak.AdminClientSecret = value
	}
	if value := os.Getenv("GOPHKEEPER_STORAGE"); value != "" {
		cfg.StorageBackend = value
	}
	if value := os.Getenv("GOPHKEEPER_PG_DSN"); value != "" {
		cfg.PostgresDSN = value
	}
	if value := os.Getenv("GOPHKEEPER_TRANSPORT_KEY"); value != "" {
		cfg.TransportKey = value
	}
	if value := os.Getenv("GOPHKEEPER_TLS_ENABLE"); value != "" {
		cfg.TLSEnabled = parseBool(value)
	}
	if value := os.Getenv("GOPHKEEPER_TLS_CERT"); value != "" {
		cfg.TLSCertFile = value
	}
	if value := os.Getenv("GOPHKEEPER_TLS_KEY"); value != "" {
		cfg.TLSKeyFile = value
	}
	if value := os.Getenv("GOPHKEEPER_TLS_CA"); value != "" {
		cfg.TLSCAFile = value
	}
}
