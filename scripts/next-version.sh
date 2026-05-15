#!/usr/bin/env sh
set -eu

if [ "$#" -lt 1 ] || [ "$#" -gt 2 ]; then
  echo "usage: sh ./scripts/next-version.sh major|minor|patch [BASE_TAG]" >&2
  exit 2
fi

bump_type=$1
base_tag=${2:-}

case "$bump_type" in
  major | minor | patch) ;;
  *)
    echo "unsupported bump type: $bump_type" >&2
    exit 2
    ;;
esac

if [ -z "$base_tag" ]; then
  base_tag=$(git tag --list 'v*.*.*' --sort=-v:refname | grep -E '^v[0-9]+\.[0-9]+\.[0-9]+$' | head -n 1 || true)
fi
if [ -z "$base_tag" ]; then
  base_tag=v0.0.0
fi

version=${base_tag#v}
old_ifs=$IFS
IFS=.
set -- $version
IFS=$old_ifs

if [ "$#" -ne 3 ]; then
  echo "base tag must be vMAJOR.MINOR.PATCH, got: $base_tag" >&2
  exit 2
fi

major=$1
minor=$2
patch=$3

for part in "$major" "$minor" "$patch"; do
  case "$part" in
    '' | *[!0-9]*)
      echo "base tag must be vMAJOR.MINOR.PATCH, got: $base_tag" >&2
      exit 2
      ;;
  esac
done

case "$bump_type" in
  major)
    major=$((major + 1))
    minor=0
    patch=0
    ;;
  minor)
    minor=$((minor + 1))
    patch=0
    ;;
  patch)
    patch=$((patch + 1))
    ;;
esac

printf 'v%s.%s.%s\n' "$major" "$minor" "$patch"
