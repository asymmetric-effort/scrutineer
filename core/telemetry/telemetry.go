package telemetry

// RecordWriter defines the interface for writing telemetry records.
type RecordWriter interface {
	// Write serialises and persists a single Record.
	Write(Record) error
	// Close flushes any buffered data and releases resources.
	Close() error
}

// RecordReader defines the interface for reading telemetry records.
type RecordReader interface {
	// Next returns the next Record. It returns io.EOF when no more records
	// are available.
	Next() (Record, error)
	// Close releases resources held by the reader.
	Close() error
}
