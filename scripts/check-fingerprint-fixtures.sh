#!/usr/bin/env sh
set -eu

required_files="
.refs/curl-impersonate/tests/signatures/chrome.yaml
.refs/curl-impersonate/tests/signatures/firefox.yaml
.refs/curl-impersonate/tests/test_impersonate.py
.refs/curl-impersonate/tests/signature.py
"

for file in $required_files; do
  if [ ! -f "$file" ]; then
    echo "missing fingerprint fixture: $file" >&2
    exit 1
  fi
  echo "found fingerprint fixture: $file"
done
