package curl

import (
	"context"
	"io"
	"net/http"
	"testing"
)

func TestParseHeaderBlock(t *testing.T) {
	spec, err := ParseHeaderBlock("HTTP/2 200\r\ncontent-type: application/json\r\nset-cookie: session=abc; Path=/\r\nset-cookie: theme=dark; Path=/\r\n\r\n")
	if err != nil {
		t.Fatalf("ParseHeaderBlock returned error: %v", err)
	}
	if spec.StatusCode != 200 || spec.Status != "200 OK" {
		t.Fatalf("status = %d %q, want 200 OK", spec.StatusCode, spec.Status)
	}
	if got := spec.Header.Get("Content-Type"); got != "application/json" {
		t.Fatalf("content-type = %q", got)
	}
	if got := spec.Header.Values("Set-Cookie"); len(got) != 2 {
		t.Fatalf("set-cookie = %v, want 2 values", got)
	}
}

func TestNewHTTPResponse(t *testing.T) {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "https://example.com", nil)
	if err != nil {
		t.Fatalf("NewRequestWithContext returned error: %v", err)
	}
	resp, err := NewHTTPResponse(req, ResponseSpec{
		StatusCode: 201,
		Header:     http.Header{"Content-Type": []string{"text/plain"}},
		Body:       []byte("created"),
	})
	if err != nil {
		t.Fatalf("NewHTTPResponse returned error: %v", err)
	}
	if resp.Status != "201 Created" || resp.StatusCode != 201 {
		t.Fatalf("status = %q/%d", resp.Status, resp.StatusCode)
	}
	if resp.Request != req {
		t.Fatal("response should retain request")
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("ReadAll returned error: %v", err)
	}
	if string(body) != "created" {
		t.Fatalf("body = %q, want created", string(body))
	}
	if resp.ContentLength != int64(len("created")) {
		t.Fatalf("content length = %d", resp.ContentLength)
	}
}

func TestParseHeaderBlockRejectsMalformedInput(t *testing.T) {
	for _, input := range []string{"", "200 OK\r\n", "HTTP/2 nope\r\n", "HTTP/1.1 200\r\nbad-header\r\n"} {
		if _, err := ParseHeaderBlock(input); err == nil {
			t.Fatalf("ParseHeaderBlock(%q) should fail", input)
		}
	}
}
