package curl

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
)

// RequestSpec is the immutable request state the native backend translates to
// curl_easy_setopt calls.
type RequestSpec struct {
	Method  string
	URL     string
	Header  http.Header
	Body    []byte
	Options Options
}

// NewRequestSpec validates and snapshots req for native execution.
func NewRequestSpec(req *http.Request, options Options) (RequestSpec, error) {
	if req == nil {
		return RequestSpec{}, fmt.Errorf("curl: nil Request")
	}
	if req.URL == nil {
		return RequestSpec{}, fmt.Errorf("curl: nil request URL")
	}
	if req.URL.Scheme != "http" && req.URL.Scheme != "https" {
		return RequestSpec{}, fmt.Errorf("curl: unsupported URL scheme %q", req.URL.Scheme)
	}
	if req.URL.Host == "" {
		return RequestSpec{}, fmt.Errorf("curl: request URL host is empty")
	}
	if options.ProfileTarget == "" {
		return RequestSpec{}, fmt.Errorf("curl: profile target is empty")
	}
	if _, err := NewNativePlan(options); err != nil {
		return RequestSpec{}, err
	}
	if options.Proxy != "" {
		if _, err := url.ParseRequestURI(options.Proxy); err != nil {
			return RequestSpec{}, fmt.Errorf("curl: invalid proxy URL: %w", err)
		}
	}

	body, err := snapshotBody(req)
	if err != nil {
		return RequestSpec{}, err
	}

	method := req.Method
	if method == "" {
		method = http.MethodGet
	}

	return RequestSpec{
		Method:  method,
		URL:     req.URL.String(),
		Header:  req.Header.Clone(),
		Body:    body,
		Options: options,
	}, nil
}

func snapshotBody(req *http.Request) ([]byte, error) {
	if req.Body == nil {
		return nil, nil
	}
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, fmt.Errorf("curl: read request body: %w", err)
	}
	req.Body.Close()
	req.Body = io.NopCloser(bytes.NewReader(body))
	if req.GetBody == nil {
		bodyCopy := append([]byte(nil), body...)
		req.GetBody = func() (io.ReadCloser, error) {
			return io.NopCloser(bytes.NewReader(bodyCopy)), nil
		}
	}
	return body, nil
}

// HeaderLines returns header lines suitable for a curl_slist.
func (s RequestSpec) HeaderLines() []string {
	lines := make([]string, 0, len(s.Header))
	names := make([]string, 0, len(s.Header))
	for name := range s.Header {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		for _, value := range s.Header[name] {
			lines = append(lines, name+": "+value)
		}
	}
	return lines
}

// OptionSteps returns the ordered request-specific native operations.
func (s RequestSpec) OptionSteps() []OptionStep {
	steps := []OptionStep{
		{Name: "CURLOPT_URL", Value: s.URL},
		{Name: "CURLOPT_CUSTOMREQUEST", Value: s.Method},
	}
	headerLines := s.HeaderLines()
	if len(headerLines) > 0 {
		steps = append(steps, OptionStep{Name: "CURLOPT_HTTPHEADER", Value: headerLines})
	}
	if len(s.Body) > 0 {
		steps = append(steps,
			OptionStep{Name: "CURLOPT_POSTFIELDSIZE_LARGE", Value: int64(len(s.Body))},
			OptionStep{Name: "CURLOPT_COPYPOSTFIELDS", Value: "buffered request body"},
		)
	}
	return steps
}
