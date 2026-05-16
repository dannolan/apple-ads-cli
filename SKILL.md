---
name: apple-ads-cli
description: Use when managing Apple Ads through the `ads` CLI, including campaign inspection, keyword changes, reports, budgets, geo targeting, raw Apple Ads API calls, or agent-safe Apple Ads optimization workflows.
---

# Apple Ads CLI

Use `ads` for Apple Ads account work. It is agent-first: inspect state, produce dry-run mutations, execute only with `--apply`, then verify.

## Rules

- Prefer `--json` on every command.
- Never mutate on the first pass. Run the same command without `--apply` and inspect the dry-run payload first.
- Use `--apply` only when the requested change is explicit or the user approved the dry-run plan.
- Use configured app context with `--app <slug>` when more than one app exists.
- Do not increase budgets, pause campaigns, delete entities, or add broad-match keywords unless the user asked for that exact class of change.
- If a typed command is missing, use `ads api`, still dry-run first for non-GET methods.

## First Inspection

```bash
ads config show --json
ads config test --json
ads smoke --json
ads acl me --json
ads campaigns list --json
ads campaigns audit --json
ads campaigns health --days 3 --json
ads account snapshot --days 7 --json
```

If credentials or app config are missing:

```bash
ads config init --org-id <org> --client-id <client> --team-id <team> --key-id <key> --private-key <pem>
ads config app add --app-id <adam-id> --name "<app name>" --countries US --currency USD --bid 1.50 --cpa-goal 5.00
```

## Reporting Workflow

Use reports before changing keywords or budgets.

```bash
ads reports summary --days 7 --json
ads reports adgroups <campaign-id> --days 7 --json
ads reports keywords <campaign-id> --days 14 --json
ads reports search-terms <campaign-id> --days 14 --json
ads reports diagnose <campaign-id> --days 7 --json
```

Use `--table` only for human scanning. Keep `--json` for agent decisions and parsing.

For larger/custom reporting, create an explicit JSON body and dry-run first:

```bash
ads reports custom --body @custom-report.json --json
ads reports custom --body @custom-report.json --apply --json
ads reports custom-list --json
ads reports custom-get <report-id> --json
```

## Mutation Pattern

Every mutation follows this two-step pattern:

```bash
ads keywords add <campaign-id> <adgroup-id> --text "term one,term two" --match EXACT --bid 1.50 --skip-existing --json
ads keywords add <campaign-id> <adgroup-id> --text "term one,term two" --match EXACT --bid 1.50 --skip-existing --apply --json
ads campaigns set-budget <campaign-id> --amount 20 --json
ads campaigns set-countries <campaign-id> --countries AU,US --json
ads campaigns rename <campaign-id> --name "ARCHIVED - Discovery" --json
ads adgroups set-bid <campaign-id> <adgroup-id> --bid 2.00 --json
ads keywords set-bid <campaign-id> <adgroup-id> <keyword-id> --bid 2.00 --json
```

Verify after execution:

```bash
ads keywords list <campaign-id> <adgroup-id> --json
ads reports summary --days 1 --json
```

## Keyword Operations

Add exact keywords:

```bash
ads keywords add <campaign-id> <adgroup-id> --text "brand,my app" --match EXACT --bid 1.50 --json
```

Add campaign negatives:

```bash
ads keywords add-negatives <campaign-id> --text "irrelevant term,free coins" --match EXACT --json
```

Find existing keywords before adding duplicates:

```bash
ads keywords find --text "photo" --json
```

Change bids conservatively:

```bash
ads keywords update-bid <campaign-id> <adgroup-id> <keyword-id> --bid 2.00 --json
```

## Campaign Structure

Audit first:

```bash
ads campaigns audit --json
```

Plan the four-campaign structure:

```bash
ads campaigns setup --prefix "<App Name>" --countries US --daily-budget 50 --json
```

Only execute after approval:

```bash
ads campaigns setup --prefix "<App Name>" --countries US --daily-budget 50 --apply --json
```

Expected campaign types:

- Brand: app/company terms, exact match.
- Category: non-branded category terms, exact match.
- Competitor: competitor brand terms, exact match.
- Discovery: search term mining and broad expansion.

## Optimization

`ads optimize` is plan-only by design:

```bash
ads optimize --days 14 --json
```

Agent policy:

- Winners: at least 2 installs and CPA at or below goal.
- Promote winners with `ads keywords add` using `EXACT`.
- Add promoted winners as Discovery negatives.
- Losers: spend with zero installs.
- Add losers as campaign negatives.
- Verify with reports after changes.

## Raw API

Use raw API for unsupported endpoints:

```bash
ads api GET /campaigns --query limit=100 --json
ads api POST /reports/campaigns --body @body.json --json
ads api POST /reports/campaigns --body @body.json --apply --json
ads api GET /me --no-org-context --json
```

Non-GET methods dry-run unless `--apply` is present.
