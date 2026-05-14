#!/usr/bin/env sh
set -eu

if [ "$#" -lt 1 ] || [ "$#" -gt 2 ]; then
  echo "usage: sh ./scripts/write-pkg-config.sh PREFIX [OUT_DIR]" >&2
  exit 2
fi

prefix=$1
out_dir=${2:-"$prefix/lib/pkgconfig"}

if [ ! -d "$prefix/include" ]; then
  echo "missing include directory: $prefix/include" >&2
  exit 1
fi

if [ ! -d "$prefix/lib" ]; then
  echo "missing lib directory: $prefix/lib" >&2
  exit 1
fi

if ! find "$prefix/lib" -maxdepth 1 -type f \( -name 'libcurl-impersonate*.so*' -o -name 'libcurl-impersonate*.a' \) | grep -q .; then
  echo "missing libcurl-impersonate library under: $prefix/lib" >&2
  exit 1
fi

mkdir -p "$out_dir"

write_pc() {
  name=$1
  lib=$2
  description=$3
  cat >"$out_dir/$name.pc" <<EOF
prefix=$prefix
exec_prefix=\${prefix}
libdir=\${exec_prefix}/lib
includedir=\${prefix}/include

Name: $name
Description: $description
Version: 0
Cflags: -I\${includedir}
Libs: -L\${libdir} -l$lib
EOF
}

write_pc "libcurl-impersonate" "curl-impersonate" "curl-impersonate generic libcurl"
write_pc "libcurl-impersonate-chrome" "curl-impersonate-chrome" "curl-impersonate Chrome backend"
write_pc "libcurl-impersonate-ff" "curl-impersonate-ff" "curl-impersonate Firefox backend"

echo "$out_dir"
