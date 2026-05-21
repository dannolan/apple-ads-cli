# API Endpoint Audit

This file records endpoint checks that matter for wrapped commands.

Checked against Apple Developer Documentation on 2026-05-21:

- Keyword reports: `POST /api/v5/reports/campaigns/{campaignId}/keywords`
- Keyword reports within an ad group: `POST /api/v5/reports/campaigns/{campaignId}/adgroups/{adgroupId}/keywords`
- Targeting keyword updates: `PUT /api/v5/campaigns/{campaignId}/adgroups/{adgroupId}/targetingkeywords/bulk`
- Search term reports: `POST /api/v5/reports/campaigns/{campaignId}/searchterms`
- Search term reports within an ad group: `POST /api/v5/reports/campaigns/{campaignId}/adgroups/{adgroupId}/searchterms`
- Campaign negative keyword deletes: `POST /api/v5/campaigns/{campaignId}/negativekeywords/delete/bulk`
- Product pages: `GET /api/v5/apps/{adamId}/product-pages`
- App eligibility: `POST /api/v5/apps/{adamId}/eligibilities/find`
- Bid recommendations: surfaced in keyword report row `insights.bidRecommendation`; API v5 does not expose a separate `GET /targetingkeywords/recommendations` endpoint.
- Impression share: async custom report flow through `POST /api/v5/custom-reports`; dry-run by default to avoid spending daily report quota.

The CLI also exposes `ads api` for endpoints not yet wrapped.

## Follow-Up Audit Targets

- Ad creation/update payload shapes for custom product pages.
- Custom report body examples and limits.
- Additional custom report filters and quotas.
