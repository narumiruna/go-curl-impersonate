package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/narumiruna/go-curl-impersonate/client"
	"github.com/narumiruna/go-curl-impersonate/impersonate"
	"github.com/narumiruna/go-curl-impersonate/internal/curl"
)

func main() {
	requestURL := flag.String("url", "", "send one request to this URL after printing diagnostics")
	profileName := flag.String("profile", "chrome", "impersonation profile for -url")
	tlsVerify := flag.Bool("tls-verify", true, "verify TLS certificates for -url")
	allowRequestError := flag.Bool("allow-request-error", false, "return success when the diagnostic request fails")
	flag.Parse()

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
	if *requestURL != "" {
		if err := sendDiagnosticRequest(*requestURL, *profileName, *tlsVerify); err != nil {
			fmt.Fprintf(os.Stderr, "request failed: %v\n", err)
			if !*allowRequestError {
				os.Exit(1)
			}
		}
	}
}

func sendDiagnosticRequest(requestURL string, profileName string, tlsVerify bool) error {
	c, err := client.NewClient(
		client.WithProfileName(profileName),
		client.WithTLSVerify(tlsVerify),
	)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodGet, requestURL, nil)
	if err != nil {
		return err
	}
	resp, err := c.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	fmt.Printf("request status: %s\n", resp.Status)
	fmt.Printf("request proto: %s\n", resp.Proto)
	return nil
}
