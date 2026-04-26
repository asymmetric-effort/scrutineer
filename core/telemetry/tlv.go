package telemetry

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"sort"
)

// File format constants.
const (
	// MagicBytes identifies a scrutineer telemetry log file.
	MagicBytes = "SCTL"
	// Version is the current TLV format version.
	Version uint16 = 1
	// HeaderSize is the fixed size of the file header in bytes.
	HeaderSize = 16
)

var (
	// ErrInvalidMagic is returned when the file header magic bytes are wrong.
	ErrInvalidMagic = errors.New("telemetry: invalid magic bytes")
	// ErrUnsupportedVersion is returned when the format version is not supported.
	ErrUnsupportedVersion = errors.New("telemetry: unsupported version")
	// ErrTruncatedHeader is returned when the file header is incomplete.
	ErrTruncatedHeader = errors.New("telemetry: truncated header")
	// ErrTruncatedRecord is returned when a record is incomplete.
	ErrTruncatedRecord = errors.New("telemetry: truncated record")
)

// FileHeader represents the 16-byte file header.
type FileHeader struct {
	Magic     [4]byte
	Version   uint16
	Flags     uint16
	CreatedAt int64
}

// WriteHeader serialises the file header to w.
func WriteHeader(w io.Writer, createdAt int64) error {
	var hdr FileHeader
	copy(hdr.Magic[:], MagicBytes)
	hdr.Version = Version
	hdr.Flags = 0
	hdr.CreatedAt = createdAt
	return binary.Write(w, binary.LittleEndian, &hdr)
}

// ReadHeader reads and validates a file header from r.
func ReadHeader(r io.Reader) (FileHeader, error) {
	var hdr FileHeader
	if err := binary.Read(r, binary.LittleEndian, &hdr); err != nil {
		if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
			return hdr, ErrTruncatedHeader
		}
		return hdr, fmt.Errorf("telemetry: reading header: %w", err)
	}
	if string(hdr.Magic[:]) != MagicBytes {
		return hdr, ErrInvalidMagic
	}
	if hdr.Version != Version {
		return hdr, ErrUnsupportedVersion
	}
	return hdr, nil
}

// EncodeRecord serialises a Record to w in TLV format.
func EncodeRecord(w io.Writer, rec Record) error {
	// Encode tags into a byte slice first so we can compute lengths.
	tagsData, tagCount := encodeTags(rec.Tags)

	// Write fixed-size record header.
	// Timestamp(8) + EventType(2) + TagCount(2) + TagsLen(4) + DetailLen(4) = 20 bytes
	if err := binary.Write(w, binary.LittleEndian, rec.Timestamp); err != nil {
		return fmt.Errorf("telemetry: writing timestamp: %w", err)
	}
	if err := binary.Write(w, binary.LittleEndian, uint16(rec.EventType)); err != nil {
		return fmt.Errorf("telemetry: writing event type: %w", err)
	}
	if err := binary.Write(w, binary.LittleEndian, tagCount); err != nil {
		return fmt.Errorf("telemetry: writing tag count: %w", err)
	}
	if err := binary.Write(w, binary.LittleEndian, uint32(len(tagsData))); err != nil {
		return fmt.Errorf("telemetry: writing tags length: %w", err)
	}
	detailLen := uint32(len(rec.Detail))
	if err := binary.Write(w, binary.LittleEndian, detailLen); err != nil {
		return fmt.Errorf("telemetry: writing detail length: %w", err)
	}

	// Write tags data.
	if len(tagsData) > 0 {
		if _, err := w.Write(tagsData); err != nil {
			return fmt.Errorf("telemetry: writing tags: %w", err)
		}
	}

	// Write detail blob.
	if len(rec.Detail) > 0 {
		if _, err := w.Write(rec.Detail); err != nil {
			return fmt.Errorf("telemetry: writing detail: %w", err)
		}
	}

	return nil
}

// DecodeRecord reads a single Record from r.
func DecodeRecord(r io.Reader) (Record, error) {
	var rec Record

	// Read fixed header fields.
	if err := binary.Read(r, binary.LittleEndian, &rec.Timestamp); err != nil {
		if errors.Is(err, io.EOF) {
			return rec, io.EOF
		}
		if errors.Is(err, io.ErrUnexpectedEOF) {
			return rec, ErrTruncatedRecord
		}
		return rec, fmt.Errorf("telemetry: reading timestamp: %w", err)
	}

	var eventType uint16
	if err := binary.Read(r, binary.LittleEndian, &eventType); err != nil {
		if errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, io.EOF) {
			return rec, ErrTruncatedRecord
		}
		return rec, fmt.Errorf("telemetry: reading event type: %w", err)
	}
	rec.EventType = EventType(eventType)

	var tagCount uint16
	if err := binary.Read(r, binary.LittleEndian, &tagCount); err != nil {
		if errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, io.EOF) {
			return rec, ErrTruncatedRecord
		}
		return rec, fmt.Errorf("telemetry: reading tag count: %w", err)
	}

	var tagsLen uint32
	if err := binary.Read(r, binary.LittleEndian, &tagsLen); err != nil {
		if errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, io.EOF) {
			return rec, ErrTruncatedRecord
		}
		return rec, fmt.Errorf("telemetry: reading tags length: %w", err)
	}

	var detailLen uint32
	if err := binary.Read(r, binary.LittleEndian, &detailLen); err != nil {
		if errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, io.EOF) {
			return rec, ErrTruncatedRecord
		}
		return rec, fmt.Errorf("telemetry: reading detail length: %w", err)
	}

	// Read tags.
	if tagsLen > 0 {
		tagsData := make([]byte, tagsLen)
		if _, err := io.ReadFull(r, tagsData); err != nil {
			if errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, io.EOF) {
				return rec, ErrTruncatedRecord
			}
			return rec, fmt.Errorf("telemetry: reading tags data: %w", err)
		}
		tags, err := decodeTags(tagsData, tagCount)
		if err != nil {
			return rec, err
		}
		rec.Tags = tags
	}

	// Read detail.
	if detailLen > 0 {
		rec.Detail = make([]byte, detailLen)
		if _, err := io.ReadFull(r, rec.Detail); err != nil {
			if errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, io.EOF) {
				return rec, ErrTruncatedRecord
			}
			return rec, fmt.Errorf("telemetry: reading detail data: %w", err)
		}
	}

	return rec, nil
}

// encodeTags serialises tags as repeated (key_len uint16 + key + val_len uint16 + val).
// Keys are sorted for deterministic output.
func encodeTags(tags map[string]string) ([]byte, uint16) {
	if len(tags) == 0 {
		return nil, 0
	}

	// Sort keys for deterministic encoding.
	keys := make([]string, 0, len(tags))
	for k := range tags {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Calculate total size.
	size := 0
	for _, k := range keys {
		v := tags[k]
		size += 2 + len(k) + 2 + len(v)
	}

	buf := make([]byte, size)
	offset := 0
	for _, k := range keys {
		v := tags[k]
		binary.LittleEndian.PutUint16(buf[offset:], uint16(len(k)))
		offset += 2
		copy(buf[offset:], k)
		offset += len(k)
		binary.LittleEndian.PutUint16(buf[offset:], uint16(len(v)))
		offset += 2
		copy(buf[offset:], v)
		offset += len(v)
	}

	return buf, uint16(len(keys))
}

// decodeTags deserialises tags from a byte slice.
func decodeTags(data []byte, count uint16) (map[string]string, error) {
	tags := make(map[string]string, count)
	offset := 0
	for i := uint16(0); i < count; i++ {
		if offset+2 > len(data) {
			return nil, ErrTruncatedRecord
		}
		keyLen := int(binary.LittleEndian.Uint16(data[offset:]))
		offset += 2
		if offset+keyLen > len(data) {
			return nil, ErrTruncatedRecord
		}
		key := string(data[offset : offset+keyLen])
		offset += keyLen

		if offset+2 > len(data) {
			return nil, ErrTruncatedRecord
		}
		valLen := int(binary.LittleEndian.Uint16(data[offset:]))
		offset += 2
		if offset+valLen > len(data) {
			return nil, ErrTruncatedRecord
		}
		val := string(data[offset : offset+valLen])
		offset += valLen

		tags[key] = val
	}
	return tags, nil
}
