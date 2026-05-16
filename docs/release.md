# Release Checklist

Releases are tag-driven. GitHub Actions owns the GitHub release, macOS zip assets, checksums, and Homebrew tap formula update.

## One-Time Setup

Create a repository secret named `HOMEBREW_TAP_TOKEN`.

The token must be able to push to `dannolan/homebrew-tap`. A fine-grained PAT should include:

- Repository access: `dannolan/homebrew-tap`
- Permissions: `Contents: Read and write`

The release workflow rewrites `Formula/apple-ads-cli.rb` as a binary formula. Homebrew installs the prebuilt `ads` binary from the matching macOS zip, so users do not need Go to install from the tap.

## Cut A Release

Run local checks:

```bash
git status -sb
go test ./...
go vet ./...
go build -o /tmp/ads ./cmd/ads
/tmp/ads --help
git ls-files cmd/ads/main.go
```

Tag and push:

```bash
scripts/release.sh v0.1.2
```

The script only validates the repo and pushes the tag. The `Release` GitHub Actions workflow then:

1. Runs tests.
2. Builds `ads` for `darwin/arm64`.
3. Builds `ads` for `darwin/amd64`.
4. Publishes GitHub release assets.
5. Generates `checksums.txt`.
6. Calculates SHA-256 values for both macOS zip files.
7. Rewrites `dannolan/homebrew-tap/Formula/apple-ads-cli.rb` with the zip URLs and checksums.
8. Commits and pushes the tap change.

## Release Assets

Each tag publishes:

- `ads_darwin_arm64.zip` for Apple Silicon Macs
- `ads_darwin_amd64.zip` for Intel Macs
- `checksums.txt`

Verify the release:

```bash
gh run list --workflow Release --limit 1
gh release view v0.1.2 --json tagName,assets
gh release download v0.1.2 --pattern 'ads_darwin_arm64.zip' --dir /tmp/apple-ads-cli-release
unzip -l /tmp/apple-ads-cli-release/ads_darwin_arm64.zip
```

Verify Homebrew:

```bash
brew update
brew uninstall apple-ads-cli 2>/dev/null || true
brew install --build-from-source dannolan/tap/apple-ads-cli
ads --help
brew test dannolan/tap/apple-ads-cli
brew audit --strict dannolan/tap/apple-ads-cli
```

## Failure Modes To Avoid

- Do not tag while `git status --short --untracked-files=all` shows release-critical files.
- Do not update the tap by hand before the tag workflow finishes.
- Do not remove `cmd/ads/main.go`; the source formula depends on that entrypoint.
- Do not claim a release is published until both GitHub release assets and the tap commit exist.
