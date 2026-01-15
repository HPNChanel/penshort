# Demo Deployment

Minimal public deployment guide for Penshort demo environments, aligned with [PROJECT.txt](../PROJECT.txt) evidence requirements.

## Target Evidence

| Requirement | How to Achieve |
|-------------|----------------|
| Public deployment | Deploy to cloud platform with public URL |
| Minimal monitoring | Health checks + uptime monitoring |
| 10 real API keys/users | Track through admin endpoints or database queries |

---

## Recommended Platforms

For OSS demo deployments, these platforms offer free or low-cost tiers:

| Platform | Pros | Cons |
|----------|------|------|
| **Fly.io** | Easy Docker deploys, global edge, free tier | Limited free resources |
| **Railway** | GitHub integration, managed Postgres/Redis | Free tier limits |
| **Render** | Free web services, easy setup | Cold starts on free tier |
| **DigitalOcean App Platform** | Predictable pricing, good DX | No free tier |

### Recommended Stack for Demo

```
┌─────────────────────────────────────────────┐
│              Fly.io / Railway               │
├─────────────────────────────────────────────┤
│  Penshort API (Docker)                      │
│  - 256MB RAM minimum                        │
│  - Single instance                          │
├─────────────────────────────────────────────┤
│  Managed PostgreSQL                         │
│  - Free tier or $5-15/mo                    │
├─────────────────────────────────────────────┤
│  Managed Redis                              │
│  - Free tier (Upstash) or $5-10/mo          │
└─────────────────────────────────────────────┘
```

---

## Fly.io Deployment

### Prerequisites

- [Fly CLI](https://fly.io/docs/hands-on/install-flyctl/) installed
- Fly.io account

### Steps

1. **Initialize Fly app**:
   ```bash
   fly launch --no-deploy
   ```

2. **Create Postgres**:
   ```bash
   fly postgres create --name penshort-db
   fly postgres attach penshort-db
   ```

3. **Create Redis** (using Upstash):
   ```bash
   fly redis create
   ```

4. **Set secrets**:
   ```bash
   fly secrets set \
     APP_ENV=production \
     BASE_URL=https://your-app.fly.dev \
     LOG_LEVEL=info \
     LOG_FORMAT=json
   ```

5. **Deploy**:
   ```bash
   fly deploy
   ```

6. **Run migrations**:
   ```bash
   fly ssh console -C "/app/api migrate up"
   ```

### fly.toml Example

```toml
app = "penshort-demo"
primary_region = "sjc"

[build]
  dockerfile = "Dockerfile"

[http_service]
  internal_port = 8080
  force_https = true
  auto_stop_machines = true
  auto_start_machines = true
  min_machines_running = 0

[[services]]
  protocol = "tcp"
  internal_port = 8080

  [[services.ports]]
    port = 80
    handlers = ["http"]

  [[services.ports]]
    port = 443
    handlers = ["tls", "http"]

  [[services.http_checks]]
    interval = 10000
    timeout = 2000
    path = "/healthz"
```

---

## Railway Deployment

### Steps

1. **Create project** from GitHub repo in Railway dashboard

2. **Add services**:
   - PostgreSQL (from Railway templates)
   - Redis (from Railway templates)

3. **Configure variables**:
   ```
   DATABASE_URL=${{Postgres.DATABASE_URL}}
   REDIS_URL=${{Redis.REDIS_URL}}
   APP_ENV=production
   BASE_URL=https://your-project.up.railway.app
   ```

4. **Deploy** automatically on push to main

---

## Minimal Monitoring Setup

### Health Check Monitoring (Free)

Use one of these services to monitor `/healthz` endpoint:

| Service | Features | Cost |
|---------|----------|------|
| [UptimeRobot](https://uptimerobot.com) | 50 monitors, 5-min intervals | Free |
| [Pingdom](https://www.pingdom.com) | 1 monitor | Free trial |
| [Better Uptime](https://betterstack.com/uptime) | Unlimited monitors | Free tier |

**Configure:**
- URL: `https://your-demo.fly.dev/healthz`
- Check interval: 5 minutes
- Alert on: HTTP status != 200

### Basic Metrics

For demo purposes, the built-in `/metrics` endpoint provides Prometheus-format metrics:

```bash
curl https://your-demo.fly.dev/metrics
```

Key metrics to watch:
- `penshort_http_requests_total` — Traffic volume
- `penshort_http_request_duration_seconds` — Latency
- `penshort_redirect_cache_hits_total` — Cache effectiveness

### Log Aggregation (Optional)

Most platforms provide built-in log viewing:
- **Fly.io**: `fly logs`
- **Railway**: Dashboard → Logs tab

For more advanced needs, consider:
- [Logtail](https://betterstack.com/logtail) (free tier)
- [Papertrail](https://www.papertrail.com/) (limited free)

---

## Tracking 10 Real API Keys/Users

### Method 1: Database Query

```sql
SELECT COUNT(*) FROM api_keys WHERE deleted_at IS NULL;
SELECT COUNT(DISTINCT user_id) FROM api_keys;
```

### Method 2: Admin Endpoint (if implemented)

```bash
curl -H "Authorization: Bearer $ADMIN_TOKEN" \
  https://your-demo.fly.dev/admin/stats
```

### Method 3: Prometheus Metrics

Add a gauge metric `penshort_api_keys_total` and query via `/metrics`.

### Evidence Documentation

Create a simple tracking log:

```markdown
## API Key Adoption Log

| Date | API Keys | Active Users | Notes |
|------|----------|--------------|-------|
| 2026-01-15 | 3 | 2 | Initial beta testers |
| 2026-01-22 | 7 | 5 | Posted on X/Twitter |
| 2026-01-30 | 12 | 10 | Target reached ✓ |
```

---

## Cost Expectations

### Minimal Demo (Free Tier)

| Component | Platform | Cost |
|-----------|----------|------|
| API | Fly.io free tier | $0 |
| PostgreSQL | Fly.io free Postgres | $0 |
| Redis | Upstash free tier | $0 |
| Monitoring | UptimeRobot | $0 |
| **Total** | | **$0/mo** |

### Production-Ready Demo

| Component | Platform | Cost |
|-----------|----------|------|
| API | Fly.io (always-on) | ~$5/mo |
| PostgreSQL | Fly.io or Supabase | ~$5-15/mo |
| Redis | Upstash Pro | ~$5/mo |
| Domain | Namecheap/Cloudflare | ~$10/yr |
| **Total** | | **~$15-25/mo** |

---

## Demo Data Seeding

For demonstration purposes, create sample data:

```bash
# Create demo API key
curl -X POST https://your-demo.fly.dev/api/v1/keys \
  -H "Content-Type: application/json" \
  -d '{"name": "demo-key"}'

# Create sample short links
curl -X POST https://your-demo.fly.dev/api/v1/links \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"url": "https://github.com/hpnchanel/penshort", "alias": "gh"}'
```

---

## See Also

- [Deployment](./deployment.md) — Full production deployment guide
- [Releasing](./releasing.md) — Release process documentation
- [RUNBOOK.md](../RUNBOOK.md) — Operational procedures
