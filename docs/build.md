# Build

## Default Development Build

The default build has no native dependency and is intended for ordinary unit
tests, docs, and API work:

```sh
go test ./...
go test -race ./...
go run ./cmd/go-curl-impersonate
```

In this mode `client.NativeAvailable()` returns `false`, and actual requests
return `curl.ErrNativeUnavailable`.

## Native Integration Target

The first native target is Linux amd64 with cgo and `curl-impersonate`.

The integration build still needs one finalized linking strategy. Acceptable
strategies are:

- `pkg-config` files for the selected curl-impersonate backend.
- Explicit `CGO_CFLAGS` and `CGO_LDFLAGS`.
- A repo-local build helper that installs headers and libraries into a known
  local directory.

You can check the current pkg-config state with:

```sh
pkg-config --list-all | rg 'curl|impersonate'
pkg-config --cflags --libs libcurl-impersonate
pkg-config --cflags --libs libcurl-impersonate-chrome
pkg-config --cflags --libs libcurl-impersonate-ff
go run ./cmd/go-curl-impersonate
sh ./scripts/check-native.sh
```

`pkg-config` itself is not enough; the search path must include a
`libcurl-impersonate*.pc` file that points to installed headers and libraries.
The diagnostic CLI checks both backend families separately:

- `curl-impersonate-chrome` prefers `libcurl-impersonate-chrome.pc`, with
  `libcurl-impersonate.pc` as a generic fallback.
- `curl-impersonate-ff` prefers `libcurl-impersonate-ff.pc`, with
  `libcurl-impersonate.pc` as a generic fallback.

If pkg-config metadata is not available, set both variables explicitly:

```sh
CGO_CFLAGS="-I/path/to/curl-impersonate/include" \
CGO_LDFLAGS="-L/path/to/curl-impersonate/lib -lcurl-impersonate" \
go test -tags=integration ./...
```

Those paths must point to real headers and libraries; the Go toolchain passes
`CGO_LDFLAGS` to the linker during `go run` and `go test`.
`scripts/check-native.sh` validates explicit env flags before invoking the Go
toolchain so missing headers/libraries fail with a focused message.

If the headers and libraries are installed under a local prefix but `.pc` files
are missing, generate repo-local metadata:

```sh
sh ./scripts/write-pkg-config.sh /path/to/curl-impersonate /tmp/curl-impersonate-pkgconfig
PKG_CONFIG_PATH=/tmp/curl-impersonate-pkgconfig sh ./scripts/check-native.sh
```

The prefix must contain `include/` and `lib/` directories with the real
curl-impersonate artifacts. The helper refuses to write `.pc` files for an empty
prefix because that would make `pkg-config` pass while the linker still fails.

`scripts/check-native.sh` validates that pkg-config metadata points to
`curl/curl.h` and matching `lib*.so` or `lib*.a` files, then compiles and links
a minimal C probe against `curl_easy_impersonate`. Until the cgo backend is
implemented, this proves the native symbol is linkable but still is not proof
that full impersonated requests work through the Go API.

## Proposed Integration Command

Once native linking exists, integration tests should run separately from default
unit tests:

```sh
go test -tags=integration ./...
go test -tags="integration native" ./...
```

The integration test command must be allowed to fail fast with a clear missing
dependency message when `curl-impersonate` is not installed.

The current `integration` build tag alone is a compiling placeholder: it
preserves the package boundary and still reports `curl.ErrNativeUnavailable`.
The `integration native` tag combination selects the cgo backend in
`internal/curl/perform_native.go` and requires `CGO_CFLAGS` / `CGO_LDFLAGS` to
point at linkable curl-impersonate headers and libraries.

`scripts/check-native.sh` validates the selected native artifacts, then runs:

```sh
go run -tags="integration native" ./cmd/go-curl-impersonate
go test -tags="integration native" ./...
```

A local Chrome and Firefox backend has been verified with:

```sh
cd .refs/curl-impersonate/build
../configure --prefix=/tmp/curl-impersonate-local
make chrome-build
make chrome-install
python3 -m pip install --prefix /tmp/gyp-next-prefix gyp-next
PATH="/tmp/gyp-next-prefix/bin:$PATH" \
PYTHONPATH=/tmp/gyp-next-prefix/lib/python3.14/site-packages \
make firefox-build
make firefox-install
cd curl-8.1.1
make install
cd ../../../..
PKG_CONFIG_PATH=/tmp/curl-impersonate-local/lib/pkgconfig \
LD_LIBRARY_PATH=/tmp/curl-impersonate-local/lib \
GOCACHE=/tmp/go-build \
sh ./scripts/check-native.sh
```

This validates both backend libraries, then runs the Go cgo backend once with
the Chrome library/profile and once with the Firefox library/profile.

Fingerprint verification uses the same native build inputs plus Python/PyYAML
and `nghttpd`:

```sh
sudo apt install python3-yaml nghttp2-server

PKG_CONFIG_PATH=/tmp/curl-impersonate-local/lib/pkgconfig \
GOCACHE=/tmp/go-build \
/usr/bin/python3 scripts/check-fingerprint.py --profile chrome

PKG_CONFIG_PATH=/tmp/curl-impersonate-local/lib/pkgconfig \
GOCACHE=/tmp/go-build \
/usr/bin/python3 scripts/check-fingerprint.py --profile firefox --skip-tls
```

The Chrome command verifies both TLS ClientHello and HTTP/2 header ordering
against upstream fixtures. The Firefox command currently verifies HTTP/2 only;
Firefox TLS still has a local `psk_key_exchange_modes` fixture mismatch tracked
in `docs/fingerprint-verification.md`.

## CI Plan

Default CI should run:

```sh
go test ./...
go test -race ./...
```

Native integration CI should be added only after the install/linking path is
reproducible on a clean runner.
