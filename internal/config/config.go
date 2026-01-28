// Package config loads runtime configuration.
package config

import (
	"os"
	"path/filepath"
	"time"

	"gophkeeper/internal/configutil"
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
		Addr:         ":8080",
		TokenSecret:  "dev-secret",
		TokenTTL:     time.Hour,
		AuthProvider: "local",
		Keycloak: KeycloakConfig{
			BaseURL:           "http://localhost:8081",
			Realm:             "gophkeeper",
			ClientID:          "gophkeeper-cli",
			ClientSecret:      "",
			AdminUser:         "admin",
			AdminPassword:     "admin",
			AdminClientID:     "admin-cli",
			AdminClientSecret: "",
		},
		StorageBackend: "memory",
		PostgresDSN:    "",
		TransportKey:   "",
		TLSEnabled:     false,
		TLSCertFile:    "",
		TLSKeyFile:     "",
		TLSCAFile:      "",
	}
}

func applyEnvOverrides(cfg *Config) {
	configutil.OverrideEnv(&cfg.Addr, "GOPHKEEPER_ADDR", configutil.ParseString)
	configutil.OverrideEnv(&cfg.TokenSecret, "GOPHKEEPER_TOKEN_SECRET", configutil.ParseString)
	configutil.OverrideEnv(&cfg.TokenTTL, "GOPHKEEPER_TOKEN_TTL", configutil.ParseDuration)
	configutil.OverrideEnv(&cfg.AuthProvider, "GOPHKEEPER_AUTH_PROVIDER", configutil.ParseString)
	configutil.OverrideEnv(&cfg.Keycloak.BaseURL, "KEYCLOAK_BASE_URL", configutil.ParseString)
	configutil.OverrideEnv(&cfg.Keycloak.Realm, "KEYCLOAK_REALM", configutil.ParseString)
	configutil.OverrideEnv(&cfg.Keycloak.ClientID, "KEYCLOAK_CLIENT_ID", configutil.ParseString)
	configutil.OverrideEnv(&cfg.Keycloak.ClientSecret, "KEYCLOAK_CLIENT_SECRET", configutil.ParseString)
	configutil.OverrideEnv(&cfg.Keycloak.AdminUser, "KEYCLOAK_ADMIN_USER", configutil.ParseString)
	configutil.OverrideEnv(&cfg.Keycloak.AdminPassword, "KEYCLOAK_ADMIN_PASSWORD", configutil.ParseString)
	configutil.OverrideEnv(&cfg.Keycloak.AdminClientID, "KEYCLOAK_ADMIN_CLIENT_ID", configutil.ParseString)
	configutil.OverrideEnv(&cfg.Keycloak.AdminClientSecret, "KEYCLOAK_ADMIN_CLIENT_SECRET", configutil.ParseString)
	configutil.OverrideEnv(&cfg.StorageBackend, "GOPHKEEPER_STORAGE", configutil.ParseString)
	configutil.OverrideEnv(&cfg.PostgresDSN, "GOPHKEEPER_PG_DSN", configutil.ParseString)
	configutil.OverrideEnv(&cfg.TransportKey, "GOPHKEEPER_TRANSPORT_KEY", configutil.ParseString)
	configutil.OverrideEnv(&cfg.TLSEnabled, "GOPHKEEPER_TLS_ENABLE", configutil.ParseBool)
	configutil.OverrideEnv(&cfg.TLSCertFile, "GOPHKEEPER_TLS_CERT", configutil.ParseString)
	configutil.OverrideEnv(&cfg.TLSKeyFile, "GOPHKEEPER_TLS_KEY", configutil.ParseString)
	configutil.OverrideEnv(&cfg.TLSCAFile, "GOPHKEEPER_TLS_CA", configutil.ParseString)
}
