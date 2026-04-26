package browser

import (
	"bufio"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/scrutineer/scrutineer/connector/browser/cdp"
)

// mockCDPHandler is a function that handles CDP messages.
type mockCDPHandler func(msg cdp.Message) *cdp.Message

// mockCDPSetup creates a mock WebSocket-based CDP server.
// Returns the ws URL and a cleanup function.
func mockCDPSetup(handler mockCDPHandler) (string, func()) {
	connCh := make(chan net.Conn, 1)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		panic(err)
	}

	srv := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("Upgrade") != "websocket" {
				http.Error(w, "not ws", http.StatusBadRequest)
				return
			}

			key := r.Header.Get("Sec-WebSocket-Key")
			magic := "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"
			h := sha1.New()
			h.Write([]byte(key + magic))
			accept := base64.StdEncoding.EncodeToString(h.Sum(nil))

			hijacker, ok := w.(http.Hijacker)
			if !ok {
				return
			}
			conn, bufrw, err := hijacker.Hijack()
			if err != nil {
				return
			}

			bufrw.WriteString("HTTP/1.1 101 Switching Protocols\r\n" +
				"Upgrade: websocket\r\n" +
				"Connection: Upgrade\r\n" +
				"Sec-WebSocket-Accept: " + accept + "\r\n\r\n")
			bufrw.Flush()

			connCh <- conn
		}),
	}

	go srv.Serve(listener)

	// Wait for a connection and handle it.
	go func() {
		conn := <-connCh
		defer conn.Close()

		reader := bufio.NewReader(conn)
		for {
			opcode, payload, err := readWSFrame(reader)
			if err != nil {
				return
			}
			if opcode == 0x8 { // close
				return
			}
			if opcode != 0x1 { // not text
				continue
			}

			var msg cdp.Message
			if err := json.Unmarshal(payload, &msg); err != nil {
				continue
			}

			resp := handler(msg)
			if resp != nil {
				data, _ := json.Marshal(resp)
				writeWSFrame(conn, data)
			}
		}
	}()

	addr := listener.Addr().String()
	return "ws://" + addr + "/devtools", func() {
		srv.Close()
		listener.Close()
	}
}

func readWSFrame(reader *bufio.Reader) (byte, []byte, error) {
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

func writeWSFrame(conn net.Conn, payload []byte) error {
	length := len(payload)
	header := []byte{0x81} // FIN + text
	switch {
	case length <= 125:
		header = append(header, byte(length))
	case length <= 65535:
		header = append(header, 126)
		buf := make([]byte, 2)
		binary.BigEndian.PutUint16(buf, uint16(length))
		header = append(header, buf...)
	default:
		header = append(header, 127)
		buf := make([]byte, 8)
		binary.BigEndian.PutUint64(buf, uint64(length))
		header = append(header, buf...)
	}
	if _, err := conn.Write(header); err != nil {
		return err
	}
	_, err := conn.Write(payload)
	return err
}

// mockCDPBrowserServer creates a mock CDP server that simulates a browser.
// It handles Target.createTarget, Target.attachToTarget, and domain enables,
// plus any custom handler for test-specific methods.
func mockCDPBrowserServer(custom mockCDPHandler) (string, func()) {
	var mu sync.Mutex
	sessions := map[string]bool{}

	return mockCDPSetup(func(msg cdp.Message) *cdp.Message {
		mu.Lock()
		defer mu.Unlock()

		switch msg.Method {
		case "Target.createTarget":
			return &cdp.Message{
				ID:     msg.ID,
				Result: json.RawMessage(`{"targetId":"page-1"}`),
			}
		case "Target.attachToTarget":
			sessions["session-1"] = true
			return &cdp.Message{
				ID:     msg.ID,
				Result: json.RawMessage(`{"sessionId":"session-1"}`),
			}
		case "Target.detachFromTarget":
			return &cdp.Message{
				ID:     msg.ID,
				Result: json.RawMessage(`{}`),
			}
		case "Page.enable", "Runtime.enable", "DOM.enable", "Network.enable":
			return &cdp.Message{
				ID:     msg.ID,
				Result: json.RawMessage(`{}`),
			}
		default:
			if custom != nil {
				return custom(msg)
			}
			return &cdp.Message{
				ID:     msg.ID,
				Result: json.RawMessage(`{}`),
			}
		}
	})
}

// setupConnectorWithMock creates a BrowserConnector connected to a mock CDP server.
func setupConnectorWithMock(custom mockCDPHandler) (*BrowserConnector, func(), error) {
	wsURL, cleanup := mockCDPBrowserServer(custom)

	b := New()
	if err := b.connectToWSURL(nil, wsURL); err != nil {
		cleanup()
		return nil, nil, fmt.Errorf("connect: %w", err)
	}

	return b, func() {
		if b.client != nil {
			b.client.Close()
		}
		cleanup()
	}, nil
}

// mockWSServerForBrowser creates a basic WebSocket server that provides
// the raw server connection, similar to the cdp package's mockWSServer.
func mockWSServerForBrowser(t *testing.T) (string, chan net.Conn, func()) {
	t.Helper()

	connCh := make(chan net.Conn, 1)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	srv := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("Upgrade") != "websocket" {
				http.Error(w, "not ws", http.StatusBadRequest)
				return
			}

			key := r.Header.Get("Sec-WebSocket-Key")
			magic := "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"
			h := sha1.New()
			h.Write([]byte(key + magic))
			accept := base64.StdEncoding.EncodeToString(h.Sum(nil))

			hijacker, ok := w.(http.Hijacker)
			if !ok {
				return
			}
			conn, bufrw, err := hijacker.Hijack()
			if err != nil {
				return
			}

			bufrw.WriteString("HTTP/1.1 101 Switching Protocols\r\n" +
				"Upgrade: websocket\r\n" +
				"Connection: Upgrade\r\n" +
				"Sec-WebSocket-Accept: " + accept + "\r\n\r\n")
			bufrw.Flush()

			connCh <- conn
		}),
	}

	go srv.Serve(listener)

	addr := listener.Addr().String()
	return "ws://" + addr + "/devtools", connCh, func() {
		srv.Close()
		listener.Close()
	}
}

// Suppress unused warning.
var _ = strings.Contains
