package client

import (
	"os"
	"path/filepath"
	"strconv"

	"gopkg.in/yaml.v3"
)

// DefaultBaseURL is the default API endpoint for the CLI.
const DefaultBaseURL = "http://localhost:8080/api"

// ConfigPath wraps configuration path for DI.
type ConfigPath string

// Config stores CLI settings.
type Config struct {
	BaseURL               string `yaml:"baseurl"`
	TransportKey          string `yaml:"transport_key"`
	TLSCAFile             string `yaml:"tls_ca_file"`
	TLSInsecureSkipVerify bool   `yaml:"tls_insecure_skip_verify"`
}

// ConfigPathFromEnv returns the config path from environment or empty.
func ConfigPathFromEnv() ConfigPath {
	if value := os.Getenv("GOPHKEEPER_CLIENT_CONFIG"); value != "" {
		return ConfigPath(value)
	}
	return ""
}

// DefaultConfigPath builds the default config path.
func DefaultConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "gophkeeper-client.yaml"
	}
	return filepath.Join(home, ".config", "gophkeeper", "client.yaml")
}

// LoadConfig loads configuration from YAML.
func LoadConfig(path ConfigPath) (Config, error) {
	cfg := Config{BaseURL: DefaultBaseURL}

	resolved := string(path)
	if resolved == "" {
		resolved = DefaultConfigPath()
	}

	data, err := os.ReadFile(resolved)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, err
	}

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return cfg, err
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = DefaultBaseURL
	}
	if envKey := os.Getenv("GOPHKEEPER_TRANSPORT_KEY"); envKey != "" {
		cfg.TransportKey = envKey
	}
	if envCA := os.Getenv("GOPHKEEPER_CLIENT_TLS_CA"); envCA != "" {
		cfg.TLSCAFile = envCA
	}
	if envInsecure := os.Getenv("GOPHKEEPER_CLIENT_TLS_INSECURE"); envInsecure != "" {
		if parsed, err := strconv.ParseBool(envInsecure); err == nil {
			cfg.TLSInsecureSkipVerify = parsed
		}
	}
	return cfg, nil
}
