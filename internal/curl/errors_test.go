package curl

import (
	"errors"
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

func TestIsKind(t *testing.T) {
	err := NewError(codeOperationTimedOut, "timeout")
	if !IsKind(err, ErrorTimeout) {
		t.Fatalf("IsKind(%v, timeout) = false", err)
	}
	if IsKind(err, ErrorDNS) {
		t.Fatalf("IsKind(%v, dns) = true", err)
	}
}
