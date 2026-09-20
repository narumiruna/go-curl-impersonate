package client

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/narumiruna/go-curl-impersonate/impersonate"
	"github.com/narumiruna/go-curl-impersonate/internal/curl"
)

func TestNewClientDefaults(t *testing.T) {
	c, err := NewClient()
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}
	config := c.Config()
	if config.Profile.Target != impersonate.DefaultChrome {
		t.Fatalf("default profile = %q, want %q", config.Profile.Target, impersonate.DefaultChrome)
	}
	if !config.FollowRedirect || !config.TLSVerify || !config.HTTP2 {
		t.Fatalf("default flags = redirects:%v tls:%v http2:%v, want all true", config.FollowRedirect, config.TLSVerify, config.HTTP2)
	}
	if config.MaxRedirects != 10 {
		t.Fatalf("default max redirects = %d, want 10", config.MaxRedirects)
	}
}

func TestNewClientOptions(t *testing.T) {
	c, err := NewClient(
		WithProfileName("firefox"),
		WithTimeout(5*time.Second),
		WithProxy("http://127.0.0.1:8080"),
		WithDefaultCookieJar(),
		WithRedirects(false),
		WithMaxRedirects(3),
		WithTLSVerify(false),
		WithHTTP2(false),
	)
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}
	config := c.Config()
	if config.Profile.Target != impersonate.DefaultFirefox {
		t.Fatalf("profile = %q, want %q", config.Profile.Target, impersonate.DefaultFirefox)
	}
	if config.Timeout != 5*time.Second {
		t.Fatalf("timeout = %v, want 5s", config.Timeout)
	}
	if config.Proxy != "http://127.0.0.1:8080" {
		t.Fatalf("proxy = %q", config.Proxy)
	}
	if config.Jar == nil {
		t.Fatal("cookie jar should be configured")
	}
	if config.FollowRedirect || config.TLSVerify || config.HTTP2 {
		t.Fatalf("flags = redirects:%v tls:%v http2:%v, want all false", config.FollowRedirect, config.TLSVerify, config.HTTP2)
	}
	if config.MaxRedirects != 3 {
		t.Fatalf("max redirects = %d, want 3", config.MaxRedirects)
	}
}

func TestPrepareRequestAddsCookiesFromJar(t *testing.T) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("cookiejar.New returned error: %v", err)
	}
	u, err := url.Parse("https://example.com/path")
	if err != nil {
		t.Fatalf("url.Parse returned error: %v", err)
	}
	jar.SetCookies(u, []*http.Cookie{{Name: "session", Value: "abc"}})
	c, err := NewClient(WithCookieJar(jar))
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}
	type contextKey struct{}
	ctx, cancel := context.WithCancel(context.WithValue(context.Background(), contextKey{}, "request-value"))
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), strings.NewReader("payload"))
	if err != nil {
		t.Fatalf("NewRequestWithContext returned error: %v", err)
	}
	replayCalls := 0
	req.GetBody = func() (io.ReadCloser, error) {
		replayCalls++
		return io.NopCloser(strings.NewReader("custom replay")), nil
	}

	prepared := c.prepareRequest(req)
	if prepared == req || prepared.Body != req.Body {
		t.Fatal("prepareRequest must clone the request but share its body")
	}
	if prepared.Context() != ctx || prepared.Context().Value(contextKey{}) != "request-value" {
		t.Fatal("prepareRequest changed the request context")
	}
	cancel()
	if !errors.Is(prepared.Context().Err(), context.Canceled) {
		t.Fatal("prepared request did not retain context cancellation")
	}
	if prepared.GetBody == nil {
		t.Fatal("prepareRequest removed GetBody")
	}
	replay, err := prepared.GetBody()
	if err != nil {
		t.Fatal(err)
	}
	defer replay.Close()
	body, err := io.ReadAll(replay)
	if err != nil || string(body) != "custom replay" || replayCalls != 1 {
		t.Fatalf("GetBody = %q, %v; calls = %d", body, err, replayCalls)
	}
	if got := prepared.Header.Get("Cookie"); got != "session=abc" {
		t.Fatalf("Cookie header = %q, want session=abc", got)
	}
	if req.Header.Get("Cookie") != "" {
		t.Fatalf("prepareRequest should not mutate original request headers")
	}
}

func TestPrepareZeroValueRequest(t *testing.T) {
	c, err := NewClient()
	if err != nil {
		t.Fatal(err)
	}
	req := &http.Request{}
	prepared := c.prepareRequest(req)
	if prepared == req || prepared.Context() == nil || prepared.Context() != req.Context() {
		t.Fatal("prepareRequest must clone a zero-value request with its background context")
	}
	if prepared.URL != nil || prepared.Body != nil || prepared.GetBody != nil {
		t.Fatal("prepareRequest changed zero-value request fields")
	}
}

func TestDoRejectsNilInputs(t *testing.T) {
	c, err := NewClient()
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name    string
		client  *Client
		req     *http.Request
		wantErr string
	}{
		{"nil client", nil, nil, "client: nil Client"},
		{"nil request", c, nil, "client: nil Request"},
		{"zero-value request", c, &http.Request{}, "curl: nil request URL"},
	} {
		t.Run(test.name, func(t *testing.T) {
			resp, err := test.client.Do(test.req)
			if resp != nil || err == nil || err.Error() != test.wantErr {
				t.Fatalf("Do = %v, %v; want nil, %q", resp, err, test.wantErr)
			}
		})
	}
}

func TestStoreResponseCookies(t *testing.T) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("cookiejar.New returned error: %v", err)
	}
	u, err := url.Parse("https://example.com/path")
	if err != nil {
		t.Fatalf("url.Parse returned error: %v", err)
	}
	c, err := NewClient(WithCookieJar(jar))
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}
	c.storeResponseCookies(u, &http.Response{
		Header: http.Header{"Set-Cookie": []string{"session=stored; Path=/"}},
	})

	cookies := jar.Cookies(u)
	if len(cookies) != 1 || cookies[0].Name != "session" || cookies[0].Value != "stored" {
		t.Fatalf("cookies = %+v, want stored session cookie", cookies)
	}
}

func TestNewClientRejectsInvalidOptions(t *testing.T) {
	if _, err := NewClient(WithProfileName("chrome999")); err == nil {
		t.Fatal("NewClient should reject unsupported profiles")
	}
	if _, err := NewClient(WithTimeout(-time.Second)); err == nil {
		t.Fatal("NewClient should reject negative timeout")
	}
	if _, err := NewClient(WithMaxRedirects(-1)); err == nil {
		t.Fatal("NewClient should reject negative max redirects")
	}
}

func TestDoDefaultBuildReturnsNativeUnavailable(t *testing.T) {
	if NativeAvailable() {
		t.Skip("native backend is available in this build")
	}
	c, err := NewClient()
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "https://example.com", nil)
	if err != nil {
		t.Fatalf("NewRequestWithContext returned error: %v", err)
	}
	_, err = c.Do(req)
	if !errors.Is(err, curl.ErrNativeUnavailable) {
		t.Fatalf("Do error = %v, want ErrNativeUnavailable", err)
	}
}

func TestDoValidatesRequestBeforeNativeBackend(t *testing.T) {
	c, err := NewClient()
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "ftp://example.com/file", nil)
	if err != nil {
		t.Fatalf("NewRequestWithContext returned error: %v", err)
	}
	_, err = c.Do(req)
	if err == nil || errors.Is(err, curl.ErrNativeUnavailable) {
		t.Fatalf("Do error = %v, want request validation error before native unavailable", err)
	}
}
