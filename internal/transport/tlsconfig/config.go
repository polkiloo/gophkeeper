package tlsconfig

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"os"
)

// ServerSettings captures TLS settings for the server.
type ServerSettings struct {
	Enabled  bool
	CertFile string
	KeyFile  string
	ClientCA string
}

// ClientSettings captures TLS settings for the client.
type ClientSettings struct {
	CAFile             string
	InsecureSkipVerify bool
}

// NewServerTLSConfig builds a TLS config for the server.
func NewServerTLSConfig(settings ServerSettings) (*tls.Config, error) {
	if !settings.Enabled {
		return nil, nil
	}
	if settings.CertFile == "" || settings.KeyFile == "" {
		return nil, errors.New("tls cert and key are required")
	}

	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS13,
		MaxVersion: tls.VersionTLS13,
	}

	if settings.ClientCA != "" {
		caCerts, err := os.ReadFile(settings.ClientCA)
		if err != nil {
			return nil, err
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(caCerts) {
			return nil, errors.New("failed to parse client ca")
		}
		tlsConfig.ClientCAs = pool
		tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert
	}

	return tlsConfig, nil
}

// NewClientTLSConfig builds a TLS config for the client.
func NewClientTLSConfig(settings ClientSettings) (*tls.Config, error) {
	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS13,
		MaxVersion: tls.VersionTLS13,
	}
	if settings.InsecureSkipVerify {
		tlsConfig.InsecureSkipVerify = true
	}
	if settings.CAFile != "" {
		caCerts, err := os.ReadFile(settings.CAFile)
		if err != nil {
			return nil, err
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(caCerts) {
			return nil, errors.New("failed to parse ca")
		}
		tlsConfig.RootCAs = pool
	}
	return tlsConfig, nil
}
