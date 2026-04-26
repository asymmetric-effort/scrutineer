// Package graphql provides GraphQL query execution, subscription, and
// introspection support built on top of the standard library HTTP client.
// It implements the GraphQL-over-HTTP specification for queries and mutations,
// the graphql-ws WebSocket sub-protocol for subscriptions, and standard
// introspection for schema discovery.
package graphql

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// Request represents a GraphQL request.
type Request struct {
	Query         string         `json:"query"`
	Variables     map[string]any `json:"variables,omitempty"`
	OperationName string         `json:"operationName,omitempty"`
}

// Response represents a GraphQL response.
type Response struct {
	Data   any            `json:"data"`
	Errors []GraphQLError `json:"errors,omitempty"`
}

// GraphQLError represents a GraphQL error.
type GraphQLError struct {
	Message    string         `json:"message"`
	Locations  []Location     `json:"locations,omitempty"`
	Path       []any          `json:"path,omitempty"`
	Extensions map[string]any `json:"extensions,omitempty"`
}

// Location identifies a position in the GraphQL query source.
type Location struct {
	Line   int `json:"line"`
	Column int `json:"column"`
}

// Execute sends a GraphQL request via HTTP POST and returns the parsed response.
// The endpoint must be a fully qualified URL (e.g. "https://api.example.com/graphql").
// Optional headers are added to the request. The provided http.Client is used for
// the underlying HTTP call; if nil, http.DefaultClient is used.
func Execute(ctx context.Context, client *http.Client, endpoint string, req Request, headers map[string]string) (*Response, error) {
	if client == nil {
		client = http.DefaultClient
	}

	// json.Marshal cannot fail for the Request type (string + map + string fields).
	body, _ := json.Marshal(req)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("creating http request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	for k, v := range headers {
		httpReq.Header.Set(k, v)
	}

	httpResp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("executing http request: %w", err)
	}
	defer httpResp.Body.Close()

	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}

	var gqlResp Response
	if err := json.Unmarshal(respBody, &gqlResp); err != nil {
		return nil, fmt.Errorf("decoding graphql response: %w", err)
	}

	return &gqlResp, nil
}
