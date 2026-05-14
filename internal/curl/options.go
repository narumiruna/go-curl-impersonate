package curl

import (
	"fmt"
	"time"
)

// NativePlan is the ordered native work a backend must perform for one request.
type NativePlan struct {
	ImpersonateTarget string
	DefaultHeaders    bool
	TimeoutMillis     int64
	Proxy             string
	FollowRedirect    bool
	MaxRedirects      int
	TLSVerify         bool
	HTTP2             bool
}

// OptionStep is a symbolic curl operation the native backend must apply.
type OptionStep struct {
	Name  string
	Value any
}

// NewNativePlan validates and normalizes options for native execution.
func NewNativePlan(options Options) (NativePlan, error) {
	if options.ProfileTarget == "" {
		return NativePlan{}, fmt.Errorf("curl: profile target is empty")
	}
	if options.Timeout < 0 {
		return NativePlan{}, fmt.Errorf("curl: timeout must not be negative")
	}
	if options.MaxRedirects < 0 {
		return NativePlan{}, fmt.Errorf("curl: max redirects must not be negative")
	}
	return NativePlan{
		ImpersonateTarget: options.ProfileTarget,
		DefaultHeaders:    options.DefaultHeaders,
		TimeoutMillis:     durationMillis(options.Timeout),
		Proxy:             options.Proxy,
		FollowRedirect:    options.FollowRedirect,
		MaxRedirects:      options.MaxRedirects,
		TLSVerify:         options.TLSVerify,
		HTTP2:             options.HTTP2,
	}, nil
}

// OptionSteps returns the ordered native operations for this request.
func (p NativePlan) OptionSteps() []OptionStep {
	steps := []OptionStep{
		{Name: "curl_easy_impersonate.target", Value: p.ImpersonateTarget},
		{Name: "curl_easy_impersonate.default_headers", Value: p.DefaultHeaders},
	}
	if p.TimeoutMillis > 0 {
		steps = append(steps, OptionStep{Name: "CURLOPT_TIMEOUT_MS", Value: p.TimeoutMillis})
	}
	if p.Proxy != "" {
		steps = append(steps, OptionStep{Name: "CURLOPT_PROXY", Value: p.Proxy})
	}
	steps = append(steps,
		OptionStep{Name: "CURLOPT_FOLLOWLOCATION", Value: p.FollowRedirect},
	)
	if p.FollowRedirect && p.MaxRedirects > 0 {
		steps = append(steps, OptionStep{Name: "CURLOPT_MAXREDIRS", Value: p.MaxRedirects})
	}
	steps = append(steps,
		OptionStep{Name: "CURLOPT_SSL_VERIFYPEER", Value: p.TLSVerify},
		OptionStep{Name: "CURLOPT_SSL_VERIFYHOST", Value: p.TLSVerify},
	)
	if p.HTTP2 {
		steps = append(steps, OptionStep{Name: "CURLOPT_HTTP_VERSION", Value: "CURL_HTTP_VERSION_2TLS"})
	}
	return steps
}

func durationMillis(duration time.Duration) int64 {
	if duration == 0 {
		return 0
	}
	millis := duration.Milliseconds()
	if millis == 0 {
		return 1
	}
	return millis
}
