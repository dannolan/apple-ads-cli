# 🍎 Apple Ads CLI

> The best CLI for Apple Ads. Agent-first, JSON-native, spend-safe, and built to work out of the box.

[![Go](https://img.shields.io/badge/Go-1.25+-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![CLI](https://img.shields.io/badge/binary-ads-111827)](#quick-start)
[![Apple Ads API](https://img.shields.io/badge/Apple%20Ads%20API-v5-111827?logo=apple&logoColor=white)](https://developer.apple.com/documentation/apple_ads)
[![License: MIT](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

```bash
# Inspect
$ ads campaigns list --json

# Dry-run
$ ads keywords add 123 456 --text "photo editor,image filter" --match EXACT --bid 1.75 --json

# Apply
$ ads keywords add 123 456 --text "photo editor,image filter" --match EXACT --bid 1.75 --apply --json
```

## ✨ Why `ads`

Apple Ads should have a great CLI. Now it does.

No Python project. No Node stack. No browser automation. No dashboard clicking. Just one Go CLI for the full inspect -> plan -> apply -> verify loop.

- 🤖 **Agent-first** - JSON output, stable commands, predictable workflows.
- 🛡️ **Spend-safe** - mutating commands dry-run first and require `--apply`.
- ⚡ **Works out of the box** - installs as `ads`; config is local and non-interactive.
- 📦 **Single compiled binary** - no runtime dependency chain after install.
- 🔎 **Full escape hatch** - `ads api` reaches raw Apple Ads API endpoints.
- 🔐 **Secret-friendly** - first-class 1Password hydration.
- 🧭 **Four-campaign aware** - Brand, Category, Competitor, Discovery.

This is unofficial. It is also the CLI Apple Ads should have had.

## 🚀 Coverage

- Campaigns, ad groups, keywords, negative keywords, reports, budgets, geo, ads, creatives, ACLs.
- Dry-run mutations everywhere spend can change.
- Clean JSON for agents, scripts, and dashboards.
- Raw API access without writing OAuth code.
- Local config in `~/.apple-ads-cli`.
- 1Password setup that avoids pasting secrets into chat.

## ⚡ Quick Start

### Install 🧰

Download a prebuilt macOS binary:

```bash
arch="$(uname -m)"
case "$arch" in
  arm64) target="arm64" ;;
  x86_64) target="amd64" ;;
  *) echo "unsupported macOS arch: $arch" >&2; exit 1 ;;
esac

curl -L -o ads.zip "https://github.com/dannolan/apple-ads-cli/releases/latest/download/ads_darwin_${target}.zip"
unzip -q -o ads.zip ads

install -m 0755 ads /usr/local/bin/ads
ads --help
```

Or build locally with Go:

```bash
go install github.com/dannolan/apple-ads-cli/cmd/ads@latest
ads --help
```

From a checkout:

```bash
go build -o ads ./cmd/ads
./ads --help
```

Homebrew builds from source:

```bash
brew tap dannolan/tap
brew install apple-ads-cli
ads --help
```

### Configure 🔑

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

Save credentials and an app profile:

```bash
ads config init \
  --org-id 123456 \
  --client-id "$APPLE_ADS_CLIENT_ID" \
  --team-id "$APPLE_ADS_TEAM_ID" \
  --key-id "$APPLE_ADS_KEY_ID" \
  --private-key ./private-key.pem

ads config app add \
  --app-id 1234567890 \
  --name "My App" \
  --countries US,CA,GB \
  --currency USD \
  --bid 1.50 \
  --cpa-goal 5.00
```

Verify access with read-only checks:

```bash
ads config test --json
ads smoke --json
ads campaigns list --json
```

## 🔐 1Password Setup

Do not paste Apple Ads credentials into agent chat. Store them in 1Password and hydrate the local CLI config with the [1Password CLI (`op`)](https://developer.1password.com/docs/cli/).

Recommended layout:

| 1Password object | Name | Fields |
| --- | --- | --- |
| API credential item | `Apple Ads API` | `org_id`, `client_id`, `team_id`, `key_id` |
| Document | `Apple Ads API Private Key` | EC private key PEM |

Then run:

```bash
ads config from-1password \
  --vault "Private" \
  --item "Apple Ads API" \
  --key-document "Apple Ads API Private Key" \
  --app-id 1234567890 \
  --app-name "My App" \
  --countries US \
  --currency USD \
  --json

ads smoke --json
```

`ads smoke` is read-only. It verifies OAuth, `/me`, supported countries, active app config, app eligibility, and campaign listing.

## 🧠 Core Workflows

### Inspect 👀

```bash
ads config show --json
ads acl me --json
ads campaigns list --json
ads campaigns audit --json
ads campaigns health --days 3 --json
ads account snapshot --days 7 --json
```

### Report 📈

```bash
ads reports summary --days 7 --json
ads reports keywords <campaign-id> --days 14 --json
ads reports search-terms <campaign-id> --days 14 --json
ads reports impression-share <campaign-id> --days 14 --json
ads reports impression-share <campaign-id> --days 14 --apply --json
ads reports diagnose <campaign-id> --days 7 --json
```

### Plan 🧾

Mutating commands return a dry-run payload until `--apply` is present.

```bash
ads campaigns setup --prefix "My App" --countries US --daily-budget 50 --json
ads keywords add <campaign-id> <adgroup-id> --text "term one,term two" --match EXACT --bid 1.50 --skip-existing --json
ads keywords add-negatives <campaign-id> --text "bad term" --match EXACT --json
ads campaigns pause <campaign-id> --json
```

### Apply ✅

```bash
ads keywords add <campaign-id> <adgroup-id> \
  --text "term one,term two" \
  --match EXACT \
  --bid 1.50 \
  --apply \
  --json
```

### Verify 🔍

```bash
ads keywords list <campaign-id> <adgroup-id> --json
ads reports summary --days 1 --json
```

## 🗺️ Command Map

| Area | Commands |
| --- | --- |
| Config | `ads config init`, `show`, `test`, `app add`, `app list`, `app use`, `from-1password` |
| Access | `ads acl list`, `me`, `search-apps`, `eligibility`, `countries` |
| Campaigns | `ads campaigns list`, `get`, `audit`, `health`, `setup`, `create`, `update`, `pause`, `enable`, `delete` |
| Ad groups | `ads adgroups list`, `create`, `pause`, `enable`, `delete` |
| Keywords | `ads keywords list`, `add`, `add-negatives`, `find`, `update-bid`, `pause`, `enable`, `delete`, `list-negatives`, `delete-negative` |
| Reports | `ads reports summary`, `adgroups`, `keywords`, `search-terms`, `ads`, `adgroup-keywords`, `adgroup-search-terms`, `impression-share`, `diagnose`, `bid-recommendations`, `custom`, `custom-list`, `custom-get` |
| Account | `ads account snapshot` |
| Budget | `ads budget list`, `get`, `status`, `create` |
| Geo | `ads geo search`, `show`, `set` |
| Ads and creatives | `ads ads list`, `create`, `delete`, `creative`, `creatives`, `product-pages`, `rejections` |
| Escape hatch | `ads api <method> <path>` |

Print the live command manifest:

```bash
ads manifest --json
```

## 💻 Examples

### Campaign Management 🎯

```bash
ads campaigns audit --json
ads campaigns setup --prefix "My App" --countries US --daily-budget 50 --json
ads campaigns setup --prefix "My App" --countries US --daily-budget 50 --apply --json
ads campaigns update <campaign-id> --body '{"dailyBudgetAmount":{"amount":"20.00","currency":"AUD"}}' --apply --json
ads campaigns rename <campaign-id> --name "ARCHIVED - Discovery" --json
ads campaigns set-budget <campaign-id> --amount 20 --json
ads campaigns set-countries <campaign-id> --countries AU,US --json
```

### Keyword Operations 🔑

```bash
ads keywords list <campaign-id> <adgroup-id> --json
ads keywords add <campaign-id> <adgroup-id> --text "brand,my app" --match EXACT --bid 1.50 --json
ads keywords add-negatives <campaign-id> --text "free coins,testflight" --match EXACT --apply --json
ads keywords find --text "photo" --json
ads adgroups set-bid <campaign-id> <adgroup-id> --bid 2.00 --json
ads keywords set-bid <campaign-id> <adgroup-id> <keyword-id> --bid 2.25 --json
ads keywords update-bid <campaign-id> <adgroup-id> <keyword-id> --bid 2.25 --apply --json
```

### Reporting 📊

```bash
ads reports summary --days 7 --json
ads reports adgroups <campaign-id> --days 7 --json
ads reports keywords <campaign-id> --days 14 --json
ads reports search-terms <campaign-id> --days 14 --json
ads reports ads <campaign-id> --days 14 --json
ads reports adgroup-keywords <campaign-id> <adgroup-id> --days 14 --json
ads reports adgroup-search-terms <campaign-id> <adgroup-id> --days 14 --json
ads reports bid-recommendations <campaign-id> <adgroup-id> --json
ads reports impression-share <campaign-id> --days 14 --json
```

`ads reports impression-share` is an async custom report wrapper. It returns a dry-run `/custom-reports` payload unless `--apply` is present.

Use `--table` for human-readable scans while keeping `--json` for agents:

```bash
ads campaigns list --table
ads reports summary --days 7 --table
ads keywords list <campaign-id> <adgroup-id> --table
```

### Budget, Geo, And Ads 💸

```bash
ads budget status --json
ads budget create --name "Q3" --amount 5000 --start 2026-07-01 --apply --json

ads geo search --query "California" --json
ads geo set <campaign-id> --countries US,CA --apply --json

ads ads product-pages --json
ads ads rejections --body '{}' --json
```

### Raw Apple Ads API 🧪

Use this when Apple exposes something before the typed CLI wraps it. Same auth, same config, same dry-run safety for mutating calls.

```bash
ads api GET /campaigns --query limit=100 --json
ads api POST /reports/campaigns --body @body.json --json
ads api PUT /campaigns/123 --body '{"campaign":{"status":"PAUSED"}}' --json
ads api PUT /campaigns/123 --body '{"campaign":{"status":"PAUSED"}}' --apply --json
ads api GET /me --no-org-context --json
```

## 🧪 Optimization

`ads optimize` generates an agent-readable optimization plan for weekly search-term maintenance. It keeps the expensive decisions visible: promote winners, block losers, and prevent Discovery from competing with your exact campaigns.

```bash
ads optimize --days 14 --json
```

Recommended policy:

1. Pull Discovery search-term reports.
2. Treat search terms with at least 2 installs and CPA at or below the app goal as winners.
3. Promote winners as exact keywords in Brand, Category, or Competitor campaigns.
4. Add promoted winners as negatives in Discovery to prevent overlap.
5. Treat search terms with spend and no installs as losers.
6. Add losers as negative keywords to the relevant campaign.

## 🤖 Agent Contract

Agent-first means one strict loop:

1. Inspect account and app state.
2. Run reports needed for the decision.
3. Produce dry-run mutation payloads without `--apply`.
4. Apply only after the plan matches intent.
5. Verify with list or report commands.

The canonical guide is [SKILL.md](SKILL.md). If you are packaging this for an agent runtime, use the bundled skill at [skills/apple-ads-cli/SKILL.md](skills/apple-ads-cli/SKILL.md).

## 🧰 Configuration Files

```text
~/.apple-ads-cli/
|-- credentials.json    # Apple Ads API credentials; private values are redacted by `ads config show`
|-- config.json         # App profiles, active app, countries, bid, and CPA goal
`-- private-key.pem     # Optional local key path when hydrated from 1Password
```

Override the location with `--config-dir` for tests, scripts, and isolated agent runs.

## 🛠️ Development

```bash
go test ./...
go build ./cmd/ads
```

Release and publishing notes live in [docs/release.md](docs/release.md). Wrapped endpoint coverage is tracked in [docs/api-audit.md](docs/api-audit.md).

## 📄 License

[MIT](LICENSE)
