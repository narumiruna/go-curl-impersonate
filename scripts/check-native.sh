#!/usr/bin/env sh
set -eu

if [ "$#" -gt 1 ]; then
  echo "usage: sh ./scripts/check-native.sh [PREFIX]" >&2
  exit 2
fi

if [ "$#" -eq 1 ]; then
  PREFIX=$1
  export PREFIX
fi

if [ -n "${PREFIX:-}" ]; then
  PREFIX=$(cd "$PREFIX" && pwd)
  if [ ! -d "$PREFIX/lib/pkgconfig" ]; then
    echo "missing pkg-config directory under PREFIX: $PREFIX/lib/pkgconfig" >&2
    exit 1
  fi
  PKG_CONFIG_PATH="$PREFIX/lib/pkgconfig${PKG_CONFIG_PATH:+:$PKG_CONFIG_PATH}"
  LD_LIBRARY_PATH="$PREFIX/lib${LD_LIBRARY_PATH:+:$LD_LIBRARY_PATH}"
  export PKG_CONFIG_PATH LD_LIBRARY_PATH
fi

packages="libcurl-impersonate libcurl-impersonate-chrome libcurl-impersonate-ff"

validate_flags() {
  package=$1
  cflags=$2
  libs=$3
  header_found=0
  lib_found=0

  for flag in $cflags; do
    case "$flag" in
      -I*)
        include_dir=${flag#-I}
        if [ -f "$include_dir/curl/curl.h" ]; then
          header_found=1
        fi
        ;;
    esac
  done

  lib_dirs=""
  lib_names=""
  for flag in $libs; do
    case "$flag" in
      -L*) lib_dirs="$lib_dirs ${flag#-L}" ;;
      -l*) lib_names="$lib_names ${flag#-l}" ;;
    esac
  done

  for lib_dir in $lib_dirs; do
    for lib_name in $lib_names; do
      if [ -f "$lib_dir/lib$lib_name.so" ] || [ -f "$lib_dir/lib$lib_name.a" ] || find "$lib_dir" -maxdepth 1 -type f -name "lib$lib_name.so.*" 2>/dev/null | grep -q .; then
        lib_found=1
      fi
    done
  done

  if [ "$header_found" -eq 0 ]; then
    echo "$package does not point to an include directory containing curl/curl.h" >&2
    return 1
  fi
  if [ "$lib_found" -eq 0 ]; then
    echo "$package does not point to a matching lib*.so or lib*.a" >&2
    return 1
  fi
}

validate_symbol() {
  package=$1
  cflags=$2
  libs=$3
  cc=${CC:-cc}
  tmp_dir=$(mktemp -d "${TMPDIR:-/tmp}/go-curl-impersonate-native.XXXXXX")
  trap 'rm -rf "$tmp_dir"' EXIT HUP INT TERM

  cat >"$tmp_dir/probe.c" <<'EOF'
#include <stddef.h>
#include <curl/curl.h>

extern CURLcode curl_easy_impersonate(CURL *data, const char *target, int default_headers);

int main(void) {
  return (int)curl_easy_impersonate(NULL, "chrome116", 1);
}
EOF

  if ! $cc $cflags "$tmp_dir/probe.c" $libs -o "$tmp_dir/probe" >"$tmp_dir/stdout" 2>"$tmp_dir/stderr"; then
    echo "$package cannot compile/link curl_easy_impersonate with ${cc}" >&2
    cat "$tmp_dir/stderr" >&2
    return 1
  fi

  rm -rf "$tmp_dir"
  trap - EXIT HUP INT TERM
}

pkg_config_available=0
if command -v pkg-config >/dev/null 2>&1; then
  pkg_config_available=1
elif [ -z "${CGO_CFLAGS:-}" ] || [ -z "${CGO_LDFLAGS:-}" ]; then
  echo "missing pkg-config" >&2
  echo "install pkg-config, pass PREFIX, or set both CGO_CFLAGS and CGO_LDFLAGS explicitly" >&2
  exit 1
fi

found=0
generic_cflags=""
generic_libs=""
chrome_cflags=""
chrome_libs=""
firefox_cflags=""
firefox_libs=""
if [ "$pkg_config_available" -eq 1 ]; then
  for package in $packages; do
    if pkg-config --exists "$package"; then
      found=1
      echo "found pkg-config package: $package"
      cflags=$(pkg-config --cflags "$package")
      libs=$(pkg-config --libs "$package")
      echo "$cflags $libs"
      validate_flags "$package" "$cflags" "$libs"
      validate_symbol "$package" "$cflags" "$libs"
      case "$package" in
        libcurl-impersonate)
          generic_cflags=$cflags
          generic_libs=$libs
          ;;
        libcurl-impersonate-chrome)
          chrome_cflags=$cflags
          chrome_libs=$libs
          ;;
        libcurl-impersonate-ff)
          firefox_cflags=$cflags
          firefox_libs=$libs
          ;;
      esac
    else
      echo "missing pkg-config package: $package" >&2
    fi
  done
fi

if [ "$found" -eq 0 ]; then
  if [ -z "${CGO_CFLAGS:-}" ] || [ -z "${CGO_LDFLAGS:-}" ]; then
    echo "no curl-impersonate pkg-config metadata found" >&2
    echo "set PKG_CONFIG_PATH to the directory containing libcurl-impersonate*.pc" >&2
    echo "or set both CGO_CFLAGS and CGO_LDFLAGS explicitly" >&2
    exit 1
  fi
  echo "using explicit CGO_CFLAGS/CGO_LDFLAGS"
  validate_flags "CGO_CFLAGS/CGO_LDFLAGS" "$CGO_CFLAGS" "$CGO_LDFLAGS"
  validate_symbol "CGO_CFLAGS/CGO_LDFLAGS" "$CGO_CFLAGS" "$CGO_LDFLAGS"
  generic_cflags=$CGO_CFLAGS
  generic_libs=$CGO_LDFLAGS
fi

run_native_go() {
  profile=$1
  cflags=$2
  libs=$3
  echo "running native Go checks for profile: $profile"
  GO_CURL_IMPERSONATE_TEST_PROFILE=$profile CGO_CFLAGS=$cflags CGO_LDFLAGS=$libs go run -tags="integration native" ./cmd/go-curl-impersonate
  GO_CURL_IMPERSONATE_TEST_PROFILE=$profile CGO_CFLAGS=$cflags CGO_LDFLAGS=$libs go test -tags="integration native" ./...
}

if [ -n "$generic_cflags" ] || [ -n "$generic_libs" ]; then
  run_native_go chrome "$generic_cflags" "$generic_libs"
fi
if [ -n "$chrome_cflags" ] || [ -n "$chrome_libs" ]; then
  run_native_go chrome "$chrome_cflags" "$chrome_libs"
fi
if [ -n "$firefox_cflags" ] || [ -n "$firefox_libs" ]; then
  run_native_go firefox "$firefox_cflags" "$firefox_libs"
fi
