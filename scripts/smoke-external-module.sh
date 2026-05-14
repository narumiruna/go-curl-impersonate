#!/usr/bin/env sh
set -eu

repo_path=$(pwd)
module_path=github.com/narumiruna/go-curl-impersonate
tmp_dir=${GO_CURL_IMPERSONATE_SMOKE_DIR:-$(mktemp -d "${TMPDIR:-/tmp}/go-curl-impersonate-consumer.XXXXXX")}
cleanup=0
if [ -z "${GO_CURL_IMPERSONATE_SMOKE_DIR:-}" ]; then
  cleanup=1
fi
trap '[ "$cleanup" -eq 0 ] || rm -rf "$tmp_dir"' EXIT HUP INT TERM

if [ "$#" -gt 1 ]; then
  echo "usage: sh ./scripts/smoke-external-module.sh [PREFIX]" >&2
  exit 2
fi

if [ "$#" -eq 1 ]; then
  PREFIX=$1
  PREFIX=$(cd "$PREFIX" && pwd)
  export PREFIX
fi

if [ -n "${PREFIX:-}" ]; then
  export PKG_CONFIG_PATH="$PREFIX/lib/pkgconfig${PKG_CONFIG_PATH:+:$PKG_CONFIG_PATH}"
  export LD_LIBRARY_PATH="$PREFIX/lib${LD_LIBRARY_PATH:+:$LD_LIBRARY_PATH}"
fi

if [ -z "${CGO_CFLAGS:-}" ] || [ -z "${CGO_LDFLAGS:-}" ]; then
  if ! command -v pkg-config >/dev/null 2>&1; then
    echo "missing pkg-config; set CGO_CFLAGS and CGO_LDFLAGS explicitly" >&2
    exit 1
  fi
  export CGO_CFLAGS=$(pkg-config --cflags libcurl-impersonate-chrome)
  export CGO_LDFLAGS=$(pkg-config --libs libcurl-impersonate-chrome)
fi

mkdir -p "$tmp_dir"
if [ -f "$tmp_dir/go.mod" ]; then
  echo "smoke directory already contains go.mod: $tmp_dir" >&2
  exit 1
fi
cd "$tmp_dir"
go mod init example.com/go-curl-impersonate-consumer
if [ -n "${GO_CURL_IMPERSONATE_MODULE_VERSION:-}" ]; then
  go get "$module_path@${GO_CURL_IMPERSONATE_MODULE_VERSION}"
else
  go mod edit -replace "$module_path=$repo_path"
  go mod edit -require "$module_path@v0.0.0"
fi

cat >main.go <<'EOF'
package main

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"

	"github.com/narumiruna/go-curl-impersonate/client"
)

func main() {
	if !client.NativeAvailable() {
		fmt.Fprintln(os.Stderr, "native backend unavailable")
		os.Exit(1)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-Consumer-Smoke"); got != "ok" {
			http.Error(w, "missing smoke header", http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("consumer smoke ok"))
	}))
	defer server.Close()

	c, err := client.NewClient(client.WithProfileName("chrome"))
	if err != nil {
		panic(err)
	}
	req, err := http.NewRequest(http.MethodGet, server.URL, nil)
	if err != nil {
		panic(err)
	}
	req.Header.Set("X-Consumer-Smoke", "ok")
	resp, err := c.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	if resp.StatusCode != http.StatusCreated || string(body) != "consumer smoke ok" {
		panic(fmt.Sprintf("unexpected response: %s %q", resp.Status, string(body)))
	}
	fmt.Printf("%s %s\n", resp.Status, string(body))
}
EOF

go mod tidy
go run -tags="integration native" .
