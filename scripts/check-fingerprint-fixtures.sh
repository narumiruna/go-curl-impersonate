#!/usr/bin/env sh
set -eu

required_files="
third_party/curl-impersonate/tests/signatures/chrome.yaml
third_party/curl-impersonate/tests/signatures/firefox.yaml
third_party/curl-impersonate/tests/test_impersonate.py
third_party/curl-impersonate/tests/signature.py
"

if [ ! -f "third_party/curl-impersonate/tests/signature.py" ]; then
  echo "curl-impersonate submodule is not initialized; skipping fixture check"
  exit 0
fi

for file in $required_files; do
  if [ ! -f "$file" ]; then
    echo "missing fingerprint fixture: $file" >&2
    exit 1
  fi
  echo "found fingerprint fixture: $file"
done
