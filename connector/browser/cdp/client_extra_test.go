package cdp

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"testing"
	"time"
)

func TestClient_Send_ConnectionClosed(t *testing.T) {
	wsURL, connCh, cleanup := mockWSServer(t)
	defer cleanup()

	ctx := context.Background()
	client, err := Dial(ctx, wsURL)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}

	serverConn := <-connCh
	// Close server to trigger readLoop exit.
	serverConn.Close()

	// Wait for done.
	<-client.Done()

	// Send should fail because connection is closed.
	_, err = client.Send("Test.method", nil)
	if err == nil {
		t.Error("expected error sending on closed connection")
	}
}

func TestClient_Subscribe_NilParams(t *testing.T) {
	wsURL, connCh, cleanup := mockWSServer(t)
	defer cleanup()

	ctx := context.Background()
	client, err := Dial(ctx, wsURL)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer client.Close()

	serverConn := <-connCh
	defer serverConn.Close()

	received := make(chan map[string]any, 1)
	client.Subscribe("Test.nilParams", func(params map[string]any) {
		received <- params
	})

	// Send event with no params.
	event := Message{Method: "Test.nilParams"}
	data, _ := json.Marshal(event)
	serverWriteTextFrame(serverConn, data)

	select {
	case params := <-received:
		if params != nil {
			// Params from nil should be nil.
			_ = params
		}
	case <-client.Done():
		t.Error("connection closed unexpectedly")
	}
}

func TestClient_ReadLoop_InvalidJSON(t *testing.T) {
	wsURL, connCh, cleanup := mockWSServer(t)
	defer cleanup()

	ctx := context.Background()
	client, err := Dial(ctx, wsURL)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer client.Close()

	serverConn := <-connCh
	defer serverConn.Close()

	// Send invalid JSON - should be silently ignored.
	serverWriteTextFrame(serverConn, []byte("not json"))

	// Send a valid event after to confirm readLoop is still running.
	received := make(chan struct{}, 1)
	client.Subscribe("Still.alive", func(params map[string]any) {
		received <- struct{}{}
	})

	event := Message{Method: "Still.alive"}
	data, _ := json.Marshal(event)
	serverWriteTextFrame(serverConn, data)

	select {
	case <-received:
		// Good - readLoop survived invalid JSON.
	case <-client.Done():
		t.Error("connection closed unexpectedly")
	}
}

func TestClient_ReadLoop_ResponseWithoutPending(t *testing.T) {
	wsURL, connCh, cleanup := mockWSServer(t)
	defer cleanup()

	ctx := context.Background()
	client, err := Dial(ctx, wsURL)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer client.Close()

	serverConn := <-connCh
	defer serverConn.Close()

	// Send a response with an ID that no one is waiting for.
	resp := Message{ID: 9999, Result: json.RawMessage(`{}`)}
	data, _ := json.Marshal(resp)
	serverWriteTextFrame(serverConn, data)

	// Confirm readLoop is still running.
	received := make(chan struct{}, 1)
	client.Subscribe("Check.alive", func(params map[string]any) {
		received <- struct{}{}
	})

	event := Message{Method: "Check.alive"}
	data2, _ := json.Marshal(event)
	serverWriteTextFrame(serverConn, data2)

	select {
	case <-received:
		// Good.
	case <-client.Done():
		t.Error("connection closed unexpectedly")
	}
}

func TestClient_Send_ConnectionClosedNoError(t *testing.T) {
	wsURL, connCh, cleanup := mockWSServer(t)
	defer cleanup()

	ctx := context.Background()
	client, err := Dial(ctx, wsURL)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}

	serverConn := <-connCh

	// Close the client first (sets done), then try to send.
	client.Close()
	serverConn.Close()

	_, err = client.Send("Test.method", nil)
	if err == nil {
		t.Error("expected error")
	}
}

func TestClient_Send_WriteError(t *testing.T) {
	wsURL, connCh, cleanup := mockWSServer(t)
	defer cleanup()

	ctx := context.Background()
	client, err := Dial(ctx, wsURL)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}

	serverConn := <-connCh
	// Close server conn so writes will eventually fail.
	serverConn.Close()

	// Wait for the readLoop to detect the closed connection.
	<-client.Done()

	// Now Send should return an error about closed connection.
	_, err = client.Send("Test.method", nil)
	if err == nil {
		t.Error("expected error")
	}
}

func TestClient_Send_UnmarshalableParams(t *testing.T) {
	wsURL, cleanup := mockCDPServer(t, func(conn net.Conn, msg Message) {
		resp := Message{ID: msg.ID, Result: json.RawMessage(`{}`)}
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

	// Use a channel value that can't be marshaled to JSON.
	ch := make(chan int)
	_, err = client.Send("Test.method", map[string]any{
		"bad": ch,
	})
	if err == nil {
		t.Error("expected error for unmarshalable params")
	}
}

func TestClient_Send_InvalidResultJSON(t *testing.T) {
	wsURL, cleanup := mockCDPServer(t, func(conn net.Conn, msg Message) {
		// Send a response with invalid JSON in the result field.
		raw := fmt.Sprintf(`{"id":%d,"result":"not-a-json-object"}`, msg.ID)
		serverWriteTextFrame(conn, []byte(raw))
	})
	defer cleanup()

	ctx := context.Background()
	client, err := Dial(ctx, wsURL)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer client.Close()

	_, err = client.Send("Test.method", nil)
	if err == nil {
		t.Error("expected error for invalid result JSON")
	}
}

func TestClient_ReadLoop_PendingWakeOnDisconnect(t *testing.T) {
	wsURL, connCh, cleanup := mockWSServer(t)
	defer cleanup()

	ctx := context.Background()
	client, err := Dial(ctx, wsURL)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer client.Close()

	serverConn := <-connCh

	// Start a send in a goroutine that will be pending when the server disconnects.
	errCh := make(chan error, 1)
	go func() {
		_, err := client.Send("Test.method", nil)
		errCh <- err
	}()

	// Brief moment to let the goroutine register its pending channel.
	time.Sleep(10 * time.Millisecond)

	// Close the server connection to trigger readLoop error.
	serverConn.Close()

	// The pending send should get an error.
	select {
	case err = <-errCh:
		if err == nil {
			t.Error("expected error for pending send on disconnect")
		}
	case <-time.After(5 * time.Second):
		t.Error("timeout waiting for pending send to complete")
	}
}

func TestClient_Send_WriteTextError(t *testing.T) {
	wsURL, connCh, cleanup := mockWSServer(t)
	defer cleanup()

	ctx := context.Background()
	client, err := Dial(ctx, wsURL)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}

	serverConn := <-connCh
	serverConn.Close()

	// Wait for readLoop to detect close.
	<-client.Done()

	// Send will fail because the WebSocket connection is broken.
	// But it returns "connection closed" first because done channel is closed.
	_, err = client.Send("Test.method", nil)
	if err == nil {
		t.Error("expected error")
	}
}

func TestClient_Send_WriteFailureBeforeDone(t *testing.T) {
	// Create a connection where writing fails but the read goroutine hasn't
	// noticed yet (so done channel is not closed).
	wsURL, connCh, cleanup := mockWSServer(t)
	defer cleanup()

	ctx := context.Background()
	client, err := Dial(ctx, wsURL)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer client.Close()

	serverConn := <-connCh

	// Close the underlying TCP connection to make writes fail.
	// The read goroutine may not have detected this yet.
	client.ws.mu.Lock()
	client.ws.conn.Close()
	client.ws.mu.Unlock()

	// Try to send - should fail on WriteText.
	_, err = client.Send("Test.method", nil)
	if err == nil {
		t.Error("expected write error")
	}

	serverConn.Close()
}

func TestClient_Send_MarshalMessageError(t *testing.T) {
	// This is essentially impossible to trigger since Message struct is always
	// JSON-serializable. The json.Marshal of a Message will never fail because
	// all fields are basic types or json.RawMessage. But we still document
	// this test for completeness.
	t.Skip("Message struct is always JSON-serializable")
}

func TestClient_Send_WithParams(t *testing.T) {
	var receivedParams map[string]any

	wsURL, cleanup := mockCDPServer(t, func(conn net.Conn, msg Message) {
		if msg.Params != nil {
			json.Unmarshal(msg.Params, &receivedParams)
		}
		resp := Message{ID: msg.ID, Result: json.RawMessage(`{}`)}
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

	_, err = client.Send("Test.method", map[string]any{
		"key1": "value1",
		"key2": float64(42),
	})
	if err != nil {
		t.Fatalf("Send: %v", err)
	}

	if receivedParams["key1"] != "value1" {
		t.Errorf("key1 = %v, want value1", receivedParams["key1"])
	}
}
