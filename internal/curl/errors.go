package curl

import (
	"errors"
	"fmt"
)

type ErrorKind string

const (
	ErrorUnknown     ErrorKind = "unknown"
	ErrorDNS         ErrorKind = "dns"
	ErrorConnect     ErrorKind = "connect"
	ErrorTimeout     ErrorKind = "timeout"
	ErrorTLS         ErrorKind = "tls"
	ErrorProxy       ErrorKind = "proxy"
	ErrorHTTP2       ErrorKind = "http2"
	ErrorImpersonate ErrorKind = "impersonate"
)

type Error struct {
	Code    int
	Kind    ErrorKind
	Message string
}

func (e *Error) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("curl: %s: %s", e.Kind, e.Message)
	}
	return fmt.Sprintf("curl: %s error code %d", e.Kind, e.Code)
}

func (e *Error) Is(target error) bool {
	other, ok := target.(*Error)
	if !ok {
		return false
	}
	if other.Kind != "" && e.Kind != other.Kind {
		return false
	}
	if other.Code != 0 && e.Code != other.Code {
		return false
	}
	return true
}

var (
	ErrDNS         = &Error{Kind: ErrorDNS}
	ErrConnect     = &Error{Kind: ErrorConnect}
	ErrTimeout     = &Error{Kind: ErrorTimeout}
	ErrTLS         = &Error{Kind: ErrorTLS}
	ErrProxy       = &Error{Kind: ErrorProxy}
	ErrHTTP2       = &Error{Kind: ErrorHTTP2}
	ErrImpersonate = &Error{Kind: ErrorImpersonate}
)

const (
	codeCouldNotResolveProxy = 5
	codeCouldNotResolveHost  = 6
	codeCouldNotConnect      = 7
	codeOperationTimedOut    = 28
	codeSSLConnectError      = 35
	codeTooManyRedirects     = 47
	codeHTTP2                = 16
)

// NewError converts a native CURLcode into a Go error category.
func NewError(code int, message string) error {
	if code == 0 {
		return nil
	}
	kind := ErrorUnknown
	switch code {
	case codeCouldNotResolveProxy:
		kind = ErrorProxy
	case codeCouldNotResolveHost:
		kind = ErrorDNS
	case codeCouldNotConnect:
		kind = ErrorConnect
	case codeHTTP2:
		kind = ErrorHTTP2
	case codeOperationTimedOut:
		kind = ErrorTimeout
	case codeSSLConnectError:
		kind = ErrorTLS
	case codeTooManyRedirects:
		kind = ErrorConnect
	}
	return &Error{Code: code, Kind: kind, Message: message}
}

func IsKind(err error, kind ErrorKind) bool {
	return errors.Is(err, &Error{Kind: kind})
}
