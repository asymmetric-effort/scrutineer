package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/scrutineer/scrutineer/core/connector"
)

// doRequest executes an HTTP request step and returns the result.
func (c *HTTPConnector) doRequest(ctx context.Context, step connector.Step) (*connector.Result, error) {
	params := step.Parameters

	method, ok := params["method"].(string)
	if !ok || method == "" {
		return nil, fmt.Errorf("missing or invalid 'method' parameter")
	}

	path, ok := params["path"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid 'path' parameter")
	}

	// Build URL.
	fullURL := c.baseURL + path

	// Apply query parameters.
	if q, ok := params["query"]; ok {
		qMap, ok := q.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("query must be a map")
		}
		u, err := url.Parse(fullURL)
		if err != nil {
			return nil, fmt.Errorf("parsing URL: %w", err)
		}
		qv := u.Query()
		for k, v := range qMap {
			qv.Set(k, fmt.Sprintf("%v", v))
		}
		u.RawQuery = qv.Encode()
		fullURL = u.String()
	}

	// Build body.
	var bodyReader io.Reader
	if body, ok := params["body"]; ok {
		switch b := body.(type) {
		case string:
			bodyReader = strings.NewReader(b)
		case map[string]any:
			data, err := json.Marshal(b)
			if err != nil {
				return nil, fmt.Errorf("marshalling body: %w", err)
			}
			bodyReader = bytes.NewReader(data)
		default:
			data, err := json.Marshal(b)
			if err != nil {
				return nil, fmt.Errorf("marshalling body: %w", err)
			}
			bodyReader = bytes.NewReader(data)
		}
	}

	// Apply step timeout if set.
	if step.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, step.Timeout)
		defer cancel()
	}

	req, err := http.NewRequestWithContext(ctx, strings.ToUpper(method), fullURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	// Apply default headers.
	for k, v := range c.headers {
		req.Header.Set(k, v)
	}

	// Apply step-specific headers (override defaults).
	if h, ok := params["headers"]; ok {
		hMap, ok := h.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("headers must be a map")
		}
		for k, v := range hMap {
			req.Header.Set(k, fmt.Sprintf("%v", v))
		}
	}

	// Apply authentication.
	if err := applyAuth(req, params); err != nil {
		return nil, fmt.Errorf("applying auth: %w", err)
	}

	start := time.Now()
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()
	elapsed := time.Since(start)

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}

	// Convert response headers.
	respHeaders := make(map[string][]string)
	for k, v := range resp.Header {
		respHeaders[k] = v
	}

	// Try to parse body as JSON.
	var parsedBody any
	rawBody := string(respBody)
	if err := json.Unmarshal(respBody, &parsedBody); err != nil {
		parsedBody = rawBody
	}

	data := map[string]any{
		"status":      resp.StatusCode,
		"status_text": resp.Status,
		"headers":     respHeaders,
		"body":        parsedBody,
		"body_raw":    rawBody,
		"elapsed_ms":  float64(elapsed.Milliseconds()),
	}

	return &connector.Result{
		Data:    data,
		Elapsed: elapsed,
		Meta:    map[string]string{"url": fullURL, "method": strings.ToUpper(method)},
	}, nil
}
