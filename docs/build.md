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

## Reference Source Layout

`third_party/curl-impersonate` is a pinned upstream submodule used by
contributors and GitHub Actions to build native artifacts and read fingerprint
fixtures. `references/curl_cffi` is an optional ignored local-only reference for
Python `curl_cffi` API and behavior comparison; it is not required by CI.

The primary native linking strategy is a local prefix containing headers,
libraries, and pkg-config metadata. Build that prefix from the pinned upstream
submodule with:

```sh
PREFIX=/tmp/curl-impersonate-local
sh ./scripts/build-curl-impersonate.sh "$PREFIX"
sh ./scripts/check-native.sh "$PREFIX"
```

`scripts/build-curl-impersonate.sh` configures
`third_party/curl-impersonate`, builds the Chrome and Firefox backends, installs
them under the prefix, and writes pkg-config metadata for the libraries that
actually exist. The generated `.pc` files are relocatable via `pcfiledir`, so
they keep working after a native bundle is unpacked somewhere else. The build
script installs `gyp-next` into a temporary prefix if the Firefox build needs
`gyp`, verifies that the `gyp` command is actually executable, and defaults to
`/usr/bin/python3` when available so local Python version managers do not leak
into the upstream NSS build. Override that with `GO_CURL_IMPERSONATE_PYTHON`
when needed. The helper intentionally invokes upstream `chrome-build` and
`firefox-build` serially because the top-level curl-impersonate Makefile is not
safe to parallelize with `make -j`.

`scripts/check-native.sh` accepts either `PREFIX`, `PKG_CONFIG_PATH`, or
explicit `CGO_CFLAGS` / `CGO_LDFLAGS`. Passing a prefix also adds
`$PREFIX/lib` to `LD_LIBRARY_PATH` for runtime checks. Direct consumer builds
still need `CGO_CFLAGS` and `CGO_LDFLAGS`; derive them from the selected backend
with `pkg-config --cflags --libs libcurl-impersonate-chrome` or
`libcurl-impersonate-ff`.

You can check the current pkg-config state with:

```sh
pkg-config --list-all | rg 'curl|impersonate'
pkg-config --cflags --libs libcurl-impersonate
pkg-config --cflags --libs libcurl-impersonate-chrome
pkg-config --cflags --libs libcurl-impersonate-ff
go run ./cmd/go-curl-impersonate
sh ./scripts/check-native.sh /tmp/curl-impersonate-local
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
go test -tags="integration native" ./...
```

Those paths must point to real headers and libraries; the Go toolchain passes
`CGO_LDFLAGS` to the linker during `go run` and `go test`.
`scripts/check-native.sh` validates explicit env flags before invoking the Go
toolchain so missing headers/libraries fail with a focused message.

If the headers and libraries are installed under a local prefix but `.pc` files
are missing, generate metadata:

```sh
sh ./scripts/write-pkg-config.sh /path/to/curl-impersonate
sh ./scripts/check-native.sh /path/to/curl-impersonate
```

The prefix must contain `include/` and `lib/` directories with the real
curl-impersonate artifacts. The helper refuses to write `.pc` files for an empty
prefix because that would make `pkg-config` pass while the linker still fails.

`scripts/check-native.sh` validates that pkg-config metadata points to
`curl/curl.h` and matching `lib*.so` or `lib*.a` files, compiles and links a
minimal C probe against `curl_easy_impersonate`, then runs the Go cgo backend
with Chrome and Firefox profiles when those backend packages are available.

## Integration Commands

Native integration tests run separately from default unit tests:

```sh
go test -tags="integration native" ./...
```

The `integration native` tag combination selects the cgo backend in
`internal/curl/perform_native.go` and requires `CGO_CFLAGS` / `CGO_LDFLAGS` to
point at linkable curl-impersonate headers and libraries. Every other tag/cgo
combination uses the same no-native implementation.

`scripts/check-native.sh` validates the selected native artifacts, then runs:

```sh
go run -tags="integration native" ./cmd/go-curl-impersonate
go test -tags="integration native" ./...
```

The native prefix can be packaged for users with:

```sh
sh ./scripts/package-native-bundle.sh /tmp/curl-impersonate-local dist
```

This writes `dist/go-curl-impersonate-native-linux-amd64.tar.gz` plus a
matching `.sha256` file. See `docs/native-distribution.md` for the consumer
distribution decision.

The same native gate was also verified from a clean environment that only kept
the required tool/runtime paths and native metadata:

```sh
env -i \
  PATH=/home/narumi/.local/go/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/home/narumi/.local/bin \
  HOME=/home/narumi \
  PREFIX=/tmp/curl-impersonate-local \
  GOCACHE=/tmp/go-build \
  sh ./scripts/check-native.sh /tmp/curl-impersonate-local
```

Fingerprint verification uses the same native build inputs plus Python/PyYAML
and `nghttpd`:

```sh
sudo apt install python3-yaml nghttp2-server

PKG_CONFIG_PATH=/tmp/curl-impersonate-local/lib/pkgconfig \
LD_LIBRARY_PATH=/tmp/curl-impersonate-local/lib \
GOCACHE=/tmp/go-build \
/usr/bin/python3 scripts/check-fingerprint.py --profile chrome

PKG_CONFIG_PATH=/tmp/curl-impersonate-local/lib/pkgconfig \
LD_LIBRARY_PATH=/tmp/curl-impersonate-local/lib \
GOCACHE=/tmp/go-build \
/usr/bin/python3 scripts/check-fingerprint.py --profile firefox
```

The Chrome and Firefox commands verify TLS ClientHello and HTTP/2 header
ordering against upstream fixtures. TLS capture intentionally keeps TLS
verification enabled and allows the later request failure; Firefox/NSS changes
ClientHello shape when verification is disabled.

## CI

Default CI runs without native artifacts or initialized submodules:

```sh
sh ./scripts/check-fingerprint-fixtures.sh
go test ./...
go test -race ./...
```

The reusable native pipeline is defined in
`.github/workflows/native-build.yml`. It checks out submodules, installs apt
dependencies, builds the native prefix with
`scripts/build-curl-impersonate.sh`, runs `scripts/check-native.sh`, verifies
Chrome and Firefox TLS/HTTP2 fingerprints, runs the external module smoke test,
and uploads the Linux amd64 native bundle as a workflow artifact. A separate
job downloads the artifact, verifies both checksums and required contents, and
runs the external-module smoke test against the unpacked bundle.
`.github/workflows/native.yml` calls it for manual runs, relevant `main`
changes, and `v*` tags other than strict SemVer release tags. The release workflow
calls the same pipeline once for SemVer tags, using the tagged module in the
consumer smoke test before publication. Pull requests keep using the default
no-native workflow unless a native run is started manually.

| Event | Native pipeline caller |
| --- | --- |
| Manual native dispatch | `native.yml` |
| Relevant `main` push | `native.yml` |
| `vMAJOR.MINOR.PATCH` tag | `release.yml` only |
| Other `v*` tag | `native.yml`; release rejects malformed release tags |
| Pull request | Default checks only; native can be dispatched manually |
