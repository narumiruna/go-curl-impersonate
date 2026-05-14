#!/usr/bin/env sh
set -eu

if [ "$#" -ne 1 ]; then
  echo "usage: sh ./scripts/build-curl-impersonate.sh PREFIX" >&2
  exit 2
fi

prefix=$1
source_dir=${CURL_IMPERSONATE_SOURCE_DIR:-third_party/curl-impersonate}
build_dir=${CURL_IMPERSONATE_BUILD_DIR:-"$source_dir/build"}
profiles=${GO_CURL_IMPERSONATE_BUILD_PROFILES:-"chrome firefox"}
python3_cmd=${GO_CURL_IMPERSONATE_PYTHON:-}

if [ -z "$python3_cmd" ]; then
  if [ -x /usr/bin/python3 ]; then
    python3_cmd=/usr/bin/python3
  elif command -v python3 >/dev/null 2>&1; then
    python3_cmd=$(command -v python3)
  fi
fi

if [ -n "$python3_cmd" ]; then
  python3_dir=$(dirname "$python3_cmd")
  PATH="$python3_dir:$PATH"
  export PATH
fi

if [ ! -f "$source_dir/configure" ]; then
  echo "missing curl-impersonate source: $source_dir" >&2
  echo "initialize submodules with: git submodule update --init --recursive" >&2
  exit 1
fi

mkdir -p "$prefix"
prefix=$(cd "$prefix" && pwd)
source_dir=$(cd "$source_dir" && pwd)
mkdir -p "$build_dir"
build_dir=$(cd "$build_dir" && pwd)

ensure_gyp_next() {
  if command -v gyp >/dev/null 2>&1 && gyp --help >/dev/null 2>&1; then
    return 0
  fi
  if [ -z "$python3_cmd" ]; then
    echo "missing python3; required to install gyp-next for Firefox builds" >&2
    exit 1
  fi
  if ! "$python3_cmd" -m pip --version >/dev/null 2>&1; then
    echo "missing python3 pip; required to install gyp-next for Firefox builds" >&2
    exit 1
  fi

  gyp_prefix=${GYP_NEXT_PREFIX:-"${RUNNER_TEMP:-${TMPDIR:-/tmp}}/go-curl-impersonate-gyp-next"}
  mkdir -p "$gyp_prefix"

  if [ -d "$gyp_prefix/lib" ]; then
    site_packages=$(find "$gyp_prefix/lib" -type d -path '*/site-packages' -print -quit 2>/dev/null || true)
    if [ -n "$site_packages" ]; then
      PATH="$gyp_prefix/bin:$PATH"
      PYTHONPATH="$site_packages${PYTHONPATH:+:$PYTHONPATH}"
      export PATH PYTHONPATH
      if command -v gyp >/dev/null 2>&1 && gyp --help >/dev/null 2>&1; then
        return 0
      fi
    fi
  fi

  if ! "$python3_cmd" -m pip install --ignore-installed --prefix "$gyp_prefix" gyp-next; then
    "$python3_cmd" -m pip install --break-system-packages --ignore-installed --prefix "$gyp_prefix" gyp-next
  fi
  PATH="$gyp_prefix/bin:$PATH"
  export PATH
  site_packages=$(find "$gyp_prefix/lib" -type d -path '*/site-packages' -print -quit 2>/dev/null || true)
  if [ -n "$site_packages" ]; then
    PYTHONPATH="$site_packages${PYTHONPATH:+:$PYTHONPATH}"
    export PYTHONPATH
  fi
  if ! command -v gyp >/dev/null 2>&1 || ! gyp --help >/dev/null 2>&1; then
    echo "gyp-next was installed but gyp is not executable" >&2
    exit 1
  fi
}

echo "configuring curl-impersonate"
(
  cd "$build_dir"
  "$source_dir/configure" --prefix="$prefix"
)

for profile in $profiles; do
  case "$profile" in
    chrome)
      echo "building Chrome curl-impersonate backend"
      make -C "$build_dir" chrome-build
      make -C "$build_dir" chrome-install
      ;;
    firefox|ff)
      ensure_gyp_next
      echo "building Firefox curl-impersonate backend"
      make -C "$build_dir" firefox-build
      make -C "$build_dir" firefox-install
      ;;
    *)
      echo "unknown build profile: $profile" >&2
      exit 2
      ;;
  esac
done

if [ ! -d "$build_dir/curl-8.1.1" ]; then
  echo "missing built curl source directory: $build_dir/curl-8.1.1" >&2
  exit 1
fi
make -C "$build_dir/curl-8.1.1" install-data MAKEFLAGS=
sh ./scripts/write-pkg-config.sh "$prefix"

echo "curl-impersonate artifacts installed under: $prefix"
echo "export PKG_CONFIG_PATH=$prefix/lib/pkgconfig\${PKG_CONFIG_PATH:+:\$PKG_CONFIG_PATH}"
echo "export LD_LIBRARY_PATH=$prefix/lib\${LD_LIBRARY_PATH:+:\$LD_LIBRARY_PATH}"
