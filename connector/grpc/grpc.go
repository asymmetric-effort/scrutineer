// Package grpc implements a gRPC connector for the scrutineer test engine.
//
// It supports unary RPCs, client streaming, server streaming, and
// bidirectional streaming via dynamic message construction (no compiled
// protobuf stubs required). Service and method discovery is performed
// through gRPC server reflection.
package grpc

import (
	"context"
	"fmt"
	"time"

	"github.com/scrutineer/scrutineer/core/connector"
	"google.golang.org/grpc"
)

// GRPCConnector implements connector.Connector for gRPC-based testing.
type GRPCConnector struct {
	conn     *grpc.ClientConn
	endpoint string
	config   map[string]any
}

// Ensure GRPCConnector satisfies the Connector interface.
var _ connector.Connector = (*GRPCConnector)(nil)

// New creates a new GRPCConnector with default settings.
func New() *GRPCConnector {
	return &GRPCConnector{}
}

// Name returns the connector identifier.
func (c *GRPCConnector) Name() string {
	return "grpc"
}

// Setup initializes the connector from the provided configuration map.
// Accepted keys: "endpoint" (string), "tls" (bool), "tls_skip_verify" (bool),
// "tls_ca_file" (string), "schema" (string, path to .proto file — for future use),
// "plaintext" (bool, default false).
func (c *GRPCConnector) Setup(ctx context.Context, config map[string]any) error {
	if config == nil {
		return fmt.Errorf("config must not be nil")
	}

	v, ok := config["endpoint"]
	if !ok {
		return fmt.Errorf("endpoint is required")
	}
	endpoint, ok := v.(string)
	if !ok || endpoint == "" {
		return fmt.Errorf("endpoint must be a non-empty string")
	}
	c.endpoint = endpoint
	c.config = config

	conn, err := dialConnection(ctx, endpoint, config)
	if err != nil {
		return fmt.Errorf("dialing gRPC endpoint: %w", err)
	}
	c.conn = conn
	return nil
}

// Execute runs a single test step. Supports actions: "unary", "server_stream",
// "client_stream", "bidi_stream".
func (c *GRPCConnector) Execute(ctx context.Context, step connector.Step) (*connector.Result, error) {
	if c.conn == nil {
		return nil, fmt.Errorf("connector not set up; call Setup first")
	}

	start := time.Now()
	var result *connector.Result
	var err error

	switch step.Action {
	case "unary":
		result, err = c.executeUnary(ctx, step)
	case "server_stream":
		result, err = c.executeServerStream(ctx, step)
	case "client_stream":
		result, err = c.executeClientStream(ctx, step)
	case "bidi_stream":
		result, err = c.executeBidiStream(ctx, step)
	default:
		return nil, fmt.Errorf("unsupported action: %s", step.Action)
	}

	if err != nil {
		return nil, err
	}
	result.Elapsed = time.Since(start)
	return result, nil
}

// Teardown cleans up resources held by the connector.
func (c *GRPCConnector) Teardown(_ context.Context) error {
	if c.conn != nil {
		err := c.conn.Close()
		c.conn = nil
		return err
	}
	return nil
}
