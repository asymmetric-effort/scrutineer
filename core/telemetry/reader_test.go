package telemetry

import (
	"bytes"
	"io"
	"testing"
)

func TestReaderRoundtrip(t *testing.T) {
	var buf bytes.Buffer
	w := NewWriter(&buf)

	records := []Record{
		{Timestamp: 1000, EventType: SuiteStart, Tags: map[string]string{"suite": "auth"}, Detail: nil},
		{Timestamp: 2000, EventType: TestStart, Tags: map[string]string{"test": "login"}, Detail: []byte("starting")},
		{Timestamp: 3000, EventType: Assertion, Tags: map[string]string{"type": "eq"}, Detail: []byte(`{"expected":200,"actual":200}`)},
		{Timestamp: 4000, EventType: TestPass, Tags: nil, Detail: nil},
		{Timestamp: 5000, EventType: SuiteEnd, Tags: map[string]string{"suite": "auth"}, Detail: []byte("done")},
	}

	for _, rec := range records {
		if err := w.Write(rec); err != nil {
			t.Fatalf("Write: %v", err)
		}
	}
	w.Close()

	r := NewReader(&buf)
	for i, want := range records {
		got, err := r.Next()
		if err != nil {
			t.Fatalf("Next[%d]: %v", i, err)
		}
		if got.Timestamp != want.Timestamp {
			t.Errorf("[%d] timestamp = %d, want %d", i, got.Timestamp, want.Timestamp)
		}
		if got.EventType != want.EventType {
			t.Errorf("[%d] eventType = %v, want %v", i, got.EventType, want.EventType)
		}
		if len(want.Tags) > 0 {
			for k, v := range want.Tags {
				if got.Tags[k] != v {
					t.Errorf("[%d] tag[%q] = %q, want %q", i, k, got.Tags[k], v)
				}
			}
		}
		if !bytes.Equal(got.Detail, want.Detail) {
			t.Errorf("[%d] detail mismatch", i)
		}
	}

	_, err := r.Next()
	if err != io.EOF {
		t.Errorf("expected io.EOF, got %v", err)
	}
	r.Close()
}

func TestReaderInvalidMagic(t *testing.T) {
	data := []byte("BAAD\x01\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00")
	r := NewReader(bytes.NewReader(data))
	_, err := r.Next()
	if err != ErrInvalidMagic {
		t.Errorf("got err = %v, want ErrInvalidMagic", err)
	}
}

func TestReaderEmptyInput(t *testing.T) {
	r := NewReader(bytes.NewReader(nil))
	_, err := r.Next()
	if err != ErrTruncatedHeader {
		t.Errorf("got err = %v, want ErrTruncatedHeader", err)
	}
}

func TestReaderNextAfterClose(t *testing.T) {
	var buf bytes.Buffer
	WriteHeader(&buf, 0)
	r := NewReader(&buf)
	r.Close()
	_, err := r.Next()
	if err != io.ErrClosedPipe {
		t.Errorf("got err = %v, want io.ErrClosedPipe", err)
	}
}

func TestReaderCloseIdempotent(t *testing.T) {
	r := NewReader(bytes.NewReader(nil))
	r.Close()
	if err := r.Close(); err != nil {
		t.Errorf("second Close: %v", err)
	}
}

func TestReaderClosesUnderlying(t *testing.T) {
	cb := &closerBuffer{}
	r := NewReader(cb)
	if err := r.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if !cb.closed {
		t.Error("underlying closer not called")
	}
}

func TestReaderHeader(t *testing.T) {
	var buf bytes.Buffer
	w := NewWriter(&buf)
	w.Write(Record{Timestamp: 1, EventType: SuiteStart})
	w.Close()

	r := NewReader(&buf)
	r.Next() // triggers header read
	hdr := r.Header()
	if string(hdr.Magic[:]) != MagicBytes {
		t.Errorf("header magic = %q, want %q", hdr.Magic, MagicBytes)
	}
	if hdr.Version != Version {
		t.Errorf("header version = %d, want %d", hdr.Version, Version)
	}
	r.Close()
}

func TestReaderHeaderOnlyNoRecords(t *testing.T) {
	var buf bytes.Buffer
	WriteHeader(&buf, 12345)
	r := NewReader(&buf)
	_, err := r.Next()
	if err != io.EOF {
		t.Errorf("got err = %v, want io.EOF", err)
	}
	r.Close()
}

func TestReaderCorruptedRecordAfterHeader(t *testing.T) {
	var buf bytes.Buffer
	WriteHeader(&buf, 0)
	// Write partial record (only 5 bytes, need at least 20 for record header).
	buf.Write([]byte{0x01, 0x02, 0x03, 0x04, 0x05})
	r := NewReader(&buf)
	_, err := r.Next()
	if err != ErrTruncatedRecord {
		t.Errorf("got err = %v, want ErrTruncatedRecord", err)
	}
	r.Close()
}
