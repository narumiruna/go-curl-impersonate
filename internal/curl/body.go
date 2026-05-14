package curl

import (
	"errors"
	"io"
)

// BodyReader provides deterministic chunks for a libcurl read callback.
type BodyReader struct {
	body []byte
	pos  int
}

func NewBodyReader(body []byte) *BodyReader {
	return &BodyReader{body: append([]byte(nil), body...)}
}

func (r *BodyReader) Read(dst []byte) (int, error) {
	if len(dst) == 0 {
		return 0, nil
	}
	if r.pos >= len(r.body) {
		return 0, io.EOF
	}
	n := copy(dst, r.body[r.pos:])
	r.pos += n
	if r.pos >= len(r.body) {
		return n, io.EOF
	}
	return n, nil
}

func (r *BodyReader) Reset() {
	r.pos = 0
}

func (r *BodyReader) Len() int {
	return len(r.body)
}

// BodyReadStatus maps a callback read into state a cgo layer can translate.
type BodyReadStatus int

const (
	BodyReadOK BodyReadStatus = iota
	BodyReadEOF
	BodyReadError
)

type BodyReadResult struct {
	N      int
	Status BodyReadStatus
	Err    error
}

func ReadBodyChunk(reader *BodyReader, dst []byte) BodyReadResult {
	if reader == nil {
		return BodyReadResult{Status: BodyReadError, Err: errors.New("curl: nil BodyReader")}
	}
	n, err := reader.Read(dst)
	switch {
	case err == nil:
		return BodyReadResult{N: n, Status: BodyReadOK}
	case errors.Is(err, io.EOF):
		return BodyReadResult{N: n, Status: BodyReadEOF}
	default:
		return BodyReadResult{N: n, Status: BodyReadError, Err: err}
	}
}
