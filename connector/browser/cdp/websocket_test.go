package cdp

import (
	"bufio"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"testing"
)

// mockWSServer creates a mock WebSocket server that accepts a single connection.
// It returns the ws:// URL and a channel that provides the server-side connection.
func mockWSServer(t *testing.T) (string, chan net.Conn, func()) {
	t.Helper()

	connCh := make(chan net.Conn, 1)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	srv := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("Upgrade") != "websocket" {
				http.Error(w, "not a websocket request", http.StatusBadRequest)
				return
			}

			key := r.Header.Get("Sec-WebSocket-Key")
			magic := "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"
			h := sha1.New()
			h.Write([]byte(key + magic))
			accept := base64.StdEncoding.EncodeToString(h.Sum(nil))

			hijacker, ok := w.(http.Hijacker)
			if !ok {
				http.Error(w, "hijack not supported", http.StatusInternalServerError)
				return
			}

			conn, bufrw, err := hijacker.Hijack()
			if err != nil {
				return
			}

			response := "HTTP/1.1 101 Switching Protocols\r\n" +
				"Upgrade: websocket\r\n" +
				"Connection: Upgrade\r\n" +
				"Sec-WebSocket-Accept: " + accept + "\r\n" +
				"\r\n"

			bufrw.WriteString(response)
			bufrw.Flush()

			connCh <- conn
		}),
	}

	go srv.Serve(listener)

	addr := listener.Addr().String()
	wsURL := "ws://" + addr + "/devtools/browser/test"

	return wsURL, connCh, func() {
		srv.Close()
		listener.Close()
	}
}

// serverWriteTextFrame writes an unmasked text frame from the server side.
func serverWriteTextFrame(conn net.Conn, payload []byte) error {
	// FIN + opText
	if _, err := conn.Write([]byte{0x81}); err != nil {
		return err
	}

	length := len(payload)
	switch {
	case length <= 125:
		if _, err := conn.Write([]byte{byte(length)}); err != nil {
			return err
		}
	case length <= 65535:
		if _, err := conn.Write([]byte{126}); err != nil {
			return err
		}
		buf := make([]byte, 2)
		binary.BigEndian.PutUint16(buf, uint16(length))
		if _, err := conn.Write(buf); err != nil {
			return err
		}
	default:
		if _, err := conn.Write([]byte{127}); err != nil {
			return err
		}
		buf := make([]byte, 8)
		binary.BigEndian.PutUint64(buf, uint64(length))
		if _, err := conn.Write(buf); err != nil {
			return err
		}
	}

	_, err := conn.Write(payload)
	return err
}

// serverReadFrame reads a masked frame from the client.
func serverReadFrame(conn net.Conn) (byte, []byte, error) {
	reader := bufio.NewReader(conn)

	b0, err := reader.ReadByte()
	if err != nil {
		return 0, nil, err
	}
	opcode := b0 & 0x0F

	b1, err := reader.ReadByte()
	if err != nil {
		return 0, nil, err
	}
	masked := b1&0x80 != 0
	length := uint64(b1 & 0x7F)

	switch length {
	case 126:
		buf := make([]byte, 2)
		if _, err := io.ReadFull(reader, buf); err != nil {
			return 0, nil, err
		}
		length = uint64(binary.BigEndian.Uint16(buf))
	case 127:
		buf := make([]byte, 8)
		if _, err := io.ReadFull(reader, buf); err != nil {
			return 0, nil, err
		}
		length = binary.BigEndian.Uint64(buf)
	}

	var mask []byte
	if masked {
		mask = make([]byte, 4)
		if _, err := io.ReadFull(reader, mask); err != nil {
			return 0, nil, err
		}
	}

	payload := make([]byte, length)
	if length > 0 {
		if _, err := io.ReadFull(reader, payload); err != nil {
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

// serverSendPing sends a ping frame from the server.
func serverSendPing(conn net.Conn, payload []byte) error {
	header := []byte{0x80 | opPing, byte(len(payload))}
	if _, err := conn.Write(header); err != nil {
		return err
	}
	if len(payload) > 0 {
		_, err := conn.Write(payload)
		return err
	}
	return nil
}

// serverSendClose sends a close frame from the server.
func serverSendClose(conn net.Conn) error {
	_, err := conn.Write([]byte{0x80 | opClose, 0})
	return err
}

func TestDialWebSocket_Success(t *testing.T) {
	wsURL, connCh, cleanup := mockWSServer(t)
	defer cleanup()

	ws, err := dialWebSocket(wsURL)
	if err != nil {
		t.Fatalf("dialWebSocket: %v", err)
	}
	defer ws.Close()

	// Server should have received a connection.
	serverConn := <-connCh
	defer serverConn.Close()

	if ws.conn == nil {
		t.Error("ws.conn is nil")
	}
}

func TestDialWebSocket_InvalidURL(t *testing.T) {
	_, err := dialWebSocket("://invalid")
	if err == nil {
		t.Error("expected error for invalid URL")
	}
}

func TestDialWebSocket_ConnectionRefused(t *testing.T) {
	_, err := dialWebSocket("ws://127.0.0.1:1/test")
	if err == nil {
		t.Error("expected error for connection refused")
	}
}

func TestDialWebSocket_Non101Response(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer listener.Close()

	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		conn.Write([]byte("HTTP/1.1 400 Bad Request\r\n\r\n"))
	}()

	_, err = dialWebSocket("ws://" + listener.Addr().String() + "/test")
	if err == nil {
		t.Error("expected error for non-101 status")
	}
	if !strings.Contains(err.Error(), "unexpected status") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestDialWebSocket_InvalidAcceptKey(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer listener.Close()

	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		reader := bufio.NewReader(conn)
		// Read the request.
		for {
			line, err := reader.ReadString('\n')
			if err != nil || strings.TrimSpace(line) == "" {
				break
			}
		}

		resp := "HTTP/1.1 101 Switching Protocols\r\n" +
			"Upgrade: websocket\r\n" +
			"Connection: Upgrade\r\n" +
			"Sec-WebSocket-Accept: wrong-key\r\n" +
			"\r\n"
		conn.Write([]byte(resp))
	}()

	_, err = dialWebSocket("ws://" + listener.Addr().String() + "/test")
	if err == nil {
		t.Error("expected error for invalid accept key")
	}
	if !strings.Contains(err.Error(), "invalid Sec-WebSocket-Accept") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestWSConn_WriteAndReadText(t *testing.T) {
	wsURL, connCh, cleanup := mockWSServer(t)
	defer cleanup()

	ws, err := dialWebSocket(wsURL)
	if err != nil {
		t.Fatalf("dialWebSocket: %v", err)
	}
	defer ws.Close()

	serverConn := <-connCh
	defer serverConn.Close()

	// Client writes, server reads.
	msg := []byte("hello from client")
	if err := ws.WriteText(msg); err != nil {
		t.Fatalf("WriteText: %v", err)
	}

	opcode, payload, err := serverReadFrame(serverConn)
	if err != nil {
		t.Fatalf("serverReadFrame: %v", err)
	}
	if opcode != opText {
		t.Errorf("opcode = %d, want %d", opcode, opText)
	}
	if string(payload) != "hello from client" {
		t.Errorf("payload = %q, want %q", payload, "hello from client")
	}
}

func TestWSConn_ReadServerMessage(t *testing.T) {
	wsURL, connCh, cleanup := mockWSServer(t)
	defer cleanup()

	ws, err := dialWebSocket(wsURL)
	if err != nil {
		t.Fatalf("dialWebSocket: %v", err)
	}
	defer ws.Close()

	serverConn := <-connCh
	defer serverConn.Close()

	// Server writes, client reads.
	serverMsg := []byte(`{"id":1,"result":{}}`)
	if err := serverWriteTextFrame(serverConn, serverMsg); err != nil {
		t.Fatalf("serverWriteTextFrame: %v", err)
	}

	data, err := ws.ReadMessage()
	if err != nil {
		t.Fatalf("ReadMessage: %v", err)
	}

	if string(data) != string(serverMsg) {
		t.Errorf("got %q, want %q", data, serverMsg)
	}
}

func TestWSConn_PingPong(t *testing.T) {
	wsURL, connCh, cleanup := mockWSServer(t)
	defer cleanup()

	ws, err := dialWebSocket(wsURL)
	if err != nil {
		t.Fatalf("dialWebSocket: %v", err)
	}
	defer ws.Close()

	serverConn := <-connCh
	defer serverConn.Close()

	// Server sends ping.
	if err := serverSendPing(serverConn, []byte("ping")); err != nil {
		t.Fatalf("serverSendPing: %v", err)
	}

	// Then server sends a text frame that the client should return after handling the ping.
	if err := serverWriteTextFrame(serverConn, []byte("after-ping")); err != nil {
		t.Fatalf("serverWriteTextFrame: %v", err)
	}

	// Client should automatically respond with pong and return the text message.
	data, err := ws.ReadMessage()
	if err != nil {
		t.Fatalf("ReadMessage: %v", err)
	}

	if string(data) != "after-ping" {
		t.Errorf("got %q, want %q", data, "after-ping")
	}
}

func TestWSConn_ServerClose(t *testing.T) {
	wsURL, connCh, cleanup := mockWSServer(t)
	defer cleanup()

	ws, err := dialWebSocket(wsURL)
	if err != nil {
		t.Fatalf("dialWebSocket: %v", err)
	}
	defer ws.Close()

	serverConn := <-connCh
	defer serverConn.Close()

	// Server sends close frame.
	if err := serverSendClose(serverConn); err != nil {
		t.Fatalf("serverSendClose: %v", err)
	}

	// Client should get an error.
	_, err = ws.ReadMessage()
	if err == nil {
		t.Error("expected error on server close")
	}
	if !strings.Contains(err.Error(), "closed by server") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestWSConn_WriteLargeFrame(t *testing.T) {
	wsURL, connCh, cleanup := mockWSServer(t)
	defer cleanup()

	ws, err := dialWebSocket(wsURL)
	if err != nil {
		t.Fatalf("dialWebSocket: %v", err)
	}
	defer ws.Close()

	serverConn := <-connCh
	defer serverConn.Close()

	// Write a large message (>125 bytes, <65535 bytes).
	msg := make([]byte, 500)
	for i := range msg {
		msg[i] = byte('A' + (i % 26))
	}
	if err := ws.WriteText(msg); err != nil {
		t.Fatalf("WriteText: %v", err)
	}

	_, payload, err := serverReadFrame(serverConn)
	if err != nil {
		t.Fatalf("serverReadFrame: %v", err)
	}
	if len(payload) != 500 {
		t.Errorf("payload length = %d, want 500", len(payload))
	}
	if string(payload) != string(msg) {
		t.Error("payload content mismatch")
	}
}

func TestWSConn_WriteAfterClose(t *testing.T) {
	wsURL, connCh, cleanup := mockWSServer(t)
	defer cleanup()

	ws, err := dialWebSocket(wsURL)
	if err != nil {
		t.Fatalf("dialWebSocket: %v", err)
	}

	serverConn := <-connCh
	defer serverConn.Close()

	ws.Close()

	err = ws.WriteText([]byte("after close"))
	if err == nil {
		t.Error("expected error writing after close")
	}
}

func TestWSConn_ConcurrentWrites(t *testing.T) {
	wsURL, connCh, cleanup := mockWSServer(t)
	defer cleanup()

	ws, err := dialWebSocket(wsURL)
	if err != nil {
		t.Fatalf("dialWebSocket: %v", err)
	}
	defer ws.Close()

	serverConn := <-connCh
	defer serverConn.Close()

	// Multiple concurrent writes should not panic.
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			msg := fmt.Sprintf("message-%d", n)
			_ = ws.WriteText([]byte(msg))
		}(i)
	}
	wg.Wait()
}

func TestComputeAcceptKey(t *testing.T) {
	// Test with a known key from RFC 6455 Section 4.2.2.
	key := "dGhlIHNhbXBsZSBub25jZQ=="
	expected := "s3pPLMBiTxaQ9kYGzzhZRbK+xOo="
	got := computeAcceptKey(key)
	if got != expected {
		t.Errorf("computeAcceptKey(%q) = %q, want %q", key, got, expected)
	}
}

func TestWSConn_ReadLargeServerFrame(t *testing.T) {
	wsURL, connCh, cleanup := mockWSServer(t)
	defer cleanup()

	ws, err := dialWebSocket(wsURL)
	if err != nil {
		t.Fatalf("dialWebSocket: %v", err)
	}
	defer ws.Close()

	serverConn := <-connCh
	defer serverConn.Close()

	// Send a large frame from server (>125 bytes).
	largePayload := make([]byte, 300)
	for i := range largePayload {
		largePayload[i] = byte('X')
	}
	if err := serverWriteTextFrame(serverConn, largePayload); err != nil {
		t.Fatalf("serverWriteTextFrame: %v", err)
	}

	data, err := ws.ReadMessage()
	if err != nil {
		t.Fatalf("ReadMessage: %v", err)
	}
	if len(data) != 300 {
		t.Errorf("data length = %d, want 300", len(data))
	}
}

func TestDialWebSocket_DefaultPort(t *testing.T) {
	// Test URL without explicit port - should fail to connect but parse correctly.
	_, err := dialWebSocket("ws://localhost/test")
	if err == nil {
		t.Error("expected error connecting to localhost:80")
	}
	// Should attempt to dial, not fail on URL parsing.
	if strings.Contains(err.Error(), "parse") {
		t.Errorf("should not be a parse error: %v", err)
	}
}
