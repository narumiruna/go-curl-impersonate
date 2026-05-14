package curl

import (
	"context"
	"fmt"
	"os"
	"strings"
)

type LinkConfig struct {
	Source  string
	Package string
	CFlags  string
	LDFlags string
}

func DetectLinkConfig(ctx context.Context, backend string) (LinkConfig, error) {
	probe, err := ProbeBackendPkgConfig(ctx, backend)
	if err == nil {
		return LinkConfig{
			Source:  "pkg-config",
			Package: probe.Package,
			CFlags:  probe.CFlags,
			LDFlags: probe.Libs,
		}, nil
	}
	envConfig, envErr := linkConfigFromEnv()
	if envErr == nil {
		return envConfig, nil
	}
	return LinkConfig{}, fmt.Errorf("%w; %v", err, envErr)
}

func linkConfigFromEnv() (LinkConfig, error) {
	cflags := strings.TrimSpace(os.Getenv("CGO_CFLAGS"))
	ldflags := strings.TrimSpace(os.Getenv("CGO_LDFLAGS"))
	if cflags == "" && ldflags == "" {
		return LinkConfig{}, fmt.Errorf("curl: CGO_CFLAGS and CGO_LDFLAGS are empty")
	}
	if cflags == "" {
		return LinkConfig{}, fmt.Errorf("curl: CGO_CFLAGS is empty")
	}
	if ldflags == "" {
		return LinkConfig{}, fmt.Errorf("curl: CGO_LDFLAGS is empty")
	}
	return LinkConfig{Source: "env", CFlags: cflags, LDFlags: ldflags}, nil
}
