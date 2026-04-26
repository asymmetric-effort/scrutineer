// Package http implements an HTTP connector for the scrutineer test engine.
package http

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/scrutineer/scrutineer/core/connector"
)

// HTTPConnector implements connector.Connector for HTTP-based testing.
type HTTPConnector struct {
	baseURL string
	client  *http.Client
	headers map[string]string
}

// Ensure HTTPConnector satisfies the Connector interface.
var _ connector.Connector = (*HTTPConnector)(nil)

// New creates a new HTTPConnector with default settings.
func New() *HTTPConnector {
	return &HTTPConnector{
		headers: make(map[string]string),
	}
}

// Name returns the connector identifier.
func (c *HTTPConnector) Name() string {
	return "http"
}

// Setup initializes the connector from the provided configuration map.
// Accepted keys: "base_url", "default_headers", "timeout",
// "tls_skip_verify", "tls_ca_file", "tls_cert_file", "tls_key_file".
func (c *HTTPConnector) Setup(_ context.Context, config map[string]any) error {
	if v, ok := config["base_url"]; ok {
		s, ok := v.(string)
		if !ok {
			return fmt.Errorf("base_url must be a string")
		}
		c.baseURL = s
	}

	if v, ok := config["default_headers"]; ok {
		switch hdr := v.(type) {
		case map[string]any:
			for k, val := range hdr {
				c.headers[k] = fmt.Sprintf("%v", val)
			}
		case map[string]string:
			for k, val := range hdr {
				c.headers[k] = val
			}
		default:
			return fmt.Errorf("default_headers must be a map")
		}
	}

	var timeout time.Duration
	if v, ok := config["timeout"]; ok {
		s, ok := v.(string)
		if !ok {
			return fmt.Errorf("timeout must be a string duration")
		}
		d, err := time.ParseDuration(s)
		if err != nil {
			return fmt.Errorf("invalid timeout: %w", err)
		}
		timeout = d
	}

	client, err := buildClient(config, timeout)
	if err != nil {
		return fmt.Errorf("building HTTP client: %w", err)
	}
	c.client = client

	return nil
}

// Execute runs a single test step. Currently supports action "request".
func (c *HTTPConnector) Execute(ctx context.Context, step connector.Step) (*connector.Result, error) {
	switch step.Action {
	case "request":
		return c.doRequest(ctx, step)
	default:
		return nil, fmt.Errorf("unsupported action: %s", step.Action)
	}
}

// Teardown cleans up resources held by the connector.
func (c *HTTPConnector) Teardown(_ context.Context) error {
	if c.client != nil {
		c.client.CloseIdleConnections()
	}
	return nil
}
