# Deployment

Deploy Penshort using Docker and Docker Compose for production environments.

## Prerequisites

- Docker 20.10+
- Docker Compose v2+
- Domain with SSL (for public deployment)

## Quick Production Deploy

### 1. Clone Repository

```bash
git clone https://github.com/penshort/penshort.git
cd penshort
```

### 2. Configure Environment

```bash
cp .env.example .env
# Edit .env with production values
```

Required variables:

| Variable | Description | Example |
|----------|-------------|---------|
| `DATABASE_URL` | PostgreSQL connection string | `postgres://user:pass@host:5432/penshort` |
| `REDIS_URL` | Redis connection string | `redis://host:6379` |
| `APP_ENV` | Environment | `production` |
| `BASE_URL` | Public URL for short links | `https://pnsh.rt` |

### 3. Run Migrations

```bash
make migrate
```

### 4. Start Services

```bash
docker compose -f docker-compose.prod.yml up -d
```

## Production docker-compose.yml

```yaml
version: '3.8'

services:
  api:
    image: ghcr.io/penshort/penshort:latest
    ports:
      - "8080:8080"
    environment:
      - APP_ENV=production
      - DATABASE_URL=${DATABASE_URL}
      - REDIS_URL=${REDIS_URL}
      - BASE_URL=${BASE_URL}
      - LOG_LEVEL=info
      - LOG_FORMAT=json
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/healthz"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 10s
    deploy:
      resources:
        limits:
          memory: 512M

  postgres:
    image: postgres:16-alpine
    environment:
      POSTGRES_USER: ${POSTGRES_USER}
      POSTGRES_PASSWORD: ${POSTGRES_PASSWORD}
      POSTGRES_DB: penshort
    volumes:
      - postgres_data:/var/lib/postgresql/data
    restart: unless-stopped

  redis:
    image: redis:7-alpine
    command: redis-server --appendonly yes
    volumes:
      - redis_data:/data
    restart: unless-stopped

volumes:
  postgres_data:
  redis_data:
```

## Environment Variables

### Required

| Variable | Default | Description |
|----------|---------|-------------|
| `DATABASE_URL` | — | PostgreSQL connection string |
| `REDIS_URL` | — | Redis connection string |

### Optional

| Variable | Default | Description |
|----------|---------|-------------|
| `APP_ENV` | `development` | Environment (development/production) |
| `APP_PORT` | `8080` | HTTP port |
| `BASE_URL` | `http://localhost:8080` | Public URL for short links |
| `LOG_LEVEL` | `info` | Log level (debug/info/warn/error) |
| `LOG_FORMAT` | `json` | Log format (json/text) |
| `READ_TIMEOUT` | `5s` | HTTP read timeout |
| `WRITE_TIMEOUT` | `10s` | HTTP write timeout |
| `SHUTDOWN_TIMEOUT` | `30s` | Graceful shutdown timeout |

## Health Checks

### Liveness Probe

```bash
curl http://localhost:8080/healthz
# {"status":"ok"}
```

Use for Kubernetes liveness probe — checks if process is alive.

### Readiness Probe

```bash
curl http://localhost:8080/readyz
# {"status":"ok","checks":{"postgres":"ok","redis":"ok"}}
```

Use for Kubernetes readiness probe — checks all dependencies.

## Monitoring

### Prometheus Metrics

```bash
curl http://localhost:8080/metrics
```

Key metrics:

| Metric | Type | Description |
|--------|------|-------------|
| `penshort_http_requests_total` | Counter | Total HTTP requests |
| `penshort_http_request_duration_seconds` | Histogram | Request latency |
| `penshort_redirect_cache_hits_total` | Counter | Redis cache hits |
| `penshort_redirect_cache_misses_total` | Counter | Redis cache misses |
| `penshort_webhook_deliveries_total` | Counter | Webhook delivery attempts |
| `penshort_analytics_queue_depth` | Gauge | Analytics queue size |

## Reverse Proxy (Nginx)

```nginx
upstream penshort {
    server 127.0.0.1:8080;
}

server {
    listen 443 ssl http2;
    server_name pnsh.rt;
    
    ssl_certificate /etc/letsencrypt/live/pnsh.rt/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/pnsh.rt/privkey.pem;
    
    location / {
        proxy_pass http://penshort;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

## Scaling

### Horizontal Scaling

Penshort is stateless — run multiple instances behind a load balancer:

```yaml
services:
  api:
    deploy:
      replicas: 3
```

### Vertical Scaling

Increase container resources:

```yaml
deploy:
  resources:
    limits:
      cpus: '2'
      memory: 1G
```

## Backup & Recovery

### PostgreSQL Backup

```bash
pg_dump $DATABASE_URL > backup.sql
```

### Restore

```bash
psql $DATABASE_URL < backup.sql
```

## Troubleshooting

| Issue | Check |
|-------|-------|
| 503 on startup | Wait for Postgres/Redis healthchecks |
| High latency | Check Redis memory, Postgres connections |
| Missing clicks | Check analytics queue depth in metrics |

See [RUNBOOK.md](../RUNBOOK.md) for detailed operational procedures.
