package main

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/narumiruna/go-curl-impersonate/internal/curl"
)

func TestRunPrintsSkimmableDiagnosticsAndActionableNextSteps(t *testing.T) {
	t.Parallel()

	deps := testDependencies()
	deps.nativeAvailable = func() bool { return false }
	deps.probePkgConfig = func(context.Context) (curl.PkgConfigProbe, error) {
		return curl.PkgConfigProbe{}, errors.New("long pkg-config failure")
	}
	deps.detectLinkConfig = func(context.Context, string) (curl.LinkConfig, error) {
		return curl.LinkConfig{}, errors.New("long link-config failure")
	}

	var stdout, stderr strings.Builder
	exitCode := run(context.Background(), nil, &stdout, &stderr, deps)

	if exitCode != 0 {
		t.Fatalf("run() exit code = %d, want 0", exitCode)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
	for _, want := range []string{
		"go-curl-impersonate diagnostics",
		"Build\n",
		"Native requests",
		"unavailable",
		"Profiles\n",
		"chrome -> chrome116",
		"Chrome backend",
		"Firefox backend",
		"Native build configuration\n",
		"Build with -tags=\"integration native\"",
		"For local builds, set PKG_CONFIG_PATH or both CGO_CFLAGS and CGO_LDFLAGS",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Errorf("stdout does not contain %q:\n%s", want, stdout.String())
		}
	}
	for _, notWant := range []string{"long pkg-config failure", "long link-config failure"} {
		if strings.Contains(stdout.String(), notWant) {
			t.Errorf("default stdout unexpectedly contains verbose detail %q:\n%s", notWant, stdout.String())
		}
	}
}

func TestRunVerbosePrintsNativeConfigurationDetails(t *testing.T) {
	t.Parallel()

	deps := testDependencies()
	deps.probePkgConfig = func(context.Context) (curl.PkgConfigProbe, error) {
		return curl.PkgConfigProbe{Package: "libcurl-impersonate", CFlags: "-I/native/include", Libs: "-L/native/lib -lcurl"}, nil
	}
	deps.detectLinkConfig = func(_ context.Context, backend string) (curl.LinkConfig, error) {
		if backend == chromeBackend {
			return curl.LinkConfig{Source: "pkg-config", Package: "libcurl-impersonate-chrome", CFlags: "-I/chrome", LDFlags: "-lchrome"}, nil
		}
		return curl.LinkConfig{}, errors.New("firefox metadata missing\nset PKG_CONFIG_PATH")
	}

	var stdout, stderr strings.Builder
	exitCode := run(context.Background(), []string{"-verbose"}, &stdout, &stderr, deps)

	if exitCode != 0 {
		t.Fatalf("run() exit code = %d, want 0", exitCode)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
	for _, want := range []string{
		"libcurl-impersonate",
		"-I/native/include",
		"-L/native/lib -lcurl",
		"libcurl-impersonate-chrome",
		"-I/chrome",
		"-lchrome",
		"firefox metadata missing",
		"set PKG_CONFIG_PATH",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Errorf("verbose stdout does not contain %q:\n%s", want, stdout.String())
		}
	}
}

func TestRunPrintsRequestResult(t *testing.T) {
	t.Parallel()

	deps := testDependencies()
	deps.sendRequest = func(_ context.Context, requestURL, profileName string, tlsVerify bool) (requestResult, error) {
		if requestURL != "https://example.com" || profileName != "firefox" || tlsVerify {
			t.Fatalf("sendRequest arguments = (%q, %q, %v)", requestURL, profileName, tlsVerify)
		}
		return requestResult{Status: "204 No Content", Proto: "HTTP/2.0"}, nil
	}

	var stdout, stderr strings.Builder
	exitCode := run(context.Background(), []string{
		"-url", "https://example.com",
		"-profile", "firefox",
		"-tls-verify=false",
	}, &stdout, &stderr, deps)

	if exitCode != 0 {
		t.Fatalf("run() exit code = %d, want 0", exitCode)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
	for _, want := range []string{
		"Request\n",
		"firefox",
		"disabled",
		"204 No Content",
		"HTTP/2.0",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Errorf("stdout does not contain %q:\n%s", want, stdout.String())
		}
	}
}

func TestRunRequestFailureControlsExitStatus(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name     string
		args     []string
		wantExit int
	}{
		{name: "failure", args: []string{"-url", "https://example.com"}, wantExit: 1},
		{name: "allowed failure", args: []string{"-url", "https://example.com", "-allow-request-error"}, wantExit: 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			deps := testDependencies()
			deps.sendRequest = func(context.Context, string, string, bool) (requestResult, error) {
				return requestResult{}, errors.New("network unavailable")
			}

			var stdout, stderr strings.Builder
			exitCode := run(context.Background(), tc.args, &stdout, &stderr, deps)

			if exitCode != tc.wantExit {
				t.Fatalf("run() exit code = %d, want %d", exitCode, tc.wantExit)
			}
			if !strings.Contains(stderr.String(), "request failed: network unavailable") {
				t.Fatalf("stderr = %q, want request failure", stderr.String())
			}
		})
	}
}

func TestRunHelpAndUnexpectedArguments(t *testing.T) {
	t.Parallel()

	t.Run("help", func(t *testing.T) {
		var stdout, stderr strings.Builder
		exitCode := run(context.Background(), []string{"-help"}, &stdout, &stderr, testDependencies())

		if exitCode != 0 {
			t.Fatalf("run() exit code = %d, want 0", exitCode)
		}
		if stdout.Len() != 0 {
			t.Fatalf("stdout = %q, want empty", stdout.String())
		}
		for _, want := range []string{"Usage:", "Options:", "Examples:", "-verbose"} {
			if !strings.Contains(stderr.String(), want) {
				t.Errorf("help does not contain %q:\n%s", want, stderr.String())
			}
		}
	})

	t.Run("unexpected argument", func(t *testing.T) {
		var stdout, stderr strings.Builder
		exitCode := run(context.Background(), []string{"https://example.com"}, &stdout, &stderr, testDependencies())

		if exitCode != 2 {
			t.Fatalf("run() exit code = %d, want 2", exitCode)
		}
		if !strings.Contains(stderr.String(), "unexpected argument") || !strings.Contains(stderr.String(), "use -url") {
			t.Fatalf("stderr = %q, want actionable argument error", stderr.String())
		}
	})
}

func testDependencies() dependencies {
	return dependencies{
		nativeAvailable: func() bool { return true },
		supportedTargets: func() []string {
			return []string{"chrome116", "ff117", "safari15_5"}
		},
		probePkgConfig: func(context.Context) (curl.PkgConfigProbe, error) {
			return curl.PkgConfigProbe{Package: "libcurl-impersonate"}, nil
		},
		detectLinkConfig: func(_ context.Context, backend string) (curl.LinkConfig, error) {
			return curl.LinkConfig{Source: "pkg-config", Package: backend}, nil
		},
		sendRequest: func(context.Context, string, string, bool) (requestResult, error) {
			return requestResult{Status: "200 OK", Proto: "HTTP/2.0"}, nil
		},
	}
}
