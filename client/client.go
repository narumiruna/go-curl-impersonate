package client

import (
	"context"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"time"

	"github.com/narumiruna/go-curl-impersonate/impersonate"
	"github.com/narumiruna/go-curl-impersonate/internal/curl"
)

// Client sends HTTP requests through curl-impersonate when built with a native
// backend.
type Client struct {
	config Config
}

// Config describes the libcurl options controlled by the high-level client.
type Config struct {
	Profile        impersonate.Profile
	Timeout        time.Duration
	Proxy          string
	Jar            http.CookieJar
	FollowRedirect bool
	MaxRedirects   int
	TLSVerify      bool
	HTTP2          bool
}

// Option configures a Client.
type Option func(*Config) error

// NewClient builds a client with Chrome impersonation defaults.
func NewClient(options ...Option) (*Client, error) {
	config := Config{
		Profile:        impersonate.Chrome(),
		FollowRedirect: true,
		MaxRedirects:   10,
		TLSVerify:      true,
		HTTP2:          true,
	}
	for _, option := range options {
		if option == nil {
			continue
		}
		if err := option(&config); err != nil {
			return nil, err
		}
	}
	return &Client{config: config}, nil
}

// Config returns a copy of the client's configuration.
func (c *Client) Config() Config {
	return c.config
}

// Do sends req through the native curl-impersonate backend.
func (c *Client) Do(req *http.Request) (*http.Response, error) {
	if c == nil {
		return nil, fmt.Errorf("client: nil Client")
	}
	if req == nil {
		return nil, fmt.Errorf("client: nil Request")
	}
	ctx := req.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	prepared := c.prepareRequest(req.WithContext(ctx))
	resp, err := curl.Perform(ctx, prepared, curl.Options{
		ProfileTarget:  c.config.Profile.Target,
		DefaultHeaders: c.config.Profile.DefaultHeaders,
		Timeout:        c.config.Timeout,
		Proxy:          c.config.Proxy,
		FollowRedirect: c.config.FollowRedirect,
		MaxRedirects:   c.config.MaxRedirects,
		TLSVerify:      c.config.TLSVerify,
		HTTP2:          c.config.HTTP2,
	})
	if err != nil {
		return nil, err
	}
	c.storeResponseCookies(prepared.URL, resp)
	return resp, nil
}

func (c *Client) prepareRequest(req *http.Request) *http.Request {
	prepared := req.Clone(req.Context())
	prepared.Body = req.Body
	if c.config.Jar == nil || prepared.URL == nil {
		return prepared
	}
	for _, cookie := range c.config.Jar.Cookies(prepared.URL) {
		prepared.AddCookie(cookie)
	}
	return prepared
}

func (c *Client) storeResponseCookies(u *url.URL, resp *http.Response) {
	if c.config.Jar == nil || u == nil || resp == nil {
		return
	}
	c.config.Jar.SetCookies(u, resp.Cookies())
}

// NativeAvailable reports whether this build can perform requests through
// libcurl-impersonate.
func NativeAvailable() bool {
	return curl.NativeAvailable()
}

// WithProfile sets the impersonation profile.
func WithProfile(profile impersonate.Profile) Option {
	return func(config *Config) error {
		if _, err := profile.Backend(); err != nil {
			return err
		}
		config.Profile = profile
		return nil
	}
}

// WithProfileName resolves and sets an impersonation profile by alias or native
// curl-impersonate target.
func WithProfileName(name string) Option {
	return func(config *Config) error {
		profile, err := impersonate.Resolve(name)
		if err != nil {
			return err
		}
		config.Profile = profile
		return nil
	}
}

func WithTimeout(timeout time.Duration) Option {
	return func(config *Config) error {
		if timeout < 0 {
			return fmt.Errorf("client: timeout must not be negative")
		}
		config.Timeout = timeout
		return nil
	}
}

func WithProxy(proxy string) Option {
	return func(config *Config) error {
		config.Proxy = proxy
		return nil
	}
}

func WithCookieJar(jar http.CookieJar) Option {
	return func(config *Config) error {
		config.Jar = jar
		return nil
	}
}

func WithDefaultCookieJar() Option {
	return func(config *Config) error {
		jar, err := cookiejar.New(nil)
		if err != nil {
			return err
		}
		config.Jar = jar
		return nil
	}
}

func WithRedirects(enabled bool) Option {
	return func(config *Config) error {
		config.FollowRedirect = enabled
		return nil
	}
}

func WithMaxRedirects(maxRedirects int) Option {
	return func(config *Config) error {
		if maxRedirects < 0 {
			return fmt.Errorf("client: max redirects must not be negative")
		}
		config.MaxRedirects = maxRedirects
		return nil
	}
}

func WithTLSVerify(enabled bool) Option {
	return func(config *Config) error {
		config.TLSVerify = enabled
		return nil
	}
}

func WithHTTP2(enabled bool) Option {
	return func(config *Config) error {
		config.HTTP2 = enabled
		return nil
	}
}
