package curl

import (
	"context"
	"testing"
)

func TestDetectLinkConfigPrefersPkgConfig(t *testing.T) {
	for _, pkg := range []string{"libcurl-impersonate-chrome", "libcurl-impersonate"} {
		t.Run(pkg, func(t *testing.T) {
			fakePkgConfig(t, pkg)
			t.Setenv("CGO_CFLAGS", "-I/env/include")
			t.Setenv("CGO_LDFLAGS", "-L/env/lib -lenv")
			config, err := DetectLinkConfig(context.Background(), "curl-impersonate-chrome")
			want := LinkConfig{
				Source:  "pkg-config",
				Package: pkg,
				CFlags:  "-I/native/include -DTEST=1",
				LDFlags: "-L/native/lib -ltest -pthread",
			}
			if err != nil || config != want {
				t.Fatalf("DetectLinkConfig = %+v, %v; want %+v, nil", config, err, want)
			}
		})
	}
}

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
