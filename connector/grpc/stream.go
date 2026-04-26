package grpc

import (
	"context"
	"fmt"

	"github.com/scrutineer/scrutineer/core/connector"
	"google.golang.org/grpc"
	grpcmd "google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
)

// resolveAndPrepare resolves the service/method and extracts the single message parameter.
func (c *GRPCConnector) resolveAndPrepare(ctx context.Context, params map[string]any) (
	fullMethod string, inputDesc, outputDesc protoreflect.MessageDescriptor, outCtx context.Context, err error,
) {
	service := getString(params, "service", "")
	method := getString(params, "method", "")
	if service == "" || method == "" {
		return "", nil, nil, ctx, fmt.Errorf("service and method are required")
	}

	rc := newReflectionClient(c.conn)
	fullMethod, inputDesc, outputDesc, err = rc.resolveMethod(ctx, service, method)
	if err != nil {
		return "", nil, nil, ctx, fmt.Errorf("resolving method: %w", err)
	}

	md := buildOutgoingMetadata(params)
	outCtx = grpcmd.NewOutgoingContext(ctx, md)
	return fullMethod, inputDesc, outputDesc, outCtx, nil
}

// buildStreamResult constructs a Result for streaming operations.
func buildStreamResult(action, service, method string, responses []any,
	headerMD, trailerMD grpcmd.MD, rpcErr error,
) *connector.Result {
	code, msg := extractStatus(rpcErr)
	return &connector.Result{
		Data: map[string]any{
			"responses":      responses,
			"status_code":    int(code),
			"status_message": msg,
			"status_name":    statusCodeName(code),
			"metadata":       metadataToMap(headerMD),
			"trailers":       metadataToMap(trailerMD),
		},
		Meta: map[string]string{
			"connector": "grpc",
			"action":    action,
			"service":   service,
			"method":    method,
		},
	}
}

// executeServerStream handles the "server_stream" action.
func (c *GRPCConnector) executeServerStream(ctx context.Context, step connector.Step) (*connector.Result, error) {
	fullMethod, inputDesc, outputDesc, ctx, err := c.resolveAndPrepare(ctx, step.Parameters)
	if err != nil {
		return nil, err
	}

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

	stream, err := c.conn.NewStream(ctx, &grpc.StreamDesc{ServerStreams: true}, fullMethod)
	if err != nil {
		return nil, fmt.Errorf("creating server stream: %w", err)
	}

	if err := stream.SendMsg(reqMsg); err != nil {
		return nil, fmt.Errorf("sending message: %w", err)
	}
	_ = stream.CloseSend()

	messages, recvErr := receiveStream(stream, outputDesc)
	headerMD, _ := stream.Header()

	responses := make([]any, len(messages))
	for i, m := range messages {
		responses[i] = dynamicMessageToMap(m)
	}

	service := getString(step.Parameters, "service", "")
	method := getString(step.Parameters, "method", "")
	return buildStreamResult("server_stream", service, method, responses,
		headerMD, stream.Trailer(), recvErr), nil
}

// executeClientStream handles the "client_stream" action.
func (c *GRPCConnector) executeClientStream(ctx context.Context, step connector.Step) (*connector.Result, error) {
	fullMethod, inputDesc, outputDesc, ctx, err := c.resolveAndPrepare(ctx, step.Parameters)
	if err != nil {
		return nil, err
	}

	messagesRaw, err := getMessagesList(step.Parameters)
	if err != nil {
		return nil, err
	}

	stream, err := c.conn.NewStream(ctx, &grpc.StreamDesc{ClientStreams: true}, fullMethod)
	if err != nil {
		return nil, fmt.Errorf("creating client stream: %w", err)
	}

	for i, msgData := range messagesRaw {
		reqMsg, err := buildDynamicMessage(inputDesc, msgData)
		if err != nil {
			return nil, fmt.Errorf("building message %d: %w", i, err)
		}
		if err := stream.SendMsg(reqMsg); err != nil {
			return nil, fmt.Errorf("sending message %d: %w", i, err)
		}
	}
	_ = stream.CloseSend()

	respMsg := dynamicpb.NewMessage(outputDesc)
	rpcErr := stream.RecvMsg(respMsg)
	headerMD, _ := stream.Header()
	code, msg := extractStatus(rpcErr)

	service := getString(step.Parameters, "service", "")
	method := getString(step.Parameters, "method", "")
	result := &connector.Result{
		Data: map[string]any{
			"status_code":    int(code),
			"status_message": msg,
			"status_name":    statusCodeName(code),
			"metadata":       metadataToMap(headerMD),
			"trailers":       metadataToMap(stream.Trailer()),
		},
		Meta: map[string]string{
			"connector": "grpc",
			"action":    "client_stream",
			"service":   service,
			"method":    method,
		},
	}

	if rpcErr == nil {
		result.Data["response"] = dynamicMessageToMap(respMsg)
	}

	return result, nil
}

// executeBidiStream handles the "bidi_stream" action.
func (c *GRPCConnector) executeBidiStream(ctx context.Context, step connector.Step) (*connector.Result, error) {
	fullMethod, inputDesc, outputDesc, ctx, err := c.resolveAndPrepare(ctx, step.Parameters)
	if err != nil {
		return nil, err
	}

	messagesRaw, err := getMessagesList(step.Parameters)
	if err != nil {
		return nil, err
	}

	stream, err := c.conn.NewStream(ctx, &grpc.StreamDesc{ServerStreams: true, ClientStreams: true}, fullMethod)
	if err != nil {
		return nil, fmt.Errorf("creating bidi stream: %w", err)
	}

	for i, msgData := range messagesRaw {
		reqMsg, err := buildDynamicMessage(inputDesc, msgData)
		if err != nil {
			return nil, fmt.Errorf("building message %d: %w", i, err)
		}
		if err := stream.SendMsg(reqMsg); err != nil {
			return nil, fmt.Errorf("sending message %d: %w", i, err)
		}
	}
	_ = stream.CloseSend()

	messages, recvErr := receiveStream(stream, outputDesc)
	headerMD, _ := stream.Header()

	responses := make([]any, len(messages))
	for i, m := range messages {
		responses[i] = dynamicMessageToMap(m)
	}

	service := getString(step.Parameters, "service", "")
	method := getString(step.Parameters, "method", "")
	return buildStreamResult("bidi_stream", service, method, responses,
		headerMD, stream.Trailer(), recvErr), nil
}

// getMessagesList extracts the "messages" parameter as a list of maps.
func getMessagesList(params map[string]any) ([]map[string]any, error) {
	v, ok := params["messages"]
	if !ok {
		return nil, fmt.Errorf("messages parameter is required for streaming actions")
	}
	rawList, ok := v.([]any)
	if !ok {
		return nil, fmt.Errorf("messages must be a list")
	}
	var messages []map[string]any
	for i, item := range rawList {
		m, ok := item.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("messages[%d] must be a map", i)
		}
		messages = append(messages, m)
	}
	return messages, nil
}
