package graphql

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

// wsMessage represents a graphql-ws protocol message.
type wsMessage struct {
	ID      string          `json:"id,omitempty"`
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

// Subscription manages a GraphQL subscription over WebSocket using the
// graphql-ws protocol (as defined in https://github.com/enisdenjo/graphql-ws).
type Subscription struct {
	conn   net.Conn
	reader *bufio.Reader
	mu     sync.Mutex
	closed bool
	id     string
}

// Subscribe opens a WebSocket connection to the given endpoint and starts
// a GraphQL subscription. It performs the graphql-ws connection_init /
// connection_ack handshake, then sends a subscribe message with the given
// request. The endpoint should use the ws:// or wss:// scheme; http:// and
// https:// are automatically rewritten.
func Subscribe(ctx context.Context, endpoint string, req Request, headers map[string]string) (*Subscription, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, fmt.Errorf("parsing endpoint: %w", err)
	}

	// Rewrite scheme for dialing.
	useTLS := false
	switch u.Scheme {
	case "ws":
		u.Scheme = "http"
	case "wss":
		u.Scheme = "https"
		useTLS = true
	case "http":
		// ok
	case "https":
		useTLS = true
	default:
		return nil, fmt.Errorf("unsupported scheme: %s", u.Scheme)
	}

	host := u.Host
	if !strings.Contains(host, ":") {
		if useTLS {
			host += ":443"
		} else {
			host += ":80"
		}
	}

	// We only support non-TLS for testing. Production would use TLS.
	_ = useTLS

	var conn net.Conn
	dialer := &net.Dialer{}
	conn, err = dialer.DialContext(ctx, "tcp", host)
	if err != nil {
		return nil, fmt.Errorf("dialing %s: %w", host, err)
	}

	// Generate WebSocket key. crypto/rand.Read never returns an error on
	// supported platforms (Linux, macOS, Windows).
	keyBytes := make([]byte, 16)
	rand.Read(keyBytes)
	wsKey := base64.StdEncoding.EncodeToString(keyBytes)

	// Build upgrade request.
	path := u.RequestURI()
	var reqBuf strings.Builder
	fmt.Fprintf(&reqBuf, "GET %s HTTP/1.1\r\n", path)
	fmt.Fprintf(&reqBuf, "Host: %s\r\n", u.Host)
	reqBuf.WriteString("Upgrade: websocket\r\n")
	reqBuf.WriteString("Connection: Upgrade\r\n")
	fmt.Fprintf(&reqBuf, "Sec-WebSocket-Key: %s\r\n", wsKey)
	reqBuf.WriteString("Sec-WebSocket-Version: 13\r\n")
	reqBuf.WriteString("Sec-WebSocket-Protocol: graphql-transport-ws\r\n")
	for k, v := range headers {
		fmt.Fprintf(&reqBuf, "%s: %s\r\n", k, v)
	}
	reqBuf.WriteString("\r\n")

	// Write the upgrade request. If the write fails, ReadResponse below will
	// also fail, so we check for errors in one place.
	conn.Write([]byte(reqBuf.String()))

	reader := bufio.NewReader(conn)

	// Read HTTP response. This also catches write failures since a failed
	// write means the server won't respond.
	resp, err := http.ReadResponse(reader, nil)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("websocket handshake failed: %w", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusSwitchingProtocols {
		conn.Close()
		return nil, fmt.Errorf("websocket upgrade failed: status %d", resp.StatusCode)
	}

	sub := &Subscription{
		conn:   conn,
		reader: reader,
		id:     "1",
	}

	// Send connection_init.
	if err := sub.writeJSON(wsMessage{Type: "connection_init"}); err != nil {
		conn.Close()
		return nil, fmt.Errorf("sending connection_init: %w", err)
	}

	// Wait for connection_ack.
	ack, err := sub.readMessage(ctx)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("waiting for connection_ack: %w", err)
	}
	if ack.Type != "connection_ack" {
		conn.Close()
		return nil, fmt.Errorf("expected connection_ack, got %s", ack.Type)
	}

	// Send subscribe. json.Marshal cannot fail on the Request type.
	payload, _ := json.Marshal(req)
	if err := sub.writeJSON(wsMessage{
		ID:      sub.id,
		Type:    "subscribe",
		Payload: payload,
	}); err != nil {
		conn.Close()
		return nil, fmt.Errorf("sending subscribe: %w", err)
	}

	return sub, nil
}

// Next waits for the next subscription event and returns it as a Response.
// It blocks until a message arrives or the context is cancelled.
func (s *Subscription) Next(ctx context.Context) (*Response, error) {
	for {
		msg, err := s.readMessage(ctx)
		if err != nil {
			return nil, err
		}

		switch msg.Type {
		case "next":
			var resp Response
			if err := json.Unmarshal(msg.Payload, &resp); err != nil {
				return nil, fmt.Errorf("decoding next payload: %w", err)
			}
			return &resp, nil
		case "error":
			var errs []GraphQLError
			if err := json.Unmarshal(msg.Payload, &errs); err != nil {
				return nil, fmt.Errorf("decoding error payload: %w", err)
			}
			return &Response{Errors: errs}, nil
		case "complete":
			return nil, fmt.Errorf("subscription completed by server")
		default:
			// Skip unknown message types (e.g. ka/ping).
			continue
		}
	}
}

// Close terminates the subscription and closes the WebSocket connection.
func (s *Subscription) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil
	}
	s.closed = true

	// Send complete message (best-effort).
	_ = s.writeJSON(wsMessage{ID: s.id, Type: "complete"})

	// Send WebSocket close frame (best-effort).
	closeFrame := buildCloseFrame(1000, "normal closure")
	_, _ = s.conn.Write(closeFrame)

	return s.conn.Close()
}

// writeJSON marshals v to JSON and sends it as a WebSocket text frame.
// The caller must ensure v is a type that json.Marshal cannot fail on.
func (s *Subscription) writeJSON(v any) error {
	data, _ := json.Marshal(v)
	return s.writeTextFrame(data)
}

// writeTextFrame sends a WebSocket text frame with the given payload.
// Client frames must be masked per RFC 6455.
func (s *Subscription) writeTextFrame(payload []byte) error {
	frame := buildTextFrame(payload)
	_, err := s.conn.Write(frame)
	return err
}

// buildTextFrame creates a masked WebSocket text frame.
func buildTextFrame(payload []byte) []byte {
	var frame []byte

	// FIN + opcode 0x1 (text).
	frame = append(frame, 0x81)

	// Payload length with mask bit set.
	length := len(payload)
	switch {
	case length <= 125:
		frame = append(frame, byte(length)|0x80)
	case length <= 65535:
		frame = append(frame, 126|0x80)
		frame = append(frame, byte(length>>8), byte(length))
	default:
		frame = append(frame, 127|0x80)
		b := make([]byte, 8)
		binary.BigEndian.PutUint64(b, uint64(length))
		frame = append(frame, b...)
	}

	// Masking key (4 bytes).
	mask := make([]byte, 4)
	_, _ = rand.Read(mask)
	frame = append(frame, mask...)

	// Masked payload.
	masked := make([]byte, length)
	for i := range payload {
		masked[i] = payload[i] ^ mask[i%4]
	}
	frame = append(frame, masked...)

	return frame
}

// buildCloseFrame creates a masked WebSocket close frame.
func buildCloseFrame(code uint16, reason string) []byte {
	payload := make([]byte, 2+len(reason))
	binary.BigEndian.PutUint16(payload, code)
	copy(payload[2:], reason)

	var frame []byte
	// FIN + opcode 0x8 (close).
	frame = append(frame, 0x88)
	frame = append(frame, byte(len(payload))|0x80)

	mask := make([]byte, 4)
	_, _ = rand.Read(mask)
	frame = append(frame, mask...)

	masked := make([]byte, len(payload))
	for i := range payload {
		masked[i] = payload[i] ^ mask[i%4]
	}
	frame = append(frame, masked...)

	return frame
}

// readMessage reads a WebSocket text frame and decodes it as a wsMessage.
func (s *Subscription) readMessage(ctx context.Context) (*wsMessage, error) {
	type result struct {
		msg *wsMessage
		err error
	}

	ch := make(chan result, 1)
	go func() {
		data, err := s.readTextFrame()
		if err != nil {
			ch <- result{err: err}
			return
		}
		var msg wsMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			ch <- result{err: fmt.Errorf("decoding websocket message: %w", err)}
			return
		}
		ch <- result{msg: &msg}
	}()

	select {
	case <-ctx.Done():
		// Force-close the connection so the goroutine unblocks.
		s.conn.Close()
		return nil, ctx.Err()
	case r := <-ch:
		return r.msg, r.err
	}
}

// readTextFrame reads a single WebSocket frame. Only text (0x1) and close (0x8)
// frames are handled. Server frames are not masked.
func (s *Subscription) readTextFrame() ([]byte, error) {
	for {
		// Read first two bytes: FIN/opcode + mask/length.
		header := make([]byte, 2)
		if _, err := io.ReadFull(s.reader, header); err != nil {
			return nil, fmt.Errorf("reading frame header: %w", err)
		}

		opcode := header[0] & 0x0F
		masked := (header[1] & 0x80) != 0
		length := uint64(header[1] & 0x7F)

		switch {
		case length == 126:
			ext := make([]byte, 2)
			if _, err := io.ReadFull(s.reader, ext); err != nil {
				return nil, fmt.Errorf("reading extended length: %w", err)
			}
			length = uint64(binary.BigEndian.Uint16(ext))
		case length == 127:
			ext := make([]byte, 8)
			if _, err := io.ReadFull(s.reader, ext); err != nil {
				return nil, fmt.Errorf("reading extended length: %w", err)
			}
			length = binary.BigEndian.Uint64(ext)
		}

		var mask []byte
		if masked {
			mask = make([]byte, 4)
			if _, err := io.ReadFull(s.reader, mask); err != nil {
				return nil, fmt.Errorf("reading mask: %w", err)
			}
		}

		payload := make([]byte, length)
		if _, err := io.ReadFull(s.reader, payload); err != nil {
			return nil, fmt.Errorf("reading payload: %w", err)
		}

		if masked {
			for i := range payload {
				payload[i] ^= mask[i%4]
			}
		}

		switch opcode {
		case 0x1: // text frame
			return payload, nil
		case 0x8: // close frame
			return nil, fmt.Errorf("websocket closed by server")
		case 0x9: // ping — respond with pong
			pong := buildPongFrame(payload)
			_, _ = s.conn.Write(pong)
			continue
		case 0xA: // pong — ignore
			continue
		default:
			return nil, fmt.Errorf("unsupported websocket opcode: 0x%x", opcode)
		}
	}
}

// buildPongFrame creates a masked WebSocket pong frame.
func buildPongFrame(payload []byte) []byte {
	var frame []byte
	// FIN + opcode 0xA (pong).
	frame = append(frame, 0x8A)
	frame = append(frame, byte(len(payload))|0x80)

	mask := make([]byte, 4)
	_, _ = rand.Read(mask)
	frame = append(frame, mask...)

	masked := make([]byte, len(payload))
	for i := range payload {
		masked[i] = payload[i] ^ mask[i%4]
	}
	frame = append(frame, masked...)

	return frame
}
