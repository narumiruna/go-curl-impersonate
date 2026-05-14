package curl

import (
	"context"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestNewRequestSpecSnapshotsGET(t *testing.T) {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "https://example.com/path?x=1", nil)
	if err != nil {
		t.Fatalf("NewRequestWithContext returned error: %v", err)
	}
	req.Header.Set("User-Agent", "go-curl-impersonate-test")

	spec, err := NewRequestSpec(req, Options{
		ProfileTarget:  "chrome116",
		DefaultHeaders: true,
		Timeout:        time.Second,
		FollowRedirect: true,
		TLSVerify:      true,
		HTTP2:          true,
	})
	if err != nil {
		t.Fatalf("NewRequestSpec returned error: %v", err)
	}
	if spec.Method != http.MethodGet {
		t.Fatalf("method = %q, want GET", spec.Method)
	}
	if spec.URL != "https://example.com/path?x=1" {
		t.Fatalf("url = %q", spec.URL)
	}
	if spec.Header.Get("User-Agent") != "go-curl-impersonate-test" {
		t.Fatalf("missing header snapshot")
	}
	if len(spec.Body) != 0 {
		t.Fatalf("body length = %d, want 0", len(spec.Body))
	}
	if spec.Options.ProfileTarget != "chrome116" || !spec.Options.DefaultHeaders {
		t.Fatalf("options = %+v", spec.Options)
	}
}

func TestNewRequestSpecSnapshotsPOSTBodyAndRestoresRequestBody(t *testing.T) {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, "https://example.com/post", strings.NewReader("payload"))
	if err != nil {
		t.Fatalf("NewRequestWithContext returned error: %v", err)
	}
	req.Header.Add("X-Test", "one")
	req.Header.Add("X-Test", "two")

	spec, err := NewRequestSpec(req, Options{ProfileTarget: "ff117"})
	if err != nil {
		t.Fatalf("NewRequestSpec returned error: %v", err)
	}
	if spec.Method != http.MethodPost {
		t.Fatalf("method = %q, want POST", spec.Method)
	}
	if string(spec.Body) != "payload" {
		t.Fatalf("body = %q, want payload", string(spec.Body))
	}
	restored, err := io.ReadAll(req.Body)
	if err != nil {
		t.Fatalf("ReadAll restored body returned error: %v", err)
	}
	if string(restored) != "payload" {
		t.Fatalf("restored body = %q, want payload", string(restored))
	}
	if req.GetBody == nil {
		t.Fatal("GetBody should be installed after snapshotting")
	}
}

func TestHeaderLinesAreDeterministic(t *testing.T) {
	spec := RequestSpec{Header: http.Header{
		"X-Zeta":  []string{"last"},
		"X-Alpha": []string{"first", "second"},
	}}
	got := spec.HeaderLines()
	want := []string{"X-Alpha: first", "X-Alpha: second", "X-Zeta: last"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("HeaderLines = %v, want %v", got, want)
	}
}

func TestRequestSpecOptionSteps(t *testing.T) {
	spec := RequestSpec{
		Method: http.MethodPost,
		URL:    "https://example.com/post",
		Header: http.Header{
			"X-Test": []string{"one"},
		},
		Body: []byte("payload"),
	}
	steps := spec.OptionSteps()
	names := make([]string, 0, len(steps))
	for _, step := range steps {
		names = append(names, step.Name)
	}
	want := []string{
		"CURLOPT_URL",
		"CURLOPT_CUSTOMREQUEST",
		"CURLOPT_HTTPHEADER",
		"CURLOPT_POSTFIELDSIZE_LARGE",
		"CURLOPT_READFUNCTION",
	}
	if !reflect.DeepEqual(names, want) {
		t.Fatalf("option step names = %v, want %v", names, want)
	}
	if steps[3].Value != int64(len("payload")) {
		t.Fatalf("body size step = %+v", steps[3])
	}
}

func TestNewRequestSpecValidatesInputs(t *testing.T) {
	tests := []struct {
		name    string
		req     *http.Request
		options Options
	}{
		{name: "nil request", req: nil, options: Options{ProfileTarget: "chrome116"}},
		{name: "missing profile", req: mustRequest(t, "https://example.com"), options: Options{}},
		{name: "unsupported scheme", req: mustRequest(t, "ftp://example.com/file"), options: Options{ProfileTarget: "chrome116"}},
		{name: "empty host", req: mustRequest(t, "https:///path"), options: Options{ProfileTarget: "chrome116"}},
		{name: "bad proxy", req: mustRequest(t, "https://example.com"), options: Options{ProfileTarget: "chrome116", Proxy: "://bad"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := NewRequestSpec(test.req, test.options); err == nil {
				t.Fatal("NewRequestSpec should return an error")
			}
		})
	}
}

func mustRequest(t *testing.T, rawURL string) *http.Request {
	t.Helper()
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, rawURL, nil)
	if err != nil {
		t.Fatalf("NewRequestWithContext(%q) returned error: %v", rawURL, err)
	}
	return req
}
