package curl

import (
	"net/http"
	"reflect"
	"testing"
	"time"
)

func TestNewOperationPlanOrdersNativeBeforeRequestSteps(t *testing.T) {
	spec := RequestSpec{
		Method: http.MethodPost,
		URL:    "https://example.com/post",
		Header: http.Header{"X-Test": []string{"one"}},
		Body:   []byte("payload"),
		Options: Options{
			ProfileTarget:  "chrome116",
			DefaultHeaders: true,
			Timeout:        time.Second,
			FollowRedirect: true,
			MaxRedirects:   3,
			TLSVerify:      true,
			HTTP2:          true,
		},
	}
	plan, err := NewOperationPlan(spec)
	if err != nil {
		t.Fatalf("NewOperationPlan returned error: %v", err)
	}
	names := make([]string, 0, len(plan.Steps))
	for _, step := range plan.Steps {
		names = append(names, step.Name)
	}
	want := []string{
		"curl_easy_impersonate.target",
		"curl_easy_impersonate.default_headers",
		"CURLOPT_TIMEOUT_MS",
		"CURLOPT_FOLLOWLOCATION",
		"CURLOPT_MAXREDIRS",
		"CURLOPT_SSL_VERIFYPEER",
		"CURLOPT_SSL_VERIFYHOST",
		"CURLOPT_HTTP_VERSION",
		"CURLOPT_URL",
		"CURLOPT_CUSTOMREQUEST",
		"CURLOPT_HTTPHEADER",
		"CURLOPT_POSTFIELDSIZE_LARGE",
		"CURLOPT_COPYPOSTFIELDS",
	}
	if !reflect.DeepEqual(names, want) {
		t.Fatalf("operation step names = %v, want %v", names, want)
	}
}
