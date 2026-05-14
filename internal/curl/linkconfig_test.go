package curl

import (
	"context"
	"testing"
)

func TestDetectLinkConfigUsesEnvFallback(t *testing.T) {
	hidePkgConfig(t)
	t.Setenv("CGO_CFLAGS", "-I/opt/curl-impersonate/include")
	t.Setenv("CGO_LDFLAGS", "-L/opt/curl-impersonate/lib -lcurl-impersonate")

	config, err := DetectLinkConfig(context.Background(), "curl-impersonate-chrome")
	if err != nil {
		t.Fatalf("DetectLinkConfig returned error: %v", err)
	}
	if config.Source != "env" {
		t.Fatalf("Source = %q, want env", config.Source)
	}
	if config.CFlags == "" || config.LDFlags == "" {
		t.Fatalf("config = %+v, want cflags and ldflags", config)
	}
}

func TestDetectLinkConfigRejectsIncompleteEnvFallback(t *testing.T) {
	hidePkgConfig(t)
	t.Setenv("CGO_CFLAGS", "-I/opt/curl-impersonate/include")
	t.Setenv("CGO_LDFLAGS", "")

	if _, err := DetectLinkConfig(context.Background(), "curl-impersonate-chrome"); err == nil {
		t.Fatal("DetectLinkConfig should reject incomplete env fallback")
	}
}

func hidePkgConfig(t *testing.T) {
	t.Helper()
	t.Setenv("PKG_CONFIG_PATH", "")
	t.Setenv("PKG_CONFIG_LIBDIR", t.TempDir())
}
