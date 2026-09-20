package curl

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

var ErrPkgConfigUnavailable = errors.New("curl: pkg-config metadata for curl-impersonate is unavailable")

// PkgConfigProbe is the result of probing native dependency metadata.
type PkgConfigProbe struct {
	Package string
	CFlags  string
	Libs    string
}

// BackendPkgConfigPackage returns the preferred pkg-config package for a
// curl-impersonate backend family.
func BackendPkgConfigPackage(backend string) (string, error) {
	switch backend {
	case "curl-impersonate-chrome":
		return "libcurl-impersonate-chrome", nil
	case "curl-impersonate-ff":
		return "libcurl-impersonate-ff", nil
	default:
		return "", fmt.Errorf("curl: unsupported curl-impersonate backend %q", backend)
	}
}

// ProbeBackendPkgConfig checks metadata for one curl-impersonate backend
// family, falling back to the generic libcurl-impersonate package.
func ProbeBackendPkgConfig(ctx context.Context, backend string) (PkgConfigProbe, error) {
	backendPackage, err := BackendPkgConfigPackage(backend)
	if err != nil {
		return PkgConfigProbe{}, err
	}
	probe, err := probePkgConfigCandidates(ctx, backendPackage, "libcurl-impersonate")
	if err != nil {
		return PkgConfigProbe{}, fmt.Errorf("%w for %s: %v", ErrPkgConfigUnavailable, backend, err)
	}
	return probe, nil
}

// ProbePkgConfig looks for a usable curl-impersonate pkg-config package.
func ProbePkgConfig(ctx context.Context) (PkgConfigProbe, error) {
	probe, err := probePkgConfigCandidates(ctx,
		"libcurl-impersonate",
		"libcurl-impersonate-chrome",
		"libcurl-impersonate-ff",
	)
	if err != nil {
		return PkgConfigProbe{}, fmt.Errorf("%w: %v", ErrPkgConfigUnavailable, err)
	}
	return probe, nil
}

func probePkgConfigCandidates(ctx context.Context, candidates ...string) (PkgConfigProbe, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	var messages []string
	for _, candidate := range candidates {
		cflags, libs, err := pkgConfig(ctx, candidate)
		if err == nil {
			return PkgConfigProbe{Package: candidate, CFlags: cflags, Libs: libs}, nil
		}
		messages = append(messages, fmt.Sprintf("%s: %v", candidate, err))
	}
	return PkgConfigProbe{}, errors.New(strings.Join(messages, "; "))
}

func pkgConfig(ctx context.Context, pkg string) (string, string, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	output, err := exec.CommandContext(ctx, "pkg-config", "--cflags", "--libs", pkg).CombinedOutput()
	if err != nil {
		return "", "", compactOutput(output, err)
	}
	fields := strings.Fields(string(output))
	var cflags []string
	var libs []string
	for _, field := range fields {
		if strings.HasPrefix(field, "-I") || strings.HasPrefix(field, "-D") {
			cflags = append(cflags, field)
			continue
		}
		libs = append(libs, field)
	}
	return strings.Join(cflags, " "), strings.Join(libs, " "), nil
}

func compactOutput(output []byte, err error) error {
	output = bytes.TrimSpace(output)
	if len(output) == 0 {
		return err
	}
	return fmt.Errorf("%v: %s", err, output)
}
