// Package cdp implements a Chrome DevTools Protocol client over WebSocket.
//
// The client communicates with Chromium-based browsers using the CDP JSON-RPC
// protocol over a minimal WebSocket implementation built on the Go standard
// library. It supports multi-target sessions, event subscriptions, and
// concurrent message handling.
package cdp

import "encoding/json"

// Message represents a CDP JSON-RPC message (request, response, or event).
type Message struct {
	ID        int64           `json:"id,omitempty"`
	Method    string          `json:"method,omitempty"`
	Params    json.RawMessage `json:"params,omitempty"`
	Result    json.RawMessage `json:"result,omitempty"`
	Error     *ErrorInfo      `json:"error,omitempty"`
	SessionID string          `json:"sessionId,omitempty"`
}

// ErrorInfo holds error details from a CDP response.
type ErrorInfo struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    string `json:"data,omitempty"`
}

// Error implements the error interface.
func (e *ErrorInfo) Error() string {
	if e.Data != "" {
		return e.Message + ": " + e.Data
	}
	return e.Message
}
