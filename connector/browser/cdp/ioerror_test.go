package cdp

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"io"
	"net"
	"testing"
	"time"
)

// badConn is a net.Conn that fails all operations after N bytes written.
type badConn struct {
	net.Conn
	buf         bytes.Buffer
	failAfterN  int
	written     int
	readData    *bytes.Reader
	readFail    bool
	closeCalled bool
}

func (b *badConn) Write(p []byte) (int, error) {
	n := len(p)
	if b.failAfterN >= 0 && b.written+n > b.failAfterN {
		// Write partial, then fail.
		remain := b.failAfterN - b.written
		if remain > 0 {
			b.written += remain
			return remain, io.ErrClosedPipe
		}
		return 0, io.ErrClosedPipe
	}
	b.written += n
	b.buf.Write(p)
	return n, nil
}

func (b *badConn) Read(p []byte) (int, error) {
	if b.readFail {
		return 0, io.ErrUnexpectedEOF
	}
	if b.readData != nil {
		return b.readData.Read(p)
	}
	return 0, io.EOF
}

func (b *badConn) Close() error {
	b.closeCalled = true
	return nil
}

func (b *badConn) LocalAddr() net.Addr  { return &net.TCPAddr{} }
func (b *badConn) RemoteAddr() net.Addr { return &net.TCPAddr{} }

func (b *badConn) SetDeadline(_ time.Time) error      { return nil }
func (b *badConn) SetReadDeadline(_ time.Time) error  { return nil }
func (b *badConn) SetWriteDeadline(_ time.Time) error { return nil }

func TestWriteFrame_FlushError(t *testing.T) {
	// The writeFrame function calls Flush at the end. By making the
	// underlying writer fail, the flush will propagate the error.
	bc := &badConn{failAfterN: 3} // Allow a few bytes, then fail on flush
	ws := &wsConn{
		conn:   bc,
		reader: bufio.NewReader(bc),
		writer: bufio.NewWriter(bc), // default 4KB buffer
	}

	// Write a message large enough that Flush will try to write to the
	// underlying connection (exceeding the 3-byte limit).
	payload := make([]byte, 5000) // larger than default buf
	err := ws.writeFrame(opText, payload)
	if err == nil {
		t.Error("expected error when flush fails")
	}
}

func TestWriteFrame_ClosedConn(t *testing.T) {
	bc := &badConn{failAfterN: 1000}
	ws := &wsConn{
		conn:   bc,
		reader: bufio.NewReader(bc),
		writer: bufio.NewWriter(bc),
		closed: true,
	}

	err := ws.writeFrame(opText, []byte("test"))
	if err == nil {
		t.Error("expected error on closed connection")
	}
}

func TestReadFrame_ReadError(t *testing.T) {
	bc := &badConn{readFail: true}
	ws := &wsConn{
		conn:   bc,
		reader: bufio.NewReader(bc),
		writer: bufio.NewWriter(bc),
	}

	_, _, err := ws.readFrame()
	if err == nil {
		t.Error("expected error when reader fails")
	}
}

func TestReadFrame_TruncatedHeader(t *testing.T) {
	// Only 1 byte of header - second read should fail.
	bc := &badConn{readData: bytes.NewReader([]byte{0x81})}
	ws := &wsConn{
		conn:   bc,
		reader: bufio.NewReader(bc),
		writer: bufio.NewWriter(bc),
	}

	_, _, err := ws.readFrame()
	if err == nil {
		t.Error("expected error for truncated header")
	}
}

func TestReadFrame_TruncatedExtLength16(t *testing.T) {
	// Header says 126 (2-byte extended), but not enough data follows.
	data := []byte{0x81, 126, 0x00} // Only 1 byte of 2-byte length.
	bc := &badConn{readData: bytes.NewReader(data)}
	ws := &wsConn{
		conn:   bc,
		reader: bufio.NewReader(bc),
		writer: bufio.NewWriter(bc),
	}

	_, _, err := ws.readFrame()
	if err == nil {
		t.Error("expected error for truncated extended length")
	}
}

func TestReadFrame_TruncatedExtLength64(t *testing.T) {
	// Header says 127 (8-byte extended), but not enough data follows.
	data := []byte{0x81, 127, 0, 0, 0} // Only 3 bytes of 8-byte length.
	bc := &badConn{readData: bytes.NewReader(data)}
	ws := &wsConn{
		conn:   bc,
		reader: bufio.NewReader(bc),
		writer: bufio.NewWriter(bc),
	}

	_, _, err := ws.readFrame()
	if err == nil {
		t.Error("expected error for truncated 64-bit length")
	}
}

func TestReadFrame_TruncatedMask(t *testing.T) {
	// Masked frame but mask key data is truncated.
	data := []byte{0x81, 0x85, 0x01, 0x02} // mask bit set, length 5, only 2 bytes of mask
	bc := &badConn{readData: bytes.NewReader(data)}
	ws := &wsConn{
		conn:   bc,
		reader: bufio.NewReader(bc),
		writer: bufio.NewWriter(bc),
	}

	_, _, err := ws.readFrame()
	if err == nil {
		t.Error("expected error for truncated mask")
	}
}

func TestReadFrame_TruncatedPayload(t *testing.T) {
	// Unmasked frame with length 10 but only 3 bytes of payload.
	data := []byte{0x81, 10, 'a', 'b', 'c'}
	bc := &badConn{readData: bytes.NewReader(data)}
	ws := &wsConn{
		conn:   bc,
		reader: bufio.NewReader(bc),
		writer: bufio.NewWriter(bc),
	}

	_, _, err := ws.readFrame()
	if err == nil {
		t.Error("expected error for truncated payload")
	}
}

func TestWriteFrame_Medium126LengthError(t *testing.T) {
	// Force error after writing 2 bytes (header), testing the 126-length path error.
	bc := &badConn{failAfterN: 2}
	ws := &wsConn{
		conn:   bc,
		reader: bufio.NewReader(bc),
		writer: bufio.NewWriterSize(bc, 1),
	}

	payload := make([]byte, 200) // triggers 126 length path
	err := ws.writeFrame(opText, payload)
	if err == nil {
		t.Error("expected error for medium frame write failure")
	}
}

func TestWriteFrame_LargeLength127Error(t *testing.T) {
	// Force error after a few bytes, testing the 127-length path error.
	bc := &badConn{failAfterN: 2}
	ws := &wsConn{
		conn:   bc,
		reader: bufio.NewReader(bc),
		writer: bufio.NewWriterSize(bc, 1),
	}

	payload := make([]byte, 70000) // triggers 127 length path
	err := ws.writeFrame(opText, payload)
	if err == nil {
		t.Error("expected error for large frame write failure")
	}
}

func TestWriteFrame_SmallPayloadFlushError(t *testing.T) {
	// Test small payload (<=125) path where flush fails.
	bc := &badConn{failAfterN: 5} // Allow header bytes but fail on payload
	ws := &wsConn{
		conn:   bc,
		reader: bufio.NewReader(bc),
		writer: bufio.NewWriterSize(bc, 4), // tiny buffer
	}

	err := ws.writeFrame(opText, []byte("hi"))
	if err == nil {
		t.Error("expected error")
	}
}

func TestWriteFrame_Medium126WriteLengthError(t *testing.T) {
	// Test 126 length path where the 2-byte length write itself fails.
	bc := &badConn{failAfterN: 1} // Fail after first header byte
	ws := &wsConn{
		conn:   bc,
		reader: bufio.NewReader(bc),
		writer: bufio.NewWriterSize(bc, 2), // tiny buffer forces flushes
	}

	payload := make([]byte, 200)
	err := ws.writeFrame(opText, payload)
	if err == nil {
		t.Error("expected error for 126 length write failure")
	}
}

func TestReadMessage_ErrorFromReadFrame(t *testing.T) {
	bc := &badConn{readFail: true}
	ws := &wsConn{
		conn:   bc,
		reader: bufio.NewReader(bc),
		writer: bufio.NewWriter(bc),
	}

	_, err := ws.ReadMessage()
	if err == nil {
		t.Error("expected error from ReadMessage when readFrame fails")
	}
}

func TestDialWebSocket_ReadStatusError(t *testing.T) {
	// Server closes connection immediately after TCP connect (no HTTP response).
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
		// Read request, then close immediately.
		buf := make([]byte, 4096)
		conn.Read(buf)
		conn.Close()
	}()

	_, err = dialWebSocket("ws://" + listener.Addr().String() + "/test")
	if err == nil {
		t.Error("expected error when server closes without response")
	}
}

func TestWriteFrame_MaskWriteError(t *testing.T) {
	// Test error writing the mask key (happens after length is written).
	// Use a writer that fails after header+length bytes are written.
	bc := &badConn{failAfterN: 10}
	ws := &wsConn{
		conn:   bc,
		reader: bufio.NewReader(bc),
		writer: bufio.NewWriterSize(bc, 8), // small buffer
	}

	// Small payload (<=125), header is 2 bytes, then 4-byte mask, then payload.
	// With buffer size 8 and failAfterN 10, the flush should fail when writing mask or payload.
	err := ws.writeFrame(opText, []byte("hello"))
	if err == nil {
		t.Error("expected error writing mask or payload")
	}
}

func TestWriteFrame_PayloadWriteError(t *testing.T) {
	// Test error writing the masked payload.
	bc := &badConn{failAfterN: 15}
	ws := &wsConn{
		conn:   bc,
		reader: bufio.NewReader(bc),
		writer: bufio.NewWriterSize(bc, 10), // small buffer
	}

	err := ws.writeFrame(opText, []byte("hello world, this is a longer message"))
	if err == nil {
		t.Error("expected error writing payload")
	}
}

func TestWriteFrame_126ExtLengthWriteError(t *testing.T) {
	// Test error writing the 2-byte extended length for medium frames.
	bc := &badConn{failAfterN: 3}
	ws := &wsConn{
		conn:   bc,
		reader: bufio.NewReader(bc),
		writer: bufio.NewWriterSize(bc, 3), // very small buffer
	}

	payload := make([]byte, 200) // triggers 126 path
	err := ws.writeFrame(opText, payload)
	if err == nil {
		t.Error("expected error")
	}
}

func TestWriteFrame_127ExtLengthWriteError(t *testing.T) {
	// Test error writing the 8-byte extended length for large frames.
	bc := &badConn{failAfterN: 4}
	ws := &wsConn{
		conn:   bc,
		reader: bufio.NewReader(bc),
		writer: bufio.NewWriterSize(bc, 3),
	}

	payload := make([]byte, 70000) // triggers 127 path
	err := ws.writeFrame(opText, payload)
	if err == nil {
		t.Error("expected error")
	}
}

func TestDialWebSocket_HeaderReadError(t *testing.T) {
	// Test error reading headers after 101 status.
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
		// Read request.
		buf := make([]byte, 4096)
		conn.Read(buf)
		// Send 101 status but then close before headers are complete.
		conn.Write([]byte("HTTP/1.1 101 Switching Protocols\r\n"))
		conn.Close()
	}()

	_, err = dialWebSocket("ws://" + listener.Addr().String() + "/test")
	if err == nil {
		t.Error("expected error for header read failure")
	}
}

func TestReadFrame_16bitExtendedLength(t *testing.T) {
	// Build a valid frame with 2-byte extended length.
	payload := make([]byte, 200)
	for i := range payload {
		payload[i] = 'X'
	}

	var buf bytes.Buffer
	buf.WriteByte(0x81) // FIN + text
	buf.WriteByte(126)  // 2-byte extended length
	lenBuf := make([]byte, 2)
	binary.BigEndian.PutUint16(lenBuf, uint16(len(payload)))
	buf.Write(lenBuf)
	buf.Write(payload)

	bc := &badConn{readData: bytes.NewReader(buf.Bytes())}
	ws := &wsConn{
		conn:   bc,
		reader: bufio.NewReader(bc),
		writer: bufio.NewWriter(bc),
	}

	op, data, err := ws.readFrame()
	if err != nil {
		t.Fatalf("readFrame: %v", err)
	}
	if op != opText {
		t.Errorf("opcode = %d, want %d", op, opText)
	}
	if len(data) != 200 {
		t.Errorf("data length = %d, want 200", len(data))
	}
}
