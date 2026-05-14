package curl

import (
	"context"
	"errors"
	"os/exec"
	"testing"
)

func TestProbePkgConfigReturnsActionableErrorWhenMetadataMissing(t *testing.T) {
	if _, err := exec.LookPath("pkg-config"); err != nil {
		t.Skip("pkg-config is not installed")
	}

	probe, err := ProbePkgConfig(context.Background())
	if err == nil {
		if probe.Package == "" || probe.Libs == "" {
			t.Fatalf("probe = %+v, want package and libs", probe)
		}
		return
	}
	if !errors.Is(err, ErrPkgConfigUnavailable) {
		t.Fatalf("error = %v, want ErrPkgConfigUnavailable", err)
	}
}

func TestBackendPkgConfigPackage(t *testing.T) {
	tests := map[string]string{
		"curl-impersonate-chrome": "libcurl-impersonate-chrome",
		"curl-impersonate-ff":     "libcurl-impersonate-ff",
	}
	for backend, want := range tests {
		got, err := BackendPkgConfigPackage(backend)
		if err != nil {
			t.Fatalf("BackendPkgConfigPackage(%q) returned error: %v", backend, err)
		}
		if got != want {
			t.Fatalf("BackendPkgConfigPackage(%q) = %q, want %q", backend, got, want)
		}
	}
	if _, err := BackendPkgConfigPackage("curl-impersonate-safari"); err == nil {
		t.Fatal("BackendPkgConfigPackage should reject unsupported backends")
	}
}

func TestProbeBackendPkgConfigReturnsActionableErrorWhenMetadataMissing(t *testing.T) {
	if _, err := exec.LookPath("pkg-config"); err != nil {
		t.Skip("pkg-config is not installed")
	}
	_, err := ProbeBackendPkgConfig(context.Background(), "curl-impersonate-chrome")
	if err == nil {
		return
	}
	if !errors.Is(err, ErrPkgConfigUnavailable) {
		t.Fatalf("error = %v, want ErrPkgConfigUnavailable", err)
	}
}
