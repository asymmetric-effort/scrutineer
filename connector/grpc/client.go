package grpc

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

// dialConnection creates a gRPC client connection based on the provided config.
func dialConnection(ctx context.Context, endpoint string, config map[string]any) (*grpc.ClientConn, error) {
	opts, err := buildDialOptions(config)
	if err != nil {
		return nil, err
	}
	return grpc.NewClient(endpoint, opts...)
}

// buildDialOptions constructs gRPC dial options from the config map.
func buildDialOptions(config map[string]any) ([]grpc.DialOption, error) {
	var opts []grpc.DialOption

	useTLS := getBool(config, "tls", false)
	plaintext := getBool(config, "plaintext", false)

	if plaintext || !useTLS {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
		return opts, nil
	}

	tlsConfig, err := buildTLSConfig(config)
	if err != nil {
		return nil, fmt.Errorf("building TLS config: %w", err)
	}
	opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)))
	return opts, nil
}

// buildTLSConfig creates a tls.Config from the config map.
func buildTLSConfig(config map[string]any) (*tls.Config, error) {
	tlsCfg := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}

	if getBool(config, "tls_skip_verify", false) {
		tlsCfg.InsecureSkipVerify = true
	}

	if caFile, ok := config["tls_ca_file"]; ok {
		caPath, ok := caFile.(string)
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
		tlsCfg.RootCAs = pool
	}

	return tlsCfg, nil
}

// getBool extracts a bool from the config map with a default value.
func getBool(config map[string]any, key string, defaultVal bool) bool {
	v, ok := config[key]
	if !ok {
		return defaultVal
	}
	b, ok := v.(bool)
	if !ok {
		return defaultVal
	}
	return b
}

// getString extracts a string from the config map with a default value.
func getString(config map[string]any, key string, defaultVal string) string {
	v, ok := config[key]
	if !ok {
		return defaultVal
	}
	s, ok := v.(string)
	if !ok {
		return defaultVal
	}
	return s
}
