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
brew_repo="$(brew --repository)"
tap_dir="${HOMEBREW_TAP_DIR:-"$brew_repo/Library/Taps/dannolan/homebrew-tap"}"
formula="$tap_dir/Formula/apple-ads-cli.rb"

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

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT
tarball="$tmpdir/apple-ads-cli-$version.tar.gz"
curl -L -s -o "$tarball" "https://github.com/dannolan/apple-ads-cli/archive/refs/tags/$version.tar.gz"
sha="$(shasum -a 256 "$tarball" | awk '{print $1}')"

brew tap dannolan/tap >/dev/null
git -C "$tap_dir" pull --ff-only

perl -0pi -e "s#url \"https://github\\.com/dannolan/apple-ads-cli/archive/refs/tags/v[^\"]+\\.tar\\.gz\"#url \"https://github.com/dannolan/apple-ads-cli/archive/refs/tags/$version.tar.gz\"#" "$formula"
perl -0pi -e "s#sha256 \"[a-f0-9]{64}\"#sha256 \"$sha\"#" "$formula"

brew uninstall apple-ads-cli 2>/dev/null || true
brew install --build-from-source dannolan/tap/apple-ads-cli
ads --help >/dev/null
brew test dannolan/tap/apple-ads-cli
brew audit --strict dannolan/tap/apple-ads-cli

cd "$tap_dir"
git add Formula/apple-ads-cli.rb
git commit -m "Update apple-ads-cli to ${version#v}"
git push origin main

echo "released $version with sha256 $sha"
