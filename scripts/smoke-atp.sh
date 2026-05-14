#!/usr/bin/env sh
set -eu

sh ./scripts/check-native.sh

if pkg-config --exists libcurl-impersonate; then
  cflags=$(pkg-config --cflags libcurl-impersonate)
  libs=$(pkg-config --libs libcurl-impersonate)
elif pkg-config --exists libcurl-impersonate-chrome; then
  cflags=$(pkg-config --cflags libcurl-impersonate-chrome)
  libs=$(pkg-config --libs libcurl-impersonate-chrome)
elif [ -n "${CGO_CFLAGS:-}" ] && [ -n "${CGO_LDFLAGS:-}" ]; then
  cflags=$CGO_CFLAGS
  libs=$CGO_LDFLAGS
else
  echo "no Chrome-capable curl-impersonate native flags found" >&2
  exit 1
fi

CGO_CFLAGS=$cflags CGO_LDFLAGS=$libs go run -tags="integration native" ./examples/basic
