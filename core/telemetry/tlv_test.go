package telemetry

import (
	"bytes"
	"encoding/binary"
	"io"
	"strings"
	"testing"
)

// --- File Header Tests ---

func TestWriteAndReadHeader(t *testing.T) {
	var buf bytes.Buffer
	ts := int64(1234567890)
	if err := WriteHeader(&buf, ts); err != nil {
		t.Fatalf("WriteHeader: %v", err)
	}
	if buf.Len() != HeaderSize {
		t.Fatalf("header size = %d, want %d", buf.Len(), HeaderSize)
	}
	hdr, err := ReadHeader(&buf)
	if err != nil {
		t.Fatalf("ReadHeader: %v", err)
	}
	if string(hdr.Magic[:]) != MagicBytes {
		t.Errorf("magic = %q, want %q", hdr.Magic, MagicBytes)
	}
	if hdr.Version != Version {
		t.Errorf("version = %d, want %d", hdr.Version, Version)
	}
	if hdr.CreatedAt != ts {
		t.Errorf("createdAt = %d, want %d", hdr.CreatedAt, ts)
	}
}

func TestReadHeaderTruncated(t *testing.T) {
	// Fewer than 16 bytes.
	data := []byte("SCTL\x01\x00")
	_, err := ReadHeader(bytes.NewReader(data))
	if err != ErrTruncatedHeader {
		t.Errorf("got err = %v, want ErrTruncatedHeader", err)
	}
}

func TestReadHeaderEmpty(t *testing.T) {
	_, err := ReadHeader(bytes.NewReader(nil))
	if err != ErrTruncatedHeader {
		t.Errorf("got err = %v, want ErrTruncatedHeader", err)
	}
}

func TestReadHeaderInvalidMagic(t *testing.T) {
	var buf bytes.Buffer
	buf.WriteString("XXXX")                                   // bad magic
	binary.Write(&buf, binary.LittleEndian, uint16(1))        // version
	binary.Write(&buf, binary.LittleEndian, uint16(0))        // flags
	binary.Write(&buf, binary.LittleEndian, int64(123456789)) // createdAt
	_, err := ReadHeader(&buf)
	if err != ErrInvalidMagic {
		t.Errorf("got err = %v, want ErrInvalidMagic", err)
	}
}

func TestReadHeaderUnsupportedVersion(t *testing.T) {
	var buf bytes.Buffer
	buf.WriteString("SCTL")
	binary.Write(&buf, binary.LittleEndian, uint16(99))       // bad version
	binary.Write(&buf, binary.LittleEndian, uint16(0))        // flags
	binary.Write(&buf, binary.LittleEndian, int64(123456789)) // createdAt
	_, err := ReadHeader(&buf)
	if err != ErrUnsupportedVersion {
		t.Errorf("got err = %v, want ErrUnsupportedVersion", err)
	}
}

// --- Record Encode/Decode Tests ---

func TestEncodeDecodeRecordRoundtrip(t *testing.T) {
	rec := Record{
		Timestamp: 9876543210,
		EventType: TestPass,
		Tags:      map[string]string{"suite": "api", "test": "login"},
		Detail:    []byte(`{"status":"ok"}`),
	}

	var buf bytes.Buffer
	if err := EncodeRecord(&buf, rec); err != nil {
		t.Fatalf("EncodeRecord: %v", err)
	}

	got, err := DecodeRecord(&buf)
	if err != nil {
		t.Fatalf("DecodeRecord: %v", err)
	}

	if got.Timestamp != rec.Timestamp {
		t.Errorf("timestamp = %d, want %d", got.Timestamp, rec.Timestamp)
	}
	if got.EventType != rec.EventType {
		t.Errorf("eventType = %v, want %v", got.EventType, rec.EventType)
	}
	if len(got.Tags) != len(rec.Tags) {
		t.Fatalf("tag count = %d, want %d", len(got.Tags), len(rec.Tags))
	}
	for k, v := range rec.Tags {
		if got.Tags[k] != v {
			t.Errorf("tag[%q] = %q, want %q", k, got.Tags[k], v)
		}
	}
	if !bytes.Equal(got.Detail, rec.Detail) {
		t.Errorf("detail = %q, want %q", got.Detail, rec.Detail)
	}
}

func TestEncodeDecodeEmptyTags(t *testing.T) {
	rec := Record{
		Timestamp: 111,
		EventType: SuiteStart,
		Tags:      nil,
		Detail:    []byte("hello"),
	}

	var buf bytes.Buffer
	if err := EncodeRecord(&buf, rec); err != nil {
		t.Fatalf("EncodeRecord: %v", err)
	}

	got, err := DecodeRecord(&buf)
	if err != nil {
		t.Fatalf("DecodeRecord: %v", err)
	}
	if got.Tags != nil {
		t.Errorf("tags = %v, want nil", got.Tags)
	}
	if !bytes.Equal(got.Detail, rec.Detail) {
		t.Errorf("detail mismatch")
	}
}

func TestEncodeDecodeEmptyMapTags(t *testing.T) {
	rec := Record{
		Timestamp: 222,
		EventType: SuiteEnd,
		Tags:      map[string]string{},
		Detail:    []byte("world"),
	}

	var buf bytes.Buffer
	if err := EncodeRecord(&buf, rec); err != nil {
		t.Fatalf("EncodeRecord: %v", err)
	}

	got, err := DecodeRecord(&buf)
	if err != nil {
		t.Fatalf("DecodeRecord: %v", err)
	}
	// Empty map encodes same as nil tags (zero tag count).
	if got.Tags != nil {
		t.Errorf("tags = %v, want nil", got.Tags)
	}
}

func TestEncodeDecodeEmptyDetail(t *testing.T) {
	rec := Record{
		Timestamp: 333,
		EventType: TestSkip,
		Tags:      map[string]string{"key": "val"},
		Detail:    nil,
	}

	var buf bytes.Buffer
	if err := EncodeRecord(&buf, rec); err != nil {
		t.Fatalf("EncodeRecord: %v", err)
	}

	got, err := DecodeRecord(&buf)
	if err != nil {
		t.Fatalf("DecodeRecord: %v", err)
	}
	if got.Detail != nil {
		t.Errorf("detail = %v, want nil", got.Detail)
	}
	if got.Tags["key"] != "val" {
		t.Errorf("tag missing")
	}
}

func TestEncodeDecodeSingleTag(t *testing.T) {
	rec := Record{
		Timestamp: 444,
		EventType: StepStart,
		Tags:      map[string]string{"only": "one"},
		Detail:    nil,
	}

	var buf bytes.Buffer
	if err := EncodeRecord(&buf, rec); err != nil {
		t.Fatalf("EncodeRecord: %v", err)
	}

	got, err := DecodeRecord(&buf)
	if err != nil {
		t.Fatalf("DecodeRecord: %v", err)
	}
	if len(got.Tags) != 1 || got.Tags["only"] != "one" {
		t.Errorf("tags = %v, want {only:one}", got.Tags)
	}
}

func TestEncodeDecodeManyTags(t *testing.T) {
	tags := make(map[string]string)
	for i := 0; i < 100; i++ {
		k := strings.Repeat("k", i+1)
		v := strings.Repeat("v", i+1)
		tags[k] = v
	}
	rec := Record{
		Timestamp: 555,
		EventType: Metric,
		Tags:      tags,
		Detail:    []byte("data"),
	}

	var buf bytes.Buffer
	if err := EncodeRecord(&buf, rec); err != nil {
		t.Fatalf("EncodeRecord: %v", err)
	}

	got, err := DecodeRecord(&buf)
	if err != nil {
		t.Fatalf("DecodeRecord: %v", err)
	}
	if len(got.Tags) != len(tags) {
		t.Fatalf("tag count = %d, want %d", len(got.Tags), len(tags))
	}
	for k, v := range tags {
		if got.Tags[k] != v {
			t.Errorf("tag[%q] mismatch", k)
		}
	}
}

func TestEncodeDecodeZeroLengthTagKeys(t *testing.T) {
	rec := Record{
		Timestamp: 666,
		EventType: Error,
		Tags:      map[string]string{"": "empty-key"},
		Detail:    nil,
	}

	var buf bytes.Buffer
	if err := EncodeRecord(&buf, rec); err != nil {
		t.Fatalf("EncodeRecord: %v", err)
	}

	got, err := DecodeRecord(&buf)
	if err != nil {
		t.Fatalf("DecodeRecord: %v", err)
	}
	if got.Tags[""] != "empty-key" {
		t.Errorf("empty key tag = %q, want %q", got.Tags[""], "empty-key")
	}
}

func TestEncodeDecodeZeroLengthTagValues(t *testing.T) {
	rec := Record{
		Timestamp: 777,
		EventType: Assertion,
		Tags:      map[string]string{"key": ""},
		Detail:    nil,
	}

	var buf bytes.Buffer
	if err := EncodeRecord(&buf, rec); err != nil {
		t.Fatalf("EncodeRecord: %v", err)
	}

	got, err := DecodeRecord(&buf)
	if err != nil {
		t.Fatalf("DecodeRecord: %v", err)
	}
	if got.Tags["key"] != "" {
		t.Errorf("tag value = %q, want empty", got.Tags["key"])
	}
}

func TestEncodeDecodeLargeDetail(t *testing.T) {
	detail := make([]byte, 1<<16) // 64KB
	for i := range detail {
		detail[i] = byte(i % 256)
	}
	rec := Record{
		Timestamp: 888,
		EventType: Response,
		Tags:      map[string]string{"size": "large"},
		Detail:    detail,
	}

	var buf bytes.Buffer
	if err := EncodeRecord(&buf, rec); err != nil {
		t.Fatalf("EncodeRecord: %v", err)
	}

	got, err := DecodeRecord(&buf)
	if err != nil {
		t.Fatalf("DecodeRecord: %v", err)
	}
	if !bytes.Equal(got.Detail, detail) {
		t.Errorf("large detail mismatch")
	}
}

func TestDecodeRecordEOF(t *testing.T) {
	_, err := DecodeRecord(bytes.NewReader(nil))
	if err != io.EOF {
		t.Errorf("got err = %v, want io.EOF", err)
	}
}

func TestDecodeRecordTruncatedTimestamp(t *testing.T) {
	// Only 4 bytes where 8 are needed for timestamp.
	data := []byte{0x01, 0x02, 0x03, 0x04}
	_, err := DecodeRecord(bytes.NewReader(data))
	if err != ErrTruncatedRecord {
		t.Errorf("got err = %v, want ErrTruncatedRecord", err)
	}
}

func TestDecodeRecordTruncatedEventType(t *testing.T) {
	// 8 bytes for timestamp, then 1 byte (need 2 for event type).
	data := make([]byte, 9)
	_, err := DecodeRecord(bytes.NewReader(data))
	if err != ErrTruncatedRecord {
		t.Errorf("got err = %v, want ErrTruncatedRecord", err)
	}
}

func TestDecodeRecordTruncatedTagCount(t *testing.T) {
	// 8 (ts) + 2 (event) + 1 (partial tag count).
	data := make([]byte, 11)
	_, err := DecodeRecord(bytes.NewReader(data))
	if err != ErrTruncatedRecord {
		t.Errorf("got err = %v, want ErrTruncatedRecord", err)
	}
}

func TestDecodeRecordTruncatedTagsLen(t *testing.T) {
	// 8 (ts) + 2 (event) + 2 (tag count) + 2 (partial tags len, need 4).
	data := make([]byte, 14)
	_, err := DecodeRecord(bytes.NewReader(data))
	if err != ErrTruncatedRecord {
		t.Errorf("got err = %v, want ErrTruncatedRecord", err)
	}
}

func TestDecodeRecordTruncatedDetailLen(t *testing.T) {
	// 8 (ts) + 2 (event) + 2 (tag count) + 4 (tags len) + 2 (partial detail len).
	data := make([]byte, 18)
	_, err := DecodeRecord(bytes.NewReader(data))
	if err != ErrTruncatedRecord {
		t.Errorf("got err = %v, want ErrTruncatedRecord", err)
	}
}

func TestDecodeRecordTruncatedTagsData(t *testing.T) {
	var buf bytes.Buffer
	binary.Write(&buf, binary.LittleEndian, int64(0))    // timestamp
	binary.Write(&buf, binary.LittleEndian, uint16(0))   // event type
	binary.Write(&buf, binary.LittleEndian, uint16(1))   // tag count = 1
	binary.Write(&buf, binary.LittleEndian, uint32(100)) // tags len = 100 (but no data follows)
	binary.Write(&buf, binary.LittleEndian, uint32(0))   // detail len

	_, err := DecodeRecord(&buf)
	if err != ErrTruncatedRecord {
		t.Errorf("got err = %v, want ErrTruncatedRecord", err)
	}
}

func TestDecodeRecordTruncatedDetailData(t *testing.T) {
	var buf bytes.Buffer
	binary.Write(&buf, binary.LittleEndian, int64(0))    // timestamp
	binary.Write(&buf, binary.LittleEndian, uint16(0))   // event type
	binary.Write(&buf, binary.LittleEndian, uint16(0))   // tag count
	binary.Write(&buf, binary.LittleEndian, uint32(0))   // tags len
	binary.Write(&buf, binary.LittleEndian, uint32(100)) // detail len = 100 (but no data follows)

	_, err := DecodeRecord(&buf)
	if err != ErrTruncatedRecord {
		t.Errorf("got err = %v, want ErrTruncatedRecord", err)
	}
}

func TestDecodeRecordTruncatedTagKeyLen(t *testing.T) {
	// Tags data that is too short: only 1 byte for key_len (needs 2).
	var buf bytes.Buffer
	binary.Write(&buf, binary.LittleEndian, int64(0))  // timestamp
	binary.Write(&buf, binary.LittleEndian, uint16(0)) // event type
	binary.Write(&buf, binary.LittleEndian, uint16(1)) // tag count = 1
	binary.Write(&buf, binary.LittleEndian, uint32(1)) // tags len = 1 (too short)
	binary.Write(&buf, binary.LittleEndian, uint32(0)) // detail len
	buf.WriteByte(0x00)                                // 1 byte of tags data

	_, err := DecodeRecord(&buf)
	if err != ErrTruncatedRecord {
		t.Errorf("got err = %v, want ErrTruncatedRecord", err)
	}
}

func TestDecodeRecordTruncatedTagKey(t *testing.T) {
	// key_len says 10 but only 2 bytes of tag data after key_len.
	tagsData := make([]byte, 4)
	binary.LittleEndian.PutUint16(tagsData[0:], 10) // key_len = 10
	// only 2 bytes after key_len

	var buf bytes.Buffer
	binary.Write(&buf, binary.LittleEndian, int64(0))              // timestamp
	binary.Write(&buf, binary.LittleEndian, uint16(0))             // event type
	binary.Write(&buf, binary.LittleEndian, uint16(1))             // tag count = 1
	binary.Write(&buf, binary.LittleEndian, uint32(len(tagsData))) // tags len
	binary.Write(&buf, binary.LittleEndian, uint32(0))             // detail len
	buf.Write(tagsData)

	_, err := DecodeRecord(&buf)
	if err != ErrTruncatedRecord {
		t.Errorf("got err = %v, want ErrTruncatedRecord", err)
	}
}

func TestDecodeRecordTruncatedTagValLen(t *testing.T) {
	// key_len=1, key="a", then only 1 byte for val_len (needs 2).
	tagsData := make([]byte, 4)
	binary.LittleEndian.PutUint16(tagsData[0:], 1) // key_len = 1
	tagsData[2] = 'a'                              // key
	tagsData[3] = 0x00                             // partial val_len

	var buf bytes.Buffer
	binary.Write(&buf, binary.LittleEndian, int64(0))              // timestamp
	binary.Write(&buf, binary.LittleEndian, uint16(0))             // event type
	binary.Write(&buf, binary.LittleEndian, uint16(1))             // tag count
	binary.Write(&buf, binary.LittleEndian, uint32(len(tagsData))) // tags len
	binary.Write(&buf, binary.LittleEndian, uint32(0))             // detail len
	buf.Write(tagsData)

	_, err := DecodeRecord(&buf)
	if err != ErrTruncatedRecord {
		t.Errorf("got err = %v, want ErrTruncatedRecord", err)
	}
}

func TestDecodeRecordTruncatedTagVal(t *testing.T) {
	// key_len=1, key="a", val_len=10, but no val data.
	tagsData := make([]byte, 6)
	binary.LittleEndian.PutUint16(tagsData[0:], 1)  // key_len = 1
	tagsData[2] = 'a'                               // key
	binary.LittleEndian.PutUint16(tagsData[3:], 10) // val_len = 10

	var buf bytes.Buffer
	binary.Write(&buf, binary.LittleEndian, int64(0))              // timestamp
	binary.Write(&buf, binary.LittleEndian, uint16(0))             // event type
	binary.Write(&buf, binary.LittleEndian, uint16(1))             // tag count
	binary.Write(&buf, binary.LittleEndian, uint32(len(tagsData))) // tags len
	binary.Write(&buf, binary.LittleEndian, uint32(0))             // detail len
	buf.Write(tagsData)

	_, err := DecodeRecord(&buf)
	if err != ErrTruncatedRecord {
		t.Errorf("got err = %v, want ErrTruncatedRecord", err)
	}
}

func TestMultipleRecords(t *testing.T) {
	records := []Record{
		{Timestamp: 1, EventType: SuiteStart, Tags: map[string]string{"suite": "s1"}, Detail: nil},
		{Timestamp: 2, EventType: TestStart, Tags: map[string]string{"test": "t1"}, Detail: []byte("start")},
		{Timestamp: 3, EventType: TestPass, Tags: nil, Detail: []byte("pass")},
		{Timestamp: 4, EventType: SuiteEnd, Tags: map[string]string{}, Detail: nil},
	}

	var buf bytes.Buffer
	for _, r := range records {
		if err := EncodeRecord(&buf, r); err != nil {
			t.Fatalf("EncodeRecord: %v", err)
		}
	}

	for i, want := range records {
		got, err := DecodeRecord(&buf)
		if err != nil {
			t.Fatalf("DecodeRecord[%d]: %v", i, err)
		}
		if got.Timestamp != want.Timestamp {
			t.Errorf("[%d] timestamp = %d, want %d", i, got.Timestamp, want.Timestamp)
		}
		if got.EventType != want.EventType {
			t.Errorf("[%d] eventType = %v, want %v", i, got.EventType, want.EventType)
		}
	}

	// After all records, should get EOF.
	_, err := DecodeRecord(&buf)
	if err != io.EOF {
		t.Errorf("expected io.EOF after all records, got %v", err)
	}
}

// errWriter always returns an error.
type errWriter struct{}

func (errWriter) Write([]byte) (int, error) { return 0, io.ErrShortWrite }

// limitedWriter fails after n bytes have been written.
type limitedWriter struct {
	remaining int
}

func (lw *limitedWriter) Write(p []byte) (int, error) {
	if lw.remaining <= 0 {
		return 0, io.ErrShortWrite
	}
	if len(p) > lw.remaining {
		n := lw.remaining
		lw.remaining = 0
		return n, io.ErrShortWrite
	}
	lw.remaining -= len(p)
	return len(p), nil
}

// errReader always returns an error that is not io.EOF.
type errReader struct{}

func (errReader) Read([]byte) (int, error) { return 0, io.ErrNoProgress }

func TestWriteHeaderError(t *testing.T) {
	err := WriteHeader(errWriter{}, 0)
	if err == nil {
		t.Error("expected error from WriteHeader with failing writer")
	}
}

func TestReadHeaderOtherError(t *testing.T) {
	_, err := ReadHeader(errReader{})
	if err == nil {
		t.Error("expected error from ReadHeader with failing reader")
	}
	// Should wrap the error, not return ErrTruncatedHeader.
	if err == ErrTruncatedHeader {
		t.Error("should not be ErrTruncatedHeader for non-EOF error")
	}
}

func TestEncodeRecordWriteErrors(t *testing.T) {
	rec := Record{
		Timestamp: 1,
		EventType: TestStart,
		Tags:      map[string]string{"k": "v"},
		Detail:    []byte("d"),
	}
	err := EncodeRecord(errWriter{}, rec)
	if err == nil {
		t.Error("expected error from EncodeRecord with failing writer")
	}
}

func TestEncodeRecordWriteErrorEventType(t *testing.T) {
	// Fail after timestamp (8 bytes).
	rec := Record{Timestamp: 1, EventType: TestStart, Tags: map[string]string{"k": "v"}, Detail: []byte("d")}
	err := EncodeRecord(&limitedWriter{remaining: 8}, rec)
	if err == nil {
		t.Error("expected error writing event type")
	}
}

func TestEncodeRecordWriteErrorTagCount(t *testing.T) {
	// Fail after timestamp(8) + eventType(2) = 10 bytes.
	rec := Record{Timestamp: 1, EventType: TestStart}
	err := EncodeRecord(&limitedWriter{remaining: 10}, rec)
	if err == nil {
		t.Error("expected error writing tag count")
	}
}

func TestEncodeRecordWriteErrorTagsLen(t *testing.T) {
	// Fail after timestamp(8) + eventType(2) + tagCount(2) = 12 bytes.
	rec := Record{Timestamp: 1, EventType: TestStart}
	err := EncodeRecord(&limitedWriter{remaining: 12}, rec)
	if err == nil {
		t.Error("expected error writing tags length")
	}
}

func TestEncodeRecordWriteErrorDetailLen(t *testing.T) {
	// Fail after timestamp(8) + eventType(2) + tagCount(2) + tagsLen(4) = 16 bytes.
	rec := Record{Timestamp: 1, EventType: TestStart}
	err := EncodeRecord(&limitedWriter{remaining: 16}, rec)
	if err == nil {
		t.Error("expected error writing detail length")
	}
}

func TestEncodeRecordWriteErrorTagsData(t *testing.T) {
	// Fail after the 20-byte record header, during tags data write.
	rec := Record{Timestamp: 1, EventType: TestStart, Tags: map[string]string{"key": "value"}}
	err := EncodeRecord(&limitedWriter{remaining: 20}, rec)
	if err == nil {
		t.Error("expected error writing tags data")
	}
}

func TestEncodeRecordWriteErrorDetailData(t *testing.T) {
	// Allow header(20) + tags to succeed, fail on detail.
	rec := Record{Timestamp: 1, EventType: TestStart, Tags: nil, Detail: []byte("detail-data")}
	err := EncodeRecord(&limitedWriter{remaining: 20}, rec)
	if err == nil {
		t.Error("expected error writing detail data")
	}
}

func TestDecodeRecordOtherReadErrors(t *testing.T) {
	// A reader that returns a non-EOF error immediately should propagate.
	_, err := DecodeRecord(errReader{})
	if err == nil {
		t.Error("expected error from DecodeRecord with failing reader")
	}
}

// limitedReader returns data for the first n bytes, then returns errReader's error.
type limitedReader struct {
	data []byte
	pos  int
}

func newLimitedReader(n int) *limitedReader {
	data := make([]byte, n)
	return &limitedReader{data: data}
}

func (lr *limitedReader) Read(p []byte) (int, error) {
	if lr.pos >= len(lr.data) {
		return 0, io.ErrNoProgress
	}
	n := copy(p, lr.data[lr.pos:])
	lr.pos += n
	if lr.pos >= len(lr.data) && n < len(p) {
		return n, io.ErrNoProgress
	}
	return n, nil
}

func TestDecodeRecordOtherErrorEventType(t *testing.T) {
	// Provide 8 bytes (timestamp), then fail with non-EOF error.
	_, err := DecodeRecord(newLimitedReader(8))
	if err == nil {
		t.Error("expected error reading event type")
	}
}

func TestDecodeRecordOtherErrorTagCount(t *testing.T) {
	// 8(ts) + 2(event) = 10 bytes, then fail.
	_, err := DecodeRecord(newLimitedReader(10))
	if err == nil {
		t.Error("expected error reading tag count")
	}
}

func TestDecodeRecordOtherErrorTagsLen(t *testing.T) {
	// 8(ts) + 2(event) + 2(tagcount) = 12 bytes, then fail.
	_, err := DecodeRecord(newLimitedReader(12))
	if err == nil {
		t.Error("expected error reading tags length")
	}
}

func TestDecodeRecordOtherErrorDetailLen(t *testing.T) {
	// 8(ts) + 2(event) + 2(tagcount) + 4(tagslen) = 16 bytes, then fail.
	_, err := DecodeRecord(newLimitedReader(16))
	if err == nil {
		t.Error("expected error reading detail length")
	}
}

func TestDecodeRecordOtherErrorTagsData(t *testing.T) {
	// Build a valid record header that claims tagsLen=10, then provide only the header.
	var buf bytes.Buffer
	binary.Write(&buf, binary.LittleEndian, int64(0))   // timestamp
	binary.Write(&buf, binary.LittleEndian, uint16(0))  // event type
	binary.Write(&buf, binary.LittleEndian, uint16(1))  // tag count
	binary.Write(&buf, binary.LittleEndian, uint32(10)) // tags len = 10
	binary.Write(&buf, binary.LittleEndian, uint32(0))  // detail len
	// After writing the header, provide the header bytes to limitedReader
	// that will fail with a non-EOF error when reading tags data.
	headerBytes := buf.Bytes()
	lr := &limitedReader{data: headerBytes}
	_, err := DecodeRecord(lr)
	if err == nil {
		t.Error("expected error reading tags data")
	}
}

func TestDecodeRecordOtherErrorDetailData(t *testing.T) {
	// Build a valid record header that claims detailLen=10 with no tags.
	var buf bytes.Buffer
	binary.Write(&buf, binary.LittleEndian, int64(0))   // timestamp
	binary.Write(&buf, binary.LittleEndian, uint16(0))  // event type
	binary.Write(&buf, binary.LittleEndian, uint16(0))  // tag count
	binary.Write(&buf, binary.LittleEndian, uint32(0))  // tags len = 0
	binary.Write(&buf, binary.LittleEndian, uint32(10)) // detail len = 10
	headerBytes := buf.Bytes()
	lr := &limitedReader{data: headerBytes}
	_, err := DecodeRecord(lr)
	if err == nil {
		t.Error("expected error reading detail data")
	}
}
