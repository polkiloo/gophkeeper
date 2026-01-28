package tlsconfig

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewServerTLSConfigDisabled(t *testing.T) {
	cfg, err := NewServerTLSConfig(ServerSettings{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg != nil {
		t.Fatalf("expected nil tls config")
	}
}

func TestNewServerTLSConfigMissingFiles(t *testing.T) {
	_, err := NewServerTLSConfig(ServerSettings{Enabled: true})
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestNewServerTLSConfigWithClientCA(t *testing.T) {
	dir := t.TempDir()
	caPath := filepath.Join(dir, "ca.pem")
	if err := writeTestCert(caPath); err != nil {
		t.Fatalf("write cert: %v", err)
	}

	cfg, err := NewServerTLSConfig(ServerSettings{
		Enabled:  true,
		CertFile: "server.pem",
		KeyFile:  "server.key",
		ClientCA: caPath,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg == nil || cfg.ClientCAs == nil || cfg.ClientAuth == 0 {
		t.Fatalf("expected client ca settings")
	}
	if cfg.MinVersion != cfg.MaxVersion {
		t.Fatalf("expected tls 1.3 only")
	}
}

func TestNewClientTLSConfigWithCA(t *testing.T) {
	dir := t.TempDir()
	caPath := filepath.Join(dir, "ca.pem")
	if err := writeTestCert(caPath); err != nil {
		t.Fatalf("write cert: %v", err)
	}

	cfg, err := NewClientTLSConfig(ClientSettings{
		CAFile:             caPath,
		InsecureSkipVerify: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg == nil || cfg.RootCAs == nil {
		t.Fatalf("expected root ca")
	}
	if !cfg.InsecureSkipVerify {
		t.Fatalf("expected insecure flag")
	}
}

func writeTestCert(path string) error {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return err
	}
	serial, err := rand.Int(rand.Reader, big.NewInt(100000))
	if err != nil {
		return err
	}

	template := x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			CommonName: "test-ca",
		},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	der, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return err
	}
	block := &pem.Block{Type: "CERTIFICATE", Bytes: der}
	return os.WriteFile(path, pem.EncodeToMemory(block), 0o600)
}
