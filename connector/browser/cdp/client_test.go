package cdp

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"sync"
	"testing"
	"time"
)

// mockCDPServer creates a mock CDP server and returns the ws URL.
// The handler function receives each decoded message and sends responses.
func mockCDPServer(t *testing.T, handler func(conn net.Conn, msg Message)) (string, func()) {
	t.Helper()

	wsURL, connCh, cleanup := mockWSServer(t)

	go func() {
		conn := <-connCh
		defer conn.Close()

		for {
			opcode, payload, err := serverReadFrameBufConn(conn)
			if err != nil {
				return
			}
			if opcode == opClose {
				return
			}
			if opcode != opText {
				continue
			}

			var msg Message
			if err := json.Unmarshal(payload, &msg); err != nil {
				continue
			}

			handler(conn, msg)
		}
	}()

	return wsURL, cleanup
}

// serverReadFrameBufConn reads a frame without creating a new bufio.Reader each time.
// For testing, we create a simple synchronous reader.
func serverReadFrameBufConn(conn net.Conn) (byte, []byte, error) {
	buf := make([]byte, 2)
	if _, err := conn.Read(buf); err != nil {
		return 0, nil, err
	}

	opcode := buf[0] & 0x0F
	masked := buf[1]&0x80 != 0
	length := uint64(buf[1] & 0x7F)

	switch length {
	case 126:
		extBuf := make([]byte, 2)
		if _, err := readFull(conn, extBuf); err != nil {
			return 0, nil, err
		}
		length = uint64(extBuf[0])<<8 | uint64(extBuf[1])
	case 127:
		extBuf := make([]byte, 8)
		if _, err := readFull(conn, extBuf); err != nil {
			return 0, nil, err
		}
		length = 0
		for i := 0; i < 8; i++ {
			length = length<<8 | uint64(extBuf[i])
		}
	}

	var mask []byte
	if masked {
		mask = make([]byte, 4)
		if _, err := readFull(conn, mask); err != nil {
			return 0, nil, err
		}
	}

	payload := make([]byte, length)
	if length > 0 {
		if _, err := readFull(conn, payload); err != nil {
			return 0, nil, err
		}
	}

	if masked {
		for i := range payload {
			payload[i] ^= mask[i%4]
		}
	}

	return opcode, payload, nil
}

func readFull(conn net.Conn, buf []byte) (int, error) {
	total := 0
	for total < len(buf) {
		n, err := conn.Read(buf[total:])
		total += n
		if err != nil {
			return total, err
		}
	}
	return total, nil
}

func TestDial_Success(t *testing.T) {
	wsURL, cleanup := mockCDPServer(t, func(conn net.Conn, msg Message) {})
	defer cleanup()

	ctx := context.Background()
	client, err := Dial(ctx, wsURL)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer client.Close()
}

func TestDial_InvalidURL(t *testing.T) {
	ctx := context.Background()
	_, err := Dial(ctx, "ws://127.0.0.1:1/invalid")
	if err == nil {
		t.Error("expected error for invalid connection")
	}
}

func TestClient_Send_Success(t *testing.T) {
	wsURL, cleanup := mockCDPServer(t, func(conn net.Conn, msg Message) {
		resp := Message{
			ID:     msg.ID,
			Result: json.RawMessage(`{"frameId":"ABC123"}`),
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

	result, err := client.Send("Page.navigate", map[string]any{"url": "http://example.com"})
	if err != nil {
		t.Fatalf("Send: %v", err)
	}

	if result["frameId"] != "ABC123" {
		t.Errorf("frameId = %v, want ABC123", result["frameId"])
	}
}

func TestClient_Send_Error(t *testing.T) {
	wsURL, cleanup := mockCDPServer(t, func(conn net.Conn, msg Message) {
		resp := Message{
			ID:    msg.ID,
			Error: &ErrorInfo{Code: -32601, Message: "Method not found"},
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

	_, err = client.Send("Invalid.method", nil)
	if err == nil {
		t.Error("expected error for invalid method")
	}
	if !strings.Contains(err.Error(), "Method not found") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestClient_Send_NilParams(t *testing.T) {
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

	result, err := client.Send("DOM.enable", nil)
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	if result == nil {
		t.Error("result should not be nil")
	}
}

func TestClient_Send_NilResult(t *testing.T) {
	wsURL, cleanup := mockCDPServer(t, func(conn net.Conn, msg Message) {
		resp := Message{ID: msg.ID}
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

	result, err := client.Send("DOM.enable", nil)
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil result, got %v", result)
	}
}

func TestClient_Subscribe(t *testing.T) {
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
	client.Subscribe("Page.loadEventFired", func(params map[string]any) {
		received <- params
	})

	// Send an event from the server.
	event := Message{
		Method: "Page.loadEventFired",
		Params: json.RawMessage(`{"timestamp":1234.5}`),
	}
	data, _ := json.Marshal(event)
	if err := serverWriteTextFrame(serverConn, data); err != nil {
		t.Fatalf("serverWriteTextFrame: %v", err)
	}

	select {
	case params := <-received:
		if params["timestamp"] != 1234.5 {
			t.Errorf("timestamp = %v, want 1234.5", params["timestamp"])
		}
	case <-time.After(2 * time.Second):
		t.Error("timeout waiting for event")
	}
}

func TestClient_Subscribe_Multiple(t *testing.T) {
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

	count := 0
	var mu sync.Mutex
	done := make(chan struct{})

	client.Subscribe("Test.event", func(params map[string]any) {
		mu.Lock()
		count++
		if count == 2 {
			close(done)
		}
		mu.Unlock()
	})
	client.Subscribe("Test.event", func(params map[string]any) {
		mu.Lock()
		count++
		if count == 2 {
			close(done)
		}
		mu.Unlock()
	})

	event := Message{
		Method: "Test.event",
		Params: json.RawMessage(`{}`),
	}
	data, _ := json.Marshal(event)
	serverWriteTextFrame(serverConn, data)

	select {
	case <-done:
		// Both handlers were called.
	case <-time.After(2 * time.Second):
		t.Error("timeout waiting for both handlers")
	}
}

func TestClient_ConcurrentSend(t *testing.T) {
	wsURL, cleanup := mockCDPServer(t, func(conn net.Conn, msg Message) {
		resp := Message{
			ID:     msg.ID,
			Result: json.RawMessage(fmt.Sprintf(`{"id":%d}`, msg.ID)),
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

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			result, err := client.Send("Test.method", nil)
			if err != nil {
				t.Errorf("Send: %v", err)
				return
			}
			if result == nil {
				t.Error("result is nil")
			}
		}()
	}
	wg.Wait()
}

func TestClient_Close(t *testing.T) {
	wsURL, connCh, cleanup := mockWSServer(t)
	defer cleanup()

	ctx := context.Background()
	client, err := Dial(ctx, wsURL)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}

	serverConn := <-connCh
	defer serverConn.Close()

	if err := client.Close(); err != nil {
		t.Errorf("Close: %v", err)
	}

	// Send after close should fail.
	_, err = client.Send("Test.method", nil)
	if err == nil {
		t.Error("expected error sending after close")
	}
}

func TestClient_Done(t *testing.T) {
	wsURL, connCh, cleanup := mockWSServer(t)
	defer cleanup()

	ctx := context.Background()
	client, err := Dial(ctx, wsURL)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}

	serverConn := <-connCh
	defer serverConn.Close()

	select {
	case <-client.Done():
		t.Error("Done should not be closed yet")
	default:
		// Good.
	}

	client.Close()

	select {
	case <-client.Done():
		// Good, Done is closed.
	case <-time.After(time.Second):
		t.Error("Done should be closed after Close()")
	}
}

func TestClient_UnsubscribedEvent(t *testing.T) {
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

	// Send an event that no one subscribed to - should not crash.
	event := Message{
		Method: "Unsubscribed.event",
		Params: json.RawMessage(`{}`),
	}
	data, _ := json.Marshal(event)
	serverWriteTextFrame(serverConn, data)

	// Send a subsequent text message to ensure readLoop is still running.
	event2 := Message{
		Method: "Another.event",
	}
	data2, _ := json.Marshal(event2)

	received := make(chan struct{}, 1)
	client.Subscribe("Another.event", func(params map[string]any) {
		received <- struct{}{}
	})

	serverWriteTextFrame(serverConn, data2)

	select {
	case <-received:
		// readLoop survived the unsubscribed event.
	case <-time.After(2 * time.Second):
		t.Error("timeout - readLoop may have crashed")
	}
}

func TestClient_ServerDisconnect(t *testing.T) {
	wsURL, connCh, cleanup := mockWSServer(t)
	defer cleanup()

	ctx := context.Background()
	client, err := Dial(ctx, wsURL)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer client.Close()

	serverConn := <-connCh

	// Close server connection abruptly.
	serverConn.Close()

	// Wait for client to detect disconnect.
	select {
	case <-client.Done():
		// Good.
	case <-time.After(2 * time.Second):
		t.Error("timeout waiting for Done after server disconnect")
	}

	// Send should fail.
	_, err = client.Send("Test.method", nil)
	if err == nil {
		t.Error("expected error after server disconnect")
	}
}
