#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 1 ]]; then
  echo "usage: scripts/release.sh vX.Y.Z" >&2
  exit 2
fi

version="$1"
if [[ ! "$version" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  echo "version must look like v0.1.2" >&2
  exit 2
fi

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

cd "$repo_root"

if [[ -n "$(git status --short --untracked-files=all)" ]]; then
  echo "working tree is not clean" >&2
  git status --short --untracked-files=all >&2
  exit 1
fi

go test ./...
go vet ./...
go build -o /tmp/ads ./cmd/ads
/tmp/ads --help >/dev/null
git ls-files cmd/ads/main.go | grep -qx "cmd/ads/main.go"

if git rev-parse "$version" >/dev/null 2>&1; then
  echo "tag $version already exists" >&2
  exit 1
fi

git tag "$version"
git push origin "$version"

echo "pushed $version"
echo "GitHub Actions will build macOS zip assets, publish the release, and update the Homebrew tap."
