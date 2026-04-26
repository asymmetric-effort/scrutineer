package telemetry

import "io"

// Reader reads telemetry records in TLV format from an underlying io.Reader.
// It implements the RecordReader interface.
type Reader struct {
	r          io.Reader
	headerDone bool
	header     FileHeader
	closed     bool
}

// NewReader creates a new Reader that reads from r.
func NewReader(r io.Reader) *Reader {
	return &Reader{r: r}
}

// Header returns the file header. It is only valid after the first call to
// Next (or after an explicit ReadHeader).
func (tr *Reader) Header() FileHeader {
	return tr.header
}

// Next reads and returns the next Record. It returns io.EOF when no more
// records are available.
func (tr *Reader) Next() (Record, error) {
	if tr.closed {
		return Record{}, io.ErrClosedPipe
	}
	if !tr.headerDone {
		hdr, err := ReadHeader(tr.r)
		if err != nil {
			return Record{}, err
		}
		tr.header = hdr
		tr.headerDone = true
	}
	return DecodeRecord(tr.r)
}

// Close marks the reader as closed. If the underlying io.Reader implements
// io.Closer, it is also closed.
func (tr *Reader) Close() error {
	if tr.closed {
		return nil
	}
	tr.closed = true
	if c, ok := tr.r.(io.Closer); ok {
		return c.Close()
	}
	return nil
}
