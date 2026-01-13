# Analytics

Query click statistics and breakdowns for your short links.

## Get Link Analytics

```bash
curl -H "Authorization: Bearer $API_KEY" \
  "http://localhost:8080/api/v1/links/{id}/analytics?from=2026-01-01&to=2026-01-31"
```

### Query Parameters

| Param | Type | Default | Description |
|-------|------|---------|-------------|
| `from` | date | 7 days ago | Start date (YYYY-MM-DD) |
| `to` | date | today | End date (YYYY-MM-DD) |
| `include` | string | `referrers,countries,daily` | Breakdown types |

### Response

```json
{
  "link_id": "01HQXK5M7Y...",
  "period": {
    "from": "2026-01-01",
    "to": "2026-01-31"
  },
  "summary": {
    "total_clicks": 1250,
    "unique_visitors": 847
  },
  "breakdown": {
    "daily": [
      { "date": "2026-01-01", "total_clicks": 42, "unique_visitors": 38 },
      { "date": "2026-01-02", "total_clicks": 55, "unique_visitors": 41 }
    ],
    "referrers": [
      { "domain": "twitter.com", "clicks": 450 },
      { "domain": "linkedin.com", "clicks": 280 },
      { "domain": "(direct)", "clicks": 200 }
    ],
    "countries": [
      { "code": "US", "name": "United States", "clicks": 520 },
      { "code": "VN", "name": "Vietnam", "clicks": 180 },
      { "code": "GB", "name": "United Kingdom", "clicks": 95 }
    ]
  },
  "generated_at": "2026-01-13T08:00:00Z"
}
```

## Selective Breakdowns

Request only specific breakdowns using the `include` parameter:

```bash
# Only daily breakdown
curl "...?include=daily"

# Only referrers and countries
curl "...?include=referrers,countries"
```

## Limits

| Constraint | Value |
|------------|-------|
| Max date range | 90 days |
| Top referrers shown | 10 |
| Top countries shown | 10 |

## Unique Visitors

Unique visitors are calculated using:
```
SHA256(IP + User-Agent + daily_salt)
```

This provides:
- Accurate daily unique counts
- No PII storage
- Salt rotation for privacy

## Real-time vs Aggregated

- **Click count** on link object: Real-time (updated on every redirect)
- **Analytics breakdowns**: Near real-time (~1 min delay for batch processing)
