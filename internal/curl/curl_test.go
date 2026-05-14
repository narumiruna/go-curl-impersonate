//go:build !native

package curl

import (
	"context"
	"net/http"
	"testing"
)

func TestDefaultBuildReportsNativeUnavailable(t *testing.T) {
	if NativeAvailable() {
		t.Fatal("default build should not report native availability")
	}

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "https://example.com", nil)
	if err != nil {
		t.Fatalf("NewRequestWithContext returned error: %v", err)
	}
	_, err = Perform(context.Background(), req, Options{ProfileTarget: "chrome116", DefaultHeaders: true})
	if err != ErrNativeUnavailable {
		t.Fatalf("Perform error = %v, want ErrNativeUnavailable", err)
	}
}
