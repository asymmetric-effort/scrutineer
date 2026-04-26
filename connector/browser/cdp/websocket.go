package cdp

import (
	"bufio"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"strings"
	"sync"
)

// WebSocket opcodes per RFC 6455.
const (
	opText  = 0x1
	opClose = 0x8
	opPing  = 0x9
	opPong  = 0xA
)

// wsConn is a minimal WebSocket client supporting text frames.
type wsConn struct {
	conn   net.Conn
	reader *bufio.Reader
	writer *bufio.Writer
	mu     sync.Mutex // guards writes
	closed bool
}

// dialWebSocket opens a WebSocket connection via the HTTP upgrade handshake.
func dialWebSocket(rawURL string) (*wsConn, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("cdp: parse url: %w", err)
	}

	host := u.Host
	if !strings.Contains(host, ":") {
		switch u.Scheme {
		case "ws":
			host += ":80"
		case "wss":
			host += ":443"
		default:
			host += ":80"
		}
	}

	conn, err := net.Dial("tcp", host)
	if err != nil {
		return nil, fmt.Errorf("cdp: dial %s: %w", host, err)
	}

	keyBytes := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, keyBytes); err != nil {
		conn.Close()
		return nil, fmt.Errorf("cdp: generate key: %w", err)
	}
	key := base64.StdEncoding.EncodeToString(keyBytes)

	path := u.RequestURI()
	reqHost := u.Host

	req := "GET " + path + " HTTP/1.1\r\n" +
		"Host: " + reqHost + "\r\n" +
		"Upgrade: websocket\r\n" +
		"Connection: Upgrade\r\n" +
		"Sec-WebSocket-Key: " + key + "\r\n" +
		"Sec-WebSocket-Version: 13\r\n" +
		"\r\n"

	if _, err := conn.Write([]byte(req)); err != nil {
		conn.Close()
		return nil, fmt.Errorf("cdp: write handshake: %w", err)
	}

	reader := bufio.NewReader(conn)

	statusLine, err := reader.ReadString('\n')
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("cdp: read status: %w", err)
	}

	if !strings.Contains(statusLine, "101") {
		conn.Close()
		return nil, fmt.Errorf("cdp: unexpected status: %s", strings.TrimSpace(statusLine))
	}

	// Read headers until blank line.
	acceptFound := false
	expectedAccept := computeAcceptKey(key)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			conn.Close()
			return nil, fmt.Errorf("cdp: read headers: %w", err)
		}
		line = strings.TrimSpace(line)
		if line == "" {
			break
		}
		if strings.HasPrefix(strings.ToLower(line), "sec-websocket-accept:") {
			val := strings.TrimSpace(line[len("sec-websocket-accept:"):])
			if val == expectedAccept {
				acceptFound = true
			}
		}
	}

	if !acceptFound {
		conn.Close()
		return nil, errors.New("cdp: missing or invalid Sec-WebSocket-Accept")
	}

	return &wsConn{
		conn:   conn,
		reader: reader,
		writer: bufio.NewWriter(conn),
	}, nil
}

// computeAcceptKey calculates the expected Sec-WebSocket-Accept value.
func computeAcceptKey(key string) string {
	const magic = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"
	h := sha1.New()
	h.Write([]byte(key + magic))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

// WriteText sends a text frame.
func (ws *wsConn) WriteText(data []byte) error {
	ws.mu.Lock()
	defer ws.mu.Unlock()
	return ws.writeFrame(opText, data)
}

// writeFrame writes a masked WebSocket frame (client must mask per RFC 6455).
func (ws *wsConn) writeFrame(opcode byte, payload []byte) error {
	if ws.closed {
		return errors.New("cdp: connection closed")
	}

	// FIN bit + opcode.
	if err := ws.writer.WriteByte(0x80 | opcode); err != nil {
		return err
	}

	length := len(payload)
	// Mask bit always set (client -> server).
	switch {
	case length <= 125:
		if err := ws.writer.WriteByte(0x80 | byte(length)); err != nil {
			return err
		}
	case length <= 65535:
		if err := ws.writer.WriteByte(0x80 | 126); err != nil {
			return err
		}
		buf := make([]byte, 2)
		binary.BigEndian.PutUint16(buf, uint16(length))
		if _, err := ws.writer.Write(buf); err != nil {
			return err
		}
	default:
		if err := ws.writer.WriteByte(0x80 | 127); err != nil {
			return err
		}
		buf := make([]byte, 8)
		binary.BigEndian.PutUint64(buf, uint64(length))
		if _, err := ws.writer.Write(buf); err != nil {
			return err
		}
	}

	// Masking key.
	mask := make([]byte, 4)
	if _, err := io.ReadFull(rand.Reader, mask); err != nil {
		return err
	}
	if _, err := ws.writer.Write(mask); err != nil {
		return err
	}

	// Masked payload.
	masked := make([]byte, length)
	for i := range payload {
		masked[i] = payload[i] ^ mask[i%4]
	}
	if _, err := ws.writer.Write(masked); err != nil {
		return err
	}

	return ws.writer.Flush()
}

// ReadMessage reads the next text message. It handles ping, pong, and close
// control frames transparently.
func (ws *wsConn) ReadMessage() ([]byte, error) {
	for {
		opcode, payload, err := ws.readFrame()
		if err != nil {
			return nil, err
		}

		switch opcode {
		case opText:
			return payload, nil
		case opPing:
			ws.mu.Lock()
			_ = ws.writeFrame(opPong, payload)
			ws.mu.Unlock()
		case opPong:
			// Ignore pong frames.
		case opClose:
			ws.mu.Lock()
			_ = ws.writeFrame(opClose, nil)
			ws.closed = true
			ws.mu.Unlock()
			return nil, errors.New("cdp: connection closed by server")
		}
	}
}

// readFrame reads a single WebSocket frame.
func (ws *wsConn) readFrame() (opcode byte, payload []byte, err error) {
	// First byte: FIN + opcode.
	b0, err := ws.reader.ReadByte()
	if err != nil {
		return 0, nil, fmt.Errorf("cdp: read frame header: %w", err)
	}
	opcode = b0 & 0x0F

	// Second byte: mask bit + length.
	b1, err := ws.reader.ReadByte()
	if err != nil {
		return 0, nil, fmt.Errorf("cdp: read frame length: %w", err)
	}
	masked := b1&0x80 != 0
	length := uint64(b1 & 0x7F)

	switch length {
	case 126:
		buf := make([]byte, 2)
		if _, err := io.ReadFull(ws.reader, buf); err != nil {
			return 0, nil, fmt.Errorf("cdp: read ext length: %w", err)
		}
		length = uint64(binary.BigEndian.Uint16(buf))
	case 127:
		buf := make([]byte, 8)
		if _, err := io.ReadFull(ws.reader, buf); err != nil {
			return 0, nil, fmt.Errorf("cdp: read ext length: %w", err)
		}
		length = binary.BigEndian.Uint64(buf)
	}

	var mask []byte
	if masked {
		mask = make([]byte, 4)
		if _, err := io.ReadFull(ws.reader, mask); err != nil {
			return 0, nil, fmt.Errorf("cdp: read mask: %w", err)
		}
	}

	payload = make([]byte, length)
	if length > 0 {
		if _, err := io.ReadFull(ws.reader, payload); err != nil {
			return 0, nil, fmt.Errorf("cdp: read payload: %w", err)
		}
	}

	if masked {
		for i := range payload {
			payload[i] ^= mask[i%4]
		}
	}

	return opcode, payload, nil
}

// Close sends a close frame and closes the underlying connection.
func (ws *wsConn) Close() error {
	ws.mu.Lock()
	if !ws.closed {
		ws.closed = true
		_ = ws.writeFrame(opClose, nil)
	}
	ws.mu.Unlock()
	return ws.conn.Close()
}
