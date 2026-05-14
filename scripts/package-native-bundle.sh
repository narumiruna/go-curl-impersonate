#!/usr/bin/env sh
set -eu

if [ "$#" -lt 1 ] || [ "$#" -gt 2 ]; then
  echo "usage: sh ./scripts/package-native-bundle.sh PREFIX [OUT_DIR]" >&2
  exit 2
fi

prefix=$1
out_dir=${2:-dist}

for dir in include lib lib/pkgconfig; do
  if [ ! -d "$prefix/$dir" ]; then
    echo "missing native bundle input directory: $prefix/$dir" >&2
    exit 1
  fi
done

platform=${GO_CURL_IMPERSONATE_NATIVE_PLATFORM:-"linux-amd64"}
name=${GO_CURL_IMPERSONATE_NATIVE_BUNDLE_NAME:-"go-curl-impersonate-native-$platform"}
mkdir -p "$out_dir"
out_dir=$(cd "$out_dir" && pwd)
tmp_dir=$(mktemp -d "${TMPDIR:-/tmp}/go-curl-impersonate-bundle.XXXXXX")
trap 'rm -rf "$tmp_dir"' EXIT HUP INT TERM
stage="$tmp_dir/$name"
mkdir -p "$stage"

cp -a "$prefix/include" "$stage/"
cp -a "$prefix/lib" "$stage/"
if [ -d "$prefix/bin" ]; then
  cp -a "$prefix/bin" "$stage/"
fi

upstream_commit=unknown
if git -C third_party/curl-impersonate rev-parse HEAD >/dev/null 2>&1; then
  upstream_commit=$(git -C third_party/curl-impersonate rev-parse HEAD)
fi
repo_commit=unknown
if git rev-parse HEAD >/dev/null 2>&1; then
  repo_commit=$(git rev-parse HEAD)
fi

cat >"$stage/VERSION" <<EOF
name=$name
platform=$platform
go_curl_impersonate_commit=$repo_commit
curl_impersonate_commit=$upstream_commit
built_at_utc=$(date -u +%Y-%m-%dT%H:%M:%SZ)
EOF

(
  cd "$stage"
  find . -type f ! -name SHA256SUMS -print | sort | xargs sha256sum > SHA256SUMS
)

tarball="$out_dir/$name.tar.gz"
tar -czf "$tarball" -C "$tmp_dir" "$name"
sha256sum "$tarball" > "$tarball.sha256"
echo "$tarball"
