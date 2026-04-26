package grpc

import (
	"context"
	"fmt"

	"github.com/scrutineer/scrutineer/core/connector"
	"google.golang.org/grpc"
	grpcmd "google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/dynamicpb"
)

// executeUnary handles the "unary" action: a single request/response RPC.
// Parameters: "service" (string), "method" (string), "message" (map), "metadata" (map).
func (c *GRPCConnector) executeUnary(ctx context.Context, step connector.Step) (*connector.Result, error) {
	service := getString(step.Parameters, "service", "")
	method := getString(step.Parameters, "method", "")
	if service == "" || method == "" {
		return nil, fmt.Errorf("service and method are required for unary action")
	}

	// Resolve the method via reflection.
	rc := newReflectionClient(c.conn)
	fullMethod, inputDesc, outputDesc, err := rc.resolveMethod(ctx, service, method)
	if err != nil {
		return nil, fmt.Errorf("resolving method: %w", err)
	}

	// Build the request message.
	var msgData map[string]any
	if v, ok := step.Parameters["message"]; ok {
		m, ok := v.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("message must be a map")
		}
		msgData = m
	}

	reqMsg, err := buildDynamicMessage(inputDesc, msgData)
	if err != nil {
		return nil, fmt.Errorf("building request message: %w", err)
	}

	// Build outgoing metadata.
	md := buildOutgoingMetadata(step.Parameters)
	ctx = grpcmd.NewOutgoingContext(ctx, md)

	// Invoke the RPC.
	respMsg := dynamicpb.NewMessage(outputDesc)
	var headerMD, trailerMD grpcmd.MD
	rpcErr := c.conn.Invoke(ctx, fullMethod, reqMsg, respMsg,
		grpc.Header(&headerMD),
		grpc.Trailer(&trailerMD),
	)

	code, msg := extractStatus(rpcErr)

	result := &connector.Result{
		Data: map[string]any{
			"status_code":    int(code),
			"status_message": msg,
			"status_name":    statusCodeName(code),
			"metadata":       metadataToMap(headerMD),
			"trailers":       metadataToMap(trailerMD),
		},
		Meta: map[string]string{
			"connector": "grpc",
			"action":    "unary",
			"service":   service,
			"method":    method,
		},
	}

	if rpcErr == nil {
		result.Data["response"] = dynamicMessageToMap(respMsg)
	}

	return result, nil
}
