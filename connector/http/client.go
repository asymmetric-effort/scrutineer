package http

import (
	"net/http"
	"time"
)

// buildClient creates an *http.Client configured from the setup parameters.
// HTTP/2 is enabled automatically by Go's net/http when TLS is used.
func buildClient(config map[string]any, timeout time.Duration) (*http.Client, error) {
	tlsCfg, err := buildTLSConfig(config)
	if err != nil {
		return nil, err
	}

	transport := &http.Transport{
		TLSClientConfig: tlsCfg,
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   timeout,
	}

	return client, nil
}
