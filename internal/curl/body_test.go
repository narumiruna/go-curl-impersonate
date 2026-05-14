package curl

import (
	"io"
	"testing"
)

func TestBodyReaderReadsChunks(t *testing.T) {
	reader := NewBodyReader([]byte("payload"))
	buf := make([]byte, 3)

	n, err := reader.Read(buf)
	if n != 3 || err != nil || string(buf) != "pay" {
		t.Fatalf("first read n=%d err=%v buf=%q", n, err, string(buf))
	}
	n, err = reader.Read(buf)
	if n != 3 || err != nil || string(buf) != "loa" {
		t.Fatalf("second read n=%d err=%v buf=%q", n, err, string(buf))
	}
	n, err = reader.Read(buf)
	if n != 1 || err != io.EOF || string(buf[:n]) != "d" {
		t.Fatalf("third read n=%d err=%v buf=%q", n, err, string(buf[:n]))
	}
	n, err = reader.Read(buf)
	if n != 0 || err != io.EOF {
		t.Fatalf("final read n=%d err=%v, want EOF", n, err)
	}
}

func TestBodyReaderReset(t *testing.T) {
	reader := NewBodyReader([]byte("abc"))
	buf := make([]byte, 2)
	_, _ = reader.Read(buf)
	reader.Reset()
	n, err := reader.Read(buf)
	if n != 2 || err != nil || string(buf) != "ab" {
		t.Fatalf("read after reset n=%d err=%v buf=%q", n, err, string(buf))
	}
	if reader.Len() != 3 {
		t.Fatalf("Len = %d, want 3", reader.Len())
	}
}

func TestReadBodyChunkStatuses(t *testing.T) {
	buf := make([]byte, 8)
	result := ReadBodyChunk(NewBodyReader([]byte("x")), buf)
	if result.N != 1 || result.Status != BodyReadEOF || result.Err != nil {
		t.Fatalf("result = %+v, want one-byte EOF result", result)
	}
	result = ReadBodyChunk(NewBodyReader([]byte("payload")), make([]byte, 3))
	if result.N != 3 || result.Status != BodyReadOK || result.Err != nil {
		t.Fatalf("result = %+v, want partial OK result", result)
	}
	result = ReadBodyChunk(nil, buf)
	if result.Status != BodyReadError || result.Err == nil {
		t.Fatalf("result = %+v, want error", result)
	}
}
