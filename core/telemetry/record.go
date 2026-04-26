package telemetry

// Record represents a single telemetry event with its associated metadata.
type Record struct {
	// Timestamp is the nanosecond-precision time the event occurred.
	Timestamp int64
	// EventType identifies the kind of event.
	EventType EventType
	// Tags contains key-value metadata for the event.
	Tags map[string]string
	// Detail holds a variable-length payload (request/response bodies,
	// error messages, stack traces, etc.).
	Detail []byte
}
