# apple-ads-cli

Agent-first Go CLI for the Apple Ads API.

The installed binary is `ads`. The repository/package name is descriptive for humans, but the daily command is short enough for agents to use repeatedly:

```bash
ads campaigns list --json
ads reports summary --days 7 --json
ads keywords add 123 456 --text "photo editor,image filter" --match EXACT --bid 1.75
ads keywords add 123 456 --text "photo editor,image filter" --match EXACT --bid 1.75 --apply
```

## Why This Exists

Apple Ads work is operational: inspect account state, form a plan, make small changes, verify. This CLI is built around that loop.

- Mutations dry-run by default and require `--apply`.
- Output is JSON-first so agents can parse it reliably.
- Commands are stable, resource-oriented, and composable.
- `ads api` exposes raw Apple Ads API calls when a typed wrapper is missing.
- Config is non-interactive and lives in `~/.apple-ads-cli`.

This is an unofficial Apple Ads tool.

## For Agents

Agent usage is documented in [SKILL.md](SKILL.md). Treat that file as the canonical operating guide for inspection, dry-run planning, approved mutation, verification, and raw API fallback workflows.

If you are packaging this for an agent runtime, use the bundled skill at [skills/apple-ads-cli/SKILL.md](skills/apple-ads-cli/SKILL.md).

## Install

```bash
go install github.com/dannolan/apple-ads-cli/cmd/ads@latest
```

From a checkout:

```bash
go build -o ads ./cmd/ads
./ads --help
```

Suggested Homebrew naming:

- Formula: `apple-ads-cli`
- Binary: `ads`

Local checks found no Homebrew formula for `apple-ads-cli` or `ads-cli` at the time this project was scaffolded. The bare `ads` formula name should be checked again before publishing, but it is fine as an installed binary name from a more specific formula.

## Homebrew Tap

Once the tap exists, users should install with:

```bash
brew tap dannolan/tap
brew install apple-ads-cli
ads --help
```

### Maintainer Setup

Create a separate tap repository:

```bash
gh repo create dannolan/homebrew-tap --public --clone
cd homebrew-tap
mkdir -p Formula
```

Tag and release this project:

```bash
cd ../apple-ads-cli
export VERSION=v0.1.1
git tag "$VERSION"
git push origin "$VERSION"
```

Download the release tarball and calculate the SHA:

```bash
curl -L -o "apple-ads-cli-$VERSION.tar.gz" \
  "https://github.com/dannolan/apple-ads-cli/archive/refs/tags/$VERSION.tar.gz"
shasum -a 256 "apple-ads-cli-$VERSION.tar.gz"
```

Create `Formula/apple-ads-cli.rb` in `dannolan/homebrew-tap`:

```ruby
class AppleAdsCli < Formula
  desc "Agent-first Go CLI for Apple Ads"
  homepage "https://github.com/dannolan/apple-ads-cli"
  url "https://github.com/dannolan/apple-ads-cli/archive/refs/tags/v0.1.1.tar.gz"
  sha256 "<sha256-from-shasum>"
  license "MIT"

  depends_on "go" => :build

  def install
    ldflags = %W[
      -s -w
      -X github.com/dannolan/apple-ads-cli/internal/cli.Version=#{version}
    ]
    system "go", "build", *std_go_args(ldflags: ldflags, output: bin/"ads"), "./cmd/ads"
  end

  test do
    assert_match "Agent-first CLI for Apple Ads", shell_output("#{bin}/ads --help")
    assert_match version.to_s, shell_output("#{bin}/ads version --json")
  end
end
```

Test and publish the tap:

```bash
brew tap dannolan/tap
brew uninstall apple-ads-cli 2>/dev/null || true
brew install --build-from-source dannolan/tap/apple-ads-cli
ads --help
brew test dannolan/tap/apple-ads-cli
brew audit --strict dannolan/tap/apple-ads-cli

git add Formula/apple-ads-cli.rb
git commit -m "Update apple-ads-cli to 0.1.1"
git push origin main
```

## Configure

Create Apple Ads API credentials in Apple Ads account settings. You need:

- Org ID
- Client ID
- Team ID
- Key ID
- EC private key PEM path

Generate an EC key if needed:

```bash
openssl ecparam -genkey -name prime256v1 -noout -out private-key.pem
openssl ec -in private-key.pem -pubout -out public-key.pem
```

Save credentials non-interactively:

```bash
ads config init \
  --org-id 123456 \
  --client-id "$APPLE_ADS_CLIENT_ID" \
  --team-id "$APPLE_ADS_TEAM_ID" \
  --key-id "$APPLE_ADS_KEY_ID" \
  --private-key ./private-key.pem
```

Add an app profile:

```bash
ads config app add \
  --app-id 1234567890 \
  --name "My App" \
  --countries US,CA,GB \
  --bid 1.50 \
  --cpa-goal 5.00
```

Verify access:

```bash
ads config test --json
ads campaigns list --json
```

### 1Password Credentials

Do not paste Apple Ads credentials into agent chat. Store them in 1Password and hydrate the local CLI config from `op`.

Recommended 1Password layout:

- Vault: `Private` or your team vault
- Item: `Apple Ads API`
- Fields: `org_id`, `client_id`, `team_id`, `key_id`
- Document: `Apple Ads API Private Key` containing the EC private key PEM

Create the scalar fields in 1Password. The CLI form is convenient, but assignment values can appear in shell history and process listings, so prefer the 1Password app or a JSON template for real secrets:

```bash
op item create --category=api_credential \
  --title "Apple Ads API" \
  --vault "Private" \
  'org_id[text]=123456' \
  'client_id[concealed]=YOUR_CLIENT_ID' \
  'team_id[concealed]=YOUR_TEAM_ID' \
  'key_id[concealed]=YOUR_KEY_ID'
```

Store the private key as a document:

```bash
op document create ./private-key.pem \
  --title "Apple Ads API Private Key" \
  --vault "Private" \
  --tags apple-ads-cli
```

Hydrate `ads` config from 1Password:

```bash
export OP_VAULT="Private"
export ADS_CONFIG_DIR="$HOME/.apple-ads-cli"
mkdir -p "$ADS_CONFIG_DIR"

op document get "Apple Ads API Private Key" \
  --vault "$OP_VAULT" \
  --out-file "$ADS_CONFIG_DIR/private-key.pem" \
  --file-mode 0600 \
  --force

ads config init \
  --org-id "$(op read "op://$OP_VAULT/Apple Ads API/org_id")" \
  --client-id "$(op read "op://$OP_VAULT/Apple Ads API/client_id")" \
  --team-id "$(op read "op://$OP_VAULT/Apple Ads API/team_id")" \
  --key-id "$(op read "op://$OP_VAULT/Apple Ads API/key_id")" \
  --private-key "$ADS_CONFIG_DIR/private-key.pem"
```

Then add the app profile and test:

```bash
ads config app add --app-id 1234567890 --name "My App" --countries US --bid 1.50 --cpa-goal 5.00
ads config test --json
```

For agents, the safe pattern is: read from `op`, write only the local config files needed by `ads`, never print raw secret values, then use `ads config show --json` to confirm values are redacted.

## Agent Contract

The short version is below. For the full agent workflow, read [SKILL.md](SKILL.md).

Agents should use this order for account work:

1. Inspect:

```bash
ads config show --json
ads acl me --json
ads campaigns list --json
ads campaigns audit --json
```

2. Report:

```bash
ads reports summary --days 7 --json
ads reports keywords <campaign-id> <adgroup-id> --days 14 --json
ads reports search-terms <campaign-id> <adgroup-id> --days 14 --json
```

3. Plan mutations without `--apply`:

```bash
ads keywords add <campaign-id> <adgroup-id> --text "term one,term two" --match EXACT --bid 1.50 --json
ads keywords add-negatives <campaign-id> --text "bad term" --match EXACT --json
ads campaigns pause <campaign-id> --json
```

4. Execute only after the dry-run payload matches intent:

```bash
ads keywords add <campaign-id> <adgroup-id> --text "term one,term two" --match EXACT --bid 1.50 --apply --json
```

5. Verify:

```bash
ads keywords list <campaign-id> <adgroup-id> --json
ads reports summary --days 1 --json
```

## Commands

### Config

```bash
ads config init --org-id ... --client-id ... --team-id ... --key-id ... --private-key ...
ads config show --json
ads config test --json
ads config app add --app-id ... --name ... --countries US --bid 1.50
ads config app list --json
ads config app use <slug>
```

### Account And Eligibility

```bash
ads acl list --json
ads acl me --json
ads acl search-apps --query "My App" --json
ads acl eligibility <app-id> --json
ads acl countries --json
```

### Campaigns

```bash
ads campaigns list --json
ads campaigns get <campaign-id> --json
ads campaigns audit --json
ads campaigns setup --prefix "My App" --countries US --daily-budget 50
ads campaigns setup --prefix "My App" --countries US --daily-budget 50 --apply
ads campaigns create --name "My App Brand" --countries US --daily-budget 50
ads campaigns update <campaign-id> --body '{"dailyBudgetAmount":{"amount":"75.00","currency":"USD"}}'
ads campaigns pause <campaign-id>
ads campaigns enable <campaign-id>
ads campaigns delete <campaign-id>
```

### Ad Groups

```bash
ads adgroups list <campaign-id> --json
ads adgroups create <campaign-id> --name "Brand Exact" --bid 1.50
ads adgroups pause <campaign-id> <adgroup-id>
ads adgroups enable <campaign-id> <adgroup-id>
ads adgroups delete <campaign-id> <adgroup-id>
```

### Keywords

```bash
ads keywords list <campaign-id> <adgroup-id> --json
ads keywords add <campaign-id> <adgroup-id> --text "brand,my app" --match EXACT --bid 1.50
ads keywords add-negatives <campaign-id> --text "free coins,testflight" --match EXACT
ads keywords find --text "photo" --json
ads keywords update-bid <campaign-id> <adgroup-id> <keyword-id> --bid 2.25
ads keywords pause <campaign-id> <adgroup-id> <keyword-id>
ads keywords enable <campaign-id> <adgroup-id> <keyword-id>
ads keywords delete <campaign-id> <adgroup-id> <keyword-id>
ads keywords list-negatives <campaign-id> --json
ads keywords delete-negative <campaign-id> <negative-keyword-id>
```

### Reports

```bash
ads reports summary --days 7 --json
ads reports adgroups <campaign-id> --days 7 --json
ads reports keywords <campaign-id> <adgroup-id> --days 14 --json
ads reports search-terms <campaign-id> <adgroup-id> --days 14 --json
ads reports ads <campaign-id> <adgroup-id> --days 14 --json
ads reports impression-share <campaign-id> --days 14 --json
ads reports bid-recommendations <campaign-id> <adgroup-id> --json
ads reports custom --body @custom-report.json
ads reports custom --body @custom-report.json --apply
ads reports custom-list --json
ads reports custom-get <report-id> --json
```

### Budget, Geo, Ads

```bash
ads budget list --json
ads budget get <budget-order-id> --json
ads budget status --json
ads budget create --name "Q3" --amount 5000 --start 2026-07-01

ads geo search --query "California" --json
ads geo show <campaign-id> --json
ads geo set <campaign-id> --countries US,CA

ads ads list <campaign-id> <adgroup-id> --json
ads ads create <campaign-id> <adgroup-id> --body @ad.json
ads ads delete <campaign-id> <adgroup-id> <ad-id>
ads ads creatives --json
ads ads product-pages --json
ads ads rejections --body '{}'
```

### Raw API Escape Hatch

Use this when Apple exposes something before the typed CLI wraps it.

```bash
ads api GET /campaigns --query limit=100 --json
ads api POST /reports/campaigns --body @body.json --json
ads api PUT /campaigns/123 --body '{"status":"PAUSED"}'
ads api PUT /campaigns/123 --body '{"status":"PAUSED"}' --apply
ads api GET /me --no-org-context --json
```

## Optimization Workflow

`ads optimize` is intentionally plan-only. It tells agents how to run a weekly optimization without hiding spend-affecting changes in one command.

```bash
ads optimize --days 14 --json
```

Recommended policy:

- Winners: search terms with at least 2 installs and CPA at or below the app goal.
- Promote winners as exact keywords in Brand, Category, or Competitor campaigns.
- Add promoted winners as negatives in Discovery to prevent overlap.
- Losers: search terms with spend and no installs.
- Add losers as negative keywords to the relevant campaign.

## Development

```bash
go test ./...
go build ./cmd/ads
```

## Releases

Before tagging or updating Homebrew, run `scripts/release.sh vX.Y.Z` or follow [docs/release.md](docs/release.md). The release checklist exists to catch missing entrypoints, ignored files, stale tarball SHAs, and formula build failures before users hit them.

## API Audit

Wrapped endpoint checks are tracked in [docs/api-audit.md](docs/api-audit.md). For unsupported or newly released Apple Ads endpoints, use `ads api`.
