package curl

import (
	"errors"
	"fmt"
	"testing"
)

func TestNewErrorMapsKnownCodes(t *testing.T) {
	tests := []struct {
		code int
		want error
	}{
		{codeCouldNotResolveHost, ErrDNS},
		{codeCouldNotConnect, ErrConnect},
		{codeOperationTimedOut, ErrTimeout},
		{codeSSLConnectError, ErrTLS},
		{codeCouldNotResolveProxy, ErrProxy},
		{codeHTTP2, ErrHTTP2},
	}
	for _, test := range tests {
		err := NewError(test.code, "message")
		if !errors.Is(err, test.want) {
			t.Fatalf("NewError(%d) = %v, want kind %v", test.code, err, test.want)
		}
	}
}

func TestNewErrorZeroReturnsNil(t *testing.T) {
	if err := NewError(0, "ok"); err != nil {
		t.Fatalf("NewError(0) = %v, want nil", err)
	}
}

func TestErrorKindMatching(t *testing.T) {
	err := NewError(codeOperationTimedOut, "timeout")
	for name, candidate := range map[string]error{
		"direct":  err,
		"wrapped": fmt.Errorf("perform: %w", err),
	} {
		t.Run(name, func(t *testing.T) {
			if !errors.Is(candidate, ErrTimeout) {
				t.Fatalf("errors.Is(%v, ErrTimeout) = false", candidate)
			}
			if errors.Is(candidate, ErrDNS) {
				t.Fatalf("errors.Is(%v, ErrDNS) = true", candidate)
			}
		})
	}
}
