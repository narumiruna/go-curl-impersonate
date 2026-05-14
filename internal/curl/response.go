package curl

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
)

// ResponseSpec is the response state collected by native callbacks.
type ResponseSpec struct {
	StatusCode int
	Status     string
	Header     http.Header
	Body       []byte
}

// ResponseCollector accumulates response data delivered by native callbacks.
type ResponseCollector struct {
	current ResponseSpec
	body    []byte
}

// AddHeaderBlock records one complete HTTP response header block.
func (c *ResponseCollector) AddHeaderBlock(block string) error {
	spec, err := ParseHeaderBlock(block)
	if err != nil {
		return err
	}
	if spec.StatusCode >= 100 && spec.StatusCode < 200 {
		return nil
	}
	c.current = spec
	c.body = nil
	return nil
}

// AppendBody appends response body bytes for the current final response.
func (c *ResponseCollector) AppendBody(chunk []byte) {
	c.body = append(c.body, chunk...)
}

// Response builds a standard Go response from the collected callback state.
func (c *ResponseCollector) Response(req *http.Request) (*http.Response, error) {
	spec := c.current
	spec.Body = append([]byte(nil), c.body...)
	return NewHTTPResponse(req, spec)
}

// NewHTTPResponse builds a standard Go response from native callback state.
func NewHTTPResponse(req *http.Request, spec ResponseSpec) (*http.Response, error) {
	if spec.StatusCode <= 0 {
		return nil, fmt.Errorf("curl: response status code is empty")
	}
	status := spec.Status
	if status == "" {
		status = strconv.Itoa(spec.StatusCode) + " " + http.StatusText(spec.StatusCode)
	}
	return &http.Response{
		StatusCode:    spec.StatusCode,
		Status:        status,
		Header:        spec.Header.Clone(),
		Body:          io.NopCloser(bytes.NewReader(spec.Body)),
		ContentLength: int64(len(spec.Body)),
		Request:       req,
	}, nil
}

// ParseHeaderBlock parses one HTTP header block as delivered by libcurl header
// callbacks.
func ParseHeaderBlock(block string) (ResponseSpec, error) {
	lines := strings.Split(strings.ReplaceAll(block, "\r\n", "\n"), "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) == "" {
		return ResponseSpec{}, fmt.Errorf("curl: empty response header block")
	}
	statusCode, status, err := parseStatusLine(lines[0])
	if err != nil {
		return ResponseSpec{}, err
	}
	header := make(http.Header)
	for _, line := range lines[1:] {
		line = strings.TrimRight(line, "\r")
		if strings.TrimSpace(line) == "" {
			continue
		}
		name, value, ok := strings.Cut(line, ":")
		if !ok {
			return ResponseSpec{}, fmt.Errorf("curl: malformed response header line %q", line)
		}
		header.Add(strings.TrimSpace(name), strings.TrimSpace(value))
	}
	return ResponseSpec{StatusCode: statusCode, Status: status, Header: header}, nil
}

func parseStatusLine(line string) (int, string, error) {
	fields := strings.Fields(strings.TrimSpace(line))
	if len(fields) < 2 || !strings.HasPrefix(fields[0], "HTTP/") {
		return 0, "", fmt.Errorf("curl: malformed response status line %q", line)
	}
	statusCode, err := strconv.Atoi(fields[1])
	if err != nil {
		return 0, "", fmt.Errorf("curl: malformed response status code %q", fields[1])
	}
	status := strconv.Itoa(statusCode)
	if len(fields) > 2 {
		status += " " + strings.Join(fields[2:], " ")
	} else if text := http.StatusText(statusCode); text != "" {
		status += " " + text
	}
	return statusCode, status, nil
}
