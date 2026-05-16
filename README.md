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

## Agent Contract

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

