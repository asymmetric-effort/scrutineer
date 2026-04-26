package http

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
)

// buildTLSConfig constructs a *tls.Config from the connector configuration.
// It enforces TLS 1.2 as the minimum version and supports TLS 1.3.
func buildTLSConfig(config map[string]any) (*tls.Config, error) {
	cfg := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}

	// InsecureSkipVerify for self-signed certificates.
	if v, ok := config["tls_skip_verify"]; ok {
		b, ok := v.(bool)
		if !ok {
			return nil, fmt.Errorf("tls_skip_verify must be a bool")
		}
		cfg.InsecureSkipVerify = b
	}

	// Custom CA file.
	if v, ok := config["tls_ca_file"]; ok {
		caPath, ok := v.(string)
		if !ok {
			return nil, fmt.Errorf("tls_ca_file must be a string")
		}
		caPEM, err := os.ReadFile(caPath)
		if err != nil {
			return nil, fmt.Errorf("reading CA file: %w", err)
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(caPEM) {
			return nil, fmt.Errorf("failed to parse CA certificate")
		}
		cfg.RootCAs = pool
	}

	// mTLS client certificate and key.
	certFile, hasCert := config["tls_cert_file"]
	keyFile, hasKey := config["tls_key_file"]
	if hasCert && hasKey {
		certPath, ok1 := certFile.(string)
		keyPath, ok2 := keyFile.(string)
		if !ok1 || !ok2 {
			return nil, fmt.Errorf("tls_cert_file and tls_key_file must be strings")
		}
		cert, err := tls.LoadX509KeyPair(certPath, keyPath)
		if err != nil {
			return nil, fmt.Errorf("loading client certificate: %w", err)
		}
		cfg.Certificates = []tls.Certificate{cert}
	} else if hasCert != hasKey {
		return nil, fmt.Errorf("both tls_cert_file and tls_key_file must be provided together")
	}

	return cfg, nil
}
