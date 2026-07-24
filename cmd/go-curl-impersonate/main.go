package main

import (
	"context"
	"errors"
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

const (
	chromeBackend  = "curl-impersonate-chrome"
	firefoxBackend = "curl-impersonate-ff"
)

type dependencies struct {
	nativeAvailable  func() bool
	supportedTargets func() []string
	probePkgConfig   func(context.Context) (curl.PkgConfigProbe, error)
	detectLinkConfig func(context.Context, string) (curl.LinkConfig, error)
	sendRequest      func(context.Context, string, string, bool) (requestResult, error)
}

type options struct {
	requestURL      string
	profileName     string
	tlsVerify       bool
	allowRequestErr bool
	verbose         bool
}

type requestResult struct {
	Status string
	Proto  string
}

func main() {
	deps := dependencies{
		nativeAvailable:  client.NativeAvailable,
		supportedTargets: impersonate.SupportedTargets,
		probePkgConfig:   curl.ProbePkgConfig,
		detectLinkConfig: curl.DetectLinkConfig,
		sendRequest:      sendDiagnosticRequest,
	}
	os.Exit(run(context.Background(), os.Args[1:], os.Stdout, os.Stderr, deps))
}

func run(ctx context.Context, args []string, stdout, stderr io.Writer, deps dependencies) int {
	opts, exitCode, ok := parseOptions(args, stderr)
	if !ok {
		return exitCode
	}

	renderDiagnostics(ctx, stdout, deps, opts.verbose)
	if opts.requestURL == "" {
		return 0
	}

	fmt.Fprintln(stdout, "\nRequest")
	writeField(stdout, "Profile", opts.profileName)
	writeField(stdout, "TLS verification", enabledLabel(opts.tlsVerify))

	result, err := deps.sendRequest(ctx, opts.requestURL, opts.profileName, opts.tlsVerify)
	if err != nil {
		fmt.Fprintf(stderr, "request failed: %v\n", err)
		if !opts.allowRequestErr {
			return 1
		}
		return 0
	}
	writeField(stdout, "Status", result.Status)
	writeField(stdout, "Protocol", result.Proto)
	return 0
}

func parseOptions(args []string, stderr io.Writer) (options, int, bool) {
	var opts options
	flags := flag.NewFlagSet("go-curl-impersonate", flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.StringVar(&opts.requestURL, "url", "", "send a GET request after running diagnostics")
	flags.StringVar(&opts.profileName, "profile", "chrome", "impersonation profile for -url")
	flags.BoolVar(&opts.tlsVerify, "tls-verify", true, "verify TLS certificates for -url")
	flags.BoolVar(&opts.allowRequestErr, "allow-request-error", false, "exit successfully when the diagnostic request fails")
	flags.BoolVar(&opts.verbose, "verbose", false, "show compiler, linker, and probe error details")
	flags.Usage = func() {
		fmt.Fprintln(stderr, "Usage:")
		fmt.Fprintln(stderr, "  go-curl-impersonate [options]")
		fmt.Fprintln(stderr, "\nInspect browser profiles and native curl-impersonate configuration.")
		fmt.Fprintln(stderr, "\nOptions:")
		flags.PrintDefaults()
		fmt.Fprintln(stderr, "\nExamples:")
		fmt.Fprintln(stderr, "  go-curl-impersonate")
		fmt.Fprintln(stderr, "  go-curl-impersonate -verbose")
		fmt.Fprintln(stderr, "  go-curl-impersonate -profile firefox -url https://example.com")
	}

	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return options{}, 0, false
		}
		return options{}, 2, false
	}
	if flags.NArg() != 0 {
		fmt.Fprintf(stderr, "unexpected argument %q; use -url to send a request\n\n", flags.Arg(0))
		flags.Usage()
		return options{}, 2, false
	}
	return opts, 0, true
}

func renderDiagnostics(ctx context.Context, stdout io.Writer, deps dependencies, verbose bool) {
	fmt.Fprintln(stdout, "go-curl-impersonate diagnostics")

	fmt.Fprintln(stdout, "\nBuild")
	if deps.nativeAvailable() {
		writeField(stdout, "Native requests", "available")
	} else {
		writeField(stdout, "Native requests", "unavailable")
		writeField(stdout, "Next step", "Build with -tags=\"integration native\"; see the Native Runtime Setup in README.md.")
	}

	fmt.Fprintln(stdout, "\nProfiles")
	defaultChrome, err := impersonate.Resolve("chrome")
	if err == nil {
		writeField(stdout, "Default alias", "chrome -> "+defaultChrome.Target)
	}
	profiles := groupProfilesByBackend(deps.supportedTargets())
	writeField(stdout, "Chrome backend", valueOrNone(strings.Join(profiles[chromeBackend], ", ")))
	writeField(stdout, "Firefox backend", valueOrNone(strings.Join(profiles[firefoxBackend], ", ")))
	if len(profiles[""]) != 0 {
		writeField(stdout, "Other", strings.Join(profiles[""], ", "))
	}

	fmt.Fprintln(stdout, "\nNative build configuration")
	probe, probeErr := deps.probePkgConfig(ctx)
	if probeErr != nil {
		writeField(stdout, "pkg-config metadata", "unavailable")
		if verbose {
			writeField(stdout, "Metadata reason", probeErr.Error())
		}
	} else {
		writeField(stdout, "pkg-config metadata", "available ("+probe.Package+")")
		if verbose {
			writeField(stdout, "C flags", valueOrNone(probe.CFlags))
			writeField(stdout, "Libraries", valueOrNone(probe.Libs))
		}
	}

	backends := []struct {
		label string
		name  string
	}{
		{label: "Chrome backend", name: chromeBackend},
		{label: "Firefox backend", name: firefoxBackend},
	}
	missingLinkConfig := false
	for _, backend := range backends {
		config, err := deps.detectLinkConfig(ctx, backend.name)
		if err != nil {
			missingLinkConfig = true
			writeField(stdout, backend.label, "unavailable")
			if verbose {
				writeField(stdout, backend.label+" reason", err.Error())
			}
			continue
		}
		writeField(stdout, backend.label, linkConfigSummary(config))
		if verbose {
			writeLinkConfigDetails(stdout, backend.label, config)
		}
	}
	if missingLinkConfig {
		writeField(stdout, "Next step", "For local builds, set PKG_CONFIG_PATH or both CGO_CFLAGS and CGO_LDFLAGS; see docs/build.md.")
	}
}

func groupProfilesByBackend(targets []string) map[string][]string {
	profiles := map[string][]string{
		chromeBackend:  {},
		firefoxBackend: {},
		"":             {},
	}
	for _, target := range targets {
		profile, err := impersonate.Resolve(target)
		if err != nil {
			profiles[""] = append(profiles[""], target)
			continue
		}
		backend, err := profile.Backend()
		if err != nil || (backend != chromeBackend && backend != firefoxBackend) {
			profiles[""] = append(profiles[""], target)
			continue
		}
		profiles[backend] = append(profiles[backend], target)
	}
	return profiles
}

func linkConfigSummary(config curl.LinkConfig) string {
	summary := "available via " + valueOrNone(config.Source)
	if config.Package != "" {
		summary += " (" + config.Package + ")"
	}
	return summary
}

func writeLinkConfigDetails(w io.Writer, label string, config curl.LinkConfig) {
	prefix := strings.TrimSuffix(label, " backend")
	writeField(w, prefix+" C flags", valueOrNone(config.CFlags))
	writeField(w, prefix+" linker flags", valueOrNone(config.LDFlags))
}

func writeField(w io.Writer, label, value string) {
	lines := strings.Split(value, "\n")
	fmt.Fprintf(w, "  %-22s %s\n", label, lines[0])
	for _, line := range lines[1:] {
		fmt.Fprintf(w, "  %-22s %s\n", "", line)
	}
}

func enabledLabel(enabled bool) string {
	if enabled {
		return "enabled"
	}
	return "disabled"
}

func valueOrNone(value string) string {
	if strings.TrimSpace(value) == "" {
		return "(none)"
	}
	return value
}

func sendDiagnosticRequest(ctx context.Context, requestURL, profileName string, tlsVerify bool) (requestResult, error) {
	c, err := client.NewClient(
		client.WithProfileName(profileName),
		client.WithTLSVerify(tlsVerify),
	)
	if err != nil {
		return requestResult{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return requestResult{}, err
	}
	resp, err := c.Do(req)
	if err != nil {
		return requestResult{}, err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	return requestResult{Status: resp.Status, Proto: resp.Proto}, nil
}
