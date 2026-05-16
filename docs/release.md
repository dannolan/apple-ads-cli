# Release Checklist

Use this checklist before tagging a release or updating the Homebrew tap.

The preferred path is the release script:

```bash
scripts/release.sh v0.1.2
```

Set `HOMEBREW_TAP_DIR=/path/to/homebrew-tap` only if you intentionally work from a nonstandard tap checkout. By default the script edits Homebrew's tapped checkout under `$(brew --repository)/Library/Taps/dannolan/homebrew-tap`.

## CLI Repository

Run local checks:

```bash
git status -sb
go test ./...
go build -o /tmp/ads ./cmd/ads
/tmp/ads --help
git ls-files cmd/ads/main.go
```

The `git ls-files` check must print `cmd/ads/main.go`. If it does not, the release tarball will not contain the binary entrypoint.

Tag and push:

```bash
git tag v0.1.2
git push origin v0.1.2
```

Calculate the release tarball SHA:

```bash
curl -L -o apple-ads-cli-0.1.2.tar.gz \
  https://github.com/dannolan/apple-ads-cli/archive/refs/tags/v0.1.2.tar.gz
shasum -a 256 apple-ads-cli-0.1.2.tar.gz
```

## Homebrew Tap

Update `dannolan/homebrew-tap/Formula/apple-ads-cli.rb`:

```ruby
url "https://github.com/dannolan/apple-ads-cli/archive/refs/tags/v0.1.2.tar.gz"
sha256 "<new-sha>"
```

Install from the tapped formula, not a direct path:

```bash
brew update
brew uninstall apple-ads-cli 2>/dev/null || true
brew install --build-from-source dannolan/tap/apple-ads-cli
ads --help
brew test dannolan/tap/apple-ads-cli
brew audit --strict dannolan/tap/apple-ads-cli
```

Commit and push the tap:

```bash
git add Formula/apple-ads-cli.rb
git commit -m "Update apple-ads-cli to 0.1.2"
git push origin main
```

## Failure Modes To Avoid

- Do not tag while `git status --short --untracked-files=all` shows release-critical files.
- Do not use broad `.gitignore` entries that can hide source directories. Ignore root binaries as `/ads`, not `ads`.
- Do not update the tap before the tag exists and the tarball SHA has been calculated from GitHub.
- Do not trust local builds alone; test the Homebrew formula from the public tap.
