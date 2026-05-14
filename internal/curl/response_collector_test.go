package curl

import (
	"context"
	"io"
	"net/http"
	"testing"
)

func TestResponseCollectorIgnoresInformationalResponses(t *testing.T) {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "https://example.com", nil)
	if err != nil {
		t.Fatalf("NewRequestWithContext returned error: %v", err)
	}
	var collector ResponseCollector
	if err := collector.AddHeaderBlock("HTTP/1.1 100 Continue\r\n\r\n"); err != nil {
		t.Fatalf("AddHeaderBlock returned error: %v", err)
	}
	if err := collector.AddHeaderBlock("HTTP/2 200\r\ncontent-type: text/plain\r\n\r\n"); err != nil {
		t.Fatalf("AddHeaderBlock returned error: %v", err)
	}
	collector.AppendBody([]byte("ok"))

	resp, err := collector.Response(req)
	if err != nil {
		t.Fatalf("Response returned error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("status code = %d, want 200", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("ReadAll returned error: %v", err)
	}
	if string(body) != "ok" {
		t.Fatalf("body = %q, want ok", string(body))
	}
}

func TestResponseCollectorKeepsLatestFinalHeaderBlock(t *testing.T) {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "https://example.com", nil)
	if err != nil {
		t.Fatalf("NewRequestWithContext returned error: %v", err)
	}
	var collector ResponseCollector
	if err := collector.AddHeaderBlock("HTTP/2 302\r\nlocation: https://example.com/final\r\n\r\n"); err != nil {
		t.Fatalf("AddHeaderBlock returned error: %v", err)
	}
	collector.AppendBody([]byte("redirect body"))
	if err := collector.AddHeaderBlock("HTTP/2 200\r\ncontent-type: application/json\r\n\r\n"); err != nil {
		t.Fatalf("AddHeaderBlock returned error: %v", err)
	}
	collector.AppendBody([]byte(`{"ok":true}`))

	resp, err := collector.Response(req)
	if err != nil {
		t.Fatalf("Response returned error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("status code = %d, want 200", resp.StatusCode)
	}
	if got := resp.Header.Get("Content-Type"); got != "application/json" {
		t.Fatalf("content-type = %q", got)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("ReadAll returned error: %v", err)
	}
	if string(body) != `{"ok":true}` {
		t.Fatalf("body = %q", string(body))
	}
}
