package http

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestBuildTLSConfigSkipVerify(t *testing.T) {
	cfg, err := buildTLSConfig(map[string]any{
		"tls_skip_verify": true,
	})
	if err != nil {
		t.Fatalf("buildTLSConfig() error: %v", err)
	}
	if !cfg.InsecureSkipVerify {
		t.Fatal("InsecureSkipVerify should be true")
	}
}

func TestBuildTLSConfigSkipVerifyInvalidType(t *testing.T) {
	_, err := buildTLSConfig(map[string]any{
		"tls_skip_verify": "yes",
	})
	if err == nil {
		t.Fatal("expected error for non-bool tls_skip_verify")
	}
}

func TestBuildTLSConfigMinVersionTLS12(t *testing.T) {
	cfg, err := buildTLSConfig(map[string]any{})
	if err != nil {
		t.Fatalf("buildTLSConfig() error: %v", err)
	}
	if cfg.MinVersion != tls.VersionTLS12 {
		t.Fatalf("MinVersion = %d, want %d (TLS 1.2)", cfg.MinVersion, tls.VersionTLS12)
	}
}

func TestBuildTLSConfigDefault(t *testing.T) {
	cfg, err := buildTLSConfig(map[string]any{})
	if err != nil {
		t.Fatalf("buildTLSConfig() error: %v", err)
	}
	if cfg.InsecureSkipVerify {
		t.Fatal("InsecureSkipVerify should be false by default")
	}
	if cfg.RootCAs != nil {
		t.Fatal("RootCAs should be nil by default")
	}
	if len(cfg.Certificates) != 0 {
		t.Fatal("Certificates should be empty by default")
	}
}

func TestBuildTLSConfigWithCustomCA(t *testing.T) {
	// Generate a self-signed CA.
	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generating CA key: %v", err)
	}

	caTemplate := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "Test CA"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
	}

	caCertDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	if err != nil {
		t.Fatalf("creating CA cert: %v", err)
	}

	tmpDir := t.TempDir()
	caFile := filepath.Join(tmpDir, "ca.pem")
	f, err := os.Create(caFile)
	if err != nil {
		t.Fatalf("creating temp CA file: %v", err)
	}
	_ = pem.Encode(f, &pem.Block{Type: "CERTIFICATE", Bytes: caCertDER})
	f.Close()

	cfg, err := buildTLSConfig(map[string]any{
		"tls_ca_file": caFile,
	})
	if err != nil {
		t.Fatalf("buildTLSConfig() error: %v", err)
	}
	if cfg.RootCAs == nil {
		t.Fatal("RootCAs should not be nil")
	}
}

func TestBuildTLSConfigWithInvalidCA(t *testing.T) {
	tmpDir := t.TempDir()
	caFile := filepath.Join(tmpDir, "bad-ca.pem")
	_ = os.WriteFile(caFile, []byte("not a cert"), 0644)

	_, err := buildTLSConfig(map[string]any{
		"tls_ca_file": caFile,
	})
	if err == nil {
		t.Fatal("expected error for invalid CA file")
	}
}

func TestBuildTLSConfigCAFileNotFound(t *testing.T) {
	_, err := buildTLSConfig(map[string]any{
		"tls_ca_file": "/nonexistent/ca.pem",
	})
	if err == nil {
		t.Fatal("expected error for missing CA file")
	}
}

func TestBuildTLSConfigCAFileInvalidType(t *testing.T) {
	_, err := buildTLSConfig(map[string]any{
		"tls_ca_file": 123,
	})
	if err == nil {
		t.Fatal("expected error for non-string tls_ca_file")
	}
}

func TestBuildTLSConfigWithClientCert(t *testing.T) {
	tmpDir := t.TempDir()

	// Generate a self-signed cert/key pair for mTLS.
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generating key: %v", err)
	}

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "Test Client"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("creating cert: %v", err)
	}

	certFile := filepath.Join(tmpDir, "client.pem")
	cf, _ := os.Create(certFile)
	_ = pem.Encode(cf, &pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	cf.Close()

	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		t.Fatalf("marshalling key: %v", err)
	}
	keyFile := filepath.Join(tmpDir, "client-key.pem")
	kf, _ := os.Create(keyFile)
	_ = pem.Encode(kf, &pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
	kf.Close()

	cfg, err := buildTLSConfig(map[string]any{
		"tls_cert_file": certFile,
		"tls_key_file":  keyFile,
	})
	if err != nil {
		t.Fatalf("buildTLSConfig() error: %v", err)
	}
	if len(cfg.Certificates) != 1 {
		t.Fatalf("expected 1 certificate, got %d", len(cfg.Certificates))
	}
}

func TestBuildTLSConfigCertWithoutKey(t *testing.T) {
	_, err := buildTLSConfig(map[string]any{
		"tls_cert_file": "/some/cert.pem",
	})
	if err == nil {
		t.Fatal("expected error when cert provided without key")
	}
}

func TestBuildTLSConfigKeyWithoutCert(t *testing.T) {
	_, err := buildTLSConfig(map[string]any{
		"tls_key_file": "/some/key.pem",
	})
	if err == nil {
		t.Fatal("expected error when key provided without cert")
	}
}

func TestBuildTLSConfigCertKeyInvalidTypes(t *testing.T) {
	_, err := buildTLSConfig(map[string]any{
		"tls_cert_file": 123,
		"tls_key_file":  456,
	})
	if err == nil {
		t.Fatal("expected error for non-string cert/key files")
	}
}

func TestBuildTLSConfigInvalidCertKeyFiles(t *testing.T) {
	_, err := buildTLSConfig(map[string]any{
		"tls_cert_file": "/nonexistent/cert.pem",
		"tls_key_file":  "/nonexistent/key.pem",
	})
	if err == nil {
		t.Fatal("expected error for missing cert/key files")
	}
}
