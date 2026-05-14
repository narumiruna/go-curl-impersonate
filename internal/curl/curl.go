package curl

import (
	"context"
	"errors"
	"net/http"
	"time"
)

var ErrNativeUnavailable = errors.New("curl: native curl-impersonate backend is unavailable")

// Options carries the request settings that will be translated to libcurl
// options by the native backend.
type Options struct {
	ProfileTarget  string
	DefaultHeaders bool
	Timeout        time.Duration
	Proxy          string
	FollowRedirect bool
	MaxRedirects   int
	TLSVerify      bool
	HTTP2          bool
}

// NativeAvailable reports whether this build can perform requests through
// libcurl-impersonate.
func NativeAvailable() bool {
	return nativeAvailable()
}

// Perform executes req with the native curl-impersonate backend.
func Perform(ctx context.Context, req *http.Request, options Options) (*http.Response, error) {
	return perform(ctx, req, options)
}
