---
name: apple-ads-cli
description: Use when managing Apple Ads through the `ads` CLI, including campaign inspection, keyword changes, reports, budgets, geo targeting, raw Apple Ads API calls, or agent-safe Apple Ads optimization workflows.
---

# Apple Ads CLI

Use `ads` for Apple Ads account work. It is agent-first: inspect state, produce dry-run mutations, execute only with `--apply`, then verify.

## Core Rules

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
```

## Reporting Workflow

```bash
ads reports summary --days 7 --json
ads reports adgroups <campaign-id> --days 7 --json
ads reports keywords <campaign-id> <adgroup-id> --days 14 --json
ads reports search-terms <campaign-id> <adgroup-id> --days 14 --json
```

## Mutation Pattern

```bash
ads keywords add <campaign-id> <adgroup-id> --text "term one,term two" --match EXACT --bid 1.50 --json
ads keywords add <campaign-id> <adgroup-id> --text "term one,term two" --match EXACT --bid 1.50 --apply --json
ads keywords list <campaign-id> <adgroup-id> --json
```

## Campaign Structure

```bash
ads campaigns audit --json
ads campaigns setup --prefix "<App Name>" --countries US --daily-budget 50 --json
ads campaigns setup --prefix "<App Name>" --countries US --daily-budget 50 --apply --json
```

Expected types: Brand, Category, Competitor, Discovery.

## Optimization

`ads optimize` is plan-only:

```bash
ads optimize --days 14 --json
```

Policy: promote search-term winners as exact keywords, add promoted terms as Discovery negatives, block losers as negatives, then verify with reports.

## Raw API

```bash
ads api GET /campaigns --query limit=100 --json
ads api POST /reports/campaigns --body @body.json --json
ads api POST /reports/campaigns --body @body.json --apply --json
ads api GET /me --no-org-context --json
```
