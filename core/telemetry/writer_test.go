package telemetry

import (
	"bytes"
	"io"
	"testing"
)

func TestWriterWritesHeader(t *testing.T) {
	var buf bytes.Buffer
	w := NewWriter(&buf)
	rec := Record{Timestamp: 100, EventType: SuiteStart}
	if err := w.Write(rec); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// Verify header is present.
	hdr, err := ReadHeader(&buf)
	if err != nil {
		t.Fatalf("ReadHeader: %v", err)
	}
	if string(hdr.Magic[:]) != MagicBytes {
		t.Errorf("magic = %q, want %q", hdr.Magic, MagicBytes)
	}

	// Verify record follows.
	got, err := DecodeRecord(&buf)
	if err != nil {
		t.Fatalf("DecodeRecord: %v", err)
	}
	if got.Timestamp != 100 {
		t.Errorf("timestamp = %d, want 100", got.Timestamp)
	}
}

func TestWriterMultipleRecords(t *testing.T) {
	var buf bytes.Buffer
	w := NewWriter(&buf)

	for i := 0; i < 5; i++ {
		rec := Record{
			Timestamp: int64(i),
			EventType: EventType(i + 1),
			Tags:      map[string]string{"i": string(rune('0' + i))},
			Detail:    []byte{byte(i)},
		}
		if err := w.Write(rec); err != nil {
			t.Fatalf("Write[%d]: %v", i, err)
		}
	}
	w.Close()

	// Skip header.
	ReadHeader(&buf)

	for i := 0; i < 5; i++ {
		got, err := DecodeRecord(&buf)
		if err != nil {
			t.Fatalf("DecodeRecord[%d]: %v", i, err)
		}
		if got.Timestamp != int64(i) {
			t.Errorf("[%d] timestamp = %d", i, got.Timestamp)
		}
	}
}

func TestWriterCloseIdempotent(t *testing.T) {
	var buf bytes.Buffer
	w := NewWriter(&buf)
	if err := w.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("second Close: %v", err)
	}
}

func TestWriterWriteAfterClose(t *testing.T) {
	var buf bytes.Buffer
	w := NewWriter(&buf)
	w.Close()
	err := w.Write(Record{})
	if err != io.ErrClosedPipe {
		t.Errorf("got err = %v, want io.ErrClosedPipe", err)
	}
}

// closerBuffer is a bytes.Buffer that also implements io.Closer.
type closerBuffer struct {
	bytes.Buffer
	closed bool
}

func (cb *closerBuffer) Close() error {
	cb.closed = true
	return nil
}

func TestWriterClosesUnderlying(t *testing.T) {
	cb := &closerBuffer{}
	w := NewWriter(cb)
	w.Write(Record{Timestamp: 1, EventType: SuiteStart})
	if err := w.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if !cb.closed {
		t.Error("underlying closer not called")
	}
}

func TestWriterHeaderError(t *testing.T) {
	w := NewWriter(errWriter{})
	err := w.Write(Record{Timestamp: 1, EventType: SuiteStart})
	if err == nil {
		t.Error("expected error from Write with failing writer")
	}
}
