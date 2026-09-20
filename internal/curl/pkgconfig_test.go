package curl

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
)

func TestProbePkgConfigCandidates(t *testing.T) {
	for _, test := range []struct {
		name       string
		backend    string
		candidates []string
	}{
		{"general", "", []string{"libcurl-impersonate", "libcurl-impersonate-chrome", "libcurl-impersonate-ff"}},
		{"chrome", "curl-impersonate-chrome", []string{"libcurl-impersonate-chrome", "libcurl-impersonate"}},
		{"firefox", "curl-impersonate-ff", []string{"libcurl-impersonate-ff", "libcurl-impersonate"}},
	} {
		// All candidates at or after firstAvailable succeed. The probe must stop
		// at the first one; len(candidates) exercises complete failure.
		for firstAvailable := 0; firstAvailable <= len(test.candidates); firstAvailable++ {
			t.Run(fmt.Sprintf("%s/first-available-%d", test.name, firstAvailable), func(t *testing.T) {
				calls := fakePkgConfig(t, strings.Join(test.candidates[firstAvailable:], " "))
				var probe PkgConfigProbe
				var err error
				// Both entry points accept a nil context.
				if test.backend == "" {
					probe, err = ProbePkgConfig(nil)
				} else {
					probe, err = ProbeBackendPkgConfig(nil, test.backend)
				}
				wantCalls := test.candidates
				if firstAvailable < len(test.candidates) {
					wantCalls = test.candidates[:firstAvailable+1]
					want := PkgConfigProbe{
						Package: test.candidates[firstAvailable],
						CFlags:  "-I/native/include -DTEST=1",
						Libs:    "-L/native/lib -ltest -pthread",
					}
					if err != nil || probe != want {
						t.Fatalf("probe = %+v, %v; want %+v, nil", probe, err, want)
					}
				} else {
					prefix := ErrPkgConfigUnavailable.Error()
					if test.backend != "" {
						prefix += " for " + test.backend
					}
					var messages []string
					for _, candidate := range test.candidates {
						messages = append(messages, fmt.Sprintf("%s: exit status 1: missing %s", candidate, candidate))
					}
					wantErr := prefix + ": " + strings.Join(messages, "; ")
					if !errors.Is(err, ErrPkgConfigUnavailable) || err.Error() != wantErr {
						t.Fatalf("error = %v, want sentinel and %q", err, wantErr)
					}
					if probe != (PkgConfigProbe{}) {
						t.Fatalf("failed probe = %+v, want zero value", probe)
					}
				}
				if got := calls(); !slices.Equal(got, wantCalls) {
					t.Fatalf("pkg-config calls = %v, want %v", got, wantCalls)
				}
			})
		}
	}
}

func TestProbeBackendPkgConfigRejectsUnsupportedBackend(t *testing.T) {
	calls := fakePkgConfig(t, "libcurl-impersonate")
	probe, err := ProbeBackendPkgConfig(context.Background(), "unsupported")
	const wantErr = `curl: unsupported curl-impersonate backend "unsupported"`
	if err == nil || err.Error() != wantErr || errors.Is(err, ErrPkgConfigUnavailable) {
		t.Fatalf("error = %v, want %q without metadata sentinel", err, wantErr)
	}
	if probe != (PkgConfigProbe{}) || len(calls()) != 0 {
		t.Fatalf("unsupported backend should not probe packages: %+v, %v", probe, calls())
	}
}

func TestProbePkgConfigCanceledContext(t *testing.T) {
	calls := fakePkgConfig(t, "libcurl-impersonate")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	probe, err := ProbePkgConfig(ctx)
	wantErr := ErrPkgConfigUnavailable.Error() + ": libcurl-impersonate: context canceled; " +
		"libcurl-impersonate-chrome: context canceled; libcurl-impersonate-ff: context canceled"
	if !errors.Is(err, ErrPkgConfigUnavailable) || err.Error() != wantErr {
		t.Fatalf("error = %v, want %q", err, wantErr)
	}
	if probe != (PkgConfigProbe{}) || len(calls()) != 0 {
		t.Fatalf("canceled probe should not execute pkg-config: %+v, %v", probe, calls())
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

// fakePkgConfig isolates subprocess probing from installed metadata and records
// every attempted package. Tests using it must not run in parallel.
func fakePkgConfig(t *testing.T, available string) func() []string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("fake pkg-config requires a POSIX executable script")
	}
	dir := t.TempDir()
	log := filepath.Join(dir, "calls")
	const script = `#!/bin/sh
if [ "$#" -ne 3 ] || [ "$1" != "--cflags" ] || [ "$2" != "--libs" ]; then
  echo "unexpected pkg-config arguments" >&2
  exit 2
fi
printf '%s\n' "$3" >> "$FAKE_PKG_CONFIG_CALLS"
case " $FAKE_PKG_CONFIG_PACKAGES " in
  *" $3 "*) printf ' -L/native/lib -I/native/include -ltest -DTEST=1 -pthread \n'; exit 0 ;;
esac
printf 'missing %s\n' "$3" >&2
exit 1
`
	if err := os.WriteFile(filepath.Join(dir, "pkg-config"), []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	t.Setenv("FAKE_PKG_CONFIG_CALLS", log)
	t.Setenv("FAKE_PKG_CONFIG_PACKAGES", available)
	return func() []string {
		t.Helper()
		data, err := os.ReadFile(log)
		if err != nil && !os.IsNotExist(err) {
			t.Fatal(err)
		}
		return strings.Fields(string(data))
	}
}
