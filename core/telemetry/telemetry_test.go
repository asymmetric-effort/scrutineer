package telemetry

import (
	"bytes"
	"testing"
)

// Verify that Writer and Reader satisfy the interfaces.
var _ RecordWriter = (*Writer)(nil)
var _ RecordReader = (*Reader)(nil)

func TestInterfaceRoundtrip(t *testing.T) {
	var buf bytes.Buffer

	var w RecordWriter = NewWriter(&buf)
	rec := Record{
		Timestamp: NowNano(),
		EventType: Request,
		Tags:      map[string]string{"method": "GET", "url": "/api/v1/status"},
		Detail:    []byte(`{"healthy":true}`),
	}
	if err := w.Write(rec); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	var r RecordReader = NewReader(&buf)
	got, err := r.Next()
	if err != nil {
		t.Fatalf("Next: %v", err)
	}
	if got.EventType != Request {
		t.Errorf("eventType = %v, want Request", got.EventType)
	}
	if got.Tags["method"] != "GET" {
		t.Errorf("tag method = %q, want GET", got.Tags["method"])
	}
	if err := r.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
}
