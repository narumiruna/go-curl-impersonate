//go:build integration && (!native || !cgo)

package curl

import (
	"context"
	"net/http"
)

func nativeAvailable() bool {
	return false
}

func perform(_ context.Context, req *http.Request, options Options) (*http.Response, error) {
	if _, err := NewRequestSpec(req, options); err != nil {
		return nil, err
	}
	return nil, ErrNativeUnavailable
}
