package graphql

import (
	"bufio"
	"context"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// testWSServer creates a minimal WebSocket server for testing the graphql-ws protocol.
// It accepts upgrade, performs connection_init/ack handshake, receives subscribe,
// sends the provided messages, then sends complete.
func testWSServer(t *testing.T, messages []Response) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(strings.ToLower(r.Header.Get("Upgrade")), "websocket") {
			http.Error(w, "not a websocket", http.StatusBadRequest)
			return
		}

		conn, err := upgradeWebSocket(w, r)
		if err != nil {
			t.Errorf("upgrade failed: %v", err)
			return
		}
		defer conn.Close()

		reader := bufio.NewReader(conn)

		// Read connection_init.
		msg, err := serverReadFrame(reader)
		if err != nil {
			t.Errorf("reading connection_init: %v", err)
			return
		}
		var initMsg wsMessage
		json.Unmarshal(msg, &initMsg)
		if initMsg.Type != "connection_init" {
			t.Errorf("expected connection_init, got %s", initMsg.Type)
			return
		}

		// Send connection_ack.
		serverWriteFrame(conn, mustJSON(wsMessage{Type: "connection_ack"}))

		// Read subscribe.
		msg, err = serverReadFrame(reader)
		if err != nil {
			t.Errorf("reading subscribe: %v", err)
			return
		}
		var subMsg wsMessage
		json.Unmarshal(msg, &subMsg)
		if subMsg.Type != "subscribe" {
			t.Errorf("expected subscribe, got %s", subMsg.Type)
			return
		}

		// Send data messages.
		for _, resp := range messages {
			payload, _ := json.Marshal(resp)
			serverWriteFrame(conn, mustJSON(wsMessage{
				ID:      subMsg.ID,
				Type:    "next",
				Payload: payload,
			}))
		}

		// Send complete.
		serverWriteFrame(conn, mustJSON(wsMessage{
			ID:   subMsg.ID,
			Type: "complete",
		}))

		// Wait briefly so client can read.
		time.Sleep(50 * time.Millisecond)
	}))
}

// upgradeWebSocket performs the server-side WebSocket upgrade handshake.
func upgradeWebSocket(w http.ResponseWriter, r *http.Request) (net.Conn, error) {
	key := r.Header.Get("Sec-WebSocket-Key")
	if key == "" {
		return nil, fmt.Errorf("missing Sec-WebSocket-Key")
	}

	// Compute accept key per RFC 6455.
	const magic = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"
	h := sha1.New()
	h.Write([]byte(key + magic))
	acceptKey := base64.StdEncoding.EncodeToString(h.Sum(nil))

	hj, ok := w.(http.Hijacker)
	if !ok {
		return nil, fmt.Errorf("server does not support hijacking")
	}

	conn, buf, err := hj.Hijack()
	if err != nil {
		return nil, err
	}

	var resp strings.Builder
	resp.WriteString("HTTP/1.1 101 Switching Protocols\r\n")
	resp.WriteString("Upgrade: websocket\r\n")
	resp.WriteString("Connection: Upgrade\r\n")
	fmt.Fprintf(&resp, "Sec-WebSocket-Accept: %s\r\n", acceptKey)
	resp.WriteString("Sec-WebSocket-Protocol: graphql-transport-ws\r\n")
	resp.WriteString("\r\n")

	buf.WriteString(resp.String())
	buf.Flush()

	return conn, nil
}

// serverReadFrame reads a WebSocket frame from a client (masked).
func serverReadFrame(reader *bufio.Reader) ([]byte, error) {
	header := make([]byte, 2)
	if _, err := io.ReadFull(reader, header); err != nil {
		return nil, err
	}

	masked := (header[1] & 0x80) != 0
	length := uint64(header[1] & 0x7F)

	switch {
	case length == 126:
		ext := make([]byte, 2)
		if _, err := io.ReadFull(reader, ext); err != nil {
			return nil, err
		}
		length = uint64(binary.BigEndian.Uint16(ext))
	case length == 127:
		ext := make([]byte, 8)
		if _, err := io.ReadFull(reader, ext); err != nil {
			return nil, err
		}
		length = binary.BigEndian.Uint64(ext)
	}

	var mask []byte
	if masked {
		mask = make([]byte, 4)
		if _, err := io.ReadFull(reader, mask); err != nil {
			return nil, err
		}
	}

	payload := make([]byte, length)
	if _, err := io.ReadFull(reader, payload); err != nil {
		return nil, err
	}

	if masked {
		for i := range payload {
			payload[i] ^= mask[i%4]
		}
	}

	return payload, nil
}

// serverWriteFrame writes an unmasked WebSocket text frame (server to client).
func serverWriteFrame(conn net.Conn, data []byte) error {
	var frame []byte
	frame = append(frame, 0x81) // FIN + text

	length := len(data)
	switch {
	case length <= 125:
		frame = append(frame, byte(length))
	case length <= 65535:
		frame = append(frame, 126)
		frame = append(frame, byte(length>>8), byte(length))
	default:
		frame = append(frame, 127)
		b := make([]byte, 8)
		binary.BigEndian.PutUint64(b, uint64(length))
		frame = append(frame, b...)
	}

	frame = append(frame, data...)
	_, err := conn.Write(frame)
	return err
}

func mustJSON(v any) []byte {
	data, _ := json.Marshal(v)
	return data
}

func TestSubscribe_ReceiveMessages(t *testing.T) {
	t.Parallel()

	messages := []Response{
		{Data: map[string]any{"counter": float64(1)}},
		{Data: map[string]any{"counter": float64(2)}},
	}

	srv := testWSServer(t, messages)
	defer srv.Close()

	// Convert http URL to ws URL.
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	sub, err := Subscribe(ctx, wsURL, Request{
		Query: "subscription { counter }",
	}, nil)
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}
	defer sub.Close()

	// Receive first message.
	resp, err := sub.Next(ctx)
	if err != nil {
		t.Fatalf("Next(1): %v", err)
	}
	data := resp.Data.(map[string]any)
	if data["counter"] != float64(1) {
		t.Errorf("expected counter=1, got %v", data["counter"])
	}

	// Receive second message.
	resp, err = sub.Next(ctx)
	if err != nil {
		t.Fatalf("Next(2): %v", err)
	}
	data = resp.Data.(map[string]any)
	if data["counter"] != float64(2) {
		t.Errorf("expected counter=2, got %v", data["counter"])
	}

	// Next call should get complete.
	_, err = sub.Next(ctx)
	if err == nil {
		t.Fatal("expected error after server complete")
	}
}

func TestSubscribe_WithHeaders(t *testing.T) {
	t.Parallel()

	var receivedAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuth = r.Header.Get("Authorization")
		if !strings.Contains(strings.ToLower(r.Header.Get("Upgrade")), "websocket") {
			http.Error(w, "not a websocket", http.StatusBadRequest)
			return
		}

		conn, err := upgradeWebSocket(w, r)
		if err != nil {
			return
		}
		defer conn.Close()

		reader := bufio.NewReader(conn)
		serverReadFrame(reader) // connection_init
		serverWriteFrame(conn, mustJSON(wsMessage{Type: "connection_ack"}))
		serverReadFrame(reader) // subscribe

		// Send one message then complete.
		payload, _ := json.Marshal(Response{Data: "ok"})
		serverWriteFrame(conn, mustJSON(wsMessage{ID: "1", Type: "next", Payload: payload}))
		serverWriteFrame(conn, mustJSON(wsMessage{ID: "1", Type: "complete"}))
		time.Sleep(50 * time.Millisecond)
	}))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	sub, err := Subscribe(ctx, wsURL, Request{Query: "subscription { x }"}, map[string]string{
		"Authorization": "Bearer secret",
	})
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}
	defer sub.Close()

	sub.Next(ctx)

	if receivedAuth != "Bearer secret" {
		t.Errorf("expected Authorization header, got %q", receivedAuth)
	}
}

func TestSubscribe_ContextCancellation(t *testing.T) {
	t.Parallel()

	// Server that never sends data after ack.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(strings.ToLower(r.Header.Get("Upgrade")), "websocket") {
			http.Error(w, "not websocket", http.StatusBadRequest)
			return
		}
		conn, err := upgradeWebSocket(w, r)
		if err != nil {
			return
		}
		defer conn.Close()

		reader := bufio.NewReader(conn)
		serverReadFrame(reader) // connection_init
		serverWriteFrame(conn, mustJSON(wsMessage{Type: "connection_ack"}))
		serverReadFrame(reader) // subscribe

		// Block forever.
		select {}
	}))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	sub, err := Subscribe(ctx, wsURL, Request{Query: "subscription { x }"}, nil)
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}

	// Cancel context while waiting for Next.
	nextCtx, nextCancel := context.WithCancel(context.Background())

	var wg sync.WaitGroup
	wg.Add(1)
	var nextErr error
	go func() {
		defer wg.Done()
		_, nextErr = sub.Next(nextCtx)
	}()

	// Give the goroutine time to start reading.
	time.Sleep(50 * time.Millisecond)
	nextCancel()

	wg.Wait()
	if nextErr == nil {
		t.Fatal("expected error from cancelled context")
	}

	sub.Close()
}

func TestSubscribe_Close(t *testing.T) {
	t.Parallel()

	srv := testWSServer(t, nil)
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	sub, err := Subscribe(ctx, wsURL, Request{Query: "subscription { x }"}, nil)
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}

	// Close should succeed.
	if err := sub.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// Double close should be safe.
	if err := sub.Close(); err != nil {
		t.Fatalf("Double Close: %v", err)
	}
}

func TestSubscribe_InvalidEndpoint(t *testing.T) {
	t.Parallel()

	_, err := Subscribe(context.Background(), "://invalid", Request{Query: "{ x }"}, nil)
	if err == nil {
		t.Fatal("expected error for invalid endpoint")
	}
}

func TestSubscribe_UnsupportedScheme(t *testing.T) {
	t.Parallel()

	_, err := Subscribe(context.Background(), "ftp://example.com/graphql", Request{Query: "{ x }"}, nil)
	if err == nil {
		t.Fatal("expected error for unsupported scheme")
	}
}

func TestSubscribe_ConnectionRefused(t *testing.T) {
	t.Parallel()

	// Use a port that nothing listens on.
	_, err := Subscribe(context.Background(), "ws://127.0.0.1:1/graphql", Request{Query: "{ x }"}, nil)
	if err == nil {
		t.Fatal("expected error for connection refused")
	}
}

func TestSubscribe_UpgradeFailed(t *testing.T) {
	t.Parallel()

	// Server that returns 400 instead of upgrading.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad request", http.StatusBadRequest)
	}))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	_, err := Subscribe(context.Background(), wsURL, Request{Query: "{ x }"}, nil)
	if err == nil {
		t.Fatal("expected error for upgrade failure")
	}
}

func TestSubscribe_BadAck(t *testing.T) {
	t.Parallel()

	// Server that sends wrong message type after init.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(strings.ToLower(r.Header.Get("Upgrade")), "websocket") {
			http.Error(w, "not websocket", http.StatusBadRequest)
			return
		}
		conn, err := upgradeWebSocket(w, r)
		if err != nil {
			return
		}
		defer conn.Close()

		reader := bufio.NewReader(conn)
		serverReadFrame(reader) // connection_init

		// Send wrong message type.
		serverWriteFrame(conn, mustJSON(wsMessage{Type: "error"}))
		time.Sleep(50 * time.Millisecond)
	}))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := Subscribe(ctx, wsURL, Request{Query: "{ x }"}, nil)
	if err == nil {
		t.Fatal("expected error for bad ack")
	}
	if !strings.Contains(err.Error(), "expected connection_ack") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestSubscribe_ErrorMessage(t *testing.T) {
	t.Parallel()

	// Server that sends an error message.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(strings.ToLower(r.Header.Get("Upgrade")), "websocket") {
			http.Error(w, "not websocket", http.StatusBadRequest)
			return
		}
		conn, err := upgradeWebSocket(w, r)
		if err != nil {
			return
		}
		defer conn.Close()

		reader := bufio.NewReader(conn)
		serverReadFrame(reader) // connection_init
		serverWriteFrame(conn, mustJSON(wsMessage{Type: "connection_ack"}))
		serverReadFrame(reader) // subscribe

		errPayload, _ := json.Marshal([]GraphQLError{{Message: "subscription failed"}})
		serverWriteFrame(conn, mustJSON(wsMessage{ID: "1", Type: "error", Payload: errPayload}))
		time.Sleep(50 * time.Millisecond)
	}))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	sub, err := Subscribe(ctx, wsURL, Request{Query: "subscription { x }"}, nil)
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}
	defer sub.Close()

	resp, err := sub.Next(ctx)
	if err != nil {
		t.Fatalf("Next: %v", err)
	}
	if len(resp.Errors) == 0 {
		t.Fatal("expected errors in response")
	}
	if resp.Errors[0].Message != "subscription failed" {
		t.Errorf("unexpected error message: %s", resp.Errors[0].Message)
	}
}

func TestSubscribe_HTTPScheme(t *testing.T) {
	t.Parallel()

	messages := []Response{
		{Data: map[string]any{"ok": true}},
	}

	srv := testWSServer(t, messages)
	defer srv.Close()

	// Use http:// scheme instead of ws:// — should still work.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	sub, err := Subscribe(ctx, srv.URL, Request{
		Query: "subscription { x }",
	}, nil)
	if err != nil {
		t.Fatalf("Subscribe with http scheme: %v", err)
	}
	defer sub.Close()

	resp, err := sub.Next(ctx)
	if err != nil {
		t.Fatalf("Next: %v", err)
	}
	if resp.Data == nil {
		t.Fatal("expected data")
	}
}

func TestSubscribe_ServerClosesImmediately(t *testing.T) {
	t.Parallel()

	// Server that accepts TCP but closes immediately (before HTTP response).
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		// Read a bit to let client send upgrade, then close.
		buf := make([]byte, 1)
		conn.Read(buf)
		conn.Close()
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = Subscribe(ctx, "ws://"+ln.Addr().String()+"/graphql", Request{Query: "{ x }"}, nil)
	if err == nil {
		t.Fatal("expected error when server closes immediately")
	}
}

func TestSubscribe_ServerClosesAfterUpgrade(t *testing.T) {
	t.Parallel()

	// Server that upgrades WebSocket but closes right after (before reading init).
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(strings.ToLower(r.Header.Get("Upgrade")), "websocket") {
			http.Error(w, "not websocket", http.StatusBadRequest)
			return
		}
		conn, err := upgradeWebSocket(w, r)
		if err != nil {
			return
		}
		// Close immediately after upgrade — client's writeJSON for init should fail
		// or readMessage for ack should fail.
		conn.Close()
	}))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := Subscribe(ctx, wsURL, Request{Query: "subscription { x }"}, nil)
	if err == nil {
		t.Fatal("expected error when server closes after upgrade")
	}
}

func TestSubscribe_ServerClosesAfterInit(t *testing.T) {
	t.Parallel()

	// Server that reads init, sends ack, reads subscribe, then closes
	// (so the subscribe write succeeds but future reads fail).
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(strings.ToLower(r.Header.Get("Upgrade")), "websocket") {
			http.Error(w, "not websocket", http.StatusBadRequest)
			return
		}
		conn, err := upgradeWebSocket(w, r)
		if err != nil {
			return
		}
		defer conn.Close()

		reader := bufio.NewReader(conn)
		serverReadFrame(reader) // connection_init
		serverWriteFrame(conn, mustJSON(wsMessage{Type: "connection_ack"}))
		serverReadFrame(reader) // subscribe
		// Close — no data sent.
	}))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	sub, err := Subscribe(ctx, wsURL, Request{Query: "subscription { x }"}, nil)
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}
	defer sub.Close()

	// Next should fail because server closed.
	_, err = sub.Next(ctx)
	if err == nil {
		t.Fatal("expected error when server closed after subscribe")
	}
}

func TestSubscribe_ServerClosesBeforeInit(t *testing.T) {
	t.Parallel()

	// Server that upgrades but closes before sending connection_ack.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(strings.ToLower(r.Header.Get("Upgrade")), "websocket") {
			http.Error(w, "not websocket", http.StatusBadRequest)
			return
		}
		conn, err := upgradeWebSocket(w, r)
		if err != nil {
			return
		}
		// Read connection_init then close immediately.
		reader := bufio.NewReader(conn)
		serverReadFrame(reader)
		conn.Close()
	}))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := Subscribe(ctx, wsURL, Request{Query: "subscription { x }"}, nil)
	if err == nil {
		t.Fatal("expected error when server closes before ack")
	}
}

func TestReadTextFrame_Direct(t *testing.T) {
	t.Parallel()

	// Test readTextFrame directly with crafted byte sequences to cover
	// extended length (126 and 127 indicators), mask reading errors, etc.

	t.Run("126-length", func(t *testing.T) {
		t.Parallel()
		// Build a frame with 2-byte extended length.
		payload := []byte(strings.Repeat("A", 200))
		var frame []byte
		frame = append(frame, 0x81) // FIN + text
		frame = append(frame, 126)  // 2-byte extended length
		frame = append(frame, byte(len(payload)>>8), byte(len(payload)))
		frame = append(frame, payload...)

		sub := &Subscription{
			reader: bufio.NewReader(strings.NewReader(string(frame))),
			conn:   &noopConn{},
		}
		data, err := sub.readTextFrame()
		if err != nil {
			t.Fatalf("readTextFrame: %v", err)
		}
		if len(data) != 200 {
			t.Errorf("expected 200 bytes, got %d", len(data))
		}
	})

	t.Run("127-length", func(t *testing.T) {
		t.Parallel()
		// Build a frame with 8-byte extended length.
		payload := []byte(strings.Repeat("B", 300))
		var frame []byte
		frame = append(frame, 0x81) // FIN + text
		frame = append(frame, 127)  // 8-byte extended length
		b := make([]byte, 8)
		binary.BigEndian.PutUint64(b, uint64(len(payload)))
		frame = append(frame, b...)
		frame = append(frame, payload...)

		sub := &Subscription{
			reader: bufio.NewReader(strings.NewReader(string(frame))),
			conn:   &noopConn{},
		}
		data, err := sub.readTextFrame()
		if err != nil {
			t.Fatalf("readTextFrame: %v", err)
		}
		if len(data) != 300 {
			t.Errorf("expected 300 bytes, got %d", len(data))
		}
	})

	t.Run("header-read-error", func(t *testing.T) {
		t.Parallel()
		// Only 1 byte available — not enough for header.
		sub := &Subscription{
			reader: bufio.NewReader(strings.NewReader("\x81")),
			conn:   &noopConn{},
		}
		_, err := sub.readTextFrame()
		if err == nil {
			t.Fatal("expected error for truncated header")
		}
	})

	t.Run("126-length-read-error", func(t *testing.T) {
		t.Parallel()
		// Header says 126 but no extended length bytes.
		sub := &Subscription{
			reader: bufio.NewReader(strings.NewReader("\x81\x7e")),
			conn:   &noopConn{},
		}
		_, err := sub.readTextFrame()
		if err == nil {
			t.Fatal("expected error for truncated 126 length")
		}
	})

	t.Run("127-length-read-error", func(t *testing.T) {
		t.Parallel()
		// Header says 127 but only 2 of 8 extended length bytes.
		sub := &Subscription{
			reader: bufio.NewReader(strings.NewReader("\x81\x7f\x00\x00")),
			conn:   &noopConn{},
		}
		_, err := sub.readTextFrame()
		if err == nil {
			t.Fatal("expected error for truncated 127 length")
		}
	})

	t.Run("mask-read-error", func(t *testing.T) {
		t.Parallel()
		// Masked frame with length 1 but no mask bytes.
		sub := &Subscription{
			reader: bufio.NewReader(strings.NewReader("\x81\x81")),
			conn:   &noopConn{},
		}
		_, err := sub.readTextFrame()
		if err == nil {
			t.Fatal("expected error for missing mask")
		}
	})

	t.Run("payload-read-error", func(t *testing.T) {
		t.Parallel()
		// Length says 10 bytes but no payload.
		sub := &Subscription{
			reader: bufio.NewReader(strings.NewReader("\x81\x0a")),
			conn:   &noopConn{},
		}
		_, err := sub.readTextFrame()
		if err == nil {
			t.Fatal("expected error for truncated payload")
		}
	})
}

// noopConn implements net.Conn with no-op methods for testing readTextFrame directly.
type noopConn struct{}

func (c *noopConn) Read([]byte) (int, error)         { return 0, io.EOF }
func (c *noopConn) Write(b []byte) (int, error)      { return len(b), nil }
func (c *noopConn) Close() error                     { return nil }
func (c *noopConn) LocalAddr() net.Addr              { return nil }
func (c *noopConn) RemoteAddr() net.Addr             { return nil }
func (c *noopConn) SetDeadline(time.Time) error      { return nil }
func (c *noopConn) SetReadDeadline(time.Time) error  { return nil }
func (c *noopConn) SetWriteDeadline(time.Time) error { return nil }

func TestReadMessage_InvalidJSON(t *testing.T) {
	t.Parallel()

	// Build a text frame with invalid JSON.
	payload := []byte("not json{{{")
	var frame []byte
	frame = append(frame, 0x81) // FIN + text
	frame = append(frame, byte(len(payload)))
	frame = append(frame, payload...)

	sub := &Subscription{
		reader: bufio.NewReader(strings.NewReader(string(frame))),
		conn:   &noopConn{},
	}

	ctx := context.Background()
	_, err := sub.readMessage(ctx)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
	if !strings.Contains(err.Error(), "decoding websocket message") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestSubscribe_WriteUpgradeError(t *testing.T) {
	t.Parallel()

	// Use a raw TCP server that accepts connection then immediately closes
	// so conn.Write for the upgrade request fails.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		// Set a tiny SO_LINGER to force RST on close.
		if tc, ok := conn.(*net.TCPConn); ok {
			tc.SetLinger(0)
		}
		conn.Close()
	}()

	// Give the server goroutine time to accept and close.
	time.Sleep(10 * time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err = Subscribe(ctx, "ws://"+ln.Addr().String()+"/graphql", Request{Query: "{ x }"}, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	// Error may be on write or read depending on timing, but it should error.
}

func TestSubscribe_WriteInitError(t *testing.T) {
	t.Parallel()

	// Server that upgrades WebSocket then closes — init write should eventually fail.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(strings.ToLower(r.Header.Get("Upgrade")), "websocket") {
			http.Error(w, "not websocket", http.StatusBadRequest)
			return
		}
		conn, err := upgradeWebSocket(w, r)
		if err != nil {
			return
		}
		// Set linger to 0 to force RST on close.
		if tc, ok := conn.(*net.TCPConn); ok {
			tc.SetLinger(0)
		}
		// Close immediately after upgrade.
		conn.Close()
	}))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := Subscribe(ctx, wsURL, Request{Query: "subscription { x }"}, nil)
	if err == nil {
		t.Fatal("expected error when server closes after upgrade")
	}
}

func TestSubscribe_WriteSubscribeError(t *testing.T) {
	t.Parallel()

	// Server that does init/ack handshake then closes before subscribe can complete.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(strings.ToLower(r.Header.Get("Upgrade")), "websocket") {
			http.Error(w, "not websocket", http.StatusBadRequest)
			return
		}
		conn, err := upgradeWebSocket(w, r)
		if err != nil {
			return
		}

		reader := bufio.NewReader(conn)
		serverReadFrame(reader) // connection_init
		serverWriteFrame(conn, mustJSON(wsMessage{Type: "connection_ack"}))

		// Set linger to 0 to force RST.
		if tc, ok := conn.(*net.TCPConn); ok {
			tc.SetLinger(0)
		}
		// Close before client can send subscribe.
		conn.Close()
	}))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := Subscribe(ctx, wsURL, Request{Query: "subscription { x }"}, nil)
	// The error might be on the write of subscribe or on a subsequent read,
	// but there should definitely be an error.
	if err == nil {
		t.Fatal("expected error when server closes after ack")
	}
}

func TestBuildTextFrame(t *testing.T) {
	t.Parallel()

	// Test small payload.
	frame := buildTextFrame([]byte("hello"))
	if frame[0] != 0x81 {
		t.Errorf("expected FIN+text opcode, got 0x%02x", frame[0])
	}
	// Mask bit should be set, length 5.
	if frame[1] != (5 | 0x80) {
		t.Errorf("expected masked length 5, got 0x%02x", frame[1])
	}

	// Test medium payload (126-65535 bytes).
	medium := make([]byte, 200)
	frame = buildTextFrame(medium)
	if frame[1] != (126 | 0x80) {
		t.Errorf("expected extended length indicator 126, got %d", frame[1]&0x7F)
	}

	// Test large payload (>65535 bytes).
	large := make([]byte, 70000)
	frame = buildTextFrame(large)
	if frame[1] != (127 | 0x80) {
		t.Errorf("expected extended length indicator 127, got %d", frame[1]&0x7F)
	}
}

func TestBuildCloseFrame(t *testing.T) {
	t.Parallel()

	frame := buildCloseFrame(1000, "goodbye")
	if frame[0] != 0x88 {
		t.Errorf("expected close opcode, got 0x%02x", frame[0])
	}
}

func TestBuildPongFrame(t *testing.T) {
	t.Parallel()

	frame := buildPongFrame([]byte("ping"))
	if frame[0] != 0x8A {
		t.Errorf("expected pong opcode, got 0x%02x", frame[0])
	}
}

func TestSubscribe_HTTPSScheme(t *testing.T) {
	t.Parallel()

	// We can't easily test actual TLS, but we can test that the scheme
	// is accepted and it tries to dial (which will fail since there's no TLS server).
	_, err := Subscribe(context.Background(), "https://127.0.0.1:1/graphql", Request{Query: "{ x }"}, nil)
	if err == nil {
		t.Fatal("expected error (connection refused to non-existent TLS server)")
	}
}

func TestSubscribe_WSSScheme(t *testing.T) {
	t.Parallel()

	_, err := Subscribe(context.Background(), "wss://127.0.0.1:1/graphql", Request{Query: "{ x }"}, nil)
	if err == nil {
		t.Fatal("expected error for wss to non-existent server")
	}
}

func TestSubscribe_HostWithoutPort(t *testing.T) {
	t.Parallel()

	// http scheme without port should default to :80.
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	_, err := Subscribe(ctx, "http://192.0.2.1/graphql", Request{Query: "{ x }"}, nil)
	// Should fail to connect (TEST-NET address), but the scheme/port logic is exercised.
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestSubscribe_HostWithoutPortTLS(t *testing.T) {
	t.Parallel()

	// https scheme without port should default to :443.
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	_, err := Subscribe(ctx, "https://192.0.2.1/graphql", Request{Query: "{ x }"}, nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestSubscribe_ServerCloseFrame(t *testing.T) {
	t.Parallel()

	// Server that sends a close frame after ack and subscribe.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(strings.ToLower(r.Header.Get("Upgrade")), "websocket") {
			http.Error(w, "not websocket", http.StatusBadRequest)
			return
		}
		conn, err := upgradeWebSocket(w, r)
		if err != nil {
			return
		}
		defer conn.Close()

		reader := bufio.NewReader(conn)
		serverReadFrame(reader) // connection_init
		serverWriteFrame(conn, mustJSON(wsMessage{Type: "connection_ack"}))
		serverReadFrame(reader) // subscribe

		// Send a close frame (opcode 0x8).
		closePayload := make([]byte, 2)
		binary.BigEndian.PutUint16(closePayload, 1000)
		var frame []byte
		frame = append(frame, 0x88) // FIN + close
		frame = append(frame, byte(len(closePayload)))
		frame = append(frame, closePayload...)
		conn.Write(frame)
	}))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	sub, err := Subscribe(ctx, wsURL, Request{Query: "subscription { x }"}, nil)
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}
	defer sub.Close()

	_, err = sub.Next(ctx)
	if err == nil {
		t.Fatal("expected error from close frame")
	}
	if !strings.Contains(err.Error(), "websocket closed by server") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestSubscribe_ServerPongFrame(t *testing.T) {
	t.Parallel()

	// Server that sends a pong frame (unsolicited), then data.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(strings.ToLower(r.Header.Get("Upgrade")), "websocket") {
			http.Error(w, "not websocket", http.StatusBadRequest)
			return
		}
		conn, err := upgradeWebSocket(w, r)
		if err != nil {
			return
		}
		defer conn.Close()

		reader := bufio.NewReader(conn)
		serverReadFrame(reader) // connection_init
		serverWriteFrame(conn, mustJSON(wsMessage{Type: "connection_ack"}))
		serverReadFrame(reader) // subscribe

		// Send an unsolicited pong frame (opcode 0xA).
		var pongFrame []byte
		pongFrame = append(pongFrame, 0x8A) // FIN + pong
		pongFrame = append(pongFrame, 0)    // no payload
		conn.Write(pongFrame)

		// Then send actual data.
		payload, _ := json.Marshal(Response{Data: "pong-test"})
		serverWriteFrame(conn, mustJSON(wsMessage{ID: "1", Type: "next", Payload: payload}))
		serverWriteFrame(conn, mustJSON(wsMessage{ID: "1", Type: "complete"}))
		time.Sleep(50 * time.Millisecond)
	}))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	sub, err := Subscribe(ctx, wsURL, Request{Query: "subscription { x }"}, nil)
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}
	defer sub.Close()

	resp, err := sub.Next(ctx)
	if err != nil {
		t.Fatalf("Next: %v", err)
	}
	if resp.Data != "pong-test" {
		t.Errorf("expected pong-test, got %v", resp.Data)
	}
}

func TestSubscribe_UnsupportedOpcode(t *testing.T) {
	t.Parallel()

	// Server that sends a binary frame (opcode 0x2).
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(strings.ToLower(r.Header.Get("Upgrade")), "websocket") {
			http.Error(w, "not websocket", http.StatusBadRequest)
			return
		}
		conn, err := upgradeWebSocket(w, r)
		if err != nil {
			return
		}
		defer conn.Close()

		reader := bufio.NewReader(conn)
		serverReadFrame(reader) // connection_init
		serverWriteFrame(conn, mustJSON(wsMessage{Type: "connection_ack"}))
		serverReadFrame(reader) // subscribe

		// Send a binary frame (opcode 0x2).
		var frame []byte
		frame = append(frame, 0x82) // FIN + binary
		frame = append(frame, 4)    // length 4
		frame = append(frame, []byte("test")...)
		conn.Write(frame)
		time.Sleep(50 * time.Millisecond)
	}))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	sub, err := Subscribe(ctx, wsURL, Request{Query: "subscription { x }"}, nil)
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}
	defer sub.Close()

	_, err = sub.Next(ctx)
	if err == nil {
		t.Fatal("expected error for unsupported opcode")
	}
	if !strings.Contains(err.Error(), "unsupported websocket opcode") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestSubscribe_MaskedServerFrame(t *testing.T) {
	t.Parallel()

	// Server that sends a masked frame (unusual but should be handled).
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(strings.ToLower(r.Header.Get("Upgrade")), "websocket") {
			http.Error(w, "not websocket", http.StatusBadRequest)
			return
		}
		conn, err := upgradeWebSocket(w, r)
		if err != nil {
			return
		}
		defer conn.Close()

		reader := bufio.NewReader(conn)
		serverReadFrame(reader) // connection_init
		serverWriteFrame(conn, mustJSON(wsMessage{Type: "connection_ack"}))
		serverReadFrame(reader) // subscribe

		// Send a masked text frame.
		data := mustJSON(wsMessage{ID: "1", Type: "next", Payload: mustJSON(Response{Data: "masked"})})
		mask := []byte{0x12, 0x34, 0x56, 0x78}
		masked := make([]byte, len(data))
		for i := range data {
			masked[i] = data[i] ^ mask[i%4]
		}
		var frame []byte
		frame = append(frame, 0x81)                 // FIN + text
		frame = append(frame, byte(len(data))|0x80) // masked + length
		frame = append(frame, mask...)              // mask key
		frame = append(frame, masked...)            // masked payload
		conn.Write(frame)

		serverWriteFrame(conn, mustJSON(wsMessage{ID: "1", Type: "complete"}))
		time.Sleep(50 * time.Millisecond)
	}))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	sub, err := Subscribe(ctx, wsURL, Request{Query: "subscription { x }"}, nil)
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}
	defer sub.Close()

	resp, err := sub.Next(ctx)
	if err != nil {
		t.Fatalf("Next: %v", err)
	}
	if resp.Data != "masked" {
		t.Errorf("expected masked, got %v", resp.Data)
	}
}

func TestSubscribe_MediumLengthFrame(t *testing.T) {
	t.Parallel()

	// Server that sends a frame with payload length requiring 2-byte extended length (126-65535).
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(strings.ToLower(r.Header.Get("Upgrade")), "websocket") {
			http.Error(w, "not websocket", http.StatusBadRequest)
			return
		}
		conn, err := upgradeWebSocket(w, r)
		if err != nil {
			return
		}
		defer conn.Close()

		reader := bufio.NewReader(conn)
		serverReadFrame(reader) // connection_init
		serverWriteFrame(conn, mustJSON(wsMessage{Type: "connection_ack"}))
		serverReadFrame(reader) // subscribe

		// Build a large payload that requires 2-byte extended length.
		// We need > 125 bytes. Create a large response data string.
		bigData := strings.Repeat("x", 200)
		payload := mustJSON(Response{Data: bigData})
		data := mustJSON(wsMessage{ID: "1", Type: "next", Payload: payload})

		var frame []byte
		frame = append(frame, 0x81) // FIN + text
		frame = append(frame, 126)  // 2-byte extended length
		frame = append(frame, byte(len(data)>>8), byte(len(data)))
		frame = append(frame, data...)
		conn.Write(frame)

		serverWriteFrame(conn, mustJSON(wsMessage{ID: "1", Type: "complete"}))
		time.Sleep(50 * time.Millisecond)
	}))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	sub, err := Subscribe(ctx, wsURL, Request{Query: "subscription { x }"}, nil)
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}
	defer sub.Close()

	resp, err := sub.Next(ctx)
	if err != nil {
		t.Fatalf("Next: %v", err)
	}
	if resp.Data == nil {
		t.Fatal("expected data")
	}
}

func TestSubscribe_InvalidNextPayload(t *testing.T) {
	t.Parallel()

	// Server that sends a "next" message with invalid JSON payload.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(strings.ToLower(r.Header.Get("Upgrade")), "websocket") {
			http.Error(w, "not websocket", http.StatusBadRequest)
			return
		}
		conn, err := upgradeWebSocket(w, r)
		if err != nil {
			return
		}
		defer conn.Close()

		reader := bufio.NewReader(conn)
		serverReadFrame(reader) // connection_init
		serverWriteFrame(conn, mustJSON(wsMessage{Type: "connection_ack"}))
		serverReadFrame(reader) // subscribe

		// Send a "next" with broken payload.
		serverWriteFrame(conn, []byte(`{"id":"1","type":"next","payload":"not valid response"}`))
		time.Sleep(50 * time.Millisecond)
	}))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	sub, err := Subscribe(ctx, wsURL, Request{Query: "subscription { x }"}, nil)
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}
	defer sub.Close()

	_, err = sub.Next(ctx)
	if err == nil {
		t.Fatal("expected error for invalid next payload")
	}
}

func TestSubscribe_InvalidErrorPayload(t *testing.T) {
	t.Parallel()

	// Server that sends an "error" message with invalid payload.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(strings.ToLower(r.Header.Get("Upgrade")), "websocket") {
			http.Error(w, "not websocket", http.StatusBadRequest)
			return
		}
		conn, err := upgradeWebSocket(w, r)
		if err != nil {
			return
		}
		defer conn.Close()

		reader := bufio.NewReader(conn)
		serverReadFrame(reader) // connection_init
		serverWriteFrame(conn, mustJSON(wsMessage{Type: "connection_ack"}))
		serverReadFrame(reader) // subscribe

		// Send an "error" with broken payload.
		serverWriteFrame(conn, []byte(`{"id":"1","type":"error","payload":"not an array"}`))
		time.Sleep(50 * time.Millisecond)
	}))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	sub, err := Subscribe(ctx, wsURL, Request{Query: "subscription { x }"}, nil)
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}
	defer sub.Close()

	_, err = sub.Next(ctx)
	if err == nil {
		t.Fatal("expected error for invalid error payload")
	}
}

func TestSubscribe_UnknownMessageType(t *testing.T) {
	t.Parallel()

	// Server that sends an unknown message type, then real data.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(strings.ToLower(r.Header.Get("Upgrade")), "websocket") {
			http.Error(w, "not websocket", http.StatusBadRequest)
			return
		}
		conn, err := upgradeWebSocket(w, r)
		if err != nil {
			return
		}
		defer conn.Close()

		reader := bufio.NewReader(conn)
		serverReadFrame(reader) // connection_init
		serverWriteFrame(conn, mustJSON(wsMessage{Type: "connection_ack"}))
		serverReadFrame(reader) // subscribe

		// Send unknown message type (should be skipped).
		serverWriteFrame(conn, mustJSON(wsMessage{Type: "ka"}))

		// Then send real data.
		payload, _ := json.Marshal(Response{Data: "after-unknown"})
		serverWriteFrame(conn, mustJSON(wsMessage{ID: "1", Type: "next", Payload: payload}))
		serverWriteFrame(conn, mustJSON(wsMessage{ID: "1", Type: "complete"}))
		time.Sleep(50 * time.Millisecond)
	}))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	sub, err := Subscribe(ctx, wsURL, Request{Query: "subscription { x }"}, nil)
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}
	defer sub.Close()

	resp, err := sub.Next(ctx)
	if err != nil {
		t.Fatalf("Next: %v", err)
	}
	if resp.Data != "after-unknown" {
		t.Errorf("expected after-unknown, got %v", resp.Data)
	}
}

func TestSubscribe_ServerPing(t *testing.T) {
	t.Parallel()

	// Server that sends a ping before data.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(strings.ToLower(r.Header.Get("Upgrade")), "websocket") {
			http.Error(w, "not websocket", http.StatusBadRequest)
			return
		}
		conn, err := upgradeWebSocket(w, r)
		if err != nil {
			return
		}
		defer conn.Close()

		reader := bufio.NewReader(conn)
		serverReadFrame(reader) // connection_init
		serverWriteFrame(conn, mustJSON(wsMessage{Type: "connection_ack"}))
		serverReadFrame(reader) // subscribe

		// Send a ping frame (opcode 0x9), unmasked.
		pingPayload := []byte("keepalive")
		var pingFrame []byte
		pingFrame = append(pingFrame, 0x89) // FIN + ping
		pingFrame = append(pingFrame, byte(len(pingPayload)))
		pingFrame = append(pingFrame, pingPayload...)
		conn.Write(pingFrame)

		// Then send actual data.
		time.Sleep(20 * time.Millisecond)
		payload, _ := json.Marshal(Response{Data: map[string]any{"val": float64(42)}})
		serverWriteFrame(conn, mustJSON(wsMessage{ID: "1", Type: "next", Payload: payload}))
		serverWriteFrame(conn, mustJSON(wsMessage{ID: "1", Type: "complete"}))
		time.Sleep(50 * time.Millisecond)
	}))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	sub, err := Subscribe(ctx, wsURL, Request{Query: "subscription { x }"}, nil)
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}
	defer sub.Close()

	// Should skip ping and receive the actual data.
	resp, err := sub.Next(ctx)
	if err != nil {
		t.Fatalf("Next: %v", err)
	}
	data := resp.Data.(map[string]any)
	if data["val"] != float64(42) {
		t.Errorf("expected val=42, got %v", data["val"])
	}
}
