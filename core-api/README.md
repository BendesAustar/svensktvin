# Svenskt Vin Core API

Go REST API for the Svenskt Vin viticulture registry system.

## Architecture

```
┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│  SvelteKit   │────▶│  Go API      │────▶│  Postgres    │
│  (SSR + SPA) │  │  │  (core-api)  │  │  │  :5434       │
│  :5173       │◀───│  :9091       │◀───│  svensktvin  │
└──────────────┘     └──────────────┘     └──────────────┘
```

## Endpoints

### Health
- `GET /health` — Health check

### Auth
- `POST /api/auth/send-link` — Send magic link (rate-limited)
- `POST /api/auth/verify` — Verify magic link, get session

### Varieties
- `GET /api/varieties` — List approved varieties (requires auth)
- `POST /api/varieties/submit` — Submit new variety for review

### Vineyards
- `GET /api/vineyards` — List user's vineyards
- `POST /api/vineyards` — Create vineyard
- `GET /api/vineyards/:id` — Get vineyard details
- `PUT /api/vineyards/:id` — Update vineyard (owner only)
- `DELETE /api/vineyards/:id` — Soft-delete vineyard (owner only)

### Blocks
- `GET /api/vineyards/:id/blocks` — List blocks
- `POST /api/vineyards/:id/blocks` — Create block
- `GET /api/vineyards/:id/blocks/:blockId` — Get block
- `PUT /api/vineyards/:id/blocks/:blockId` — Update block
- `DELETE /api/vineyards/:id/blocks/:blockId` — Soft-delete block
- `POST /api/blocks/search-varieties` — Fuzzy-search varieties

### Harvests
- `GET /api/vineyards/:id/harvests` — List harvest records
- `POST /api/vineyards/:id/harvests` — Create harvest record
- `GET /api/vineyards/:id/harvests/:recordId` — Get harvest record
- `PUT /api/vineyards/:id/harvests/:recordId` — Update harvest record
- `DELETE /api/vineyards/:id/harvests/:recordId` — Soft-delete harvest

### Benchmarks
- `GET /api/vineyards/:id/benchmarks` — Get benchmark data

### Members
- `GET /api/vineyards/:id/members` — List members
- `POST /api/vineyards/:id/members` — Add member (owner)
- `PUT /api/vineyards/:id/members/:userId` — Update member role (owner)
- `DELETE /api/vineyards/:id/members/:userId` — Remove member (owner)

## Running

```bash
# Dev
go run ./cmd/api

# Build
go build -o bin/api ./cmd/api

# Docker
docker build -t svensktvin/core-api .
```

## Configuration

Copy `config.yaml` and adjust:

```yaml
api:
  port: 9091

database:
  url: "postgres://user:pass@host:port/dbname"

auth:
  session_expiry: "720h"

rate_limit:
  auth_requests: 3
  auth_window: "5m"
  write_requests: 30
  write_window: "1m"
```
