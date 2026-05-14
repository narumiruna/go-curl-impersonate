package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/narumiruna/go-curl-impersonate/client"
	"github.com/narumiruna/go-curl-impersonate/impersonate"
	"github.com/narumiruna/go-curl-impersonate/internal/curl"
)

func main() {
	fmt.Printf("native backend available: %v\n", client.NativeAvailable())
	fmt.Printf("supported targets: %s\n", strings.Join(impersonate.SupportedTargets(), ", "))
	probe, err := curl.ProbePkgConfig(context.Background())
	if err != nil {
		fmt.Fprintf(os.Stderr, "pkg-config probe: %v\n", err)
	} else {
		fmt.Printf("pkg-config package: %s\n", probe.Package)
		fmt.Printf("pkg-config cflags: %s\n", probe.CFlags)
		fmt.Printf("pkg-config libs: %s\n", probe.Libs)
	}
	for _, backend := range []string{"curl-impersonate-chrome", "curl-impersonate-ff"} {
		probe, err := curl.ProbeBackendPkgConfig(context.Background(), backend)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s pkg-config probe: %v\n", backend, err)
		} else {
			fmt.Printf("%s pkg-config package: %s\n", backend, probe.Package)
		}
		config, err := curl.DetectLinkConfig(context.Background(), backend)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s link config: %v\n", backend, err)
			continue
		}
		fmt.Printf("%s link source: %s\n", backend, config.Source)
	}
	if !client.NativeAvailable() {
		fmt.Fprintln(os.Stderr, "requests require a build with curl-impersonate integration enabled")
	}
}
