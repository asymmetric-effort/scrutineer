package cdp

import (
	"context"
	"encoding/json"
	"net"
	"strings"
	"testing"
)

func TestNewSession_Success(t *testing.T) {
	wsURL, cleanup := mockCDPServer(t, func(conn net.Conn, msg Message) {
		if msg.Method == "Target.attachToTarget" {
			resp := Message{
				ID:     msg.ID,
				Result: json.RawMessage(`{"sessionId":"session-abc-123"}`),
			}
			data, _ := json.Marshal(resp)
			serverWriteTextFrame(conn, data)
		}
	})
	defer cleanup()

	ctx := context.Background()
	client, err := Dial(ctx, wsURL)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer client.Close()

	session, err := client.NewSession("target-1")
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}

	if session.SessionID() != "session-abc-123" {
		t.Errorf("SessionID = %q, want %q", session.SessionID(), "session-abc-123")
	}

	if session.TargetID() != "target-1" {
		t.Errorf("TargetID = %q, want %q", session.TargetID(), "target-1")
	}
}

func TestNewSession_Error(t *testing.T) {
	wsURL, cleanup := mockCDPServer(t, func(conn net.Conn, msg Message) {
		resp := Message{
			ID:    msg.ID,
			Error: &ErrorInfo{Code: -32000, Message: "Target not found"},
		}
		data, _ := json.Marshal(resp)
		serverWriteTextFrame(conn, data)
	})
	defer cleanup()

	ctx := context.Background()
	client, err := Dial(ctx, wsURL)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer client.Close()

	_, err = client.NewSession("invalid-target")
	if err == nil {
		t.Error("expected error for invalid target")
	}
}

func TestNewSession_NoSessionID(t *testing.T) {
	wsURL, cleanup := mockCDPServer(t, func(conn net.Conn, msg Message) {
		resp := Message{
			ID:     msg.ID,
			Result: json.RawMessage(`{}`),
		}
		data, _ := json.Marshal(resp)
		serverWriteTextFrame(conn, data)
	})
	defer cleanup()

	ctx := context.Background()
	client, err := Dial(ctx, wsURL)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer client.Close()

	_, err = client.NewSession("target-1")
	if err == nil {
		t.Error("expected error when no sessionId in response")
	}
	if !strings.Contains(err.Error(), "no sessionId") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestSession_Send(t *testing.T) {
	wsURL, cleanup := mockCDPServer(t, func(conn net.Conn, msg Message) {
		if msg.Method == "Target.attachToTarget" {
			resp := Message{
				ID:     msg.ID,
				Result: json.RawMessage(`{"sessionId":"sess-1"}`),
			}
			data, _ := json.Marshal(resp)
			serverWriteTextFrame(conn, data)
			return
		}

		// Verify session ID is included.
		if msg.SessionID != "sess-1" {
			resp := Message{
				ID:    msg.ID,
				Error: &ErrorInfo{Message: "wrong session"},
			}
			data, _ := json.Marshal(resp)
			serverWriteTextFrame(conn, data)
			return
		}

		resp := Message{
			ID:     msg.ID,
			Result: json.RawMessage(`{"value":42}`),
		}
		data, _ := json.Marshal(resp)
		serverWriteTextFrame(conn, data)
	})
	defer cleanup()

	ctx := context.Background()
	client, err := Dial(ctx, wsURL)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer client.Close()

	session, err := client.NewSession("target-1")
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}

	result, err := session.Send("Runtime.evaluate", map[string]any{
		"expression": "1 + 1",
	})
	if err != nil {
		t.Fatalf("Session.Send: %v", err)
	}

	if result["value"] != float64(42) {
		t.Errorf("value = %v, want 42", result["value"])
	}
}

func TestSession_Close(t *testing.T) {
	wsURL, cleanup := mockCDPServer(t, func(conn net.Conn, msg Message) {
		resp := Message{
			ID: msg.ID,
		}
		if msg.Method == "Target.attachToTarget" {
			resp.Result = json.RawMessage(`{"sessionId":"sess-1"}`)
		} else if msg.Method == "Target.detachFromTarget" {
			resp.Result = json.RawMessage(`{}`)
		}
		data, _ := json.Marshal(resp)
		serverWriteTextFrame(conn, data)
	})
	defer cleanup()

	ctx := context.Background()
	client, err := Dial(ctx, wsURL)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer client.Close()

	session, err := client.NewSession("target-1")
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}

	err = session.Close()
	if err != nil {
		t.Errorf("Session.Close: %v", err)
	}
}
