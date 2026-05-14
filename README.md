# go-curl-impersonate

Go bindings and a high-level client API for `lwthiker/curl-impersonate`.

This repository is in early implementation. The checked-in Go module currently
contains the public package skeleton, profile resolution, client option model,
basic tests, and documentation for the native dependency work. Actual HTTP
requests through `libcurl-impersonate` are still behind the native integration
work described in `docs/plans/2026-05-14_go-curl-cffi-plan.md`.

## Goal

`go-curl-impersonate` should provide a Go package that sends HTTP requests
through `curl-impersonate`, preserving browser-like TLS and HTTP/2 fingerprints
without reimplementing those fingerprints in Go.

The first release target is Linux amd64 with cgo and a native
`curl-impersonate` runtime library.

## Current Status

- Go module exists at `github.com/narumiruna/go-curl-impersonate`.
- `impersonate` resolves browser aliases such as `chrome` and `firefox` to
  native curl-impersonate targets from the checked-in reference.
- `client` exposes `NewClient`, `Do`, and option helpers for profile, cookies,
  timeout, proxy, redirects, TLS verification, and HTTP/2 intent.
- Default builds do not link native libraries yet. `Do` returns a native backend
  unavailable error until the integration build is implemented.

## Packages

- `impersonate`: browser profile definitions, aliases, and backend family
  mapping.
- `client`: high-level request API intended to wrap curl-impersonate.
- `internal/curl`: low-level boundary for native libcurl/cgo work.
- `cmd/go-curl-impersonate`: diagnostic CLI for supported profiles and native
  backend availability.

## Example

```go
package main

import (
	"context"
	"net/http"
	"time"

	"github.com/narumiruna/go-curl-impersonate/client"
)

func main() {
	c, err := client.NewClient(
		client.WithProfileName("chrome"),
		client.WithTimeout(20*time.Second),
	)
	if err != nil {
		panic(err)
	}
	if !client.NativeAvailable() {
		return
	}

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "https://app.atptour.com/api/v2/gateway/livematches/website?scoringTournamentLevel=tour", nil)
	if err != nil {
		panic(err)
	}
	resp, err := c.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
}
```

The example API is stable enough for implementation work, but it will not make
a network request until the native curl-impersonate backend is enabled.

## Development

Ubuntu prerequisites for building or checking native artifacts:

```sh
sudo apt install build-essential pkg-config cmake ninja-build curl autoconf automake libtool python3-pip libnss3 nss-plugin-pem ca-certificates zlib1g-dev unzip
```

These packages provide the compiler/tooling and runtime dependencies. They do
not install `libcurl-impersonate`; `scripts/check-native.sh` still expects real
`libcurl-impersonate*.so` or `libcurl-impersonate*.a` artifacts, headers, and
pkg-config metadata or explicit `CGO_CFLAGS`/`CGO_LDFLAGS`.

Run the default checks:

```sh
go test ./...
go test -race ./...
```

Run the diagnostic CLI:

```sh
go run ./cmd/go-curl-impersonate
sh ./scripts/check-native.sh
sh ./scripts/smoke-atp.sh
```

`go test -tags=integration ./...` keeps using the no-native placeholder.
`sh ./scripts/check-native.sh` validates native artifacts and then runs the
real cgo backend with `go test -tags="integration native" ./...`.

Native dependency and integration-test details are tracked in:

- `docs/build.md`
- `docs/native-api.md`
- `docs/fingerprint-verification.md`
- `docs/api-scope.md`

If curl-impersonate is installed under a local prefix without pkg-config
metadata, `scripts/write-pkg-config.sh` can generate the `.pc` files needed by
`scripts/check-native.sh`.

## Concurrency

`client.Client` is intended to be shared by goroutines after construction. Its
configuration is immutable. The native backend must not share one libcurl easy
handle across concurrent requests; a future handle pool must lease one easy
handle per request for the full perform/reset/cleanup cycle.

## References

- `.refs/curl-impersonate`: upstream curl-impersonate reference clone.
- `.refs/curl_cffi`: Python curl_cffi reference clone.
- `docs/plans/2026-05-14_go-curl-cffi-plan.md`: implementation plan.
