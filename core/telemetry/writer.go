package telemetry

import "io"

// Writer writes telemetry records in TLV format to an underlying io.Writer.
// It implements the RecordWriter interface.
type Writer struct {
	w          io.Writer
	headerDone bool
	closed     bool
}

// NewWriter creates a new Writer that writes to w.
func NewWriter(w io.Writer) *Writer {
	return &Writer{w: w}
}

// Write serialises and writes a single Record. On the first call it also
// writes the file header.
func (tw *Writer) Write(rec Record) error {
	if tw.closed {
		return io.ErrClosedPipe
	}
	if !tw.headerDone {
		if err := WriteHeader(tw.w, NowNano()); err != nil {
			return err
		}
		tw.headerDone = true
	}
	return EncodeRecord(tw.w, rec)
}

// Close marks the writer as closed. If the underlying io.Writer implements
// io.Closer, it is also closed.
func (tw *Writer) Close() error {
	if tw.closed {
		return nil
	}
	tw.closed = true
	if c, ok := tw.w.(io.Closer); ok {
		return c.Close()
	}
	return nil
}
