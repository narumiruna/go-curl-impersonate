package client

import (
	"context"
	"errors"
	"net/http"
	"net/http/cookiejar"
	"net/url"
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
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, u.String(), nil)
	if err != nil {
		t.Fatalf("NewRequestWithContext returned error: %v", err)
	}

	prepared := c.prepareRequest(req)
	if got := prepared.Header.Get("Cookie"); got != "session=abc" {
		t.Fatalf("Cookie header = %q, want session=abc", got)
	}
	if req.Header.Get("Cookie") != "" {
		t.Fatalf("prepareRequest should not mutate original request headers")
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
