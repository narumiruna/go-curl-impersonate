# go-curl-impersonate 🦾

> Go bindings and a high-level HTTP client for [`lwthiker/curl-impersonate`](https://github.com/lwthiker/curl-impersonate) — send HTTP requests with **real browser TLS and HTTP/2 fingerprints** directly from Go.

When websites block automated requests by inspecting TLS handshakes or HTTP/2 settings, `go-curl-impersonate` lets your Go code appear as Chrome or Firefox at the network level — without reimplementing fingerprints yourself.

## ✨ Features

- 🌐 **Browser-accurate fingerprints** — Chrome and Firefox TLS/HTTP/2 profiles verified against upstream fixtures
- 🔌 **Familiar API** — wraps `*http.Request` / `*http.Response`; drop-in alongside the standard library
- ⚙️ **Flexible options** — profile, cookies, timeout, proxy, redirect policy, TLS verification, HTTP/2 intent
- 🪶 **Dependency-light default build** — compiles without cgo; native backend activated by a build tag
- 🔒 **Concurrency-safe** — `client.Client` is safe to share across goroutines after construction

## 📦 Installation

### 1. Add the Go module

```sh
go get github.com/narumiruna/go-curl-impersonate@latest
```

### 2. Install the native runtime bundle (Linux amd64)

The default build compiles without native libraries. To make real impersonated
requests you need a compatible `curl-impersonate` runtime. Release builds
publish a pre-built Linux amd64 bundle:

```sh
version=v0.1.0 # replace with the release tag you want
curl -LO "https://github.com/narumiruna/go-curl-impersonate/releases/download/${version}/go-curl-impersonate-native-linux-amd64.tar.gz"
tar -xzf go-curl-impersonate-native-linux-amd64.tar.gz
export GO_CURL_IMPERSONATE_NATIVE="$PWD/go-curl-impersonate-native-linux-amd64"
export PKG_CONFIG_PATH="$GO_CURL_IMPERSONATE_NATIVE/lib/pkgconfig${PKG_CONFIG_PATH:+:$PKG_CONFIG_PATH}"
export LD_LIBRARY_PATH="$GO_CURL_IMPERSONATE_NATIVE/lib${LD_LIBRARY_PATH:+:$LD_LIBRARY_PATH}"
```

### 3. Build with native tags

```sh
go run -tags="integration native" ./examples/basic/main.go
go install -tags="integration native" github.com/narumiruna/go-curl-impersonate/cmd/go-curl-impersonate@latest
```

### Diagnostic CLI

```sh
go install github.com/narumiruna/go-curl-impersonate/cmd/go-curl-impersonate@latest
go-curl-impersonate   # prints supported profiles and native backend availability
```

## 🚀 Quick Start

```go
package main

import (
	"context"
	"fmt"
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
		fmt.Println("native curl-impersonate backend is not available in this build")
		return
	}

	req, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodGet,
		"https://example.com",
		nil,
	)
	if err != nil {
		panic(err)
	}

	resp, err := c.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	fmt.Println(resp.Status)
}
```

> **Note:** `client.NativeAvailable()` returns `false` unless the binary was
> built with `-tags="integration native"` and the runtime library is in
> `LD_LIBRARY_PATH`. See the [Installation](#-installation) section above.

## 📚 API Overview

| Package | Purpose |
|---|---|
| `impersonate` | Browser profile definitions, aliases (`chrome`, `firefox`), and backend family mapping |
| `client` | High-level request API — `NewClient`, `Do`, and option helpers |
| `internal/curl` | Low-level cgo boundary for native libcurl work |
| `cmd/go-curl-impersonate` | Diagnostic CLI for supported profiles and native backend availability |

For full API documentation see [`docs/api-scope.md`](docs/api-scope.md) and
[`docs/native-api.md`](docs/native-api.md).

## 🛠️ Development

### Prerequisites (Ubuntu)

```sh
sudo apt install build-essential pkg-config cmake ninja-build curl \
  autoconf automake autotools-dev libtool python3-pip python3-yaml \
  libnss3 nss-plugin-pem ca-certificates zlib1g-dev bzip2 xz-utils \
  unzip mercurial nghttp2-server
```

> These packages provide the compiler toolchain and runtime dependencies.
> They do **not** install `libcurl-impersonate`; see
> [`docs/build.md`](docs/build.md) for the full native build guide.

### Run tests

```sh
go test ./...
go test -race ./...
```

### Run the diagnostic CLI and scripts

```sh
go run ./cmd/go-curl-impersonate
sh ./scripts/check-native.sh
sh ./scripts/smoke-atp.sh
/usr/bin/python3 scripts/check-fingerprint.py --profile chrome
```

### Local prefix builds

```sh
sh ./scripts/build-curl-impersonate.sh /tmp/curl-impersonate-local
sh ./scripts/check-native.sh /tmp/curl-impersonate-local
sh ./scripts/smoke-external-module.sh /tmp/curl-impersonate-local
```

If curl-impersonate is installed under a local prefix without pkg-config
metadata, `scripts/write-pkg-config.sh` can generate the `.pc` files needed
by `scripts/check-native.sh`.

### Documentation

- [`docs/build.md`](docs/build.md) — native build guide
- [`docs/quickstart.md`](docs/quickstart.md) — consumer quickstart
- [`docs/native-api.md`](docs/native-api.md) — native backend API
- [`docs/native-distribution.md`](docs/native-distribution.md) — bundle packaging
- [`docs/fingerprint-verification.md`](docs/fingerprint-verification.md) — fingerprint testing
- [`docs/api-scope.md`](docs/api-scope.md) — public API scope

## 🚢 Release

The **Bump Version** workflow creates an annotated `vMAJOR.MINOR.PATCH` tag
from the latest SemVer tag. The repository must define `secrets.PAT_TOKEN`
with tag-push permission because tags pushed with the default `GITHUB_TOKEN`
do not trigger follow-up workflows.

The **Release** workflow runs on `v*.*.*` tag pushes. It runs Go checks,
builds curl-impersonate, verifies native backend and fingerprints, validates
external module consumption, packages the Linux amd64 native bundle, and
uploads the bundle plus checksum to the GitHub Release.

## ⚡ Concurrency

`client.Client` is safe to share across goroutines after construction — its
configuration is immutable. The native backend does not share a single libcurl
easy handle across concurrent requests; a future handle pool will lease one
easy handle per request for the full perform/reset/cleanup cycle.

## 🗺️ Current Status

> This module is in early implementation. The public API is stable enough for
> integration work; full release readiness is tracked in
> `docs/plans/2026-05-15_github-actions-library-distribution-plan.md`.

- ✅ Go module at `github.com/narumiruna/go-curl-impersonate`
- ✅ Browser alias resolution (`chrome`, `firefox` → native targets)
- ✅ High-level client with profile, cookies, timeout, proxy, redirect, TLS, HTTP/2 options
- ✅ Chrome and Firefox TLS/HTTP/2 fingerprints verified against upstream fixtures
- ✅ Linux amd64 native bundle packaging wired into CI
- 🔲 Default build returns a "native backend unavailable" error without `-tags="integration native"`
- 🔲 `third_party/curl-impersonate` is a contributor/CI submodule — not a consumer install path

## 🔗 References

- [`third_party/curl-impersonate`](third_party/curl-impersonate) — pinned upstream curl-impersonate submodule
- [`docs/plans/2026-05-14_go-curl-cffi-plan.md`](docs/plans/2026-05-14_go-curl-cffi-plan.md) — implementation plan
- [lwthiker/curl-impersonate](https://github.com/lwthiker/curl-impersonate) — upstream curl-impersonate project
