# API Endpoint Audit

This file records endpoint checks that matter for wrapped commands.

Checked against Apple Developer Documentation on 2026-05-16:

- Keyword reports: `POST /api/v5/reports/campaigns/{campaignId}/keywords`
- Search term reports: `POST /api/v5/reports/campaigns/{campaignId}/searchterms`
- Product pages: `GET /api/v5/apps/{adamId}/product-pages`
- App eligibility: `POST /api/v5/apps/{adamId}/eligibilities/find`

The CLI also exposes `ads api` for endpoints not yet wrapped.

## Follow-Up Audit Targets

- Ad creation/update payload shapes for custom product pages.
- Keyword recommendation endpoint payloads and filters.
- Custom report body examples and limits.
- Search term report timezone and selector rules against live account responses.
