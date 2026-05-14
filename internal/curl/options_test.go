package curl

import (
	"reflect"
	"testing"
	"time"
)

func TestNewNativePlan(t *testing.T) {
	plan, err := NewNativePlan(Options{
		ProfileTarget:  "chrome116",
		DefaultHeaders: true,
		Timeout:        1500 * time.Millisecond,
		Proxy:          "http://127.0.0.1:8080",
		FollowRedirect: true,
		MaxRedirects:   3,
		TLSVerify:      true,
		HTTP2:          true,
	})
	if err != nil {
		t.Fatalf("NewNativePlan returned error: %v", err)
	}
	if plan.ImpersonateTarget != "chrome116" || !plan.DefaultHeaders {
		t.Fatalf("plan = %+v", plan)
	}
	if plan.TimeoutMillis != 1500 {
		t.Fatalf("timeout millis = %d, want 1500", plan.TimeoutMillis)
	}
	if plan.Proxy != "http://127.0.0.1:8080" || !plan.FollowRedirect || !plan.TLSVerify || !plan.HTTP2 {
		t.Fatalf("plan = %+v", plan)
	}
	if plan.MaxRedirects != 3 {
		t.Fatalf("max redirects = %d, want 3", plan.MaxRedirects)
	}
}

func TestNewNativePlanRoundsSubMillisecondTimeout(t *testing.T) {
	plan, err := NewNativePlan(Options{ProfileTarget: "chrome116", Timeout: time.Nanosecond})
	if err != nil {
		t.Fatalf("NewNativePlan returned error: %v", err)
	}
	if plan.TimeoutMillis != 1 {
		t.Fatalf("timeout millis = %d, want 1", plan.TimeoutMillis)
	}
}

func TestNativePlanOptionSteps(t *testing.T) {
	plan, err := NewNativePlan(Options{
		ProfileTarget:  "chrome116",
		DefaultHeaders: true,
		Timeout:        time.Second,
		Proxy:          "http://127.0.0.1:8080",
		FollowRedirect: true,
		MaxRedirects:   3,
		TLSVerify:      true,
		HTTP2:          true,
	})
	if err != nil {
		t.Fatalf("NewNativePlan returned error: %v", err)
	}
	steps := plan.OptionSteps()
	names := make([]string, 0, len(steps))
	for _, step := range steps {
		names = append(names, step.Name)
	}
	want := []string{
		"curl_easy_impersonate.target",
		"curl_easy_impersonate.default_headers",
		"CURLOPT_TIMEOUT_MS",
		"CURLOPT_PROXY",
		"CURLOPT_FOLLOWLOCATION",
		"CURLOPT_MAXREDIRS",
		"CURLOPT_SSL_VERIFYPEER",
		"CURLOPT_SSL_VERIFYHOST",
		"CURLOPT_HTTP_VERSION",
	}
	if !reflect.DeepEqual(names, want) {
		t.Fatalf("option step names = %v, want %v", names, want)
	}
}

func TestNewNativePlanValidatesOptions(t *testing.T) {
	if _, err := NewNativePlan(Options{}); err == nil {
		t.Fatal("NewNativePlan should reject empty profile")
	}
	if _, err := NewNativePlan(Options{ProfileTarget: "chrome116", Timeout: -time.Second}); err == nil {
		t.Fatal("NewNativePlan should reject negative timeout")
	}
	if _, err := NewNativePlan(Options{ProfileTarget: "chrome116", MaxRedirects: -1}); err == nil {
		t.Fatal("NewNativePlan should reject negative max redirects")
	}
}
