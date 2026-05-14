//go:build integration && native

package client

import (
	"errors"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/narumiruna/go-curl-impersonate/internal/curl"
)

func TestNativeClientLocalHTTP(t *testing.T) {
	profileName := os.Getenv("GO_CURL_IMPERSONATE_TEST_PROFILE")
	if profileName == "" {
		profileName = "chrome"
	}
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("cookiejar.New returned error: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/get":
			if got := r.Header.Get("X-Test-Header"); got != "header-value" {
				t.Errorf("X-Test-Header = %q, want header-value", got)
			}
			http.SetCookie(w, &http.Cookie{Name: "session", Value: "stored", Path: "/"})
			w.Header().Set("X-Response-Header", "response-value")
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte("get response"))
		case "/post":
			if got := r.Method; got != http.MethodPost {
				t.Errorf("method = %q, want POST", got)
			}
			if got := r.Header.Get("Cookie"); !strings.Contains(got, "session=stored") {
				t.Errorf("Cookie header = %q, want stored session", got)
			}
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Errorf("ReadAll body returned error: %v", err)
			}
			if got := string(body); got != "request body" {
				t.Errorf("body = %q, want request body", got)
			}
			w.WriteHeader(http.StatusAccepted)
			_, _ = w.Write([]byte("post response"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	c, err := NewClient(WithProfileName(profileName), WithCookieJar(jar))
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}

	getReq, err := http.NewRequest(http.MethodGet, server.URL+"/get", nil)
	if err != nil {
		t.Fatalf("NewRequest GET returned error: %v", err)
	}
	getReq.Header.Set("X-Test-Header", "header-value")
	getResp, err := c.Do(getReq)
	if err != nil {
		t.Fatalf("GET Do returned error: %v", err)
	}
	defer getResp.Body.Close()
	if getResp.StatusCode != http.StatusCreated {
		t.Fatalf("GET status = %d, want 201", getResp.StatusCode)
	}
	if got := getResp.Header.Get("X-Response-Header"); got != "response-value" {
		t.Fatalf("X-Response-Header = %q, want response-value", got)
	}
	getBody, err := io.ReadAll(getResp.Body)
	if err != nil {
		t.Fatalf("ReadAll GET body returned error: %v", err)
	}
	if got := string(getBody); got != "get response" {
		t.Fatalf("GET body = %q, want get response", got)
	}

	postReq, err := http.NewRequest(http.MethodPost, server.URL+"/post", strings.NewReader("request body"))
	if err != nil {
		t.Fatalf("NewRequest POST returned error: %v", err)
	}
	postResp, err := c.Do(postReq)
	if err != nil {
		t.Fatalf("POST Do returned error: %v", err)
	}
	defer postResp.Body.Close()
	if postResp.StatusCode != http.StatusAccepted {
		t.Fatalf("POST status = %d, want 202", postResp.StatusCode)
	}
	postBody, err := io.ReadAll(postResp.Body)
	if err != nil {
		t.Fatalf("ReadAll POST body returned error: %v", err)
	}
	if got := string(postBody); got != "post response" {
		t.Fatalf("POST body = %q, want post response", got)
	}
}

func TestNativeClientRedirectProxyTimeoutAndHTTP2(t *testing.T) {
	profileName := nativeTestProfile()

	t.Run("redirects", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/start":
				http.Redirect(w, r, "/final", http.StatusFound)
			case "/final":
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("redirected"))
			default:
				http.NotFound(w, r)
			}
		}))
		defer server.Close()

		c, err := NewClient(WithProfileName(profileName))
		if err != nil {
			t.Fatalf("NewClient returned error: %v", err)
		}
		req, err := http.NewRequest(http.MethodGet, server.URL+"/start", nil)
		if err != nil {
			t.Fatalf("NewRequest returned error: %v", err)
		}
		resp, err := c.Do(req)
		if err != nil {
			t.Fatalf("Do returned error: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status = %d, want 200", resp.StatusCode)
		}
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("ReadAll returned error: %v", err)
		}
		if got := string(body); got != "redirected" {
			t.Fatalf("body = %q, want redirected", got)
		}
	})

	t.Run("proxy", func(t *testing.T) {
		var seenProxyRequest atomic.Bool
		proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			seenProxyRequest.Store(true)
			if got := r.URL.String(); got != "http://example.invalid/proxied" {
				t.Errorf("proxy request URL = %q, want absolute target URL", got)
			}
			w.WriteHeader(http.StatusAccepted)
			_, _ = w.Write([]byte("proxied"))
		}))
		defer proxy.Close()

		c, err := NewClient(WithProfileName(profileName), WithProxy(proxy.URL))
		if err != nil {
			t.Fatalf("NewClient returned error: %v", err)
		}
		req, err := http.NewRequest(http.MethodGet, "http://example.invalid/proxied", nil)
		if err != nil {
			t.Fatalf("NewRequest returned error: %v", err)
		}
		resp, err := c.Do(req)
		if err != nil {
			t.Fatalf("Do returned error: %v", err)
		}
		defer resp.Body.Close()
		if !seenProxyRequest.Load() {
			t.Fatal("proxy server did not receive the request")
		}
		if resp.StatusCode != http.StatusAccepted {
			t.Fatalf("status = %d, want 202", resp.StatusCode)
		}
	})

	t.Run("timeout", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(250 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		c, err := NewClient(WithProfileName(profileName), WithTimeout(20*time.Millisecond))
		if err != nil {
			t.Fatalf("NewClient returned error: %v", err)
		}
		req, err := http.NewRequest(http.MethodGet, server.URL, nil)
		if err != nil {
			t.Fatalf("NewRequest returned error: %v", err)
		}
		resp, err := c.Do(req)
		if resp != nil {
			resp.Body.Close()
		}
		if !errors.Is(err, curl.ErrTimeout) {
			t.Fatalf("Do error = %v, want timeout", err)
		}
	})

	t.Run("http2", func(t *testing.T) {
		server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.ProtoMajor != 2 {
				t.Errorf("request proto = %s, want HTTP/2", r.Proto)
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("h2"))
		}))
		server.EnableHTTP2 = true
		server.StartTLS()
		defer server.Close()

		c, err := NewClient(WithProfileName(profileName), WithTLSVerify(false), WithHTTP2(true))
		if err != nil {
			t.Fatalf("NewClient returned error: %v", err)
		}
		req, err := http.NewRequest(http.MethodGet, server.URL, nil)
		if err != nil {
			t.Fatalf("NewRequest returned error: %v", err)
		}
		resp, err := c.Do(req)
		if err != nil {
			t.Fatalf("Do returned error: %v", err)
		}
		defer resp.Body.Close()
		if resp.ProtoMajor != 2 {
			t.Fatalf("response proto = %s, want HTTP/2", resp.Proto)
		}
	})
}

func nativeTestProfile() string {
	profileName := os.Getenv("GO_CURL_IMPERSONATE_TEST_PROFILE")
	if profileName == "" {
		return "chrome"
	}
	return profileName
}
