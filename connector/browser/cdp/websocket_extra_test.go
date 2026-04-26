package cdp

import (
	"encoding/binary"
	"net"
	"testing"
)

func TestWSConn_WriteVeryLargeFrame(t *testing.T) {
	wsURL, connCh, cleanup := mockWSServer(t)
	defer cleanup()

	ws, err := dialWebSocket(wsURL)
	if err != nil {
		t.Fatalf("dialWebSocket: %v", err)
	}
	defer ws.Close()

	serverConn := <-connCh
	defer serverConn.Close()

	// Write a message >65535 bytes to exercise the 8-byte length path.
	msg := make([]byte, 70000)
	for i := range msg {
		msg[i] = byte('B' + (i % 20))
	}
	if err := ws.WriteText(msg); err != nil {
		t.Fatalf("WriteText: %v", err)
	}

	_, payload, err := serverReadFrame(serverConn)
	if err != nil {
		t.Fatalf("serverReadFrame: %v", err)
	}
	if len(payload) != 70000 {
		t.Errorf("payload length = %d, want 70000", len(payload))
	}
}

func TestWSConn_ReadMediumServerFrame(t *testing.T) {
	wsURL, connCh, cleanup := mockWSServer(t)
	defer cleanup()

	ws, err := dialWebSocket(wsURL)
	if err != nil {
		t.Fatalf("dialWebSocket: %v", err)
	}
	defer ws.Close()

	serverConn := <-connCh
	defer serverConn.Close()

	// Send a medium frame (126-65535 bytes) from server using 2-byte length.
	mediumPayload := make([]byte, 300)
	for i := range mediumPayload {
		mediumPayload[i] = byte('M')
	}
	serverWriteTextFrame(serverConn, mediumPayload)

	data, err := ws.ReadMessage()
	if err != nil {
		t.Fatalf("ReadMessage: %v", err)
	}
	if len(data) != 300 {
		t.Errorf("length = %d, want 300", len(data))
	}
}

func TestWSConn_ReadVeryLargeServerFrame(t *testing.T) {
	wsURL, connCh, cleanup := mockWSServer(t)
	defer cleanup()

	ws, err := dialWebSocket(wsURL)
	if err != nil {
		t.Fatalf("dialWebSocket: %v", err)
	}
	defer ws.Close()

	serverConn := <-connCh
	defer serverConn.Close()

	// Send a large frame using 8-byte extended length.
	largePayload := make([]byte, 70000)
	for i := range largePayload {
		largePayload[i] = byte('L')
	}

	// Write frame with 8-byte length manually.
	header := []byte{0x81, 127} // FIN + text, 127 = 8-byte length follows
	lenBuf := make([]byte, 8)
	binary.BigEndian.PutUint64(lenBuf, uint64(len(largePayload)))
	header = append(header, lenBuf...)
	serverConn.Write(header)
	serverConn.Write(largePayload)

	data, err := ws.ReadMessage()
	if err != nil {
		t.Fatalf("ReadMessage: %v", err)
	}
	if len(data) != 70000 {
		t.Errorf("length = %d, want 70000", len(data))
	}
}

func TestWSConn_ReadMaskedServerFrame(t *testing.T) {
	wsURL, connCh, cleanup := mockWSServer(t)
	defer cleanup()

	ws, err := dialWebSocket(wsURL)
	if err != nil {
		t.Fatalf("dialWebSocket: %v", err)
	}
	defer ws.Close()

	serverConn := <-connCh
	defer serverConn.Close()

	// Send a masked frame from server (unusual but spec-legal).
	payload := []byte("masked server data")
	mask := []byte{0x12, 0x34, 0x56, 0x78}
	masked := make([]byte, len(payload))
	for i := range payload {
		masked[i] = payload[i] ^ mask[i%4]
	}

	header := []byte{0x81, 0x80 | byte(len(payload))} // FIN + text, masked
	header = append(header, mask...)
	serverConn.Write(header)
	serverConn.Write(masked)

	data, err := ws.ReadMessage()
	if err != nil {
		t.Fatalf("ReadMessage: %v", err)
	}
	if string(data) != "masked server data" {
		t.Errorf("data = %q, want %q", data, "masked server data")
	}
}

func TestWSConn_ReadPongFrame(t *testing.T) {
	wsURL, connCh, cleanup := mockWSServer(t)
	defer cleanup()

	ws, err := dialWebSocket(wsURL)
	if err != nil {
		t.Fatalf("dialWebSocket: %v", err)
	}
	defer ws.Close()

	serverConn := <-connCh
	defer serverConn.Close()

	// Send a pong frame (should be ignored).
	pongHeader := []byte{0x80 | opPong, 0}
	serverConn.Write(pongHeader)

	// Then send a text frame.
	serverWriteTextFrame(serverConn, []byte("after-pong"))

	data, err := ws.ReadMessage()
	if err != nil {
		t.Fatalf("ReadMessage: %v", err)
	}
	if string(data) != "after-pong" {
		t.Errorf("data = %q, want %q", data, "after-pong")
	}
}

func TestWSConn_DoubleClose(t *testing.T) {
	wsURL, connCh, cleanup := mockWSServer(t)
	defer cleanup()

	ws, err := dialWebSocket(wsURL)
	if err != nil {
		t.Fatalf("dialWebSocket: %v", err)
	}

	serverConn := <-connCh
	defer serverConn.Close()

	// Close twice should not panic.
	ws.Close()
	err = ws.Close()
	// Second close may error but should not panic.
	_ = err
}

func TestWSConn_WriteMediumFrame(t *testing.T) {
	wsURL, connCh, cleanup := mockWSServer(t)
	defer cleanup()

	ws, err := dialWebSocket(wsURL)
	if err != nil {
		t.Fatalf("dialWebSocket: %v", err)
	}
	defer ws.Close()

	serverConn := <-connCh
	defer serverConn.Close()

	// Write message between 126 and 65535 bytes.
	msg := make([]byte, 200)
	for i := range msg {
		msg[i] = byte('C')
	}
	if err := ws.WriteText(msg); err != nil {
		t.Fatalf("WriteText: %v", err)
	}

	_, payload, err := serverReadFrame(serverConn)
	if err != nil {
		t.Fatalf("serverReadFrame: %v", err)
	}
	if len(payload) != 200 {
		t.Errorf("len = %d, want 200", len(payload))
	}
}

func TestWSConn_EmptyMessage(t *testing.T) {
	wsURL, connCh, cleanup := mockWSServer(t)
	defer cleanup()

	ws, err := dialWebSocket(wsURL)
	if err != nil {
		t.Fatalf("dialWebSocket: %v", err)
	}
	defer ws.Close()

	serverConn := <-connCh
	defer serverConn.Close()

	// Server sends empty text frame.
	serverWriteTextFrame(serverConn, []byte{})

	data, err := ws.ReadMessage()
	if err != nil {
		t.Fatalf("ReadMessage: %v", err)
	}
	if len(data) != 0 {
		t.Errorf("expected empty message, got %d bytes", len(data))
	}
}

func TestDialWebSocket_WSSDefaultPort(t *testing.T) {
	// Test wss URL without port - should fail but handle scheme correctly.
	_, err := dialWebSocket("wss://localhost/test")
	if err == nil {
		t.Error("expected error")
	}
}

func TestWSConn_ReadFrameConnectionClosed(t *testing.T) {
	wsURL, connCh, cleanup := mockWSServer(t)
	defer cleanup()

	ws, err := dialWebSocket(wsURL)
	if err != nil {
		t.Fatalf("dialWebSocket: %v", err)
	}
	defer ws.Close()

	serverConn := <-connCh
	// Close server connection immediately.
	serverConn.Close()

	_, err = ws.ReadMessage()
	if err == nil {
		t.Error("expected error on closed connection")
	}
}

func TestDialWebSocket_MissingAcceptHeader(t *testing.T) {
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
		// Read request.
		buf := make([]byte, 4096)
		conn.Read(buf)
		// Send 101 without accept header.
		resp := "HTTP/1.1 101 Switching Protocols\r\n" +
			"Upgrade: websocket\r\n" +
			"Connection: Upgrade\r\n" +
			"\r\n"
		conn.Write([]byte(resp))
	}()

	_, err = dialWebSocket("ws://" + listener.Addr().String() + "/test")
	if err == nil {
		t.Error("expected error for missing Sec-WebSocket-Accept")
	}
}

func TestDialWebSocket_UnknownSchemeDefaultPort(t *testing.T) {
	// Test URL with unknown scheme (not ws or wss) - defaults to :80.
	_, err := dialWebSocket("http://localhost/test")
	if err == nil {
		t.Error("expected error")
	}
}
