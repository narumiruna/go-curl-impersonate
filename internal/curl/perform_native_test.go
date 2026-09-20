//go:build integration && native && cgo

package curl

import (
	"errors"
	"fmt"
	"slices"
	"testing"
)

func TestNativeResourcesFree(t *testing.T) {
	names := []string{"target", "proxy", "URL", "method", "headers", "body"}
	for stopAfter := 0; stopAfter <= len(names); stopAfter++ {
		t.Run(fmt.Sprintf("allocations-%d", stopAfter), func(t *testing.T) {
			var resources nativeResources
			var freed []string
			setupErr := errors.New("setup failed")
			err := func() error {
				defer resources.free()
				for i, name := range names {
					if i == stopAfter {
						return setupErr
					}
					resources = append(resources, func() { freed = append(freed, name) })
				}
				return nil
			}()
			if (stopAfter < len(names) && err != setupErr) || (stopAfter == len(names) && err != nil) {
				t.Fatalf("setup error = %v after %d allocations", err, stopAfter)
			}
			want := slices.Clone(names[:stopAfter])
			slices.Reverse(want)
			if !slices.Equal(freed, want) {
				t.Fatalf("release order = %v, want %v", freed, want)
			}
			if resources != nil {
				t.Fatal("cleanup retained allocation closures")
			}
			resources.free()
			if !slices.Equal(freed, want) {
				t.Fatalf("repeated cleanup released resources twice: %v", freed)
			}
		})
	}
}

func TestPerformRejectsUnknownImpersonationTarget(t *testing.T) {
	req := mustRequest(t, "http://127.0.0.1:1/")
	resp, err := Perform(req.Context(), req, Options{ProfileTarget: "not-a-real-profile"})
	if resp != nil {
		resp.Body.Close()
		t.Fatal("failed impersonation returned a response")
	}
	const want = "curl_easy_impersonate: curl: unknown: A libcurl function was given a bad argument"
	if err == nil || err.Error() != want {
		t.Fatalf("Perform error = %v, want %q", err, want)
	}
	var nativeErr *Error
	if !errors.As(err, &nativeErr) || nativeErr.Code != 43 || nativeErr.Kind != ErrorUnknown {
		t.Fatalf("Perform error = %v, want wrapped CURLE_BAD_FUNCTION_ARGUMENT (43)", err)
	}
}
