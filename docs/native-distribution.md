# Native Distribution

This project is a Go library wrapper around `curl-impersonate`, so Go module
source distribution and native library distribution are separate problems.

## Decision

Phase 1 uses a Linux amd64 native release bundle built from the pinned
`third_party/curl-impersonate` submodule. The bundle contains:

- `include/`
- `lib/`
- `lib/pkgconfig/`
- `VERSION`
- `SHA256SUMS`
- optional `bin/` wrapper tools when upstream installs them

Consumers unpack the bundle, set `PKG_CONFIG_PATH` and `LD_LIBRARY_PATH`, then
build their Go code with `-tags="integration native"`.

This is intentionally not hidden behind git submodules. Submodules help
contributors and GitHub Actions reproduce native builds, but `go get` downloads
the Go module source and does not initialize this repository's submodules for
consumers.

## Options

| Option | Fit | Tradeoff |
| --- | --- | --- |
| Release native bundle | Selected for Phase 1 | Simple boundary, easy to inspect, works with pkg-config, but users must download and set env vars. |
| Platform-specific Go artifact module | Candidate for Phase 2 | Could make `go get` fetch native files, but needs size, Go proxy, license, and update-policy validation. |
| Runtime loader / embedded bundle | Candidate for Phase 2 | Closest to `curl_cffi` wheel ergonomics, but dynamic loading, extraction paths, callbacks, and security updates need careful design. |
| System pkg-config install | Supported fallback | Good for developers and distributions, but not a complete project-owned consumer path. |

## Why A Submodule Is Not Enough

`third_party/curl-impersonate` pins the upstream source for CI and contributor
builds. It does not solve consumer installation because:

- `go get` does not build native libraries.
- `go install` does not run post-install hooks.
- Go module archives do not initialize git submodules for downstream users.
- The native runtime still needs shared libraries discoverable by the dynamic
  loader.

## Build And Package Flow

```sh
PREFIX=/tmp/curl-impersonate-local
sh ./scripts/build-curl-impersonate.sh "$PREFIX"
sh ./scripts/check-native.sh "$PREFIX"
sh ./scripts/package-native-bundle.sh "$PREFIX" dist
```

The package script writes
`dist/go-curl-impersonate-native-linux-amd64.tar.gz` and a matching `.sha256`
file. The tarball includes its own `SHA256SUMS` manifest.

## Runtime Loader Prototype

`scripts/prototype-runtime-loader.sh` is an experiment for a future
`curl_cffi`-like path. It creates a temporary external Go module, `go get`s this
module, unsets `CGO_CFLAGS`, `CGO_LDFLAGS`, and `LD_LIBRARY_PATH`, then uses
`dlopen` to load `libcurl-impersonate-chrome.so` from the native bundle. The
prototype resolves the Chrome profile through the public `impersonate` package
and calls `curl_easy_impersonate` through symbols loaded at runtime.

```sh
sh ./scripts/prototype-runtime-loader.sh /tmp/curl-impersonate-local
```

Expected output:

```text
runtime loader prototype ok: chrome116
```

This is not the selected Phase 1 library path. The current `client` native
backend still compiles against `curl/curl.h` and links through cgo flags. The
prototype only proves that a runtime-loader design can locate the bundle and
call the core native symbol without compile-time curl-impersonate cgo flags.

## Follow-Up Criteria

Before replacing the Phase 1 bundle with a more `curl_cffi`-like path, prototype
one of these outcomes:

- A platform artifact module can be fetched with `go get`, stays within an
  acceptable module size, and exposes stable cgo flags without polluting the
  main module.
- A runtime loader can locate or extract the native libraries safely, works in a
  temporary external module, and has a clear update story for upstream security
  rebuilds.
